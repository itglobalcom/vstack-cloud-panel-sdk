package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// The unified task model: state, type, progress, both timestamps and resources[]
// all decode, and the legacy per-resource field the service does fill is kept.
func TestParseTaskResponseUnifiedModel(t *testing.T) {
	body := []byte(`{"task":{
		"id":"l2t8814","type":"CreateServer","progress_percent":100,
		"is_completed":"Completed",
		"created":"2026-08-20T10:00:00Z","completed":"2026-08-20T10:03:21Z",
		"location_id":"ds-msk",
		"server_id":"l2s4412",
		"resources":[{"type":"server","id":"l2s4412"},{"type":"volume","id":"771"}]
	}}`)

	var wrap taskResponseWrap
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	task := wrap.Task
	if task == nil {
		t.Fatal("task must not be nil")
	}

	if task.ID != "l2t8814" || task.Type != "CreateServer" {
		t.Errorf("identity: %+v", task)
	}
	if task.ProgressPercent == nil || *task.ProgressPercent != 100 {
		t.Errorf("progress_percent = %v, want 100", task.ProgressPercent)
	}
	if task.Completed == nil || *task.Completed != "2026-08-20T10:03:21Z" {
		t.Errorf("completed = %v, want 2026-08-20T10:03:21Z", task.Completed)
	}
	if task.LocationID != "ds-msk" || task.ServerID != "l2s4412" {
		t.Errorf("scope/legacy fields: %+v", task)
	}
	if got := task.ResourceID(entities.TaskResourceVolume); got != "771" {
		t.Errorf("ResourceID(volume) = %q, want %q", got, "771")
	}
	if got := task.ResourceID(entities.TaskResourceNetwork); got != "" {
		t.Errorf("ResourceID(network) = %q, want %q", got, "")
	}
}

// A K8s cluster id arrives in "k8s_cluster_id"; the old "cluster_id" spelling
// never matched the contract and left the field empty.
func TestParseTaskResponseKubernetesClusterID(t *testing.T) {
	body := []byte(`{"task":{
		"id":"k8s_f12","type":"CreateCluster","progress_percent":40,
		"is_completed":"InProgress","created":"2026-08-20T10:00:00Z",
		"k8s_cluster_id":"305",
		"resources":[{"type":"cluster","id":"305"}]
	}}`)

	var wrap taskResponseWrap
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := wrap.Task.KubernetesClusterID; got != "305" {
		t.Errorf("KubernetesClusterID = %q, want %q", got, "305")
	}
	if got := wrap.Task.ResourceID(entities.TaskResourceCluster); got != "305" {
		t.Errorf("ResourceID(cluster) = %q, want %q", got, "305")
	}
}

// A ptr record has no legacy field at all: resources[] is the only place it
// appears, which is why callers must read it rather than the deprecated ids.
func TestParseTaskResponsePtrRecordOnlyInResources(t *testing.T) {
	body := []byte(`{"task":{
		"id":"dns42","type":"CreatePtrRecord","progress_percent":100,
		"is_completed":"Completed",
		"created":"2026-08-20T10:00:00Z","completed":"2026-08-20T10:00:13Z",
		"domain_id":"example.com",
		"resources":[{"type":"domain","id":"example.com"},{"type":"ptr","id":"7781"}]
	}}`)

	var wrap taskResponseWrap
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	task := wrap.Task
	if got := task.ResourceID(entities.TaskResourcePTR); got != "7781" {
		t.Errorf("ResourceID(ptr) = %q, want %q", got, "7781")
	}
	if task.DomainName != "example.com" {
		t.Errorf("DomainName = %q, want %q", task.DomainName, "example.com")
	}
	if got := task.RecordID; got != 0 {
		t.Errorf("RecordID = %d, want 0 (a ptr task carries no record id)", got)
	}
}

