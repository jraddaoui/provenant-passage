// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package aips

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/jraddaoui/provenant-passage/internal/enums"
	"github.com/jraddaoui/provenant-passage/internal/persistence"
)

var ErrStartRejected = errors.New("workflow start definitely rejected")

type ExecutionState string

const (
	ExecutionAbsent    ExecutionState = "absent"
	ExecutionRunning   ExecutionState = "running"
	ExecutionCompleted ExecutionState = "completed"
	ExecutionFailed    ExecutionState = "failed"
	ExecutionCanceled  ExecutionState = "canceled"
)

type ImportRequest struct {
	UUID            uuid.UUID
	Name, SourceKey string
	Profile         enums.Profile
	CreatedAt       time.Time
}

func (r ImportRequest) WorkflowID() string { return "import-aip-" + r.UUID.String() }
func Request(a *persistence.AIP) ImportRequest {
	return ImportRequest{
		UUID:      a.UUID,
		Name:      a.Name,
		SourceKey: SourceKey(a.UUID, a.Name),
		Profile:   a.Profile,
		CreatedAt: a.CreatedAt,
	}
}

type WorkflowClient interface {
	Start(context.Context, ImportRequest) error
	Inspect(context.Context, string) (ExecutionState, error)
}
