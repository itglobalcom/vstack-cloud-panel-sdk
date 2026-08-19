package sdk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const (
	tasksBaseURL = "tasks"
)

// Task completion statuses
const (
	taskStatusCompleted = "Completed"
	taskStatusFailed    = "Failed"
)

// TaskID wraps a task ID from API responses
type TaskID struct {
	ID string `json:"task_id,omitempty"`
}

// Response wrapper types
type taskResponseWrap struct {
	Task *entities.TaskResponse `json:"task,omitempty"`
}

// GetTask retrieves a specific task by ID.
//
// TaskID must be a base task ID (for example "l{N}t{N}", "dns{N}",
// "k8s_{f|m}{N}"). A VMware task ID (of the form "vmw{N}") is rejected before the
// request is sent — use GetVmwareTask for those. Both kinds share the "tasks/{id}"
// endpoint but answer with different bodies: a VMware task carrying a
// server_id/network_id has numeric fields where entities.TaskResponse expects
// strings, so decoding one here used to fail with an unmarshal error that looked
// like a broken API, and one without those fields decoded but left IsCompleted
// empty, hanging the wait on an already-finished task.
func (c *CloudClient) GetTask(ctx context.Context, taskID string) (*entities.TaskResponse, error) {
	// Reject a VMware task id up front. Without this the request succeeds and
	// the failure surfaces either as an unmarshal error (when the task carries a
	// numeric server_id/network_id) or as a silently empty IsCompleted, which then
	// hangs waitTaskCompletion on an already-finished task.
	if IsVmwareTaskID(taskID) {
		return nil, fmt.Errorf("task ID %q is a VMware task ID; use GetVmwareTask instead", taskID)
	}
	path := fmt.Sprintf("%s/%s", tasksBaseURL, taskID)

	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var wrap taskResponseWrap
	if err := c.doJSON(req, &wrap); err != nil {
		return nil, err
	}

	return wrap.Task, nil
}

// waitTaskCompletion waits for a task to complete with PollingTimeout
func (c *CloudClient) waitTaskCompletion(ctx context.Context, taskID string) (*entities.TaskResponse, error) {
	return c.waitTaskCompletionWithTimeout(ctx, taskID, c.config.PollingTimeout)
}

// waitTaskCompletionWithTimeout waits for a task to complete with custom timeout
func (c *CloudClient) waitTaskCompletionWithTimeout(ctx context.Context, taskID string, timeout time.Duration) (*entities.TaskResponse, error) {
	c.logger.Info("Starting to wait for task %s (timeout: %v, interval: %v)",
		taskID, timeout, c.config.PollingInterval)

	// Create context with polling timeout
	pollingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(c.config.PollingInterval)
	defer ticker.Stop()

	attempt := 0
	startTime := time.Now()

	for {
		attempt++
		c.logger.Debug("Polling attempt %d for task %s", attempt, taskID)

		task, err := c.GetTask(pollingCtx, taskID)
		if err != nil {
			// Check if context has expired
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				elapsed := time.Since(startTime)
				c.logger.Error("Task %s polling timeout after %v (%d attempts)",
					taskID, elapsed, attempt)
				return nil, fmt.Errorf("task %s did not complete within %v: %w",
					taskID, elapsed, err)
			}

			c.logger.Error("Failed to get task %s status (attempt %d): %v",
				taskID, attempt, err)
			return nil, fmt.Errorf("failed to get task %s: %w", taskID, err)
		}

		// Log current status
		c.logger.Debug("Task %s status: %s (attempt %d, elapsed: %v)",
			taskID, task.IsCompleted, attempt, time.Since(startTime))

		// Check completion
		switch task.IsCompleted {
		case taskStatusCompleted:
			elapsed := time.Since(startTime)
			c.logger.Info("Task %s completed successfully after %v (%d attempts)",
				taskID, elapsed, attempt)
			return task, nil

		case taskStatusFailed:
			elapsed := time.Since(startTime)
			c.logger.Error("Task %s failed after %v (%d attempts)",
				taskID, elapsed, attempt)
			return nil, fmt.Errorf("task %s failed with status %s: %w", taskID, task.IsCompleted, ErrTaskFailed)

		default:
			// Task is still running, wait for next iteration
			c.logger.Debug("Task %s is still in progress: status=%s", taskID, task.IsCompleted)
		}

		// Wait for next tick or timeout
		select {
		case <-pollingCtx.Done():
			// Context expired - polling timeout
			elapsed := time.Since(startTime)
			c.logger.Error("Task %s polling timeout after %v (%d attempts, last status: %s)",
				taskID, elapsed, attempt, task.IsCompleted)
			return nil, fmt.Errorf("task %s did not complete within %v (last status: %s): %w",
				taskID, elapsed, task.IsCompleted, pollingCtx.Err())

		case <-ticker.C:
			// Continue polling
			continue
		}
	}
}

