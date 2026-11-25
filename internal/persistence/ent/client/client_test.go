// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package client_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"github.com/jraddaoui/provenant-passage/internal/db"
	"github.com/jraddaoui/provenant-passage/internal/enums"
	"github.com/jraddaoui/provenant-passage/internal/persistence"
	"github.com/jraddaoui/provenant-passage/internal/persistence/ent/client"
)

func TestMySQLContract(t *testing.T) {
	dsn := os.Getenv("PASSAGE_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("PASSAGE_TEST_MYSQL_DSN is required for MySQL integration")
	}
	ctx := t.Context()
	// Use a temporary schema: queued fixture rows must never be discovered by
	// the running application's workflow reconciler. The DSN needs CREATE/DROP.
	settings, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if settings.DBName != "" {
		t.Fatal("PASSAGE_TEST_MYSQL_DSN must omit the database name; tests create and drop an isolated schema")
	}
	admin, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	name := "passage_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE `"+name+"`"); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(cleanup, "DROP DATABASE `"+name+"`"); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	settings.DBName = name
	dsn = settings.FormatDSN()
	for range 2 {
		if err := db.Migrate(ctx, dsn); err != nil {
			t.Fatal(err)
		}
	}
	conn, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	svc := client.New(conn)
	first := &persistence.AIP{UUID: uuid.New(), Name: "original name.tar", Profile: enums.ProfileFilesAndMetadata}
	second := &persistence.AIP{UUID: uuid.New(), Name: "other.7z", Profile: enums.ProfileOriginalAndExtracted}
	for _, item := range []*persistence.AIP{first, second} {
		if err := svc.Create(ctx, item); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { svc.Delete(ctx, item.UUID) })
		if item.CreatedAt.Nanosecond() != 0 || item.CreatedAt.Location() != time.UTC {
			t.Fatal("creation time must be whole-second UTC")
		}
		if item.Status != enums.StatusQueued {
			t.Fatal("initial status")
		}
	}
	got, err := svc.Read(ctx, first.UUID)
	if err != nil || !got.CreatedAt.Equal(first.CreatedAt) {
		t.Fatalf("persisted time mismatch: %v", err)
	}
	if err := svc.Create(ctx, first); !errors.Is(err, persistence.ErrNotValid) {
		t.Fatalf("duplicate UUID: %v", err)
	}
	for _, item := range []*persistence.AIP{
		{UUID: uuid.Nil, Name: "bad.tar"},
		{UUID: uuid.New(), Name: "bad.tar", Profile: enums.Profile("b")},
	} {
		if err := svc.Create(ctx, item); !errors.Is(err, persistence.ErrNotValid) {
			t.Fatalf("accepted invalid AIP: %v", err)
		}
	}
	items, page, err := svc.List(ctx, 1, 0)
	if err != nil || len(items) != 1 || items[0].UUID != second.UUID || page.Total < 2 || page.Limit != 1 {
		t.Fatalf("latest first/page: %+v %v", page, err)
	}
	key := "aips/" + first.UUID.String() + "/object.zip"
	if _, err := svc.Update(ctx, first.UUID, enums.StatusCompleted, ""); !errors.Is(err, persistence.ErrNotValid) {
		t.Fatalf("completed without key: %v", err)
	}
	if _, err := svc.Update(ctx, first.UUID, enums.StatusCompleted, key); err != nil {
		t.Fatal(err)
	}
	for _, status := range []enums.Status{enums.StatusProcessing, enums.StatusError, enums.StatusCanceled} {
		got, err := svc.Update(ctx, first.UUID, status, "")
		if err != nil || got.Status != enums.StatusCompleted || got.OCFLObjectKey == nil || *got.OCFLObjectKey != key ||
			!got.CreatedAt.Equal(first.CreatedAt) {
			t.Fatalf("terminal state regressed: %+v %v", got, err)
		}
	}
	if _, err := svc.Read(ctx, uuid.New()); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("unknown UUID: %v", err)
	}
}
