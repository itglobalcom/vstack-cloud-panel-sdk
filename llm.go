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
	llmBaseURL         = "llm"
	llmConnectionPath  = "connection"
	llmGroupsPath      = "groups"
	llmModelsSubPath   = "models"
	llmTagsSubPath     = "tags"
	llmDisableSubPath  = "disable"
	llmAccessKeysPath  = "access-keys"
	llmSecretSubPath   = "secret"
	llmLockSubPath     = "lock"
	llmUnlockSubPath   = "unlock"
	llmModelsPath      = "models"
	llmPackagePath     = "package"
	llmPackagePlansURL = "package-plans"
	llmUsagePath       = "usage"
	llmRequestsPath    = "requests"
)

// Response types for LLM operations
type (
	// GetLLMConnectionResponse represents the LLM connection response
	GetLLMConnectionResponse struct {
		Connection *entities.LLMConnection `json:"connection,omitempty"`
	}

	// ListLLMGroupsResponse represents the LLM groups list response
	ListLLMGroupsResponse struct {
		Groups []entities.LLMGroup `json:"groups,omitempty"`
	}

	// GetLLMGroupResponse represents a single LLM group response
	GetLLMGroupResponse struct {
		Group *entities.LLMGroupDetails `json:"group,omitempty"`
	}

	// ListLLMAccessKeysResponse represents the LLM access keys list response
	ListLLMAccessKeysResponse struct {
		AccessKeys []entities.LLMAccessKey `json:"access_keys,omitempty"`
	}

	// GetLLMAccessKeyResponse represents a single LLM access key response
	GetLLMAccessKeyResponse struct {
		AccessKey *entities.LLMAccessKey `json:"access_key,omitempty"`
	}

	// RevealLLMAccessKeySecretResponse represents the access key secret response
	RevealLLMAccessKeySecretResponse struct {
		Secret string `json:"secret,omitempty"`
	}

	// GetLLMModelResponse represents a single LLM model response
	GetLLMModelResponse struct {
		Model *entities.LLMModelDetails `json:"model,omitempty"`
	}

	// GetLLMPackageResponse represents the current LLM package response
	GetLLMPackageResponse struct {
		Package *entities.LLMCurrentPackage `json:"package,omitempty"`
	}

	// ListLLMPackagePlansResponse represents the LLM package plans list response
	ListLLMPackagePlansResponse struct {
		PackagePlans []entities.LLMPackagePlan `json:"package_plans,omitempty"`
	}

	// GetLLMUsageResponse represents the LLM usage response
	GetLLMUsageResponse struct {
		Usage *entities.LLMUsage `json:"usage,omitempty"`
	}
)

// buildLLMPath constructs llm[/<part>…]; empty parts are skipped.
func buildLLMPath(parts ...string) string {
	path := llmBaseURL
	for _, part := range parts {
		if part != "" {
			path = fmt.Sprintf("%s/%s", path, part)
		}
	}
	return path
}

// buildLLMGroupPath constructs llm/groups/{groupID}[/<sub>].
func buildLLMGroupPath(groupID entities.LLMGroupID, sub string) string {
	return buildLLMPath(llmGroupsPath, strconv.Itoa(int(groupID)), sub)
}

// buildLLMAccessKeyPath constructs llm/groups/{groupID}/access-keys[/{accessKeyID}][/<sub>];
// a zero accessKeyID addresses the collection of the group.
func buildLLMAccessKeyPath(groupID entities.LLMGroupID, accessKeyID entities.LLMAccessKeyID, sub string) string {
	key := ""
	if accessKeyID != 0 {
		key = strconv.Itoa(int(accessKeyID))
	}
	return buildLLMPath(llmGroupsPath, strconv.Itoa(int(groupID)), llmAccessKeysPath, key, sub)
}

// buildLLMAccessKeyListPath constructs llm/access-keys with the optional group filter.
func buildLLMAccessKeyListPath(groupID entities.LLMGroupID) string {
	path := buildLLMPath(llmAccessKeysPath)
	if groupID != 0 {
		params := url.Values{}
		params.Set("group_id", strconv.Itoa(int(groupID)))
		path = fmt.Sprintf("%s?%s", path, params.Encode())
	}
	return path
}

