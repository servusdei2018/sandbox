// Package agent provides agent detection and registry functionality.
package agent

import (
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// binaryToType maps known binary names (as received on the CLI) to their
// canonical Agent Type.
var binaryToType = map[string]Type{
	"claude":   TypeClaudeCode,
	"gemini":   TypeGeminiCLI,
	"codex":    TypeCodex,
	"kilo":     TypeKilocode,
	"kilocode": TypeKilocode,
	"opencode": TypeOpenCode,
	"python":   TypePython,
	"python3":  TypePython,
	"pip":      TypePython,
	"pip3":     TypePython,
	"node":     TypeNode,
	"npm":      TypeNode,
	"npx":      TypeNode,
	"bun":      TypeBun,
	"bunx":     TypeBun,
	"go":       TypeGo,
	"cargo":    TypeRust,
	"rustc":    TypeRust,
	"ruby":     TypeRuby,
	"gem":      TypeRuby,
	"php":      TypePHP,
	"composer": TypePHP,
	"java":     TypeJava,
	"javac":    TypeJava,
	"mvn":      TypeJava,
	"aider":    TypeAider,
}

// Detect inspects args and env to determine which agent is being invoked
// and returns an Agent with its default Docker image.
//
// Detection priority (highest → lowest):
//  1. AGENT_NAME environment variable
//  2. First element of args (binary name, path-stripped)
//  3. Generic fallback (alpine:latest)
//
// imageOverrides (from user config) may replace the default image.
func Detect(args []string, env map[string]string, imageOverrides map[string]string, logger *zap.Logger) (*Agent, error) {
	logger.Debug("detecting agent",
		zap.Strings("args", args),
	)

	agentType := TypeGeneric

	// Priority 1: $AGENT_NAME environment variable.
	if name, ok := env["AGENT_NAME"]; ok && name != "" {
		if t, matched := binaryToType[strings.ToLower(name)]; matched {
			agentType = t
			logger.Debug("agent detected via AGENT_NAME env var",
				zap.String("agent", string(agentType)),
			)
		}
	}

	// Priority 2: Parse the first CLI argument.
	if agentType == TypeGeneric && len(args) > 0 {
		// Strip any leading path components (e.g., "/usr/local/bin/python3" → "python3").
		binary := strings.ToLower(filepath.Base(args[0]))
		if t, matched := binaryToType[binary]; matched {
			agentType = t
			logger.Debug("agent detected from binary name",
				zap.String("binary", binary),
				zap.String("agent", string(agentType)),
			)
		}
	}

	// Resolve the image for this agent type, applying any user overrides.
	image := DefaultImage(agentType)
	if overrides, ok := imageOverrides[string(agentType)]; ok && overrides != "" {
		image = overrides
	}
	// Also check the "default" key for a global fallback override.
	if agentType == TypeGeneric {
		if dflt, ok := imageOverrides["default"]; ok && dflt != "" {
			image = dflt
		}
	}

	a := &Agent{
		Name:       agentType,
		Image:      image,
		Entrypoint: DefaultEntrypoint(agentType),
	}

	logger.Info("agent detected",
		zap.String("name", string(a.Name)),
		zap.String("image", a.Image),
	)

	return a, nil
}

// envToMap converts a slice of "KEY=VALUE" strings (as returned by os.Environ)
// into a map for convenient lookup.
func EnvToMap(environ []string) map[string]string {
	m := make(map[string]string, len(environ))
	for _, kv := range environ {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}

// HostEnv returns the current process environment as a map.
func HostEnv() map[string]string {
	return EnvToMap(os.Environ())
}