// A VMware task decodes through the base model too: it sends none of the legacy
// per-resource fields, so its server is readable only through resources[].
func TestParseTaskResponseVmwareTask(t *testing.T) {
	body := []byte(`{"task":{
		"id":"vmw901","type":"ServerCreate","progress_percent":100,
		"is_completed":"Completed",
		"created":"2026-08-20T10:00:00Z","completed":"2026-08-20T10:05:00Z",
		"resources":[{"type":"server","id":"48214"}]
	}}`)

	var wrap taskResponseWrap
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	task := wrap.Task
	if !task.IsSucceeded() {
		t.Errorf("IsSucceeded() = false for status %q", task.IsCompleted)
	}
	if task.ServerID != "" {
		t.Errorf("ServerID = %q, want empty (a VMware task has no legacy ids)", task.ServerID)
	}
	if got := task.ResourceID(entities.TaskResourceServer); got != "48214" {
		t.Errorf("ResourceID(server) = %q, want %q", got, "48214")
	}
}

// The API omits null fields, so an unfinished task arrives without "completed"
// and a task that touched nothing without "resources" — that must decode to zero
// values rather than fail.
func TestParseTaskResponseOmittedFields(t *testing.T) {
	body := []byte(`{"task":{"id":"l2t1","type":"DeleteServer","progress_percent":0,"is_completed":"New","created":"2026-08-20T10:00:00Z"}}`)

	var wrap taskResponseWrap
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if wrap.Task.Completed != nil {
		t.Errorf("Completed = %v, want nil", wrap.Task.Completed)
	}
	if wrap.Task.Resources != nil {
		t.Errorf("Resources = %v, want nil", wrap.Task.Resources)
	}
	if wrap.Task.IsTerminal() {
		t.Error("a New task must not be terminal")
	}
}

// is_completed is a five-member enum, not a boolean: Canceled is terminal but
// not a success, and a waiter that knows only Completed/Failed polls a canceled
// task until it times out.
func TestTaskStatePredicates(t *testing.T) {
	cases := map[string]struct {
		terminal  bool
		succeeded bool
		failed    bool
	}{
		entities.TaskStateNew:        {false, false, false},
		entities.TaskStateInProgress: {false, false, false},
		entities.TaskStateCompleted:  {true, true, false},
		entities.TaskStateFailed:     {true, false, true},
		entities.TaskStateCanceled:   {true, false, false},
	}

	for state, want := range cases {
		task := &entities.TaskResponse{IsCompleted: state}
		if got := task.IsTerminal(); got != want.terminal {
			t.Errorf("IsTerminal(%q) = %v, want %v", state, got, want.terminal)
		}
		if got := task.IsSucceeded(); got != want.succeeded {
			t.Errorf("IsSucceeded(%q) = %v, want %v", state, got, want.succeeded)
		}
		if got := task.IsFailed(); got != want.failed {
			t.Errorf("IsFailed(%q) = %v, want %v", state, got, want.failed)
		}
		if got := entities.IsTaskStateTerminal(state); got != want.terminal {
			t.Errorf("IsTaskStateTerminal(%q) = %v, want %v", state, got, want.terminal)
		}
	}
}

// The wire values are pinned: the API compares them case-sensitively. The
// published VmwareTaskState*/VmwareTaskResource* names are defined as these
// constants, so pinning them here pins both spellings.
func TestTaskStateWireValues(t *testing.T) {
	cases := map[string]string{
		entities.TaskStateNew:         "New",
		entities.TaskStateInProgress:  "InProgress",
		entities.TaskStateCompleted:   "Completed",
		entities.TaskStateFailed:      "Failed",
		entities.TaskStateCanceled:    "Canceled",
		entities.TaskResourceServer:   "server",
		entities.TaskResourceNetwork:  "network",
		entities.TaskResourceVolume:   "volume",
		entities.TaskResourceSnapshot: "snapshot",
		entities.TaskResourceNIC:      "nic",
		entities.TaskResourceGateway:  "gateway",
		entities.TaskResourceDomain:   "domain",
		entities.TaskResourceRecord:   "record",
		entities.TaskResourcePTR:      "ptr",
		entities.TaskResourceCluster:  "cluster",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("wire value = %q, want %q", got, want)
		}
	}
}

