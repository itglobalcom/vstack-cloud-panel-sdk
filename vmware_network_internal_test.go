package sdk

import (
	"encoding/json"
	"testing"
)

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
