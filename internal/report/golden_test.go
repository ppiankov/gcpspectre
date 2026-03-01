package report

import (
	"bytes"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/ppiankov/gcpspectre/internal/analyzer"
	gcptype "github.com/ppiankov/gcpspectre/internal/gcp"
)

var update = flag.Bool("update", false, "Update golden files")

func goldenData() Data {
	return Data{
		Tool:      "gcpspectre",
		Version:   "0.1.0",
		Timestamp: time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
		Target:    Target{Type: "gcp-projects", URIHash: "sha256:abc123"},
		Config:    ReportConfig{Projects: []string{"test-project"}, IdleDays: 7, StaleDays: 90, MinMonthlyCost: 1.0},
		Findings: []gcptype.Finding{
			{
				ID: gcptype.FindingIdleInstance, Severity: gcptype.SeverityHigh,
				ResourceType: gcptype.ResourceInstance, ResourceID: "100",
				ResourceName: "idle-vm", Project: "test-project", Zone: "us-central1-a",
				Message: "CPU 2.0% over 7 days", EstimatedMonthlyWaste: 24.46,
				Metadata: map[string]any{"machine_type": "e2-medium", "avg_cpu_percent": 2.0},
			},
			{
				ID: gcptype.FindingDetachedDisk, Severity: gcptype.SeverityHigh,
				ResourceType: gcptype.ResourceDisk, ResourceID: "200",
				ResourceName: "orphan-disk", Project: "test-project", Zone: "us-central1-a",
				Message: "Detached 30 days, pd-ssd 100 GiB", EstimatedMonthlyWaste: 17.00,
				Metadata: map[string]any{"disk_type": "pd-ssd", "size_gib": 100},
			},
			{
				ID: gcptype.FindingUnusedAddress, Severity: gcptype.SeverityMedium,
				ResourceType: gcptype.ResourceAddress, ResourceID: "addr-1",
				Project: "test-project", Message: "Static IP not in use", EstimatedMonthlyWaste: 7.30,
			},
		},
		Summary: analyzer.Summary{
			TotalResourcesScanned: 50,
			TotalFindings:         3,
			TotalMonthlyWaste:     48.76,
			BySeverity:            map[string]int{"high": 2, "medium": 1},
			ByResourceType:        map[string]int{"compute_instance": 1, "persistent_disk": 1, "static_ip": 1},
			ProjectsScanned:       1,
		},
	}
}

func TestGoldenJSON(t *testing.T) {
	var buf bytes.Buffer
	r := &JSONReporter{Writer: &buf}
	if err := r.Generate(goldenData()); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	compareOrUpdate(t, "testdata/report.json", buf.Bytes())
}

func TestGoldenSARIF(t *testing.T) {
	var buf bytes.Buffer
	r := &SARIFReporter{Writer: &buf}
	if err := r.Generate(goldenData()); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	compareOrUpdate(t, "testdata/report.sarif", buf.Bytes())
}

func compareOrUpdate(t *testing.T, path string, got []byte) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("update golden: %v", err)
		}
		t.Logf("Updated golden file: %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update to create)", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("output differs from golden file %s (run with -update to refresh)\ngot:\n%s", path, got)
	}
}
