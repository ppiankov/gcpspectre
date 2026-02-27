package gcp

import (
	"context"
	"testing"
)

func TestFirewallScanner_Type(t *testing.T) {
	s := NewFirewallScanner(nil, "proj")
	if s.Type() != ResourceFirewall {
		t.Errorf("Type() = %q, want %q", s.Type(), ResourceFirewall)
	}
}

func TestFirewallScanner_UnusedRules(t *testing.T) {
	compute := &mockComputeAPI{
		instances: []ComputeInstance{
			{ID: 1, Name: "web-vm", Status: "RUNNING", Tags: []string{"http-server", "https-server"}},
		},
		firewalls: []FirewallRule{
			{ID: 1, Name: "allow-http", Network: "default", Direction: "INGRESS", TargetTags: []string{"http-server"}},
			{ID: 2, Name: "allow-ssh-old", Network: "default", Direction: "INGRESS", TargetTags: []string{"ssh-legacy"}},
			{ID: 3, Name: "allow-all", Network: "default", Direction: "INGRESS"}, // no target tags = applies to all
		},
	}

	s := NewFirewallScanner(compute, "proj")
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
	if f.ID != FindingUnusedFirewall {
		t.Errorf("ID = %q, want %q", f.ID, FindingUnusedFirewall)
	}
	if f.ResourceName != "allow-ssh-old" {
		t.Errorf("ResourceName = %q, want %q", f.ResourceName, "allow-ssh-old")
	}
	if f.Severity != SeverityLow {
		t.Errorf("Severity = %q, want %q", f.Severity, SeverityLow)
	}
}

func TestFirewallScanner_SkipDisabled(t *testing.T) {
	compute := &mockComputeAPI{
		instances: []ComputeInstance{},
		firewalls: []FirewallRule{
			{ID: 1, Name: "disabled-rule", Direction: "INGRESS", TargetTags: []string{"no-match"}, Disabled: true},
		},
	}

	s := NewFirewallScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0 (disabled rules skipped)", len(result.Findings))
	}
}

func TestFirewallScanner_ExcludeByID(t *testing.T) {
	compute := &mockComputeAPI{
		instances: []ComputeInstance{},
		firewalls: []FirewallRule{
			{ID: 1, Name: "unused", Direction: "INGRESS", TargetTags: []string{"no-match"}},
		},
	}
	cfg := ScanConfig{Exclude: ExcludeConfig{ResourceIDs: map[string]bool{"1": true}}}

	s := NewFirewallScanner(compute, "proj")
	result, err := s.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0", len(result.Findings))
	}
}

func TestFirewallScanner_AllUsed(t *testing.T) {
	compute := &mockComputeAPI{
		instances: []ComputeInstance{
			{ID: 1, Name: "vm-1", Status: "RUNNING", Tags: []string{"web", "ssh"}},
		},
		firewalls: []FirewallRule{
			{ID: 1, Name: "allow-web", Direction: "INGRESS", TargetTags: []string{"web"}},
			{ID: 2, Name: "allow-ssh", Direction: "INGRESS", TargetTags: []string{"ssh"}},
		},
	}

	s := NewFirewallScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %d, want 0 (all rules have matching tags)", len(result.Findings))
	}
}

func TestFirewallScanner_Empty(t *testing.T) {
	compute := &mockComputeAPI{}
	s := NewFirewallScanner(compute, "proj")
	result, err := s.Scan(context.Background(), ScanConfig{})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ResourcesScanned != 0 {
		t.Errorf("ResourcesScanned = %d, want 0", result.ResourcesScanned)
	}
}

func TestAnyTagUsed(t *testing.T) {
	tags := map[string]bool{"web": true, "ssh": true}
	tests := []struct {
		name       string
		targetTags []string
		want       bool
	}{
		{"match", []string{"web"}, true},
		{"no match", []string{"db"}, false},
		{"partial match", []string{"db", "ssh"}, true},
		{"empty tags", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := anyTagUsed(tt.targetTags, tags)
			if got != tt.want {
				t.Errorf("anyTagUsed = %v, want %v", got, tt.want)
			}
		})
	}
}
