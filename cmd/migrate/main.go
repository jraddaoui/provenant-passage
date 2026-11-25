// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/jraddaoui/provenant-passage/internal/config"
	"github.com/jraddaoui/provenant-passage/internal/db"
)

func main() {
	file := flag.String("config", "", "configuration file")
	flag.Parse()
	cfg, err := config.Load(*file)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := db.Migrate(ctx, cfg.Database.DSN); err != nil {
		log.Fatal(err)
	}
}
