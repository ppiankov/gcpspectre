package gcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ppiankov/gcpspectre/internal/pricing"
)

// FunctionsScanner detects idle Cloud Functions.
type FunctionsScanner struct {
	functions  CloudFunctionsAPI
	monitoring MonitoringAPI
	project    string
}

// NewFunctionsScanner creates a scanner for Cloud Functions.
func NewFunctionsScanner(functions CloudFunctionsAPI, monitoring MonitoringAPI, project string) *FunctionsScanner {
	return &FunctionsScanner{functions: functions, monitoring: monitoring, project: project}
}

// Type returns the resource type.
func (s *FunctionsScanner) Type() ResourceType {
	return ResourceCloudFunction
}

// Scan examines all Cloud Functions in the project for idle resources.
func (s *FunctionsScanner) Scan(ctx context.Context, cfg ScanConfig) (*ScanResult, error) {
	if s.functions == nil {
		return &ScanResult{}, nil
	}

	fns, err := s.functions.ListFunctions(ctx, s.project)
	if err != nil {
		return nil, fmt.Errorf("list functions: %w", err)
	}

	result := &ScanResult{ResourcesScanned: len(fns)}
	if len(fns) == 0 {
		return result, nil
	}

	var activeNames []string
	activeMap := make(map[string]CloudFunction)
	for _, fn := range fns {
		if cfg.Exclude.ResourceIDs != nil && cfg.Exclude.ResourceIDs[fn.Name] {
			continue
		}
		if shouldExcludeLabels(fn.Labels, cfg.Exclude.Labels) {
			continue
		}
		if fn.State == "ACTIVE" {
			activeNames = append(activeNames, fn.Name)
			activeMap[fn.Name] = fn
		}
	}

	if len(activeNames) == 0 || s.monitoring == nil {
		return result, nil
	}

	execMap, err := s.monitoring.FetchMetricMean(ctx, s.project,
		"cloudfunctions.googleapis.com/function/execution_count",
		"function_name", activeNames, cfg.IdleDays)
	if err != nil {
		slog.Warn("Failed to fetch function execution metrics", "project", s.project, "error", err)
		return result, nil
	}

	for _, name := range activeNames {
		execCount, ok := execMap[name]
		if !ok || execCount == 0 {
			fn := activeMap[name]
			cost := pricing.MonthlyFunctionCost(fn.Region)
			result.Findings = append(result.Findings, Finding{
				ID:                    FindingFunctionIdle,
				Severity:              SeverityMedium,
				ResourceType:          ResourceCloudFunction,
				ResourceID:            fn.Name,
				ResourceName:          fn.Name,
				Project:               s.project,
				Zone:                  fn.Region,
				Message:               fmt.Sprintf("Cloud Function %s has 0 executions over %d days", fn.Name, cfg.IdleDays),
				EstimatedMonthlyWaste: cost,
				Metadata: map[string]any{
					"runtime": fn.Runtime,
					"state":   fn.State,
				},
			})
		}
	}

	return result, nil
}
