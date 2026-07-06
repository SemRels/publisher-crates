// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The publisher-crates Authors

package plugin

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeRunner struct {
	invocations []Invocation
	failAt      int
	err         error
}

func (f *fakeRunner) Run(_ context.Context, invocation Invocation, _, _ io.Writer) error {
	f.invocations = append(f.invocations, invocation)
	if f.err != nil && len(f.invocations)-1 == f.failAt {
		return f.err
	}
	return nil
}

func TestLoadConfigRequiresTokenUnlessDryRun(t *testing.T) {
	t.Parallel()

	_, err := LoadConfig(func(key string) string {
		switch key {
		case "SEMREL_NEXT_VERSION":
			return "v1.2.3"
		default:
			return ""
		}
	})

	require.EqualError(t, err, "SEMREL_PLUGIN_CARGO_REGISTRY_TOKEN is required unless SEMREL_DRY_RUN=true")
}

func TestBuildInvocationsWithSinglePackageManifestAndDryRun(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConfig(func(key string) string {
		switch key {
		case "SEMREL_VERSION":
			return "v1.2.3"
		case "SEMREL_DRY_RUN":
			return "true"
		case "SEMREL_PLUGIN_CARGO_WORKSPACE_MEMBER":
			return "crate-a"
		case "SEMREL_PLUGIN_CARGO_MANIFEST_PATH":
			return "crates/crate-a/Cargo.toml"
		default:
			return ""
		}
	})
	require.NoError(t, err)

	invocations := BuildInvocations(cfg)
	require.Len(t, invocations, 1)
	require.Equal(t, []string{"publish", "--manifest-path", "crates/crate-a/Cargo.toml", "--package", "crate-a", "--dry-run"}, invocations[0].Args)
	require.Empty(t, invocations[0].Env)
}

func TestBuildInvocationsMultiplePackagesUsesAliasAndEnvToken(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConfig(func(key string) string {
		switch key {
		case "SEMREL_VERSION":
			return "1.2.3"
		case "SEMREL_PLUGIN_CARGO_REGISTRY_TOKEN":
			return "super-secret"
		case "SEMREL_PLUGIN_CARGO_WORKSPACE_MEMBER":
			return "crate-a, crate-b"
		case "SEMREL_PLUGIN_CARGO_PACKAGE":
			return "crate-b,crate-c"
		default:
			return ""
		}
	})
	require.NoError(t, err)

	invocations := BuildInvocations(cfg)
	require.Len(t, invocations, 3)
	require.Equal(t, []string{"crate-a", "crate-b", "crate-c"}, cfg.Packages)
	for i, pkg := range []string{"crate-a", "crate-b", "crate-c"} {
		require.Equal(t, []string{"publish", "--package", pkg}, invocations[i].Args)
		require.Equal(t, []string{"CARGO_REGISTRY_TOKEN=super-secret"}, invocations[i].Env)
		require.NotContains(t, invocations[i].Args, "super-secret")
	}
}

func TestPublishStopsOnFirstPackageFailure(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Version:  "1.2.3",
		Token:    "super-secret",
		Packages: []string{"crate-a", "crate-b", "crate-c"},
	}
	runner := &fakeRunner{failAt: 1, err: errors.New("publish failed")}

	err := Publish(context.Background(), cfg, runner, &bytes.Buffer{}, &bytes.Buffer{})

	require.EqualError(t, err, "package \"crate-b\": publish failed")
	require.Len(t, runner.invocations, 2)
	require.Equal(t, "crate-b", cfg.Packages[1])
}

func TestPublishWithoutPackageRunsSingleInvocation(t *testing.T) {
	t.Parallel()

	cfg := Config{Version: "1.2.3", Token: "super-secret"}
	runner := &fakeRunner{}

	err := Publish(context.Background(), cfg, runner, &bytes.Buffer{}, &bytes.Buffer{})

	require.NoError(t, err)
	require.Len(t, runner.invocations, 1)
	require.Equal(t, []string{"publish"}, runner.invocations[0].Args)
	require.Equal(t, []string{"CARGO_REGISTRY_TOKEN=super-secret"}, runner.invocations[0].Env)
}
