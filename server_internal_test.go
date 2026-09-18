package sdk

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// An installation always sends id, state, addresses and components; the sign-in,
// the key and the reason of a failure are omitted when empty.
func TestParseServerApplicationInstallations(t *testing.T) {
	body := []byte(`{"server":{
		"id":"l1s7","location_id":"dc-1","state":"Active","application_ids":["n8n","wordpress"],
		"application_installations":[
			{
				"id":"n8n","state":"Installed","installed_at":"2026-09-18T08:12:44Z",
				"addresses":[
					{"service":"web","address":"https://10.0.0.1"},
					{"service":"webhook","address":"https://10.0.0.1/webhook"}
				],
				"components":[
					{"name":"web","kind":"Web","observed_status":"running","address":"https://10.0.0.1"},
					{"name":"db","kind":"Database"}
				],
				"app_login":"admin","app_password":"s3cret","llm_key_id":"llm-42"
			},
			{
				"id":"wordpress","state":"Failed","outcome_reason":"DeployTimeout",
				"installed_at":"2026-09-18T08:20:01Z","addresses":[],"components":[]
			}
		]
	}}`)

	var resp GetServerResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Server == nil {
		t.Fatal("server must be parsed")
	}
	if len(resp.Server.ApplicationInstallations) != 2 {
		t.Fatalf("got %d installations, want 2", len(resp.Server.ApplicationInstallations))
	}

	installed := resp.Server.ApplicationInstallations[0]
	if installed.ID != "n8n" || installed.State != entities.ApplicationInstallStateInstalled {
		t.Errorf("identity/state: %+v", installed)
	}
	if installed.OutcomeReason != "" {
		t.Errorf("outcome_reason = %q, want empty for a successful installation", installed.OutcomeReason)
	}
	if installed.InstalledAt != "2026-09-18T08:12:44Z" {
		t.Errorf("installed_at = %q", installed.InstalledAt)
	}
	// The sign-in of the application is the reason to read the installation at
	// all: it is issued during the install and served nowhere else.
	if installed.AppLogin != "admin" || installed.AppPassword != "s3cret" || installed.LLMKeyID != "llm-42" {
		t.Errorf("sign-in and key: %+v", installed)
	}

	wantAddresses := []entities.ApplicationAddress{
		{Service: "web", Address: "https://10.0.0.1"},
		{Service: "webhook", Address: "https://10.0.0.1/webhook"},
	}
	if len(installed.Addresses) != len(wantAddresses) {
		t.Fatalf("got %d addresses, want %d", len(installed.Addresses), len(wantAddresses))
	}
	for i, want := range wantAddresses {
		if installed.Addresses[i] != want {
			t.Errorf("addresses[%d] = %+v, want %+v", i, installed.Addresses[i], want)
		}
	}

	wantComponents := []entities.ApplicationComponent{
		{Name: "web", Kind: entities.ApplicationComponentKindWeb, ObservedStatus: "running", Address: "https://10.0.0.1"},
		{Name: "db", Kind: entities.ApplicationComponentKindDatabase},
	}
	if len(installed.Components) != len(wantComponents) {
		t.Fatalf("got %d components, want %d", len(installed.Components), len(wantComponents))
	}
	for i, want := range wantComponents {
		if installed.Components[i] != want {
			t.Errorf("components[%d] = %+v, want %+v", i, installed.Components[i], want)
		}
	}

	// A failed installation names the reason and still reports the moment of the
	// terminal outcome.
	failed := resp.Server.ApplicationInstallations[1]
	if failed.State != entities.ApplicationInstallStateFailed {
		t.Errorf("state = %q, want %q", failed.State, entities.ApplicationInstallStateFailed)
	}
	if failed.OutcomeReason != entities.ApplicationOutcomeReasonDeployTimeout {
		t.Errorf("outcome_reason = %q, want %q", failed.OutcomeReason, entities.ApplicationOutcomeReasonDeployTimeout)
	}
	if failed.InstalledAt != "2026-09-18T08:20:01Z" {
		t.Errorf("installed_at = %q, want the moment of the terminal outcome", failed.InstalledAt)
	}
	if failed.AppLogin != "" || failed.AppPassword != "" || failed.LLMKeyID != "" {
		t.Errorf("omitted sign-in must stay empty: %+v", failed)
	}
	if len(failed.Addresses) != 0 || len(failed.Components) != 0 {
		t.Errorf("empty lists: %+v", failed)
	}
}

