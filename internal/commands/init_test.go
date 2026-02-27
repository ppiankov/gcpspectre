package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteIfNotExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	err := writeIfNotExists(path, "content", false)
	if err != nil {
		t.Fatalf("writeIfNotExists: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "content" {
		t.Errorf("content = %q, want %q", string(data), "content")
	}
}

func TestWriteIfNotExistsAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	_ = os.WriteFile(path, []byte("original"), 0o644)

	err := writeIfNotExists(path, "new content", false)
	if err == nil {
		t.Fatal("expected error for existing file")
	}

	// Original content should be preserved
	data, _ := os.ReadFile(path)
	if string(data) != "original" {
		t.Errorf("content = %q, want %q (preserved)", string(data), "original")
	}
}

func TestWriteIfNotExistsForce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	_ = os.WriteFile(path, []byte("original"), 0o644)

	err := writeIfNotExists(path, "overwritten", true)
	if err != nil {
		t.Fatalf("writeIfNotExists --force: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "overwritten" {
		t.Errorf("content = %q, want %q (overwritten)", string(data), "overwritten")
	}
}

func TestWriteIfNotExistsNestedDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "test.yaml")

	err := writeIfNotExists(path, "nested", false)
	if err != nil {
		t.Fatalf("writeIfNotExists nested: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "nested" {
		t.Errorf("content = %q, want %q", string(data), "nested")
	}
}

func TestSampleConfigContent(t *testing.T) {
	if !strings.Contains(sampleConfig, "projects:") {
		t.Error("sample config should contain projects section")
	}
	if !strings.Contains(sampleConfig, "idle_days:") {
		t.Error("sample config should contain idle_days")
	}
	if !strings.Contains(sampleConfig, "stale_days:") {
		t.Error("sample config should contain stale_days")
	}
}
