# Changelog

All notable changes to GCPSpectre will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-02-27

### Added

- Multi-project scanning with bounded concurrency (max 4 projects, 10 scanners per project)
- 7 resource scanners: Compute instances (idle CPU, stopped), persistent disks (detached), static IPs (unused), snapshots (stale), instance groups (empty), Cloud SQL (idle CPU), firewall rules (unused target tags)
- Cloud Monitoring integration for CPU utilization metrics
- Embedded on-demand pricing data via `go:embed` for cost estimation
- Analyzer with minimum cost filtering and summary aggregation
- 4 output formats: text (terminal table), JSON (`spectre/v1` envelope), SARIF (v2.1.0), SpectreHub (`spectrehub/v1`)
- Configuration via `.gcpspectre.yaml` with `gcpspectre init` generator
- Resource exclusion by ID and labels
- GoReleaser config for multi-platform releases (Linux, macOS, Windows; amd64, arm64)
- Homebrew formula via GoReleaser brews section
- CI/CD: GitHub Actions for build, test, lint

[0.1.0]: https://github.com/ppiankov/gcpspectre/releases/tag/v0.1.0
