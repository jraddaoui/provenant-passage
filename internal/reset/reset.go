// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

// Package reset implements the explicitly manual local PoC reset. The caller
// must stop application pods before invoking it, so local activities cannot
// continue writing after their Temporal execution is terminated.
package reset

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	mysql "github.com/go-sql-driver/mysql"
	temporalapi_enums "go.temporal.io/api/enums/v1"
	temporalapi_serviceerror "go.temporal.io/api/serviceerror"
	temporalapi_workflowservice "go.temporal.io/api/workflowservice/v1"

	"github.com/jraddaoui/provenant-passage/internal/config"
	"github.com/jraddaoui/provenant-passage/internal/db"
)

type TemporalAdmin interface {
	ListWorkflow(
		context.Context,
		*temporalapi_workflowservice.ListWorkflowExecutionsRequest,
	) (*temporalapi_workflowservice.ListWorkflowExecutionsResponse, error)
	TerminateWorkflow(context.Context, string, string, string, ...any) error
	DescribeWorkflowExecution(
		context.Context,
		string,
		string,
	) (*temporalapi_workflowservice.DescribeWorkflowExecutionResponse, error)
}

func ValidateScope(c config.Config) error {
	dsn, err := mysql.ParseDSN(c.Database.DSN)
	if err != nil || dsn.DBName != "provenant" || c.Temporal.Namespace != "provenant" {
		return errors.New("reset requires application database and Temporal namespace provenant")
	}
	for name, raw := range map[string]string{"am-aips": c.Source.URL, "ocfl-objects": c.Output.URL} {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "s3" || u.Host != name || u.Path != "" || u.Query().Get("endpoint") == "" {
			return fmt.Errorf("reset requires explicit S3 endpoint and bucket %s", name)
		}
	}
	return nil
}

func TerminateImports(ctx context.Context, tc TemporalAdmin) error {
	// Visibility is eventually consistent. Re-scan from the beginning and inspect
	// actual execution status until the running query no longer returns imports.
	for {
		result, err := tc.ListWorkflow(ctx, &temporalapi_workflowservice.ListWorkflowExecutionsRequest{
			Namespace: "provenant", Query: "WorkflowType = 'import-aip' AND ExecutionStatus = 'Running'", PageSize: 100,
		})
		if err != nil {
			return fmt.Errorf("list imports: %w", err)
		}
		if len(result.Executions) == 0 {
			return nil
		}
		for _, info := range result.Executions {
			id, run := info.Execution.GetWorkflowId(), info.Execution.GetRunId()
			if err := tc.TerminateWorkflow(ctx, id, run, "manual Provenant Passage reset"); err != nil {
				if _, closed := errors.AsType[*temporalapi_serviceerror.NotFound](err); !closed {
					return fmt.Errorf("terminate %s: %w", id, err)
				}
			}
			for {
				desc, err := tc.DescribeWorkflowExecution(ctx, id, run)
				if err != nil {
					return fmt.Errorf("confirm termination %s: %w", id, err)
				}
				if desc.GetWorkflowExecutionInfo().GetStatus() != temporalapi_enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
					break
				}
				if err := wait(ctx); err != nil {
					return err
				}
			}
		}
		if err := wait(ctx); err != nil {
			return err
		}
	}
}

func wait(ctx context.Context) error {
	timer := time.NewTimer(250 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// RecreateDatabase never touches Temporal's infrastructure databases.
func RecreateDatabase(ctx context.Context, dsn, rootPassword string) error {
	c, err := mysql.ParseDSN(dsn)
	if err != nil || c.DBName != "provenant" || rootPassword == "" {
		return errors.New("invalid reset database credentials/scope")
	}
	c.User, c.Passwd, c.DBName = "root", rootPassword, ""
	conn, err := db.Open(ctx, c.FormatDSN())
	if err != nil {
		return err
	}
	defer conn.Close()
	for _, query := range []string{"DROP DATABASE IF EXISTS provenant", "CREATE DATABASE provenant CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"} {
		if _, err := conn.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	return db.Migrate(ctx, dsn)
}

func RecreateBucket(ctx context.Context, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Host != "am-aips" && u.Host != "ocfl-objects") || u.Query().Get("endpoint") == "" {
		return errors.New("invalid reset bucket scope")
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(u.Query().Get("region")))
	if err != nil {
		return err
	}
	client := s3.NewFromConfig(
		cfg,
		func(o *s3.Options) { o.BaseEndpoint = aws.String(u.Query().Get("endpoint")); o.UsePathStyle = true },
	)
	bucket := aws.String(u.Host)
	// Drain from the beginning so deleting keys cannot invalidate pagination.
	for {
		objects, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: bucket, MaxKeys: aws.Int32(1000)})
		if _, missing := errors.AsType[*types.NoSuchBucket](err); missing {
			break
		}
		if err != nil {
			return err
		}
		if len(objects.Contents) == 0 {
			break
		}
		for _, object := range objects.Contents {
			if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: bucket, Key: object.Key}); err != nil {
				return err
			}
		}
	}
	_, err = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: bucket})
	if _, missing := errors.AsType[*types.NoSuchBucket](err); err != nil && !missing {
		return err
	}
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: bucket})
	return err
}
