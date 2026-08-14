package sdk

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

// testClient builds a client that never reaches the network: every case below
// fails validation before a request is issued.
func testClient(t *testing.T) *CloudClient {
	t.Helper()
	cfg, err := NewConfig("test-key", "https://api.example.com")
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestIsVmwareTaskID(t *testing.T) {
	cases := map[string]bool{
		"vmw198487": true,
		"vmw1":      true,
		"vmw":       true,
		"l1t2":      false,
		"dns42":     false,
		"k8s_f7":    false,
		"":          false,
		"Vmw1":      false, // the prefix is case-sensitive
	}
	for in, want := range cases {
		if got := IsVmwareTaskID(in); got != want {
			t.Errorf("IsVmwareTaskID(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestVmwareTaskIDNilSafe(t *testing.T) {
	var nilRef *VmwareTaskID
	if !nilRef.IsZero() {
		t.Error("(*VmwareTaskID)(nil).IsZero() = false, want true")
	}
	if got := nilRef.String(); got != "" {
		t.Errorf("(*VmwareTaskID)(nil).String() = %q, want empty", got)
	}

	empty := &VmwareTaskID{}
	if !empty.IsZero() {
		t.Error("&VmwareTaskID{}.IsZero() = false, want true")
	}

	set := &VmwareTaskID{ID: "vmw1"}
	if set.IsZero() {
		t.Error("&VmwareTaskID{ID: \"vmw1\"}.IsZero() = true, want false")
	}
	if got := set.String(); got != "vmw1" {
		t.Errorf("String() = %q, want %q", got, "vmw1")
	}
}

// The two task families share GET /tasks/{id} but answer with different bodies,
// so each accessor must refuse the other's id instead of misdecoding it.
func TestTaskIDSpacesAreKeptApart(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()

	if _, err := c.GetTask(ctx, "vmw198487"); err == nil {
		t.Error("GetTask accepted a VMware task id, want an error")
	} else if !strings.Contains(err.Error(), "GetVmwareTask") {
		t.Errorf("GetTask error should point at GetVmwareTask, got: %v", err)
	}

	for _, id := range []string{"l1t2", "dns42"} {
		if _, err := c.GetVmwareTask(ctx, id); err == nil {
			t.Errorf("GetVmwareTask(%q) accepted a base task id, want an error", id)
		} else if !strings.Contains(err.Error(), "GetTask") {
			t.Errorf("GetVmwareTask(%q) error should point at GetTask, got: %v", id, err)
		}
	}

	if _, err := c.WaitVmwareTaskWithTimeout(ctx, "l1t2", time.Second); err == nil {
		t.Error("WaitVmwareTaskWithTimeout accepted a base task id, want an error")
	}
	if _, err := c.GetVmwareTask(ctx, ""); err == nil {
		t.Error("GetVmwareTask accepted an empty id, want an error")
	}
}

// A no-op mutation is reported as a nil reference; awaiting it must be a no-op
// rather than an error or a hang.
func TestWaitVmwareTaskRefTolerationOfNoTask(t *testing.T) {
	c := testClient(t)
	for _, ref := range []*VmwareTaskID{nil, {}} {
		task, err := c.WaitVmwareTaskRef(context.Background(), ref)
		if err != nil {
			t.Errorf("WaitVmwareTaskRef(%v) = error %v, want nil", ref, err)
		}
		if task != nil {
			t.Errorf("WaitVmwareTaskRef(%v) returned a task, want nil", ref)
		}
	}
}

// The wait floor is a minimum, not a value: a larger configured timeout wins, a
// smaller one is ignored.
func TestVmwareWaitFloor(t *testing.T) {
	cases := []struct {
		configured time.Duration
		want       time.Duration
	}{
		{configured: time.Minute, want: VmwareTaskWaitDefaultTimeout},
		{configured: VmwareTaskWaitDefaultTimeout, want: VmwareTaskWaitDefaultTimeout},
		{configured: 2 * VmwareTaskWaitDefaultTimeout, want: 2 * VmwareTaskWaitDefaultTimeout},
	}
	for _, tc := range cases {
		cfg, err := NewConfig("k", "https://api.example.com", WithPollingTimeout(tc.configured))
		if err != nil {
			t.Fatalf("NewConfig: %v", err)
		}
		c, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		got := c.config.PollingTimeout
		if got < VmwareTaskWaitDefaultTimeout {
			got = VmwareTaskWaitDefaultTimeout
		}
		if got != tc.want {
			t.Errorf("configured %v: effective VMware wait = %v, want %v", tc.configured, got, tc.want)
		}
	}
}

func TestBuildVmwarePaths(t *testing.T) {
	cases := []struct {
		got, want string
	}{
		{buildVmwareServerPath(7), "vmware/servers/7"},
		{buildVmwareServerPath(7, "volumes"), "vmware/servers/7/volumes"},
		{buildVmwareServerPath(7, "volumes", "9"), "vmware/servers/7/volumes/9"},
		{buildVmwareServerPath(7, "power", "on"), "vmware/servers/7/power/on"},
		{buildVmwareNetworkPath(3), "vmware/networks/3"},
		{buildVmwareNetworkPath(3, "servers"), "vmware/networks/3/servers"},
		{buildVmwareEdgePath(3, "firewall"), "vmware/networks/3/edge/firewall"},
		{buildVmwareEdgePath(3, "nat", "663"), "vmware/networks/3/edge/nat/663"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("path = %q, want %q", tc.got, tc.want)
		}
	}
}

// An empty value of a declared query parameter makes the API answer HTTP 500, so
// the SDK must omit the parameter entirely rather than send it empty.
func TestWithQueryOmitsEmptyParams(t *testing.T) {
	if got := withQuery("vmware/images", url.Values{}); got != "vmware/images" {
		t.Errorf("withQuery with no params = %q, want the bare path", got)
	}
	if got := withQuery("vmware/images", nil); got != "vmware/images" {
		t.Errorf("withQuery(nil) = %q, want the bare path", got)
	}
	params := url.Values{}
	params.Set("location_id", "5")
	if got := withQuery("vmware/images", params); got != "vmware/images?location_id=5" {
		t.Errorf("withQuery = %q", got)
	}
}

// Every method validates its ids before issuing a request, so a zero or negative
// id is reported locally instead of hitting /vmware/servers/0.
func TestVmwareMethodsRejectNonPositiveIDs(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()

	serverCalls := map[string]func(int) error{
		"GetVmwareServer":         func(id int) error { _, err := c.GetVmwareServer(ctx, id); return err },
		"DeleteVmwareServer":      func(id int) error { _, err := c.DeleteVmwareServer(ctx, id); return err },
		"GetVmwareServerVolumes":  func(id int) error { _, err := c.GetVmwareServerVolumes(ctx, id); return err },
		"GetVmwareServerNICs":     func(id int) error { _, err := c.GetVmwareServerNICs(ctx, id); return err },
		"GetVmwareSnapshot":       func(id int) error { _, err := c.GetVmwareSnapshot(ctx, id); return err },
		"GetVmwareServerFirewall": func(id int) error { _, err := c.GetVmwareServerFirewall(ctx, id); return err },
		"PowerOnVmwareServer":     func(id int) error { _, err := c.PowerOnVmwareServer(ctx, id); return err },
		"RestoreVmwareSnapshot":   func(id int) error { _, err := c.RestoreVmwareSnapshot(ctx, id); return err },
		"WaitVmwareServerGone":    func(id int) error { return c.WaitVmwareServerGone(ctx, id) },
	}
	networkCalls := map[string]func(int) error{
		"GetVmwareNetwork":       func(id int) error { _, err := c.GetVmwareNetwork(ctx, id); return err },
		"DeleteVmwareNetwork":    func(id int) error { _, err := c.DeleteVmwareNetwork(ctx, id); return err },
		"GetVmwareEdgeFirewall":  func(id int) error { _, err := c.GetVmwareEdgeFirewall(ctx, id); return err },
		"GetVmwareEdgeNAT":       func(id int) error { _, err := c.GetVmwareEdgeNAT(ctx, id); return err },
		"GetVmwareEdgeVPN":       func(id int) error { _, err := c.GetVmwareEdgeVPN(ctx, id); return err },
		"WaitVmwareNetworkGone":  func(id int) error { return c.WaitVmwareNetworkGone(ctx, id) },
		"WaitVmwareNetworkState": func(id int) error { _, err := c.WaitVmwareNetworkState(ctx, id, "active"); return err },
	}
	for _, group := range []map[string]func(int) error{serverCalls, networkCalls} {
		for name, call := range group {
			for _, id := range []int{0, -1} {
				if err := call(id); err == nil {
					t.Errorf("%s(%d) returned no error, want a local validation failure", name, id)
				}
			}
		}
	}

	// Sub-resource ids are validated too, not just the parent.
	if _, err := c.GetVmwareVolume(ctx, 1, 0); err == nil {
		t.Error("GetVmwareVolume with volumeID 0 returned no error")
	}
	if _, err := c.DeleteVmwareNIC(ctx, 1, 0); err == nil {
		t.Error("DeleteVmwareNIC with nicID 0 returned no error")
	}
	if _, err := c.DeleteVmwareEdgeNATRule(ctx, 1, 0); err == nil {
		t.Error("DeleteVmwareEdgeNATRule with ruleID 0 returned no error")
	}
	if _, err := c.DeleteVmwareEdgeVPNTunnel(ctx, 1, 0); err == nil {
		t.Error("DeleteVmwareEdgeVPNTunnel with tunnelID 0 returned no error")
	}

	// An optional location filter is validated only when it is set.
	zero := 0
	if _, err := c.GetVmwareServerList(ctx, &zero); err == nil {
		t.Error("GetVmwareServerList(&0) returned no error")
	}
	if _, err := c.GetVmwareImageList(ctx, &zero, nil); err == nil {
		t.Error("GetVmwareImageList(&0) returned no error")
	}
}

// A nil request must be reported rather than marshalled into a null body.
func TestVmwareMutatorsRejectNilRequests(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()

	if _, err := c.CreateVmwareServer(ctx, nil); err == nil {
		t.Error("CreateVmwareServer(nil) returned no error")
	}
	if err := c.VerifyVmwareServer(ctx, nil); err == nil {
		t.Error("VerifyVmwareServer(nil) returned no error")
	}
	if _, err := c.CreateVmwareIsolatedNetwork(ctx, nil); err == nil {
		t.Error("CreateVmwareIsolatedNetwork(nil) returned no error")
	}
	if _, err := c.EditVmwareNetwork(ctx, 1, nil); err == nil {
		t.Error("EditVmwareNetwork(nil) returned no error")
	}
	if _, err := c.UpdateVmwareServerFirewall(ctx, 1, nil); err == nil {
		t.Error("UpdateVmwareServerFirewall(nil) returned no error")
	}
	if _, err := c.UpsertVmwareEdgeNATRule(ctx, 1, nil); err == nil {
		t.Error("UpsertVmwareEdgeNATRule(nil) returned no error")
	}
}

func TestVmwareErrorHelpers(t *testing.T) {
	cases := []struct {
		name  string
		code  int
		check func(error) bool
	}{
		{"IsVmwareLocationNotFound", APICodeVmwareLocationNotFound, IsVmwareLocationNotFound},
		{"IsVmwareNoFreePublicNetwork", APICodeVmwareNoFreePublicNetwork, IsVmwareNoFreePublicNetwork},
		{"IsNetworkInUse", APICodeNetworkInUse, IsNetworkInUse},
		{"IsConflict", APICodeConflict, IsConflict},
	}
	for _, tc := range cases {
		match := &RequestError{StatusCode: 400, Codes: []int{tc.code}}
		if !tc.check(match) {
			t.Errorf("%s did not recognize code %d", tc.name, tc.code)
		}
		other := &RequestError{StatusCode: 400, Codes: []int{-1}}
		if tc.check(other) {
			t.Errorf("%s matched an unrelated code", tc.name)
		}
		if tc.check(nil) {
			t.Errorf("%s(nil) = true, want false", tc.name)
		}
	}

	// "Location not found" arrives as HTTP 400, so IsNotFound must not claim it.
	locationGone := &RequestError{StatusCode: 400, Codes: []int{APICodeVmwareLocationNotFound}}
	if IsNotFound(locationGone) {
		t.Error("IsNotFound matched a 400 location-not-found; it should stay 404-only")
	}
}
