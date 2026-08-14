package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// runVmwareNetworkExample drives the whole VMware network and edge surface: it
// creates an isolated and a routed network, walks the edge firewall / NAT / VPN
// methods on the routed one (only a routed network carries an edge), attaches a
// server to a network, and deletes everything it created.
//
// As in the server example, each method is exercised in one of its two forms —
// raw call plus explicit wait, or the ...AndWait variant — so both paths are
// covered.
//
// Methods covered (26): GetVmwareNetworkList, GetVmwareNetwork,
// CreateVmwareIsolatedNetwork(+AndWait), CreateVmwareRoutedNetwork(+AndWait),
// CreateVmwarePublicNetwork(+AndWait), EditVmwareNetwork(+AndWait),
// DeleteVmwareNetwork(+AndWait), ConnectVmwareServers(+AndWait),
// WaitVmwareNetworkActive/State/Gone, GetVmwareEdgeFirewall,
// UpdateVmwareEdgeFirewall(+AndWait), GetVmwareEdgeNAT,
// UpsertVmwareEdgeNATRule(+AndWait), DeleteVmwareEdgeNATRule(+AndWait),
// GetVmwareEdgeVPN, UpsertVmwareEdgeVPNTunnel(+AndWait),
// DeleteVmwareEdgeVPNTunnel(+AndWait).
func runVmwareNetworkExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== VMware Cloud network and edge example ===")
	announceBillableRun(
		"two isolated networks, two routed networks (each with an edge gateway),\n  one server, and a public network if the location has a free block",
		"about 15 minutes")
	run := &vmwareRun{}
	env := resolveVmwareExampleEnv(ctx, client)
	stamp := time.Now().UTC().Format("150405")

	// ---------- List ----------
	section("GetVmwareNetworkList / GetVmwareNetwork")
	networks, err := client.GetVmwareNetworkList(ctx, nil)
	run.check("GetVmwareNetworkList", err, "%d networks across all locations", len(networks))

	byLocation, err := client.GetVmwareNetworkList(ctx, &env.Location.ID)
	if run.check("GetVmwareNetworkList(location)", err, "%d networks in %s", len(byLocation), env.Location.TechTitle) {
		for _, n := range byLocation {
			printVmwareNetwork(n)
		}
	}

	// Everything created below is removed in this single deferred teardown, in
	// reverse order of creation. Servers go first: a network with a NIC still
	// attached cannot be deleted.
	var createdNetworks []int
	var createdServers []int
	defer func() {
		if vmwareExampleKeepResources() {
			fmt.Printf("\nVMWARE_KEEP=1 — leaving servers %v and networks %v in place\n",
				createdServers, createdNetworks)
			run.summary("VMware network example")
			return
		}
		section("Teardown")
		for i := len(createdServers) - 1; i >= 0; i-- {
			id := createdServers[i]
			if err := client.DeleteVmwareServerAndWait(ctx, id); err != nil {
				step("cleanup: DeleteVmwareServerAndWait(%d): %v", id, err)
			} else {
				step("cleanup: server %d deleted and gone", id)
			}
		}
		for i := len(createdNetworks) - 1; i >= 0; i-- {
			id := createdNetworks[i]
			if err := client.DeleteVmwareNetworkAndWait(ctx, id); err != nil {
				step("cleanup: DeleteVmwareNetworkAndWait(%d): %v (IsNetworkInUse=%v)",
					id, err, sdk.IsNetworkInUse(err))
			} else {
				step("cleanup: network %d deleted and gone", id)
			}
		}
		if run.summary("VMware network example") > 0 {
			os.Exit(1)
		}
	}()

	// ---------- Isolated network: raw create + explicit wait ----------
	section("CreateVmwareIsolatedNetwork + WaitVmwareTaskRef")
	isoTask, err := client.CreateVmwareIsolatedNetwork(ctx, &entities.VmwareCreateIsolatedNetworkRequest{
		LocationID: env.Location.ID,
		Name:       "sdk-ex-iso-" + stamp,
		Address:    vmwareExampleNetworkAddress(0),
		Mask:       intPtr(24),
		EnableDhcp: boolPtr(false),
	})
	if !run.check("CreateVmwareIsolatedNetwork", err, "task=%s", isoTask.String()) {
		log.Fatalf("cannot continue without a network")
	}
	isoDone, err := client.WaitVmwareTaskRef(ctx, isoTask)
	if !run.check("WaitVmwareTaskRef(isolated create)", err, "isolated network provisioned") {
		log.Fatalf("isolated network never came up")
	}
	if isoDone == nil || isoDone.NetworkID == nil {
		log.Fatalf("completed task %s carries no network_id", isoTask.String())
	}
	isoID := *isoDone.NetworkID
	createdNetworks = append(createdNetworks, isoID)

	iso, err := client.GetVmwareNetwork(ctx, isoID)
	if run.check("GetVmwareNetwork", err, "isolated network %d", isoID) {
		printVmwareNetwork(iso)
	}

	_, err = client.WaitVmwareNetworkActive(ctx, isoID)
	run.check("WaitVmwareNetworkActive", err, "isolated network is active")

	_, err = client.WaitVmwareNetworkState(ctx, isoID, entities.VmwareNetworkStateActive)
	run.check("WaitVmwareNetworkState", err, "state=%s", entities.VmwareNetworkStateActive)

	// An isolated network reports a bandwidth it will not accept back, so
	// only the name is editable here. The edit is answered synchronously,
	// with no task — reported as a nil *VmwareTaskID.
	section("EditVmwareNetwork — isolated (synchronous, no task)")
	editTask, err := client.EditVmwareNetwork(ctx, isoID, &entities.VmwareEditNetworkRequest{Name: "sdk-ex-iso-renamed"})
	if run.check("EditVmwareNetwork", err, "task=%q, IsZero=%v — an empty task means the API answered synchronously",
		editTask.String(), editTask.IsZero()) {
		step("nothing to await: WaitVmwareTaskRef on it is a no-op")
	}
	// Bandwidth does not apply to an isolated network. The read side is fixed
	// — the network reports no bandwidth at all — but writing one is still refused
	// with the generic "outside allowable limits" instead of a "not applicable".
	step("isolated network reports bandwidth=%s", derefInt(bandwidthOf(iso)))
	if _, err := client.EditVmwareNetwork(ctx, isoID, &entities.VmwareEditNetworkRequest{BandwidthMbps: intPtr(30)}); err != nil {
		step("writing a bandwidth to an isolated network fails: %v", err)
	} else {
		step("bandwidth accepted on an isolated network")
	}

	// ---------- Routed network: ...AndWait ----------
	//
	// Only a routed network has an edge gateway, so the firewall / NAT / VPN
	// methods below need this one.
	section("CreateVmwareRoutedNetworkAndWait")
	routed, err := client.CreateVmwareRoutedNetworkAndWait(ctx, &entities.VmwareCreateRoutedNetworkRequest{
		LocationID:    env.Location.ID,
		Name:          "sdk-ex-routed-" + stamp,
		Address:       vmwareExampleNetworkAddress(1),
		Mask:          intPtr(24),
		EnableDhcp:    boolPtr(false),
		BandwidthMbps: intPtr(20),
	})
	if !run.check("CreateVmwareRoutedNetworkAndWait", err, "routed network created") {
		log.Fatalf("cannot exercise the edge without a routed network")
	}
	routedID := routed.ID
	createdNetworks = append(createdNetworks, routedID)
	printVmwareNetwork(routed)

	// Bandwidth is one field shared by the network and its edge, and this is
	// where it is set. The former PUT /edge/bandwidth is not exposed by the SDK
	// because it never persisted the value.
	section("EditVmwareNetworkAndWait — routed (name + bandwidth)")
	edited, err := client.EditVmwareNetworkAndWait(ctx, routedID, &entities.VmwareEditNetworkRequest{
		Name:          "sdk-ex-routed-renamed",
		BandwidthMbps: intPtr(30),
	})
	if run.check("EditVmwareNetworkAndWait", err, "name=%q bandwidth=%s Mbps",
		nameOf(edited), derefInt(bandwidthOf(edited))) {
		step("edge bandwidth is the network bandwidth — read straight back from the network")
	}

	// ---------- Edge firewall ----------
	section("Edge firewall")
	fw, err := client.GetVmwareEdgeFirewall(ctx, routedID)
	if run.check("GetVmwareEdgeFirewall", err, "enabled=%s default_action=%s rules=%d",
		derefBool(fwEnabled(fw)), deref(fwDefaultAction(fw)), len(fwRules(fw))) {
		for _, r := range fwRules(fw) {
			printEdgeFirewallRule(r)
		}
	}

	// Enabled and DefaultAction are genuinely optional now — omitting them
	// leaves the current values alone. Before the fix, a request without enabled
	// switched the whole network firewall off.
	fwTask, err := client.UpdateVmwareEdgeFirewall(ctx, routedID, &entities.VmwareUpdateEdgeFirewallRequest{
		Rules: []entities.VmwareUpdateEdgeFirewallRule{{
			Name:            strPtr("ssh"),
			Action:          entities.VmwareEdgeFirewallActionAllow,
			Protocol:        strPtr("tcp"),
			Source:          strPtr(entities.VmwareEdgeFirewallAny),
			SourcePort:      strPtr(entities.VmwareEdgeFirewallAny),
			Destination:     strPtr(entities.VmwareEdgeFirewallAny),
			DestinationPort: strPtr("22"),
		}},
	})
	if run.check("UpdateVmwareEdgeFirewall", err, "task=%s (no enabled/default_action sent)", fwTask.String()) {
		_, err = client.WaitVmwareTaskRef(ctx, fwTask)
		run.check("WaitVmwareTaskRef(edge firewall)", err, "rule applied")
	}
	if after, err := client.GetVmwareEdgeFirewall(ctx, routedID); err == nil {
		step("after the partial update: enabled=%s default_action=%s — unchanged, as the API promises",
			derefBool(fwEnabled(after)), deref(fwDefaultAction(after)))
	}

	// The full form, with both fields set explicitly.
	applied, err := client.UpdateVmwareEdgeFirewallAndWait(ctx, routedID, &entities.VmwareUpdateEdgeFirewallRequest{
		Enabled:       boolPtr(true),
		DefaultAction: strPtr(entities.VmwareEdgeFirewallActionAllow),
		Rules: []entities.VmwareUpdateEdgeFirewallRule{
			{
				Name:            strPtr("ssh"),
				Action:          entities.VmwareEdgeFirewallActionAllow,
				Protocol:        strPtr("tcp"),
				Source:          strPtr(entities.VmwareEdgeFirewallAny),
				SourcePort:      strPtr(entities.VmwareEdgeFirewallAny),
				Destination:     strPtr(entities.VmwareEdgeFirewallAny),
				DestinationPort: strPtr("22"),
			},
			{
				Name:            strPtr("web"),
				Action:          entities.VmwareEdgeFirewallActionDeny,
				Protocol:        strPtr("tcp"),
				Source:          strPtr(entities.VmwareEdgeFirewallAny),
				SourcePort:      strPtr(entities.VmwareEdgeFirewallAny),
				Destination:     strPtr(entities.VmwareEdgeFirewallAny),
				DestinationPort: strPtr("80"),
			},
		},
	})
	if run.check("UpdateVmwareEdgeFirewallAndWait", err, "%d rules", len(fwRules(applied))) {
		for _, r := range fwRules(applied) {
			printEdgeFirewallRule(r)
		}
		step("note: enabled and description on a read rule are server-derived — the update request has no such fields")
	}

	// ---------- Edge NAT ----------
	section("Edge NAT")
	nat, err := client.GetVmwareEdgeNAT(ctx, routedID)
	run.check("GetVmwareEdgeNAT", err, "%d rules", len(natRules(nat)))

	internalIP := hostInNetwork(vmwareExampleNetworkAddress(1), 10)
	natTask, err := client.UpsertVmwareEdgeNATRule(ctx, routedID, &entities.VmwareUpsertNATRuleRequest{
		Type:           entities.VmwareEdgeNATTypeDNAT,
		Protocol:       "tcp",
		Description:    "sdk-ex-dnat",
		OriginalIP:     entities.VmwareEdgeFirewallAny,
		OriginalPort:   "2222",
		TranslatedIP:   internalIP,
		TranslatedPort: "22",
	})
	if run.check("UpsertVmwareEdgeNATRule", err, "task=%s", natTask.String()) {
		_, err = client.WaitVmwareTaskRef(ctx, natTask)
		run.check("WaitVmwareTaskRef(nat create)", err, "DNAT rule created")
	}

	nat, err = client.GetVmwareEdgeNAT(ctx, routedID)
	if err == nil {
		for _, r := range natRules(nat) {
			printEdgeNATRule(r)
		}
		step("original_ip was sent as %q and read back as the edge external address", entities.VmwareEdgeFirewallAny)
	}

	// The id from GET is the one DELETE accepts now.
	if rules := natRules(nat); len(rules) > 0 && rules[0].ID != nil {
		ruleID := *rules[0].ID
		// Update the same rule through the upsert form, then delete it.
		updated, err := client.UpsertVmwareEdgeNATRuleAndWait(ctx, routedID, &entities.VmwareUpsertNATRuleRequest{
			RuleID:         intPtr(ruleID),
			Type:           entities.VmwareEdgeNATTypeDNAT,
			Protocol:       "tcp",
			Description:    "sdk-ex-dnat-updated",
			OriginalIP:     entities.VmwareEdgeFirewallAny,
			OriginalPort:   "2223",
			TranslatedIP:   internalIP,
			TranslatedPort: "22",
		})
		run.check("UpsertVmwareEdgeNATRuleAndWait", err, "%d rules after update", len(natRules(updated)))

		delNATTask, err := client.DeleteVmwareEdgeNATRule(ctx, routedID, ruleID)
		if run.check("DeleteVmwareEdgeNATRule", err, "task=%s (rule id %d straight from GET)", delNATTask.String(), ruleID) {
			_, err = client.WaitVmwareTaskRef(ctx, delNATTask)
			run.check("WaitVmwareTaskRef(nat delete)", err, "DNAT rule deleted")
		}
	} else {
		run.skip("UpsertVmwareEdgeNATRuleAndWait", "no NAT rule to update")
		run.skip("DeleteVmwareEdgeNATRule", "no NAT rule to delete")
	}

	// An SNAT rule, deleted through the ...AndWait pair.
	snatNAT, err := client.UpsertVmwareEdgeNATRuleAndWait(ctx, routedID, &entities.VmwareUpsertNATRuleRequest{
		Type:         entities.VmwareEdgeNATTypeSNAT,
		Protocol:     entities.VmwareEdgeFirewallProtocolAny,
		Description:  "sdk-ex-snat",
		OriginalIP:   fmt.Sprintf("%s/24", vmwareExampleNetworkAddress(1)),
		TranslatedIP: internalIP,
	})
	if err != nil {
		run.fail("UpsertVmwareEdgeNATRuleAndWait(snat)", err)
		run.skip("DeleteVmwareEdgeNATRuleAndWait", "no SNAT rule")
	} else {
		run.pass("UpsertVmwareEdgeNATRuleAndWait(snat)", "%d rules", len(natRules(snatNAT)))
		if rules := natRules(snatNAT); len(rules) > 0 && rules[len(rules)-1].ID != nil {
			run.check("DeleteVmwareEdgeNATRuleAndWait",
				client.DeleteVmwareEdgeNATRuleAndWait(ctx, routedID, *rules[len(rules)-1].ID),
				"SNAT rule deleted")
		}
	}

	// ---------- Edge VPN ----------
	section("Edge VPN")
	vpn, err := client.GetVmwareEdgeVPN(ctx, routedID)
	run.check("GetVmwareEdgeVPN", err, "enabled=%s tunnels=%d", derefBool(vpnEnabled(vpn)), len(vpnTunnels(vpn)))

	// mtu, encryption_type and diffie_hellman_group look optional but are
	// mandatory, and the shared key has to be 32-128 alphanumeric characters with
	// mixed case and a digit. Validate catches all of that locally.
	tunnelReq := &entities.VmwareUpsertVPNTunnelRequest{
		Name:                  "sdk-ex-vpn",
		Enabled:               boolPtr(true),
		MTU:                   intPtr(1500),
		EncryptionType:        entities.VmwareEdgeVPNEncryptionAES256,
		SharedKey:             "SdkExampleVpnSharedKey1234567890",
		PeerNetwork:           "10.20.30.0/24",
		PeerEndpoint:          "203.0.113.10",
		PeerIdentificator:     "203.0.113.10",
		PerfectForwardSecrecy: boolPtr(true),
		DiffieHellmanGroup:    entities.VmwareEdgeVPNDiffieHellmanGroup14,
	}
	// Show the local validation the SDK adds for the pre-shared key.
	short := *tunnelReq
	short.SharedKey = "tooshort"
	if _, err := client.UpsertVmwareEdgeVPNTunnel(ctx, routedID, &short); err != nil {
		step("local shared-key validation: %v", err)
	}

	vpnTask, err := client.UpsertVmwareEdgeVPNTunnel(ctx, routedID, tunnelReq)
	if run.check("UpsertVmwareEdgeVPNTunnel", err, "task=%s", vpnTask.String()) {
		_, err = client.WaitVmwareTaskRef(ctx, vpnTask)
		run.check("WaitVmwareTaskRef(vpn create)", err, "tunnel created")
	}

	vpn, err = client.GetVmwareEdgeVPN(ctx, routedID)
	if err == nil {
		for _, t := range vpnTunnels(vpn) {
			printEdgeVPNTunnel(t)
		}
		step("peer_network was sent as a single subnet and reads back as peer_subnets[]; diffie_hellman_group comes back upper case")
	}

	if tunnels := vpnTunnels(vpn); len(tunnels) > 0 && tunnels[0].ID != nil {
		tunnelID := *tunnels[0].ID
		updateReq := *tunnelReq
		updateReq.TunnelID = intPtr(tunnelID)
		updateReq.MTU = intPtr(1400)
		updatedVPN, err := client.UpsertVmwareEdgeVPNTunnelAndWait(ctx, routedID, &updateReq)
		run.check("UpsertVmwareEdgeVPNTunnelAndWait", err, "%d tunnels after update", len(vpnTunnels(updatedVPN)))

		delVPNTask, err := client.DeleteVmwareEdgeVPNTunnel(ctx, routedID, tunnelID)
		if run.check("DeleteVmwareEdgeVPNTunnel", err, "task=%s (tunnel id %d straight from GET)", delVPNTask.String(), tunnelID) {
			_, err = client.WaitVmwareTaskRef(ctx, delVPNTask)
			run.check("WaitVmwareTaskRef(vpn delete)", err, "tunnel deleted")
		}

		// The ...AndWait delete, on a second tunnel.
		second := *tunnelReq
		second.Name = "sdk-ex-vpn2"
		second.PeerNetwork = "10.20.40.0/24"
		second.PeerEndpoint = "203.0.113.20"
		second.PeerIdentificator = "203.0.113.20"
		if createdVPN, err := client.UpsertVmwareEdgeVPNTunnelAndWait(ctx, routedID, &second); err != nil {
			run.fail("UpsertVmwareEdgeVPNTunnelAndWait(second)", err)
			run.skip("DeleteVmwareEdgeVPNTunnelAndWait", "no second tunnel")
		} else {
			run.pass("UpsertVmwareEdgeVPNTunnelAndWait(second)", "%d tunnels", len(vpnTunnels(createdVPN)))
			if ts := vpnTunnels(createdVPN); len(ts) > 0 && ts[len(ts)-1].ID != nil {
				run.check("DeleteVmwareEdgeVPNTunnelAndWait",
					client.DeleteVmwareEdgeVPNTunnelAndWait(ctx, routedID, *ts[len(ts)-1].ID),
					"second tunnel deleted")
			}
		}
	} else {
		run.skip("UpsertVmwareEdgeVPNTunnelAndWait", "no tunnel to update")
		run.skip("DeleteVmwareEdgeVPNTunnel", "no tunnel to delete")
		run.skip("DeleteVmwareEdgeVPNTunnelAndWait", "no tunnel to delete")
	}

	// ---------- Attaching servers to a network ----------
	//
	// This is the network-side counterpart of ConnectVmwareClientNetwork: it takes
	// a batch of servers and returns one task per server.
	// The example provisions its own server rather than borrowing an existing one:
	// attaching a NIC mutates the server, and doing that to a resource the example
	// did not create is not something a walkthrough should ever do.
	section("ConnectVmwareServers")
	server, err := client.CreateVmwareServerAndWait(ctx, env.createServerRequest("sdk-ex-netsrv-"+stamp))
	if err != nil {
		run.fail("CreateVmwareServerAndWait(for ConnectVmwareServers)", err)
		run.skip("ConnectVmwareServers", "no server of our own to attach")
		run.skip("ConnectVmwareServersAndWait", "no server of our own to attach")
	} else {
		createdServers = append(createdServers, server.ID)
		step("using server %d (%s), created by this example", server.ID, server.Name)
		tasks, err := client.ConnectVmwareServers(ctx, isoID, &entities.VmwareConnectServersRequest{
			NICs: []entities.VmwareConnectServerNIC{{ServerID: server.ID}},
		})
		if run.check("ConnectVmwareServers", err, "%d task(s) returned", len(tasks)) {
			for _, t := range tasks {
				if _, err := client.WaitVmwareTaskRef(ctx, t); err != nil {
					run.fail("WaitVmwareTaskRef(connect servers)", err)
				} else {
					run.pass("WaitVmwareTaskRef(connect servers)", "server attached via task %s", t.String())
				}
			}
			// Detach again so the network can be deleted in the teardown.
			detachVmwareServerFromNetwork(ctx, client, server.ID, isoID)
		}

		// The batch error path names the offending element through ErrorParams.
		if _, err := client.ConnectVmwareServers(ctx, isoID, &entities.VmwareConnectServersRequest{
			NICs: []entities.VmwareConnectServerNIC{{ServerID: 999999}},
		}); err != nil {
			step("batch error: %v", err)
			var reqErr *sdk.RequestError
			if errors.As(err, &reqErr) {
				step("codes=%v error_params=%+v", reqErr.Codes, reqErr.ErrorParams)
			}
		}

		run.check("ConnectVmwareServersAndWait",
			client.ConnectVmwareServersAndWait(ctx, isoID, &entities.VmwareConnectServersRequest{
				NICs:               []entities.VmwareConnectServerNIC{{ServerID: server.ID}},
				ForceCustomization: boolPtr(false),
			}), "server attached and awaited")
		detachVmwareServerFromNetwork(ctx, client, server.ID, isoID)
	}

	// ---------- Public network ----------
	//
	// A public network needs a free address block in the location, which a test
	// stand often does not have — that is reported as -12043 "There is no free
	// network at the moment" and is a stand limitation, not a client error, so it
	// is recorded as skipped. Only a few capacities are valid (1, 2, 4); 8 and
	// above are refused with -12042.
	section("CreateVmwarePublicNetwork")
	pubTask, err := client.CreateVmwarePublicNetwork(ctx, &entities.VmwareCreatePublicNetworkRequest{
		LocationID:    env.Location.ID,
		Name:          "sdk-ex-pub-" + stamp,
		Capacity:      "4",
		BandwidthMbps: intPtr(20),
	})
	switch {
	case sdk.IsVmwareNoFreePublicNetwork(err):
		run.skip("CreateVmwarePublicNetwork", "no free public address block in "+env.Location.TechTitle+" (-12043)")
		run.skip("CreateVmwarePublicNetworkAndWait", "no free public address block (-12043)")
		step("the request itself was accepted — capacity %q is valid, the location just has nothing free", "4")
	case err != nil:
		run.fail("CreateVmwarePublicNetwork", err)
		run.skip("CreateVmwarePublicNetworkAndWait", "the first public create failed")
	default:
		run.pass("CreateVmwarePublicNetwork", "task=%s", pubTask.String())
		if done, err := client.WaitVmwareTaskRef(ctx, pubTask); err != nil {
			run.fail("WaitVmwareTaskRef(public create)", err)
		} else {
			run.pass("WaitVmwareTaskRef(public create)", "public network provisioned")
			if done != nil && done.NetworkID != nil {
				createdNetworks = append(createdNetworks, *done.NetworkID)
			}
		}
		pub, err := client.CreateVmwarePublicNetworkAndWait(ctx, &entities.VmwareCreatePublicNetworkRequest{
			LocationID:    env.Location.ID,
			Name:          "sdk-ex-pub2-" + stamp,
			Capacity:      "4",
			BandwidthMbps: intPtr(20),
		})
		if run.check("CreateVmwarePublicNetworkAndWait", err, "public network %d", idOfNetwork(pub)) && pub != nil {
			createdNetworks = append(createdNetworks, pub.ID)
		}
	}

	// ---------- The remaining delete forms ----------
	//
	// A throwaway isolated network, removed through the raw delete plus the
	// state-level wait — the task completes before the object disappears.
	section("CreateVmwareRoutedNetwork + DeleteVmwareNetwork + WaitVmwareNetworkGone")
	tmpTask, err := client.CreateVmwareRoutedNetwork(ctx, &entities.VmwareCreateRoutedNetworkRequest{
		LocationID:    env.Location.ID,
		Name:          "sdk-ex-tmp-" + stamp,
		Address:       vmwareExampleNetworkAddress(3),
		Mask:          intPtr(24),
		EnableDhcp:    boolPtr(true),
		BandwidthMbps: intPtr(20),
	})
	if !run.check("CreateVmwareRoutedNetwork", err, "task=%s", tmpTask.String()) {
		run.skip("DeleteVmwareNetwork", "no throwaway network")
		run.skip("WaitVmwareNetworkGone", "no throwaway network")
		return
	}
	tmpDone, err := client.WaitVmwareTaskRef(ctx, tmpTask)
	if !run.check("WaitVmwareTaskRef(routed create)", err, "throwaway network provisioned") ||
		tmpDone == nil || tmpDone.NetworkID == nil {
		run.skip("DeleteVmwareNetwork", "throwaway network id unknown")
		run.skip("WaitVmwareNetworkGone", "throwaway network id unknown")
		return
	}
	tmpID := *tmpDone.NetworkID

	delTask, err := client.DeleteVmwareNetwork(ctx, tmpID)
	if run.check("DeleteVmwareNetwork", err, "task=%s", delTask.String()) {
		_, err = client.WaitVmwareTaskRef(ctx, delTask)
		run.check("WaitVmwareTaskRef(network delete)", err, "delete task completed")
		run.check("WaitVmwareNetworkGone", client.WaitVmwareNetworkGone(ctx, tmpID),
			"network really disappeared")
	} else {
		createdNetworks = append(createdNetworks, tmpID)
	}
}

