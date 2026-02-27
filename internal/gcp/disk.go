package gcp

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ppiankov/gcpspectre/internal/pricing"
)

const detachedThresholdDays = 7

// DiskScanner detects detached persistent disks.
type DiskScanner struct {
	compute ComputeAPI
	project string
}

// NewDiskScanner creates a scanner for persistent disks.
func NewDiskScanner(compute ComputeAPI, project string) *DiskScanner {
	return &DiskScanner{compute: compute, project: project}
}

// Type returns the resource type.
func (s *DiskScanner) Type() ResourceType {
	return ResourceDisk
}

// Scan examines all persistent disks in the project for detached volumes.
func (s *DiskScanner) Scan(ctx context.Context, cfg ScanConfig) (*ScanResult, error) {
	disks, err := s.compute.ListDisks(ctx, s.project)
	if err != nil {
		return nil, fmt.Errorf("list disks: %w", err)
	}

	result := &ScanResult{ResourcesScanned: len(disks)}
	now := time.Now().UTC()

	for _, disk := range disks {
		id := strconv.FormatUint(disk.ID, 10)
		if cfg.Exclude.ResourceIDs != nil && cfg.Exclude.ResourceIDs[id] {
			continue
		}
		if shouldExcludeLabels(disk.Labels, cfg.Exclude.Labels) {
			continue
		}

		if len(disk.Users) > 0 {
			continue
		}

		detachedSince := disk.LastAttach
		if detachedSince.IsZero() {
			detachedSince = disk.CreateTime
		}
		if detachedSince.IsZero() {
			continue
		}

		daysDetached := int(now.Sub(detachedSince).Hours() / 24)
		if daysDetached < detachedThresholdDays {
			continue
		}

		cost := pricing.MonthlyDiskCost(disk.DiskType, int(disk.SizeGB), disk.Zone)
		result.Findings = append(result.Findings, Finding{
			ID:                    FindingDetachedDisk,
			Severity:              SeverityHigh,
			ResourceType:          ResourceDisk,
			ResourceID:            id,
			ResourceName:          disk.Name,
			Project:               s.project,
			Zone:                  disk.Zone,
			Message:               fmt.Sprintf("Detached %d days, %s %d GiB", daysDetached, disk.DiskType, disk.SizeGB),
			EstimatedMonthlyWaste: cost,
			Metadata: map[string]any{
				"disk_type":     disk.DiskType,
				"size_gib":      disk.SizeGB,
				"days_detached": daysDetached,
			},
		})
	}

	return result, nil
}
