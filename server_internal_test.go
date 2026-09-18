package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// The applications block of an order is refused with four distinct codes and the
// caller acts on each of them differently: name another application, drop the key
// flag, fill the value in, remove the parameter. Each predicate must therefore
// answer for its own code only, and the codes themselves are fixed by the
// contract, not by the SDK.
func TestApplicationOrderErrorPredicates(t *testing.T) {
	fixed := map[string]struct {
		got  int
		want int
	}{
		"APICodeApplicationNotFound":                {APICodeApplicationNotFound, -19053},
		"APICodeApplicationLLMKeyDisabled":          {APICodeApplicationLLMKeyDisabled, -19968},
		"APICodeApplicationRequiredParameterNotSet": {APICodeApplicationRequiredParameterNotSet, -19969},
		"APICodeApplicationParameterNotDeclared":    {APICodeApplicationParameterNotDeclared, -19981},
	}
	for name, c := range fixed {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", name, c.got, c.want)
		}
	}

	predicates := map[string]struct {
		match func(error) bool
		code  int
	}{
		"IsApplicationNotFound":                {IsApplicationNotFound, APICodeApplicationNotFound},
		"IsApplicationLLMKeyDisabled":          {IsApplicationLLMKeyDisabled, APICodeApplicationLLMKeyDisabled},
		"IsApplicationRequiredParameterNotSet": {IsApplicationRequiredParameterNotSet, APICodeApplicationRequiredParameterNotSet},
		"IsApplicationParameterNotDeclared":    {IsApplicationParameterNotDeclared, APICodeApplicationParameterNotDeclared},
	}
	probed := []int{
		APICodeApplicationNotFound,
		APICodeApplicationLLMKeyDisabled,
		APICodeApplicationRequiredParameterNotSet,
		APICodeApplicationParameterNotDeclared,
		APICodeConflict,
	}

	for name, p := range predicates {
		for _, code := range probed {
			// The order is refused before anything is created, so the codes arrive
			// with HTTP 400, and the methods wrap the error with %w.
			err := fmt.Errorf("failed to create server: %w", &RequestError{
				Status:     "400 Bad Request",
				StatusCode: http.StatusBadRequest,
				Codes:      []int{code},
			})
			want := code == p.code
			if got := p.match(err); got != want {
				t.Errorf("%s(code %d) = %v, want %v", name, code, got, want)
			}
		}
		if p.match(nil) {
			t.Errorf("%s(nil) must be false", name)
		}
		if p.match(errors.New("dial tcp: connection refused")) {
			t.Errorf("%s must be false for an error that is not an API error", name)
		}
	}

	// An application the catalog does not offer is refused with HTTP 400, not 404:
	// the two not-founds must not answer for each other.
	unknownApplication := &RequestError{
		Status:     "400 Bad Request",
		StatusCode: http.StatusBadRequest,
		Codes:      []int{APICodeApplicationNotFound},
	}
	if IsNotFound(unknownApplication) {
		t.Error("IsNotFound must not cover the 400 an unknown application is refused with")
	}
	missingServer := &RequestError{Status: "404 Not Found", StatusCode: http.StatusNotFound}
	if IsApplicationNotFound(missingServer) {
		t.Error("IsApplicationNotFound must not cover a plain 404")
	}
}

// A refused order must reach the caller as an error the predicates work on, and
// the parametric refusals must bring the names of the parameters at fault along:
// error_params is the only thing that says which of the values sent to fix.
func TestCreateServerApplicationParameterRefusal(t *testing.T) {
	cases := map[string]struct {
		body           string
		match          func(error) bool
		wantParameters []string
	}{
		"required parameter has no value": {
			`{"errors":[
				{"code":-19969,"message":"Required application parameter 'N8N_ENCRYPTION_KEY' has no value","error_params":[{"name":"Parameter","value":"N8N_ENCRYPTION_KEY"}]},
				{"code":-19969,"message":"Required application parameter 'N8N_HOST' has no value","error_params":[{"name":"Parameter","value":"N8N_HOST"}]}
			]}`,
			IsApplicationRequiredParameterNotSet,
			[]string{"N8N_ENCRYPTION_KEY", "N8N_HOST"},
		},
		"parameter is not declared": {
			`{"errors":[
				{"code":-19981,"message":"Application parameter 'N8N_TIMEZONE' is not declared by the application","error_params":[{"name":"Parameter","value":"N8N_TIMEZONE"}]}
			]}`,
			IsApplicationParameterNotDeclared,
			[]string{"N8N_TIMEZONE"},
		},
	}

	for name, tc := range cases {
		client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(tc.body))
		}))

		task, err := client.CreateServer(context.Background(), &entities.CreateServerRequest{
			LocationID:   "dc-1",
			ImageID:      "img-ubuntu-24",
			CPU:          2,
			RamMB:        4096,
			Name:         "srv",
			Volumes:      []entities.VolumeSpec{{Name: "boot", SizeMB: 40960}},
			Applications: []entities.ApplicationSpec{{ID: "n8n"}},
		})
		if err == nil {
			t.Fatalf("%s: a refused order must return an error", name)
		}
		if task != nil {
			t.Errorf("%s: a refused order must not return a task, got %+v", name, task)
		}
		if !tc.match(err) {
			t.Errorf("%s: the predicate must answer for the error the method returns: %v", name, err)
		}

		var re *RequestError
		if !errors.As(err, &re) {
			t.Fatalf("%s: the API error must survive the wrapping: %v", name, err)
		}
		got := applicationParameterNames(re)
		if len(got) != len(tc.wantParameters) {
			t.Fatalf("%s: got %d parameter name(s) %v, want %d %v", name, len(got), got, len(tc.wantParameters), tc.wantParameters)
		}
		for i, want := range tc.wantParameters {
			if got[i] != want {
				t.Errorf("%s: parameter[%d] = %q, want %q", name, i, got[i], want)
			}
		}
	}
}

