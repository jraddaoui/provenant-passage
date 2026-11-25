// SPDX-FileCopyrightText: Artefactual Systems Inc. and Enduro contributors
// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package aips

import (
	"context"
	"encoding/json"
	"time"

	api "github.com/jraddaoui/provenant-passage/internal/api/gen/aips"
	"github.com/jraddaoui/provenant-passage/internal/api/gen/http/aips/client"
	"github.com/jraddaoui/provenant-passage/internal/api/gen/http/aips/server"
)

type EventSerializer struct{}

func (*EventSerializer) Marshal(e *api.AIPEvent) ([]byte, error) {
	return json.Marshal(server.NewMonitorResponseBody(e))
}

func (*EventSerializer) Unmarshal(data []byte) (*api.AIPEvent, error) {
	var body client.MonitorResponseBody
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	if err := client.ValidateMonitorResponseBody(&body); err != nil {
		return nil, err
	}
	return client.NewMonitorAIPEventOK(&body), nil
}

func (s *Service) Monitor(ctx context.Context, stream api.MonitorServerStream) error {
	defer stream.Close()
	sub, err := s.Events.Subscribe(ctx)
	if err != nil {
		return s.internalError("subscribe to AIP events", err)
	}
	defer sub.Close()
	message := "Hello"
	ping := &api.AIPEvent{Value: api.NewValuePingEvent(&api.PingEvent{Message: &message})}
	if err := stream.SendWithContext(ctx, ping); err != nil {
		return nil
	}
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := stream.SendWithContext(ctx, ping); err != nil {
				return nil
			}
		case e, ok := <-sub.C():
			if !ok || e == nil {
				return nil
			}
			if err := stream.SendWithContext(ctx, e); err != nil {
				return nil
			}
		}
	}
}
