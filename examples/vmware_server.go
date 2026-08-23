package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	sdk "github.com/itglobalcom/vstack-cloud-panel-sdk"
	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// runVmwareServerExample drives the whole VMware server surface against a live
// project: it orders a server, walks every method that touches it, and deletes
// what it created.
//
// Each method is exercised in one of its two forms on purpose — some through the
// raw call plus an explicit wait, some through the ...AndWait variant — so both
// paths are covered. A platform-level rejection is reported and the run continues;
// the tally at the end says what passed.
//
// Methods covered (47): VerifyVmwareServer, CreateVmwareServer(+AndWait),
// GetVmwareServerList, GetVmwareServer, WaitVmwareServerActive/State/Gone,
// GetVmwareServerFirewall, UpdateVmwareServerFirewall(+AndWait),
// GetVmwareServerVolumes, GetVmwareVolume, CreateVmwareVolume(+AndWait),
// EditVmwareVolume(+AndWait), DeleteVmwareVolume(+AndWait), GetVmwareSnapshot,
// CreateVmwareSnapshot(+AndWait), RestoreVmwareSnapshot(+AndWait),
// DeleteVmwareSnapshot(+AndWait), GetVmwareServerNICs,
// ConnectVmwareClientNetwork(+AndWait), ConnectVmwareSharedNetwork(+AndWait),
// UpdateVmwareNIC(+AndWait), DeleteVmwareNIC(+AndWait),
// PowerOn/PowerOff/Shutdown/Reboot/Reset VmwareServer(+AndWait),
// ChangeVmwareServerConfiguration(+AndWait), RenameVmwareServer,
// ChangeVmwareServerComputerName(+AndWait),
// EnableVmwareServerNestedHypervisor(+AndWait),
// DisableVmwareServerNestedHypervisor(+AndWait), CopyVmwareServer(+AndWait),
// RebuildVmwareServer(+AndWait), DeleteVmwareServer(+AndWait).
func runVmwareServerExample(ctx context.Context, client *sdk.CloudClient) {
	fmt.Println("=== VMware Cloud server example ===")
	if vmwareExampleSkipLong() {
		announceBillableRun("one server, one isolated network", "about 40 minutes")
	} else {
		announceBillableRun(
			"up to four servers (one at a time: the original, a second one, two copies,\n  plus the replacements two rebuilds create) and one isolated network",
			"about 100 minutes — set VMWARE_SKIP_LONG=1 to cut it to ~40")
	}
	run := &vmwareRun{}
	env := resolveVmwareExampleEnv(ctx, client)
	stamp := time.Now().UTC().Format("150405")

	// ---------- Verify (dry run) ----------
	section("VerifyVmwareServer — validate an order without placing it")
	order := env.createServerRequest("sdk-ex-srv-" + stamp)
	run.check("VerifyVmwareServer", client.VerifyVmwareServer(ctx, order),
		"order accepted: %d CPU, %d MB RAM, %d MB %s system disk",
		order.CPUCount, order.RamMB, order.SystemDiskSizeMB, order.SystemDiskType)

	// A deliberately invalid order is rejected locally, before any request goes out.
	bad := env.createServerRequest("")
	if err := client.VerifyVmwareServer(ctx, bad); err != nil {
		step("local validation works: %v", err)
	}

	// ---------- Create: raw call + explicit wait ----------
	//
	// The raw form returns the order (new server id + task id). This is the shape a
	// provider stores in state, so wait on it explicitly here rather than through
	// ...AndWait, which is shown further down for the copy.
	section("CreateVmwareServer + WaitVmwareTask")
	created, err := client.CreateVmwareServer(ctx, order)
	if err != nil {
		log.Fatalf("CreateVmwareServer: %v", err)
	}
	run.pass("CreateVmwareServer", "server_id=%d task=%s", created.ServerID, created.TaskID)
	serverID := created.ServerID

	// From here on everything the example created must be cleaned up even if a step
	// fails. Networks are torn down after the server, since a network with a NIC
	// still attached cannot be deleted.
	var extraNetworks []int
	defer func() {
		if vmwareExampleKeepResources() {
			fmt.Printf("\nVMWARE_KEEP=1 — leaving server %d and networks %v in place\n", serverID, extraNetworks)
			run.summary("VMware server example")
			return
		}
		section("Teardown")
		run.check("DeleteVmwareServerAndWait", client.DeleteVmwareServerAndWait(ctx, serverID),
			"server %d deleted and gone", serverID)
		// Backstop for a rebuild whose ...AndWait form timed out: it creates a
		// replacement server and returns no id for it.
		sweepVmwareExampleServers(ctx, client, env.Location.ID, stamp, serverID)
		for _, networkID := range extraNetworks {
			if err := client.DeleteVmwareNetworkAndWait(ctx, networkID); err != nil {
				step("cleanup: DeleteVmwareNetworkAndWait(%d): %v", networkID, err)
			} else {
				step("cleanup: network %d deleted", networkID)
			}
		}
		if run.summary("VMware server example") > 0 {
			os.Exit(1)
		}
	}()

	task, err := client.WaitVmwareTask(ctx, created.TaskID)
	if !run.check("WaitVmwareTask", err, "provisioning finished") {
		log.Fatalf("server %d never came up", serverID)
	}
	if task != nil {
		step("task %s: type=%s state=%s progress=%s completed=%s",
			task.ID, task.Type, task.State, derefInt(task.ProgressPercent), deref(task.Completed))
	}

	// ---------- Read ----------
	section("GetVmwareServer / GetVmwareServerList / WaitVmwareServerActive")
	server, err := client.GetVmwareServer(ctx, serverID)
	if run.check("GetVmwareServer", err, "reading server %d", serverID) {
		printVmwareServer(server)
	}

	servers, err := client.GetVmwareServerList(ctx, nil)
	run.check("GetVmwareServerList", err, "%d servers across all locations", len(servers))

	byLocation, err := client.GetVmwareServerList(ctx, &env.Location.ID)
	run.check("GetVmwareServerList(location)", err, "%d servers in %s", len(byLocation), env.Location.TechTitle)

	// A completed task is not a settled resource; this is the state-level wait.
	server, err = client.WaitVmwareServerActive(ctx, serverID)
	if run.check("WaitVmwareServerActive", err, "state=%s power=%v", stateOf(server), powerOf(server)) {
		// Vm_tools_installed is a live-only field — present here, nil in the
		// list response above.
		step("vm_tools_installed=%s (list responses omit it)", derefBool(server.VmToolsInstalled))
	}

	_, err = client.WaitVmwareServerState(ctx, serverID,
		entities.VmwareServerStateActive, entities.VmwareServerStatePoweredOff)
	run.check("WaitVmwareServerState", err, "reached one of active/powered_off")

	// ---------- Server firewall ----------
	//
	// Name and traffic_direction are exposed by the API, so a full rule set
	// round-trips. The read type is the write type, which makes read-modify-write a
	// direct copy.
	section("Server firewall")
	rules, err := client.GetVmwareServerFirewall(ctx, serverID)
	run.check("GetVmwareServerFirewall", err, "%d rules (an empty set is a 200, not a 404)", len(rules))

	wanted := []entities.VmwareServerFirewallRule{{
		Name:             "ssh-in",
		TrafficDirection: entities.VmwareTrafficDirectionIncoming,
		Action:           entities.VmwareFirewallActionAllow,
		Protocol:         "tcp",
		Source:           strPtr(entities.VmwareFirewallAny),
		SourcePort:       strPtr(entities.VmwareFirewallAny),
		Destination:      strPtr(entities.VmwareFirewallAny),
		DestinationPort:  strPtr("22"),
	}}
	fwTask, err := client.UpdateVmwareServerFirewall(ctx, serverID,
		&entities.VmwareUpdateServerFirewallRequest{Rules: wanted})
	if run.check("UpdateVmwareServerFirewall", err, "task=%s", fwTask.String()) {
		_, err = client.WaitVmwareTaskRef(ctx, fwTask)
		run.check("WaitVmwareTaskRef", err, "firewall rule applied")
	}

	// Read-modify-write: append a second rule to what the API returned.
	rules, err = client.GetVmwareServerFirewall(ctx, serverID)
	if err == nil {
		rules = append(rules, entities.VmwareServerFirewallRule{
			Name:             "all-out",
			TrafficDirection: entities.VmwareTrafficDirectionOutgoing,
			Action:           entities.VmwareFirewallActionAllow,
			Protocol:         entities.VmwareFirewallProtocolAny,
			Source:           strPtr(entities.VmwareFirewallAny),
			SourcePort:       strPtr(entities.VmwareFirewallAny),
			Destination:      strPtr(entities.VmwareFirewallAny),
			DestinationPort:  strPtr(entities.VmwareFirewallAny),
		})
		applied, err := client.UpdateVmwareServerFirewallAndWait(ctx, serverID,
			&entities.VmwareUpdateServerFirewallRequest{Rules: rules})
		if run.check("UpdateVmwareServerFirewallAndWait", err, "%d rules after read-modify-write", len(applied)) {
			for _, r := range applied {
				step("%-8s %-8s %-5s %-4s %s:%s -> %s:%s", r.Name, r.TrafficDirection, r.Action, r.Protocol,
					deref(r.Source), deref(r.SourcePort), deref(r.Destination), deref(r.DestinationPort))
			}
		}
	}

	// An empty rule set clears the firewall. Depending on the API this is answered
	// with a task or applied synchronously, so the SDK reports the no-task case as
	// (nil, nil); the example prints whichever shape came back.
	cleared, err := client.UpdateVmwareServerFirewall(ctx, serverID,
		&entities.VmwareUpdateServerFirewallRequest{Rules: nil})
	if run.check("UpdateVmwareServerFirewall(clear)", err, "task=%q, IsZero=%v",
		cleared.String(), cleared.IsZero()) {
		if cleared.IsZero() {
			step("no task: the API applied it synchronously — WaitVmwareTaskRef is a no-op")
		} else {
			step("a task was started, so clearing is asynchronous like any other update")
		}
		if _, err := client.WaitVmwareTaskRef(ctx, cleared); err != nil {
			run.fail("WaitVmwareTaskRef(firewall clear)", err)
		} else {
			run.pass("WaitVmwareTaskRef(firewall clear)", "firewall cleared")
		}
	}

	// ---------- Volumes ----------
	section("Volumes")
	volReq := &entities.VmwareCreateVolumeRequest{
		Name:     "sdk-ex-vol",
		DiskType: env.DiskType.Title,
		SizeMB:   env.DiskType.MinMB,
	}
	volTask, err := client.CreateVmwareVolume(ctx, serverID, volReq)
	if run.check("CreateVmwareVolume", err, "task=%s", volTask.String()) {
		_, err = client.WaitVmwareTaskRef(ctx, volTask)
		run.check("WaitVmwareTaskRef(volume)", err, "volume created")
	}

	volumes, err := client.GetVmwareServerVolumes(ctx, serverID)
	run.check("GetVmwareServerVolumes", err, "%d data volumes", len(volumes))
	for _, v := range volumes {
		step("#%d %s — %d MB, type %s", v.ID, v.Name, v.SizeMB, deref(v.DiskType))
	}

	if len(volumes) > 0 {
		volumeID := volumes[0].ID
		// sizeMB tracks the size the volume is expected to have, so a failed read or
		// resize cannot feed a zero (or a nil dereference) into the next request — a
		// volume can only grow, and only by whole steps.
		sizeMB := volumes[0].SizeMB
		if vol, err := client.GetVmwareVolume(ctx, serverID, volumeID); run.check("GetVmwareVolume", err, "volume %d", volumeID) {
			step("%s — %d MB, type %s", vol.Name, vol.SizeMB, deref(vol.DiskType))
			sizeMB = vol.SizeMB
		}

		sizeMB += env.DiskType.StepMB
		grown, err := client.EditVmwareVolumeAndWait(ctx, serverID, volumeID,
			&entities.VmwareEditVolumeRequest{Name: "sdk-ex-vol-grown", SizeMB: sizeMB})
		if run.check("EditVmwareVolumeAndWait", err, "resized to %d MB", sizeOf(grown)) && grown != nil {
			sizeMB = grown.SizeMB
		}

		// The raw pair, growing it once more.
		sizeMB += env.DiskType.StepMB
		editVolTask, err := client.EditVmwareVolume(ctx, serverID, volumeID,
			&entities.VmwareEditVolumeRequest{Name: "sdk-ex-vol-grown2", SizeMB: sizeMB})
		if run.check("EditVmwareVolume", err, "task=%s", editVolTask.String()) {
			_, err = client.WaitVmwareTaskRef(ctx, editVolTask)
			run.check("WaitVmwareTaskRef(volume edit)", err, "volume resized to %d MB", sizeMB)
		}

		delVolTask, err := client.DeleteVmwareVolume(ctx, serverID, volumeID)
		if run.check("DeleteVmwareVolume", err, "task=%s", delVolTask.String()) {
			_, err = client.WaitVmwareTaskRef(ctx, delVolTask)
			run.check("WaitVmwareTaskRef(volume delete)", err, "volume deleted")
		}
	} else {
		run.skip("GetVmwareVolume", "no volume to read")
		run.skip("EditVmwareVolumeAndWait", "no volume to resize")
		run.skip("EditVmwareVolume", "no volume to resize")
		run.skip("DeleteVmwareVolume", "no volume to delete")
	}

	// The ...AndWait pair, on a second volume.
	if run.check("CreateVmwareVolumeAndWait",
		client.CreateVmwareVolumeAndWait(ctx, serverID, &entities.VmwareCreateVolumeRequest{
			Name:     "sdk-ex-vol2",
			DiskType: env.DiskType.Title,
			SizeMB:   env.DiskType.MinMB,
		}), "second volume created") {
		volumes, err = client.GetVmwareServerVolumes(ctx, serverID)
		if err == nil && len(volumes) > 0 {
			run.check("DeleteVmwareVolumeAndWait",
				client.DeleteVmwareVolumeAndWait(ctx, serverID, volumes[len(volumes)-1].ID),
				"second volume deleted")
		}
	}

	// ---------- Snapshot ----------
	//
	// A server holds at most one snapshot, and "no snapshot" is reported as a 404.
	section("Snapshot")
	if _, err := client.GetVmwareSnapshot(ctx, serverID); err != nil {
		step("GetVmwareSnapshot before creating one: %v (IsNotFound=%v — the normal \"no snapshot\" signal)",
			err, sdk.IsNotFound(err))
	}

	snapTask, err := client.CreateVmwareSnapshot(ctx, serverID, &entities.VmwareCreateSnapshotRequest{Name: "sdk-ex-snap"})
	if run.check("CreateVmwareSnapshot", err, "task=%s", snapTask.String()) {
		_, err = client.WaitVmwareTaskRef(ctx, snapTask)
		run.check("WaitVmwareTaskRef(snapshot)", err, "snapshot created")
	}

	snap, err := client.GetVmwareSnapshot(ctx, serverID)
	if run.check("GetVmwareSnapshot", err, "snapshot present") {
		step("%s — created %s", snap.Name, snap.Created)
	}

	_, err = client.RestoreVmwareSnapshotAndWait(ctx, serverID)
	run.check("RestoreVmwareSnapshotAndWait", err, "server restored to its snapshot")

	run.check("DeleteVmwareSnapshotAndWait", client.DeleteVmwareSnapshotAndWait(ctx, serverID), "snapshot deleted")

	// The remaining snapshot pair: the raw restore and the raw delete.
	if snap2, err := client.CreateVmwareSnapshotAndWait(ctx, serverID, &entities.VmwareCreateSnapshotRequest{Name: "sdk-ex-snap2"}); err != nil {
		run.fail("CreateVmwareSnapshotAndWait", err)
		run.skip("RestoreVmwareSnapshot", "no snapshot")
		run.skip("DeleteVmwareSnapshot", "no snapshot")
	} else {
		run.pass("CreateVmwareSnapshotAndWait", "second snapshot %q created", snap2.Name)
		restoreTask, err := client.RestoreVmwareSnapshot(ctx, serverID)
		if run.check("RestoreVmwareSnapshot", err, "task=%s", restoreTask.String()) {
			_, err = client.WaitVmwareTaskRef(ctx, restoreTask)
			run.check("WaitVmwareTaskRef(restore)", err, "restored")
		}
		delSnapTask, err := client.DeleteVmwareSnapshot(ctx, serverID)
		if run.check("DeleteVmwareSnapshot", err, "task=%s", delSnapTask.String()) {
			_, err = client.WaitVmwareTaskRef(ctx, delSnapTask)
			run.check("WaitVmwareTaskRef(snapshot delete)", err, "snapshot deleted")
		}
	}

	// ---------- NICs ----------
	section("NICs")
	nics, err := client.GetVmwareServerNICs(ctx, serverID)
	if run.check("GetVmwareServerNICs", err, "%d NICs", len(nics)) {
		for _, n := range nics {
			// Bandwidth_mbps is read back now.
			step("#%d number=%d primary=%v network=%d ip=%s mac=%s bandwidth=%d Mbps",
				n.ID, n.Number, n.IsPrimary, n.NetworkID, deref(n.IP), n.Mac, n.BandwidthMbps)
		}
	}

	// A client network to attach to. The isolated kind is the cheapest.
	nicNet, err := client.CreateVmwareIsolatedNetworkAndWait(ctx, &entities.VmwareCreateIsolatedNetworkRequest{
		LocationID: env.Location.ID,
		Name:       "sdk-ex-nicnet-" + stamp,
		Address:    vmwareExampleNetworkAddress(2),
		Mask:       intPtr(24),
		EnableDhcp: boolPtr(false),
	})
	if err != nil {
		run.fail("CreateVmwareIsolatedNetworkAndWait(for NIC)", err)
		run.skip("ConnectVmwareClientNetworkAndWait", "no client network")
		run.skip("UpdateVmwareNICAndWait", "no client network")
		run.skip("DeleteVmwareNICAndWait", "no client network")
	} else {
		step("client network %d (%s/%s) ready", nicNet.ID, deref(nicNet.Address), derefInt(nicNet.Mask))
		extraNetworks = append(extraNetworks, nicNet.ID)

		attached, err := client.ConnectVmwareClientNetworkAndWait(ctx, serverID,
			&entities.VmwareConnectClientNetworkRequest{NetworkID: nicNet.ID})
		if run.check("ConnectVmwareClientNetworkAndWait", err, "%d NICs after attach", len(attached)) {
			newNIC := findNICByNetwork(attached, nicNet.ID)
			if newNIC == nil {
				run.skip("UpdateVmwareNICAndWait", "attached NIC not found")
				run.skip("DeleteVmwareNICAndWait", "attached NIC not found")
			} else {
				step("attached NIC #%d ip=%s", newNIC.ID, deref(newNIC.IP))
				// On a client-network NIC the editable field is the static IP — the
				// network was created with DHCP off, which is what makes that legal.
				// Bandwidth is NOT editable here (see the shared NIC below).
				updated, err := client.UpdateVmwareNICAndWait(ctx, serverID, newNIC.ID,
					&entities.VmwareUpdateNICRequest{
						NetworkID: nicNet.ID,
						IP:        hostInNetwork(vmwareExampleNetworkAddress(2), 5),
					})
				if run.check("UpdateVmwareNICAndWait", err, "%d NICs after update", len(updated)) {
					if n := findNICByNetwork(updated, nicNet.ID); n != nil {
						step("static IP is now %s", deref(n.IP))
					}
				}

				run.check("DeleteVmwareNICAndWait",
					client.DeleteVmwareNICAndWait(ctx, serverID, newNIC.ID), "NIC detached")
			}
		}

		// The raw attach/detach pair, on the same network.
		connTask, err := client.ConnectVmwareClientNetwork(ctx, serverID,
			&entities.VmwareConnectClientNetworkRequest{NetworkID: nicNet.ID})
		if run.check("ConnectVmwareClientNetwork", err, "task=%s", connTask.String()) {
			if _, err := client.WaitVmwareTaskRef(ctx, connTask); err != nil {
				run.fail("WaitVmwareTaskRef(nic attach)", err)
			} else {
				run.pass("WaitVmwareTaskRef(nic attach)", "NIC attached")
				nics, _ = client.GetVmwareServerNICs(ctx, serverID)
				if n := findNICByNetwork(nics, nicNet.ID); n != nil {
					delNICTask, err := client.DeleteVmwareNIC(ctx, serverID, n.ID)
					if run.check("DeleteVmwareNIC", err, "task=%s", delNICTask.String()) {
						_, err = client.WaitVmwareTaskRef(ctx, delNICTask)
						run.check("WaitVmwareTaskRef(nic delete)", err, "NIC detached")
					}
				}
			}
		}
	}

	// The raw UpdateVmwareNIC, exercising the bandwidth path — which only exists for
	// a NIC on a shared/public network. Sending bandwidth_mbps for a client-network
	// NIC is refused with "The network bandwidth cannot be changed".
	nics, err = client.GetVmwareServerNICs(ctx, serverID)
	if err != nil {
		run.skip("UpdateVmwareNIC", "cannot list NICs")
	} else if shared := findPrimaryVmwareNIC(nics); shared == nil {
		run.skip("UpdateVmwareNIC", "no shared-network NIC to reshape")
	} else {
		step("reshaping shared NIC #%d on network %d (currently %d Mbps)",
			shared.ID, shared.NetworkID, shared.BandwidthMbps)
		nicTask, err := client.UpdateVmwareNIC(ctx, serverID, shared.ID,
			&entities.VmwareUpdateNICRequest{NetworkID: shared.NetworkID, BandwidthMbps: intPtr(20)})
		if run.check("UpdateVmwareNIC", err, "task=%s", nicTask.String()) {
			if _, err := client.WaitVmwareTaskRef(ctx, nicTask); err != nil {
				run.fail("WaitVmwareTaskRef(nic update)", err)
			} else {
				run.pass("WaitVmwareTaskRef(nic update)", "bandwidth applied")
				if updated, err := client.GetVmwareServerNICs(ctx, serverID); err == nil {
					if n := findVmwareNICByID(updated, shared.ID); n != nil {
						step("NIC #%d bandwidth reads back as %d Mbps", n.ID, n.BandwidthMbps)
					}
				}
			}
		}
	}

	// A shared (public) network NIC. There is no network id to pass — the platform
	// picks the shared network for the location.
	sharedTask, err := client.ConnectVmwareSharedNetwork(ctx, serverID,
		&entities.VmwareConnectSharedNetworkRequest{BandwidthMbps: 10})
	if run.check("ConnectVmwareSharedNetwork", err, "task=%s", sharedTask.String()) {
		if _, err := client.WaitVmwareTaskRef(ctx, sharedTask); err != nil {
			run.fail("WaitVmwareTaskRef(shared attach)", err)
		} else {
			run.pass("WaitVmwareTaskRef(shared attach)", "shared NIC attached")
		}
	}
	sharedNICs, err := client.ConnectVmwareSharedNetworkAndWait(ctx, serverID,
		&entities.VmwareConnectSharedNetworkRequest{BandwidthMbps: 10, IsIPv6: boolPtr(false)})
	if run.check("ConnectVmwareSharedNetworkAndWait", err, "%d NICs after attach", len(sharedNICs)) {
		// Detach the extra shared NICs again, keeping the primary one.
		for _, n := range sharedNICs {
			if n.IsPrimary || n.Number == 0 {
				continue
			}
			if err := client.DeleteVmwareNICAndWait(ctx, serverID, n.ID); err != nil {
				step("cleanup: DeleteVmwareNICAndWait(%d): %v", n.ID, err)
			}
		}
	}

	// ---------- Power ----------
	//
	// Graceful shutdown and reboot go through VMware Tools; a hard off/reset does
	// not. Both are exercised.
	section("Power")
	powered, err := client.ShutdownVmwareServerAndWait(ctx, serverID)
	run.check("ShutdownVmwareServerAndWait", err, "power=%v state=%s", powerOf(powered), stateOf(powered))

	powered, err = client.PowerOnVmwareServerAndWait(ctx, serverID)
	run.check("PowerOnVmwareServerAndWait", err, "power=%v state=%s", powerOf(powered), stateOf(powered))

	powered, err = client.RebootVmwareServerAndWait(ctx, serverID)
	run.check("RebootVmwareServerAndWait", err, "power=%v state=%s", powerOf(powered), stateOf(powered))

	powered, err = client.ResetVmwareServerAndWait(ctx, serverID)
	run.check("ResetVmwareServerAndWait", err, "power=%v state=%s", powerOf(powered), stateOf(powered))

	// The raw pairs.
	if t, err := client.RebootVmwareServer(ctx, serverID); run.check("RebootVmwareServer", err, "task=%s", t.String()) {
		_, err = client.WaitVmwareTaskRef(ctx, t)
		run.check("WaitVmwareTaskRef(reboot)", err, "rebooted")
	}
	if t, err := client.ShutdownVmwareServer(ctx, serverID); run.check("ShutdownVmwareServer", err, "task=%s", t.String()) {
		_, err = client.WaitVmwareTaskRef(ctx, t)
		run.check("WaitVmwareTaskRef(shutdown)", err, "shut down")
	}
	if t, err := client.PowerOnVmwareServer(ctx, serverID); run.check("PowerOnVmwareServer", err, "task=%s", t.String()) {
		_, err = client.WaitVmwareTaskRef(ctx, t)
		run.check("WaitVmwareTaskRef(power on)", err, "powered on")
	}
	if t, err := client.ResetVmwareServer(ctx, serverID); run.check("ResetVmwareServer", err, "task=%s", t.String()) {
		_, err = client.WaitVmwareTaskRef(ctx, t)
		run.check("WaitVmwareTaskRef(reset)", err, "reset")
	}

	// A hard power-off, needed for the configuration change below when the image
	// does not support CPU hot-add.
	if t, err := client.PowerOffVmwareServer(ctx, serverID); run.check("PowerOffVmwareServer", err, "task=%s", t.String()) {
		_, err = client.WaitVmwareTaskRef(ctx, t)
		run.check("WaitVmwareTaskRef(power off)", err, "powered off")
	}
	// The ...AndWait form: power on, then off again, leaving the server off.
	powered, err = client.PowerOnVmwareServerAndWait(ctx, serverID)
	run.check("PowerOnVmwareServerAndWait(again)", err, "power=%v", powerOf(powered))
	powered, err = client.PowerOffVmwareServerAndWait(ctx, serverID)
	run.check("PowerOffVmwareServerAndWait", err, "power=%v state=%s", powerOf(powered), stateOf(powered))

	// ---------- Configuration, names ----------
	section("Configuration and names")
	server, err = client.GetVmwareServer(ctx, serverID)
	if err == nil {
		reconfigured, err := client.ChangeVmwareServerConfigurationAndWait(ctx, serverID,
			&entities.VmwareChangeConfigurationRequest{
				CPU:              server.CPU + 1,
				RamMB:            server.RamMB * 2,
				SystemDiskSizeMB: server.SystemDiskMB,
			})
		if run.check("ChangeVmwareServerConfigurationAndWait", err, "cpu=%d ram=%d MB",
			cpuOf(reconfigured), ramOf(reconfigured)) {
			// The raw pair: put the configuration back.
			t, err := client.ChangeVmwareServerConfiguration(ctx, serverID,
				&entities.VmwareChangeConfigurationRequest{
					CPU:              server.CPU,
					RamMB:            server.RamMB,
					SystemDiskSizeMB: server.SystemDiskMB,
				})
			if run.check("ChangeVmwareServerConfiguration", err, "task=%s", t.String()) {
				_, err = client.WaitVmwareTaskRef(ctx, t)
				run.check("WaitVmwareTaskRef(reconfigure)", err, "configuration restored")
			}
		}
	}

	// Renaming is synchronous — no task, hence no ...AndWait variant. The new name
	// keeps the stamp so the teardown sweep can still recognize the server (and the
	// replacement a rebuild makes from it) as belonging to this run.
	run.check("RenameVmwareServer",
		client.RenameVmwareServer(ctx, serverID,
			&entities.VmwareRenameServerRequest{Name: "sdk-ex-srv-renamed-" + stamp}),
		"display name changed (synchronous, no task)")

	// The guest hostname is stored upper-cased, so it will not read back
	// exactly as sent.
	renamed, err := client.ChangeVmwareServerComputerNameAndWait(ctx, serverID,
		&entities.VmwareComputerNameRequest{ComputerName: "sdk-ex-host", ForceCustomization: boolPtr(false)})
	if run.check("ChangeVmwareServerComputerNameAndWait", err, "computer_name=%s", computerNameOf(renamed)) {
		step("sent \"sdk-ex-host\", read back %q — the backend upper-cases it", computerNameOf(renamed))
	}
	if t, err := client.ChangeVmwareServerComputerName(ctx, serverID,
		&entities.VmwareComputerNameRequest{ComputerName: "sdk-ex-host2"}); run.check("ChangeVmwareServerComputerName", err, "task=%s", t.String()) {
		_, err = client.WaitVmwareTaskRef(ctx, t)
		run.check("WaitVmwareTaskRef(computer name)", err, "hostname changed")
	}

	// ---------- Nested virtualization ----------
	//
	// The switch power-cycles a running server; the server is off at this point, so
	// it stays off. Both forms are exercised on each direction, and the raw call
	// deliberately repeats the state the ...AndWait form has just reached: that is
	// the idempotent outcome, answered with no task at all.
	section("Nested hypervisor")
	if !env.Location.NestedHypervisorSupported {
		step("location %s reports nested_hypervisor_supported=false — the switch is expected to be refused",
			env.Location.TechTitle)
	}
	nested, err := client.EnableVmwareServerNestedHypervisorAndWait(ctx, serverID)
	if run.check("EnableVmwareServerNestedHypervisorAndWait", err, "nested_hypervisor=%v power=%v",
		nestedHypervisorOf(nested), powerOf(nested)) {
		if t, err := client.EnableVmwareServerNestedHypervisor(ctx, serverID); run.check(
			"EnableVmwareServerNestedHypervisor", err, "task=%q (empty = already enabled)", t.String()) {
			if t == nil {
				step("idempotent repeat: no task started, nothing to await")
			} else {
				_, err = client.WaitVmwareTaskRef(ctx, t)
				run.check("WaitVmwareTaskRef(nested hypervisor enable)", err, "nested virtualization enabled")
			}
		}
	} else {
		// The refusals worth telling apart: a GPU server, a suspended server and a
		// location whose VDCs do not offer nested virtualization at all.
		switch {
		case sdk.IsVmwareOperationNotSupportedForGpuServer(err):
			step("refused because the server has a GPU allocation — the two are mutually exclusive")
		case sdk.IsVmwareServerSuspended(err):
			step("refused because the server is suspended — resume it and retry")
		case sdk.IsVmwareNestedHypervisorNotSupportedInLocation(err):
			step("refused because no VDC available here supports nested virtualization")
		}
	}

	nested, err = client.DisableVmwareServerNestedHypervisorAndWait(ctx, serverID)
	if run.check("DisableVmwareServerNestedHypervisorAndWait", err, "nested_hypervisor=%v power=%v",
		nestedHypervisorOf(nested), powerOf(nested)) {
		if t, err := client.DisableVmwareServerNestedHypervisor(ctx, serverID); run.check(
			"DisableVmwareServerNestedHypervisor", err, "task=%q (empty = already disabled)", t.String()) {
			if t == nil {
				step("idempotent repeat: no task started, nothing to await")
			} else {
				_, err = client.WaitVmwareTaskRef(ctx, t)
				run.check("WaitVmwareTaskRef(nested hypervisor disable)", err, "nested virtualization disabled")
			}
		}
	}

	// ---------- Copy and rebuild (the long ones) ----------
	if vmwareExampleSkipLong() {
		for _, m := range []string{
			"CreateVmwareServerAndWait",
			"CopyVmwareServer", "CopyVmwareServerAndWait",
			"RebuildVmwareServer", "RebuildVmwareServerAndWait",
		} {
			run.skip(m, "VMWARE_SKIP_LONG=1")
		}
		fmt.Println("\n(unset VMWARE_SKIP_LONG to include the second create, copy and rebuild)")
		return
	}

	// The ...AndWait form of create, on a throwaway server that is then removed
	// through the raw delete plus the state-level wait. A completed delete task does
	// not guarantee the object is gone — after a rebuild it has been seen lingering
	// in "deleting" for minutes — so the wait makes the outcome unconditional.
	section("CreateVmwareServerAndWait — several minutes")
	extra, err := client.CreateVmwareServerAndWait(ctx, env.createServerRequest("sdk-ex-srv2-"+stamp))
	if run.check("CreateVmwareServerAndWait", err, "server id=%d state=%s", idOf(extra), stateOf(extra)) {
		t, err := client.DeleteVmwareServer(ctx, extra.ID)
		if run.check("DeleteVmwareServer", err, "task=%s", t.String()) {
			_, err = client.WaitVmwareTaskRef(ctx, t)
			run.check("WaitVmwareTaskRef(delete)", err, "delete task completed")
			run.check("WaitVmwareServerGone", client.WaitVmwareServerGone(ctx, extra.ID),
				"server confirmed gone (see the log for whether it was already gone or still deleting)")
		}
	}

	section("CopyVmwareServer — several minutes")
	copyOrder, err := client.CopyVmwareServer(ctx, serverID, &entities.VmwareCopyServerRequest{Name: "sdk-ex-copy-" + stamp})
	if run.check("CopyVmwareServer", err, "new server_id=%d task=%s", serverIDOf(copyOrder), taskIDOf(copyOrder)) {
		if _, err := client.WaitVmwareTaskRef(ctx, &sdk.VmwareTaskID{ID: copyOrder.TaskID}); err != nil {
			run.fail("WaitVmwareTaskRef(copy)", err)
		} else {
			run.pass("WaitVmwareTaskRef(copy)", "copy provisioned")
		}
		run.check("DeleteVmwareServerAndWait(copy)",
			client.DeleteVmwareServerAndWait(ctx, copyOrder.ServerID), "copy deleted and gone")
	}

	section("CopyVmwareServerAndWait — several minutes")
	copied, err := client.CopyVmwareServerAndWait(ctx, serverID, &entities.VmwareCopyServerRequest{Name: "sdk-ex-copy2-" + stamp})
	if run.check("CopyVmwareServerAndWait", err, "copy id=%d state=%s", idOf(copied), stateOf(copied)) {
		run.check("DeleteVmwareServerAndWait(copy2)",
			client.DeleteVmwareServerAndWait(ctx, copied.ID), "second copy deleted and gone")
	}

	// Rebuild replaces the server: a NEW id comes back and the original is scheduled
	// for deletion. Measured at ~20 minutes, so it gets its own timeout.
	//
	// The two-step form is used here precisely because the replacement exists as soon
	// as the POST returns: the teardown is pointed at the new id immediately, so a
	// failed wait cannot leave a running server behind.
	section("RebuildVmwareServer — ~20 minutes, replaces the server")
	rebuildOrder, err := client.RebuildVmwareServer(ctx, serverID, &entities.VmwareRebuildServerRequest{ImageID: env.Image.ID})
	if run.check("RebuildVmwareServer", err, "new server_id=%d task=%s", serverIDOf(rebuildOrder), taskIDOf(rebuildOrder)) {
		step("original server %d is replaced by %d and is now deleting", serverID, rebuildOrder.ServerID)
		serverID = rebuildOrder.ServerID
		if _, err := client.WaitVmwareTaskWithTimeout(ctx, rebuildOrder.TaskID, 45*time.Minute); err != nil {
			run.fail("WaitVmwareTaskWithTimeout(rebuild)", err)
		} else {
			run.pass("WaitVmwareTaskWithTimeout(rebuild)", "rebuild finished, surviving server is %d", serverID)
		}
	}

	// The ...AndWait form, on the surviving server. If its wait fails it cannot hand
	// back the new id — the SDK names it in the error instead — so the teardown falls
	// back to sweeping the example's own servers by name.
	section("RebuildVmwareServerAndWait — ~20 minutes")
	rebuilt, err := client.RebuildVmwareServerAndWait(ctx, serverID, &entities.VmwareRebuildServerRequest{ImageID: env.Image.ID})
	if run.check("RebuildVmwareServerAndWait", err, "new server id=%d state=%s", idOf(rebuilt), stateOf(rebuilt)) {
		serverID = rebuilt.ID
	} else {
		step("the replacement id is only in the error above; teardown will sweep by name")
	}
}

