package gcp

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/ppiankov/gcpspectre/internal/pricing"
)

// LBScanner detects idle, unhealthy, or backend-less load balancers.
type LBScanner struct {
	compute    ComputeAPI
	monitoring MonitoringAPI
	project    string
}

// NewLBScanner creates a scanner for load balancer resources.
func NewLBScanner(compute ComputeAPI, monitoring MonitoringAPI, project string) *LBScanner {
	return &LBScanner{compute: compute, monitoring: monitoring, project: project}
}

// Type returns the resource type.
func (s *LBScanner) Type() ResourceType {
	return ResourceLoadBalancer
}

// Scan examines forwarding rules and backend services for waste.
func (s *LBScanner) Scan(ctx context.Context, cfg ScanConfig) (*ScanResult, error) {
	forwardingRules, err := s.compute.ListForwardingRules(ctx, s.project)
	if err != nil {
		return nil, fmt.Errorf("list forwarding rules: %w", err)
	}

	backendServices, err := s.compute.ListBackendServices(ctx, s.project)
	if err != nil {
		return nil, fmt.Errorf("list backend services: %w", err)
	}

	result := &ScanResult{ResourcesScanned: len(forwardingRules) + len(backendServices)}

	// Build backend service lookup
	bsMap := make(map[string]BackendServiceInfo)
	for _, bs := range backendServices {
		bsMap[bs.Name] = bs
	}

	// Check for backend services with no backends
	for _, bs := range backendServices {
		id := strconv.FormatUint(bs.ID, 10)
		if cfg.Exclude.ResourceIDs != nil && cfg.Exclude.ResourceIDs[id] {
			continue
		}
		if bs.Backends == 0 {
			cost := pricing.MonthlyLBCost(s.regionFromRules(forwardingRules))
			result.Findings = append(result.Findings, Finding{
				ID:                    FindingLBNoBackends,
				Severity:              SeverityHigh,
				ResourceType:          ResourceLoadBalancer,
				ResourceID:            id,
				ResourceName:          bs.Name,
				Project:               s.project,
				Message:               fmt.Sprintf("Backend service %s has no backends configured", bs.Name),
				EstimatedMonthlyWaste: cost,
				Metadata: map[string]any{
					"protocol": bs.Protocol,
				},
			})
		}
		if !bs.HealthOK && bs.Backends > 0 {
			result.Findings = append(result.Findings, Finding{
				ID:           FindingLBUnhealthy,
				Severity:     SeverityHigh,
				ResourceType: ResourceLoadBalancer,
				ResourceID:   id,
				ResourceName: bs.Name,
				Project:      s.project,
				Message:      fmt.Sprintf("Backend service %s has unhealthy backends", bs.Name),
				Metadata: map[string]any{
					"protocol": bs.Protocol,
					"backends": bs.Backends,
				},
			})
		}
	}

	// Check forwarding rules for idle traffic
	if len(forwardingRules) > 0 && s.monitoring != nil {
		var ruleIDs []string
		ruleMap := make(map[string]ForwardingRuleInfo)
		for _, fr := range forwardingRules {
			id := strconv.FormatUint(fr.ID, 10)
			if cfg.Exclude.ResourceIDs != nil && cfg.Exclude.ResourceIDs[id] {
				continue
			}
			ruleIDs = append(ruleIDs, fr.Name)
			ruleMap[fr.Name] = fr
		}

		if len(ruleIDs) > 0 {
			reqMap, err := s.monitoring.FetchMetricMean(ctx, s.project,
				"loadbalancing.googleapis.com/https/request_count",
				"forwarding_rule_name", ruleIDs, cfg.IdleDays)
			if err != nil {
				slog.Warn("Failed to fetch LB request metrics", "project", s.project, "error", err)
			} else {
				for _, name := range ruleIDs {
					avgReqs, ok := reqMap[name]
					if !ok || avgReqs == 0 {
						fr := ruleMap[name]
						cost := pricing.MonthlyLBCost(fr.Region)
						result.Findings = append(result.Findings, Finding{
							ID:                    FindingLBIdle,
							Severity:              SeverityMedium,
							ResourceType:          ResourceLoadBalancer,
							ResourceID:            strconv.FormatUint(fr.ID, 10),
							ResourceName:          fr.Name,
							Project:               s.project,
							Zone:                  fr.Region,
							Message:               fmt.Sprintf("Forwarding rule %s has 0 requests over %d days", fr.Name, cfg.IdleDays),
							EstimatedMonthlyWaste: cost,
							Metadata: map[string]any{
								"ip_address":            fr.IPAddress,
								"load_balancing_scheme": fr.LoadBalancingScheme,
							},
						})
					}
				}
			}
		}
	}

	return result, nil
}

func (s *LBScanner) regionFromRules(rules []ForwardingRuleInfo) string {
	for _, r := range rules {
		if r.Region != "" {
			return r.Region
		}
	}
	return "us-central1"
}
