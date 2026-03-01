package gcp

import "time"

// Severity levels for findings.
type Severity string

const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
)

// ResourceType identifies the GCP resource being audited.
type ResourceType string

const (
	ResourceInstance      ResourceType = "compute_instance"
	ResourceDisk          ResourceType = "persistent_disk"
	ResourceAddress       ResourceType = "static_ip"
	ResourceSnapshot      ResourceType = "snapshot"
	ResourceInstanceGroup ResourceType = "instance_group"
	ResourceCloudSQL      ResourceType = "cloud_sql"
	ResourceFirewall      ResourceType = "firewall_rule"
	ResourceCloudNAT      ResourceType = "cloud_nat"
	ResourceCloudFunction ResourceType = "cloud_function"
	ResourceLoadBalancer  ResourceType = "load_balancer"
)

// FindingID identifies the type of waste detected.
type FindingID string

const (
	FindingIdleInstance           FindingID = "IDLE_INSTANCE"
	FindingStoppedInstance        FindingID = "STOPPED_INSTANCE"
	FindingDetachedDisk           FindingID = "DETACHED_DISK"
	FindingUnusedAddress          FindingID = "UNUSED_ADDRESS"
	FindingStaleSnapshot          FindingID = "STALE_SNAPSHOT"
	FindingEmptyInstanceGroup     FindingID = "EMPTY_INSTANCE_GROUP"
	FindingUnhealthyInstanceGroup FindingID = "UNHEALTHY_INSTANCE_GROUP"
	FindingIdleCloudSQL           FindingID = "IDLE_CLOUD_SQL"
	FindingUnusedFirewall         FindingID = "UNUSED_FIREWALL"
	FindingNATIdle                FindingID = "NAT_IDLE"
	FindingNATLowTraffic          FindingID = "NAT_LOW_TRAFFIC"
	FindingFunctionIdle           FindingID = "FUNCTION_IDLE"
	FindingLBIdle                 FindingID = "LB_IDLE"
	FindingLBUnhealthy            FindingID = "LB_UNHEALTHY"
	FindingLBNoBackends           FindingID = "LB_NO_BACKENDS"
)

// Finding represents a single waste detection result.
type Finding struct {
	ID                    FindingID      `json:"id"`
	Severity              Severity       `json:"severity"`
	ResourceType          ResourceType   `json:"resource_type"`
	ResourceID            string         `json:"resource_id"`
	ResourceName          string         `json:"resource_name,omitempty"`
	Project               string         `json:"project"`
	Zone                  string         `json:"zone,omitempty"`
	Message               string         `json:"message"`
	EstimatedMonthlyWaste float64        `json:"estimated_monthly_waste"`
	Metadata              map[string]any `json:"metadata,omitempty"`
}

// ScanResult holds all findings from scanning a set of resources.
type ScanResult struct {
	Findings         []Finding `json:"findings"`
	Errors           []string  `json:"errors,omitempty"`
	ResourcesScanned int       `json:"resources_scanned"`
	ProjectsScanned  int       `json:"projects_scanned"`
}

// ScanConfig holds parameters that control scanning behavior.
type ScanConfig struct {
	IdleDays       int
	StaleDays      int
	MinMonthlyCost float64
	Exclude        ExcludeConfig
}

// ExcludeConfig holds resource exclusion rules.
type ExcludeConfig struct {
	ResourceIDs map[string]bool
	Labels      map[string]string
}

// ScanProgress reports scanning progress to callers.
type ScanProgress struct {
	Project   string
	Scanner   string
	Message   string
	Timestamp time.Time
}
