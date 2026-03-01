package gcp

import (
	"context"
	"testing"
)

func TestFunctionsScanner_Idle(t *testing.T) {
	functions := &mockCloudFunctionsAPI{
		functions: []CloudFunction{
			{Name: "idle-fn", Region: "us-central1", Runtime: "go122", State: "ACTIVE"},
		},
	}
	monitoring := &mockMonitoringAPI{
		results: map[string]float64{}, // no executions
	}

	s := NewFunctionsScanner(functions, monitoring, "test-project")
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
	if result.Findings[0].ID != FindingFunctionIdle {
		t.Errorf("FindingID = %s, want %s", result.Findings[0].ID, FindingFunctionIdle)
	}
}

func TestFunctionsScanner_Active(t *testing.T) {
	functions := &mockCloudFunctionsAPI{
		functions: []CloudFunction{
			{Name: "busy-fn", Region: "us-central1", Runtime: "go122", State: "ACTIVE"},
		},
	}
	monitoring := &mockMonitoringAPI{
		results: map[string]float64{
			"busy-fn": 1500, // has executions
		},
	}

	s := NewFunctionsScanner(functions, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings for active function, got %d", len(result.Findings))
	}
}

func TestFunctionsScanner_ExcludeByLabel(t *testing.T) {
	functions := &mockCloudFunctionsAPI{
		functions: []CloudFunction{
			{Name: "idle-fn", Region: "us-central1", Runtime: "go122", State: "ACTIVE", Labels: map[string]string{"env": "dev"}},
		},
	}
	monitoring := &mockMonitoringAPI{results: map[string]float64{}}

	s := NewFunctionsScanner(functions, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{
		IdleDays: 7,
		Exclude:  ExcludeConfig{Labels: map[string]string{"env": "dev"}},
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings after label exclude, got %d", len(result.Findings))
	}
}

func TestFunctionsScanner_ExcludeByID(t *testing.T) {
	functions := &mockCloudFunctionsAPI{
		functions: []CloudFunction{
			{Name: "idle-fn", Region: "us-central1", Runtime: "go122", State: "ACTIVE"},
		},
	}
	monitoring := &mockMonitoringAPI{results: map[string]float64{}}

	s := NewFunctionsScanner(functions, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{
		IdleDays: 7,
		Exclude:  ExcludeConfig{ResourceIDs: map[string]bool{"idle-fn": true}},
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings after ID exclude, got %d", len(result.Findings))
	}
}

func TestFunctionsScanner_NilClient(t *testing.T) {
	s := NewFunctionsScanner(nil, nil, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 0 {
		t.Errorf("ResourcesScanned = %d, want 0", result.ResourcesScanned)
	}
}

func TestFunctionsScanner_Empty(t *testing.T) {
	functions := &mockCloudFunctionsAPI{}
	monitoring := &mockMonitoringAPI{}

	s := NewFunctionsScanner(functions, monitoring, "test-project")
	result, err := s.Scan(context.Background(), ScanConfig{IdleDays: 7})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 0 {
		t.Errorf("ResourcesScanned = %d, want 0", result.ResourcesScanned)
	}
}
