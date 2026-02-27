package gcp

import (
	"context"
	"testing"
)

func TestCloudSQLScanner_Type(t *testing.T) {
	s := NewCloudSQLScanner(nil, nil, "proj")
	if s.Type() != ResourceCloudSQL {
		t.Errorf("Type() = %q, want %q", s.Type(), ResourceCloudSQL)
	}
}

func TestCloudSQLScanner_IdleInstances(t *testing.T) {
	cloudSQL := &mockCloudSQLAPI{
		instances: []CloudSQLInstance{
			{Name: "idle-db", Region: "us-central1", Tier: "db-f1-micro", State: "RUNNABLE", DatabaseVersion: "MYSQL_8_0"},
			{Name: "busy-db", Region: "us-central1", Tier: "db-n1-standard-2", State: "RUNNABLE", DatabaseVersion: "POSTGRES_15"},
		},
	}
	monitoring := &mockMonitoringAPI{
		results: map[string]float64{
			"idle-db": 0.02, // 2% CPU
			"busy-db": 0.60, // 60% CPU
		},
	}

	s := NewCloudSQLScanner(cloudSQL, monitoring, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
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
	if f.ID != FindingIdleCloudSQL {
		t.Errorf("ID = %q, want %q", f.ID, FindingIdleCloudSQL)
	}
	if f.ResourceName != "idle-db" {
		t.Errorf("ResourceName = %q, want %q", f.ResourceName, "idle-db")
	}
	if f.EstimatedMonthlyWaste == 0 {
		t.Error("expected non-zero waste")
	}
}

func TestCloudSQLScanner_NilCloudSQLAPI(t *testing.T) {
	s := NewCloudSQLScanner(nil, nil, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 0 {
		t.Errorf("ResourcesScanned = %d, want 0", result.ResourcesScanned)
	}
}

func TestCloudSQLScanner_NilMonitoring(t *testing.T) {
	cloudSQL := &mockCloudSQLAPI{
		instances: []CloudSQLInstance{
			{Name: "db-1", Region: "us-central1", Tier: "db-f1-micro", State: "RUNNABLE"},
		},
	}

	s := NewCloudSQLScanner(cloudSQL, nil, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0 (no monitoring)", len(result.Findings))
	}
}

func TestCloudSQLScanner_ExcludeByID(t *testing.T) {
	cloudSQL := &mockCloudSQLAPI{
		instances: []CloudSQLInstance{
			{Name: "idle-db", Region: "us-central1", Tier: "db-f1-micro", State: "RUNNABLE"},
		},
	}
	monitoring := &mockMonitoringAPI{
		results: map[string]float64{"idle-db": 0.01},
	}
	cfg := ScanConfig{IdleDays: 7, Exclude: ExcludeConfig{ResourceIDs: map[string]bool{"idle-db": true}}}

	s := NewCloudSQLScanner(cloudSQL, monitoring, "proj")
	result, err := s.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0", len(result.Findings))
	}
}

func TestCloudSQLScanner_SkipStopped(t *testing.T) {
	cloudSQL := &mockCloudSQLAPI{
		instances: []CloudSQLInstance{
			{Name: "stopped-db", Region: "us-central1", Tier: "db-f1-micro", State: "STOPPED"},
		},
	}
	monitoring := &mockMonitoringAPI{
		results: map[string]float64{"stopped-db": 0.0},
	}

	s := NewCloudSQLScanner(cloudSQL, monitoring, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0 (stopped instances skipped)", len(result.Findings))
	}
}

func TestCloudSQLScanner_Empty(t *testing.T) {
	cloudSQL := &mockCloudSQLAPI{}
	s := NewCloudSQLScanner(cloudSQL, nil, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 0 {
		t.Errorf("ResourcesScanned = %d, want 0", result.ResourcesScanned)
	}
}