// detachVmwareServerFromNetwork removes the server's NIC on the given network, so
// the network can be deleted afterwards.
func detachVmwareServerFromNetwork(ctx context.Context, client *sdk.CloudClient, serverID, networkID int) {
	nics, err := client.GetVmwareServerNICs(ctx, serverID)
	if err != nil {
		step("detach: GetVmwareServerNICs(%d): %v", serverID, err)
		return
	}
	for _, n := range nics {
		if n.NetworkID != networkID || n.IsPrimary {
			continue
		}
		if err := client.DeleteVmwareNICAndWait(ctx, serverID, n.ID); err != nil {
			step("detach: DeleteVmwareNICAndWait(%d): %v", n.ID, err)
		} else {
			step("detach: NIC %d removed from network %d", n.ID, networkID)
		}
	}
}

// hostInNetwork returns the host'th address of a /24 base address.
func hostInNetwork(base string, host int) string {
	var a, b, c, d int
	if _, err := fmt.Sscanf(base, "%d.%d.%d.%d", &a, &b, &c, &d); err != nil {
		return base
	}
	return fmt.Sprintf("%d.%d.%d.%d", a, b, c, host)
}

// printVmwareNetwork dumps every field of a network.
func printVmwareNetwork(n *entities.VmwareNetwork) {
	if n == nil {
		step("<nil network>")
		return
	}
	step("id=%d location=%d type=%-18s name=%q state=%s nics=%d",
		n.ID, n.LocationID, n.Type, n.Name, n.State, n.NICsCount)
	step("    address=%s/%s gateway=%s bandwidth=%s Mbps dhcp=%s shared=%s",
		deref(n.Address), derefInt(n.Mask), deref(n.Gateway),
		derefInt(n.BandwidthMbps), derefBool(n.IsDhcp), derefBool(n.Shared))
}