func TestIsAlreadyCompletedTaskID(t *testing.T) {
	cases := map[string]bool{
		AlreadyCompletedTaskID: true,
		"already_completed":    false,
		"l2t8814":              false,
		"":                     false,
	}
	for id, want := range cases {
		if got := IsAlreadyCompletedTaskID(id); got != want {
			t.Errorf("IsAlreadyCompletedTaskID(%q) = %v, want %v", id, got, want)
		}
	}
}

// The synthetic task is completed by definition: the API answers Completed for
// it unconditionally, so the wait must produce that answer without a request.
func TestWaitTaskCompletionSyntheticTask(t *testing.T) {
	client := newTestClient(t, noRequestHandler(t))

	task, err := client.waitTaskCompletion(context.Background(), AlreadyCompletedTaskID)
	if err != nil {
		t.Fatalf("waiting for the synthetic task must succeed, got %v", err)
	}
	if task.ID != AlreadyCompletedTaskID || !task.IsSucceeded() {
		t.Errorf("synthetic task = %+v, want id %q in state %q",
			task, AlreadyCompletedTaskID, entities.TaskStateCompleted)
	}
}

// A VMware task is readable through GetTask but must not be awaited here: the
// base timeouts are minutes and VMware tasks run for tens of them.
func TestWaitTaskCompletionRejectsForeignAndEmptyID(t *testing.T) {
	client := newTestClient(t, noRequestHandler(t))
	ctx := context.Background()

	if _, err := client.waitTaskCompletion(ctx, "vmw901"); err == nil {
		t.Error("waitTaskCompletion must reject a VMware task ID")
	}
	if _, err := client.waitTaskCompletion(ctx, ""); err == nil {
		t.Error("waitTaskCompletion must reject an empty task ID")
	}
	if _, err := client.GetTask(ctx, ""); err == nil {
		t.Error("GetTask must reject an empty task ID")
	}
}

// The wait leaves the loop on the first terminal state and reports failure for
// every terminal state that is not a success. Canceled is the case a waiter that
// knows only Completed/Failed polls until it times out.
func TestWaitTaskCompletionTerminalStates(t *testing.T) {
	for _, state := range []string{entities.TaskStateFailed, entities.TaskStateCanceled} {
		t.Run(state, func(t *testing.T) {
			var reads int32
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&reads, 1)
				writeTaskState(w, "l2t8814", state)
			}))

			_, err := client.waitTaskCompletion(context.Background(), "l2t8814")
			if err == nil {
				t.Fatalf("a %s task must be reported as an error", state)
			}
			if !errors.Is(err, ErrTaskFailed) {
				t.Errorf("error must wrap ErrTaskFailed, got %v", err)
			}
			if !strings.Contains(err.Error(), state) {
				t.Errorf("error must name the state %q, got %v", state, err)
			}
			if got := atomic.LoadInt32(&reads); got != 1 {
				t.Errorf("a terminal state must be read once, got %d reads", got)
			}
		})
	}
}

// The happy path keeps polling through the non-terminal states and returns the
// completed task.
func TestWaitTaskCompletionPollsUntilCompleted(t *testing.T) {
	states := []string{entities.TaskStateNew, entities.TaskStateInProgress, entities.TaskStateCompleted}
	var reads int32

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := int(atomic.AddInt32(&reads, 1)) - 1
		if i >= len(states) {
			i = len(states) - 1
		}
		writeTaskState(w, "l2t8814", states[i])
	}))

	task, err := client.waitTaskCompletion(context.Background(), "l2t8814")
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if !task.IsSucceeded() {
		t.Errorf("task state = %q, want %q", task.IsCompleted, entities.TaskStateCompleted)
	}
	if got := atomic.LoadInt32(&reads); got != int32(len(states)) {
		t.Errorf("got %d reads, want %d", got, len(states))
	}
}

// writeTaskState answers a task read with the given state.
func writeTaskState(w http.ResponseWriter, taskID, state string) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"task":{"id":%q,"type":"CreateServer","progress_percent":50,"is_completed":%q,"created":"2026-08-20T10:00:00Z"}}`,
		taskID, state)
}

// noRequestHandler fails the test if the client sends anything: it guards the
// checks that must answer before a request is built.
func noRequestHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	})
}
