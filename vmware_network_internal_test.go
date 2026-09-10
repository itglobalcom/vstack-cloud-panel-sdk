package sdk

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

func TestBuildVmwareEdgePath(t *testing.T) {
	cases := map[string]string{
		buildVmwareEdgePath(77, "bandwidth"):  "vmware/networks/77/edge/bandwidth",
		buildVmwareEdgePath(77, "nat", "3"):   "vmware/networks/77/edge/nat/3",
		buildVmwareNetworkPath(77, "servers"): "vmware/networks/77/servers",
		buildVmwareNetworkPath(77):            "vmware/networks/77",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
	}
}

// The edge bandwidth endpoint takes {"bandwidth_mbps": N} on PUT and always
// answers with a task — there is no synchronous no-op case.
func TestUpdateVmwareEdgeBandwidth(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]int
	)

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"task_id":"vmw123"}`))
	}))

	task, err := client.UpdateVmwareEdgeBandwidth(context.Background(), 77,
		&entities.VmwareUpdateEdgeBandwidthRequest{BandwidthMbps: 30})
	if err != nil {
		t.Fatalf("UpdateVmwareEdgeBandwidth: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPut)
	}
	if want := "/api/v1/vmware/networks/77/edge/bandwidth"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	if gotBody["bandwidth_mbps"] != 30 {
		t.Errorf("body[bandwidth_mbps] = %d, want 30", gotBody["bandwidth_mbps"])
	}
	if task == nil || task.ID != "vmw123" {
		t.Errorf("task = %v, want vmw123", task)
	}
}

func TestUpdateVmwareEdgeBandwidthRejectsBadArguments(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("%s %s reached the API, want rejection before the request", r.Method, r.URL.Path)
	}))

	cases := map[string]struct {
		networkID int
		req       *entities.VmwareUpdateEdgeBandwidthRequest
	}{
		"network id":  {0, &entities.VmwareUpdateEdgeBandwidthRequest{BandwidthMbps: 30}},
		"nil request": {77, nil},
		"bandwidth":   {77, &entities.VmwareUpdateEdgeBandwidthRequest{BandwidthMbps: 0}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := client.UpdateVmwareEdgeBandwidth(context.Background(), tc.networkID, tc.req); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

// edge_external_ip is the address a DNAT rule needs as original_ip; the network
// read is the only operation of the contract that publishes it, and a network
// without an edge omits it.
func TestParseVmwareNetworkEdgeExternalIP(t *testing.T) {
	var routed vmwareNetworkResponse
	if err := json.Unmarshal([]byte(
		`{"network":{"id":77,"type":"routed_client","name":"app","edge_external_ip":"203.0.113.10","bandwidth_mbps":30,"state":"active"}}`,
	), &routed); err != nil {
		t.Fatalf("unmarshal routed: %v", err)
	}
	if routed.Network.EdgeExternalIP == nil {
		t.Fatal("EdgeExternalIP = nil, want the published address")
	}
	if got := *routed.Network.EdgeExternalIP; got != "203.0.113.10" {
		t.Errorf("EdgeExternalIP = %q, want %q", got, "203.0.113.10")
	}

	var isolated vmwareNetworkResponse
	if err := json.Unmarshal([]byte(
		`{"network":{"id":78,"type":"private_client","name":"db","state":"active"}}`,
	), &isolated); err != nil {
		t.Fatalf("unmarshal isolated: %v", err)
	}
	if isolated.Network.EdgeExternalIP != nil {
		t.Errorf("EdgeExternalIP = %q for a network without an edge, want nil", *isolated.Network.EdgeExternalIP)
	}
}
