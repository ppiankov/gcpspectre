package gcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ppiankov/gcpspectre/internal/pricing"
)

const idleCloudSQLCPUThreshold = 5.0

// CloudSQLScanner detects idle Cloud SQL instances.
type CloudSQLScanner struct {
	cloudSQL   CloudSQLAPI
	monitoring MonitoringAPI
	project    string
}

// NewCloudSQLScanner creates a scanner for Cloud SQL instances.
func NewCloudSQLScanner(cloudSQL CloudSQLAPI, monitoring MonitoringAPI, project string) *CloudSQLScanner {
	return &CloudSQLScanner{cloudSQL: cloudSQL, monitoring: monitoring, project: project}
}

// Type returns the resource type.
func (s *CloudSQLScanner) Type() ResourceType {
	return ResourceCloudSQL
}

// Scan examines all Cloud SQL instances in the project for idle resources.
func (s *CloudSQLScanner) Scan(ctx context.Context, cfg ScanConfig) (*ScanResult, error) {
	if s.cloudSQL == nil {
		return &ScanResult{}, nil
	}

	instances, err := s.cloudSQL.ListInstances(ctx, s.project)
	if err != nil {
		return nil, fmt.Errorf("list Cloud SQL instances: %w", err)
	}

	result := &ScanResult{ResourcesScanned: len(instances)}
	if len(instances) == 0 {
		return result, nil
	}

	var runnableNames []string
	runnableMap := make(map[string]CloudSQLInstance)
	for _, inst := range instances {
		if cfg.Exclude.ResourceIDs != nil && cfg.Exclude.ResourceIDs[inst.Name] {
			continue
		}
		if shouldExcludeLabels(inst.Labels, cfg.Exclude.Labels) {
			continue
		}
		if inst.State == "RUNNABLE" {
			runnableNames = append(runnableNames, inst.Name)
			runnableMap[inst.Name] = inst
		}
	}

	if len(runnableNames) == 0 || s.monitoring == nil {
		return result, nil
	}

	cpuMap, err := s.monitoring.FetchMetricMean(ctx, s.project,
		"cloudsql.googleapis.com/database/cpu/utilization",
		"database_id", runnableNames, cfg.IdleDays)
	if err != nil {
		slog.Warn("Failed to fetch Cloud SQL CPU metrics", "project", s.project, "error", err)
		return result, nil
	}

	for _, name := range runnableNames {
		avgCPU, ok := cpuMap[name]
		if !ok {
			continue
		}
		cpuPercent := avgCPU * 100
		if cpuPercent < idleCloudSQLCPUThreshold {
			inst := runnableMap[name]
			cost := pricing.MonthlyCloudSQLCost(inst.Tier, inst.Region)
			result.Findings = append(result.Findings, Finding{
				ID:                    FindingIdleCloudSQL,
				Severity:              SeverityHigh,
				ResourceType:          ResourceCloudSQL,
				ResourceID:            inst.Name,
				ResourceName:          inst.Name,
				Project:               s.project,
				Zone:                  inst.Region,
				Message:               fmt.Sprintf("CPU %.1f%% over %d days", cpuPercent, cfg.IdleDays),
				EstimatedMonthlyWaste: cost,
				Metadata: map[string]any{
					"tier":             inst.Tier,
					"database_version": inst.DatabaseVersion,
					"avg_cpu_percent":  cpuPercent,
				},
			})
		}
	}

	return result, nil
}
