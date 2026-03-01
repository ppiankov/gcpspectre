package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// DryRunPlan describes what a scan would do without executing.
type DryRunPlan struct {
	Projects       []string         `json:"projects"`
	Scanners       []string         `json:"scanners"`
	IdleDays       int              `json:"idle_days"`
	StaleDays      int              `json:"stale_days"`
	MinMonthlyCost float64          `json:"min_monthly_cost"`
	Exclusions     DryRunExclusions `json:"exclusions"`
	ConfigPath     string           `json:"config_path"`
}

// DryRunExclusions describes configured exclusion rules.
type DryRunExclusions struct {
	ResourceIDs []string `json:"resource_ids,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

var scannerNames = []string{
	"compute_instance",
	"persistent_disk",
	"static_ip",
	"snapshot",
	"instance_group",
	"cloud_sql",
	"firewall_rule",
	"cloud_nat",
	"cloud_function",
	"load_balancer",
}

func printDryRun(cmd *cobra.Command, projects []string) error {
	excludeLabels := mergeSlices(cfg.Exclude.Labels, scanFlags.excludeLabels)

	plan := DryRunPlan{
		Projects:       projects,
		Scanners:       scannerNames,
		IdleDays:       scanFlags.idleDays,
		StaleDays:      scanFlags.staleDays,
		MinMonthlyCost: scanFlags.minMonthlyCost,
		Exclusions: DryRunExclusions{
			ResourceIDs: cfg.Exclude.ResourceIDs,
			Labels:      excludeLabels,
		},
		ConfigPath: findConfigPath(),
	}

	w := cmd.OutOrStdout()
	if scanFlags.format == "json" {
		return printDryRunJSON(w, plan)
	}
	return printDryRunText(w, plan)
}

func printDryRunJSON(w io.Writer, plan DryRunPlan) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(plan)
}

func printDryRunText(w io.Writer, plan DryRunPlan) error {
	fmt.Fprintf(w, "Scan Plan (dry-run)\n\n")
	fmt.Fprintf(w, "Projects:\n")
	for _, p := range plan.Projects {
		fmt.Fprintf(w, "  - %s\n", p)
	}
	fmt.Fprintf(w, "\nScanners:\n")
	for _, s := range plan.Scanners {
		fmt.Fprintf(w, "  - %s\n", s)
	}
	fmt.Fprintf(w, "\nSettings:\n")
	fmt.Fprintf(w, "  idle-days:       %d\n", plan.IdleDays)
	fmt.Fprintf(w, "  stale-days:      %d\n", plan.StaleDays)
	fmt.Fprintf(w, "  min-monthly-cost: %.2f\n", plan.MinMonthlyCost)
	if len(plan.Exclusions.ResourceIDs) > 0 || len(plan.Exclusions.Labels) > 0 {
		fmt.Fprintf(w, "\nExclusions:\n")
		for _, id := range plan.Exclusions.ResourceIDs {
			fmt.Fprintf(w, "  resource-id: %s\n", id)
		}
		for _, l := range plan.Exclusions.Labels {
			fmt.Fprintf(w, "  label: %s\n", l)
		}
	}
	fmt.Fprintf(w, "\nConfig: %s\n", plan.ConfigPath)
	return nil
}

func findConfigPath() string {
	candidates := []string{".gcpspectre.yaml", ".gcpspectre.yml"}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err == nil {
			if fileExists(abs) {
				return abs
			}
		}
	}
	return "none"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// mergeSlices combines two string slices, deduplicating entries.
func mergeSlices(a, b []string) []string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var result []string
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
