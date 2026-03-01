package gcp

import (
	"context"
	"fmt"
	"strings"

	cloudfunctions "google.golang.org/api/cloudfunctions/v2"
)

// GCPCloudFunctionsClient implements CloudFunctionsAPI using the Cloud Functions v2 API.
type GCPCloudFunctionsClient struct {
	service *cloudfunctions.Service
}

// NewCloudFunctionsClient creates a CloudFunctionsAPI backed by GCP Application Default Credentials.
func NewCloudFunctionsClient(ctx context.Context) (*GCPCloudFunctionsClient, error) {
	svc, err := cloudfunctions.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("create Cloud Functions client: %w", err)
	}
	return &GCPCloudFunctionsClient{service: svc}, nil
}

func (c *GCPCloudFunctionsClient) ListFunctions(ctx context.Context, project string) ([]CloudFunction, error) {
	parent := fmt.Sprintf("projects/%s/locations/-", project)
	var result []CloudFunction

	call := c.service.Projects.Locations.Functions.List(parent).Context(ctx)
	err := call.Pages(ctx, func(resp *cloudfunctions.ListFunctionsResponse) error {
		for _, fn := range resp.Functions {
			name := fn.Name
			// Extract short name from full resource name
			if parts := strings.Split(fn.Name, "/"); len(parts) > 0 {
				name = parts[len(parts)-1]
			}
			region := ""
			if parts := strings.Split(fn.Name, "/"); len(parts) >= 4 {
				region = parts[3]
			}
			runtime := ""
			if fn.BuildConfig != nil {
				runtime = fn.BuildConfig.Runtime
			}
			result = append(result, CloudFunction{
				Name:    name,
				Region:  region,
				Project: project,
				Runtime: runtime,
				State:   fn.State,
				Labels:  fn.Labels,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list Cloud Functions: %w", err)
	}
	return result, nil
}
