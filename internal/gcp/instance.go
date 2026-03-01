package gcp

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/ppiankov/gcpspectre/internal/pricing"
)

const (
	stoppedThresholdDays = 30
	idleCPUThreshold     = 5.0
)

// InstanceScanner detects idle and stopped compute instances.
type InstanceScanner struct {
	compute    ComputeAPI
	monitoring MonitoringAPI
	project    string
}

// NewInstanceScanner creates a scanner for compute instances.
func NewInstanceScanner(compute ComputeAPI, monitoring MonitoringAPI, project string) *InstanceScanner {
	return &InstanceScanner{compute: compute, monitoring: monitoring, project: project}
}

// Type returns the resource type.
func (s *InstanceScanner) Type() ResourceType {
	return ResourceInstance
}

// Scan examines all compute instances in the project for waste.
func (s *InstanceScanner) Scan(ctx context.Context, cfg ScanConfig) (*ScanResult, error) {
	instances, err := s.compute.ListInstances(ctx, s.project)
	if err != nil {
		return nil, fmt.Errorf("list instances: %w", err)
	}

	result := &ScanResult{ResourcesScanned: len(instances)}
	if len(instances) == 0 {
		return result, nil
	}

	now := time.Now().UTC()
	var runningIDs []string
	runningMap := make(map[string]ComputeInstance)

	for _, inst := range instances {
		id := strconv.FormatUint(inst.ID, 10)
		if cfg.Exclude.ResourceIDs != nil && cfg.Exclude.ResourceIDs[id] {
			continue
		}
		if shouldExcludeLabels(inst.Labels, cfg.Exclude.Labels) {
			continue
		}

		if inst.Status == "STOPPED" {
			stoppedAt := inst.LastStarted
			if stoppedAt.IsZero() {
				stoppedAt = inst.CreateTime
			}
			if stoppedAt.IsZero() {
				continue
			}
			daysStopped := int(now.Sub(stoppedAt).Hours() / 24)
			if daysStopped >= stoppedThresholdDays {
				cost := pricing.MonthlyInstanceCost(inst.MachineType, inst.Zone)
				result.Findings = append(result.Findings, Finding{
					ID:                    FindingStoppedInstance,
					Severity:              SeverityHigh,
					ResourceType:          ResourceInstance,
					ResourceID:            id,
					ResourceName:          inst.Name,
					Project:               s.project,
					Zone:                  inst.Zone,
					Message:               fmt.Sprintf("Stopped for %d days", daysStopped),
					EstimatedMonthlyWaste: cost,
					Metadata: map[string]any{
						"machine_type": inst.MachineType,
						"days_stopped": daysStopped,
						"state":        "stopped",
					},
				})
			}
			continue
		}

		if inst.Status == "RUNNING" {
			runningIDs = append(runningIDs, id)
			runningMap[id] = inst
		}
	}

	if len(runningIDs) > 0 && s.monitoring != nil {
		cpuMap, err := s.monitoring.FetchMetricMean(ctx, s.project,
			"compute.googleapis.com/instance/cpu/utilization",
			"instance_id", runningIDs, cfg.IdleDays)
		if err != nil {
			slog.Warn("Failed to fetch CPU metrics", "project", s.project, "error", err)
		} else {
			for _, id := range runningIDs {
				avgCPU, ok := cpuMap[id]
				if !ok {
					continue
				}
				// GCP reports CPU utilization as a fraction (0.0–1.0), convert to percentage
				cpuPercent := avgCPU * 100
				if cpuPercent < idleCPUThreshold {
					inst := runningMap[id]
					cost := pricing.MonthlyInstanceCost(inst.MachineType, inst.Zone)
					result.Findings = append(result.Findings, Finding{
						ID:                    FindingIdleInstance,
						Severity:              SeverityHigh,
						ResourceType:          ResourceInstance,
						ResourceID:            id,
						ResourceName:          inst.Name,
						Project:               s.project,
						Zone:                  inst.Zone,
						Message:               fmt.Sprintf("CPU %.1f%% over %d days", cpuPercent, cfg.IdleDays),
						EstimatedMonthlyWaste: cost,
						Metadata: map[string]any{
							"machine_type":    inst.MachineType,
							"avg_cpu_percent": cpuPercent,
							"state":           "running",
						},
					})
				}
			}
		}
	}

	return result, nil
}
