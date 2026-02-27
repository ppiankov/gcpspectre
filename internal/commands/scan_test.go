package commands

import (
	"bytes"
	"testing"

	"github.com/ppiankov/gcpspectre/internal/report"
)

func TestSelectReporter(t *testing.T) {
	tests := []struct {
		format  string
		wantErr bool
	}{
		{"text", false},
		{"json", false},
		{"sarif", false},
		{"spectrehub", false},
		{"unknown", true},
	}
	for _, tt := range tests {
		r, err := selectReporter(tt.format, "")
		if tt.wantErr {
			if err == nil {
				t.Errorf("selectReporter(%q) expected error", tt.format)
			}
			continue
		}
		if err != nil {
			t.Errorf("selectReporter(%q) error: %v", tt.format, err)
			continue
		}
		if r == nil {
			t.Errorf("selectReporter(%q) returned nil", tt.format)
		}
	}
}

func TestSelectReporterFile(t *testing.T) {
	path := t.TempDir() + "/out.json"
	r, err := selectReporter("json", path)
	if err != nil {
		t.Fatalf("selectReporter with file: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil reporter")
	}
	// Verify it's a JSONReporter
	if _, ok := r.(*report.JSONReporter); !ok {
		t.Errorf("expected JSONReporter, got %T", r)
	}
}

func TestResolveProjects(t *testing.T) {
	// Save and restore global state
	oldProjects := projects
	oldCfg := cfg
	defer func() {
		projects = oldProjects
		cfg = oldCfg
	}()

	// Flag projects take priority
	projects = []string{"flag-proj"}
	cfg.Projects = []string{"config-proj"}
	got := resolveProjects()
	if len(got) != 1 || got[0] != "flag-proj" {
		t.Errorf("resolveProjects with flag = %v, want [flag-proj]", got)
	}

	// Fall back to config
	projects = nil
	got = resolveProjects()
	if len(got) != 1 || got[0] != "config-proj" {
		t.Errorf("resolveProjects with config = %v, want [config-proj]", got)
	}

	// No projects
	cfg.Projects = nil
	got = resolveProjects()
	if got != nil {
		t.Errorf("resolveProjects with nothing = %v, want nil", got)
	}
}

func TestApplyConfigDefaults(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg.Format = "json"
	cfg.IdleDays = 14
	cfg.StaleDays = 180
	cfg.MinMonthlyCost = 5.0

	// Reset to default values (which triggers config override)
	scanFlags.format = "text"
	scanFlags.idleDays = 7
	scanFlags.staleDays = 90
	scanFlags.minMonthlyCost = 1.0

	applyConfigDefaults()

	if scanFlags.format != "json" {
		t.Errorf("format = %q, want json", scanFlags.format)
	}
	if scanFlags.idleDays != 14 {
		t.Errorf("idleDays = %d, want 14", scanFlags.idleDays)
	}
	if scanFlags.staleDays != 180 {
		t.Errorf("staleDays = %d, want 180", scanFlags.staleDays)
	}
	if scanFlags.minMonthlyCost != 5.0 {
		t.Errorf("minMonthlyCost = %f, want 5.0", scanFlags.minMonthlyCost)
	}
}

func TestSelectReporterTypes(t *testing.T) {
	r, _ := selectReporter("text", "")
	if _, ok := r.(*report.TextReporter); !ok {
		t.Errorf("text format: expected TextReporter, got %T", r)
	}
	r, _ = selectReporter("sarif", "")
	if _, ok := r.(*report.SARIFReporter); !ok {
		t.Errorf("sarif format: expected SARIFReporter, got %T", r)
	}
	r, _ = selectReporter("spectrehub", "")
	if _, ok := r.(*report.SpectreHubReporter); !ok {
		t.Errorf("spectrehub format: expected SpectreHubReporter, got %T", r)
	}
}

// Verify reporter can generate output without crashing
func TestReporterGenerate(t *testing.T) {
	var buf bytes.Buffer
	r := &report.TextReporter{Writer: &buf}
	err := r.Generate(report.Data{Tool: "test", Version: "0.0.0"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty output")
	}
}
