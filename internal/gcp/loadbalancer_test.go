package gcp

import (
	"context"
	"testing"
)

func TestLBScanner_NoBackends(t *testing.T) {
	compute := &mockComputeAPI{
		forwardingRules: []ForwardingRuleInfo{
			{ID: 1, Name: "fr-1", Region: "us-central1", Target: "bs-empty", IPAddress: "1.2.3.4", LoadBalancingScheme: "EXTERNAL"},
		},
		backendServices: []BackendServiceInfo{
			{ID: 10, Name: "bs-empty", Backends: 0, Protocol: "HTTP", HealthOK: true},
		},
	}
	monitoring := &mockMonitoringAPI{results: map[string]float64{}}

	s := NewLBScanner(compute, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	hasNoBackends := false
	for _, f := range result.Findings {
		if f.ID == FindingLBNoBackends {
			hasNoBackends = true
		}
	}
	if !hasNoBackends {
		t.Error("expected LB_NO_BACKENDS finding")
	}
}

func TestLBScanner_Unhealthy(t *testing.T) {
	compute := &mockComputeAPI{
		backendServices: []BackendServiceInfo{
			{ID: 10, Name: "bs-sick", Backends: 2, Protocol: "HTTP", HealthOK: false},
		},
	}
	monitoring := &mockMonitoringAPI{results: map[string]float64{}}

	s := NewLBScanner(compute, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	hasUnhealthy := false
	for _, f := range result.Findings {
		if f.ID == FindingLBUnhealthy {
			hasUnhealthy = true
		}
	}
	if !hasUnhealthy {
		t.Error("expected LB_UNHEALTHY finding")
	}
}

func TestLBScanner_IdleForwardingRule(t *testing.T) {
	compute := &mockComputeAPI{
		forwardingRules: []ForwardingRuleInfo{
			{ID: 1, Name: "idle-fr", Region: "us-central1", IPAddress: "1.2.3.4", LoadBalancingScheme: "EXTERNAL"},
		},
		backendServices: []BackendServiceInfo{
			{ID: 10, Name: "bs-ok", Backends: 2, Protocol: "HTTP", HealthOK: true},
		},
	}
	monitoring := &mockMonitoringAPI{
		results: map[string]float64{}, // no traffic
	}

	s := NewLBScanner(compute, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	hasIdle := false
	for _, f := range result.Findings {
		if f.ID == FindingLBIdle {
			hasIdle = true
		}
	}
	if !hasIdle {
		t.Error("expected LB_IDLE finding")
	}
}

func TestLBScanner_Healthy(t *testing.T) {
	compute := &mockComputeAPI{
		forwardingRules: []ForwardingRuleInfo{
			{ID: 1, Name: "busy-fr", Region: "us-central1", IPAddress: "1.2.3.4", LoadBalancingScheme: "EXTERNAL"},
		},
		backendServices: []BackendServiceInfo{
			{ID: 10, Name: "bs-ok", Backends: 2, Protocol: "HTTP", HealthOK: true},
		},
	}
	monitoring := &mockMonitoringAPI{
		results: map[string]float64{
			"busy-fr": 5000, // active traffic
		},
	}

	s := NewLBScanner(compute, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings for healthy LB, got %d", len(result.Findings))
	}
}

func TestLBScanner_ExcludeByID(t *testing.T) {
	compute := &mockComputeAPI{
		backendServices: []BackendServiceInfo{
			{ID: 10, Name: "bs-empty", Backends: 0, Protocol: "HTTP"},
		},
	}
	monitoring := &mockMonitoringAPI{results: map[string]float64{}}

	s := NewLBScanner(compute, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{
		IdleDays: 7,
		Exclude:  ExcludeConfig{ResourceIDs: map[string]bool{"10": true}},
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings after exclude, got %d", len(result.Findings))
	}
}

func TestLBScanner_Empty(t *testing.T) {
	compute := &mockComputeAPI{}
	monitoring := &mockMonitoringAPI{}

	s := NewLBScanner(compute, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 0 {
		t.Errorf("ResourcesScanned = %d, want 0", result.ResourcesScanned)
	}
}
