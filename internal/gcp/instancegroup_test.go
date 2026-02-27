package gcp

import (
	"context"
	"testing"
)

func TestInstanceGroupScanner_Type(t *testing.T) {
	s := NewInstanceGroupScanner(nil, "proj")
	if s.Type() != ResourceInstanceGroup {
		t.Errorf("Type() = %q, want %q", s.Type(), ResourceInstanceGroup)
	}
}

func TestInstanceGroupScanner_EmptyGroups(t *testing.T) {
	compute := &mockComputeAPI{
		instanceGroups: []InstanceGroupInfo{
			{ID: 1, Name: "empty-mig", Zone: "us-central1-a", Size: 0, IsManaged: true},
			{ID: 2, Name: "healthy-mig", Zone: "us-central1-a", Size: 3, IsManaged: true},
			{ID: 3, Name: "empty-uig", Zone: "us-central1-b", Size: 0, IsManaged: false},
		},
	}

	s := NewInstanceGroupScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 3 {
		t.Errorf("ResourcesScanned = %d, want 3", result.ResourcesScanned)
	}
	if len(result.Findings) != 2 {
		t.Fatalf("Findings = %d, want 2", len(result.Findings))
	}

	for _, f := range result.Findings {
		if f.ID != FindingEmptyInstanceGroup {
			t.Errorf("ID = %q, want %q", f.ID, FindingEmptyInstanceGroup)
		}
		if f.Severity != SeverityMedium {
			t.Errorf("Severity = %q, want %q", f.Severity, SeverityMedium)
		}
	}
}

func TestInstanceGroupScanner_ExcludeByID(t *testing.T) {
	compute := &mockComputeAPI{
		instanceGroups: []InstanceGroupInfo{
			{ID: 1, Name: "empty-mig", Zone: "us-central1-a", Size: 0},
		},
	}
	cfg := ScanConfig{Exclude: ExcludeConfig{ResourceIDs: map[string]bool{"1": true}}}

	s := NewInstanceGroupScanner(compute, "proj")
	result, err := s.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0", len(result.Findings))
	}
}

func TestInstanceGroupScanner_AllNonEmpty(t *testing.T) {
	compute := &mockComputeAPI{
		instanceGroups: []InstanceGroupInfo{
			{ID: 1, Name: "mig-1", Zone: "us-central1-a", Size: 3},
			{ID: 2, Name: "mig-2", Zone: "us-central1-b", Size: 1},
		},
	}

	s := NewInstanceGroupScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0 (all non-empty)", len(result.Findings))
	}
}

func TestInstanceGroupScanner_Empty(t *testing.T) {
	compute := &mockComputeAPI{}
	s := NewInstanceGroupScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 0 {
		t.Errorf("ResourcesScanned = %d, want 0", result.ResourcesScanned)
	}
}
