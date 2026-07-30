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

// Пути услуги VMware (относительно /api/v1/, который добавляет buildURL).
const (
	vmwareLocationsURL       = "vmware/locations"
	vmwareImagesURL          = "vmware/images"
	vmwareDiskTypesURL       = "vmware/disk-types"
	vmwareStorageProfilesURL = "vmware/storage-profiles"
	vmwareGpuModelsURL       = "vmware/gpu-models"
	// Задачи vmware обслуживаются общим ресурсом задач: /api/v1/tasks/{task_id},
	// где task_id — строка вида vmw{N}.
	vmwareTasksBaseURL = "tasks"
)

// Ответы метаинформации / задач VMware.
type (
	vmwareLocationsResponse struct {
		Locations []*entities.VmwareLocation `json:"locations,omitempty"`
	}
	vmwareImagesResponse struct {
		Images []*entities.VmwareImage `json:"images,omitempty"`
	}
	vmwareDiskTypesResponse struct {
		DiskTypes []*entities.VmwareDiskType `json:"disk_types,omitempty"`
	}
	vmwareStorageProfilesResponse struct {
		StorageProfiles []*entities.VmwareStorageProfile `json:"storage_profiles,omitempty"`
	}
	vmwareGpuModelsResponse struct {
		GpuModels []*entities.VmwareGpuModel `json:"gpu_models,omitempty"`
	}
	vmwareTaskResponse struct {
		Task *entities.VmwareTask `json:"task,omitempty"`
	}
)

// withQuery добавляет query-параметры к пути (buildURL сохраняет строку запроса как есть).
func withQuery(path string, params url.Values) string {
	if len(params) == 0 {
		return path
	}
	return path + "?" + params.Encode()
}

// GetVmwareLocationList возвращает справочник локаций VMware.
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

// GetVmwareImageList возвращает шаблоны/образы (опционально по локации / только GPU).
func (c *CloudClient) GetVmwareImageList(ctx context.Context, locationID *int, gpuOnly *bool) ([]*entities.VmwareImage, error) {
	params := url.Values{}
	if locationID != nil {
		params.Set("location_id", strconv.Itoa(*locationID))
	}
	if gpuOnly != nil {
		params.Set("gpu_only", strconv.FormatBool(*gpuOnly))
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

// GetVmwareDiskTypeList возвращает типы дисков (опционально по локации).
func (c *CloudClient) GetVmwareDiskTypeList(ctx context.Context, locationID *int) ([]*entities.VmwareDiskType, error) {
	params := url.Values{}
	if locationID != nil {
		params.Set("location_id", strconv.Itoa(*locationID))
	}
	req, err := c.newRequest(ctx, http.MethodGet, withQuery(vmwareDiskTypesURL, params), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create vmware disk types request: %w", err)
	}
	var resp vmwareDiskTypesResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list vmware disk types: %w", err)
	}
	return resp.DiskTypes, nil
}

// GetVmwareStorageProfileList возвращает storage-профили (опционально по локации / типу диска).
func (c *CloudClient) GetVmwareStorageProfileList(ctx context.Context, locationID *int, diskTypeID *int) ([]*entities.VmwareStorageProfile, error) {
	params := url.Values{}
	if locationID != nil {
		params.Set("location_id", strconv.Itoa(*locationID))
	}
	if diskTypeID != nil {
		params.Set("disk_type_id", strconv.Itoa(*diskTypeID))
	}
	req, err := c.newRequest(ctx, http.MethodGet, withQuery(vmwareStorageProfilesURL, params), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create vmware storage profiles request: %w", err)
	}
	var resp vmwareStorageProfilesResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list vmware storage profiles: %w", err)
	}
	return resp.StorageProfiles, nil
}

// GetVmwareGpuModelList возвращает модели GPU (опционально по локации).
func (c *CloudClient) GetVmwareGpuModelList(ctx context.Context, locationID *int) ([]*entities.VmwareGpuModel, error) {
	params := url.Values{}
	if locationID != nil {
		params.Set("location_id", strconv.Itoa(*locationID))
	}
	req, err := c.newRequest(ctx, http.MethodGet, withQuery(vmwareGpuModelsURL, params), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create vmware gpu models request: %w", err)
	}
	var resp vmwareGpuModelsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list vmware gpu models: %w", err)
	}
	return resp.GpuModels, nil
}

// GetVmwareTask возвращает статус задачи VMware по её id (строка вида vmw{N}).
func (c *CloudClient) GetVmwareTask(ctx context.Context, taskID string) (*entities.VmwareTask, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID is required")
	}
	path := fmt.Sprintf("%s/%s", vmwareTasksBaseURL, taskID)
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

// WaitVmwareTask опрашивает задачу VMware до достижения терминального состояния
// (используя PollingTimeout/PollingInterval из конфигурации клиента).
func (c *CloudClient) WaitVmwareTask(ctx context.Context, taskID string) (*entities.VmwareTask, error) {
	return c.WaitVmwareTaskWithTimeout(ctx, taskID, c.config.PollingTimeout)
}

// WaitVmwareTaskWithTimeout опрашивает задачу VMware до терминального состояния
// с указанным таймаутом. Возвращает ошибку, если задача завершилась failed/canceled.
func (c *CloudClient) WaitVmwareTaskWithTimeout(ctx context.Context, taskID string, timeout time.Duration) (*entities.VmwareTask, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID is required")
	}

	pollingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(c.config.PollingInterval)
	defer ticker.Stop()

	for {
		task, err := c.GetVmwareTask(pollingCtx, taskID)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return nil, fmt.Errorf("vmware task %s did not complete within %v: %w", taskID, timeout, err)
			}
			return nil, fmt.Errorf("failed to get vmware task %s: %w", taskID, err)
		}

		if task.IsFailed() || task.State == entities.VmwareTaskStateCanceled {
			msg := task.State
			if task.Error != nil && *task.Error != "" {
				msg = fmt.Sprintf("%s (%s)", task.State, *task.Error)
			}
			return task, fmt.Errorf("vmware task %s finished with state %s", taskID, msg)
		}
		if task.IsCompleted() {
			return task, nil
		}

		select {
		case <-pollingCtx.Done():
			return nil, fmt.Errorf("vmware task %s did not complete within %v (last state: %s): %w",
				taskID, timeout, task.State, pollingCtx.Err())
		case <-ticker.C:
			continue
		}
	}
}
