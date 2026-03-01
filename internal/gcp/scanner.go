package gcp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// ResourceScanner is the interface each resource-type scanner implements.
type ResourceScanner interface {
	Scan(ctx context.Context, cfg ScanConfig) (*ScanResult, error)
	Type() ResourceType
}

// MultiProjectScanner orchestrates scanning across multiple GCP projects.
type MultiProjectScanner struct {
	compute     ComputeAPI
	monitoring  MonitoringAPI
	cloudSQL    CloudSQLAPI
	functions   CloudFunctionsAPI
	pubsub      PubSubAPI
	projects    []string
	concurrency int
	scanConfig  ScanConfig
	progressFn  func(ScanProgress)
}

// NewMultiProjectScanner creates a scanner that runs across the specified projects.
func NewMultiProjectScanner(compute ComputeAPI, monitoring MonitoringAPI, cloudSQL CloudSQLAPI, projects []string, concurrency int, scanCfg ScanConfig) *MultiProjectScanner {
	if concurrency <= 0 {
		concurrency = 4
	}
	return &MultiProjectScanner{
		compute:     compute,
		monitoring:  monitoring,
		cloudSQL:    cloudSQL,
		projects:    projects,
		concurrency: concurrency,
		scanConfig:  scanCfg,
	}
}

// SetFunctionsAPI sets the Cloud Functions API client for function scanning.
func (s *MultiProjectScanner) SetFunctionsAPI(functions CloudFunctionsAPI) {
	s.functions = functions
}

// SetPubSubAPI sets the Pub/Sub API client for topic and subscription scanning.
func (s *MultiProjectScanner) SetPubSubAPI(pubsub PubSubAPI) {
	s.pubsub = pubsub
}

// SetProgressFn sets a callback for progress updates.
func (s *MultiProjectScanner) SetProgressFn(fn func(ScanProgress)) {
	s.progressFn = fn
}

// ScanAll runs all resource scanners across all configured projects.
func (s *MultiProjectScanner) ScanAll(ctx context.Context) (*ScanResult, error) {
	var (
		mu       sync.Mutex
		combined ScanResult
	)

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(s.concurrency)

	for _, project := range s.projects {
		project := project
		g.Go(func() error {
			slog.Info("Scanning project", "project", project)
			result, err := s.scanProject(ctx, project)
			if err != nil {
				mu.Lock()
				combined.Errors = append(combined.Errors, fmt.Sprintf("%s: %v", project, err))
				mu.Unlock()
				slog.Warn("Project scan failed", "project", project, "error", err)
				return nil
			}

			mu.Lock()
			combined.Findings = append(combined.Findings, result.Findings...)
			combined.Errors = append(combined.Errors, result.Errors...)
			combined.ResourcesScanned += result.ResourcesScanned
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	combined.ProjectsScanned = len(s.projects)
	return &combined, nil
}

func (s *MultiProjectScanner) scanProject(ctx context.Context, project string) (*ScanResult, error) {
	scanners := s.buildScanners(project)

	var (
		mu     sync.Mutex
		result ScanResult
	)

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10)

	for _, scanner := range scanners {
		scanner := scanner
		g.Go(func() error {
			slog.Debug("Running scanner", "type", scanner.Type(), "project", project)
			if s.progressFn != nil {
				s.progressFn(ScanProgress{
					Project:   project,
					Scanner:   string(scanner.Type()),
					Message:   fmt.Sprintf("scanning %s", scanner.Type()),
					Timestamp: time.Now(),
				})
			}

			sr, err := scanner.Scan(ctx, s.scanConfig)
			if err != nil {
				mu.Lock()
				result.Errors = append(result.Errors, fmt.Sprintf("%s/%s: %v", project, scanner.Type(), err))
				mu.Unlock()
				slog.Warn("Scanner failed", "type", scanner.Type(), "project", project, "error", err)
				return nil
			}

			mu.Lock()
			result.Findings = append(result.Findings, sr.Findings...)
			result.ResourcesScanned += sr.ResourcesScanned
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *MultiProjectScanner) buildScanners(project string) []ResourceScanner {
	return []ResourceScanner{
		NewInstanceScanner(s.compute, s.monitoring, project),
		NewDiskScanner(s.compute, project),
		NewAddressScanner(s.compute, project),
		NewSnapshotScanner(s.compute, project),
		NewInstanceGroupScanner(s.compute, project),
		NewCloudSQLScanner(s.cloudSQL, s.monitoring, project),
		NewFirewallScanner(s.compute, project),
		NewNATScanner(s.compute, s.monitoring, project),
		NewFunctionsScanner(s.functions, s.monitoring, project),
		NewLBScanner(s.compute, s.monitoring, project),
		NewPubSubScanner(s.pubsub, s.monitoring, project),
	}
}
