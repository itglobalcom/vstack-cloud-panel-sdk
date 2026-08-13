package sdk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// VMware service paths (relative to /api/v1/, which is prepended by buildURL).
// C-1: comments translated to English.
const (
	vmwareLocationsURL = "vmware/locations"
	vmwareImagesURL    = "vmware/images"
	vmwareGPUModelsURL = "vmware/gpu-models"
	// SDK-1: the dedicated vmwareTasksBaseURL constant was removed; VMware tasks
	// are served by the shared tasks resource, so tasksBaseURL from task.go is
	// reused (task ids are strings of the form "vmw{N}").
)

// VMware metainfo / task response wrappers.
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
	vmwareTaskResponse struct {
		Task *entities.VmwareTask `json:"task,omitempty"`
	}
)

// SDK-2: the withQuery helper is a general-purpose helper and now lives in
// client.go; its definition is removed here and callers use the package-level
// withQuery directly.

// GetVmwareLocationList returns the VMware locations catalog.
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
// SDK-5: locationID is a pointer because 0 is an ambiguous "unset" value.
// API-11: gpu is a three-state filter — VmwareImageGPURequired ("required", GPU-only images),
// VmwareImageGPUUnsupported ("unsupported", non-GPU images), or nil for no GPU filter.
func (c *CloudClient) GetVmwareImageList(ctx context.Context, locationID *int, gpu *string) ([]*entities.VmwareImage, error) {
	// SDK-4: validate the optional id before sending it.
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

// API-11 redesign: GetVmwareDiskTypeList and GetVmwareStorageProfileList were removed.
// Disk types now travel inside each VmwareLocation (VmwareLocation.DiskTypes), and storage
// profiles are an internal detail no longer exposed by the public API.

// GetVmwareGPUModelList returns the GPU models, optionally filtered by location.
//
// SDK-5: pointers intentionally kept for the optional int filter.
func (c *CloudClient) GetVmwareGPUModelList(ctx context.Context, locationID *int) ([]*entities.VmwareGPUModel, error) {
	// SDK-4: validate the optional id before sending it.
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

// GetVmwareTask returns the status of a VMware task by its id (a string of the
// form "vmw{N}").
//
// Warning (C-4): VMware task ids ("vmw{N}") live in a separate id space from
// base tasks and yield a different response shape. Never pass a VMware task id
// to the base GetTask, and never pass a base task id here.
func (c *CloudClient) GetVmwareTask(ctx context.Context, taskID string) (*entities.VmwareTask, error) {
	// SDK-4: GetVmwareTask already validates the empty taskID.
	if taskID == "" {
		return nil, fmt.Errorf("task ID is required")
	}
	// SDK-1: reuse tasksBaseURL from task.go instead of a duplicated constant.
	path := fmt.Sprintf("%s/%s", tasksBaseURL, taskID)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create vmware task request: %w", err)
	}
	var resp vmwareTaskResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get vmware task %s: %w", taskID, err)
	}
	if resp.Task == nil {
		return nil, fmt.Errorf("vmware task %s not found in response: %w", taskID, ErrNotFound)
	}
	return resp.Task, nil
}

// vmwareTaskWaitDefaultTimeout is the default wait applied by WaitVmwareTask.
// SDK-S7: VMware operations run far longer than the 2m base PollingTimeout
// default (server create/copy ~4m, rebuild >12m), so VMware task waiting has its
// own floor. The base PollingTimeout is left unchanged. WithPollingTimeout still
// overrides upward. Note: rebuild has been measured above 12m and may exceed even
// 15m — for such operations pass an explicit larger WithPollingTimeout, or call
// WaitVmwareTaskWithTimeout directly.
const vmwareTaskWaitDefaultTimeout = 15 * time.Minute