// WaitServerActive waits for a server to transition to Active state
func (c *CloudClient) WaitServerActive(ctx context.Context, serverID string) (*entities.Server, error) {
	return c.WaitServerActiveWithTimeout(ctx, serverID, c.config.PollingTimeout)
}

// WaitServerActiveWithTimeout waits for a server to become Active with custom timeout
func (c *CloudClient) WaitServerActiveWithTimeout(ctx context.Context, serverID string, timeout time.Duration) (*entities.Server, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server ID is required")
	}

	c.logger.Info("Waiting for server %s to become Active (timeout: %v, interval: %v)",
		serverID, timeout, c.config.PollingInterval)

	// Create context with polling timeout
	pollingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(c.config.PollingInterval)
	defer ticker.Stop()

	attempt := 0
	startTime := time.Now()

	for {
		attempt++
		c.logger.Debug("Polling attempt %d for server %s state", attempt, serverID)

		server, err := c.GetServer(pollingCtx, serverID)
		if err != nil {
			// Check if context has expired
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				elapsed := time.Since(startTime)
				c.logger.Error("Server %s state polling timeout after %v (%d attempts)",
					serverID, elapsed, attempt)
				return nil, fmt.Errorf("server %s did not become Active within %v: %w",
					serverID, elapsed, err)
			}

			c.logger.Error("Failed to get server %s status (attempt %d): %v",
				serverID, attempt, err)
			return nil, fmt.Errorf("failed to get server %s: %w", serverID, err)
		}

		// Log current state
		c.logger.Debug("Server %s state: %s (attempt %d, elapsed: %v)",
			serverID, server.State, attempt, time.Since(startTime))

		// Check state
		switch server.State {
		case entities.ServerStateActive:
			elapsed := time.Since(startTime)
			c.logger.Info("Server %s is now Active after %v (%d attempts)",
				serverID, elapsed, attempt)
			return server, nil

		case entities.ServerStateBlocked:
			elapsed := time.Since(startTime)
			c.logger.Error("Server %s is Blocked after %v (%d attempts)",
				serverID, elapsed, attempt)
			return nil, fmt.Errorf("server %s is in Blocked state", serverID)

		default:
			// Server is in New or Busy state, continue waiting
			c.logger.Debug("Server %s is still in %s state", serverID, server.State)
		}

		// Wait for next tick or timeout
		select {
		case <-pollingCtx.Done():
			elapsed := time.Since(startTime)
			c.logger.Error("Server %s state polling timeout after %v (%d attempts, last state: %s)",
				serverID, elapsed, attempt, server.State)
			return nil, fmt.Errorf("server %s did not become Active within %v (last state: %s): %w",
				serverID, elapsed, server.State, pollingCtx.Err())

		case <-ticker.C:
			continue
		}
	}
}

// WaitServerTaskCompletion waits for task completion and then for server to become Active.
//
// TaskID must be a base task ID. It is polled through GetTask, which rejects a
// VMware task ID ("vmw{N}"); wait for VMware tasks with WaitVmwareTask instead. Note
// also that serverID is a base string server ID — VMware servers are keyed by int.
func (c *CloudClient) WaitServerTaskCompletion(ctx context.Context, serverID string, taskID string) (*entities.Server, error) {
	// First wait for task to complete
	if _, err := c.waitTaskCompletion(ctx, taskID); err != nil {
		return nil, fmt.Errorf("task %s failed: %w", taskID, err)
	}

	// Then wait for server to become Active
	return c.WaitServerActive(ctx, serverID)
}

