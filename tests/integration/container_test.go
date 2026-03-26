//go:build integration

// Package integration provides end-to-end tests that require a running Docker daemon.
// Run with: go test -tags=integration ./tests/integration/...
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	cnt "github.com/servusdei2018/sandbox/pkg/container"
)

func TestContainerExecution_EchoHello(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	manager, err := cnt.NewManager(logger)
	require.NoError(t, err)
	defer manager.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Pull alpine if needed.
	require.NoError(t, manager.PullIfMissing(ctx, "alpine:latest"))

	cfg := &cnt.Config{
		Image:        "alpine:latest",
		Cmd:          []string{"echo", "hello"},
		WorkspaceDir: t.TempDir(),
		MountTarget:  "/work",
		Env:          []string{},
		Timeout:      time.Minute,
		RemoveOnExit: true,
		NetworkMode:  "bridge",
	}

	containerID, err := manager.Create(ctx, cfg)
	require.NoError(t, err)
	require.NotEmpty(t, containerID)

	// Verify labels.
	inspect, err := manager.Inspect(ctx, containerID)
	require.NoError(t, err)
	require.Equal(t, "true", inspect.Config.Labels["sandbox"])

	exitCode, err := manager.Run(ctx, containerID, false)
	require.NoError(t, err)
	require.Equal(t, 0, exitCode)

	// Cleanup.
	_ = manager.Remove(ctx, containerID)
}

func TestContainerExecution_ExitCode(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	manager, err := cnt.NewManager(logger)
	require.NoError(t, err)
	defer manager.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	require.NoError(t, manager.PullIfMissing(ctx, "alpine:latest"))

	cfg := &cnt.Config{
		Image:        "alpine:latest",
		Cmd:          []string{"sh", "-c", "exit 42"},
		WorkspaceDir: t.TempDir(),
		MountTarget:  "/work",
		Env:          []string{},
		Timeout:      time.Minute,
		RemoveOnExit: true,
		NetworkMode:  "bridge",
	}

	containerID, err := manager.Create(ctx, cfg)
	require.NoError(t, err)

	exitCode, err := manager.Run(ctx, containerID, false)
	require.NoError(t, err)
	require.Equal(t, 42, exitCode)

	_ = manager.Remove(ctx, containerID)
}

func TestContainerExecution_WorkspaceMounted(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	manager, err := cnt.NewManager(logger)
	require.NoError(t, err)
	defer manager.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	require.NoError(t, manager.PullIfMissing(ctx, "alpine:latest"))

	// Write a sentinel file in the workspace.
	wsDir := t.TempDir()
	require.NoError(t, writeFile(wsDir+"/sentinel.txt", "exists"))

	// The container should see the file at /work/sentinel.txt.
	cfg := &cnt.Config{
		Image:        "alpine:latest",
		Cmd:          []string{"sh", "-c", "test -f /work/sentinel.txt"},
		WorkspaceDir: wsDir,
		MountTarget:  "/work",
		Env:          []string{},
		Timeout:      time.Minute,
		RemoveOnExit: true,
		NetworkMode:  "bridge",
	}

	containerID, err := manager.Create(ctx, cfg)
	require.NoError(t, err)

	exitCode, err := manager.Run(ctx, containerID, false)
	require.NoError(t, err)
	require.Equal(t, 0, exitCode, "sentinel file should be visible at /work/sentinel.txt")

	_ = manager.Remove(ctx, containerID)
}