// WaitVmwareTask polls a VMware task until it reaches a terminal state.
//
// SDK-S7: the wait defaults to a VMware-specific floor (vmwareTaskWaitDefaultTimeout),
// not the 2m base PollingTimeout; an explicitly larger WithPollingTimeout wins.
//
// Warning (C-4): VMware task ids ("vmw{N}") must not be passed to the base
// WaitServerTaskCompletion, and base task ids must not be passed here — the id
// spaces and response shapes differ.
func (c *CloudClient) WaitVmwareTask(ctx context.Context, taskID string) (*entities.VmwareTask, error) {
	timeout := c.config.PollingTimeout
	if timeout < vmwareTaskWaitDefaultTimeout {
		timeout = vmwareTaskWaitDefaultTimeout
	}
	return c.WaitVmwareTaskWithTimeout(ctx, taskID, timeout)
}

// WaitVmwareTaskWithTimeout polls a VMware task until it reaches a terminal
// state within the given timeout. It returns an error if the task finishes in
// a failed or canceled state.
//
// Warning (C-4): VMware task ids ("vmw{N}") must not be passed to the base
// WaitServerTaskCompletion, and base task ids must not be passed here — the id
// spaces and response shapes differ.
func (c *CloudClient) WaitVmwareTaskWithTimeout(ctx context.Context, taskID string, timeout time.Duration) (*entities.VmwareTask, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID is required")
	}

	c.logger.Info("Starting to wait for vmware task %s (timeout: %v, interval: %v)",
		taskID, timeout, c.config.PollingInterval)

	pollingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(c.config.PollingInterval)
	defer ticker.Stop()

	attempt := 0
	startTime := time.Now()

	for {
		attempt++
		c.logger.Debug("Polling attempt %d for vmware task %s", attempt, taskID)

		task, err := c.GetVmwareTask(pollingCtx, taskID)
		if err != nil {
			// Check whether the context has expired.
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				elapsed := time.Since(startTime)
				c.logger.Error("Vmware task %s polling timeout after %v (%d attempts)",
					taskID, elapsed, attempt)
				return nil, fmt.Errorf("vmware task %s did not complete within %v: %w",
					taskID, elapsed, err)
			}

			c.logger.Error("Failed to get vmware task %s status (attempt %d): %v",
				taskID, attempt, err)
			return nil, fmt.Errorf("failed to get vmware task %s: %w", taskID, err)
		}

		c.logger.Debug("Vmware task %s state: %s (attempt %d, elapsed: %v)",
			taskID, task.State, attempt, time.Since(startTime))

		// SDK-3 + C-15: use the single IsTerminal() definition to leave the loop,
		// then classify the terminal state as completed or failed instead of
		// re-expanding the terminal states inline.
		if task.IsTerminal() {
			elapsed := time.Since(startTime)
			if task.IsCompleted() {
				c.logger.Info("Vmware task %s completed successfully after %v (%d attempts)",
					taskID, elapsed, attempt)
				return task, nil
			}
			// Terminal but not completed: failed or canceled.
			state := task.State
			if task.Error != nil && *task.Error != "" {
				state = fmt.Sprintf("%s (%s)", task.State, *task.Error)
			}
			c.logger.Error("Vmware task %s finished with state %s after %v (%d attempts)",
				taskID, state, elapsed, attempt)
			// SDK-3: return task = nil on failure, matching the base package.
			return nil, fmt.Errorf("vmware task %s finished with state %s", taskID, state)
		}

		// Task is still running, wait for the next iteration.
		c.logger.Debug("Vmware task %s is still in progress: state=%s", taskID, task.State)

		select {
		case <-pollingCtx.Done():
			elapsed := time.Since(startTime)
			c.logger.Error("Vmware task %s polling timeout after %v (%d attempts, last state: %s)",
				taskID, elapsed, attempt, task.State)
			return nil, fmt.Errorf("vmware task %s did not complete within %v (last state: %s): %w",
				taskID, elapsed, task.State, pollingCtx.Err())
		case <-ticker.C:
			continue
		}
	}
}
