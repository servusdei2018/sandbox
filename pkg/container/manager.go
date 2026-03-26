// Package container provides Docker lifecycle management for sandbox containers.
package container

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/moby/term"
	"go.uber.org/zap"

	"github.com/servusdei2018/sandbox/pkg/config"
)

const defaultStopTimeout = 10

// Manager wraps the Docker client and provides high-level container lifecycle
// operations needed by the sandbox CLI.
type Manager struct {
	client *client.Client
	logger *zap.Logger
}

// Config holds parameters for creating and running a sandbox container.
type Config struct {
	// Image is the Docker image to use, e.g. "alpine:latest".
	Image string

	// Cmd is the command (and args) to execute inside the container.
	Cmd []string

	// Entrypoint is the optional entrypoint override for the container.
	Entrypoint []string

	// WorkspaceDir is the host path that will be bind-mounted to /work.
	WorkspaceDir string

	// MountTarget is the path inside the container for the workspace mount.
	// Defaults to "/work" if empty.
	MountTarget string

	// Env is the filtered list of "KEY=VALUE" environment variables.
	Env []string

	// Timeout is the maximum duration the container may run.
	Timeout time.Duration

	// RemoveOnExit removes the container after it stops.
	RemoveOnExit bool

	// NetworkMode controls the Docker network mode (e.g. "bridge").
	NetworkMode string

	// Tty enables a TTY for the container.
	Tty bool

	// AttachStdin allows piping host stdin to the container.
	AttachStdin bool

	// Security holds confinement and resource-limit settings.
	Security config.SecurityConfig
}

// NewManager creates a Manager using the Docker daemon reachable from the
// environment (DOCKER_HOST, socket, etc.).
//
// An error is returned if the client cannot be initialised — this typically means Docker is not running.
func NewManager(logger *zap.Logger) (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	return &Manager{client: cli, logger: logger.Named("container")}, nil
}

// Close releases client resources.
func (m *Manager) Close() error {
	return m.client.Close()
}

// PullIfMissing pulls the image if it is not already present in the local
// daemon cache. Progress is streamed to a Zap debug logger.
func (m *Manager) PullIfMissing(ctx context.Context, imageName string) error {
	m.logger.Debug("checking image availability", zap.String("image", imageName))

	// Check if the image already exists locally.
	_, err := m.client.ImageInspect(ctx, imageName)
	if err == nil {
		m.logger.Debug("image already present", zap.String("image", imageName))
		return nil
	}
	if !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to inspect image %s: %w", imageName, err)
	}

	m.logger.Info("pulling image", zap.String("image", imageName))

	reader, err := m.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			m.logger.Warn("failed to close image pull reader", zap.Error(err))
		}
	}()

	// Consume and log pull progress events.
	type progressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	}
	type pullEvent struct {
		Status         string         `json:"status"`
		ProgressDetail progressDetail `json:"progressDetail"`
		ID             string         `json:"id"`
		Error          string         `json:"error"`
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		var evt pullEvent
		if err := json.Unmarshal(scanner.Bytes(), &evt); err == nil {
			if evt.Error != "" {
				return fmt.Errorf("pull error: %s", evt.Error)
			}
			m.logger.Debug("image pull progress",
				zap.String("id", evt.ID),
				zap.String("status", evt.Status),
			)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read pull response: %w", err)
	}

	return nil
}

// Create creates a new container using the provided config and returns the
// container ID on success.
func (m *Manager) Create(ctx context.Context, cfg *Config) (string, error) {
	mountTarget := cfg.MountTarget
	if mountTarget == "" {
		mountTarget = "/work"
	}

	m.logger.Info("creating container",
		zap.String("image", cfg.Image),
		zap.String("workspace", cfg.WorkspaceDir),
		zap.String("mount_target", mountTarget),
		zap.Strings("cmd", cfg.Cmd),
	)

	secOpts, err := BuildSecurityOptions(cfg.Security)
	if err != nil {
		return "", fmt.Errorf("failed to build security options: %w", err)
	}

	hostCfg := &container.HostConfig{
		Binds:          []string{fmt.Sprintf("%s:%s", cfg.WorkspaceDir, mountTarget)},
		NetworkMode:    container.NetworkMode(cfg.NetworkMode),
		AutoRemove:     false, // We handle removal explicitly so we can log it.
		CapDrop:        secOpts.CapDrop,
		SecurityOpt:    secOpts.SecurityOpt,
		ReadonlyRootfs: secOpts.ReadonlyRootfs,
		Tmpfs:          secOpts.Tmpfs,
		Resources:      secOpts.Resources,
	}

	entrypoint := cfg.Entrypoint
	if len(secOpts.SecurityOpt) > 0 {
		m.logger.Debug("applying security options", zap.Strings("opts", secOpts.SecurityOpt))
	}

	containerCfg := &container.Config{
		Image:        cfg.Image,
		Cmd:          cfg.Cmd,
		Entrypoint:   entrypoint,
		Env:          cfg.Env,
		WorkingDir:   mountTarget,
		User:         secOpts.User,
		Labels:       map[string]string{"sandbox": "true"},
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  cfg.AttachStdin,
		Tty:          cfg.Tty,
		OpenStdin:    cfg.AttachStdin,
		StdinOnce:    cfg.AttachStdin,
	}

	resp, err := m.client.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	m.logger.Info("container created",
		zap.String("container_id", resp.ID[:12]),
		zap.Duration("timeout", cfg.Timeout),
	)

	return resp.ID, nil
}

