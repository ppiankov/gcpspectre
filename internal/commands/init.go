package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initFlags struct {
	force bool
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate sample configuration file",
	Long:  `Creates a sample .gcpspectre.yaml configuration file with default settings.`,
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVar(&initFlags.force, "force", false, "Overwrite existing files")
	rootCmd.AddCommand(initCmd)
}

func runInit(_ *cobra.Command, _ []string) error {
	configPath := ".gcpspectre.yaml"

	if err := writeIfNotExists(configPath, sampleConfig, initFlags.force); err != nil {
		return err
	}

	fmt.Printf("Created %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Edit .gcpspectre.yaml to add your project IDs")
	fmt.Println("  2. Authenticate: gcloud auth application-default login")
	fmt.Println("  3. Run: gcpspectre scan")
	return nil
}

func writeIfNotExists(path, content string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists (use --force to overwrite)", path)
		}
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	return os.WriteFile(path, []byte(content), 0o644)
}

const sampleConfig = `# gcpspectre configuration
# See: https://github.com/ppiankov/gcpspectre

# GCP project IDs to scan (required)
projects:
  # - my-project-id
  # - another-project

# Lookback window for utilization metrics (days)
idle_days: 7

# Age threshold for stale snapshots (days)
stale_days: 90

# Minimum monthly cost to report ($)
min_monthly_cost: 1.0

# Output format: text, json, sarif, or spectrehub
format: text

# Scan timeout
timeout: 10m

# Resources to exclude from scanning
# exclude:
#   resource_ids:
#     - "1234567890"
#   labels:
#     env: production
`