// The block of installations arrives in three shapes and they must stay apart:
// missing from the response, present and empty, filled. What a missing block
// means is the publisher's call and is not settled, so this fixes the
// difference between the shapes, not what any of them says.
func TestServerApplicationInstallationsShapes(t *testing.T) {
	cases := map[string]struct {
		body    string
		wantNil bool
		wantLen int
	}{
		"block missing": {
			`{"server":{"id":"l1s7","application_ids":[]}}`,
			true, 0,
		},
		"block empty": {
			`{"server":{"id":"l1s7","application_ids":[],"application_installations":[]}}`,
			false, 0,
		},
		"block filled": {
			`{"server":{"id":"l1s7","application_ids":["n8n"],"application_installations":[
				{"id":"n8n","state":"Installing","addresses":[],"components":[]}
			]}}`,
			false, 1,
		},
	}

	for name, tc := range cases {
		var resp GetServerResponse
		if err := json.Unmarshal([]byte(tc.body), &resp); err != nil {
			t.Fatalf("%s: unmarshal: %v", name, err)
		}
		if resp.Server == nil {
			t.Fatalf("%s: server must be parsed", name)
		}

		got := resp.Server.ApplicationInstallations
		if (got == nil) != tc.wantNil {
			t.Errorf("%s: ApplicationInstallations == nil is %v, want %v", name, got == nil, tc.wantNil)
		}
		if len(got) != tc.wantLen {
			t.Errorf("%s: got %d installations, want %d", name, len(got), tc.wantLen)
		}
	}
}

// The state of an installation travels as a PascalCase string, and the
// spellings are fixed by the contract, not by the SDK.
func TestApplicationInstallationClosedSetsOnTheWire(t *testing.T) {
	fixed := map[string]struct {
		got  string
		want string
	}{
		"ApplicationInstallStateInstalling":           {string(entities.ApplicationInstallStateInstalling), "Installing"},
		"ApplicationInstallStateInstalled":            {string(entities.ApplicationInstallStateInstalled), "Installed"},
		"ApplicationInstallStateFailed":               {string(entities.ApplicationInstallStateFailed), "Failed"},
		"ApplicationOutcomeReasonRegistrationFailed":  {string(entities.ApplicationOutcomeReasonRegistrationFailed), "RegistrationFailed"},
		"ApplicationOutcomeReasonDockerNotReady":      {string(entities.ApplicationOutcomeReasonDockerNotReady), "DockerNotReady"},
		"ApplicationOutcomeReasonServiceCreateFailed": {string(entities.ApplicationOutcomeReasonServiceCreateFailed), "ServiceCreateFailed"},
		"ApplicationOutcomeReasonDeployTimeout":       {string(entities.ApplicationOutcomeReasonDeployTimeout), "DeployTimeout"},
		"ApplicationOutcomeReasonHostUnreachable":     {string(entities.ApplicationOutcomeReasonHostUnreachable), "HostUnreachable"},
		"ApplicationOutcomeReasonLLMKeyValueMissing":  {string(entities.ApplicationOutcomeReasonLLMKeyValueMissing), "LlmKeyValueMissing"},
		"ApplicationComponentKindWeb":                 {string(entities.ApplicationComponentKindWeb), "Web"},
		"ApplicationComponentKindWorker":              {string(entities.ApplicationComponentKindWorker), "Worker"},
		"ApplicationComponentKindDatabase":            {string(entities.ApplicationComponentKindDatabase), "Database"},
	}
	for name, c := range fixed {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", name, c.got, c.want)
		}
	}
}

