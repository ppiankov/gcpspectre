package gcp

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestMultiProjectScanner_ScanAll(t *testing.T) {
	compute := &mockComputeAPI{
		instances: []ComputeInstance{
			{ID: 1, Name: "stopped-vm", Zone: "us-central1-a", MachineType: "e2-medium", Status: "STOPPED", LastStarted: time.Now().AddDate(0, 0, -60)},
		},
		disks: []PersistentDisk{
			{ID: 1, Name: "orphan-disk", Zone: "us-central1-a", DiskType: "pd-ssd", SizeGB: 100, CreateTime: time.Now().AddDate(0, 0, -30)},
		},
		addresses: []StaticAddress{
			{ID: 1, Name: "unused-ip", Region: "us-central1", Address: "34.68.1.1", Status: "RESERVED"},
		},
		snapshots: []DiskSnapshot{
			{ID: 1, Name: "old-snap", DiskSizeGB: 200, CreateTime: time.Now().AddDate(0, 0, -120), StorageLocations: []string{"us-central1"}},
		},
	}

	scanner := NewMultiProjectScanner(compute, nil, nil, []string{"proj-a", "proj-b"}, 2, ScanConfig{IdleDays: 7, StaleDays: 90})
	result, err := scanner.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("ScanAll: %v", err)
	}
	if result.ProjectsScanned != 2 {
		t.Errorf("ProjectsScanned = %d, want 2", result.ProjectsScanned)
	}
	// 4 findings per project (instance+disk+address+snapshot) × 2 projects = 8
	// Other scanners (instancegroup, cloudsql, firewall, nat, functions, lb, pubsub) produce 0 findings with empty mock data
	if len(result.Findings) != 8 {
		t.Errorf("Findings = %d, want 8", len(result.Findings))
	}
}

func TestMultiProjectScanner_PartialFailure(t *testing.T) {
	compute := &mockComputeAPI{err: errors.New("API error")}
	scanner := NewMultiProjectScanner(compute, nil, nil, []string{"proj-a"}, 1, ScanConfig{})
	result, err := scanner.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("ScanAll should not return error: %v", err)
	}
	// 8 scanner types using compute fail; CloudSQL + Functions + PubSub return empty (nil client guard)
	if len(result.Errors) != 8 {
		t.Errorf("Errors = %d, want 8", len(result.Errors))
	}
}

func TestMultiProjectScanner_DefaultConcurrency(t *testing.T) {
	compute := &mockComputeAPI{}
	scanner := NewMultiProjectScanner(compute, nil, nil, []string{"proj"}, 0, ScanConfig{})
	result, err := scanner.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("ScanAll: %v", err)
	}
	if result.ProjectsScanned != 1 {
		t.Errorf("ProjectsScanned = %d, want 1", result.ProjectsScanned)
	}
}

func TestMultiProjectScanner_Progress(t *testing.T) {
	compute := &mockComputeAPI{}
	scanner := NewMultiProjectScanner(compute, nil, nil, []string{"proj"}, 1, ScanConfig{})

	var progressCalls atomic.Int32
	scanner.SetProgressFn(func(p ScanProgress) {
		progressCalls.Add(1)
	})

	_, err := scanner.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("ScanAll: %v", err)
	}
	// 11 scanner types × 1 project = 11 progress calls
	if progressCalls.Load() != 11 {
		t.Errorf("progress calls = %d, want 11", progressCalls.Load())
	}
}

func TestMultiProjectScanner_EmptyProjects(t *testing.T) {
	compute := &mockComputeAPI{}
	scanner := NewMultiProjectScanner(compute, nil, nil, nil, 1, ScanConfig{})
	result, err := scanner.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("ScanAll: %v", err)
	}
	if result.ProjectsScanned != 0 {
		t.Errorf("ProjectsScanned = %d, want 0", result.ProjectsScanned)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0", len(result.Findings))
	}
}
