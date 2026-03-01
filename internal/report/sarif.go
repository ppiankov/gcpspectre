package report

import (
	"encoding/json"
	"fmt"

	gcptype "github.com/ppiankov/gcpspectre/internal/gcp"
)

const sarifSchema = "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json"

type sarifReport struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name          string              `json:"name"`
	Version       string              `json:"version"`
	Rules         []sarifRule         `json:"rules"`
	Notifications []sarifNotification `json:"notifications,omitempty"`
}

type sarifNotification struct {
	ID      string       `json:"id"`
	Message sarifMessage `json:"message"`
	Level   string       `json:"level"`
}

type sarifRule struct {
	ID               string            `json:"id"`
	ShortDescription sarifMessage      `json:"shortDescription"`
	DefaultConfig    sarifDefaultLevel `json:"defaultConfiguration"`
}

type sarifDefaultLevel struct {
	Level string `json:"level"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string         `json:"ruleId"`
	Level     string         `json:"level"`
	Message   sarifMessage   `json:"message"`
	Locations []sarifLoc     `json:"locations,omitempty"`
	Props     map[string]any `json:"properties,omitempty"`
}

type sarifLoc struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}

type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
}

type sarifArtifact struct {
	URI string `json:"uri"`
}

// Generate writes SARIF v2.1.0 output.
func (r *SARIFReporter) Generate(data Data) error {
	rules := buildSARIFRules()
	results := make([]sarifResult, 0, len(data.Findings))

	for _, f := range data.Findings {
		uri := fmt.Sprintf("gcp://%s/%s/%s", f.Project, f.ResourceType, f.ResourceID)
		if f.Zone != "" {
			uri = fmt.Sprintf("gcp://%s/%s/%s/%s", f.Project, f.Zone, f.ResourceType, f.ResourceID)
		}
		results = append(results, sarifResult{
			RuleID:  string(f.ID),
			Level:   sarifLevel(f.Severity),
			Message: sarifMessage{Text: f.Message},
			Locations: []sarifLoc{
				{
					PhysicalLocation: sarifPhysical{
						ArtifactLocation: sarifArtifact{URI: uri},
					},
				},
			},
			Props: map[string]any{
				"resourceName":          f.ResourceName,
				"estimatedMonthlyWaste": f.EstimatedMonthlyWaste,
				"metadata":              f.Metadata,
			},
		})
	}

	report := sarifReport{
		Schema:  sarifSchema,
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:          data.Tool,
						Version:       data.Version,
						Rules:         rules,
						Notifications: buildSARIFNotifications(data.Errors),
					},
				},
				Results: results,
			},
		},
	}

	enc := json.NewEncoder(r.Writer)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encode SARIF report: %w", err)
	}
	return nil
}

func sarifLevel(s gcptype.Severity) string {
	switch s {
	case gcptype.SeverityHigh:
		return "error"
	case gcptype.SeverityMedium:
		return "warning"
	default:
		return "note"
	}
}

func buildSARIFRules() []sarifRule {
	return []sarifRule{
		{ID: string(gcptype.FindingIdleInstance), ShortDescription: sarifMessage{Text: "Idle compute instance"}, DefaultConfig: sarifDefaultLevel{Level: "error"}},
		{ID: string(gcptype.FindingStoppedInstance), ShortDescription: sarifMessage{Text: "Stopped compute instance"}, DefaultConfig: sarifDefaultLevel{Level: "error"}},
		{ID: string(gcptype.FindingDetachedDisk), ShortDescription: sarifMessage{Text: "Detached persistent disk"}, DefaultConfig: sarifDefaultLevel{Level: "error"}},
		{ID: string(gcptype.FindingUnusedAddress), ShortDescription: sarifMessage{Text: "Unused static IP"}, DefaultConfig: sarifDefaultLevel{Level: "warning"}},
		{ID: string(gcptype.FindingStaleSnapshot), ShortDescription: sarifMessage{Text: "Stale snapshot"}, DefaultConfig: sarifDefaultLevel{Level: "warning"}},
		{ID: string(gcptype.FindingEmptyInstanceGroup), ShortDescription: sarifMessage{Text: "Empty instance group"}, DefaultConfig: sarifDefaultLevel{Level: "error"}},
		{ID: string(gcptype.FindingUnhealthyInstanceGroup), ShortDescription: sarifMessage{Text: "Unhealthy instance group"}, DefaultConfig: sarifDefaultLevel{Level: "error"}},
		{ID: string(gcptype.FindingIdleCloudSQL), ShortDescription: sarifMessage{Text: "Idle Cloud SQL instance"}, DefaultConfig: sarifDefaultLevel{Level: "error"}},
		{ID: string(gcptype.FindingUnusedFirewall), ShortDescription: sarifMessage{Text: "Unused firewall rule"}, DefaultConfig: sarifDefaultLevel{Level: "note"}},
		{ID: string(gcptype.FindingNATIdle), ShortDescription: sarifMessage{Text: "Idle Cloud NAT gateway"}, DefaultConfig: sarifDefaultLevel{Level: "warning"}},
		{ID: string(gcptype.FindingNATLowTraffic), ShortDescription: sarifMessage{Text: "Low-traffic Cloud NAT gateway"}, DefaultConfig: sarifDefaultLevel{Level: "note"}},
		{ID: string(gcptype.FindingFunctionIdle), ShortDescription: sarifMessage{Text: "Idle Cloud Function"}, DefaultConfig: sarifDefaultLevel{Level: "warning"}},
		{ID: string(gcptype.FindingLBIdle), ShortDescription: sarifMessage{Text: "Idle load balancer"}, DefaultConfig: sarifDefaultLevel{Level: "warning"}},
		{ID: string(gcptype.FindingLBUnhealthy), ShortDescription: sarifMessage{Text: "Load balancer with unhealthy backends"}, DefaultConfig: sarifDefaultLevel{Level: "error"}},
		{ID: string(gcptype.FindingLBNoBackends), ShortDescription: sarifMessage{Text: "Load balancer with no backends"}, DefaultConfig: sarifDefaultLevel{Level: "error"}},
	}
}

func buildSARIFNotifications(errors []ScanError) []sarifNotification {
	if len(errors) == 0 {
		return nil
	}
	notifications := make([]sarifNotification, 0, len(errors))
	for i, e := range errors {
		level := "warning"
		if !e.Recoverable {
			level = "error"
		}
		notifications = append(notifications, sarifNotification{
			ID:      fmt.Sprintf("scanner-error-%d", i),
			Message: sarifMessage{Text: e.String()},
			Level:   level,
		})
	}
	return notifications
}
