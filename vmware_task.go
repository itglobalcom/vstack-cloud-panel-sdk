package sdk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// VmwareTaskIDPrefix is the prefix every VMware task id carries ("vmw{N}").
// The backend routes GET /tasks/{id} by this prefix, and it is what lets the SDK
// keep the two task families apart.
const VmwareTaskIDPrefix = "vmw"

// VmwareTaskID references a VMware background task.
//
// This is deliberately a distinct type from TaskID even though both decode
// the same {"task_id": …} body. The two families differ in how they are awaited
// and in how their resource ids are shaped, so passing one family's reference to
// the other family's helper is a bug; keeping the types apart makes it a compile
// error at every call site that hands a task reference around.
type VmwareTaskID struct {
	ID string `json:"task_id,omitempty"`
}

// String returns the raw task id, so a VmwareTaskID can be logged or formatted
// directly.
func (t *VmwareTaskID) String() string {
	if t == nil {
		return ""
	}
	return t.ID
}

// IsZero reports whether the reference carries no task. VMware endpoints answer
// some no-op mutations synchronously, without starting a task; the SDK reports
// that as a nil *VmwareTaskID, and this helper makes the check safe on a nil
// receiver.
func (t *VmwareTaskID) IsZero() bool {
	return t == nil || t.ID == ""
}

// IsVmwareTaskID reports whether taskID belongs to the VMware id space
// ("vmw{N}") rather than to the base one ("l{N}t{N}", "dns{N}", "k8s_{f|m}{N}").
func IsVmwareTaskID(taskID string) bool {
	return strings.HasPrefix(taskID, VmwareTaskIDPrefix)
}

// VMware metainfo / task response wrapper.
type vmwareTaskResponse struct {
	Task *entities.VmwareTask `json:"task,omitempty"`
}

// GetVmwareTask returns the status of a VMware task by its id (a string of the
// form "vmw{N}").
//
// A base task id is rejected before the request is sent. Its body would decode
// into entities.VmwareTask — the unified task model is the same for every
// service — but VmwareTask.ServerID and NetworkID read a resource id as a decimal
// integer, which is how VMware and only VMware addresses a resource: a vStack
// task's encoded ids would come back as "not present". Use GetTask for base task
// ids; it decodes any family.
func (c *CloudClient) GetVmwareTask(ctx context.Context, taskID string) (*entities.VmwareTask, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID is required")
	}
	// Fail loudly on a base task id instead of handing back a model whose
	// resource accessors cannot read it.
	if !IsVmwareTaskID(taskID) {
		return nil, fmt.Errorf("task ID %q is not a VMware task ID (expected the %q prefix); use GetTask for base tasks", taskID, VmwareTaskIDPrefix)
	}
	// Reuse tasksBaseURL from task.go instead of a duplicated constant.
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

// VmwareTaskWaitDefaultTimeout is the default wait applied by WaitVmwareTask.
//
// VMware operations run far longer than the 2m base PollingTimeout, and
// their duration varies wildly for identical work: the same
// server order took 4m30s in one run and 21m30s in another, and two rebuilds of the
// same server took 4m40s and 24m30s (earlier measurements reached ~26m). VMware task
// waiting therefore has its own floor, and the base PollingTimeout is left alone.
//
// This is a FLOOR, not a value: WithPollingTimeout raises the wait but cannot lower
// it below this. To wait for less, call WaitVmwareTaskWithTimeout.
//
// The observed maxima sit only 4-5 minutes below this floor, so raise it with
// WithPollingTimeout for rebuild, or wherever a timeout would be costly.
const VmwareTaskWaitDefaultTimeout = 30 * time.Minute

// WaitVmwareTask polls a VMware task until it reaches a terminal state.
//
// The wait is at least VmwareTaskWaitDefaultTimeout; a larger
// WithPollingTimeout wins, a smaller one is ignored (use
// WaitVmwareTaskWithTimeout to wait for less).
//
// A base task id is rejected — see GetVmwareTask.
func (c *CloudClient) WaitVmwareTask(ctx context.Context, taskID string) (*entities.VmwareTask, error) {
	timeout := c.config.PollingTimeout
	if timeout < VmwareTaskWaitDefaultTimeout {
		// Say so rather than silently ignoring the configured value: a caller who set
		// a smaller PollingTimeout and then sees a longer wait should be able to find
		// out why from the log, not by reading the source.
		c.logger.Info("Configured PollingTimeout %v is below the VMware wait floor %v; waiting %v for task %s (use WaitVmwareTaskWithTimeout to wait less)",
			timeout, VmwareTaskWaitDefaultTimeout, VmwareTaskWaitDefaultTimeout, taskID)
		timeout = VmwareTaskWaitDefaultTimeout
	}
	return c.WaitVmwareTaskWithTimeout(ctx, taskID, timeout)
}

