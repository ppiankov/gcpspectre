package gcp

import (
	"context"
	"fmt"
	"strconv"
)

// InstanceGroupScanner detects empty and unhealthy instance groups.
type InstanceGroupScanner struct {
	compute ComputeAPI
	project string
}

// NewInstanceGroupScanner creates a scanner for instance groups.
func NewInstanceGroupScanner(compute ComputeAPI, project string) *InstanceGroupScanner {
	return &InstanceGroupScanner{compute: compute, project: project}
}

// Type returns the resource type.
func (s *InstanceGroupScanner) Type() ResourceType {
	return ResourceInstanceGroup
}

// Scan examines all instance groups in the project for empty or unhealthy groups.
func (s *InstanceGroupScanner) Scan(ctx context.Context, cfg ScanConfig) (*ScanResult, error) {
	groups, err := s.compute.ListInstanceGroups(ctx, s.project)
	if err != nil {
		return nil, fmt.Errorf("list instance groups: %w", err)
	}

	result := &ScanResult{ResourcesScanned: len(groups)}

	for _, g := range groups {
		id := strconv.FormatUint(g.ID, 10)
		if cfg.Exclude.ResourceIDs != nil && cfg.Exclude.ResourceIDs[id] {
			continue
		}

		if g.Size == 0 {
			result.Findings = append(result.Findings, Finding{
				ID:           FindingEmptyInstanceGroup,
				Severity:     SeverityMedium,
				ResourceType: ResourceInstanceGroup,
				ResourceID:   id,
				ResourceName: g.Name,
				Project:      s.project,
				Zone:         g.Zone,
				Message:      "Instance group has 0 instances",
				Metadata: map[string]any{
					"is_managed": g.IsManaged,
				},
			})
		}
	}

	return result, nil
}
