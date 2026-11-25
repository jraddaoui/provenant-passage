// SPDX-FileCopyrightText: Artefactual Systems Inc. and Enduro contributors
// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package workflow

import (
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/artefactual-sdps/temporal-activities/archivezip"
	"github.com/artefactual-sdps/temporal-activities/bucketdownload"
	"github.com/artefactual-sdps/temporal-activities/bucketupload"
	"github.com/artefactual-sdps/temporal-activities/removepaths"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/jraddaoui/provenant-passage/internal/aips"
	"github.com/jraddaoui/provenant-passage/internal/enums"
	"github.com/jraddaoui/provenant-passage/internal/workflow/activities"
)

type Options struct {
	WorkflowQueue, ActivityQueue                                                       string
	SessionCreationTimeout, SessionExecutionTimeout, ActivityTimeout, HeartbeatTimeout time.Duration
}
type Workflow struct{ Options Options }

// A running activity may time out after its session has failed. ErrSessionFailed
// is only returned automatically when scheduling a subsequent activity, so also
// inspect the owning session before deciding whether the whole pipeline retries.
func sessionResult(info *temporalsdk_workflow.SessionInfo, err error) error {
	if info != nil && info.SessionState == temporalsdk_workflow.SessionStateFailed {
		return errors.Join(temporalsdk_workflow.ErrSessionFailed, err)
	}
	return err
}

func (w *Workflow) databaseContext(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		TaskQueue:              w.Options.WorkflowQueue,
		StartToCloseTimeout:    30 * time.Second,
		ScheduleToCloseTimeout: 2 * time.Minute,
		RetryPolicy:            &temporalsdk_temporal.RetryPolicy{InitialInterval: time.Second, MaximumAttempts: 5},
	})
}

func (w *Workflow) Execute(ctx temporalsdk_workflow.Context, r aips.ImportRequest) (err error) {
	dbctx := w.databaseContext(ctx)
	defer func() {
		if err == nil {
			return
		}
		disconnected, _ := temporalsdk_workflow.NewDisconnectedContext(ctx)
		disconnected = w.databaseContext(disconnected)
		status := enums.StatusError
		if ctx.Err() != nil {
			status = enums.StatusCanceled
		}
		updateErr := temporalsdk_workflow.ExecuteActivity(disconnected, activities.UpdateStatusName,
			activities.StatusParams{UUID: r.UUID, Status: status}).Get(disconnected, nil)
		// Keep a visible workflow failure if the terminal database write exhausts
		// its retries. The API reconciler repairs this after DB recovery.
		if updateErr != nil {
			err = errors.Join(err, fmt.Errorf("terminal AIP update: %w", updateErr))
		}
	}()
	if err = temporalsdk_workflow.ExecuteActivity(dbctx, activities.UpdateStatusName,
		activities.StatusParams{UUID: r.UUID, Status: enums.StatusProcessing}).Get(dbctx, nil); err != nil {
		return err
	}
	var key string
	for attempt := 1; attempt <= 3; attempt++ {
		key, err = w.session(ctx, r, attempt)
		if err == nil {
			break
		}
		if ctx.Err() != nil {
			return err
		}
		if !errors.Is(err, temporalsdk_workflow.ErrSessionFailed) && !temporalsdk_temporal.IsCanceledError(err) {
			return err
		}
		if attempt < 3 {
			temporalsdk_workflow.GetLogger(ctx).
				Warn("Session lost; redownloading into a new workspace", "attempt", attempt)
			if sleepErr := temporalsdk_workflow.Sleep(ctx, 5*time.Second); sleepErr != nil {
				return sleepErr
			}
		}
	}
	if err != nil {
		return err
	}
	return temporalsdk_workflow.ExecuteActivity(dbctx, activities.UpdateStatusName,
		activities.StatusParams{UUID: r.UUID, Status: enums.StatusCompleted, Key: key}).Get(dbctx, nil)
}

