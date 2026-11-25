// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/jraddaoui/provenant-passage/internal/enums"
)

var (
	ErrNotFound = errors.New("not found")
	ErrNotValid = errors.New("invalid data")
)

type AIP struct {
	UUID          uuid.UUID
	Name          string
	Status        enums.Status
	CreatedAt     time.Time
	Profile       enums.Profile
	OCFLObjectKey *string
}

type PageResult struct{ Limit, Offset, Total int }

//go:generate go tool mockgen -typed -destination=fake/mock_persistence.go -package=fake . Service
type Service interface {
	Create(context.Context, *AIP) error
	Read(context.Context, uuid.UUID) (*AIP, error)
	List(context.Context, int, int) ([]*AIP, *PageResult, error)
	Update(context.Context, uuid.UUID, enums.Status, string) (*AIP, error)
	Delete(context.Context, uuid.UUID) error
	Active(context.Context, int, int) ([]*AIP, error)
}
