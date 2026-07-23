package sdk

import (
	"context"
	"fmt"
	"net/http"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const (
	affinityGroupBaseURL = "affinity-groups"
)

// Response types
type (
	GetAffinityGroupResponse struct {
		AffinityGroup *entities.AffinityGroup `json:"affinity_group,omitempty"`
	}

	ListAffinityGroupsResponse struct {
		AffinityGroups []*entities.AffinityGroup `json:"affinity_groups,omitempty"`
	}
)

// buildAffinityGroupPath constructs the path for affinity group operations
func buildAffinityGroupPath(groupID string) string {
	if groupID == "" {
		return affinityGroupBaseURL
	}
	return fmt.Sprintf("%s/%s", affinityGroupBaseURL, groupID)
}

// GetAffinityGroup retrieves a specific affinity group by ID
func (c *CloudClient) GetAffinityGroup(ctx context.Context, groupID string) (*entities.AffinityGroup, error) {
	if groupID == "" {
		return nil, fmt.Errorf("affinity group ID is required")
	}

	path := buildAffinityGroupPath(groupID)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for affinity group %s: %w", groupID, err)
	}

	var resp GetAffinityGroupResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get affinity group %s: %w", groupID, err)
	}

	if resp.AffinityGroup == nil {
		return nil, fmt.Errorf("affinity group %s not found in response: %w", groupID, ErrNotFound)
	}

	return resp.AffinityGroup, nil
}

// GetAffinityGroupList retrieves all affinity groups
func (c *CloudClient) GetAffinityGroupList(ctx context.Context) ([]*entities.AffinityGroup, error) {
	req, err := c.newRequest(ctx, http.MethodGet, affinityGroupBaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list affinity groups request: %w", err)
	}

	var resp ListAffinityGroupsResponse
	if err := c.doJSON(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list affinity groups: %w", err)
	}

	return resp.AffinityGroups, nil
}

// CreateAffinityGroup creates a new affinity or anti-affinity group
func (c *CloudClient) CreateAffinityGroup(ctx context.Context, req *entities.CreateAffinityGroupRequest) (*entities.AffinityGroup, error) {
	if req == nil {
		return nil, fmt.Errorf("create affinity group request is required")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid create affinity group request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, affinityGroupBaseURL, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create affinity group request: %w", err)
	}

	var resp GetAffinityGroupResponse
	if err := c.doJSON(httpReq, &resp); err != nil {
		return nil, fmt.Errorf("failed to create affinity group: %w", err)
	}

	if resp.AffinityGroup == nil {
		return nil, fmt.Errorf("affinity group not found in response: %w", ErrNotFound)
	}

	return resp.AffinityGroup, nil
}

// DeleteAffinityGroup deletes an affinity group
func (c *CloudClient) DeleteAffinityGroup(ctx context.Context, groupID string) error {
	if groupID == "" {
		return fmt.Errorf("affinity group ID is required")
	}

	path := buildAffinityGroupPath(groupID)
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request for affinity group %s: %w", groupID, err)
	}

	if err := c.doJSON(req, nil); err != nil {
		return fmt.Errorf("failed to delete affinity group %s: %w", groupID, err)
	}

	return nil
}
