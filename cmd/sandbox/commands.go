package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/servusdei2018/sandbox/pkg/agent"
	"github.com/servusdei2018/sandbox/pkg/config"
	cnt "github.com/servusdei2018/sandbox/pkg/container"
	sandboxlog "github.com/servusdei2018/sandbox/pkg/log"
)

// ExitError is a custom error type that carries an exit code.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}

// runCmd returns the "sandbox run <agent> [args...]" subcommand.
func runCmd() *cobra.Command {
	var (
		imageFlagOverride string
		timeoutFlag       string
		workspaceFlag     string
		keepContainer     bool
		seccompFlag       string
	)

	cmd := &cobra.Command{
		Use:   "run <binary> [args...]",
		Short: "Run a command or agent in a sandbox container",
		Long: `Run executes the given binary (and optional arguments) inside an isolated
Docker container, with the current working directory bind-mounted to /work.

Examples:
  sandbox run echo hello
  sandbox run python -c "print('hello')"
  sandbox run claude --help`,
		Args: cobra.MinimumNArgs(1),
		// DisableFlagParsing would break our own flags (--image, --timeout, etc.).
		// Instead, SetInterspersed(false) below stops pflag from consuming agent
		// flags like -c, -e, --help that appear after the first positional arg.
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := sandboxlog.Logger
			if logger == nil {
				logger = sandboxlog.Noop()
			}

			logger.Info("sandbox run requested",
				zap.Strings("args", args),
			)

			cfg, err := config.Load(logger)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if cfgWriteErr := config.WriteDefault(logger); cfgWriteErr != nil {
				logger.Warn("could not write default config file", zap.Error(cfgWriteErr))
			}

			// Apply config-level image overrides to the agent registry.
			agent.OverrideImages(cfg.Images)

			hostEnv := agent.HostEnv()
			detectedAgent, err := agent.Detect(args, hostEnv, cfg.Images, logger)
			if err != nil {
				return fmt.Errorf("agent detection failed: %w", err)
			}

			// Determine final image: CLI flag wins, then detected agent image.
			finalImage := detectedAgent.Image
			if imageFlagOverride != "" {
				finalImage = imageFlagOverride
				logger.Info("image overridden by flag", zap.String("image", finalImage))
			}

			rawTimeout := cfg.Container.Timeout
			if timeoutFlag != "" {
				rawTimeout = timeoutFlag
			}
			timeout, err := time.ParseDuration(rawTimeout)
			if err != nil {
				return fmt.Errorf("invalid timeout %q: %w", rawTimeout, err)
			}
			logger.Info("parsed timeout", zap.Duration("duration", timeout))

			wsDir := workspaceFlag
			if wsDir == "" {
				wsDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("could not determine current directory: %w", err)
				}
			}

			// Resolve symlinks so Docker bind-mount works on all systems.
			wsDir, err = filepath.EvalSymlinks(wsDir)
			if err != nil {
				return fmt.Errorf("failed to resolve workspace path %s: %w", wsDir, err)
			}

			filteredEnv := cnt.FilterEnv(hostEnv, cfg.EnvWhitelist, cfg.EnvBlocklist, logger)

			manager, err := cnt.NewManager(logger)
			if err != nil {
				return fmt.Errorf("could not connect to Docker daemon: %w", err)
			}
			defer func() {
				if err := manager.Close(); err != nil {
					logger.Warn("failed to close docker manager", zap.Error(err))
				}
			}()

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			if err := manager.PullIfMissing(ctx, finalImage); err != nil {
				return fmt.Errorf("image pull failed: %w", err)
			}

			containerCmd := args
			var entrypoint []string
			// If an entrypoint is defined for this agent, use it.
			if len(detectedAgent.Entrypoint) > 0 {
				entrypoint = detectedAgent.Entrypoint

				// Strip the binary name from args if it matches the agent's expected binary.
				// This allows "sandbox run claude --help" to map correctly
				// when "claude" is the entrypoint.
				if len(args) > 0 && detectedAgent.Name != agent.TypeGeneric {
					containerCmd = args[1:]
				}
			}

			containerCfg := &cnt.Config{
				Image:        finalImage,
				Cmd:          containerCmd,
				Entrypoint:   entrypoint,
				WorkspaceDir: wsDir,
				MountTarget:  cfg.Paths.Workspace,
				Env:          filteredEnv,
				Timeout:      timeout,
				RemoveOnExit: cfg.Container.Remove && !keepContainer,
				NetworkMode:  cfg.Container.NetworkMode,
				Tty:          true,
				AttachStdin:  true,
				Security:     cfg.Security,
			}

			if seccompFlag != "" {
				containerCfg.Security.SeccompProfilePath = seccompFlag
			}

			containerID, err := manager.Create(ctx, containerCfg)
			if err != nil {
				return fmt.Errorf("sandbox setup failed: %w", err)
			}

			// We use signal.NotifyContext above to handle graceful shutdown.
			// The context will be canceled on SIGINT/SIGTERM, which will cause
			// manager.Run to return (as it blocks on ContainerWait which
			// respects context cancellation).

			exitCode, err := manager.Run(ctx, containerID, containerCfg.Tty)
			if err != nil {
				// If the context was canceled, it's likely a signal or timeout.
				if ctx.Err() != nil {
					if errors.Is(ctx.Err(), context.DeadlineExceeded) {
						logger.Warn("container execution timed out",
							zap.Duration("timeout", timeout),
						)
					} else {
						logger.Info("container execution interrupted", zap.Error(ctx.Err()))
					}

					// Try to stop the container before returning.
					stopCtx, stopCancel := context.WithTimeout(context.Background(), 15*time.Second)
					defer stopCancel()
					_ = manager.Stop(stopCtx, containerID)
					return &ExitError{Code: 130}
				}

				logger.Error("container execution failed",
					zap.String("agent", string(detectedAgent.Name)),
					zap.String("image", finalImage),
					zap.Error(err),
				)
				return fmt.Errorf("sandbox execution failed: %w", err)
			}

			// Cleanup: remove container unless --keep was set.
			if containerCfg.RemoveOnExit {
				cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cleanupCancel()
				if err := manager.Remove(cleanupCtx, containerID); err != nil {
					logger.Warn("failed to remove container",
						zap.String("container_id", containerID[:12]),
						zap.Error(err),
					)
				}
			}

			if exitCode != 0 {
				return &ExitError{Code: exitCode}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&imageFlagOverride, "image", "i", "", "Docker image to use (overrides auto-detection)")
	cmd.Flags().StringVarP(&timeoutFlag, "timeout", "t", "", "Maximum execution time (e.g. 30m, 1h)")
	cmd.Flags().StringVarP(&workspaceFlag, "workspace", "w", "", "Host directory to mount as /work (defaults to cwd)")
	cmd.Flags().BoolVarP(&keepContainer, "keep", "k", false, "Do not remove the container after execution")
	cmd.Flags().StringVar(&seccompFlag, "seccomp", "", "Path to a custom seccomp JSON profile")

	// Stop flag parsing at the first non-flag argument so agent flags
	// (e.g. python -c, node -e, claude --help) are not consumed by Cobra.
	cmd.Flags().SetInterspersed(false)

	return cmd
}

