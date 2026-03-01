package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ppiankov/gcpspectre/internal/analyzer"
	gcptype "github.com/ppiankov/gcpspectre/internal/gcp"
)

func sampleData() Data {
	return Data{
		Tool:      "gcpspectre",
		Version:   "0.1.0",
		Timestamp: time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC),
		Target:    Target{Type: "gcp_projects", URIHash: "sha256:abc123"},
		Config:    ReportConfig{Projects: []string{"my-project"}, IdleDays: 7, StaleDays: 90, MinMonthlyCost: 1.0},
		Findings: []gcptype.Finding{
			{
				ID: gcptype.FindingIdleInstance, Severity: gcptype.SeverityHigh,
				ResourceType: gcptype.ResourceInstance, ResourceID: "123456",
				ResourceName: "idle-vm", Project: "my-project", Zone: "us-central1-a",
				Message: "CPU < 5% over 7 days", EstimatedMonthlyWaste: 24.46,
			},
			{
				ID: gcptype.FindingUnusedAddress, Severity: gcptype.SeverityMedium,
				ResourceType: gcptype.ResourceAddress, ResourceID: "addr-1",
				Project: "my-project", Message: "Static IP not in use", EstimatedMonthlyWaste: 7.30,
			},
		},
		Summary: analyzer.Summary{
			TotalResourcesScanned: 50,
			TotalFindings:         2,
			TotalMonthlyWaste:     31.76,
			BySeverity:            map[string]int{"high": 1, "medium": 1},
			ByResourceType:        map[string]int{"compute_instance": 1, "static_ip": 1},
			ProjectsScanned:       1,
		},
	}
}

func TestJSONReporter(t *testing.T) {
	var buf bytes.Buffer
	r := &JSONReporter{Writer: &buf}
	if err := r.Generate(sampleData()); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"$schema": "spectre/v1"`) {
		t.Error("missing spectre/v1 schema")
	}
	if !strings.Contains(output, `"tool": "gcpspectre"`) {
		t.Error("missing tool field")
	}

	var envelope map[string]any
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestTextReporter(t *testing.T) {
	var buf bytes.Buffer
	r := &TextReporter{Writer: &buf}
	if err := r.Generate(sampleData()); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "gcpspectre") {
		t.Error("missing gcpspectre header")
	}
	if !strings.Contains(output, "idle-vm") {
		t.Error("missing finding resource name")
	}
	if !strings.Contains(output, "Summary") {
		t.Error("missing summary section")
	}
}

func TestTextReporterEmpty(t *testing.T) {
	data := sampleData()
	data.Findings = nil
	data.Summary.TotalFindings = 0

	var buf bytes.Buffer
	r := &TextReporter{Writer: &buf}
	if err := r.Generate(data); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !strings.Contains(buf.String(), "No idle resources found") {
		t.Error("expected 'No idle resources found' for empty findings")
	}
}

func TestTextReporterWithErrors(t *testing.T) {
	data := sampleData()
	data.Errors = []ScanError{{Scanner: "my-project", ResourceType: "compute_instance", Message: "timeout", Recoverable: true}}

	var buf bytes.Buffer
	r := &TextReporter{Writer: &buf}
	if err := r.Generate(data); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !strings.Contains(buf.String(), "Warnings (1)") {
		t.Error("expected warnings section")
	}
}

func TestSARIFReporter(t *testing.T) {
	var buf bytes.Buffer
	r := &SARIFReporter{Writer: &buf}
	if err := r.Generate(sampleData()); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var sarif map[string]any
	if err := json.Unmarshal(buf.Bytes(), &sarif); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if sarif["version"] != "2.1.0" {
		t.Errorf("SARIF version = %v, want 2.1.0", sarif["version"])
	}
	if !strings.Contains(buf.String(), "gcp://") {
		t.Error("expected gcp:// URI in SARIF locations")
	}
}

func TestSARIFReporterZoneInURI(t *testing.T) {
	data := sampleData()
	data.Findings = data.Findings[:1] // keep only the zoned finding

	var buf bytes.Buffer
	r := &SARIFReporter{Writer: &buf}
	if err := r.Generate(data); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !strings.Contains(buf.String(), "gcp://my-project/us-central1-a/compute_instance/123456") {
		t.Error("expected zone in SARIF URI for zonal resources")
	}
}

func TestSpectreHubReporter(t *testing.T) {
	var buf bytes.Buffer
	r := &SpectreHubReporter{Writer: &buf}
	if err := r.Generate(sampleData()); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"schema": "spectre/v1"`) {
		t.Error("missing spectre/v1 schema")
	}
}

func TestSarifLevel(t *testing.T) {
	tests := []struct {
		severity gcptype.Severity
		want     string
	}{
		{gcptype.SeverityHigh, "error"},
		{gcptype.SeverityMedium, "warning"},
		{gcptype.SeverityLow, "note"},
	}
	for _, tt := range tests {
		got := sarifLevel(tt.severity)
		if got != tt.want {
			t.Errorf("sarifLevel(%q) = %q, want %q", tt.severity, got, tt.want)
		}
	}
}

func TestBuildSARIFRules(t *testing.T) {
	rules := buildSARIFRules()
	if len(rules) != 19 {
		t.Errorf("expected 19 SARIF rules, got %d", len(rules))
	}
}

func TestFormatMapSorted(t *testing.T) {
	m := map[string]int{"z": 3, "a": 1, "m": 2}
	parts := formatMapSorted(m)
	if len(parts) != 3 {
		t.Fatalf("len = %d, want 3", len(parts))
	}
	if parts[0] != "a=1" {
		t.Errorf("parts[0] = %q, want a=1", parts[0])
	}
	if parts[2] != "z=3" {
		t.Errorf("parts[2] = %q, want z=3", parts[2])
	}
}