// buildLLMUsagePath constructs llm/usage with the period, the breakdown and the
// filters as query; an empty breakdown and zero filters are left out.
func buildLLMUsagePath(r *entities.LLMUsageRequest) string {
	params := url.Values{}
	params.Set("from", r.From)
	params.Set("to", r.To)
	if r.GroupBy != "" {
		params.Set("group_by", string(r.GroupBy))
	}
	if r.GroupID != 0 {
		params.Set("group_id", strconv.Itoa(int(r.GroupID)))
	}
	if r.AccessKeyID != 0 {
		params.Set("access_key_id", strconv.Itoa(int(r.AccessKeyID)))
	}
	if r.ModelID != 0 {
		params.Set("model_id", strconv.Itoa(int(r.ModelID)))
	}
	return fmt.Sprintf("%s?%s", buildLLMPath(llmUsagePath), params.Encode())
}

// validateLLMAccessKeyRef checks the pair addressing an access key.
func validateLLMAccessKeyRef(groupID entities.LLMGroupID, accessKeyID entities.LLMAccessKeyID) error {
	if err := groupID.Validate(); err != nil {
		return err
	}
	return accessKeyID.Validate()
}

// llmTaskResourceID reads the decimal id of the resource of the given type the
// completed task reports.
func llmTaskResourceID(task *entities.TaskResponse, resourceType string) (int, error) {
	raw := task.ResourceID(resourceType)
	if raw == "" {
		return 0, fmt.Errorf("task %s reports no %s resource: %w", task.ID, resourceType, ErrNotFound)
	}
	id, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("task %s reports %s resource id %q, not a decimal integer: %w", task.ID, resourceType, raw, err)
	}
	return id, nil
}

// llmTask sends a request answered with a task reference.
func (c *CloudClient) llmTask(ctx context.Context, method, path string, body any, action string) (*TaskID, error) {
	req, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s request: %w", action, err)
	}

	var task TaskID
	if err := c.doJSON(req, &task); err != nil {
		return nil, fmt.Errorf("failed to %s: %w", action, err)
	}

	return &task, nil
}

// GetLLMConnection retrieves the LLM service connection of the project: not
// connected, connecting (with the connection task) or connected.
func (c *CloudClient) GetLLMConnection(ctx context.Context) (*entities.LLMConnection, error) {
	req, err := c.newRequest(ctx, http.MethodGet, buildLLMPath(llmConnectionPath), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get LLM connection request: %w", err)
	}

	var resp GetLLMConnectionResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get LLM connection: %w", err)
	}

	if resp.Connection == nil {
		return nil, fmt.Errorf("LLM connection not found in response: %w", ErrNotFound)
	}

	return resp.Connection, nil
}

// ConnectLLM connects the LLM service to the project with its first group and
// first access key, and returns the connection task. A project already
// connected is refused with the API code ServiceAlreadyConnected.
func (c *CloudClient) ConnectLLM(ctx context.Context, req *entities.ConnectLLMRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("connect LLM request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid connect LLM request: %w", err)
	}

	return c.llmTask(ctx, http.MethodPost, buildLLMPath(llmConnectionPath), req, "connect LLM")
}

// ConnectLLMAndWait connects the LLM service, waits for the connection task and
// returns the connection.
func (c *CloudClient) ConnectLLMAndWait(ctx context.Context, req *entities.ConnectLLMRequest) (*entities.LLMConnection, error) {
	task, err := c.ConnectLLM(ctx, req)
	if err != nil {
		return nil, err
	}

	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for LLM connection: %w", err)
	}

	return c.GetLLMConnection(ctx)
}

// GetLLMGroupList retrieves the LLM groups of the project. A group being
// created comes without ID and State and with its TaskID.
func (c *CloudClient) GetLLMGroupList(ctx context.Context) ([]entities.LLMGroup, error) {
	req, err := c.newRequest(ctx, http.MethodGet, buildLLMPath(llmGroupsPath), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list LLM groups request: %w", err)
	}

	var resp ListLLMGroupsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list LLM groups: %w", err)
	}

	return resp.Groups, nil
}

