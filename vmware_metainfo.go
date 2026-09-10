package sdk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// VMware service paths (relative to /api/v1/, which is prepended by buildURL).
const (
	vmwareLocationsURL = "vmware/locations"
	vmwareImagesURL    = "vmware/images"
	vmwareGPUModelsURL = "vmware/gpu-models"
)

// VMware metainfo response wrappers.
type (
	vmwareLocationsResponse struct {
		Locations []*entities.VmwareLocation `json:"locations,omitempty"`
	}
	vmwareImagesResponse struct {
		Images []*entities.VmwareImage `json:"images,omitempty"`
	}
	vmwareGPUModelsResponse struct {
		GPUModels []*entities.VmwareGPUModel `json:"gpu_models,omitempty"`
	}
)

// GetVmwareLocationList returns the VMware locations catalog.
//
// Each location carries its own disk_types (VmwareLocation.DiskTypes): the
// standalone /vmware/disk-types and /vmware/storage-profiles catalogs exist only
// under the AdminV2 prefix, so this is the only source of the disk-type names and
// size limits that the create/verify requests accept.
func (c *CloudClient) GetVmwareLocationList(ctx context.Context) ([]*entities.VmwareLocation, error) {
	req, err := c.newRequest(ctx, http.MethodGet, vmwareLocationsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create vmware locations request: %w", err)
	}
	var resp vmwareLocationsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list vmware locations: %w", err)
	}
	return resp.Locations, nil
}

// GetVmwareImageList lists images, optionally filtered by location and GPU support.
//
// LocationID is a pointer because 0 is an ambiguous "unset" value.
//
// Gpu is a three-state filter — VmwareImageGPURequired ("required",
// GPU-only images), VmwareImageGPUUnsupported ("unsupported", images that cannot
// use a GPU), or nil for no filter. Any other value is rejected by the API with
// HTTP 400.
//
// Both filters are omitted from the query string when unset rather than sent
// empty: the API answers an empty value of a declared query parameter with HTTP
// 500 and an empty body, so the SDK never emits one.
func (c *CloudClient) GetVmwareImageList(ctx context.Context, locationID *int, gpu *string) ([]*entities.VmwareImage, error) {
	if locationID != nil && *locationID <= 0 {
		return nil, fmt.Errorf("location ID must be positive")
	}
	params := url.Values{}
	if locationID != nil {
		params.Set("location_id", strconv.Itoa(*locationID))
	}
	if gpu != nil && *gpu != "" {
		params.Set("gpu", *gpu)
	}
	req, err := c.newRequest(ctx, http.MethodGet, withQuery(vmwareImagesURL, params), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create vmware images request: %w", err)
	}
	var resp vmwareImagesResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list vmware images: %w", err)
	}
	return resp.Images, nil
}

// This family has no disk-type or storage-profile method: disk types travel inside
// each VmwareLocation (VmwareLocation.DiskTypes), and storage profiles are an
// internal detail the public API does not expose.

// GetVmwareGPUModelList returns the GPU slicing profiles, optionally filtered by
// location.
//
// Pointers intentionally kept for the optional int filter.
//
// The returned ID is NOT unique — see VmwareGPUModel.
func (c *CloudClient) GetVmwareGPUModelList(ctx context.Context, locationID *int) ([]*entities.VmwareGPUModel, error) {
	if locationID != nil && *locationID <= 0 {
		return nil, fmt.Errorf("location ID must be positive")
	}
	params := url.Values{}
	if locationID != nil {
		params.Set("location_id", strconv.Itoa(*locationID))
	}
	req, err := c.newRequest(ctx, http.MethodGet, withQuery(vmwareGPUModelsURL, params), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create vmware gpu models request: %w", err)
	}
	var resp vmwareGPUModelsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list vmware gpu models: %w", err)
	}
	return resp.GPUModels, nil
}
