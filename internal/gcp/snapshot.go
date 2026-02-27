package gcp

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ppiankov/gcpspectre/internal/pricing"
)

// SnapshotScanner detects stale disk snapshots.
type SnapshotScanner struct {
	compute ComputeAPI
	project string
}

// NewSnapshotScanner creates a scanner for disk snapshots.
func NewSnapshotScanner(compute ComputeAPI, project string) *SnapshotScanner {
	return &SnapshotScanner{compute: compute, project: project}
}

// Type returns the resource type.
func (s *SnapshotScanner) Type() ResourceType {
	return ResourceSnapshot
}

// Scan examines all disk snapshots in the project for stale entries.
func (s *SnapshotScanner) Scan(ctx context.Context, cfg ScanConfig) (*ScanResult, error) {
	snapshots, err := s.compute.ListSnapshots(ctx, s.project)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}

	result := &ScanResult{ResourcesScanned: len(snapshots)}
	now := time.Now().UTC()

	for _, snap := range snapshots {
		id := strconv.FormatUint(snap.ID, 10)
		if cfg.Exclude.ResourceIDs != nil && cfg.Exclude.ResourceIDs[id] {
			continue
		}
		if shouldExcludeLabels(snap.Labels, cfg.Exclude.Labels) {
			continue
		}

		if snap.CreateTime.IsZero() {
			continue
		}

		ageDays := int(now.Sub(snap.CreateTime).Hours() / 24)
		if ageDays < cfg.StaleDays {
			continue
		}

		sizeGB := snap.DiskSizeGB
		if snap.StorageBytes > 0 {
			sizeGB = snap.StorageBytes / (1024 * 1024 * 1024)
			if snap.StorageBytes%(1024*1024*1024) > 0 {
				sizeGB++
			}
		}

		region := "us-central1"
		if len(snap.StorageLocations) > 0 {
			region = snap.StorageLocations[0]
		}

		cost := pricing.MonthlySnapshotCost(int(sizeGB), region)
		result.Findings = append(result.Findings, Finding{
			ID:                    FindingStaleSnapshot,
			Severity:              SeverityMedium,
			ResourceType:          ResourceSnapshot,
			ResourceID:            id,
			ResourceName:          snap.Name,
			Project:               s.project,
			Message:               fmt.Sprintf("Snapshot %d days old, %d GiB", ageDays, sizeGB),
			EstimatedMonthlyWaste: cost,
			Metadata: map[string]any{
				"age_days":    ageDays,
				"size_gib":    sizeGB,
				"source_disk": snap.SourceDisk,
			},
		})
	}

	return result, nil
}
