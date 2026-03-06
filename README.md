# gcpspectre

[![CI](https://github.com/ppiankov/gcpspectre/actions/workflows/ci.yml/badge.svg)](https://github.com/ppiankov/gcpspectre/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ppiankov/gcpspectre)](https://goreportcard.com/report/github.com/ppiankov/gcpspectre)
[![ANCC](https://img.shields.io/badge/ANCC-compliant-brightgreen)](https://ancc.dev)

**gcpspectre** — GCP resource waste auditor with cost estimates. Part of [SpectreHub](https://github.com/ppiankov/spectrehub).

## What it is

- Scans Compute Engine instances, persistent disks, static IPs, snapshots, instance groups, Cloud SQL, and firewall rules
- Detects idle, orphaned, and oversized resources using Cloud Monitoring metrics
- Estimates monthly waste in USD per finding
- Supports configurable thresholds and exclusions
- Outputs text, JSON, SARIF, and SpectreHub formats

## What it is NOT

- Not a real-time monitor — point-in-time scanner
- Not a remediation tool — reports only, never modifies resources
- Not a security scanner — checks utilization, not vulnerabilities
- Not a billing replacement — uses embedded on-demand pricing

## Quick start

### Homebrew

```sh
brew tap ppiankov/tap
brew install gcpspectre
```

### From source

```sh
git clone https://github.com/ppiankov/gcpspectre.git
cd gcpspectre
make build
```

### Usage

```sh
gcpspectre scan --project my-project --format json
```

## CLI commands

| Command | Description |
|---------|-------------|
| `gcpspectre scan` | Scan GCP project for idle and wasteful resources |
| `gcpspectre init` | Generate config file and service account |
| `gcpspectre version` | Print version |

## SpectreHub integration

gcpspectre feeds GCP resource waste findings into [SpectreHub](https://github.com/ppiankov/spectrehub) for unified visibility across your infrastructure.

```sh
spectrehub collect --tool gcpspectre
```

## Safety

gcpspectre operates in **read-only mode**. It inspects and reports — never modifies, deletes, or alters your resources.

## Documentation

| Document | Contents |
|----------|----------|
| [CLI Reference](docs/cli-reference.md) | Full command reference, flags, and configuration |

## License

MIT — see [LICENSE](LICENSE).

---

Built by [Obsta Labs](https://obstalabs.dev)
