// Package agent provides agent detection and registry functionality.
package agent

// Type represents a named agent type supported by the sandbox.
type Type string

const (
	TypeClaudeCode Type = "claude-code"
	TypeGeminiCLI  Type = "gemini-cli"
	TypeCodex      Type = "codex"
	TypeKilocode   Type = "kilocode"
	TypeOpenCode   Type = "opencode"
	TypePython     Type = "python"
	TypeNode       Type = "node"
	TypeBun        Type = "bun"
	TypeGo         Type = "go"
	TypeGeneric    Type = "generic"
)

// Agent holds metadata for a detected agent.
type Agent struct {
	// Name is the canonical agent type identifier (e.g. "claude-code").
	Name Type

	// Image is the default Docker image to use for this agent.
	Image string

	// Entrypoint is an optional entrypoint override for the container.
	// If empty the image's default ENTRYPOINT/CMD is used.
	Entrypoint []string
}

// entry is an internal registry record mapping an agent type to its defaults.
type entry struct {
	Image      string
	Entrypoint []string
}

// defaultRegistry maps agent types to their default images. Individual entries
// may be overridden by user configuration.
var defaultRegistry = map[Type]entry{
	TypeClaudeCode: {
		Image:      "ghcr.io/servusdei2018/sandbox-claude:latest",
		Entrypoint: []string{"claude"},
	},
	TypeGeminiCLI: {
		Image:      "ghcr.io/servusdei2018/sandbox-gemini:latest",
		Entrypoint: []string{"gemini"},
	},
	TypeCodex: {
		Image:      "ghcr.io/servusdei2018/sandbox-codex:latest",
		Entrypoint: []string{"codex"},
	},
	TypeKilocode: {
		Image:      "ghcr.io/servusdei2018/sandbox-kilocode:latest",
		Entrypoint: []string{"kilo"},
	},
	TypeOpenCode: {
		Image:      "ghcr.io/servusdei2018/sandbox-opencode:latest",
		Entrypoint: []string{"opencode"},
	},
	TypePython:  {Image: "python:3.13-alpine"},
	TypeNode:    {Image: "node:24-alpine"},
	TypeBun:     {Image: "oven/bun:alpine"},
	TypeGo:      {Image: "golang:1.26-alpine"},
	TypeGeneric: {Image: "alpine:latest"},
}

// DefaultImage returns the default Docker image for the given agent type.
// If the type is not registered, "alpine:latest" is returned as a safe fallback.
func DefaultImage(t Type) string {
	if e, ok := defaultRegistry[t]; ok {
		return e.Image
	}
	return "alpine:latest"
}

// DefaultEntrypoint returns the default entrypoint for the given agent type.
func DefaultEntrypoint(t Type) []string {
	if e, ok := defaultRegistry[t]; ok {
		return e.Entrypoint
	}
	return nil
}

// OverrideImages merges a user-supplied image map (from config) into the
// registry, allowing per-agent image overrides without replacing the entire
// registry.
func OverrideImages(overrides map[string]string) {
	for k, img := range overrides {
		t := Type(k)
		e := defaultRegistry[t]
		e.Image = img
		defaultRegistry[t] = e
	}
}
