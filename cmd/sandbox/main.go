// main is the entry point for the sandbox CLI. It sets up the Cobra command
// tree and delegates execution to the appropriate subcommand.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	sandboxlog "github.com/servusdei2018/sandbox/pkg/log"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "sandbox",
		Short: "Run coding agents in isolated Docker containers",
		Long: `sandbox runs coding agents (Claude Code,
Gemini CLI, Codex, Kiro, OpenCode, and more) inside isolated Docker containers
with the current working directory bind-mounted to /work.`,
		Version:       Version,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			logLevel, _ := cmd.Flags().GetString("log-level")
			logFormat, _ := cmd.Flags().GetString("log-format")
			if logLevel == "" {
				logLevel = "error"
			}
			if logFormat == "" {
				logFormat = "console"
			}
			if err := sandboxlog.Init(logLevel, logFormat); err != nil {
				return fmt.Errorf("failed to initialize logger: %w", err)
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().String("log-level", "error", "Log level: debug, info, warn, error")
	rootCmd.PersistentFlags().String("log-format", "console", "Log format: console, json")

	rootCmd.AddCommand(runCmd())
	rootCmd.AddCommand(pruneCmd())
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(configCmd())

	if err := rootCmd.Execute(); err != nil {
		var exitErr *ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}

		// Logger may not be initialised yet if PersistentPreRunE wasn't called.
		if sandboxlog.Logger != nil {
			sandboxlog.Logger.Error("command failed", zap.Error(err))
		} else {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		os.Exit(1)
	}
}