// WaitGatewayActive waits for a gateway to transition to Active state.
//
// A gateway stays Busy for a while after the task of an operation has already
// completed, and a change issued in that window is either rejected outright
// (-19803, "a conflict occurred during the competitive change of the object")
// or accepted and then fails as a task. Waiting for Active is therefore part of
// completing a gateway operation, not an optional extra — every *AndWait method
// in gateway.go does it.
func (c *CloudClient) WaitGatewayActive(ctx context.Context, gatewayID string) (*entities.Gateway, error) {
	return c.WaitGatewayActiveWithTimeout(ctx, gatewayID, c.config.PollingTimeout)
}

// WaitGatewayActiveWithTimeout waits for a gateway to become Active with custom timeout
func (c *CloudClient) WaitGatewayActiveWithTimeout(ctx context.Context, gatewayID string, timeout time.Duration) (*entities.Gateway, error) {
	if gatewayID == "" {
		return nil, fmt.Errorf("gateway ID is required")
	}

	c.logger.Info("Waiting for gateway %s to become Active (timeout: %v, interval: %v)",
		gatewayID, timeout, c.config.PollingInterval)

	pollingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(c.config.PollingInterval)
	defer ticker.Stop()

	attempt := 0
	startTime := time.Now()

	for {
		attempt++
		c.logger.Debug("Polling attempt %d for gateway %s state", attempt, gatewayID)

		gateway, err := c.GetGateway(pollingCtx, gatewayID)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				elapsed := time.Since(startTime)
				c.logger.Error("Gateway %s state polling timeout after %v (%d attempts)",
					gatewayID, elapsed, attempt)
				return nil, fmt.Errorf("gateway %s did not become Active within %v: %w",
					gatewayID, elapsed, err)
			}

			c.logger.Error("Failed to get gateway %s status (attempt %d): %v", gatewayID, attempt, err)
			return nil, fmt.Errorf("failed to get gateway %s: %w", gatewayID, err)
		}

		c.logger.Debug("Gateway %s state: %s (attempt %d, elapsed: %v)",
			gatewayID, gateway.State, attempt, time.Since(startTime))

		switch gateway.State {
		case entities.GatewayStateActive:
			c.logger.Info("Gateway %s is now Active after %v (%d attempts)",
				gatewayID, time.Since(startTime), attempt)
			return gateway, nil

		case entities.GatewayStateBlocked:
			c.logger.Error("Gateway %s is Blocked after %v (%d attempts)",
				gatewayID, time.Since(startTime), attempt)
			return nil, fmt.Errorf("gateway %s is in Blocked state", gatewayID)

		default:
			// New or Busy — keep waiting.
			c.logger.Debug("Gateway %s is still in %s state", gatewayID, gateway.State)
		}

		select {
		case <-pollingCtx.Done():
			elapsed := time.Since(startTime)
			c.logger.Error("Gateway %s state polling timeout after %v (%d attempts, last state: %s)",
				gatewayID, elapsed, attempt, gateway.State)
			return nil, fmt.Errorf("gateway %s did not become Active within %v (last state: %s): %w",
				gatewayID, elapsed, gateway.State, pollingCtx.Err())

		case <-ticker.C:
			continue
		}
	}
}

// WaitGatewayTaskCompletion waits for task completion and then for the gateway
// to become Active again, so that the next change to the same gateway is not
// rejected as a competitive change.
func (c *CloudClient) WaitGatewayTaskCompletion(ctx context.Context, gatewayID string, taskID string) (*entities.Gateway, error) {
	if _, err := c.waitTaskCompletion(ctx, taskID); err != nil {
		return nil, fmt.Errorf("task %s failed: %w", taskID, err)
	}

	return c.WaitGatewayActive(ctx, gatewayID)
}
