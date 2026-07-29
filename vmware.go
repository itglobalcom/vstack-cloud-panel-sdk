package sdk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const (
	vmwareBasePath = "vmware"

	vmwareLocationsPath       = "locations"
	vmwareImagesPath          = "images"
	vmwareDiskTypesPath       = "disk-types"
	vmwareStorageProfilesPath = "storage-profiles"
	vmwareGPUModelsPath       = "gpu-models"
)

// Response types for VMware catalog (lookup) operations
type (
	// ListVMwareLocationsResponse represents a VMware locations list response
	ListVMwareLocationsResponse struct {
		Locations []entities.VMwareLocation `json:"locations,omitempty"`
	}

	// ListVMwareDiskTypesResponse represents a VMware disk types list response
	ListVMwareDiskTypesResponse struct {
		DiskTypes []entities.VMwareDiskType `json:"disk_types,omitempty"`
	}

	// ListVMwareGPUModelsResponse represents a VMware GPU models list response
	ListVMwareGPUModelsResponse struct {
		GPUModels []entities.VMwareGPUModel `json:"gpu_models,omitempty"`
	}

	// ListVMwareStorageProfilesResponse represents a VMware storage profiles list response
	ListVMwareStorageProfilesResponse struct {
		StorageProfiles []entities.VMwareStorageProfile `json:"storage_profiles,omitempty"`
	}

	// ListVMwareImagesResponse represents a VMware images list response
	ListVMwareImagesResponse struct {
		Images []entities.VMwareImage `json:"images,omitempty"`
	}
)

// buildVMwarePath constructs the path for a VMware catalog resource, appending
// the query filters when any are set.
func buildVMwarePath(resource string, filters url.Values) string {
	path := fmt.Sprintf("%s/%s", vmwareBasePath, resource)
	if len(filters) == 0 {
		return path
	}
	return fmt.Sprintf("%s?%s", path, filters.Encode())
}

// addIDFilter adds a numeric filter to the query.
//
// Zero means "filter not specified" — a deliberate contract: the API treats an
// absent parameter as no filter, and 0 is not a member of DCLocationEnum, so it
// can never be a real location / disk type identifier.
//
// A negative value is a caller error and is reported as such instead of being
// dropped: silently omitting the filter would answer a broken id with the full
// unfiltered list, whereas the API answers an unknown (but positive)
// location_id with 400 / APICodeDCLocationDoesNotExist. The check runs before
// the request is built, so a broken id never reaches the API.
func addIDFilter(filters url.Values, name string, value int) error {
	if value < 0 {
		return fmt.Errorf("%s must be greater than 0", name)
	}
	if value > 0 {
		filters.Set(name, strconv.Itoa(value))
	}
	return nil
}

// GetVMwareLocations retrieves the VMware locations connected to the partner
// and available to the project
func (c *CloudClient) GetVMwareLocations(ctx context.Context) ([]entities.VMwareLocation, error) {
	path := buildVMwarePath(vmwareLocationsPath, nil)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list VMware locations request: %w", err)
	}

	var resp ListVMwareLocationsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list VMware locations: %w", err)
	}

	return resp.Locations, nil
}

// GetVMwareDiskTypes retrieves the VMware disk types allowed for the project.
// locationID is optional — pass 0 to list the disk types of every location.
func (c *CloudClient) GetVMwareDiskTypes(ctx context.Context, locationID int) ([]entities.VMwareDiskType, error) {
	filters := url.Values{}
	if err := addIDFilter(filters, "location_id", locationID); err != nil {
		return nil, err
	}

	path := buildVMwarePath(vmwareDiskTypesPath, filters)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list VMware disk types request: %w", err)
	}

	var resp ListVMwareDiskTypesResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list VMware disk types: %w", err)
	}

	return resp.DiskTypes, nil
}

// GetVMwareGPUModels retrieves the VMware GPU models available to the partner.
// locationID is optional — pass 0 to list the models of every location.
func (c *CloudClient) GetVMwareGPUModels(ctx context.Context, locationID int) ([]entities.VMwareGPUModel, error) {
	filters := url.Values{}
	if err := addIDFilter(filters, "location_id", locationID); err != nil {
		return nil, err
	}

	path := buildVMwarePath(vmwareGPUModelsPath, filters)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list VMware GPU models request: %w", err)
	}

	var resp ListVMwareGPUModelsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list VMware GPU models: %w", err)
	}

	return resp.GPUModels, nil
}

// GetVMwareStorageProfiles retrieves the active VMware storage profiles.
// Both filters are optional — pass 0 to skip a filter.
func (c *CloudClient) GetVMwareStorageProfiles(ctx context.Context, locationID, diskTypeID int) ([]entities.VMwareStorageProfile, error) {
	filters := url.Values{}
	if err := addIDFilter(filters, "location_id", locationID); err != nil {
		return nil, err
	}
	if err := addIDFilter(filters, "disk_type_id", diskTypeID); err != nil {
		return nil, err
	}

	path := buildVMwarePath(vmwareStorageProfilesPath, filters)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list VMware storage profiles request: %w", err)
	}

	var resp ListVMwareStorageProfilesResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list VMware storage profiles: %w", err)
	}

	return resp.StorageProfiles, nil
}

// GetVMwareImages retrieves the VMware OS images available to the project.
// locationID is optional — pass 0 to list the images of every location.
// gpuOnly restricts the result to GPU-only images; false (the API default)
// returns every image.
func (c *CloudClient) GetVMwareImages(ctx context.Context, locationID int, gpuOnly bool) ([]entities.VMwareImage, error) {
	filters := url.Values{}
	if err := addIDFilter(filters, "location_id", locationID); err != nil {
		return nil, err
	}
	// Only send gpu_only when it changes the API default (false), so that the
	// unfiltered call stays a plain GET without a query string.
	if gpuOnly {
		filters.Set("gpu_only", "true")
	}

	path := buildVMwarePath(vmwareImagesPath, filters)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list VMware images request: %w", err)
	}

	var resp ListVMwareImagesResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list VMware images: %w", err)
	}

	return resp.Images, nil
}