// Run starts the container identified by containerID, streams its stdout and
// stderr to the current process, and returns the container's exit code.
func (m *Manager) Run(ctx context.Context, containerID string, tty bool) (int, error) {
	shortID := containerID[:12]

	// Put the host terminal into raw mode if TTY enabled.
	if tty {
		fd := os.Stdin.Fd()
		state, err := term.SetRawTerminal(fd)
		if err != nil {
			m.logger.Warn("failed to set raw terminal", zap.Error(err))
		} else {
			defer func() {
				if err := term.RestoreTerminal(fd, state); err != nil {
					m.logger.Warn("failed to restore terminal", zap.Error(err))
				}
			}()
		}

		// Helper to resize the container.
		resize := func() {
			ws, err := term.GetWinsize(fd)
			if err != nil {
				m.logger.Debug("failed to get window size", zap.Error(err))
				return
			}
			err = m.client.ContainerResize(ctx, containerID, container.ResizeOptions{
				Height: uint(ws.Height),
				Width:  uint(ws.Width),
			})
			if err != nil {
				m.logger.Debug("failed to resize container", zap.Error(err))
			}
		}

		// Initial resize with a short delay to ensure the container-side
		// TTY has been established and the application is ready.
		go func() {
			time.Sleep(100 * time.Millisecond)
			resize()
		}()

		// Handle terminal resize (SIGWINCH).
		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGWINCH)
			defer signal.Stop(sigChan)

			for {
				select {
				case <-ctx.Done():
					return
				case <-sigChan:
					resize()
				}
			}
		}()
	}

	m.logger.Info("starting container", zap.String("container_id", shortID))

	start := time.Now()

	attachResp, err := m.client.ContainerAttach(ctx, containerID, container.AttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
		Stdin:  tty,
	})
	if err != nil {
		return 1, fmt.Errorf("failed to attach to container %s: %w", shortID, err)
	}

	statusCh, errCh := m.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	if err := m.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		attachResp.Close()
		return 1, fmt.Errorf("failed to start container %s: %w", shortID, err)
	}

	if tty {
		go func() {
			if _, err := io.Copy(attachResp.Conn, os.Stdin); err != nil && err != io.EOF {
				m.logger.Debug("stdin copy ended", zap.Error(err))
			}
			if closer, ok := attachResp.Conn.(interface{ CloseWrite() error }); ok {
				_ = closer.CloseWrite()
			}
		}()
	}

	ioDone := make(chan struct{})
	go func() {
		defer close(ioDone)
		var err error
		if tty {
			_, err = io.Copy(os.Stdout, attachResp.Reader)
		} else {
			_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, attachResp.Reader)
		}

		if err != nil && err != io.EOF {
			// "use of closed network connection" is expected when we explicitly
			// close the attach response after the container exits.
			if !strings.Contains(err.Error(), "use of closed network connection") {
				m.logger.Warn("error copying container output",
					zap.String("container_id", shortID),
					zap.Error(err),
				)
			}
		}
	}()

	// Wait for the container to finish. We use ContainerWait to block until
	// the container reaches a non-running state, then inspect for exit code.
	select {
	case err := <-errCh:
		if err != nil {
			attachResp.Close()
			<-ioDone
			return 1, fmt.Errorf("error waiting for container %s: %w", shortID, err)
		}
	case <-statusCh:
	}

	<-ioDone
	attachResp.Close()

	// ContainerInspect gives us the definitive exit code from Docker's state.
	inspect, err := m.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return 1, fmt.Errorf("failed to inspect container %s: %w", shortID, err)
	}

	exitCode := inspect.State.ExitCode
	elapsed := time.Since(start)
	m.logger.Info("container completed",
		zap.String("container_id", shortID),
		zap.Int("exit_code", exitCode),
		zap.Duration("elapsed", elapsed),
	)

	return exitCode, nil
}

// Stop stops the container with a 10-second grace period.
func (m *Manager) Stop(ctx context.Context, containerID string) error {
	shortID := containerID[:12]
	m.logger.Info("stopping container", zap.String("container_id", shortID))

	timeout := defaultStopTimeout
	if err := m.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", shortID, err)
	}

	m.logger.Info("container stopped", zap.String("container_id", shortID))
	return nil
}

// Remove removes the container (forcefully if necessary).
func (m *Manager) Remove(ctx context.Context, containerID string) error {
	shortID := containerID[:12]
	m.logger.Debug("removing container", zap.String("container_id", shortID))

	if err := m.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", shortID, err)
	}

	m.logger.Debug("container removed", zap.String("container_id", shortID))
	return nil
}

// Inspect returns the container's low-level state from the Docker daemon.
func (m *Manager) Inspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	return m.client.ContainerInspect(ctx, containerID)
}

// Prune removes all stopped sandbox containers (those with label sandbox=true).
func (m *Manager) Prune(ctx context.Context) error {
	m.logger.Info("pruning stopped sandbox containers")

	report, err := m.client.ContainersPrune(ctx, filters.NewArgs(filters.Arg("label", "sandbox=true")))
	if err != nil {
		return fmt.Errorf("failed to prune containers: %w", err)
	}

	m.logger.Info("pruned containers",
		zap.Int("count", len(report.ContainersDeleted)),
		zap.Uint64("space_reclaimed", report.SpaceReclaimed),
	)
	return nil
}
