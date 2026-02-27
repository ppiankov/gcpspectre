package analyzer

import (
	gcptype "github.com/ppiankov/gcpspectre/internal/gcp"
)

// Summary holds aggregated statistics about scan findings.
type Summary struct {
	TotalResourcesScanned int            `json:"total_resources_scanned"`
	TotalFindings         int            `json:"total_findings"`
	TotalMonthlyWaste     float64        `json:"total_monthly_waste"`
	BySeverity            map[string]int `json:"by_severity"`
	ByResourceType        map[string]int `json:"by_resource_type"`
	ProjectsScanned       int            `json:"projects_scanned"`
}

// AnalysisResult holds filtered findings and computed summary.
type AnalysisResult struct {
	Findings []gcptype.Finding `json:"findings"`
	Summary  Summary           `json:"summary"`
	Errors   []string          `json:"errors,omitempty"`
}

// AnalyzerConfig controls analysis behavior.
type AnalyzerConfig struct {
	MinMonthlyCost float64
}
