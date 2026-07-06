// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The publisher-crates Authors

package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const cargoTokenEnv = "CARGO_REGISTRY_TOKEN"

// Config captures semrel and plugin environment.
type Config struct {
	Version      string
	DryRun       bool
	Token        string
	ManifestPath string
	Packages     []string
}

// Invocation describes one cargo publish execution.
type Invocation struct {
	Name string
	Args []string
	Env  []string
}

// Runner executes one cargo publish invocation.
type Runner interface {
	Run(ctx context.Context, invocation Invocation, stdout, stderr io.Writer) error
}

// ExecRunner executes invocations using os/exec.
type ExecRunner struct{}

// LoadConfig parses environment variables and validates required settings.
func LoadConfig(getenv func(string) string) (Config, error) {
	cfg := Config{
		Version:      strings.TrimPrefix(strings.TrimSpace(firstNonEmpty(getenv("SEMREL_VERSION"), getenv("SEMREL_NEXT_VERSION"))), "v"),
		DryRun:       strings.EqualFold(strings.TrimSpace(getenv("SEMREL_DRY_RUN")), "true"),
		Token:        strings.TrimSpace(getenv("SEMREL_PLUGIN_CARGO_REGISTRY_TOKEN")),
		ManifestPath: strings.TrimSpace(getenv("SEMREL_PLUGIN_CARGO_MANIFEST_PATH")),
		Packages: mergePackageLists(
			parseCSV(getenv("SEMREL_PLUGIN_CARGO_WORKSPACE_MEMBER")),
			parseCSV(getenv("SEMREL_PLUGIN_CARGO_PACKAGE")),
		),
	}

	if cfg.Version == "" {
		return Config{}, errors.New("SEMREL_VERSION is required")
	}
	if !cfg.DryRun && cfg.Token == "" {
		return Config{}, errors.New("SEMREL_PLUGIN_CARGO_REGISTRY_TOKEN is required unless SEMREL_DRY_RUN=true")
	}

	return cfg, nil
}

// BuildInvocations creates one or more cargo publish executions.
func BuildInvocations(cfg Config) []Invocation {
	packages := cfg.Packages
	if len(packages) == 0 {
		packages = []string{""}
	}

	invocations := make([]Invocation, 0, len(packages))
	for _, pkg := range packages {
		args := []string{"publish"}
		if cfg.ManifestPath != "" {
			args = append(args, "--manifest-path", cfg.ManifestPath)
		}
		if pkg != "" {
			args = append(args, "--package", pkg)
		}
		if cfg.DryRun {
			args = append(args, "--dry-run")
		}

		invocation := Invocation{Name: "cargo", Args: args}
		if cfg.Token != "" {
			invocation.Env = []string{cargoTokenEnv + "=" + cfg.Token}
		}
		invocations = append(invocations, invocation)
	}

	return invocations
}

// Publish executes cargo publish for the configured package set.
func Publish(ctx context.Context, cfg Config, runner Runner, stdout, stderr io.Writer) error {
	invocations := BuildInvocations(cfg)
	for index, invocation := range invocations {
		if err := runner.Run(ctx, invocation, stdout, stderr); err != nil {
			if len(cfg.Packages) > 0 {
				return fmt.Errorf("package %q: %w", cfg.Packages[index], err)
			}
			return err
		}
	}
	return nil
}

// Run executes cargo publish using the current process environment.
func (ExecRunner) Run(ctx context.Context, invocation Invocation, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, invocation.Name, invocation.Args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), invocation.Env...)

	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("%s not found on PATH", invocation.Name)
		}
		return fmt.Errorf("%s %s: %w", invocation.Name, strings.Join(invocation.Args, " "), err)
	}

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func parseCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func mergePackageLists(lists ...[]string) []string {
	seen := make(map[string]struct{})
	merged := make([]string, 0)
	for _, list := range lists {
		for _, item := range list {
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			merged = append(merged, item)
		}
	}
	return merged
}