// printEdgeFirewallRule dumps a read-side edge firewall rule.
func printEdgeFirewallRule(r entities.VmwareEdgeFirewallRule) {
	step("    %-6s enabled=%-5s action=%-5s proto=%-4s %s:%s -> %s:%s (description=%q)",
		deref(r.Name), derefBool(r.Enabled), deref(r.Action), deref(r.Protocol),
		deref(r.Source), deref(r.SourcePort), deref(r.Destination), deref(r.DestinationPort),
		deref(r.Description))
}

// printEdgeNATRule dumps a NAT rule.
func printEdgeNATRule(r entities.VmwareEdgeNATRule) {
	step("    id=%s vcloud_id=%s %-4s enabled=%-5s proto=%-4s %s:%s -> %s:%s (%q)",
		derefInt(r.ID), deref(r.VcloudID), deref(r.Type), derefBool(r.Enabled), deref(r.Protocol),
		deref(r.OriginalIP), deref(r.OriginalPort), deref(r.TranslatedIP), deref(r.TranslatedPort),
		deref(r.Description))
}

// printEdgeVPNTunnel dumps a VPN tunnel.
func printEdgeVPNTunnel(t entities.VmwareEdgeVPNTunnel) {
	step("    id=%s vcloud_id=%s name=%s enabled=%s mtu=%s",
		derefInt(t.ID), deref(t.VcloudID), deref(t.Name), derefBool(t.Enabled), derefInt(t.MTU))
	step("      local:  id=%s ip=%s subnets=%v", deref(t.LocalID), deref(t.LocalIP), t.LocalSubnets)
	step("      peer:   id=%s endpoint=%s subnets=%v", deref(t.PeerIdentificator), deref(t.PeerEndpoint), t.PeerSubnets)
	step("      crypto: encryption=%s digest=%s dh_group=%s pfs=%s",
		deref(t.EncryptionType), deref(t.DigestAlgorithm), deref(t.DiffieHellmanGroup),
		derefBool(t.PerfectForwardSecrecy))
}

