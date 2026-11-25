// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package temporal

import (
	"context"
	"errors"
	"fmt"

	temporalapi_enums "go.temporal.io/api/enums/v1"
	temporalapi_serviceerror "go.temporal.io/api/serviceerror"
	temporalsdk_client "go.temporal.io/sdk/client"

	"github.com/jraddaoui/provenant-passage/internal/aips"
)

const WorkflowName = "import-aip"

type Client struct {
	Client    temporalsdk_client.Client
	TaskQueue string
}

func (c *Client) Start(ctx context.Context, r aips.ImportRequest) error {
	_, err := c.Client.ExecuteWorkflow(ctx, temporalsdk_client.StartWorkflowOptions{
		ID: r.WorkflowID(), TaskQueue: c.TaskQueue,
		WorkflowIDReusePolicy:    temporalapi_enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowIDConflictPolicy: temporalapi_enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}, WorkflowName, r)
	if _, ok := errors.AsType[*temporalapi_serviceerror.WorkflowExecutionAlreadyStarted](err); ok {
		return nil
	}
	var invalid *temporalapi_serviceerror.InvalidArgument
	var denied *temporalapi_serviceerror.PermissionDenied
	var missing *temporalapi_serviceerror.NamespaceNotFound
	var precondition *temporalapi_serviceerror.FailedPrecondition
	if errors.As(err, &invalid) || errors.As(err, &denied) || errors.As(err, &missing) ||
		errors.As(err, &precondition) {
		return fmt.Errorf("%w: %v", aips.ErrStartRejected, err)
	}
	return err
}

func (c *Client) Inspect(ctx context.Context, id string) (aips.ExecutionState, error) {
	res, err := c.Client.DescribeWorkflowExecution(ctx, id, "")
	if _, ok := errors.AsType[*temporalapi_serviceerror.NotFound](err); ok {
		return aips.ExecutionAbsent, nil
	}
	if err != nil {
		return "", err
	}
	switch res.WorkflowExecutionInfo.Status {
	case temporalapi_enums.WORKFLOW_EXECUTION_STATUS_RUNNING:
		return aips.ExecutionRunning, nil
	case temporalapi_enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return aips.ExecutionCompleted, nil
	case temporalapi_enums.WORKFLOW_EXECUTION_STATUS_CANCELED:
		return aips.ExecutionCanceled, nil
	default:
		return aips.ExecutionFailed, nil
	}
}
