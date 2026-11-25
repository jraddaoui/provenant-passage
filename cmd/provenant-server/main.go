// SPDX-FileCopyrightText: Artefactual Systems Inc. and Enduro contributors
// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/artefactual-sdps/temporal-activities/archivezip"
	"github.com/artefactual-sdps/temporal-activities/bucketdownload"
	"github.com/artefactual-sdps/temporal-activities/bucketupload"
	"github.com/artefactual-sdps/temporal-activities/removepaths"
	"github.com/go-logr/logr"
	"github.com/oklog/run"
	"github.com/redis/go-redis/v9"
	temporaltools "go.artefactual.dev/tools/temporal"
	"go.opentelemetry.io/otel/trace/noop"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_client "go.temporal.io/sdk/client"
	temporalsdk_interceptor "go.temporal.io/sdk/interceptor"
	temporalsdk_log "go.temporal.io/sdk/log"
	temporalsdk_worker "go.temporal.io/sdk/worker"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/s3blob"

	"github.com/jraddaoui/provenant-passage/internal/aips"
	"github.com/jraddaoui/provenant-passage/internal/api"
	"github.com/jraddaoui/provenant-passage/internal/config"
	"github.com/jraddaoui/provenant-passage/internal/db"
	"github.com/jraddaoui/provenant-passage/internal/event"
	"github.com/jraddaoui/provenant-passage/internal/persistence/ent/client"
	"github.com/jraddaoui/provenant-passage/internal/temporal"
	"github.com/jraddaoui/provenant-passage/internal/version"
	"github.com/jraddaoui/provenant-passage/internal/workflow"
	"github.com/jraddaoui/provenant-passage/internal/workflow/activities"
)

func main() {
	file := flag.String("config", "", "TOML configuration file")
	showVersion := flag.Bool("version", false, "show local build version")
	flag.Parse()
	if *showVersion {
		fmt.Println(version.Info("provenant-server"))
		return
	}
	if err := serve(*file); err != nil {
		slog.Error("Server stopped", "error", err)
		os.Exit(1)
	}
}

