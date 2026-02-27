package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadFromYAML(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`projects:
  - my-project
  - other-project
idle_days: 14
stale_days: 60
min_monthly_cost: 5.0
format: json
timeout: 5m
exclude:
  resource_ids:
    - "1234567890"
  labels:
    - "env=dev"
`)
	if err := os.WriteFile(filepath.Join(dir, ".gcpspectre.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.Projects) != 2 {
		t.Errorf("Projects len = %d, want 2", len(cfg.Projects))
	}
	if cfg.IdleDays != 14 {
		t.Errorf("IdleDays = %d, want 14", cfg.IdleDays)
	}
	if cfg.StaleDays != 60 {
		t.Errorf("StaleDays = %d, want 60", cfg.StaleDays)
	}
	if cfg.MinMonthlyCost != 5.0 {
		t.Errorf("MinMonthlyCost = %f, want 5.0", cfg.MinMonthlyCost)
	}
	if cfg.Format != "json" {
		t.Errorf("Format = %q, want json", cfg.Format)
	}
	if len(cfg.Exclude.ResourceIDs) != 1 {
		t.Errorf("Exclude.ResourceIDs len = %d, want 1", len(cfg.Exclude.ResourceIDs))
	}
}

func TestLoadFromYML(t *testing.T) {
	dir := t.TempDir()
	content := []byte("idle_days: 3\n")
	if err := os.WriteFile(filepath.Join(dir, ".gcpspectre.yml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.IdleDays != 3 {
		t.Errorf("IdleDays = %d, want 3", cfg.IdleDays)
	}
}

func TestLoadNoFile(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.IdleDays != 0 {
		t.Errorf("IdleDays = %d, want 0 (zero value)", cfg.IdleDays)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gcpspectre.yaml"), []byte("{{invalid"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestTimeoutDuration(t *testing.T) {
	tests := []struct {
		timeout string
		want    time.Duration
	}{
		{"5m", 5 * time.Minute},
		{"30s", 30 * time.Second},
		{"", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		cfg := Config{Timeout: tt.timeout}
		got := cfg.TimeoutDuration()
		if got != tt.want {
			t.Errorf("TimeoutDuration(%q) = %v, want %v", tt.timeout, got, tt.want)
		}
	}
}

func TestYAMLPriorityOverYML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gcpspectre.yaml"), []byte("idle_days: 10\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gcpspectre.yml"), []byte("idle_days: 20\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.IdleDays != 10 {
		t.Errorf("IdleDays = %d, want 10 (YAML takes priority over YML)", cfg.IdleDays)
	}
}
