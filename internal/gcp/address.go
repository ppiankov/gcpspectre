package gcp

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ppiankov/gcpspectre/internal/pricing"
)

// AddressScanner detects unused static IP addresses.
type AddressScanner struct {
	compute ComputeAPI
	project string
}

// NewAddressScanner creates a scanner for static IP addresses.
func NewAddressScanner(compute ComputeAPI, project string) *AddressScanner {
	return &AddressScanner{compute: compute, project: project}
}

// Type returns the resource type.
func (s *AddressScanner) Type() ResourceType {
	return ResourceAddress
}

// Scan examines all static IP addresses in the project for unused addresses.
func (s *AddressScanner) Scan(ctx context.Context, cfg ScanConfig) (*ScanResult, error) {
	addresses, err := s.compute.ListAddresses(ctx, s.project)
	if err != nil {
		return nil, fmt.Errorf("list addresses: %w", err)
	}

	result := &ScanResult{ResourcesScanned: len(addresses)}

	for _, addr := range addresses {
		id := strconv.FormatUint(addr.ID, 10)
		if cfg.Exclude.ResourceIDs != nil && cfg.Exclude.ResourceIDs[id] {
			continue
		}

		if addr.Status != "RESERVED" {
			continue
		}

		cost := pricing.MonthlyAddressCost(addr.Region)
		result.Findings = append(result.Findings, Finding{
			ID:                    FindingUnusedAddress,
			Severity:              SeverityMedium,
			ResourceType:          ResourceAddress,
			ResourceID:            id,
			ResourceName:          addr.Name,
			Project:               s.project,
			Zone:                  addr.Region,
			Message:               fmt.Sprintf("Static IP %s not associated with any resource", addr.Address),
			EstimatedMonthlyWaste: cost,
			Metadata: map[string]any{
				"address":      addr.Address,
				"address_type": addr.AddressType,
				"region":       addr.Region,
			},
		})
	}

	return result, nil
}
