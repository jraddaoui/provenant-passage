// SPDX-FileCopyrightText: Artefactual Systems Inc. and Enduro contributors
// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package aips

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/mholt/archives"

	api "github.com/jraddaoui/provenant-passage/internal/api/gen/aips"
	"github.com/jraddaoui/provenant-passage/internal/enums"
	"github.com/jraddaoui/provenant-passage/internal/persistence"
)

func ValidateName(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\") || !utf8.ValidString(name) {
		return errors.New("filename must be an ordinary archive basename")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return errors.New("filename contains a control character")
		}
	}
	return nil
}

// cleanup is independent of the request's cancellation, but always bounded.
func cleanup(ctx context.Context, fn func(context.Context) error) error {
	c, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	return fn(c)
}

func (s *Service) Upload(ctx context.Context, p *api.UploadPayload, req io.ReadCloser) (*api.UploadResult, error) {
	defer req.Close()
	profile, err := enums.ParseProfileWithDefault(p.Profile)
	if err != nil {
		return nil, api.MakeNotValid(err)
	}
	media, params, err := mime.ParseMediaType(p.ContentType)
	if err != nil || media != "multipart/form-data" || params["boundary"] == "" {
		return nil, api.MakeNotValid(errors.New("expected multipart/form-data with boundary"))
	}
	mr := multipart.NewReader(req, params["boundary"])
	part, err := mr.NextPart()
	if err != nil {
		return nil, api.MakeNotValid(errors.New("expected exactly one file"))
	}
	// Part.FileName sanitizes paths. Inspect the raw parameter so unsafe names
	// are rejected rather than silently changing archive identity.
	disposition, fields, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
	name := fields["filename"]
	if err != nil || disposition != "form-data" || fields["name"] != "file" {
		return nil, api.MakeNotValid(errors.New("expected a file form part"))
	}
	if err := ValidateName(name); err != nil {
		return nil, api.MakeNotValid(err)
	}
	ext := filepath.Ext(name)
	if ext != ".tar" && ext != ".7z" {
		return nil, api.MakeNotValid(errors.New("accepted formats: .tar and .7z"))
	}
	// Identify from content only: passing the extension also accepts bogus data.
	format, stream, err := archives.Identify(ctx, "", part)
	if err != nil || format == nil || format.Extension() != ext {
		return nil, api.MakeNotValid(errors.New("archive content does not match its extension"))
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, s.internalError("generate request UUID", err)
	}
	key := SourceKey(id, name)
	removeSource := func() error { return cleanup(ctx, func(c context.Context) error { return s.Source.Delete(c, key) }) }
	wr, err := s.Source.NewWriter(ctx, key)
	if err != nil {
		return nil, s.internalError("open source upload", errors.Join(err, removeSource()))
	}
	_, copyErr := io.Copy(wr, stream)
	closeErr := wr.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return nil, s.internalError("write source upload", errors.Join(err, removeSource()))
	}
	if _, err := mr.NextPart(); err != io.EOF {
		cleanupErr := removeSource()
		if cleanupErr != nil {
			return nil, s.internalError("clean invalid upload", cleanupErr)
		}
		return nil, api.MakeNotValid(errors.New("expected exactly one file and valid multipart framing"))
	}
	a := &persistence.AIP{UUID: id, Name: name, Profile: profile}
	createCtx, stopCreate := context.WithTimeout(ctx, 10*time.Second)
	createErr := s.Persistence.Create(createCtx, a)
	stopCreate()
	if createErr != nil {
		// A failed response can follow a committed INSERT. Check by UUID before
		// deleting the archive needed by an already persisted request.
		var existing *persistence.AIP
		readErr := cleanup(
			ctx,
			func(c context.Context) error { var e error; existing, e = s.Persistence.Read(c, id); return e },
		)
		switch {
		case readErr == nil:
			a = existing
		case errors.Is(readErr, persistence.ErrNotFound):
			return nil, s.internalError("create AIP", errors.Join(createErr, removeSource()))
		default:
			return nil, s.internalError("resolve AIP creation", errors.Join(createErr, readErr))
		}
	}
	r := Request(a)
	startCtx, stopStart := context.WithTimeout(ctx, 10*time.Second)
	startErr := s.Workflows.Start(startCtx, r)
	stopStart()
	if startErr != nil {
		if !errors.Is(startErr, ErrStartRejected) {
			var state ExecutionState
			inspectErr := cleanup(ctx, func(c context.Context) error {
				var e error
				state, e = s.Workflows.Inspect(c, r.WorkflowID())
				return e
			})
			if inspectErr == nil && state != ExecutionAbsent {
				s.PublishCreated(context.WithoutCancel(ctx), a)
				return &api.UploadResult{UUID: id.String()}, nil
			}
			// Even an immediate not-found may race a timed-out start. Preserve
			// queued evidence; the bounded reconciler retries the same workflow ID.
			s.Logger.Error(errors.Join(startErr, inspectErr), "Unresolved workflow start", "uuid", id)
			return nil, api.MakeUnavailable(
				fmt.Errorf("workflow start unresolved for AIP %s; retained for reconciliation", id),
			)
		}
		deleteErr := cleanup(ctx, func(c context.Context) error { return s.Persistence.Delete(c, id) })
		if deleteErr == nil {
			deleteErr = removeSource()
		}
		return nil, s.internalError("start import workflow", errors.Join(startErr, deleteErr))
	}
	s.PublishCreated(context.WithoutCancel(ctx), a)
	return &api.UploadResult{UUID: id.String()}, nil
}
