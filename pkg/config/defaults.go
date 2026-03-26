package config

import "github.com/spf13/viper"

// applyDefaults registers all hardcoded default values with the Viper instance.
//
// These values are the lowest priority and will be overridden by the config
// file or any CLI flags.
func applyDefaults(v *viper.Viper) {
	v.SetDefault("images.default", "alpine:latest")
	v.SetDefault("images.claude", "ghcr.io/servusdei2018/sandbox-claude:latest")
	v.SetDefault("images.gemini", "ghcr.io/servusdei2018/sandbox-gemini:latest")
	v.SetDefault("images.codex", "ghcr.io/servusdei2018/sandbox-codex:latest")
	v.SetDefault("images.kilocode", "ghcr.io/servusdei2018/sandbox-kilocode:latest")
	v.SetDefault("images.opencode", "ghcr.io/servusdei2018/sandbox-opencode:latest")
	v.SetDefault("images.python", "python:3.13-alpine")
	v.SetDefault("images.node", "node:24-alpine")
	v.SetDefault("images.bun", "oven/bun:alpine")
	v.SetDefault("images.go", "golang:1.26-alpine")

	// Environment variable whitelist: pass these from host if they exist.
	v.SetDefault("env_whitelist", []string{
		"LANG",
		"LC_ALL",
		"LC_CTYPE",
		"SHELL",
		"TERM",
		"COLORTERM",
		"XTERM_VERSION",
		"TZ",
	})

	// Environment variable blocklist: never pass these regardless of whitelist.
	v.SetDefault("env_blocklist", []string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
		"AWS_*",
		"GCP_*",
		"GOOGLE_APPLICATION_CREDENTIALS",
		"GITHUB_TOKEN",
		"GIT_PASSWORD",
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"COHERE_API_KEY",
	})

	// Container defaults.
	v.SetDefault("container.timeout", "30m")
	v.SetDefault("container.network_mode", "bridge")
	v.SetDefault("container.remove", true)

	// Logging defaults.
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "console")

	// Path defaults.
	v.SetDefault("paths.workspace", "/work")
	v.SetDefault("paths.config_dir", "~/.sandbox")
	v.SetDefault("paths.cache_dir", "~/.sandbox/cache")
}