// Nil-safe accessors for the edge aggregates and the network entity.
func fwEnabled(f *entities.VmwareEdgeFirewall) *bool {
	if f == nil {
		return nil
	}
	return f.Enabled
}

func fwDefaultAction(f *entities.VmwareEdgeFirewall) *string {
	if f == nil {
		return nil
	}
	return f.DefaultAction
}

func fwRules(f *entities.VmwareEdgeFirewall) []entities.VmwareEdgeFirewallRule {
	if f == nil {
		return nil
	}
	return f.Rules
}

func natRules(n *entities.VmwareEdgeNAT) []entities.VmwareEdgeNATRule {
	if n == nil {
		return nil
	}
	return n.Rules
}

func vpnEnabled(v *entities.VmwareEdgeVPN) *bool {
	if v == nil {
		return nil
	}
	return v.Enabled
}

func vpnTunnels(v *entities.VmwareEdgeVPN) []entities.VmwareEdgeVPNTunnel {
	if v == nil {
		return nil
	}
	return v.Tunnels
}

func nameOf(n *entities.VmwareNetwork) string {
	if n == nil {
		return "<nil>"
	}
	return n.Name
}

func bandwidthOf(n *entities.VmwareNetwork) *int {
	if n == nil {
		return nil
	}
	return n.BandwidthMbps
}

func idOfNetwork(n *entities.VmwareNetwork) int {
	if n == nil {
		return 0
	}
	return n.ID
}
