// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The publisher-crates Authors

package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/SemRels/publisher-crates/internal/plugin"
)

const pluginSchemaVersion = 1

func main() {
	os.Exit(run(os.Stdout, os.Stderr, os.Getenv))
}

func run(stdout, stderr io.Writer, getenv func(string) string) int {
	_, _ = fmt.Fprintf(stderr, "plugin_schema_version=%d\n", pluginSchemaVersion)

	cfg, err := plugin.LoadConfig(getenv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "publisher-crates:", err)
		return 1
	}

	if err := plugin.Publish(context.Background(), cfg, plugin.ExecRunner{}, stdout, stderr); err != nil {
		_, _ = fmt.Fprintln(stderr, "publisher-crates:", err)
		return 1
	}

	if cfg.DryRun {
		if len(cfg.Packages) == 0 {
			_, _ = fmt.Fprintln(stdout, "publisher-crates: dry-run validation completed")
			return 0
		}
		_, _ = fmt.Fprintf(stdout, "publisher-crates: dry-run validation completed for %d package(s)\n", len(cfg.Packages))
		return 0
	}

	if len(cfg.Packages) == 0 {
		_, _ = fmt.Fprintln(stdout, "publisher-crates: published crate")
		return 0
	}

	_, _ = fmt.Fprintf(stdout, "publisher-crates: published %d package(s)\n", len(cfg.Packages))
	return 0
}
