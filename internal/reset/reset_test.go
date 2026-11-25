// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package reset

import (
	"context"
	"errors"
	"testing"

	temporalapi_common "go.temporal.io/api/common/v1"
	temporalapi_enums "go.temporal.io/api/enums/v1"
	temporalapi_workflow "go.temporal.io/api/workflow/v1"
	temporalapi_workflowservice "go.temporal.io/api/workflowservice/v1"

	"github.com/jraddaoui/provenant-passage/internal/config"
)

type fakeTemporal struct {
	TemporalAdmin
	terminated bool
	fail       bool
}

func (f *fakeTemporal) ListWorkflow(
	ctx context.Context,
	r *temporalapi_workflowservice.ListWorkflowExecutionsRequest,
) (*temporalapi_workflowservice.ListWorkflowExecutionsResponse, error) {
	if f.terminated {
		return &temporalapi_workflowservice.ListWorkflowExecutionsResponse{}, nil
	}
	return &temporalapi_workflowservice.ListWorkflowExecutionsResponse{
		Executions: []*temporalapi_workflow.WorkflowExecutionInfo{
			{Execution: &temporalapi_common.WorkflowExecution{WorkflowId: "import-aip-test", RunId: "run"}},
		},
	}, nil
}

func (f *fakeTemporal) TerminateWorkflow(ctx context.Context, id, run, reason string, details ...any) error {
	if f.fail {
		return errors.New("termination rejected")
	}
	f.terminated = true
	return nil
}

func (f *fakeTemporal) DescribeWorkflowExecution(
	ctx context.Context,
	id, run string,
) (*temporalapi_workflowservice.DescribeWorkflowExecutionResponse, error) {
	if !f.terminated {
		return nil, errors.New("not terminated")
	}
	return &temporalapi_workflowservice.DescribeWorkflowExecutionResponse{
		WorkflowExecutionInfo: &temporalapi_workflow.WorkflowExecutionInfo{
			Status: temporalapi_enums.WORKFLOW_EXECUTION_STATUS_TERMINATED,
		},
	}, nil
}

func TestTerminateBeforeReset(t *testing.T) {
	for _, fail := range []bool{false, true} {
		f := &fakeTemporal{fail: fail}
		err := TerminateImports(t.Context(), f)
		if fail && err == nil {
			t.Fatal("reset continued after termination failure")
		}
		if !fail && (err != nil || !f.terminated) {
			t.Fatalf("termination: %v", err)
		}
	}
}

func TestResetScope(t *testing.T) {
	t.Chdir(t.TempDir())
	c, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateScope(c); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*config.Config){
		func(c *config.Config) { c.Temporal.Namespace = "default" },
		func(c *config.Config) { c.Source.URL = "s3://other?region=us-east-1" },
		func(c *config.Config) { c.Output.URL = "s3://other?region=us-east-1" },
		func(c *config.Config) { c.Database.DSN = "root:placeholder@tcp(mysql:3306)/other" },
	} {
		modified := c
		mutate(&modified)
		if err := ValidateScope(modified); err == nil {
			t.Fatal("reset accepted unrelated resources")
		}
	}
}