// GetLLMGroup retrieves an LLM group with its deployment state and spend.
func (c *CloudClient) GetLLMGroup(ctx context.Context, groupID entities.LLMGroupID) (*entities.LLMGroupDetails, error) {
	if err := groupID.Validate(); err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodGet, buildLLMGroupPath(groupID, ""), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for LLM group %d: %w", groupID, err)
	}

	var resp GetLLMGroupResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get LLM group %d: %w", groupID, err)
	}

	if resp.Group == nil {
		return nil, fmt.Errorf("LLM group %d not found in response: %w", groupID, ErrNotFound)
	}

	return resp.Group, nil
}

// CreateLLMGroup creates an LLM group and returns the task; the group ID is the
// TaskResourceLLMGroup resource of the completed task. A name already taken,
// including by a disabled group, is refused with the API code GroupAlreadyExists.
func (c *CloudClient) CreateLLMGroup(ctx context.Context, req *entities.CreateLLMGroupRequest) (*TaskID, error) {
	if req == nil {
		return nil, fmt.Errorf("create LLM group request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create LLM group request: %w", err)
	}

	return c.llmTask(ctx, http.MethodPost, buildLLMPath(llmGroupsPath), req, "create LLM group")
}

// CreateLLMGroupAndWait creates an LLM group, waits for the task and returns the group.
func (c *CloudClient) CreateLLMGroupAndWait(ctx context.Context, req *entities.CreateLLMGroupRequest) (*entities.LLMGroupDetails, error) {
	task, err := c.CreateLLMGroup(ctx, req)
	if err != nil {
		return nil, err
	}

	done, err := c.waitTaskCompletion(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for LLM group creation: %w", err)
	}

	id, err := llmTaskResourceID(done, entities.TaskResourceLLMGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to identify the created LLM group: %w", err)
	}

	return c.GetLLMGroup(ctx, entities.LLMGroupID(id))
}

// SetLLMGroupModels replaces the whole set of the group models and returns the task.
func (c *CloudClient) SetLLMGroupModels(ctx context.Context, groupID entities.LLMGroupID, req *entities.SetLLMGroupModelsRequest) (*TaskID, error) {
	if err := groupID.Validate(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("set LLM group models request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid set LLM group models request: %w", err)
	}

	return c.llmTask(ctx, http.MethodPut, buildLLMGroupPath(groupID, llmModelsSubPath), req,
		fmt.Sprintf("set models of LLM group %d", groupID))
}

// SetLLMGroupModelsAndWait replaces the group models, waits for the task and
// returns the group.
func (c *CloudClient) SetLLMGroupModelsAndWait(ctx context.Context, groupID entities.LLMGroupID, req *entities.SetLLMGroupModelsRequest) (*entities.LLMGroupDetails, error) {
	task, err := c.SetLLMGroupModels(ctx, groupID, req)
	if err != nil {
		return nil, err
	}

	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for models of LLM group %d: %w", groupID, err)
	}

	return c.GetLLMGroup(ctx, groupID)
}

// SetLLMGroupTags replaces the whole tag set of the group. The change is
// synchronous: no task is started.
func (c *CloudClient) SetLLMGroupTags(ctx context.Context, groupID entities.LLMGroupID, req *entities.SetLLMTagsRequest) error {
	if err := groupID.Validate(); err != nil {
		return err
	}
	if req == nil {
		return fmt.Errorf("set LLM group tags request is required")
	}
	if err := req.Validate(); err != nil {
		return fmt.Errorf("invalid set LLM group tags request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodPut, buildLLMGroupPath(groupID, llmTagsSubPath), req)
	if err != nil {
		return fmt.Errorf("failed to create set tags request for LLM group %d: %w", groupID, err)
	}

	if err := c.doJSON(httpReq, nil); err != nil {
		return fmt.Errorf("failed to set tags of LLM group %d: %w", groupID, err)
	}

	return nil
}

// DisableLLMGroup disables the group and its access keys and returns the task.
// It cannot be undone: the group stays listed with the disabled state.
func (c *CloudClient) DisableLLMGroup(ctx context.Context, groupID entities.LLMGroupID) (*TaskID, error) {
	if err := groupID.Validate(); err != nil {
		return nil, err
	}

	return c.llmTask(ctx, http.MethodPost, buildLLMGroupPath(groupID, llmDisableSubPath), nil,
		fmt.Sprintf("disable LLM group %d", groupID))
}

// DisableLLMGroupAndWait disables the group, waits for the task and returns the group.
func (c *CloudClient) DisableLLMGroupAndWait(ctx context.Context, groupID entities.LLMGroupID) (*entities.LLMGroupDetails, error) {
	task, err := c.DisableLLMGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}

	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for LLM group %d to be disabled: %w", groupID, err)
	}

	return c.GetLLMGroup(ctx, groupID)
}