// applicationParameterNames collects the parameters a refusal names in
// error_params under "Parameter", in the order the API listed them.
func applicationParameterNames(err *RequestError) []string {
	names := make([]string, 0, len(err.ErrorParams))
	for _, param := range err.ErrorParams {
		if param.Name != "Parameter" {
			continue
		}
		if name, ok := param.Value.(string); ok {
			names = append(names, name)
		}
	}

	return names
}

// Reading a server is how the caller learns what the install produced — the
// sign-in and the addresses are served nowhere else — so the block has to
// survive the methods, not only json.Unmarshal.
func TestServerMethodsServeApplicationInstallations(t *testing.T) {
	const installation = `"application_installations":[{
		"id":"n8n","state":"Installed","installed_at":"2026-09-18T08:12:44Z",
		"addresses":[{"service":"web","address":"https://10.0.0.1"}],
		"components":[{"name":"web","kind":"Web","observed_status":"running"}],
		"app_login":"admin","app_password":"s3cret"
	}]`

	var lastRequest atomic.Value

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastRequest.Store(r.Method + " " + r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/servers/l1s7":
			_, _ = w.Write([]byte(`{"server":{"id":"l1s7","state":"Active","application_ids":["n8n"],` + installation + `}}`))
		case "/api/v1/servers":
			_, _ = w.Write([]byte(`{"servers":[{"id":"l1s7","state":"Active","application_ids":["n8n"],` + installation + `}]}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.RequestURI())
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	ctx := context.Background()

	server, err := client.GetServer(ctx, "l1s7")
	if err != nil {
		t.Fatalf("GetServer: %v", err)
	}
	if got, want := lastRequest.Load(), "GET /api/v1/servers/l1s7"; got != want {
		t.Errorf("GetServer sent %v, want %q", got, want)
	}
	assertInstalledApplication(t, "GetServer", server)

	servers, err := client.GetServerList(ctx)
	if err != nil {
		t.Fatalf("GetServerList: %v", err)
	}
	if got, want := lastRequest.Load(), "GET /api/v1/servers"; got != want {
		t.Errorf("GetServerList sent %v, want %q", got, want)
	}
	if len(servers) != 1 {
		t.Fatalf("got %d servers, want 1", len(servers))
	}
	if servers[0] == nil {
		t.Fatal("GetServerList: the server must be parsed")
	}
	assertInstalledApplication(t, "GetServerList", servers[0])
}

func assertInstalledApplication(t *testing.T, method string, server *entities.Server) {
	t.Helper()

	if len(server.ApplicationInstallations) != 1 {
		t.Fatalf("%s: got %d installations, want 1", method, len(server.ApplicationInstallations))
	}
	installed := server.ApplicationInstallations[0]
	if installed.ID != "n8n" || installed.State != entities.ApplicationInstallStateInstalled {
		t.Errorf("%s: identity/state: %+v", method, installed)
	}
	if installed.AppLogin != "admin" || installed.AppPassword != "s3cret" {
		t.Errorf("%s: sign-in: %+v", method, installed)
	}
	if len(installed.Addresses) != 1 {
		t.Fatalf("%s: got %d addresses, want 1", method, len(installed.Addresses))
	}
	if want := (entities.ApplicationAddress{Service: "web", Address: "https://10.0.0.1"}); installed.Addresses[0] != want {
		t.Errorf("%s: addresses[0] = %+v, want %+v", method, installed.Addresses[0], want)
	}
	if len(installed.Components) != 1 {
		t.Fatalf("%s: got %d components, want 1", method, len(installed.Components))
	}
	if installed.Components[0].Kind != entities.ApplicationComponentKindWeb {
		t.Errorf("%s: components[0] = %+v", method, installed.Components[0])
	}
}

// A server that cannot be read must be tellable from one that has no
// installations: both the 404 of the API and a 200 without a server are a
// not-found, and an empty id never reaches the network at all.
func TestGetServerRefusals(t *testing.T) {
	var requests int32

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/servers/gone" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"server not found"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	ctx := context.Background()

	if _, err := client.GetServer(ctx, ""); err == nil {
		t.Error("an empty server id must be refused")
	}
	if got := atomic.LoadInt32(&requests); got != 0 {
		t.Errorf("an empty server id must be refused before the request, got %d request(s)", got)
	}

	if _, err := client.GetServer(ctx, "gone"); !IsNotFound(err) {
		t.Errorf("IsNotFound = false for the 404 GetServer returns: %v", err)
	}

	if _, err := client.GetServer(ctx, "l1s7"); !IsNotFound(err) {
		t.Errorf("IsNotFound = false for a response that carries no server: %v", err)
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
