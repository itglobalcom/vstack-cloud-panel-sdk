package sdk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// newTestClient points a client at a stub server and strips the retry waits, so
// tests stay fast and deterministic.
func newTestClient(t *testing.T, handler http.Handler) *CloudClient {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	config, err := NewConfig("test-key", server.URL,
		WithPollingInterval(5*time.Millisecond),
		WithPollingTimeout(2*time.Second),
		WithMaxRetries(0),
		WithRetryWaitMinMax(time.Millisecond, time.Millisecond),
		WithLogLevel(Error),
	)
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	return client
}

// A gateway stays Busy for a while after the task of an operation has already
// completed, so waiting for Active is what makes the next change safe.
func TestWaitGatewayActive(t *testing.T) {
	var reads int32

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := entities.GatewayStateBusy
		if atomic.AddInt32(&reads, 1) >= 3 {
			state = entities.GatewayStateActive
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"gateway":{"id":"l1e1","state":"` + state + `"}}`))
	}))

	gateway, err := client.WaitGatewayActive(context.Background(), "l1e1")
	if err != nil {
		t.Fatalf("WaitGatewayActive: %v", err)
	}
	if gateway.State != entities.GatewayStateActive {
		t.Errorf("state = %q, want Active", gateway.State)
	}
	if got := atomic.LoadInt32(&reads); got < 3 {
		t.Errorf("polled %d time(s), expected to keep polling while Busy", got)
	}
}

func TestWaitGatewayActiveBlocked(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"gateway":{"id":"l1e1","state":"Blocked"}}`))
	}))

	if _, err := client.WaitGatewayActive(context.Background(), "l1e1"); err == nil {
		t.Error("a Blocked gateway must be an error, not an endless wait")
	}
}

// Deleting a gateway that is already gone answers HTTP 500, not 404 — callers
// still have to be able to tell "already deleted" from a real failure.
func TestDeleteGatewayAlreadyGone(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))

	err := client.DeleteGateway(context.Background(), "l1e1")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !IsNotFound(err) {
		t.Errorf("IsNotFound = false for an already-deleted gateway: %v", err)
	}
}

// A genuine failure must stay a failure: the gateway is still there.
func TestDeleteGatewayServerError(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"gateway":{"id":"l1e1","state":"Active"}}`))
	}))

	err := client.DeleteGateway(context.Background(), "l1e1")
	if err == nil {
		t.Fatal("expected an error")
	}
	if IsNotFound(err) {
		t.Errorf("a 500 on a gateway that still exists must not look like not-found: %v", err)
	}
}

// The same quirk applies to a NIC that is already disconnected.
func TestDisconnectNetworkNICAlreadyGone(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"gateway":{"id":"l1e1","state":"Active","nics":[{"id":7,"network_id":"l1n1"}]}}`))
	}))

	err := client.DisconnectNetwork(context.Background(), "l1e1", 42)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !IsNotFound(err) {
		t.Errorf("IsNotFound = false for an already-disconnected NIC: %v", err)
	}

	// A NIC that is still attached means the 500 was real.
	if err := client.DisconnectNetwork(context.Background(), "l1e1", 7); err == nil {
		t.Error("expected an error")
	} else if IsNotFound(err) {
		t.Errorf("a 500 on a NIC that still exists must not look like not-found: %v", err)
	}
}

// A rule set replaces the whole list, so a transient task failure has to be
// retried rather than handed to the caller.
func TestUpdateRulesRetriesFailedTask(t *testing.T) {
	var tasks int32

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPut:
			n := atomic.AddInt32(&tasks, 1)
			_, _ = w.Write([]byte(`{"task_id":"t` + string(rune('0'+n)) + `"}`))

		case strings.HasPrefix(r.URL.Path, "/api/v1/tasks/"):
			// The first task fails, the second one completes.
			status := "Completed"
			if strings.HasSuffix(r.URL.Path, "t1") {
				status = "Failed"
			}
			_, _ = w.Write([]byte(`{"task":{"id":"t","is_completed":"` + status + `"}}`))

		default:
			_, _ = w.Write([]byte(`{"gateway":{"id":"l1e1","state":"Active"}}`))
		}
	}))

	err := client.UpdateFirewallRulesAndWait(context.Background(), "l1e1",
		&entities.UpdateFirewallRulesRequest{FirewallRules: []entities.FirewallRule{{
			Action: entities.FirewallActionAllow, Direction: entities.FirewallDirectionIn,
			Protocol: entities.ProtocolTCP, Source: "0.0.0.0/0", Destination: "10.0.0.0/24",
		}}})
	if err != nil {
		t.Fatalf("a transient task failure must be retried: %v", err)
	}
	if got := atomic.LoadInt32(&tasks); got != 2 {
		t.Errorf("the rule set was sent %d time(s), want 2 (one failure + one retry)", got)
	}
}

// A task that keeps failing must surface as a failed task, not as success.
func TestUpdateRulesGivesUpOnFailedTask(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut:
			_, _ = w.Write([]byte(`{"task_id":"t1"}`))
		case strings.HasPrefix(r.URL.Path, "/api/v1/tasks/"):
			_, _ = w.Write([]byte(`{"task":{"id":"t1","is_completed":"Failed"}}`))
		default:
			_, _ = w.Write([]byte(`{"gateway":{"id":"l1e1","state":"Active"}}`))
		}
	}))

	err := client.UpdateNATRulesAndWait(context.Background(), "l1e1",
		&entities.UpdateNATRulesRequest{NATRules: []entities.NATRule{{
			Type: entities.NATTypeSNAT, Protocol: entities.ProtocolIP,
			Source: "10.0.0.0/24", Destination: "0.0.0.0/0", Translated: "203.0.113.5",
		}}})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !IsTaskFailed(err) {
		t.Errorf("IsTaskFailed = false: %v", err)
	}
}

// The rule payload is validated before it reaches the API: the SDK's own
// constants are the contract, and a bad enum otherwise comes back as an
// unhelpful "the request body is not formatted".
func TestUpdateRulesValidatesPayload(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("request must not reach the API: %s %s", r.Method, r.URL.Path)
	}))

	_, err := client.UpdateNATRules(context.Background(), "l1e1", &entities.UpdateNATRulesRequest{
		NATRules: []entities.NATRule{{Type: "dnat", Protocol: "TCP",
			Source: "0.0.0.0/0", Destination: "1.2.3.4/32", Translated: "10.0.0.5"}},
	})
	if err == nil {
		t.Error("an invalid NAT rule type must be rejected before the request")
	}

	_, err = client.UpdateFirewallRules(context.Background(), "l1e1", &entities.UpdateFirewallRulesRequest{
		FirewallRules: []entities.FirewallRule{{Action: "Allow", Direction: "In", Protocol: "SCTP",
			Source: "0.0.0.0/0", Destination: "0.0.0.0/0"}},
	})
	if err == nil {
		t.Error("an invalid firewall protocol must be rejected before the request")
	}
}
