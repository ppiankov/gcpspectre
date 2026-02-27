# GCPSpectre

[![CI](https://github.com/ppiankov/gcpspectre/actions/workflows/ci.yml/badge.svg)](https://github.com/ppiankov/gcpspectre/actions/workflows/ci.yml)
[![Go 1.24+](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

GCP resource waste auditor. Finds idle, orphaned, and oversized resources costing money for nothing.

Part of the [Spectre family](https://github.com/ppiankov) of infrastructure cleanup tools.

## What it is

GCPSpectre scans your Google Cloud projects for resources that are running but not doing useful work. It checks Cloud Monitoring metrics, attachment status, and usage patterns to identify waste across Compute Engine instances, persistent disks, static IPs, snapshots, instance groups, Cloud SQL instances, and firewall rules. Each finding includes an estimated monthly cost so you can prioritize cleanup by dollar impact.

## What it is NOT

- Not a real-time monitoring tool. GCPSpectre is a point-in-time scanner, not a daemon.
- Not a remediation tool. It reports waste and lets you decide what to do.
- Not a security scanner. It checks for idle resources, not misconfigurations or vulnerabilities.
- Not a billing replacement. Cost estimates are approximations based on embedded on-demand pricing, not your actual committed use discounts.
- Not a capacity planner. It flags underutilization, not rightsizing recommendations.

## Philosophy

*Principiis obsta* -- resist the beginnings.

Compute and storage are 50-70% of every cloud bill, and every project has waste. The longer idle resources sit, the harder they are to identify and the more they cost. GCPSpectre surfaces these conditions early -- in scheduled audits, in CI, in cost reviews -- so they can be addressed before they compound.

The tool presents evidence and lets humans decide. It does not auto-terminate instances, does not guess intent, and does not use ML where deterministic checks suffice.

## Installation

```bash
# Homebrew
brew install ppiankov/tap/gcpspectre

# From source
git clone https://github.com/ppiankov/gcpspectre.git
cd gcpspectre && make build
```

## Quick start

```bash
# Scan specific projects
gcpspectre scan --project my-project-id

# Scan multiple projects
gcpspectre scan --project proj-1 --project proj-2

# JSON output for automation
gcpspectre scan --format json --output report.json

# SARIF output for GitHub Security tab
gcpspectre scan --format sarif --output results.sarif

# Generate config file
gcpspectre init
```

Requires valid GCP credentials via Application Default Credentials (`gcloud auth application-default login`).

## What it audits

| Resource | Finding | Signal | Severity |
|----------|---------|--------|----------|
| Compute instances | `IDLE_INSTANCE` | CPU < 5% over idle window | high |
| Compute instances | `STOPPED_INSTANCE` | Stopped > 30 days | high |
| Persistent disks | `DETACHED_DISK` | Not attached to any instance | high |
| Static IPs | `UNUSED_ADDRESS` | Reserved but not in use | medium |
| Snapshots | `STALE_SNAPSHOT` | Older than stale threshold | medium |
| Instance groups | `EMPTY_INSTANCE_GROUP` | Zero instances in group | low |
| Cloud SQL | `IDLE_CLOUD_SQL` | CPU < 5% over idle window | high |
| Firewall rules | `UNUSED_FIREWALL` | Target tags match no running instance | low |

## Usage

```bash
gcpspectre scan [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | | GCP project ID (repeatable) |
| `--idle-days` | `7` | Lookback window for utilization metrics |
| `--stale-days` | `90` | Age threshold for snapshots |
| `--min-monthly-cost` | `1.0` | Minimum monthly cost to report ($) |
| `--format` | `text` | Output format: `text`, `json`, `sarif`, `spectrehub` |
| `-o, --output` | stdout | Output file path |
| `--timeout` | `10m` | Scan timeout |
| `--verbose` | `false` | Enable verbose logging |

**Other commands:**

| Command | Description |
|---------|-------------|
| `gcpspectre init` | Generate `.gcpspectre.yaml` config file |
| `gcpspectre version` | Print version, commit, and build date |

## Configuration

GCPSpectre reads `.gcpspectre.yaml` from the current directory:

```yaml
projects:
  - my-project-id
  - another-project
idle_days: 14
stale_days: 180
min_monthly_cost: 5.0
format: json
exclude:
  resource_ids:
    - "1234567890"
  labels:
    env: production
```

Generate a sample config with `gcpspectre init`.

## GCP permissions

GCPSpectre requires read-only access. The following roles provide sufficient permissions:

- `roles/compute.viewer` -- Compute Engine instances, disks, addresses, snapshots, instance groups, firewalls
- `roles/monitoring.viewer` -- Cloud Monitoring metrics for CPU utilization
- `roles/cloudsql.viewer` -- Cloud SQL instance listing

Or grant these specific permissions:

- `compute.instances.list`, `compute.disks.list`, `compute.addresses.list`, `compute.snapshots.list`, `compute.instanceGroups.list`, `compute.firewalls.list`
- `monitoring.timeSeries.list`
- `cloudsql.instances.list`

## Output formats

**Text** (default): Human-readable table with severity, resource, project, waste, and message.

**JSON** (`--format json`): `spectre/v1` envelope with findings and summary:
```json
{
  "$schema": "spectre/v1",
  "tool": "gcpspectre",
  "version": "0.1.0",
  "findings": [...],
  "summary": {
    "total_resources_scanned": 150,
    "total_findings": 5,
    "total_monthly_waste": 250.00
  }
}
```

**SARIF** (`--format sarif`): SARIF v2.1.0 for GitHub Security tab integration.

**SpectreHub** (`--format spectrehub`): `spectrehub/v1` envelope for SpectreHub ingestion.

## Architecture

```
gcpspectre/
├── cmd/gcpspectre/main.go         # Entry point (21 lines, LDFLAGS)
├── internal/
│   ├── commands/                  # Cobra CLI: scan, init, version
│   ├── gcp/                       # GCP SDK clients + 7 resource scanners
│   │   ├── client.go              # Resource types + API interfaces
│   │   ├── client_gcp.go          # Real GCP SDK implementations
│   │   ├── scanner.go             # MultiProjectScanner orchestrator
│   │   ├── instance.go            # Compute: idle CPU, stopped instances
│   │   ├── disk.go                # Persistent disks: detached
│   │   ├── address.go             # Static IPs: unused
│   │   ├── snapshot.go            # Snapshots: stale
│   │   ├── instancegroup.go       # Instance groups: empty
│   │   ├── cloudsql.go            # Cloud SQL: idle CPU
│   │   └── firewall.go            # Firewall rules: unused target tags
│   ├── pricing/                   # Embedded on-demand pricing (go:embed)
│   ├── analyzer/                  # Filter by min cost, compute summary
│   ├── config/                    # YAML config loader
│   ├── logging/                   # Structured logging setup
│   └── report/                    # Text, JSON, SARIF, SpectreHub reporters
├── Makefile
└── go.mod
```

Key design decisions:

- `cmd/gcpspectre/main.go` is minimal -- a single `Execute()` call with LDFLAGS version injection.
- All logic lives in `internal/` to prevent external import.
- Each resource type has its own scanner implementing `ResourceScanner` interface.
- Cloud Monitoring queries all time series for a metric type, then filters by resource ID locally.
- Two-level bounded concurrency: max 4 projects, max 10 scanner goroutines per project.
- Pricing data is embedded via `go:embed` with curated on-demand rates.
- Scanner errors are collected, not fatal -- one scanner failure does not abort the whole scan.
- API interfaces allow complete mock testing without GCP credentials.

## Project status

**Status: Beta** -- **v0.1.0** -- Pre-1.0

| Milestone | Status |
|-----------|--------|
| 7 resource scanners (instances, disks, IPs, snapshots, instance groups, Cloud SQL, firewalls) | Complete |
| Multi-project parallel scanning with bounded concurrency | Complete |
| Embedded on-demand pricing with per-finding cost estimates | Complete |
| 4 output formats (text, JSON, SARIF, SpectreHub) | Complete |
| Config file + init command | Complete |
| CI pipeline (test/lint/build) | Complete |
| Homebrew distribution | Complete |
| 118 tests, all scannable code >85% coverage | Complete |
| API stability guarantees | Partial |
| v1.0 release | Planned |

Pre-1.0: CLI flags and config schemas may change between minor versions. JSON output structure (`spectre/v1`) is stable.

## Known limitations

- **Approximate pricing.** Cost estimates use embedded on-demand rates, not your actual pricing (committed use discounts, sustained use discounts). Treat estimates as directional, not exact.
- **Cloud Monitoring data lag.** Metrics may take up to 15 minutes to appear. Very recently provisioned resources may not have enough data for idle detection.
- **No cross-org support.** Scans individual projects. Use `--project` flags or config to scan multiple projects.
- **No rightsizing.** Flags underutilized resources but does not recommend smaller machine types.
- **Firewall rule heuristic.** Only checks target tag attachment to running instances. Rules using service accounts or IP ranges are not flagged.
- **Single metric thresholds.** CPU < 5% is a simple heuristic. Some workloads (batch, cron) may appear idle but are not.

## License

MIT License -- see [LICENSE](LICENSE).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Issues and pull requests welcome.

Part of the Spectre family:
[AWSSpectre](https://github.com/ppiankov/awsspectre) |
[GCSSpectre](https://github.com/ppiankov/gcsspectre) |
[IAMSpectre](https://github.com/ppiankov/iamspectre) |
[S3Spectre](https://github.com/ppiankov/s3spectre) |
[VaultSpectre](https://github.com/ppiankov/vaultspectre)