// sweepVmwareExampleServers deletes leftover servers from THIS run: the ones whose
// name carries both the example prefix and this run's stamp. It is the backstop for
// the one case the example cannot track by id — a rebuild whose ...AndWait form timed
// out, which creates a replacement server and returns no id for it.
//
// Matching on the stamp as well as the prefix matters: two examples (or two
// concurrent runs) share the prefix, and a sweep on the prefix alone would delete
// each other's servers.
func sweepVmwareExampleServers(ctx context.Context, client *sdk.CloudClient, locationID int, stamp string, except int) {
	servers, err := client.GetVmwareServerList(ctx, &locationID)
	if err != nil {
		step("sweep: GetVmwareServerList: %v", err)
		return
	}
	for _, s := range servers {
		if s.ID == except {
			continue
		}
		if !strings.HasPrefix(s.Name, vmwareExampleServerPrefix) || !strings.Contains(s.Name, stamp) {
			continue
		}
		step("sweep: deleting leftover server %d (%q) from this run", s.ID, s.Name)
		if err := client.DeleteVmwareServerAndWait(ctx, s.ID); err != nil {
			step("sweep: DeleteVmwareServerAndWait(%d): %v", s.ID, err)
		}
	}
}

// printVmwareServer dumps every field of a server.
func printVmwareServer(s *entities.VmwareServer) {
	step("id=%d project=%d location=%d name=%q computer_name=%s",
		s.ID, s.ProjectID, s.LocationID, s.Name, deref(s.ComputerName))
	step("state=%s power=%v cpu=%d ram=%d MB system_disk=%d MB type=%s image=%d created=%s",
		s.State, s.IsPowerOn, s.CPU, s.RamMB, s.SystemDiskMB, deref(s.SystemDiskType), s.ImageID, s.Created)
	step("vm_tools_installed=%s gpu=%s nested_hypervisor=%v nics=%d",
		derefBool(s.VmToolsInstalled), formatGPU(s.GPU), s.NestedHypervisor, len(s.NICs))
}

