package gcp

import (
	"context"
	"testing"
	"time"
)

func TestSnapshotScanner_Type(t *testing.T) {
	s := NewSnapshotScanner(nil, "proj")
	if s.Type() != ResourceSnapshot {
		t.Errorf("Type() = %q, want %q", s.Type(), ResourceSnapshot)
	}
}

func TestSnapshotScanner_StaleSnapshots(t *testing.T) {
	compute := &mockComputeAPI{
		snapshots: []DiskSnapshot{
			{
				ID:               1,
				Name:             "old-snap",
				DiskSizeGB:       200,
				StorageBytes:     200 * 1024 * 1024 * 1024,
				CreateTime:       time.Now().AddDate(0, 0, -120),
				StorageLocations: []string{"us-central1"},
			},
			{
				ID:               2,
				Name:             "recent-snap",
				DiskSizeGB:       100,
				CreateTime:       time.Now().AddDate(0, 0, -10),
				StorageLocations: []string{"us-central1"},
			},
		},
	}

	s := NewSnapshotScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{StaleDays: 90})
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
	if f.ID != FindingStaleSnapshot {
		t.Errorf("ID = %q, want %q", f.ID, FindingStaleSnapshot)
	}
	if f.ResourceName != "old-snap" {
		t.Errorf("ResourceName = %q, want %q", f.ResourceName, "old-snap")
	}
	if f.EstimatedMonthlyWaste == 0 {
		t.Error("expected non-zero waste")
	}
}

func TestSnapshotScanner_StorageBytesRoundUp(t *testing.T) {
	compute := &mockComputeAPI{
		snapshots: []DiskSnapshot{
			{
				ID:               1,
				Name:             "partial-snap",
				DiskSizeGB:       100,
				StorageBytes:     50*1024*1024*1024 + 1, // 50 GiB + 1 byte → rounds up to 51
				CreateTime:       time.Now().AddDate(0, 0, -100),
				StorageLocations: []string{"us-central1"},
			},
		},
	}

	s := NewSnapshotScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{StaleDays: 90})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatal("expected 1 finding")
	}
	sizeGiB := result.Findings[0].Metadata["size_gib"].(int64)
	if sizeGiB != 51 {
		t.Errorf("size_gib = %d, want 51 (rounded up)", sizeGiB)
	}
}

func TestSnapshotScanner_FallbackDiskSize(t *testing.T) {
	compute := &mockComputeAPI{
		snapshots: []DiskSnapshot{
			{
				ID:               1,
				Name:             "no-storage-bytes",
				DiskSizeGB:       100,
				StorageBytes:     0, // no StorageBytes → use DiskSizeGB
				CreateTime:       time.Now().AddDate(0, 0, -100),
				StorageLocations: []string{"us-central1"},
			},
		},
	}

	s := NewSnapshotScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{StaleDays: 90})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatal("expected 1 finding")
	}
	sizeGiB := result.Findings[0].Metadata["size_gib"].(int64)
	if sizeGiB != 100 {
		t.Errorf("size_gib = %d, want 100 (fallback to DiskSizeGB)", sizeGiB)
	}
}

func TestSnapshotScanner_DefaultRegion(t *testing.T) {
	compute := &mockComputeAPI{
		snapshots: []DiskSnapshot{
			{
				ID:         1,
				Name:       "no-location",
				DiskSizeGB: 50,
				CreateTime: time.Now().AddDate(0, 0, -100),
				// no StorageLocations → default to us-central1
			},
		},
	}

	s := NewSnapshotScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{StaleDays: 90})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatal("expected 1 finding")
	}
	if result.Findings[0].EstimatedMonthlyWaste == 0 {
		t.Error("expected non-zero waste with default region")
	}
}

func TestSnapshotScanner_ExcludeByID(t *testing.T) {
	compute := &mockComputeAPI{
		snapshots: []DiskSnapshot{
			{ID: 1, Name: "snap", DiskSizeGB: 100, CreateTime: time.Now().AddDate(0, 0, -100), StorageLocations: []string{"us"}},
		},
	}
	cfg := ScanConfig{StaleDays: 90, Exclude: ExcludeConfig{ResourceIDs: map[string]bool{"1": true}}}

	s := NewSnapshotScanner(compute, "proj")
	result, err := s.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0", len(result.Findings))
	}
}

func TestSnapshotScanner_ExcludeByLabel(t *testing.T) {
	compute := &mockComputeAPI{
		snapshots: []DiskSnapshot{
			{
				ID: 1, Name: "snap", DiskSizeGB: 100,
				CreateTime:       time.Now().AddDate(0, 0, -100),
				StorageLocations: []string{"us-central1"},
				Labels:           map[string]string{"backup": "daily"},
			},
		},
	}
	cfg := ScanConfig{StaleDays: 90, Exclude: ExcludeConfig{Labels: map[string]string{"backup": "daily"}}}

	s := NewSnapshotScanner(compute, "proj")
	result, err := s.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0 (excluded by label)", len(result.Findings))
	}
}

func TestSnapshotScanner_Empty(t *testing.T) {
	compute := &mockComputeAPI{}
	s := NewSnapshotScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{StaleDays: 90})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 0 {
		t.Errorf("ResourcesScanned = %d, want 0", result.ResourcesScanned)
	}
}