// WaitVmwareTaskWithTimeout polls a VMware task until it reaches a terminal
// state within the given timeout. It returns an error if the task finishes in
// a failed or canceled state.
//
// Note that a completed task does NOT guarantee the affected resource has
// settled: on rebuild the task reports completed while the replaced server is
// still "deleting". Follow up with WaitVmwareServerState / WaitVmwareServerGone
// (or the ...AndWait method) when the resource state matters.
//
// A base task id is rejected — see GetVmwareTask.
func (c *CloudClient) WaitVmwareTaskWithTimeout(ctx context.Context, taskID string, timeout time.Duration) (*entities.VmwareTask, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID is required")
	}
	// Reject a base task id here too, for the same reason as GetVmwareTask: the
	// returned model would misread its resource ids.
	if !IsVmwareTaskID(taskID) {
		return nil, fmt.Errorf("task ID %q is not a VMware task ID (expected the %q prefix); use WaitServerTaskCompletion for base tasks", taskID, VmwareTaskIDPrefix)
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

		// Use the single IsTerminal() definition to leave the loop,
		// then classify the terminal state as completed or failed instead of
		// re-expanding the terminal states inline.
		if task.IsTerminal() {
			elapsed := time.Since(startTime)
			if task.IsCompleted() {
				c.logger.Info("Vmware task %s completed successfully after %v (%d attempts)",
					taskID, elapsed, attempt)
				return task, nil
			}
			// Terminal but not completed: failed or canceled. The unified Task
			// model carries no error text, so the state is all we can report.
			c.logger.Error("Vmware task %s finished with state %s after %v (%d attempts)",
				taskID, task.State, elapsed, attempt)
			// Return task = nil on failure, matching the base package.
			return nil, fmt.Errorf("vmware task %s finished with state %s", taskID, task.State)
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

// WaitVmwareTaskRef waits for the task a VMware mutator returned and reports the
// completed task. A nil or empty reference means the API answered synchronously
// and there is nothing to await, in which case it returns (nil, nil).
//
// This is the building block behind every VMware "...AndWait" method; it is
// exported so callers that keep the raw task reference can await it without
// re-implementing the nil / no-op check.
func (c *CloudClient) WaitVmwareTaskRef(ctx context.Context, task *VmwareTaskID) (*entities.VmwareTask, error) {
	if task.IsZero() {
		return nil, nil
	}
	return c.WaitVmwareTask(ctx, task.ID)
}

// awaitVmwareTask is the error-only form of WaitVmwareTaskRef used by the
// ...AndWait methods that have no entity to return.
func (c *CloudClient) awaitVmwareTask(ctx context.Context, task *VmwareTaskID) error {
	_, err := c.WaitVmwareTaskRef(ctx, task)
	return err
}

// vmwarePollTicker sets up the polling context and ticker shared by the
// resource-state waiters below, using the same interval as the task waiter and
// the VMware wait floor.
func (c *CloudClient) vmwarePollTicker(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc, *time.Ticker) {
	pollingCtx, cancel := context.WithTimeout(ctx, timeout)
	return pollingCtx, cancel, time.NewTicker(c.config.PollingInterval)
}

// vmwareResourceWaitTimeout returns the timeout used by the resource-state
// waiters: the VMware floor, or a larger configured PollingTimeout.
func (c *CloudClient) vmwareResourceWaitTimeout() time.Duration {
	if c.config.PollingTimeout > VmwareTaskWaitDefaultTimeout {
		return c.config.PollingTimeout
	}
	return VmwareTaskWaitDefaultTimeout
}

// WaitVmwareServerState polls a server until its state is one of wantStates and
// returns it.
//
// A completed task is not the same as a settled resource: after a rebuild the
// task reports completed while the replaced server is still "deleting", and the
// server only reaches its final state minutes later. Terraform-style callers
// that need the resource itself to be ready should wait on the state, not just
// on the task.
//
// The wait fails fast if the server enters VmwareServerStateError, and it fails
// if the server disappears (use WaitVmwareServerGone to wait for deletion).
func (c *CloudClient) WaitVmwareServerState(ctx context.Context, serverID int, wantStates ...string) (*entities.VmwareServer, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server ID must be greater than 0")
	}
	if len(wantStates) == 0 {
		return nil, fmt.Errorf("at least one target state is required")
	}

	want := make(map[string]bool, len(wantStates))
	for _, s := range wantStates {
		want[s] = true
	}

	timeout := c.vmwareResourceWaitTimeout()
	c.logger.Info("Waiting for vmware server %d to reach state %v (timeout: %v, interval: %v)",
		serverID, wantStates, timeout, c.config.PollingInterval)

	pollingCtx, cancel, ticker := c.vmwarePollTicker(ctx, timeout)
	defer cancel()
	defer ticker.Stop()

	attempt := 0
	startTime := time.Now()

	for {
		attempt++
		server, err := c.GetVmwareServer(pollingCtx, serverID)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return nil, fmt.Errorf("vmware server %d did not reach state %v within %v: %w",
					serverID, wantStates, time.Since(startTime), err)
			}
			return nil, fmt.Errorf("failed to get vmware server %d: %w", serverID, err)
		}

		c.logger.Debug("Vmware server %d state: %s (attempt %d, elapsed: %v)",
			serverID, server.State, attempt, time.Since(startTime))

		if want[server.State] {
			c.logger.Info("Vmware server %d reached state %s after %v (%d attempts)",
				serverID, server.State, time.Since(startTime), attempt)
			return server, nil
		}
		// Fail fast on the terminal error state instead of polling to the timeout,
		// unless the caller is explicitly waiting for it.
		if server.State == entities.VmwareServerStateError {
			return nil, fmt.Errorf("vmware server %d is in state %s", serverID, server.State)
		}

		select {
		case <-pollingCtx.Done():
			return nil, fmt.Errorf("vmware server %d did not reach state %v within %v (last state: %s): %w",
				serverID, wantStates, time.Since(startTime), server.State, pollingCtx.Err())
		case <-ticker.C:
			continue
		}
	}
}

