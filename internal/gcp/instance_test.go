package gcp

import (
	"context"
	"testing"
	"time"
)

func TestInstanceScanner_Type(t *testing.T) {
	s := NewInstanceScanner(nil, nil, "proj")
	if s.Type() != ResourceInstance {
		t.Errorf("Type() = %q, want %q", s.Type(), ResourceInstance)
	}
}

func TestInstanceScanner_StoppedInstances(t *testing.T) {
	compute := &mockComputeAPI{
		instances: []ComputeInstance{
			{
				ID:          1,
				Name:        "stopped-vm",
				Zone:        "us-central1-a",
				MachineType: "e2-medium",
				Status:      "STOPPED",
				LastStarted: time.Now().AddDate(0, 0, -60),
			},
			{
				ID:          2,
				Name:        "recently-stopped",
				Zone:        "us-central1-a",
				MachineType: "e2-small",
				Status:      "STOPPED",
				LastStarted: time.Now().AddDate(0, 0, -5),
			},
		},
	}

	s := NewInstanceScanner(compute, nil, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7, StaleDays: 90})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 2 {
		t.Errorf("ResourcesScanned = %d, want 2", result.ResourcesScanned)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("Findings = %d, want 1", len(result.Findings))
	}

	f := result.Findings[0]
	if f.ID != FindingStoppedInstance {
		t.Errorf("ID = %q, want %q", f.ID, FindingStoppedInstance)
	}
	if f.ResourceName != "stopped-vm" {
		t.Errorf("ResourceName = %q, want %q", f.ResourceName, "stopped-vm")
	}
	if f.EstimatedMonthlyWaste == 0 {
		t.Error("expected non-zero waste estimate")
	}
}

func TestInstanceScanner_IdleCPU(t *testing.T) {
	compute := &mockComputeAPI{
		instances: []ComputeInstance{
			{ID: 1, Name: "idle-vm", Zone: "us-central1-a", MachineType: "e2-medium", Status: "RUNNING"},
			{ID: 2, Name: "busy-vm", Zone: "us-central1-a", MachineType: "e2-medium", Status: "RUNNING"},
		},
	}
	monitoring := &mockMonitoringAPI{
		results: map[string]float64{
			"1": 0.02, // 2% CPU (GCP reports as fraction 0.0–1.0)
			"2": 0.50, // 50% CPU
		},
	}

	s := NewInstanceScanner(compute, monitoring, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("Findings = %d, want 1", len(result.Findings))
	}

	f := result.Findings[0]
	if f.ID != FindingIdleInstance {
		t.Errorf("ID = %q, want %q", f.ID, FindingIdleInstance)
	}
	if f.ResourceName != "idle-vm" {
		t.Errorf("ResourceName = %q, want %q", f.ResourceName, "idle-vm")
	}
}

func TestInstanceScanner_ExcludeByID(t *testing.T) {
	compute := &mockComputeAPI{
		instances: []ComputeInstance{
			{ID: 1, Name: "stopped-vm", Zone: "us-central1-a", MachineType: "e2-medium", Status: "STOPPED", LastStarted: time.Now().AddDate(0, 0, -60)},
		},
	}
	cfg := ScanConfig{
		Exclude: ExcludeConfig{ResourceIDs: map[string]bool{"1": true}},
	}

	s := NewInstanceScanner(compute, nil, "proj")
	result, err := s.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0 (excluded)", len(result.Findings))
	}
}

func TestInstanceScanner_ExcludeByLabel(t *testing.T) {
	compute := &mockComputeAPI{
		instances: []ComputeInstance{
			{
				ID: 1, Name: "stopped-vm", Zone: "us-central1-a", MachineType: "e2-medium",
				Status: "STOPPED", LastStarted: time.Now().AddDate(0, 0, -60),
				Labels: map[string]string{"env": "production"},
			},
		},
	}
	cfg := ScanConfig{
		Exclude: ExcludeConfig{Labels: map[string]string{"env": "production"}},
	}

	s := NewInstanceScanner(compute, nil, "proj")
	result, err := s.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0 (excluded by label)", len(result.Findings))
	}
}

func TestInstanceScanner_Empty(t *testing.T) {
	compute := &mockComputeAPI{}
	s := NewInstanceScanner(compute, nil, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 0 {
		t.Errorf("ResourcesScanned = %d, want 0", result.ResourcesScanned)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0", len(result.Findings))
	}
}

func TestInstanceScanner_NilMonitoring(t *testing.T) {
	compute := &mockComputeAPI{
		instances: []ComputeInstance{
			{ID: 1, Name: "running-vm", Status: "RUNNING", Zone: "us-central1-a", MachineType: "e2-medium"},
		},
	}
	s := NewInstanceScanner(compute, nil, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0 (no monitoring)", len(result.Findings))
	}
}

func TestInstanceScanner_StoppedFallbackCreateTime(t *testing.T) {
	compute := &mockComputeAPI{
		instances: []ComputeInstance{
			{
				ID: 1, Name: "old-vm", Zone: "us-central1-a", MachineType: "e2-medium",
				Status:     "STOPPED",
				CreateTime: time.Now().AddDate(0, 0, -90),
				// LastStarted is zero — fallback to CreateTime
			},
		},
	}

	s := NewInstanceScanner(compute, nil, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("Findings = %d, want 1", len(result.Findings))
	}
	if result.Findings[0].ID != FindingStoppedInstance {
		t.Errorf("ID = %q, want %q", result.Findings[0].ID, FindingStoppedInstance)
	}
}
