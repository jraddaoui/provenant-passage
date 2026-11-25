// SPDX-FileCopyrightText: Artefactual Systems Inc. and Enduro contributors
// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package aips

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	api "github.com/jraddaoui/provenant-passage/internal/api/gen/aips"
	"github.com/jraddaoui/provenant-passage/internal/enums"
	"github.com/jraddaoui/provenant-passage/internal/event"
	"github.com/jraddaoui/provenant-passage/internal/persistence"
)

type Service struct {
	Persistence    persistence.Service
	Source, Output Store
	Workflows      WorkflowClient
	Events         event.Service[*api.AIPEvent]
	Logger         logr.Logger
}

var _ api.Service = (*Service)(nil)

func Public(a *persistence.AIP) *api.ProvenantAip {
	return &api.ProvenantAip{
		UUID: a.UUID.String(), Name: a.Name, Status: string(a.Status),
		CreatedAt: a.CreatedAt.UTC().Format(time.RFC3339), Profile: string(a.Profile), OcflObjectKey: a.OCFLObjectKey,
	}
}

func SourceKey(id uuid.UUID, name string) string { return "aips/" + id.String() + "/" + name }
func OutputKey(id uuid.UUID, name string) string { return "aips/" + id.String() + "/" + name }

func (s *Service) internalError(action string, err error) error {
	s.Logger.Error(err, action)
	return api.MakeInternalError(errors.New(action))
}

func (s *Service) read(ctx context.Context, text string) (*persistence.AIP, error) {
	id, err := uuid.Parse(text)
	if err != nil {
		return nil, api.MakeNotValid(errors.New("invalid UUID"))
	}
	a, err := s.Persistence.Read(ctx, id)
	if errors.Is(err, persistence.ErrNotFound) {
		return nil, api.MakeNotFound(errors.New("AIP not found"))
	}
	if err != nil {
		return nil, s.internalError("read AIP", err)
	}
	return a, nil
}

func (s *Service) List(ctx context.Context, p *api.ListPayload) (*api.ListResult, error) {
	limit, offset := 0, 0
	if p.Limit != nil {
		limit = *p.Limit
	}
	if p.Offset != nil {
		offset = *p.Offset
	}
	rows, page, err := s.Persistence.List(ctx, limit, offset)
	if err != nil {
		return nil, s.internalError("list AIPs", err)
	}
	items := make(api.ProvenantAipCollection, 0, len(rows))
	for _, a := range rows {
		items = append(items, Public(a))
	}
	return &api.ListResult{
		Items: items,
		Page:  &api.ProvenantPage{Limit: page.Limit, Offset: page.Offset, Total: page.Total},
	}, nil
}

func (s *Service) Show(ctx context.Context, p *api.ShowPayload) (*api.ProvenantAip, error) {
	a, err := s.read(ctx, p.UUID)
	if err != nil {
		return nil, err
	}
	return Public(a), nil
}

func (s *Service) Download(ctx context.Context, p *api.DownloadPayload) (*api.DownloadResult, io.ReadCloser, error) {
	a, err := s.read(ctx, p.UUID)
	if err != nil {
		return nil, nil, err
	}
	if a.Status != enums.StatusCompleted || a.OCFLObjectKey == nil || *a.OCFLObjectKey == "" {
		return nil, nil, api.MakeNotReady(errors.New("OCFL download is not ready"))
	}
	r, err := s.Output.NewReader(ctx, *a.OCFLObjectKey)
	if errors.Is(err, ErrObjectNotFound) {
		return nil, nil, api.MakeNotFound(errors.New("OCFL object not found"))
	}
	if err != nil {
		return nil, nil, s.internalError("open OCFL download", err)
	}
	return &api.DownloadResult{
		ContentType: "application/zip", ContentLength: r.Size(),
		ContentDisposition: fmt.Sprintf(`attachment; filename="%s-ocfl.zip"`, a.UUID),
	}, r, nil
}

func (s *Service) PublishCreated(ctx context.Context, a *persistence.AIP) {
	s.Events.PublishEvent(
		ctx,
		&api.AIPEvent{Value: api.NewValueAipCreatedEvent(&api.AIPCreatedEvent{UUID: a.UUID.String(), Item: Public(a)})},
	)
}

func PublishUpdated(ctx context.Context, events event.Service[*api.AIPEvent], a *persistence.AIP) {
	events.PublishEvent(
		ctx,
		&api.AIPEvent{Value: api.NewValueAipUpdatedEvent(&api.AIPUpdatedEvent{UUID: a.UUID.String(), Item: Public(a)})},
	)
}
