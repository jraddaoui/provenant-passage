// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package aips

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/textproto"
	"testing"
	"time"

	"github.com/google/uuid"

	api "github.com/jraddaoui/provenant-passage/internal/api/gen/aips"
	"github.com/jraddaoui/provenant-passage/internal/enums"
	"github.com/jraddaoui/provenant-passage/internal/event"
	"github.com/jraddaoui/provenant-passage/internal/persistence"
)

var errInjected = errors.New("errInjected failure")

type testStore struct {
	Store
	data                          map[string][]byte
	writeErr, closeErr, deleteErr error
	cleanupCanceled               bool
}
type testWriter struct {
	bytes.Buffer
	store *testStore
	key   string
}

func (w *testWriter) Write(p []byte) (int, error) {
	if w.store.writeErr != nil {
		return 0, w.store.writeErr
	}
	return w.Buffer.Write(p)
}
func (w *testWriter) Close() error { w.store.data[w.key] = w.Bytes(); return w.store.closeErr }
func (s *testStore) NewWriter(ctx context.Context, key string) (io.WriteCloser, error) {
	return &testWriter{store: s, key: key}, nil
}

func (s *testStore) Delete(ctx context.Context, key string) error {
	s.cleanupCanceled = ctx.Err() != nil
	if s.deleteErr != nil {
		return s.deleteErr
	}
	delete(s.data, key)
	return nil
}

type testPersistence struct {
	persistence.Service
	rows                 map[uuid.UUID]*persistence.AIP
	createErr, deleteErr error
}

func (s *testPersistence) Create(ctx context.Context, a *persistence.AIP) error {
	if s.createErr != nil {
		return s.createErr
	}
	a.CreatedAt = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	a.Status = enums.StatusQueued
	copy := *a
	s.rows[a.UUID] = &copy
	return nil
}

func (s *testPersistence) Read(ctx context.Context, id uuid.UUID) (*persistence.AIP, error) {
	a, ok := s.rows[id]
	if !ok {
		return nil, persistence.ErrNotFound
	}
	return a, nil
}

func (s *testPersistence) Delete(ctx context.Context, id uuid.UUID) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	delete(s.rows, id)
	return nil
}

type testWorkflows struct {
	startErr, inspectErr error
	state                ExecutionState
	request              ImportRequest
	cancel               context.CancelFunc
}

func (w *testWorkflows) Start(ctx context.Context, r ImportRequest) error {
	w.request = r
	if w.cancel != nil {
		w.cancel()
	}
	return w.startErr
}

func (w *testWorkflows) Inspect(ctx context.Context, id string) (ExecutionState, error) {
	return w.state, w.inspectErr
}

type testEvents struct {
	event.Service[*api.AIPEvent]
	events []*api.AIPEvent
}

func (e *testEvents) PublishEvent(ctx context.Context, v *api.AIPEvent) {
	e.events = append(e.events, v)
}

func setupUpload() (*Service, *testStore, *testPersistence, *testWorkflows, *testEvents) {
	b := &testStore{data: map[string][]byte{}}
	p := &testPersistence{rows: map[uuid.UUID]*persistence.AIP{}}
	w := &testWorkflows{}
	e := &testEvents{}
	return &Service{Source: b, Output: b, Persistence: p, Workflows: w, Events: e}, b, p, w, e
}