// formatGPU renders the optional GPU allocation.
func formatGPU(g *entities.VmwareGPU) string {
	if g == nil {
		return "<none>"
	}
	return fmt.Sprintf("model=%d vram=%d MB cards=%d", g.ModelID, g.VramMB, g.CardCount)
}

// findNICByNetwork returns the NIC attached to the given network, or nil.
func findNICByNetwork(nics []*entities.VmwareNIC, networkID int) *entities.VmwareNIC {
	for _, n := range nics {
		if n.NetworkID == networkID {
			return n
		}
	}
	return nil
}

// findPrimaryVmwareNIC returns the server's primary NIC — the one on the shared
// network it was ordered with, and the only one whose bandwidth can be changed.
func findPrimaryVmwareNIC(nics []*entities.VmwareNIC) *entities.VmwareNIC {
	for _, n := range nics {
		if n.IsPrimary {
			return n
		}
	}
	return nil
}

// findVmwareNICByID returns the NIC with the given id, or nil.
func findVmwareNICByID(nics []*entities.VmwareNIC, nicID int) *entities.VmwareNIC {
	for _, n := range nics {
		if n.ID == nicID {
			return n
		}
	}
	return nil
}

// Nil-safe accessors, so a failed step can still be printed.
func stateOf(s *entities.VmwareServer) string {
	if s == nil {
		return "<nil>"
	}
	return s.State
}

func powerOf(s *entities.VmwareServer) any {
	if s == nil {
		return "<nil>"
	}
	return s.IsPowerOn
}

func cpuOf(s *entities.VmwareServer) int {
	if s == nil {
		return 0
	}
	return s.CPU
}

func ramOf(s *entities.VmwareServer) int {
	if s == nil {
		return 0
	}
	return s.RamMB
}

func idOf(s *entities.VmwareServer) int {
	if s == nil {
		return 0
	}
	return s.ID
}

func nestedHypervisorOf(s *entities.VmwareServer) any {
	if s == nil {
		return "<nil>"
	}
	return s.NestedHypervisor
}

func computerNameOf(s *entities.VmwareServer) string {
	if s == nil {
		return "<nil>"
	}
	return deref(s.ComputerName)
}

func sizeOf(v *entities.VmwareVolume) int {
	if v == nil {
		return 0
	}
	return v.SizeMB
}

func serverIDOf(o *entities.VmwareServerOrder) int {
	if o == nil {
		return 0
	}
	return o.ServerID
}

func taskIDOf(o *entities.VmwareServerOrder) string {
	if o == nil {
		return "<nil>"
	}
	return o.TaskID
}
