// SPDX-FileCopyrightText: Artefactual Systems Inc. and Enduro contributors
// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

// Package version provides explicit local build metadata without VCS discovery.
package version

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed VERSION.txt
var number string

var Long = ""

func Info(appName string) string {
	if Long == "" {
		return fmt.Sprintf("%s version %s-local", appName, strings.TrimSpace(number))
	}
	return fmt.Sprintf("%s version %s", appName, Long)
}
