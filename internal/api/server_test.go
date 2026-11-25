// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-logr/logr"

	gen "github.com/jraddaoui/provenant-passage/internal/api/gen/aips"
)

type (
	httpService struct {
		gen.Service
		uploaded bool
		reader   *httpReader
	}
	httpReader struct {
		*strings.Reader
		closed bool
	}
)

func (r *httpReader) Close() error { r.closed = true; return nil }
func (s *httpService) Upload(ctx context.Context, p *gen.UploadPayload, r io.ReadCloser) (*gen.UploadResult, error) {
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if string(data) != "streamed body" || p.Profile != "files_and_metadata" {
		return nil, errors.New("stream contract changed")
	}
	s.uploaded = true
	return &gen.UploadResult{UUID: "11111111-2222-4333-8444-555555555555"}, nil
}

func (s *httpService) Download(
	ctx context.Context,
	p *gen.DownloadPayload,
) (*gen.DownloadResult, io.ReadCloser, error) {
	s.reader = &httpReader{Reader: strings.NewReader("zip bytes")}
	return &gen.DownloadResult{
		ContentType:        "application/zip",
		ContentLength:      9,
		ContentDisposition: `attachment; filename="object.zip"`,
	}, s.reader, nil
}

func (s *httpService) Show(ctx context.Context, p *gen.ShowPayload) (*gen.ProvenantAip, error) {
	return nil, gen.MakeNotFound(errors.New("not found"))
}

func (s *httpService) Monitor(ctx context.Context, stream gen.MonitorServerStream) error {
	defer stream.Close()
	msg := "Hello"
	return stream.SendWithContext(ctx, &gen.AIPEvent{Value: gen.NewValuePingEvent(&gen.PingEvent{Message: &msg})})
}

func TestGeneratedHTTPStreamingAndValidation(t *testing.T) {
	s := &httpService{}
	h := Handler(s, logr.Discard())
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"POST", "/aips/upload", "streamed body", 202},
		{"POST", "/aips/upload?profile=b", "streamed body", 400},
		{"GET", "/aips/not-a-uuid", "", 400},
		{"GET", "/aips/11111111-2222-4333-8444-555555555555", "", 404},
		{"GET", "/aips/11111111-2222-4333-8444-555555555555/download", "", 200},
		{"GET", "/aips/monitor", "", 200},
	} {
		t.Run(tc.path, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "multipart/form-data; boundary=test")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if strings.HasSuffix(tc.path, "/download") &&
				(w.Body.String() != "zip bytes" || !s.reader.closed || w.Header().Get("Content-Type") != "application/zip") {
				t.Fatal("download not streamed/closed correctly")
			}
			if strings.HasSuffix(tc.path, "/monitor") &&
				(!strings.Contains(w.Body.String(), "data:") || !strings.Contains(w.Body.String(), "ping_event")) {
				t.Fatal("missing SSE ping")
			}
		})
	}
	if !s.uploaded {
		t.Fatal("upload did not reach service")
	}
}
