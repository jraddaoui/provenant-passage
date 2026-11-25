// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package workflow

import (
	"errors"
	"testing"

	temporalapi_enums "go.temporal.io/api/enums/v1"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
)

func TestFailedSessionHeartbeatTimeoutRestartsPipeline(t *testing.T) {
	// Observed in the live worker-kill experiment: the creation activity fails
	// first, then the in-flight importer returns a heartbeat timeout, not cancellation.
	timeout := temporalsdk_temporal.NewTimeoutError(temporalapi_enums.TIMEOUT_TYPE_HEARTBEAT, nil)
	failed := &temporalsdk_workflow.SessionInfo{SessionState: temporalsdk_workflow.SessionStateFailed}
	err := sessionResult(failed, timeout)
	if !errors.Is(err, temporalsdk_workflow.ErrSessionFailed) || !errors.Is(err, timeout) {
		t.Fatal("lost worker must restart with original timeout retained", err)
	}
	open := &temporalsdk_workflow.SessionInfo{SessionState: temporalsdk_workflow.SessionStateOpen}
	if errors.Is(sessionResult(open, timeout), temporalsdk_workflow.ErrSessionFailed) {
		t.Fatal("ordinary activity timeout misclassified as lost session")
	}
}
