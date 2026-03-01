package gcp

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/ppiankov/gcpspectre/internal/pricing"
)

const natLowTrafficThreshold = 1024 // bytes/sec — below this is low traffic

// NATScanner detects idle and low-traffic Cloud NAT gateways.
type NATScanner struct {
	compute    ComputeAPI
	monitoring MonitoringAPI
	project    string
}

// NewNATScanner creates a scanner for Cloud NAT resources.
func NewNATScanner(compute ComputeAPI, monitoring MonitoringAPI, project string) *NATScanner {
	return &NATScanner{compute: compute, monitoring: monitoring, project: project}
}

// Type returns the resource type.
func (s *NATScanner) Type() ResourceType {
	return ResourceCloudNAT
}

// Scan examines Cloud Routers for idle or low-traffic NAT configurations.
func (s *NATScanner) Scan(ctx context.Context, cfg ScanConfig) (*ScanResult, error) {
	routers, err := s.compute.ListRouters(ctx, s.project)
	if err != nil {
		return nil, fmt.Errorf("list routers: %w", err)
	}

	// Collect NAT gateways from routers
	type natEntry struct {
		routerID   uint64
		routerName string
		natName    string
		region     string
	}

	var natEntries []natEntry
	var natIDs []string
	for _, r := range routers {
		for _, n := range r.NATs {
			id := fmt.Sprintf("%d/%s", r.ID, n.Name)
			if cfg.Exclude.ResourceIDs != nil && cfg.Exclude.ResourceIDs[id] {
				continue
			}
			natEntries = append(natEntries, natEntry{
				routerID:   r.ID,
				routerName: r.Name,
				natName:    n.Name,
				region:     r.Region,
			})
			natIDs = append(natIDs, r.Name)
		}
	}

	result := &ScanResult{ResourcesScanned: len(natEntries)}
	if len(natEntries) == 0 || s.monitoring == nil {
		return result, nil
	}

	// Query NAT sent bytes metric keyed by router name
	bytesMap, err := s.monitoring.FetchMetricMean(ctx, s.project,
		"router.googleapis.com/nat/sent_bytes_count",
		"router_id", natIDs, cfg.IdleDays)
	if err != nil {
		slog.Warn("Failed to fetch NAT metrics", "project", s.project, "error", err)
		return result, nil
	}

	for _, entry := range natEntries {
		avgBytes, ok := bytesMap[entry.routerName]
		if !ok {
			// No metric data — NAT is completely idle
			cost := pricing.MonthlyNATCost(entry.region)
			result.Findings = append(result.Findings, Finding{
				ID:                    FindingNATIdle,
				Severity:              SeverityMedium,
				ResourceType:          ResourceCloudNAT,
				ResourceID:            strconv.FormatUint(entry.routerID, 10),
				ResourceName:          fmt.Sprintf("%s/%s", entry.routerName, entry.natName),
				Project:               s.project,
				Zone:                  entry.region,
				Message:               fmt.Sprintf("Cloud NAT %s on router %s has no traffic over %d days", entry.natName, entry.routerName, cfg.IdleDays),
				EstimatedMonthlyWaste: cost,
				Metadata: map[string]any{
					"router":   entry.routerName,
					"nat_name": entry.natName,
				},
			})
			continue
		}

		if avgBytes < natLowTrafficThreshold {
			cost := pricing.MonthlyNATCost(entry.region)
			result.Findings = append(result.Findings, Finding{
				ID:                    FindingNATLowTraffic,
				Severity:              SeverityLow,
				ResourceType:          ResourceCloudNAT,
				ResourceID:            strconv.FormatUint(entry.routerID, 10),
				ResourceName:          fmt.Sprintf("%s/%s", entry.routerName, entry.natName),
				Project:               s.project,
				Zone:                  entry.region,
				Message:               fmt.Sprintf("Cloud NAT %s avg %.0f bytes/sec over %d days", entry.natName, avgBytes, cfg.IdleDays),
				EstimatedMonthlyWaste: cost,
				Metadata: map[string]any{
					"router":        entry.routerName,
					"nat_name":      entry.natName,
					"avg_bytes_sec": avgBytes,
				},
			})
		}
	}

	return result, nil
}