// WaitVmwareServerActive polls a server until it reports
// VmwareServerStateActive.
func (c *CloudClient) WaitVmwareServerActive(ctx context.Context, serverID int) (*entities.VmwareServer, error) {
	return c.WaitVmwareServerState(ctx, serverID, entities.VmwareServerStateActive)
}

// WaitVmwareServerGone polls a server until it no longer exists, and returns
// immediately if it is already gone.
//
// A completed task is not a reliable "the object is gone" signal. After a rebuild
// the replaced server has been observed sitting in state "deleting" for minutes
// past the completion of its task; a plain delete usually has it gone by then.
// Since the difference is not something a caller can predict, waiting on the
// object rather than on the task makes the outcome unconditional — at the cost of
// one extra GET when there was nothing to wait for.
func (c *CloudClient) WaitVmwareServerGone(ctx context.Context, serverID int) error {
	if serverID <= 0 {
		return fmt.Errorf("server ID must be greater than 0")
	}

	timeout := c.vmwareResourceWaitTimeout()
	c.logger.Info("Waiting for vmware server %d to disappear (timeout: %v, interval: %v)",
		serverID, timeout, c.config.PollingInterval)

	pollingCtx, cancel, ticker := c.vmwarePollTicker(ctx, timeout)
	defer cancel()
	defer ticker.Stop()

	attempt := 0
	startTime := time.Now()
	lastState := ""

	for {
		attempt++
		server, err := c.GetVmwareServer(pollingCtx, serverID)
		if err != nil {
			if IsNotFound(err) {
				c.logger.Info("Vmware server %d is gone after %v (%d attempts)",
					serverID, time.Since(startTime), attempt)
				return nil
			}
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return fmt.Errorf("vmware server %d was still present after %v: %w",
					serverID, time.Since(startTime), err)
			}
			return fmt.Errorf("failed to get vmware server %d: %w", serverID, err)
		}
		lastState = server.State

		c.logger.Debug("Vmware server %d still present in state %s (attempt %d, elapsed: %v)",
			serverID, lastState, attempt, time.Since(startTime))

		select {
		case <-pollingCtx.Done():
			return fmt.Errorf("vmware server %d was still present after %v (last state: %s): %w",
				serverID, time.Since(startTime), lastState, pollingCtx.Err())
		case <-ticker.C:
			continue
		}
	}
}

