package gcp

import (
	"context"
	"time"
)

// ComputeInstance represents a GCP compute instance.
type ComputeInstance struct {
	ID          uint64
	Name        string
	Zone        string
	Project     string
	MachineType string // short name, e.g. "e2-medium"
	Status      string // RUNNING, STOPPED, TERMINATED, etc.
	Labels      map[string]string
	Tags        []string // network tags
	LastStarted time.Time
	CreateTime  time.Time
}

// PersistentDisk represents a GCP persistent disk.
type PersistentDisk struct {
	ID         uint64
	Name       string
	Zone       string
	Project    string
	DiskType   string // "pd-standard", "pd-ssd", "pd-balanced"
	SizeGB     int64
	Status     string // READY, CREATING, FAILED, etc.
	Users      []string
	Labels     map[string]string
	LastAttach time.Time
	CreateTime time.Time
}

// StaticAddress represents a GCP static IP address.
type StaticAddress struct {
	ID          uint64
	Name        string
	Region      string
	Project     string
	Address     string
	Status      string // IN_USE, RESERVED
	Users       []string
	AddressType string // INTERNAL, EXTERNAL
	CreateTime  time.Time
}

// DiskSnapshot represents a GCP disk snapshot.
type DiskSnapshot struct {
	ID               uint64
	Name             string
	Project          string
	SourceDisk       string
	DiskSizeGB       int64
	StorageBytes     int64
	Status           string
	Labels           map[string]string
	CreateTime       time.Time
	StorageLocations []string
}

// InstanceGroupInfo represents a GCP instance group.
type InstanceGroupInfo struct {
	ID        uint64
	Name      string
	Zone      string
	Project   string
	Size      int
	IsManaged bool
}

// CloudSQLInstance represents a GCP Cloud SQL instance.
type CloudSQLInstance struct {
	Name            string
	Project         string
	Region          string
	Tier            string // e.g., "db-f1-micro"
	State           string // RUNNABLE, STOPPED, etc.
	DatabaseVersion string
	Labels          map[string]string
}

// FirewallRule represents a GCP VPC firewall rule.
type FirewallRule struct {
	ID         uint64
	Name       string
	Project    string
	Network    string
	Direction  string // INGRESS, EGRESS
	Priority   int64
	TargetTags []string
	Disabled   bool
}

// RouterInfo represents a GCP Cloud Router with NAT configurations.
type RouterInfo struct {
	ID      uint64
	Name    string
	Region  string
	Project string
	NATs    []NATConfig
}

// NATConfig represents a single Cloud NAT configuration on a router.
type NATConfig struct {
	Name string
}

// CloudFunction represents a GCP Cloud Function (v2).
type CloudFunction struct {
	Name    string
	Region  string
	Project string
	Runtime string
	State   string
	Labels  map[string]string
}

// ForwardingRuleInfo represents a GCP forwarding rule.
type ForwardingRuleInfo struct {
	ID                  uint64
	Name                string
	Region              string
	Project             string
	Target              string
	IPAddress           string
	LoadBalancingScheme string
}

// BackendServiceInfo represents a GCP backend service.
type BackendServiceInfo struct {
	ID       uint64
	Name     string
	Project  string
	Backends int
	Protocol string
	HealthOK bool
}

// ComputeAPI abstracts GCP Compute Engine list operations.
type ComputeAPI interface {
	ListInstances(ctx context.Context, project string) ([]ComputeInstance, error)
	ListDisks(ctx context.Context, project string) ([]PersistentDisk, error)
	ListAddresses(ctx context.Context, project string) ([]StaticAddress, error)
	ListSnapshots(ctx context.Context, project string) ([]DiskSnapshot, error)
	ListInstanceGroups(ctx context.Context, project string) ([]InstanceGroupInfo, error)
	ListFirewalls(ctx context.Context, project string) ([]FirewallRule, error)
	ListRouters(ctx context.Context, project string) ([]RouterInfo, error)
	ListForwardingRules(ctx context.Context, project string) ([]ForwardingRuleInfo, error)
	ListBackendServices(ctx context.Context, project string) ([]BackendServiceInfo, error)
}

// CloudSQLAPI abstracts GCP Cloud SQL Admin operations.
type CloudSQLAPI interface {
	ListInstances(ctx context.Context, project string) ([]CloudSQLInstance, error)
}

// CloudFunctionsAPI abstracts GCP Cloud Functions list operations.
type CloudFunctionsAPI interface {
	ListFunctions(ctx context.Context, project string) ([]CloudFunction, error)
}

// MonitoringAPI abstracts GCP Cloud Monitoring metric queries.
type MonitoringAPI interface {
	FetchMetricMean(ctx context.Context, project, metricType, resourceLabel string, resourceIDs []string, lookbackDays int) (map[string]float64, error)
}
