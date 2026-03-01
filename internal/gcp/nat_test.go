package gcp

import (
	"context"
	"testing"
)

func TestNATScanner_Idle(t *testing.T) {
	compute := &mockComputeAPI{
		routers: []RouterInfo{
			{ID: 1, Name: "router-1", Region: "us-central1", NATs: []NATConfig{{Name: "nat-1"}}},
		},
	}
	monitoring := &mockMonitoringAPI{
		results: map[string]float64{}, // no data = idle
	}

	s := NewNATScanner(compute, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 1 {
		t.Errorf("ResourcesScanned = %d, want 1", result.ResourcesScanned)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("Findings = %d, want 1", len(result.Findings))
	}
	if result.Findings[0].ID != FindingNATIdle {
		t.Errorf("FindingID = %s, want %s", result.Findings[0].ID, FindingNATIdle)
	}
}

func TestNATScanner_LowTraffic(t *testing.T) {
	compute := &mockComputeAPI{
		routers: []RouterInfo{
			{ID: 1, Name: "router-1", Region: "us-central1", NATs: []NATConfig{{Name: "nat-1"}}},
		},
	}
	monitoring := &mockMonitoringAPI{
		results: map[string]float64{
			"router-1": 500, // below natLowTrafficThreshold (1024)
		},
	}

	s := NewNATScanner(compute, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("Findings = %d, want 1", len(result.Findings))
	}
	if result.Findings[0].ID != FindingNATLowTraffic {
		t.Errorf("FindingID = %s, want %s", result.Findings[0].ID, FindingNATLowTraffic)
	}
}

func TestNATScanner_Healthy(t *testing.T) {
	compute := &mockComputeAPI{
		routers: []RouterInfo{
			{ID: 1, Name: "router-1", Region: "us-central1", NATs: []NATConfig{{Name: "nat-1"}}},
		},
	}
	monitoring := &mockMonitoringAPI{
		results: map[string]float64{
			"router-1": 50000, // healthy traffic
		},
	}

	s := NewNATScanner(compute, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings for healthy NAT, got %d", len(result.Findings))
	}
}

func TestNATScanner_ExcludeByID(t *testing.T) {
	compute := &mockComputeAPI{
		routers: []RouterInfo{
			{ID: 1, Name: "router-1", Region: "us-central1", NATs: []NATConfig{{Name: "nat-1"}}},
		},
	}
	monitoring := &mockMonitoringAPI{results: map[string]float64{}}

	s := NewNATScanner(compute, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{
		IdleDays: 7,
		Exclude:  ExcludeConfig{ResourceIDs: map[string]bool{"1/nat-1": true}},
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings after exclude, got %d", len(result.Findings))
	}
}

func TestNATScanner_Empty(t *testing.T) {
	compute := &mockComputeAPI{}
	monitoring := &mockMonitoringAPI{}

	s := NewNATScanner(compute, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 0 {
		t.Errorf("ResourcesScanned = %d, want 0", result.ResourcesScanned)
	}
}
