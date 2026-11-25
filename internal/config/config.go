// SPDX-FileCopyrightText: Artefactual Systems Inc. and Enduro contributors
// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"

	"github.com/jraddaoui/provenant-passage/internal/event"
)

type Config struct {
	API          struct{ Listen string }
	HealthListen string
	Database     struct{ DSN string }
	Temporal     struct {
		Address, Namespace, WorkflowQueue, ActivityQueue                                   string
		SessionCapacity                                                                    int
		SessionCreationTimeout, SessionExecutionTimeout, ActivityTimeout, HeartbeatTimeout time.Duration
	}
	Source, Output             struct{ URL string }
	Event                      event.Config
	ScratchDir, ImporterBinary string
	Log                        struct{ Level, Format string }
}

func Load(file string) (Config, error) {
	v := viper.New()
	v.SetConfigType("toml")
	v.SetEnvPrefix("PROVENANT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	defaults := map[string]any{
		"api.listen":                       "0.0.0.0:9000",
		"healthListen":                     "0.0.0.0:9001",
		"database.dsn":                     "provenant:provenant-dev@tcp(mysql.provenant:3306)/provenant",
		"temporal.address":                 "temporal-frontend.provenant:7233",
		"temporal.namespace":               "provenant",
		"temporal.workflowQueue":           "provenant-import",
		"temporal.activityQueue":           "provenant-import-sessions",
		"temporal.sessionCapacity":         1,
		"temporal.sessionCreationTimeout":  10 * time.Minute,
		"temporal.sessionExecutionTimeout": 6 * time.Hour,
		"temporal.activityTimeout":         time.Hour,
		"temporal.heartbeatTimeout":        20 * time.Second,
		"source.url":                       "s3://am-aips?region=us-east-1&endpoint=http://seaweedfs.provenant:8333&use_path_style=true&disable_https=true&awssdk=v2",
		"output.url":                       "s3://ocfl-objects?region=us-east-1&endpoint=http://seaweedfs.provenant:8333&use_path_style=true&disable_https=true&awssdk=v2",
		"event.redisAddress":               "redis://redis.provenant:6379",
		"event.redisChannel":               "provenant-aips-events",
		"scratchDir":                       "/tmp/provenant-passage",
		"importerBinary":                   "/usr/local/bin/provenant-import",
		"log.level":                        "info",
		"log.format":                       "json",
	}
	for key, value := range defaults {
		v.SetDefault(key, value)
	}
	if file != "" {
		v.SetConfigFile(file)
		if err := v.ReadInConfig(); err != nil {
			return Config{}, fmt.Errorf("read configuration: %w", err)
		}
	} else {
		for _, name := range []string{"provenant.toml", "provenant.local.toml"} {
			f, err := os.Open(name)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return Config{}, err
			}
			err = v.MergeConfig(f)
			closeErr := f.Close()
			if err = errors.Join(err, closeErr); err != nil {
				return Config{}, err
			}
		}
	}
	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return c, err
	}
	return c, c.Validate()
}

func (c Config) Validate() error {
	for _, raw := range []string{c.Source.URL, c.Output.URL} {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "s3" || u.Host == "" {
			return errors.New("source and output must be S3 bucket URLs")
		}
	}
	if _, err := redis.ParseURL(c.Event.RedisAddress); err != nil {
		return errors.New("invalid Redis address")
	}
	if c.Event.RedisChannel == "" || c.Database.DSN == "" || c.API.Listen == "" || c.HealthListen == "" ||
		c.ImporterBinary == "" ||
		c.ScratchDir == "" {
		return errors.New("missing required service configuration")
	}
	t := c.Temporal
	if t.Address == "" || t.Namespace == "" || t.WorkflowQueue == "" || t.ActivityQueue == "" ||
		t.WorkflowQueue == t.ActivityQueue {
		return errors.New("temporal needs address, namespace and distinct workflow/activity queues")
	}
	if t.SessionCapacity < 1 || t.HeartbeatTimeout < 2*time.Second || t.ActivityTimeout <= 2*t.HeartbeatTimeout ||
		t.SessionCreationTimeout <= 0 ||
		t.SessionExecutionTimeout < t.ActivityTimeout {
		return errors.New("invalid Temporal capacity/timeouts")
	}
	if c.Log.Format != "json" && c.Log.Format != "text" {
		return errors.New("log format must be json or text")
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(c.Log.Level)); err != nil {
		return err
	}
	return nil
}