// GetLLMAccessKeyList retrieves the access keys of the project; a non-zero
// groupID keeps the keys of that group only. A key being issued comes without
// ID and with its TaskID.
func (c *CloudClient) GetLLMAccessKeyList(ctx context.Context, groupID entities.LLMGroupID) ([]entities.LLMAccessKey, error) {
	if groupID != 0 {
		if err := groupID.Validate(); err != nil {
			return nil, err
		}
	}

	req, err := c.newRequest(ctx, http.MethodGet, buildLLMAccessKeyListPath(groupID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list LLM access keys request: %w", err)
	}

	var resp ListLLMAccessKeysResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list LLM access keys: %w", err)
	}

	return resp.AccessKeys, nil
}

// GetLLMAccessKey retrieves an access key of the group.
func (c *CloudClient) GetLLMAccessKey(ctx context.Context, groupID entities.LLMGroupID, accessKeyID entities.LLMAccessKeyID) (*entities.LLMAccessKey, error) {
	if err := validateLLMAccessKeyRef(groupID, accessKeyID); err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodGet, buildLLMAccessKeyPath(groupID, accessKeyID, ""), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for LLM access key %d: %w", accessKeyID, err)
	}

	var resp GetLLMAccessKeyResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get LLM access key %d: %w", accessKeyID, err)
	}

	if resp.AccessKey == nil {
		return nil, fmt.Errorf("LLM access key %d not found in response: %w", accessKeyID, ErrNotFound)
	}

	return resp.AccessKey, nil
}

// IssueLLMAccessKey issues an access key in the group and returns the task; the
// key ID is the TaskResourceLLMAccessKey resource of the completed task.
func (c *CloudClient) IssueLLMAccessKey(ctx context.Context, groupID entities.LLMGroupID, req *entities.IssueLLMAccessKeyRequest) (*TaskID, error) {
	if err := groupID.Validate(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("issue LLM access key request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid issue LLM access key request: %w", err)
	}

	return c.llmTask(ctx, http.MethodPost, buildLLMAccessKeyPath(groupID, 0, ""), req,
		fmt.Sprintf("issue access key in LLM group %d", groupID))
}

// IssueLLMAccessKeyAndWait issues an access key, waits for the task and returns the key.
func (c *CloudClient) IssueLLMAccessKeyAndWait(ctx context.Context, groupID entities.LLMGroupID, req *entities.IssueLLMAccessKeyRequest) (*entities.LLMAccessKey, error) {
	task, err := c.IssueLLMAccessKey(ctx, groupID, req)
	if err != nil {
		return nil, err
	}

	done, err := c.waitTaskCompletion(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for LLM access key issue: %w", err)
	}

	id, err := llmTaskResourceID(done, entities.TaskResourceLLMAccessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to identify the issued LLM access key: %w", err)
	}

	return c.GetLLMAccessKey(ctx, groupID, entities.LLMAccessKeyID(id))
}

// RevealLLMAccessKeySecret returns the secret of the access key. The API hands
// it out once: a second call is refused with the API code
// AccessKeySecretAlreadyRevealed, and a key migrated without its secret with
// MigratedAccessKeySecretIsNotStored. A request retried after a lost response
// gets the first of these refusals.
func (c *CloudClient) RevealLLMAccessKeySecret(ctx context.Context, groupID entities.LLMGroupID, accessKeyID entities.LLMAccessKeyID) (string, error) {
	if err := validateLLMAccessKeyRef(groupID, accessKeyID); err != nil {
		return "", err
	}

	req, err := c.newRequest(ctx, http.MethodPost, buildLLMAccessKeyPath(groupID, accessKeyID, llmSecretSubPath), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create reveal secret request for LLM access key %d: %w", accessKeyID, err)
	}

	var resp RevealLLMAccessKeySecretResponse
	if err := c.doJSON(req, &resp); err != nil {
		return "", fmt.Errorf("failed to reveal secret of LLM access key %d: %w", accessKeyID, err)
	}

	if resp.Secret == "" {
		return "", fmt.Errorf("secret of LLM access key %d not found in response: %w", accessKeyID, ErrNotFound)
	}

	return resp.Secret, nil
}

