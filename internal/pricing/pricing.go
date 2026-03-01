package pricing

import (
	"encoding/json"
	"log/slog"
	"strings"
)

const hoursPerMonth = 730

// pricingDB holds the parsed pricing data keyed by resource type, then instance/volume type, then region.
var pricingDB map[string]map[string]map[string]float64

func init() {
	if err := json.Unmarshal(pricingData, &pricingDB); err != nil {
		slog.Warn("Failed to parse embedded pricing data", "error", err)
		pricingDB = make(map[string]map[string]map[string]float64)
	}
}

// lookupHourly returns the hourly on-demand price for a resource type, instance type, and region.
func lookupHourly(resourceType, instanceType, region string) (float64, bool) {
	types, ok := pricingDB[resourceType]
	if !ok {
		return 0, false
	}
	regions, ok := types[instanceType]
	if !ok {
		return 0, false
	}
	price, ok := regions[region]
	if !ok {
		price, ok = regions["us-central1"]
		if !ok {
			return 0, false
		}
	}
	return price, true
}

// lookupMonthly returns the monthly flat rate for a resource type and region.
func lookupMonthly(resourceType, region string) (float64, bool) {
	types, ok := pricingDB[resourceType]
	if !ok {
		return 0, false
	}
	regions, ok := types["default"]
	if !ok {
		return 0, false
	}
	price, ok := regions[region]
	if !ok {
		price, ok = regions["us-central1"]
		if !ok {
			return 0, false
		}
	}
	return price, true
}

// regionFromZone extracts the region from a GCP zone (e.g., "us-central1-a" -> "us-central1").
func regionFromZone(zone string) string {
	parts := strings.Split(zone, "-")
	if len(parts) >= 3 {
		return strings.Join(parts[:len(parts)-1], "-")
	}
	return zone
}

// MonthlyInstanceCost returns the estimated monthly cost for a GCP compute instance.
func MonthlyInstanceCost(machineType, zone string) float64 {
	region := regionFromZone(zone)
	hourly, ok := lookupHourly("compute_instance", machineType, region)
	if !ok {
		return 0
	}
	return hourly * hoursPerMonth
}

// MonthlyDiskCost returns the estimated monthly cost for a persistent disk.
func MonthlyDiskCost(diskType string, sizeGiB int, zone string) float64 {
	region := regionFromZone(zone)
	perGiB, ok := lookupHourly("persistent_disk", diskType, region)
	if !ok {
		return 0
	}
	return perGiB * float64(sizeGiB)
}

// MonthlyAddressCost returns the monthly cost of an unused static external IP.
func MonthlyAddressCost(region string) float64 {
	cost, _ := lookupMonthly("static_ip", region)
	return cost
}

// MonthlySnapshotCost returns the estimated monthly cost for a snapshot.
func MonthlySnapshotCost(sizeGiB int, region string) float64 {
	perGiB, ok := lookupHourly("snapshot", "default", region)
	if !ok {
		return 0
	}
	return perGiB * float64(sizeGiB)
}

// MonthlyCloudSQLCost returns the estimated monthly cost for a Cloud SQL instance.
func MonthlyCloudSQLCost(tier, region string) float64 {
	hourly, ok := lookupHourly("cloud_sql", tier, region)
	if !ok {
		return 0
	}
	return hourly * hoursPerMonth
}

// MonthlyNATCost returns the estimated monthly cost for a Cloud NAT gateway.
func MonthlyNATCost(region string) float64 {
	cost, _ := lookupMonthly("cloud_nat", region)
	return cost
}

// MonthlyFunctionCost returns the estimated monthly base cost for an idle Cloud Function.
func MonthlyFunctionCost(region string) float64 {
	cost, _ := lookupMonthly("cloud_function", region)
	return cost
}

// MonthlyLBCost returns the estimated monthly cost for a forwarding rule.
func MonthlyLBCost(region string) float64 {
	cost, _ := lookupMonthly("load_balancer", region)
	return cost
}
