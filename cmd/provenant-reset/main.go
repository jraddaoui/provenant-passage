// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	temporalsdk_client "go.temporal.io/sdk/client"

	"github.com/jraddaoui/provenant-passage/internal/config"
	"github.com/jraddaoui/provenant-passage/internal/reset"
)

func run() error {
	file := flag.String("config", "/etc/provenant/provenant.toml", "configuration path")
	confirm := flag.Bool(
		"confirm-reset-provenant",
		false,
		"confirm application pods are stopped and delete local PoC application data",
	)
	flag.Parse()
	if !*confirm {
		return errors.New("requires --confirm-reset-provenant; invoke through hack/reset.sh")
	}
	c, err := config.Load(*file)
	if err != nil {
		return err
	}
	if err := reset.ValidateScope(c); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	tc, err := temporalsdk_client.DialContext(
		ctx,
		temporalsdk_client.Options{HostPort: c.Temporal.Address, Namespace: c.Temporal.Namespace},
	)
	if err != nil {
		return err
	}
	defer tc.Close()
	if err := reset.TerminateImports(ctx, tc); err != nil {
		return err
	}
	for _, bucket := range []string{c.Source.URL, c.Output.URL} {
		if err := reset.RecreateBucket(ctx, bucket); err != nil {
			return err
		}
	}
	if err := reset.RecreateDatabase(ctx, c.Database.DSN, os.Getenv("MYSQL_ROOT_PASSWORD")); err != nil {
		return err
	}
	fmt.Println("Reset application database and two buckets; Temporal history retained.")
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
