// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package aips

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace/noop"

	api "github.com/jraddaoui/provenant-passage/internal/api/gen/aips"
	"github.com/jraddaoui/provenant-passage/internal/enums"
	"github.com/jraddaoui/provenant-passage/internal/event"
	"github.com/jraddaoui/provenant-passage/internal/persistence"
)

type monitorStream struct{ events chan *api.AIPEvent }

func (s *monitorStream) Send(e *api.AIPEvent) error {
	return s.SendWithContext(context.Background(), e)
}

func (s *monitorStream) SendWithContext(ctx context.Context, e *api.AIPEvent) error {
	select {
	case s.events <- e:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *monitorStream) Close() error { return nil }

func TestLiveRedisMonitorFanout(t *testing.T) {
	address := os.Getenv("PASSAGE_TEST_REDIS_URL")
	if address == "" {
		t.Skip("PASSAGE_TEST_REDIS_URL required for live Redis integration")
	}
	channel := "passage-test-" + uuid.NewString()
	newEvents := func() event.Service[*api.AIPEvent] {
		t.Helper()
		s, err := event.NewServiceRedis(logr.Discard(), noop.NewTracerProvider(), address, channel, &EventSerializer{})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	publisher := newEvents()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	streams := []*monitorStream{{events: make(chan *api.AIPEvent, 8)}, {events: make(chan *api.AIPEvent, 8)}}
	done := make(chan error, 2)
	for _, stream := range streams {
		svc := &Service{Events: newEvents(), Logger: logr.Discard()}
		go func() { done <- svc.Monitor(ctx, stream) }()
		select {
		case e := <-stream.events:
			if e.Value.Kind() != api.ValueKindPingEvent {
				t.Fatal("missing initial ping")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("monitor did not subscribe")
		}
	}
	a := &persistence.AIP{
		UUID:      uuid.New(),
		Name:      "live.tar",
		Profile:   enums.ProfileFilesAndMetadata,
		Status:    enums.StatusCompleted,
		CreatedAt: time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC),
	}
	PublishUpdated(ctx, publisher, a)
	for _, stream := range streams {
		select {
		case e := <-stream.events:
			v, ok := e.Value.AsAipUpdatedEvent()
			if !ok || v.UUID != a.UUID.String() || v.Item.Status != "completed" {
				t.Fatal("update did not fan out")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("update missing from independent monitor instance")
		}
	}
	cancel()
	for range streams {
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("monitor failed to close")
		}
	}
	opts, err := redis.ParseURL(address)
	if err != nil {
		t.Fatal(err)
	}
	client := redis.NewClient(opts)
	defer client.Close()
	deadline := time.Now().Add(3 * time.Second)
	for {
		subs, err := client.PubSubNumSub(t.Context(), channel).Result()
		if err != nil {
			t.Fatal(err)
		}
		if subs[channel] == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Redis subscriptions leaked")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
