// SPDX-FileCopyrightText: Artefactual Systems Inc. and Enduro contributors
// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"github.com/jraddaoui/provenant-passage/internal/enums"
	"github.com/jraddaoui/provenant-passage/internal/persistence"
	"github.com/jraddaoui/provenant-passage/internal/persistence/ent/db"
	"github.com/jraddaoui/provenant-passage/internal/persistence/ent/db/aip"
)

type client struct{ ent *db.Client }

func New(conn *sql.DB) persistence.Service {
	return &client{ent: db.NewClient(db.Driver(entsql.OpenDB(dialect.MySQL, conn)))}
}

func dbError(err error) error {
	switch {
	case err == nil:
		return nil
	case db.IsNotFound(err):
		return fmt.Errorf("%w: %v", persistence.ErrNotFound, err)
	case db.IsConstraintError(err), db.IsValidationError(err):
		return fmt.Errorf("%w: %v", persistence.ErrNotValid, err)
	default:
		return err
	}
}

func convert(a *db.AIP) *persistence.AIP {
	return &persistence.AIP{
		UUID: a.UUID, Name: a.Name, Status: a.Status,
		CreatedAt: a.CreatedAt.UTC(), Profile: a.Profile, OCFLObjectKey: a.OcflObjectKey,
	}
}

func (c *client) Create(ctx context.Context, a *persistence.AIP) error {
	if a.UUID == uuid.Nil || a.Name == "" {
		return persistence.ErrNotValid
	}
	p, err := enums.ParseProfileWithDefault(string(a.Profile))
	if err != nil {
		return fmt.Errorf("%w: %v", persistence.ErrNotValid, err)
	}
	row, err := c.ent.AIP.Create().SetUUID(a.UUID).SetName(a.Name).
		SetProfile(p).SetStatus(enums.StatusQueued).
		SetCreatedAt(time.Now().UTC().Truncate(time.Second)).Save(ctx)
	if err != nil {
		return dbError(err)
	}
	*a = *convert(row)
	return nil
}

func (c *client) Read(ctx context.Context, id uuid.UUID) (*persistence.AIP, error) {
	a, err := c.ent.AIP.Query().Where(aip.UUID(id)).Only(ctx)
	if err != nil {
		return nil, dbError(err)
	}
	return convert(a), nil
}

func (c *client) List(ctx context.Context, limit, offset int) ([]*persistence.AIP, *persistence.PageResult, error) {
	limit, offset = persistence.Page(limit, offset)
	tx, err := c.ent.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }() // Read-only transaction; safe after commit.
	total, err := tx.AIP.Query().Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	rows, err := tx.AIP.Query().Order(db.Desc(aip.FieldID)).Limit(limit).Offset(offset).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	items := make([]*persistence.AIP, 0, len(rows))
	for _, row := range rows {
		items = append(items, convert(row))
	}
	return items, &persistence.PageResult{Limit: limit, Offset: offset, Total: total}, nil
}

func (c *client) Update(ctx context.Context, id uuid.UUID, status enums.Status, key string) (*persistence.AIP, error) {
	if _, err := enums.ParseStatus(string(status)); err != nil {
		return nil, persistence.ErrNotValid
	}
	if (status == enums.StatusCompleted) != (key != "") || status == enums.StatusQueued {
		return nil, persistence.ErrNotValid
	}
	// Atomic conditional writes prevent retries from downgrading terminal state.
	q := c.ent.AIP.Update().
		Where(aip.UUID(id), aip.StatusIn(enums.StatusQueued, enums.StatusProcessing)).
		SetStatus(status)
	if key != "" {
		q.SetOcflObjectKey(key)
	} else {
		q.ClearOcflObjectKey()
	}
	if _, err := q.Save(ctx); err != nil {
		return nil, dbError(err)
	}
	return c.Read(ctx, id)
}

func (c *client) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := c.ent.AIP.Delete().Where(aip.UUID(id)).Exec(ctx)
	return dbError(err)
}

func (c *client) Active(ctx context.Context, limit, offset int) ([]*persistence.AIP, error) {
	rows, err := c.ent.AIP.Query().Where(aip.StatusIn(enums.StatusQueued, enums.StatusProcessing)).
		Order(db.Asc(aip.FieldID)).Limit(limit).Offset(offset).All(ctx)
	if err != nil {
		return nil, dbError(err)
	}
	items := make([]*persistence.AIP, 0, len(rows))
	for _, row := range rows {
		items = append(items, convert(row))
	}
	return items, nil
}