func (w *Workflow) session(
	ctx temporalsdk_workflow.Context,
	r aips.ImportRequest,
	attempt int,
) (key string, err error) {
	options := temporalsdk_workflow.ActivityOptions{
		TaskQueue:              w.Options.ActivityQueue,
		StartToCloseTimeout:    w.Options.ActivityTimeout,
		ScheduleToCloseTimeout: 3 * w.Options.ActivityTimeout,
		HeartbeatTimeout:       w.Options.HeartbeatTimeout,
		WaitForCancellation:    true,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumInterval: 10 * time.Second,
			MaximumAttempts: 3,
		},
	}
	ctx = temporalsdk_workflow.WithActivityOptions(ctx, options)
	sess, err := temporalsdk_workflow.CreateSession(ctx, &temporalsdk_workflow.SessionOptions{
		CreationTimeout: w.Options.SessionCreationTimeout, ExecutionTimeout: w.Options.SessionExecutionTimeout,
		HeartbeatTimeout: w.Options.HeartbeatTimeout / 2,
	})
	if err != nil {
		return "", err
	}
	defer temporalsdk_workflow.CompleteSession(sess)
	var workspace string
	defer func() {
		info := temporalsdk_workflow.GetSessionInfo(sess)
		err = sessionResult(info, err)
		if workspace == "" || info == nil || info.SessionState != temporalsdk_workflow.SessionStateOpen {
			return
		}
		// Disconnect cancellation while preserving the owning session. Never
		// send an abandoned path to an arbitrary worker after session loss.
		cleanup, _ := temporalsdk_workflow.NewDisconnectedContext(sess)
		cleanup = temporalsdk_workflow.WithActivityOptions(cleanup, temporalsdk_workflow.ActivityOptions{
			StartToCloseTimeout:    time.Minute,
			ScheduleToCloseTimeout: 2 * time.Minute,
			HeartbeatTimeout:       20 * time.Second,
			RetryPolicy:            &temporalsdk_temporal.RetryPolicy{MaximumAttempts: 1},
			WaitForCancellation:    true,
		})
		if err != nil {
			diagnosticErr := temporalsdk_workflow.ExecuteActivity(cleanup, activities.RetainReportsName,
				activities.ImportParams{Workspace: workspace, Request: r}).Get(cleanup, nil)
			if diagnosticErr != nil {
				err = errors.Join(err, fmt.Errorf("retain diagnostic reports: %w", diagnosticErr))
			}
		}
		// The reusable removepaths activity does not heartbeat. Its execution
		// remains bounded and routed to this session's owner.
		cleanup = temporalsdk_workflow.WithHeartbeatTimeout(cleanup, 0)
		cleanupErr := temporalsdk_workflow.ExecuteActivity(cleanup, removepaths.Name,
			&removepaths.Params{Paths: []string{workspace}}).Get(cleanup, nil)
		if cleanupErr != nil {
			temporalsdk_workflow.GetLogger(ctx).Error("Local scratch cleanup failed", "error", cleanupErr)
		}
	}()
	short := temporalsdk_workflow.WithHeartbeatTimeout(sess, 0)
	if err = temporalsdk_workflow.ExecuteActivity(short, activities.CreateWorkspaceName,
		activities.WorkspaceParams{UUID: r.UUID, Attempt: attempt}).Get(short, &workspace); err != nil {
		return "", err
	}
	if err = temporalsdk_workflow.ExecuteActivity(sess, bucketdownload.Name,
		&bucketdownload.Params{DirPath: workspace, FileName: r.Name, Key: r.SourceKey}).Get(sess, nil); err != nil {
		return "", err
	}
	p := activities.ImportParams{Workspace: workspace, Request: r}
	// CLI/content failures are terminal. Session loss still restarts the entire
	// pipeline with a new workspace and the original request time and profile.
	importCtx := temporalsdk_workflow.WithRetryPolicy(sess, temporalsdk_temporal.RetryPolicy{MaximumAttempts: 1})
	if err = temporalsdk_workflow.ExecuteActivity(importCtx, activities.RunImporterName, p).
		Get(importCtx, nil); err != nil {
		return "", err
	}
	if err = temporalsdk_workflow.ExecuteActivity(sess, activities.InspectReportsName, p).Get(sess, nil); err != nil {
		return "", err
	}
	zipPath := path.Join(workspace, "object.zip")
	if err = temporalsdk_workflow.ExecuteActivity(sess, archivezip.Name,
		&archivezip.Params{SourceDir: path.Join(workspace, "object"), DestPath: zipPath}).Get(sess, nil); err != nil {
		return "", err
	}
	if err = temporalsdk_workflow.ExecuteActivity(sess, activities.ValidateZIPName, zipPath).
		Get(sess, nil); err != nil {
		return "", err
	}
	for _, name := range []string{"object.zip", "import-event.json", "import-report.md"} {
		local := zipPath
		if name != "object.zip" {
			local = path.Join(workspace, "reports", name)
		}
		if err = temporalsdk_workflow.ExecuteActivity(sess, bucketupload.Name,
			&bucketupload.Params{Path: local, Key: aips.OutputKey(r.UUID, name)}).Get(sess, nil); err != nil {
			return "", err
		}
	}
	return aips.OutputKey(r.UUID, "object.zip"), nil
}
