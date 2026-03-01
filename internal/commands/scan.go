package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ppiankov/gcpspectre/internal/analyzer"
	"github.com/ppiankov/gcpspectre/internal/gcp"
	"github.com/ppiankov/gcpspectre/internal/report"
	"github.com/spf13/cobra"
)

var scanFlags struct {
	idleDays       int
	staleDays      int
	format         string
	outputFile     string
	minMonthlyCost float64
	timeout        time.Duration
	excludeLabels  []string
	failOn         string
	threshold      int
	dryRun         bool
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan GCP resources for waste",
	Long: `Scan GCP resources across projects to find idle, orphaned, and oversized
resources. Reports estimated monthly waste in USD for each finding.

Requires Application Default Credentials:
  gcloud auth application-default login`,
	RunE: runScan,
}

func init() {
	scanCmd.Flags().IntVar(&scanFlags.idleDays, "idle-days", 7, "Lookback window for utilization metrics (days)")
	scanCmd.Flags().IntVar(&scanFlags.staleDays, "stale-days", 90, "Age threshold for stale snapshots (days)")
	scanCmd.Flags().StringVar(&scanFlags.format, "format", "text", "Output format: text, json, sarif, spectrehub")
	scanCmd.Flags().StringVarP(&scanFlags.outputFile, "output", "o", "", "Output file path (default: stdout)")
	scanCmd.Flags().Float64Var(&scanFlags.minMonthlyCost, "min-monthly-cost", 1.0, "Minimum monthly cost to report ($)")
	scanCmd.Flags().DurationVar(&scanFlags.timeout, "timeout", 10*time.Minute, "Scan timeout")
	scanCmd.Flags().StringSliceVar(&scanFlags.excludeLabels, "exclude-label", nil, "Exclude resources by label (key=value or key-only, repeatable)")
	scanCmd.Flags().StringVar(&scanFlags.failOn, "fail-on", "", "Exit non-zero when findings meet severity (high, medium, low)")
	scanCmd.Flags().IntVar(&scanFlags.threshold, "threshold", 1, "Minimum finding count to trigger non-zero exit (requires --fail-on)")
	scanCmd.Flags().BoolVar(&scanFlags.dryRun, "dry-run", false, "Print scan plan without executing")

	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	if scanFlags.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, scanFlags.timeout)
		defer cancel()
	}

	applyConfigDefaults()

	projectList := resolveProjects()
	if len(projectList) == 0 {
		return fmt.Errorf("no projects specified; use --project flag or set projects in .gcpspectre.yaml")
	}
	slog.Info("Scanning projects", "count", len(projectList), "projects", projectList)

	if scanFlags.dryRun {
		return printDryRun(cmd, projectList)
	}

	computeClient, err := gcp.NewComputeClient(ctx)
	if err != nil {
		return enhanceError("initialize GCP Compute client", err)
	}
	defer computeClient.Close()

	monitoringClient, err := gcp.NewMonitoringClient(ctx)
	if err != nil {
		return enhanceError("initialize GCP Monitoring client", err)
	}
	defer monitoringClient.Close()

	cloudSQLClient, err := gcp.NewCloudSQLClient(ctx)
	if err != nil {
		slog.Warn("Cloud SQL client unavailable, skipping SQL scans", "error", err)
	}

	functionsClient, err := gcp.NewCloudFunctionsClient(ctx)
	if err != nil {
		slog.Warn("Cloud Functions client unavailable, skipping function scans", "error", err)
	}

	scanCfg := gcp.ScanConfig{
		IdleDays:       scanFlags.idleDays,
		StaleDays:      scanFlags.staleDays,
		MinMonthlyCost: scanFlags.minMonthlyCost,
		Exclude: gcp.ExcludeConfig{
			ResourceIDs: parseResourceIDs(cfg.Exclude.ResourceIDs),
			Labels:      mergeExcludeLabels(cfg.Exclude.Labels, scanFlags.excludeLabels),
		},
	}

	scanner := gcp.NewMultiProjectScanner(computeClient, monitoringClient, cloudSQLClient, projectList, 4, scanCfg)
	if functionsClient != nil {
		scanner.SetFunctionsAPI(functionsClient)
	}
	result, err := scanner.ScanAll(ctx)
	if err != nil {
		return enhanceError("scan resources", err)
	}

	analysis := analyzer.Analyze(result, analyzer.AnalyzerConfig{
		MinMonthlyCost: scanFlags.minMonthlyCost,
	})

	data := report.Data{
		Tool:      "gcpspectre",
		Version:   version,
		Timestamp: time.Now().UTC(),
		Target: report.Target{
			Type:    "gcp-projects",
			URIHash: computeTargetHash(projectList),
		},
		Config: report.ReportConfig{
			Projects:       projectList,
			IdleDays:       scanFlags.idleDays,
			StaleDays:      scanFlags.staleDays,
			MinMonthlyCost: scanFlags.minMonthlyCost,
		},
		Findings: analysis.Findings,
		Summary:  analysis.Summary,
		Errors:   convertToScanErrors(analysis.Errors),
	}

	reporter, err := selectReporter(scanFlags.format, scanFlags.outputFile)
	if err != nil {
		return err
	}
	if err := reporter.Generate(data); err != nil {
		return err
	}

	if exitCode := report.ComputeExitCode(data.Findings, scanFlags.failOn, scanFlags.threshold); exitCode != 0 {
		return ExitCodeError{Code: exitCode}
	}
	return nil
}

func resolveProjects() []string {
	if len(projects) > 0 {
		return projects
	}
	if len(cfg.Projects) > 0 {
		return cfg.Projects
	}
	return nil
}

func applyConfigDefaults() {
	if scanFlags.format == "text" && cfg.Format != "" {
		scanFlags.format = cfg.Format
	}
	if scanFlags.idleDays == 7 && cfg.IdleDays > 0 {
		scanFlags.idleDays = cfg.IdleDays
	}
	if scanFlags.staleDays == 90 && cfg.StaleDays > 0 {
		scanFlags.staleDays = cfg.StaleDays
	}
	if scanFlags.minMonthlyCost == 1.0 && cfg.MinMonthlyCost > 0 {
		scanFlags.minMonthlyCost = cfg.MinMonthlyCost
	}
}

func selectReporter(format, outputFile string) (report.Reporter, error) {
	w := os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return nil, fmt.Errorf("create output file: %w", err)
		}
		w = f
	}

	switch format {
	case "json":
		return &report.JSONReporter{Writer: w}, nil
	case "text":
		return &report.TextReporter{Writer: w}, nil
	case "sarif":
		return &report.SARIFReporter{Writer: w}, nil
	case "spectrehub":
		return &report.SpectreHubReporter{Writer: w}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s (use text, json, sarif, or spectrehub)", format)
	}
}