func serve(file string) error {
	cfg, err := config.Load(file)
	if err != nil {
		return err
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.Log.Level)); err != nil {
		return err
	}
	logOptions := &slog.HandlerOptions{Level: level}
	var handler slog.Handler = slog.NewJSONHandler(os.Stderr, logOptions)
	if cfg.Log.Format == "text" {
		handler = slog.NewTextHandler(os.Stderr, logOptions)
	}
	slogLogger := slog.New(handler)
	slog.SetDefault(slogLogger)
	logger := logr.FromSlogHandler(handler)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startup, stopStartup := context.WithTimeout(ctx, 3*time.Minute)
	defer stopStartup()
	if err := db.Migrate(startup, cfg.Database.DSN); err != nil {
		return fmt.Errorf("migrate application database: %w", err)
	}
	conn, err := db.Open(startup, cfg.Database.DSN)
	if err != nil {
		return err
	}
	defer conn.Close()
	persistence := client.New(conn)
	source, err := blob.OpenBucket(startup, cfg.Source.URL)
	if err != nil {
		return fmt.Errorf("open source bucket: %w", err)
	}
	defer source.Close()
	output, err := blob.OpenBucket(startup, cfg.Output.URL)
	if err != nil {
		return fmt.Errorf("open output bucket: %w", err)
	}
	defer output.Close()
	// Verify Redis availability during startup; subsequent publication failures
	// remain best effort and do not interrupt Temporal's work delivery.
	redisOptions, err := redis.ParseURL(cfg.Event.RedisAddress)
	if err != nil {
		return err
	}
	redisCheck := redis.NewClient(redisOptions)
	err = redisCheck.Ping(startup).Err()
	_ = redisCheck.Close()
	if err != nil {
		return fmt.Errorf("connect Redis: %w", err)
	}
	events, err := event.NewServiceRedis(
		logger,
		noop.NewTracerProvider(),
		cfg.Event.RedisAddress,
		cfg.Event.RedisChannel,
		&aips.EventSerializer{},
	)
	if err != nil {
		return err
	}
	tc, err := temporalsdk_client.DialContext(
		startup,
		temporalsdk_client.Options{
			HostPort:  cfg.Temporal.Address,
			Namespace: cfg.Temporal.Namespace,
			Logger:    temporalsdk_log.NewStructuredLogger(slogLogger),
		},
	)
	if err != nil {
		return err
	}
	defer tc.Close()
	wf := &workflow.Workflow{Options: workflow.Options{
		WorkflowQueue:           cfg.Temporal.WorkflowQueue,
		ActivityQueue:           cfg.Temporal.ActivityQueue,
		SessionCreationTimeout:  cfg.Temporal.SessionCreationTimeout,
		SessionExecutionTimeout: cfg.Temporal.SessionExecutionTimeout,
		ActivityTimeout:         cfg.Temporal.ActivityTimeout,
		HeartbeatTimeout:        cfg.Temporal.HeartbeatTimeout,
	}}
	acts := &activities.Activities{
		ImporterBinary: cfg.ImporterBinary,
		ScratchDir:     cfg.ScratchDir,
		Persistence:    persistence,
		Events:         events,
		Output:         output,
	}
	workflowWorker := temporalsdk_worker.New(
		tc,
		cfg.Temporal.WorkflowQueue,
		temporalsdk_worker.Options{WorkerStopTimeout: 30 * time.Second},
	)
	workflowWorker.RegisterWorkflowWithOptions(
		wf.Execute,
		temporalsdk_workflow.RegisterOptions{Name: temporal.WorkflowName},
	)
	workflowWorker.RegisterActivityWithOptions(
		acts.UpdateStatus,
		temporalsdk_activity.RegisterOptions{Name: activities.UpdateStatusName},
	)
	sessionWorker := temporalsdk_worker.New(tc, cfg.Temporal.ActivityQueue, temporalsdk_worker.Options{
		DisableWorkflowWorker:              true,
		EnableSessionWorker:                true,
		MaxConcurrentSessionExecutionSize:  cfg.Temporal.SessionCapacity,
		MaxConcurrentActivityExecutionSize: 2 * cfg.Temporal.SessionCapacity,
		WorkerStopTimeout:                  30 * time.Second,
		Interceptors: []temporalsdk_interceptor.WorkerInterceptor{
			temporaltools.NewLoggerInterceptor(logger),
		},
	})
	for name, fn := range map[string]any{
		activities.CreateWorkspaceName: acts.CreateWorkspace, activities.RunImporterName: acts.RunImporter,
		activities.InspectReportsName: acts.InspectReports, activities.ValidateZIPName: acts.ValidateZIP,
		activities.RetainReportsName: acts.RetainReports, archivezip.Name: acts.ZIP,
		bucketdownload.Name: bucketdownload.New(source).Execute, bucketupload.Name: bucketupload.New(output).Execute,
		removepaths.Name: removepaths.New().Execute,
	} {
		sessionWorker.RegisterActivityWithOptions(fn, temporalsdk_activity.RegisterOptions{Name: name})
	}
	if err := workflowWorker.Start(); err != nil {
		return err
	}
	defer workflowWorker.Stop()
	if err := sessionWorker.Start(); err != nil {
		return err
	}
	defer sessionWorker.Stop()
	svc := &aips.Service{
		Persistence: persistence,
		Source:      aips.Bucket{Bucket: source},
		Output:      aips.Bucket{Bucket: output},
		Events:      events,
		Logger:      logger,
		Workflows:   &temporal.Client{Client: tc, TaskQueue: cfg.Temporal.WorkflowQueue},
	}
	server := api.Server(cfg.API.Listen, svc, logger)
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	healthMux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		c, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := conn.PingContext(c); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	health := &http.Server{Addr: cfg.HealthListen, Handler: healthMux, ReadHeaderTimeout: 5 * time.Second}
	var group run.Group
	for _, srv := range []*http.Server{server, health} {
		group.Add(func() error {
			err := srv.ListenAndServe()
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err
		}, func(error) {
			shutdown, stop := context.WithTimeout(context.Background(), 10*time.Second)
			defer stop()
			if err := srv.Shutdown(shutdown); err != nil {
				logger.Error(err, "HTTP shutdown")
				_ = srv.Close()
			}
		})
	}
	group.Add(func() error {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			if err := svc.Reconcile(ctx); err != nil && !errors.Is(err, context.Canceled) {
				logger.Error(err, "AIP reconciliation")
			}
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			}
		}
	}, func(error) { cancel() })
	group.Add(run.SignalHandler(ctx, os.Interrupt, syscall.SIGTERM))
	logger.Info("Provenant Passage ready", "api", cfg.API.Listen, "namespace", cfg.Temporal.Namespace)
	err = group.Run()
	if _, ok := errors.AsType[run.SignalError](err); ok {
		return nil
	}
	return err
}