// WaitVmwareNetworkState polls a network until its state is one of wantStates
// and returns it.
//
// There is no published enum for VmwareNetwork.State; the observed values are
// collected as VmwareNetworkState* constants.
func (c *CloudClient) WaitVmwareNetworkState(ctx context.Context, networkID int, wantStates ...string) (*entities.VmwareNetwork, error) {
	if networkID <= 0 {
		return nil, fmt.Errorf("network ID must be greater than 0")
	}
	if len(wantStates) == 0 {
		return nil, fmt.Errorf("at least one target state is required")
	}

	want := make(map[string]bool, len(wantStates))
	for _, s := range wantStates {
		want[s] = true
	}

	timeout := c.vmwareResourceWaitTimeout()
	c.logger.Info("Waiting for vmware network %d to reach state %v (timeout: %v, interval: %v)",
		networkID, wantStates, timeout, c.config.PollingInterval)

	pollingCtx, cancel, ticker := c.vmwarePollTicker(ctx, timeout)
	defer cancel()
	defer ticker.Stop()

	attempt := 0
	startTime := time.Now()

	for {
		attempt++
		network, err := c.GetVmwareNetwork(pollingCtx, networkID)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return nil, fmt.Errorf("vmware network %d did not reach state %v within %v: %w",
					networkID, wantStates, time.Since(startTime), err)
			}
			return nil, fmt.Errorf("failed to get vmware network %d: %w", networkID, err)
		}

		c.logger.Debug("Vmware network %d state: %s (attempt %d, elapsed: %v)",
			networkID, network.State, attempt, time.Since(startTime))

		if want[network.State] {
			c.logger.Info("Vmware network %d reached state %s after %v (%d attempts)",
				networkID, network.State, time.Since(startTime), attempt)
			return network, nil
		}

		select {
		case <-pollingCtx.Done():
			return nil, fmt.Errorf("vmware network %d did not reach state %v within %v (last state: %s): %w",
				networkID, wantStates, time.Since(startTime), network.State, pollingCtx.Err())
		case <-ticker.C:
			continue
		}
	}
}

// WaitVmwareNetworkActive polls a network until it reports
// VmwareNetworkStateActive.
func (c *CloudClient) WaitVmwareNetworkActive(ctx context.Context, networkID int) (*entities.VmwareNetwork, error) {
	return c.WaitVmwareNetworkState(ctx, networkID, entities.VmwareNetworkStateActive)
}

// WaitVmwareNetworkGone polls a network until it no longer exists.
func (c *CloudClient) WaitVmwareNetworkGone(ctx context.Context, networkID int) error {
	if networkID <= 0 {
		return fmt.Errorf("network ID must be greater than 0")
	}

	timeout := c.vmwareResourceWaitTimeout()
	c.logger.Info("Waiting for vmware network %d to disappear (timeout: %v, interval: %v)",
		networkID, timeout, c.config.PollingInterval)

	pollingCtx, cancel, ticker := c.vmwarePollTicker(ctx, timeout)
	defer cancel()
	defer ticker.Stop()

	attempt := 0
	startTime := time.Now()
	lastState := ""

	for {
		attempt++
		network, err := c.GetVmwareNetwork(pollingCtx, networkID)
		if err != nil {
			if IsNotFound(err) {
				c.logger.Info("Vmware network %d is gone after %v (%d attempts)",
					networkID, time.Since(startTime), attempt)
				return nil
			}
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return fmt.Errorf("vmware network %d was still present after %v: %w",
					networkID, time.Since(startTime), err)
			}
			return fmt.Errorf("failed to get vmware network %d: %w", networkID, err)
		}
		lastState = network.State

		c.logger.Debug("Vmware network %d still present in state %s (attempt %d, elapsed: %v)",
			networkID, lastState, attempt, time.Since(startTime))

		select {
		case <-pollingCtx.Done():
			return fmt.Errorf("vmware network %d was still present after %v (last state: %s): %w",
				networkID, time.Since(startTime), lastState, pollingCtx.Err())
		case <-ticker.C:
			continue
		}
	}
}