// LockLLMAccessKey locks the access key by the client and returns the task; req
// carries the optional reason and may be nil.
func (c *CloudClient) LockLLMAccessKey(ctx context.Context, groupID entities.LLMGroupID, accessKeyID entities.LLMAccessKeyID, req *entities.LockLLMAccessKeyRequest) (*TaskID, error) {
	if err := validateLLMAccessKeyRef(groupID, accessKeyID); err != nil {
		return nil, err
	}

	var body any
	if req != nil {
		body = req
	}

	return c.llmTask(ctx, http.MethodPost, buildLLMAccessKeyPath(groupID, accessKeyID, llmLockSubPath), body,
		fmt.Sprintf("lock LLM access key %d", accessKeyID))
}

// LockLLMAccessKeyAndWait locks the access key, waits for the task and returns the key.
func (c *CloudClient) LockLLMAccessKeyAndWait(ctx context.Context, groupID entities.LLMGroupID, accessKeyID entities.LLMAccessKeyID, req *entities.LockLLMAccessKeyRequest) (*entities.LLMAccessKey, error) {
	task, err := c.LockLLMAccessKey(ctx, groupID, accessKeyID, req)
	if err != nil {
		return nil, err
	}

	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for LLM access key %d to be locked: %w", accessKeyID, err)
	}

	return c.GetLLMAccessKey(ctx, groupID, accessKeyID)
}

// UnlockLLMAccessKey removes the client lock of the access key and returns the task.
func (c *CloudClient) UnlockLLMAccessKey(ctx context.Context, groupID entities.LLMGroupID, accessKeyID entities.LLMAccessKeyID) (*TaskID, error) {
	if err := validateLLMAccessKeyRef(groupID, accessKeyID); err != nil {
		return nil, err
	}

	return c.llmTask(ctx, http.MethodPost, buildLLMAccessKeyPath(groupID, accessKeyID, llmUnlockSubPath), nil,
		fmt.Sprintf("unlock LLM access key %d", accessKeyID))
}

// UnlockLLMAccessKeyAndWait unlocks the access key, waits for the task and returns the key.
func (c *CloudClient) UnlockLLMAccessKeyAndWait(ctx context.Context, groupID entities.LLMGroupID, accessKeyID entities.LLMAccessKeyID) (*entities.LLMAccessKey, error) {
	task, err := c.UnlockLLMAccessKey(ctx, groupID, accessKeyID)
	if err != nil {
		return nil, err
	}

	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("failed to wait for LLM access key %d to be unlocked: %w", accessKeyID, err)
	}

	return c.GetLLMAccessKey(ctx, groupID, accessKeyID)
}

// DeleteLLMAccessKey deletes the access key and returns the task; the API
// always answers with a task reference.
func (c *CloudClient) DeleteLLMAccessKey(ctx context.Context, groupID entities.LLMGroupID, accessKeyID entities.LLMAccessKeyID) (*TaskID, error) {
	if err := validateLLMAccessKeyRef(groupID, accessKeyID); err != nil {
		return nil, err
	}

	return c.llmTask(ctx, http.MethodDelete, buildLLMAccessKeyPath(groupID, accessKeyID, ""), nil,
		fmt.Sprintf("delete LLM access key %d", accessKeyID))
}

// DeleteLLMAccessKeyAndWait deletes the access key and waits for the task.
func (c *CloudClient) DeleteLLMAccessKeyAndWait(ctx context.Context, groupID entities.LLMGroupID, accessKeyID entities.LLMAccessKeyID) error {
	task, err := c.DeleteLLMAccessKey(ctx, groupID, accessKeyID)
	if err != nil {
		return err
	}

	if _, err := c.waitTaskCompletion(ctx, task.ID); err != nil {
		return fmt.Errorf("failed to wait for LLM access key %d deletion: %w", accessKeyID, err)
	}

	return nil
}

