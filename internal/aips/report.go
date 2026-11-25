// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package aips

import (
	"context"
	"errors"
	"io"

	api "github.com/jraddaoui/provenant-passage/internal/api/gen/aips"
	"github.com/jraddaoui/provenant-passage/internal/enums"
)

func (s *Service) Report(ctx context.Context, p *api.ReportPayload) (*api.ReportResult, io.ReadCloser, error) {
	var filename, contentType string
	switch p.Format {
	case "json":
		filename, contentType = "import-event.json", "application/json"
	case "md":
		filename, contentType = "import-report.md", "text/markdown; charset=utf-8"
	default:
		return nil, nil, api.MakeNotValid(errors.New("invalid report format"))
	}
	a, err := s.read(ctx, p.UUID)
	if err != nil {
		return nil, nil, err
	}
	if a.Status != enums.StatusCompleted {
		return nil, nil, api.MakeNotReady(errors.New("import report is not ready"))
	}
	r, err := s.Output.NewReader(ctx, OutputKey(a.UUID, filename))
	if errors.Is(err, ErrObjectNotFound) {
		return nil, nil, api.MakeNotFound(errors.New("import report not found"))
	}
	if err != nil {
		return nil, nil, s.internalError("open import report", err)
	}
	return &api.ReportResult{ContentType: contentType, ContentLength: r.Size()}, r, nil
}