func tarBytes(t *testing.T) []byte {
	t.Helper()
	var data bytes.Buffer
	tw := tar.NewWriter(&data)
	if err := tw.WriteHeader(&tar.Header{Name: "bag/file.txt", Mode: 0o600, Size: 4}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("data")); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func uploadBody(t *testing.T, names []string, payload []byte) (*api.UploadPayload, io.ReadCloser) {
	t.Helper()
	var data bytes.Buffer
	mw := multipart.NewWriter(&data)
	for _, name := range names {
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", `form-data; name="file"; filename="`+name+`"`)
		p, err := mw.CreatePart(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = p.Write(payload); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &api.UploadPayload{
		ContentType: mw.FormDataContentType(),
		Profile:     "files_and_metadata",
	}, io.NopCloser(
		bytes.NewReader(data.Bytes()),
	)
}

func TestUploadPreservesIdentityAndTime(t *testing.T) {
	s, b, p, w, e := setupUpload()
	archive := tarBytes(t)
	payload, body := uploadBody(t, []string{"Original archive.tar"}, archive)
	result, err := s.Upload(t.Context(), payload, body)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.MustParse(result.UUID)
	row := p.rows[id]
	if w.request.Name != "Original archive.tar" || w.request.CreatedAt != row.CreatedAt ||
		w.request.Profile != enums.ProfileFilesAndMetadata {
		t.Fatalf("request lost identity: %+v", w.request)
	}
	if !bytes.Equal(b.data[SourceKey(id, row.Name)], archive) {
		t.Fatal("archive changed")
	}
	if len(e.events) != 1 || e.events[0].Value.Kind() != api.ValueKindAipCreatedEvent {
		t.Fatal("missing created event")
	}
}

func TestUploadFailures(t *testing.T) {
	for _, scenario := range []string{"profile", "empty", "two files", "unsafe name", "extension", "content", "copy", "close", "database", "start", "cancel", "cleanup"} {
		t.Run(scenario, func(t *testing.T) {
			s, b, p, w, e := setupUpload()
			names := []string{"source.tar"}
			archive := tarBytes(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			switch scenario {
			case "empty":
				names = nil
			case "two files":
				names = append(names, "second.tar")
			case "unsafe name":
				names = []string{"../source.tar"}
			case "extension":
				names = []string{"source.zip"}
			case "content":
				archive = []byte("not an archive")
			case "copy":
				b.writeErr = errInjected
			case "close":
				b.closeErr = errInjected
			case "database":
				p.createErr = persistence.ErrNotValid
			case "start":
				w.startErr = ErrStartRejected
			case "cancel":
				w.startErr = ErrStartRejected
				w.cancel = cancel
			case "cleanup":
				w.startErr = ErrStartRejected
				b.deleteErr = errInjected
			}
			payload, body := uploadBody(t, names, archive)
			if scenario == "profile" {
				payload.Profile = "b"
			}
			if _, err := s.Upload(ctx, payload, body); err == nil {
				t.Fatal("accepted invalid upload")
			}
			if len(p.rows) != 0 || len(e.events) != 0 || (scenario != "cleanup" && len(b.data) != 0) {
				t.Fatalf("leaked state: rows=%d objects=%d events=%d", len(p.rows), len(b.data), len(e.events))
			}
			if b.cleanupCanceled {
				t.Fatal("cleanup used canceled context")
			}
		})
	}
}

func TestAmbiguousWorkflowStart(t *testing.T) {
	for _, resolved := range []bool{true, false} {
		t.Run(map[bool]string{true: "accepted", false: "unresolved"}[resolved], func(t *testing.T) {
			s, b, p, w, e := setupUpload()
			w.startErr = context.DeadlineExceeded
			if resolved {
				w.state = ExecutionRunning
			} else {
				w.inspectErr = errInjected
			}
			payload, body := uploadBody(t, []string{"source.tar"}, tarBytes(t))
			_, err := s.Upload(t.Context(), payload, body)
			if resolved && err != nil {
				t.Fatal(err)
			}
			if !resolved && err == nil {
				t.Fatal("unresolved start reported accepted")
			}
			if len(p.rows) != 1 || len(b.data) != 1 {
				t.Fatal("ambiguous start was rolled back")
			}
			if resolved && len(e.events) != 1 {
				t.Fatal("missing created event")
			}
		})
	}
}

func TestArchiveBasenames(t *testing.T) {
	for _, name := range []string{"", ".", "..", "../a.tar", `a\b.tar`, "/a.tar", "a\x00.tar", "a\n.tar"} {
		if err := ValidateName(name); err == nil {
			t.Errorf("accepted %q", name)
		}
	}
	for _, name := range []string{"original name.tar", "archive.7z", "a-b_1.tar"} {
		if err := ValidateName(name); err != nil {
			t.Errorf("rejected %q: %v", name, err)
		}
	}
}
