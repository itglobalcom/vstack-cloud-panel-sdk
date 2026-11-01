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

	projectConsumptionPath         = "project/consumption"
	projectConsumptionResourcesSub = "resources"
	projectConsumptionDailySub     = "daily"
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

	// GetProjectConsumptionResponse represents project consumption by service response
	GetProjectConsumptionResponse struct {
		Consumption *entities.ProjectConsumption `json:"consumption,omitempty"`
	}

	// GetProjectConsumptionResourcesResponse represents project consumption by resource response
	GetProjectConsumptionResourcesResponse struct {
		Consumption *entities.ProjectConsumptionResources `json:"consumption,omitempty"`
	}

	// GetProjectConsumptionDailyResponse represents project consumption by day response
	GetProjectConsumptionDailyResponse struct {
		Consumption *entities.ProjectConsumptionDaily `json:"consumption,omitempty"`
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

// buildProjectConsumptionPath builds project/consumption[/<sub>] with the period
// and the service filter as query; empty values are left out.
func buildProjectConsumptionPath(sub string, period *entities.ConsumptionPeriodRequest, service entities.ConsumptionService) string {
	path := projectConsumptionPath
	if sub != "" {
		path = fmt.Sprintf("%s/%s", path, sub)
	}

	params := url.Values{}
	if period != nil {
		if period.From != "" {
			params.Set("from", period.From)
		}
		if period.To != "" {
			params.Set("to", period.To)
		}
	}
	if service != "" {
		params.Set("service", string(service))
	}
	if len(params) == 0 {
		return path
	}
	return fmt.Sprintf("%s?%s", path, params.Encode())
}

// GetProjectConsumption retrieves the project consumption over a period broken
// down by service. A nil period or empty dates take the API defaults: from the
// first day of the current month to today.
func (c *CloudClient) GetProjectConsumption(ctx context.Context, period *entities.ConsumptionPeriodRequest) (*entities.ProjectConsumption, error) {
	if period != nil {
		if err := period.Validate(); err != nil {
			return nil, fmt.Errorf("invalid consumption period: %w", err)
		}
	}

	req, err := c.newRequest(ctx, http.MethodGet, buildProjectConsumptionPath("", period, ""), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get project consumption request: %w", err)
	}

	var resp GetProjectConsumptionResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get project consumption: %w", err)
	}

	if resp.Consumption == nil {
		return nil, fmt.Errorf("consumption not found in response: %w", ErrNotFound)
	}

	return resp.Consumption, nil
}

// GetProjectConsumptionResources retrieves the project consumption over a period
// broken down by root resource. An empty service reads every service; a nil
// period or empty dates take the API defaults.
func (c *CloudClient) GetProjectConsumptionResources(ctx context.Context, period *entities.ConsumptionPeriodRequest, service entities.ConsumptionService) (*entities.ProjectConsumptionResources, error) {
	if period != nil {
		if err := period.Validate(); err != nil {
			return nil, fmt.Errorf("invalid consumption period: %w", err)
		}
	}

	path := buildProjectConsumptionPath(projectConsumptionResourcesSub, period, service)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get project consumption resources request: %w", err)
	}

	var resp GetProjectConsumptionResourcesResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get project consumption resources: %w", err)
	}

	if resp.Consumption == nil {
		return nil, fmt.Errorf("consumption not found in response: %w", ErrNotFound)
	}

	return resp.Consumption, nil
}

// GetProjectConsumptionDaily retrieves the project consumption over a period by
// day, every day of the period included. A nil period or empty dates take the
// API defaults.
func (c *CloudClient) GetProjectConsumptionDaily(ctx context.Context, period *entities.ConsumptionPeriodRequest) (*entities.ProjectConsumptionDaily, error) {
	if period != nil {
		if err := period.Validate(); err != nil {
			return nil, fmt.Errorf("invalid consumption period: %w", err)
		}
	}

	path := buildProjectConsumptionPath(projectConsumptionDailySub, period, "")
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get project consumption daily request: %w", err)
	}

	var resp GetProjectConsumptionDailyResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get project consumption daily: %w", err)
	}

	if resp.Consumption == nil {
		return nil, fmt.Errorf("consumption not found in response: %w", ErrNotFound)
	}

	return resp.Consumption, nil
}
