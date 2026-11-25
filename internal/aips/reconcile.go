// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package aips

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jraddaoui/provenant-passage/internal/enums"
)

// Reconcile repairs interrupted starts and terminal writes using Temporal's
// authoritative execution status. Each sweep and external operation is bounded.
// Call at startup and periodically; it never deletes source evidence.
func (s *Service) Reconcile(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	// Snapshot IDs before changing statuses so offset paging cannot skip rows.
	const pageSize = 100
	var requests []ImportRequest
	var statuses []enums.Status
	for offset := 0; offset < 10000; offset += pageSize {
		items, err := s.Persistence.Active(ctx, pageSize, offset)
		if err != nil {
			return err
		}
		for _, a := range items {
			if a.Status == enums.StatusQueued && time.Since(a.CreatedAt) < time.Minute {
				continue // Allow the bounded upload initialization to finish first.
			}
			requests = append(requests, Request(a))
			statuses = append(statuses, a.Status)
		}
		if len(items) < pageSize {
			break
		}
	}
	var errs error
	for i, r := range requests {
		if ctx.Err() != nil {
			return errors.Join(errs, ctx.Err())
		}
		c, stop := context.WithTimeout(ctx, 3*time.Second)
		state, err := s.Workflows.Inspect(c, r.WorkflowID())
		stop()
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("inspect %s: %w", r.UUID, err))
			continue
		}
		var status enums.Status
		switch state {
		case ExecutionRunning:
			continue
		case ExecutionAbsent:
			if statuses[i] == enums.StatusQueued {
				c, stop := context.WithTimeout(ctx, 3*time.Second)
				err = s.Workflows.Start(c, r)
				stop()
				if errors.Is(err, ErrStartRejected) {
					status = enums.StatusError
				} else {
					if err != nil {
						errs = errors.Join(errs, err)
					}
					continue
				}
			} else {
				status = enums.StatusError
			}
		case ExecutionCanceled:
			status = enums.StatusCanceled
		default:
			status = enums.StatusError
		}
		a, err := s.Persistence.Update(ctx, r.UUID, status, "")
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		PublishUpdated(ctx, s.Events, a)
	}
	return errs
}
