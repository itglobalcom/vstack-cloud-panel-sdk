package sdk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const (
	projectPath      = "project"
	locationsPath    = "locations"
	imagesPath       = "images"
	applicationsPath = "applications"
)

// Response types for metadata operations
type (
	// GetProjectResponse represents project response
	GetProjectResponse struct {
		Project *entities.Project `json:"project,omitempty"`
	}

	// ListLocationsResponse represents locations list response
	ListLocationsResponse struct {
		Locations []entities.Location `json:"locations,omitempty"`
	}

	// ListImagesResponse represents images list response
	ListImagesResponse struct {
		Images []entities.Image `json:"images,omitempty"`
	}

	// ListApplicationsResponse represents applications list response
	ListApplicationsResponse struct {
		Applications []entities.Application `json:"applications,omitempty"`
	}
)

// GetProject retrieves current project information including ID and balance
func (c *CloudClient) GetProject(ctx context.Context) (*entities.Project, error) {
	req, err := c.newRequest(ctx, http.MethodGet, projectPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get project request: %w", err)
	}

	var resp GetProjectResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if resp.Project == nil {
		return nil, fmt.Errorf("project not found in response: %w", ErrNotFound)
	}

	return resp.Project, nil
}

// GetLocations retrieves list of available locations with volume size limits
func (c *CloudClient) GetLocations(ctx context.Context) ([]entities.Location, error) {
	req, err := c.newRequest(ctx, http.MethodGet, locationsPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list locations request: %w", err)
	}

	var resp ListLocationsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list locations: %w", err)
	}

	return resp.Locations, nil
}

// GetImages retrieves list of available OS images
func (c *CloudClient) GetImages(ctx context.Context) ([]entities.Image, error) {
	req, err := c.newRequest(ctx, http.MethodGet, imagesPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list images request: %w", err)
	}

	var resp ListImagesResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	return resp.Images, nil
}

// GetApplications retrieves list of available applications
// locationID is optional - if provided, filters applications by location
func (c *CloudClient) GetApplications(ctx context.Context, locationID string) ([]entities.Application, error) {
	path := applicationsPath

	// Add location filter if provided
	if locationID != "" {
		params := url.Values{}
		params.Add("location_id", locationID)
		path = fmt.Sprintf("%s?%s", path, params.Encode())
	}

	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list applications request: %w", err)
	}

	var resp ListApplicationsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	return resp.Applications, nil
}
