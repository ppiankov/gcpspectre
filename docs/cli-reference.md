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

**SpectreHub** (`--format spectrehub`): `spectre/v1` envelope for SpectreHub ingestion.


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

