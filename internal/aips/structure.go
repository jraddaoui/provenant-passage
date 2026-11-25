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

func (s *Service) Structure(ctx context.Context, p *api.StructurePayload) (*api.StructureResult, io.ReadCloser, error) {
	store, filename := s.Source, "am-aip-tree.txt"
	switch p.Kind {
	case "aip":
	case "ocfl":
		store, filename = s.Output, "ocfl-tree.txt"
	default:
		return nil, nil, api.MakeNotValid(errors.New("invalid structure kind"))
	}
	a, err := s.read(ctx, p.UUID)
	if err != nil {
		return nil, nil, err
	}
	if a.Status != enums.StatusCompleted {
		return nil, nil, api.MakeNotReady(errors.New("package structure is not ready"))
	}
	r, err := store.NewReader(ctx, SourceKey(a.UUID, filename))
	if errors.Is(err, ErrObjectNotFound) {
		return nil, nil, api.MakeNotFound(errors.New("package structure not found"))
	}
	if err != nil {
		return nil, nil, s.internalError("open package structure", err)
	}
	return &api.StructureResult{ContentType: "text/plain; charset=utf-8", ContentLength: r.Size()}, r, nil
}