// SetLLMAccessKeyTags replaces the whole tag set of the access key. The change
// is synchronous: no task is started.
func (c *CloudClient) SetLLMAccessKeyTags(ctx context.Context, groupID entities.LLMGroupID, accessKeyID entities.LLMAccessKeyID, req *entities.SetLLMTagsRequest) error {
	if err := validateLLMAccessKeyRef(groupID, accessKeyID); err != nil {
		return err
	}
	if req == nil {
		return fmt.Errorf("set LLM access key tags request is required")
	}
	if err := req.Validate(); err != nil {
		return fmt.Errorf("invalid set LLM access key tags request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodPut, buildLLMAccessKeyPath(groupID, accessKeyID, llmTagsSubPath), req)
	if err != nil {
		return fmt.Errorf("failed to create set tags request for LLM access key %d: %w", accessKeyID, err)
	}

	if err := c.doJSON(httpReq, nil); err != nil {
		return fmt.Errorf("failed to set tags of LLM access key %d: %w", accessKeyID, err)
	}

	return nil
}

// GetLLMModelList retrieves the model catalog of the project with the credit
// price and the time the prices were collected.
func (c *CloudClient) GetLLMModelList(ctx context.Context) (*entities.LLMModelCatalog, error) {
	req, err := c.newRequest(ctx, http.MethodGet, buildLLMPath(llmModelsPath), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list LLM models request: %w", err)
	}

	var resp entities.LLMModelCatalog
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list LLM models: %w", err)
	}

	return &resp, nil
}

// GetLLMModel retrieves a catalog model with its prices, providers and capabilities.
func (c *CloudClient) GetLLMModel(ctx context.Context, modelID entities.LLMModelID) (*entities.LLMModelDetails, error) {
	if err := modelID.Validate(); err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodGet, buildLLMPath(llmModelsPath, strconv.Itoa(int(modelID))), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for LLM model %d: %w", modelID, err)
	}

	var resp GetLLMModelResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get LLM model %d: %w", modelID, err)
	}

	if resp.Model == nil {
		return nil, fmt.Errorf("LLM model %d not found in response: %w", modelID, ErrNotFound)
	}

	return resp.Model, nil
}

// GetLLMPackage retrieves the package state of the project: the current
// package, the spend of the open period, the credit price and the stop by the
// package.
func (c *CloudClient) GetLLMPackage(ctx context.Context) (*entities.LLMCurrentPackage, error) {
	req, err := c.newRequest(ctx, http.MethodGet, buildLLMPath(llmPackagePath), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get LLM package request: %w", err)
	}

	var resp GetLLMPackageResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get LLM package: %w", err)
	}

	if resp.Package == nil {
		return nil, fmt.Errorf("LLM package not found in response: %w", ErrNotFound)
	}

	return resp.Package, nil
}

// GetLLMPackagePlanList retrieves the package plans available to the project.
func (c *CloudClient) GetLLMPackagePlanList(ctx context.Context) ([]entities.LLMPackagePlan, error) {
	req, err := c.newRequest(ctx, http.MethodGet, buildLLMPath(llmPackagePlansURL), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list LLM package plans request: %w", err)
	}

	var resp ListLLMPackagePlansResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list LLM package plans: %w", err)
	}

	return resp.PackagePlans, nil
}

// GetLLMUsage retrieves the usage of the project over the period of req, with
// the rows of its breakdown and the state of the usage collection.
func (c *CloudClient) GetLLMUsage(ctx context.Context, req *entities.LLMUsageRequest) (*entities.LLMUsage, error) {
	if req == nil {
		return nil, fmt.Errorf("LLM usage request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid LLM usage request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodGet, buildLLMUsagePath(req), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get LLM usage request: %w", err)
	}

	var resp GetLLMUsageResponse
	if err := c.doJSON(httpReq, &resp); err != nil {
		return nil, fmt.Errorf("failed to get LLM usage: %w", err)
	}

	if resp.Usage == nil {
		return nil, fmt.Errorf("LLM usage not found in response: %w", ErrNotFound)
	}

	return resp.Usage, nil
}

// GetLLMRequestList retrieves the request log of the project access keys over
// the last 7 days, up to 200 rows.
func (c *CloudClient) GetLLMRequestList(ctx context.Context) (*entities.LLMRequestLog, error) {
	req, err := c.newRequest(ctx, http.MethodGet, buildLLMPath(llmRequestsPath), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list LLM requests request: %w", err)
	}

	var resp entities.LLMRequestLog
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list LLM requests: %w", err)
	}

	return &resp, nil
}
