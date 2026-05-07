package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// ProjectManifest represents a .sandbox.yml file in a user's repository.
type ProjectManifest struct {
	// Setup is a list of shell commands to run before executing the agent.
	Setup []string `yaml:"setup"`
}

// Validate performs structural validation on the manifest, returning a
// user-friendly error if any field is malformed.
func (m *ProjectManifest) Validate() error {
	var errs []string

	for i, cmd := range m.Setup {
		trimmed := strings.TrimSpace(cmd)
		if trimmed == "" {
			errs = append(errs, fmt.Sprintf("setup[%d]: command must not be empty or whitespace-only", i))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid .sandbox.yml:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// LoadManifest attempts to load a .sandbox.yml from the specified directory.
// If the file does not exist, it returns an empty manifest and no error.
// Unknown YAML keys are rejected to catch typos early.
func LoadManifest(workspaceDir string, logger *zap.Logger) (*ProjectManifest, error) {
	manifestPath := filepath.Join(workspaceDir, ".sandbox.yml")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Debug("no .sandbox.yml manifest found in workspace", zap.String("path", workspaceDir))
			return &ProjectManifest{}, nil
		}
		return nil, err
	}

	logger.Debug("reading project manifest", zap.String("path", manifestPath))

	var manifest ProjectManifest

	// Decode with KnownFields so typos like "steup" are caught immediately.
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		if errors.Is(err, io.EOF) {
			// Empty file is a valid no-op manifest.
			return &ProjectManifest{}, nil
		}
		return nil, fmt.Errorf("failed to parse %s: %w", manifestPath, err)
	}

	// Structural validation.
	if err := manifest.Validate(); err != nil {
		return nil, err
	}

	logger.Info("manifest loaded successfully",
		zap.String("path", manifestPath),
		zap.Int("setup_commands", len(manifest.Setup)),
	)

	return &manifest, nil
}
