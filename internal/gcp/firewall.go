package gcp

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
)

// FirewallScanner detects unused firewall rules.
type FirewallScanner struct {
	compute ComputeAPI
	project string
}

// NewFirewallScanner creates a scanner for firewall rules.
func NewFirewallScanner(compute ComputeAPI, project string) *FirewallScanner {
	return &FirewallScanner{compute: compute, project: project}
}

// Type returns the resource type.
func (s *FirewallScanner) Type() ResourceType {
	return ResourceFirewall
}

// Scan examines all firewall rules for unused entries.
// A rule is considered unused if it targets specific network tags
// but no running instances in the project have any of those tags.
func (s *FirewallScanner) Scan(ctx context.Context, cfg ScanConfig) (*ScanResult, error) {
	rules, err := s.compute.ListFirewalls(ctx, s.project)
	if err != nil {
		return nil, fmt.Errorf("list firewalls: %w", err)
	}

	result := &ScanResult{ResourcesScanned: len(rules)}
	if len(rules) == 0 {
		return result, nil
	}

	// Collect all network tags from running instances
	usedTags, err := s.collectInstanceTags(ctx)
	if err != nil {
		slog.Warn("Failed to list instances for tag check", "project", s.project, "error", err)
		return result, nil
	}

	for _, rule := range rules {
		id := strconv.FormatUint(rule.ID, 10)
		if cfg.Exclude.ResourceIDs != nil && cfg.Exclude.ResourceIDs[id] {
			continue
		}

		if rule.Disabled {
			continue
		}

		// Rules without target tags apply to all instances — not unused
		if len(rule.TargetTags) == 0 {
			continue
		}

		if !anyTagUsed(rule.TargetTags, usedTags) {
			result.Findings = append(result.Findings, Finding{
				ID:           FindingUnusedFirewall,
				Severity:     SeverityLow,
				ResourceType: ResourceFirewall,
				ResourceID:   id,
				ResourceName: rule.Name,
				Project:      s.project,
				Message:      fmt.Sprintf("Firewall rule targets tags %v but no instances use them", rule.TargetTags),
				Metadata: map[string]any{
					"network":     rule.Network,
					"direction":   rule.Direction,
					"target_tags": rule.TargetTags,
				},
			})
		}
	}

	return result, nil
}

func (s *FirewallScanner) collectInstanceTags(ctx context.Context) (map[string]bool, error) {
	instances, err := s.compute.ListInstances(ctx, s.project)
	if err != nil {
		return nil, err
	}
	tags := make(map[string]bool)
	for _, inst := range instances {
		if inst.Status != "RUNNING" {
			continue
		}
		for _, tag := range inst.Tags {
			tags[tag] = true
		}
	}
	return tags, nil
}

func anyTagUsed(targetTags []string, usedTags map[string]bool) bool {
	for _, tag := range targetTags {
		if usedTags[tag] {
			return true
		}
	}
	return false
}
