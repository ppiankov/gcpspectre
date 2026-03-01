package commands

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPrintDryRunText(t *testing.T) {
	plan := DryRunPlan{
		Projects:       []string{"my-project"},
		Scanners:       scannerNames,
		IdleDays:       7,
		StaleDays:      90,
		MinMonthlyCost: 1.0,
		ConfigPath:     "none",
	}

	var buf strings.Builder
	if err := printDryRunText(&buf, plan); err != nil {
		t.Fatalf("printDryRunText: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "my-project") {
		t.Error("missing project name")
	}
	if !strings.Contains(output, "compute_instance") {
		t.Error("missing scanner name")
	}
	if !strings.Contains(output, "dry-run") {
		t.Error("missing dry-run header")
	}
}

func TestPrintDryRunJSON(t *testing.T) {
	plan := DryRunPlan{
		Projects:       []string{"proj-a", "proj-b"},
		Scanners:       scannerNames,
		IdleDays:       14,
		StaleDays:      180,
		MinMonthlyCost: 5.0,
		Exclusions: DryRunExclusions{
			Labels: []string{"env=prod"},
		},
		ConfigPath: "none",
	}

	var buf strings.Builder
	if err := printDryRunJSON(&buf, plan); err != nil {
		t.Fatalf("printDryRunJSON: %v", err)
	}

	var parsed DryRunPlan
	if err := json.Unmarshal([]byte(buf.String()), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(parsed.Projects) != 2 {
		t.Errorf("projects = %d, want 2", len(parsed.Projects))
	}
	if parsed.IdleDays != 14 {
		t.Errorf("idle_days = %d, want 14", parsed.IdleDays)
	}
}

func TestPrintDryRunTextWithExclusions(t *testing.T) {
	plan := DryRunPlan{
		Projects: []string{"proj"},
		Scanners: scannerNames,
		Exclusions: DryRunExclusions{
			ResourceIDs: []string{"123"},
			Labels:      []string{"env=prod"},
		},
		ConfigPath: "none",
	}

	var buf strings.Builder
	if err := printDryRunText(&buf, plan); err != nil {
		t.Fatalf("printDryRunText: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "resource-id: 123") {
		t.Error("missing resource ID exclusion")
	}
	if !strings.Contains(output, "label: env=prod") {
		t.Error("missing label exclusion")
	}
}

func TestMergeSlices(t *testing.T) {
	if got := mergeSlices(nil, nil); got != nil {
		t.Errorf("nil+nil: got %v, want nil", got)
	}

	got := mergeSlices([]string{"a", "b"}, []string{"b", "c"})
	if len(got) != 3 {
		t.Errorf("len = %d, want 3", len(got))
	}
}
