package commands

import (
	"log/slog"

	"github.com/ppiankov/gcpspectre/internal/config"
	"github.com/ppiankov/gcpspectre/internal/logging"
	"github.com/spf13/cobra"
)

var (
	verbose  bool
	projects []string
	version  string
	commit   string
	date     string
	cfg      config.Config
)

var rootCmd = &cobra.Command{
	Use:   "gcpspectre",
	Short: "gcpspectre — GCP resource waste auditor",
	Long: `gcpspectre finds idle, orphaned, and oversized GCP resources that cost money
for nothing. It scans compute instances, persistent disks, static IPs, snapshots,
instance groups, Cloud SQL instances, and firewall rules across projects.

Each finding includes an estimated monthly waste in USD.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logging.Init(verbose)
		loaded, err := config.Load(".")
		if err != nil {
			slog.Warn("Failed to load config file", "error", err)
		} else {
			cfg = loaded
		}
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command with injected build info.
func Execute(v, c, d string) error {
	version = v
	commit = c
	date = d
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringSliceVar(&projects, "project", nil, "GCP project ID (repeatable)")
	rootCmd.AddCommand(versionCmd)
}