// An ordered application is refused before the request is built, and the message
// names the position in the list — that is the only way the caller can tell
// which of the applications it has to fix.
func TestCreateServerRequestValidateApplications(t *testing.T) {
	base := func() entities.CreateServerRequest {
		return entities.CreateServerRequest{
			LocationID: "dc-1",
			ImageID:    "img-ubuntu-24",
			CPU:        2,
			RamMB:      4096,
			Name:       "srv",
			Volumes:    []entities.VolumeSpec{{Name: "boot", SizeMB: 40960}},
		}
	}

	cases := map[string]struct {
		applications []entities.ApplicationSpec
		wantErr      string
	}{
		"no applications at all": {
			nil,
			"",
		},
		"application with parameters and a key": {
			[]entities.ApplicationSpec{{
				ID:          "n8n",
				Parameters:  map[string]string{"N8N_HOST": "localhost"},
				IssueLLMKey: true,
			}},
			"",
		},
		"application without id": {
			[]entities.ApplicationSpec{{Parameters: map[string]string{"N8N_HOST": "localhost"}}},
			"applications[0]: id is required",
		},
		"second application without id": {
			[]entities.ApplicationSpec{{ID: "n8n"}, {}},
			"applications[1]: id is required",
		},
		"parameter without a name": {
			[]entities.ApplicationSpec{{ID: "n8n", Parameters: map[string]string{"": "localhost"}}},
			"applications[0]: parameter name is required",
		},
		"second application's parameter without a name": {
			[]entities.ApplicationSpec{
				{ID: "n8n"},
				{ID: "wordpress", Parameters: map[string]string{"": "localhost"}},
			},
			"applications[1]: parameter name is required",
		},
	}

	for name, tc := range cases {
		r := base()
		r.Applications = tc.applications

		err := r.Validate()
		switch {
		case tc.wantErr == "":
			if err != nil {
				t.Errorf("%s: Validate() = %v, want nil", name, err)
			}
		case err == nil:
			t.Errorf("%s: Validate() = nil, want %q", name, tc.wantErr)
		case err.Error() != tc.wantErr:
			t.Errorf("%s: Validate() = %q, want %q", name, err.Error(), tc.wantErr)
		}
	}
}

// The order carries an application as id + issue_llm_key and adds the parameters
// only when there are any; the optional fields of the order itself stay out of
// the body while they are empty.
func TestCreateServerSendsApplications(t *testing.T) {
	var sent atomic.Value

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		sent.Store(raw)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"task_id":"l1t9"}`))
	}))

	req := &entities.CreateServerRequest{
		LocationID: "dc-1",
		ImageID:    "img-ubuntu-24",
		CPU:        2,
		RamMB:      4096,
		Name:       "srv",
		Volumes:    []entities.VolumeSpec{{Name: "boot", SizeMB: 40960}},
		Networks:   []entities.NetworkSpec{{BandwidthMbps: 10}},
		Applications: []entities.ApplicationSpec{
			{ID: "n8n", Parameters: map[string]string{"N8N_HOST": "localhost"}, IssueLLMKey: true},
			{ID: "wordpress"},
		},
	}

	if _, err := client.CreateServer(context.Background(), req); err != nil {
		t.Fatalf("CreateServer: %v", err)
	}

	raw, ok := sent.Load().([]byte)
	if !ok {
		t.Fatal("the order must reach the server")
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal order: %v", err)
	}
	for _, field := range []string{"application_ids", "ssh_key_ids", "tags", "affinity_group_id", "server_init_script"} {
		if _, ok := body[field]; ok {
			t.Errorf("%q must stay out of the order while it is empty, got %s", field, raw)
		}
	}
	if _, ok := body["applications"]; !ok {
		t.Fatalf("the order must carry applications, got %s", raw)
	}

	var applications []map[string]json.RawMessage
	if err := json.Unmarshal(body["applications"], &applications); err != nil {
		t.Fatalf("unmarshal applications: %v", err)
	}
	if len(applications) != 2 {
		t.Fatalf("got %d applications, want 2", len(applications))
	}

	if got, want := fieldNames(applications[0]), "id,issue_llm_key,parameters"; got != want {
		t.Errorf("application with parameters carries %q, want %q", got, want)
	}
	if got, want := string(applications[0]["id"]), `"n8n"`; got != want {
		t.Errorf("id = %s, want %s", got, want)
	}
	if got, want := string(applications[0]["parameters"]), `{"N8N_HOST":"localhost"}`; got != want {
		t.Errorf("parameters = %s, want %s", got, want)
	}
	if got, want := string(applications[0]["issue_llm_key"]), "true"; got != want {
		t.Errorf("issue_llm_key = %s, want %s", got, want)
	}

	// Without parameters the field disappears, but the key flag is still sent:
	// the publisher reads it as "no key asked for", not as "not specified".
	if got, want := fieldNames(applications[1]), "id,issue_llm_key"; got != want {
		t.Errorf("application without parameters carries %q, want %q", got, want)
	}
	if got, want := string(applications[1]["issue_llm_key"]), "false"; got != want {
		t.Errorf("issue_llm_key = %s, want %s", got, want)
	}
}

func fieldNames(value map[string]json.RawMessage) string {
	names := make([]string, 0, len(value))
	for name := range value {
		names = append(names, name)
	}
	sort.Strings(names)

	return strings.Join(names, ",")
}
