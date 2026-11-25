// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigurationPrecedence(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.WriteFile(
		filepath.Join(root, "provenant.toml"),
		[]byte("[api]\nlisten='127.0.0.1:9010'\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "provenant.local.toml"),
		[]byte("[api]\nlisten='127.0.0.1:9011'\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if c.API.Listen != "127.0.0.1:9011" {
		t.Fatal("local override ignored")
	}
	c, err = Load(filepath.Join(root, "provenant.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.API.Listen != "127.0.0.1:9010" {
		t.Fatal("explicit file overridden")
	}
	t.Setenv("PROVENANT_API_LISTEN", "127.0.0.1:9012")
	t.Setenv("PROVENANT_EVENT_REDISCHANNEL", "separate-instance-test")
	c, err = Load("")
	if err != nil {
		t.Fatal(err)
	}
	if c.API.Listen != "127.0.0.1:9012" || c.Event.RedisChannel != "separate-instance-test" {
		t.Fatal("environment ignored")
	}
	if _, err := Load("absent.toml"); err == nil {
		t.Fatal("missing explicit configuration accepted")
	}
}

func TestRejectInvalidConfiguration(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, tc := range []struct{ key, value string }{
		{"PROVENANT_SOURCE_URL", "file:///tmp/aips"},
		{"PROVENANT_TEMPORAL_SESSIONCAPACITY", "0"},
		{"PROVENANT_TEMPORAL_ACTIVITYTIMEOUT", "1s"},
		{"PROVENANT_EVENT_REDISADDRESS", "http://redis"},
		{"PROVENANT_LOG_FORMAT", "invalid"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			if _, err := Load(""); err == nil {
				t.Fatal("accepted invalid configuration")
			}
		})
	}
}
