package gcp

import (
	"context"
	"testing"
)

func TestAddressScanner_Type(t *testing.T) {
	s := NewAddressScanner(nil, "proj")
	if s.Type() != ResourceAddress {
		t.Errorf("Type() = %q, want %q", s.Type(), ResourceAddress)
	}
}

func TestAddressScanner_UnusedAddresses(t *testing.T) {
	compute := &mockComputeAPI{
		addresses: []StaticAddress{
			{
				ID:          1,
				Name:        "unused-ip",
				Region:      "us-central1",
				Address:     "34.68.1.1",
				Status:      "RESERVED",
				AddressType: "EXTERNAL",
			},
			{
				ID:          2,
				Name:        "in-use-ip",
				Region:      "us-central1",
				Address:     "34.68.1.2",
				Status:      "IN_USE",
				Users:       []string{"projects/proj/zones/us-central1-a/instances/vm-1"},
				AddressType: "EXTERNAL",
			},
		},
	}

	s := NewAddressScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{})
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
	if f.ID != FindingUnusedAddress {
		t.Errorf("ID = %q, want %q", f.ID, FindingUnusedAddress)
	}
	if f.ResourceName != "unused-ip" {
		t.Errorf("ResourceName = %q, want %q", f.ResourceName, "unused-ip")
	}
	if f.EstimatedMonthlyWaste == 0 {
		t.Error("expected non-zero waste")
	}
}

func TestAddressScanner_ExcludeByID(t *testing.T) {
	compute := &mockComputeAPI{
		addresses: []StaticAddress{
			{ID: 1, Name: "unused", Region: "us-central1", Address: "34.68.1.1", Status: "RESERVED"},
		},
	}
	cfg := ScanConfig{Exclude: ExcludeConfig{ResourceIDs: map[string]bool{"1": true}}}

	s := NewAddressScanner(compute, "proj")
	result, err := s.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0", len(result.Findings))
	}
}

func TestAddressScanner_Empty(t *testing.T) {
	compute := &mockComputeAPI{}
	s := NewAddressScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 0 {
		t.Errorf("ResourcesScanned = %d, want 0", result.ResourcesScanned)
	}
}

func TestAddressScanner_AllInUse(t *testing.T) {
	compute := &mockComputeAPI{
		addresses: []StaticAddress{
			{ID: 1, Name: "ip-1", Region: "us-central1", Address: "34.1.1.1", Status: "IN_USE"},
			{ID: 2, Name: "ip-2", Region: "us-central1", Address: "34.1.1.2", Status: "IN_USE"},
		},
	}

	s := NewAddressScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0 (all in use)", len(result.Findings))
	}
}
