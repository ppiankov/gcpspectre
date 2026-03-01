package report

import (
	"testing"

	gcptype "github.com/ppiankov/gcpspectre/internal/gcp"
)

func TestComputeExitCode(t *testing.T) {
	findings := []gcptype.Finding{
		{Severity: gcptype.SeverityHigh},
		{Severity: gcptype.SeverityMedium},
		{Severity: gcptype.SeverityLow},
		{Severity: gcptype.SeverityLow},
	}

	tests := []struct {
		name      string
		failOn    string
		threshold int
		want      int
	}{
		{"empty failOn returns OK", "", 1, ExitOK},
		{"low matches all 4", "low", 1, ExitThresholdExceeded},
		{"low threshold 5 not exceeded", "low", 5, ExitOK},
		{"medium matches 2", "medium", 1, ExitThresholdExceeded},
		{"medium threshold 3 not exceeded", "medium", 3, ExitOK},
		{"high matches 1", "high", 1, ExitThresholdExceeded},
		{"high threshold 2 not exceeded", "high", 2, ExitOK},
		{"no findings always OK", "low", 1, ExitOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := findings
			if tt.name == "no findings always OK" {
				input = nil
			}
			got := ComputeExitCode(input, tt.failOn, tt.threshold)
			if got != tt.want {
				t.Errorf("ComputeExitCode = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSeverityRank(t *testing.T) {
	if severityRank(gcptype.SeverityLow) >= severityRank(gcptype.SeverityMedium) {
		t.Error("low should rank below medium")
	}
	if severityRank(gcptype.SeverityMedium) >= severityRank(gcptype.SeverityHigh) {
		t.Error("medium should rank below high")
	}
}
