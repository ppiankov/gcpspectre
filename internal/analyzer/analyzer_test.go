package analyzer

import (
	"testing"

	gcptype "github.com/ppiankov/gcpspectre/internal/gcp"
)

func TestAnalyzeFiltersByMinCost(t *testing.T) {
	result := &gcptype.ScanResult{
		Findings: []gcptype.Finding{
			{ID: gcptype.FindingIdleInstance, EstimatedMonthlyWaste: 50.0, Severity: gcptype.SeverityHigh, ResourceType: gcptype.ResourceInstance},
			{ID: gcptype.FindingUnusedAddress, EstimatedMonthlyWaste: 3.65, Severity: gcptype.SeverityMedium, ResourceType: gcptype.ResourceAddress},
			{ID: gcptype.FindingDetachedDisk, EstimatedMonthlyWaste: 10.0, Severity: gcptype.SeverityHigh, ResourceType: gcptype.ResourceDisk},
		},
		ResourcesScanned: 100,
		ProjectsScanned:  2,
	}

	analysis := Analyze(result, AnalyzerConfig{MinMonthlyCost: 5.0})

	if len(analysis.Findings) != 2 {
		t.Errorf("Findings len = %d, want 2 (filtered out $3.65 address)", len(analysis.Findings))
	}
	if analysis.Summary.TotalFindings != 2 {
		t.Errorf("TotalFindings = %d, want 2", analysis.Summary.TotalFindings)
	}
	if analysis.Summary.TotalMonthlyWaste != 60.0 {
		t.Errorf("TotalMonthlyWaste = %f, want 60.0", analysis.Summary.TotalMonthlyWaste)
	}
	if analysis.Summary.TotalResourcesScanned != 100 {
		t.Errorf("TotalResourcesScanned = %d, want 100", analysis.Summary.TotalResourcesScanned)
	}
	if analysis.Summary.ProjectsScanned != 2 {
		t.Errorf("ProjectsScanned = %d, want 2", analysis.Summary.ProjectsScanned)
	}
}

func TestAnalyzeNoFilter(t *testing.T) {
	result := &gcptype.ScanResult{
		Findings: []gcptype.Finding{
			{ID: gcptype.FindingIdleInstance, EstimatedMonthlyWaste: 1.0, Severity: gcptype.SeverityHigh, ResourceType: gcptype.ResourceInstance},
		},
		ResourcesScanned: 10,
		ProjectsScanned:  1,
	}

	analysis := Analyze(result, AnalyzerConfig{MinMonthlyCost: 0})

	if len(analysis.Findings) != 1 {
		t.Errorf("Findings len = %d, want 1", len(analysis.Findings))
	}
}

func TestAnalyzeEmptyFindings(t *testing.T) {
	result := &gcptype.ScanResult{
		Findings:         nil,
		ResourcesScanned: 50,
		ProjectsScanned:  3,
	}

	analysis := Analyze(result, AnalyzerConfig{MinMonthlyCost: 1.0})

	if len(analysis.Findings) != 0 {
		t.Errorf("Findings len = %d, want 0", len(analysis.Findings))
	}
	if analysis.Summary.TotalFindings != 0 {
		t.Errorf("TotalFindings = %d, want 0", analysis.Summary.TotalFindings)
	}
}

func TestAnalyzeBySeverity(t *testing.T) {
	result := &gcptype.ScanResult{
		Findings: []gcptype.Finding{
			{Severity: gcptype.SeverityHigh, EstimatedMonthlyWaste: 10.0, ResourceType: gcptype.ResourceInstance},
			{Severity: gcptype.SeverityHigh, EstimatedMonthlyWaste: 20.0, ResourceType: gcptype.ResourceDisk},
			{Severity: gcptype.SeverityMedium, EstimatedMonthlyWaste: 5.0, ResourceType: gcptype.ResourceAddress},
			{Severity: gcptype.SeverityLow, EstimatedMonthlyWaste: 1.0, ResourceType: gcptype.ResourceFirewall},
		},
		ProjectsScanned: 1,
	}

	analysis := Analyze(result, AnalyzerConfig{MinMonthlyCost: 0})

	if analysis.Summary.BySeverity["high"] != 2 {
		t.Errorf("BySeverity[high] = %d, want 2", analysis.Summary.BySeverity["high"])
	}
	if analysis.Summary.BySeverity["medium"] != 1 {
		t.Errorf("BySeverity[medium] = %d, want 1", analysis.Summary.BySeverity["medium"])
	}
	if analysis.Summary.ByResourceType["compute_instance"] != 1 {
		t.Errorf("ByResourceType[compute_instance] = %d, want 1", analysis.Summary.ByResourceType["compute_instance"])
	}
}

func TestAnalyzePreservesErrors(t *testing.T) {
	result := &gcptype.ScanResult{
		Errors: []string{"project-a: timeout", "project-b: permission denied"},
	}

	analysis := Analyze(result, AnalyzerConfig{})

	if len(analysis.Errors) != 2 {
		t.Errorf("Errors len = %d, want 2", len(analysis.Errors))
	}
}
