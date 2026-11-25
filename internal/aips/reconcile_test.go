// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package aips

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jraddaoui/provenant-passage/internal/enums"
	"github.com/jraddaoui/provenant-passage/internal/persistence"
)

type reconcilePersistence struct {
	*testPersistence
	updates []enums.Status
}

func TestReconcileDoesNotRaceFreshUpload(t *testing.T) {
	s, _, base, w, _ := setupUpload()
	p := &reconcilePersistence{testPersistence: base}
	s.Persistence = p
	id := uuid.New()
	p.rows[id] = &persistence.AIP{
		UUID: id, Name: "upload.tar", Status: enums.StatusQueued,
		Profile: enums.ProfileFilesAndMetadata, CreatedAt: time.Now().UTC().Truncate(time.Second),
	}
	w.state = ExecutionAbsent
	if err := s.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	if w.request.UUID != uuid.Nil {
		t.Fatal("reconciler raced the in-flight upload initialization")
	}
}

func (p *reconcilePersistence) Active(ctx context.Context, limit, offset int) ([]*persistence.AIP, error) {
	if offset > 0 {
		return nil, nil
	}
	items := []*persistence.AIP{}
	for _, a := range p.rows {
		items = append(items, a)
	}
	return items, nil
}

func (p *reconcilePersistence) Update(
	ctx context.Context,
	id uuid.UUID,
	status enums.Status,
	key string,
) (*persistence.AIP, error) {
	p.updates = append(p.updates, status)
	p.rows[id].Status = status
	return p.rows[id], nil
}

func TestReconcileInterruptedStartsAndTerminalFailure(t *testing.T) {
	for _, tc := range []struct {
		name    string
		initial enums.Status
		state   ExecutionState
		want    enums.Status
		starts  bool
	}{
		{"queued without workflow", enums.StatusQueued, ExecutionAbsent, "", true},
		{"queued running", enums.StatusQueued, ExecutionRunning, "", false},
		{"failed terminal update", enums.StatusProcessing, ExecutionFailed, enums.StatusError, false},
		{"canceled", enums.StatusProcessing, ExecutionCanceled, enums.StatusCanceled, false},
		{"missing processing workflow", enums.StatusProcessing, ExecutionAbsent, enums.StatusError, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, base, w, _ := setupUpload()
			p := &reconcilePersistence{testPersistence: base}
			s.Persistence = p
			id := uuid.New()
			created := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
			p.rows[id] = &persistence.AIP{
				UUID:      id,
				Name:      "source.tar",
				Profile:   enums.ProfileOriginalAndExtracted,
				Status:    tc.initial,
				CreatedAt: created,
			}
			w.state = tc.state
			if err := s.Reconcile(t.Context()); err != nil {
				t.Fatal(err)
			}
			if tc.starts &&
				(w.request.UUID != id || w.request.CreatedAt != created || w.request.Profile != enums.ProfileOriginalAndExtracted) {
				t.Fatal("reconciliation changed import identity")
			}
			if !tc.starts && w.request.UUID != uuid.Nil {
				t.Fatal("unexpected workflow start")
			}
			if tc.want != "" && (len(p.updates) != 1 || p.updates[0] != tc.want) {
				t.Fatalf("terminal status: %v", p.updates)
			}
		})
	}
}
