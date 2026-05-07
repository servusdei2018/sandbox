package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1536 * 1024, "1.5 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatSize(%d) = %q, expected %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestGetStats(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "cache_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Test with non-existent directory
	size, count, err := GetStats(filepath.Join(tmpDir, "nonexistent"))
	if err != nil {
		t.Errorf("Expected no error for nonexistent dir, got %v", err)
	}
	if size != 0 || count != 0 {
		t.Errorf("Expected 0 size and 0 count for nonexistent dir, got %d, %d", size, count)
	}

	// Create some files
	file1 := filepath.Join(tmpDir, "file1.txt")
	err = os.WriteFile(file1, []byte("hello"), 0644) // 5 bytes
	if err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}

	subdir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subdir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	file2 := filepath.Join(subdir, "file2.txt")
	err = os.WriteFile(file2, []byte("world!"), 0644) // 6 bytes
	if err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}

	// Calculate stats
	size, count, err = GetStats(tmpDir)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedSize := int64(11) // 5 + 6
	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

func TestClean(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "cache_clean_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Test with non-existent directory
	err = Clean(filepath.Join(tmpDir, "nonexistent"))
	if err != nil {
		t.Errorf("Expected no error for nonexistent dir, got %v", err)
	}

	// Populate the directory
	file1 := filepath.Join(tmpDir, "file1.txt")
	_ = os.WriteFile(file1, []byte("test"), 0644)

	subdir := filepath.Join(tmpDir, "subdir")
	_ = os.Mkdir(subdir, 0755)
	file2 := filepath.Join(subdir, "file2.txt")
	_ = os.WriteFile(file2, []byte("test"), 0644)

	// Verify it's not empty
	entries, _ := os.ReadDir(tmpDir)
	if len(entries) == 0 {
		t.Fatalf("Expected directory to not be empty")
	}

	// Clean the directory
	err = Clean(tmpDir)
	if err != nil {
		t.Errorf("Unexpected error during clean: %v", err)
	}

	// Verify directory still exists but is empty
	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("Expected directory to still exist, got error: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("Expected path to still be a directory")
	}

	entries, _ = os.ReadDir(tmpDir)
	if len(entries) != 0 {
		t.Errorf("Expected directory to be empty, found %d entries", len(entries))
	}
}
