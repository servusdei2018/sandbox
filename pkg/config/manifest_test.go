package config

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestLoadManifest(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("no manifest", func(t *testing.T) {
		dir := t.TempDir()
		manifest, err := LoadManifest(dir, logger)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if manifest == nil {
			t.Fatal("expected non-nil manifest")
		}
		if len(manifest.Setup) != 0 {
			t.Errorf("expected empty setup, got %d items", len(manifest.Setup))
		}
	})

	t.Run("valid manifest", func(t *testing.T) {
		dir := t.TempDir()
		content := `
setup:
  - apt-get update
  - apt-get install -y jq
`
		err := os.WriteFile(filepath.Join(dir, ".sandbox.yml"), []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write test manifest: %v", err)
		}

		manifest, err := LoadManifest(dir, logger)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(manifest.Setup) != 2 {
			t.Fatalf("expected 2 setup commands, got %d", len(manifest.Setup))
		}
		if manifest.Setup[0] != "apt-get update" {
			t.Errorf("unexpected setup command: %s", manifest.Setup[0])
		}
		if manifest.Setup[1] != "apt-get install -y jq" {
			t.Errorf("unexpected setup command: %s", manifest.Setup[1])
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, ".sandbox.yml"), []byte("!!invalid: yaml: ["), 0644)
		if err != nil {
			t.Fatalf("failed to write test manifest: %v", err)
		}

		_, err = LoadManifest(dir, logger)
		if err == nil {
			t.Fatal("expected error parsing invalid yaml, got nil")
		}
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		dir := t.TempDir()
		content := `
steup:
  - echo typo
`
		err := os.WriteFile(filepath.Join(dir, ".sandbox.yml"), []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write test manifest: %v", err)
		}

		_, err = LoadManifest(dir, logger)
		if err == nil {
			t.Fatal("expected error for unknown key 'steup', got nil")
		}
	})

	t.Run("empty setup command rejected", func(t *testing.T) {
		dir := t.TempDir()
		content := `
setup:
  - apt-get update
  - "   "
  - echo hello
`
		err := os.WriteFile(filepath.Join(dir, ".sandbox.yml"), []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write test manifest: %v", err)
		}

		_, err = LoadManifest(dir, logger)
		if err == nil {
			t.Fatal("expected error for empty setup command, got nil")
		}
	})

	t.Run("empty manifest is valid", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, ".sandbox.yml"), []byte(""), 0644)
		if err != nil {
			t.Fatalf("failed to write test manifest: %v", err)
		}

		manifest, err := LoadManifest(dir, logger)
		if err != nil {
			t.Fatalf("expected no error for empty manifest, got %v", err)
		}
		if manifest == nil {
			t.Fatal("expected non-nil manifest")
		}
	})
}

func TestProjectManifest_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		m := &ProjectManifest{Setup: []string{"echo hello", "make build"}}
		if err := m.Validate(); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("empty setup list is valid", func(t *testing.T) {
		m := &ProjectManifest{}
		if err := m.Validate(); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("blank entry", func(t *testing.T) {
		m := &ProjectManifest{Setup: []string{"echo hello", "", "make build"}}
		if err := m.Validate(); err == nil {
			t.Error("expected error for blank entry, got nil")
		}
	})
}