// configCmd returns the "sandbox config show" subcommand.
func configCmd() *cobra.Command {
	parent := &cobra.Command{
		Use:   "config",
		Short: "Manage sandbox configuration",
	}

	parent.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show the current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := sandboxlog.Logger
			if logger == nil {
				logger = sandboxlog.Noop()
			}

			cfg, err := config.Load(logger)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			fmt.Printf("Workspace mount point : %s\n", cfg.Paths.Workspace)
			fmt.Printf("Config directory      : %s\n", cfg.Paths.ConfigDir)
			fmt.Printf("Cache directory       : %s\n", cfg.Paths.CacheDir)
			fmt.Printf("Container timeout     : %s\n", cfg.Container.Timeout)
			fmt.Printf("Network mode          : %s\n", cfg.Container.NetworkMode)
			fmt.Printf("Remove on exit        : %v\n", cfg.Container.Remove)
			fmt.Printf("Log level             : %s\n", cfg.Logging.Level)
			fmt.Printf("Log format            : %s\n", cfg.Logging.Format)
			fmt.Printf("Default image         : %s\n", cfg.Images["default"])
			return nil
		},
	})

	parent.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Create the default config file at ~/.sandbox/config.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := sandboxlog.Logger
			if logger == nil {
				logger = sandboxlog.Noop()
			}
			return config.WriteDefault(logger)
		},
	})

	return parent
}

// pruneCmd returns the "sandbox prune" subcommand.
func pruneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "Remove stopped sandbox containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := sandboxlog.Logger
			if logger == nil {
				logger = sandboxlog.Noop()
			}

			manager, err := cnt.NewManager(logger)
			if err != nil {
				return fmt.Errorf("could not connect to Docker daemon: %w", err)
			}
			defer func() {
				if err := manager.Close(); err != nil {
					logger.Warn("failed to close docker manager", zap.Error(err))
				}
			}()

			return manager.Prune(context.Background())
		},
	}
}
