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

// GetTask retrieves a specific task by ID
func (c *CloudClient) GetTask(ctx context.Context, taskID string) (*entities.TaskResponse, error) {
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
			return nil, fmt.Errorf("task %s failed with status: %s", taskID, task.IsCompleted)

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

// WaitServerTaskCompletion waits for task completion and then for server to become Active
func (c *CloudClient) WaitServerTaskCompletion(ctx context.Context, serverID string, taskID string) (*entities.Server, error) {
	// First wait for task to complete
	if _, err := c.waitTaskCompletion(ctx, taskID); err != nil {
		return nil, fmt.Errorf("task %s failed: %w", taskID, err)
	}

	// Then wait for server to become Active
	return c.WaitServerActive(ctx, serverID)
}
