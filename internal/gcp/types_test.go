package gcp

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFindingJSONRoundTrip(t *testing.T) {
	f := Finding{
		ID:                    FindingIdleInstance,
		Severity:              SeverityHigh,
		ResourceType:          ResourceInstance,
		ResourceID:            "1234567890",
		ResourceName:          "my-vm",
		Project:               "my-project",
		Zone:                  "us-central1-a",
		Message:               "CPU < 5% over 7 days",
		EstimatedMonthlyWaste: 42.50,
		Metadata:              map[string]any{"cpu_avg": 2.3},
	}

	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Finding
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.ID != f.ID {
		t.Errorf("ID = %q, want %q", got.ID, f.ID)
	}
	if got.Project != f.Project {
		t.Errorf("Project = %q, want %q", got.Project, f.Project)
	}
	if got.Zone != f.Zone {
		t.Errorf("Zone = %q, want %q", got.Zone, f.Zone)
	}
	if got.EstimatedMonthlyWaste != f.EstimatedMonthlyWaste {
		t.Errorf("EstimatedMonthlyWaste = %f, want %f", got.EstimatedMonthlyWaste, f.EstimatedMonthlyWaste)
	}
}

func TestScanResultJSONRoundTrip(t *testing.T) {
	sr := ScanResult{
		Findings: []Finding{
			{ID: FindingDetachedDisk, Severity: SeverityHigh, ResourceType: ResourceDisk, ResourceID: "disk-1", Project: "proj-a"},
		},
		Errors:           []string{"zone-b: timeout"},
		ResourcesScanned: 50,
		ProjectsScanned:  2,
	}

	data, err := json.Marshal(sr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ScanResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got.Findings) != 1 {
		t.Fatalf("Findings len = %d, want 1", len(got.Findings))
	}
	if got.ProjectsScanned != 2 {
		t.Errorf("ProjectsScanned = %d, want 2", got.ProjectsScanned)
	}
	if len(got.Errors) != 1 {
		t.Errorf("Errors len = %d, want 1", len(got.Errors))
	}
}

func TestScanProgressFields(t *testing.T) {
	now := time.Now()
	p := ScanProgress{
		Project:   "my-project",
		Scanner:   "instance",
		Message:   "scanning zone us-central1-a",
		Timestamp: now,
	}
	if p.Project != "my-project" {
		t.Errorf("Project = %q, want my-project", p.Project)
	}
	if p.Timestamp != now {
		t.Errorf("Timestamp mismatch")
	}
}

func TestResourceTypeConstants(t *testing.T) {
	types := []ResourceType{
		ResourceInstance, ResourceDisk, ResourceAddress,
		ResourceSnapshot, ResourceInstanceGroup, ResourceCloudSQL, ResourceFirewall,
	}
	seen := make(map[ResourceType]bool)
	for _, rt := range types {
		if seen[rt] {
			t.Errorf("duplicate ResourceType: %s", rt)
		}
		seen[rt] = true
	}
	if len(types) != 7 {
		t.Errorf("expected 7 resource types, got %d", len(types))
	}
}

func TestFindingIDConstants(t *testing.T) {
	ids := []FindingID{
		FindingIdleInstance, FindingStoppedInstance, FindingDetachedDisk,
		FindingUnusedAddress, FindingStaleSnapshot, FindingEmptyInstanceGroup,
		FindingUnhealthyInstanceGroup, FindingIdleCloudSQL, FindingUnusedFirewall,
	}
	seen := make(map[FindingID]bool)
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate FindingID: %s", id)
		}
		seen[id] = true
	}
	if len(ids) != 9 {
		t.Errorf("expected 9 finding IDs, got %d", len(ids))
	}
}
