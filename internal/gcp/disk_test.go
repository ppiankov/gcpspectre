package gcp

import (
	"context"
	"testing"
	"time"
)

func TestDiskScanner_Type(t *testing.T) {
	s := NewDiskScanner(nil, "proj")
	if s.Type() != ResourceDisk {
		t.Errorf("Type() = %q, want %q", s.Type(), ResourceDisk)
	}
}

func TestDiskScanner_DetachedDisks(t *testing.T) {
	compute := &mockComputeAPI{
		disks: []PersistentDisk{
			{
				ID:         1,
				Name:       "orphan-disk",
				Zone:       "us-central1-a",
				DiskType:   "pd-ssd",
				SizeGB:     100,
				Users:      nil,
				CreateTime: time.Now().AddDate(0, 0, -30),
			},
			{
				ID:         2,
				Name:       "attached-disk",
				Zone:       "us-central1-a",
				DiskType:   "pd-standard",
				SizeGB:     200,
				Users:      []string{"projects/proj/zones/us-central1-a/instances/vm-1"},
				CreateTime: time.Now().AddDate(0, 0, -30),
			},
			{
				ID:         3,
				Name:       "new-detached",
				Zone:       "us-central1-a",
				DiskType:   "pd-standard",
				SizeGB:     50,
				Users:      nil,
				CreateTime: time.Now().AddDate(0, 0, -3),
			},
		},
	}

	s := NewDiskScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 3 {
		t.Errorf("ResourcesScanned = %d, want 3", result.ResourcesScanned)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("Findings = %d, want 1", len(result.Findings))
	}

	f := result.Findings[0]
	if f.ID != FindingDetachedDisk {
		t.Errorf("ID = %q, want %q", f.ID, FindingDetachedDisk)
	}
	if f.ResourceName != "orphan-disk" {
		t.Errorf("ResourceName = %q, want %q", f.ResourceName, "orphan-disk")
	}
	if f.EstimatedMonthlyWaste == 0 {
		t.Error("expected non-zero waste")
	}
}

func TestDiskScanner_LastAttachTime(t *testing.T) {
	compute := &mockComputeAPI{
		disks: []PersistentDisk{
			{
				ID:         1,
				Name:       "old-disk",
				Zone:       "us-central1-a",
				DiskType:   "pd-ssd",
				SizeGB:     50,
				Users:      nil,
				LastAttach: time.Now().AddDate(0, 0, -20),
				CreateTime: time.Now().AddDate(0, -6, 0),
			},
		},
	}

	s := NewDiskScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatal("expected 1 finding using LastAttach time")
	}
	days := result.Findings[0].Metadata["days_detached"].(int)
	if days < 18 || days > 22 {
		t.Errorf("days_detached = %d, expected ~20", days)
	}
}

func TestDiskScanner_ExcludeByID(t *testing.T) {
	compute := &mockComputeAPI{
		disks: []PersistentDisk{
			{ID: 1, Name: "disk", DiskType: "pd-ssd", SizeGB: 100, CreateTime: time.Now().AddDate(0, 0, -30)},
		},
	}
	cfg := ScanConfig{Exclude: ExcludeConfig{ResourceIDs: map[string]bool{"1": true}}}

	s := NewDiskScanner(compute, "proj")
	result, err := s.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0", len(result.Findings))
	}
}

func TestDiskScanner_ExcludeByLabel(t *testing.T) {
	compute := &mockComputeAPI{
		disks: []PersistentDisk{
			{
				ID: 1, Name: "disk", DiskType: "pd-ssd", SizeGB: 100,
				CreateTime: time.Now().AddDate(0, 0, -30),
				Labels:     map[string]string{"team": "keep"},
			},
		},
	}
	cfg := ScanConfig{Exclude: ExcludeConfig{Labels: map[string]string{"team": "keep"}}}

	s := NewDiskScanner(compute, "proj")
	result, err := s.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0 (excluded by label)", len(result.Findings))
	}
}

func TestDiskScanner_Empty(t *testing.T) {
	compute := &mockComputeAPI{}
	s := NewDiskScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 0 {
		t.Errorf("ResourcesScanned = %d, want 0", result.ResourcesScanned)
	}
}
