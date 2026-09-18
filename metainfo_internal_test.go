package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

// The catalog always sends id, location_id, images, llm_key_enabled and
// parameters; every other field is omitted when empty, so an entry arrives in
// either of the two shapes below.
func TestParseApplicationsResponse(t *testing.T) {
	body := []byte(`{"applications":[
		{
			"id":"n8n","location_id":"dc-1","images":["img-ubuntu-24","img-debian-12"],
			"category":"Automation","documentation_url":"https://docs.n8n.io",
			"credentials_mode":"ServicePassword","llm_key_enabled":true,
			"recommended_cpu":2,"recommended_ram_mb":4096,"recommended_storage_mb":40960,
			"parameters":[
				{"name":"N8N_ENCRYPTION_KEY","required":true,"secret":true},
				{"name":"N8N_HOST","required":false,"default":"localhost","secret":false},
				{"name":"N8N_LLM_KEY","required":true,"secret":true,"llm_kind":"Key"},
				{"name":"N8N_LLM_ENDPOINT","required":true,"secret":false,"llm_kind":"Endpoint"}
			]
		},
		{"id":"wordpress","location_id":"dc-1","images":["img-ubuntu-24"],"llm_key_enabled":false,"parameters":[]}
	]}`)

	var resp ListApplicationsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Applications) != 2 {
		t.Fatalf("got %d applications, want 2", len(resp.Applications))
	}

	app := resp.Applications[0]
	if app.ID != "n8n" || app.LocationID != "dc-1" {
		t.Errorf("identity: %+v", app)
	}
	// images is what the order picks the template out of, so the catalog
	// extension must not cost it.
	if len(app.Images) != 2 || app.Images[0] != "img-ubuntu-24" || app.Images[1] != "img-debian-12" {
		t.Errorf("images = %v, want [img-ubuntu-24 img-debian-12]", app.Images)
	}
	if app.Category != "Automation" || app.DocumentationURL != "https://docs.n8n.io" {
		t.Errorf("category/documentation: %+v", app)
	}
	if app.CredentialsMode != entities.ApplicationCredentialsModeServicePassword {
		t.Errorf("credentials_mode = %q, want %q", app.CredentialsMode, entities.ApplicationCredentialsModeServicePassword)
	}
	if !app.LLMKeyEnabled {
		t.Error("llm_key_enabled = false for an application the catalog offers a key for")
	}
	if app.RecommendedCPU != 2 || app.RecommendedRamMB != 4096 || app.RecommendedStorageMB != 40960 {
		t.Errorf("recommended resources: %+v", app)
	}

	// The parameters name what the order has to carry: which value is required,
	// which one must not be echoed back and which one belongs to the model access.
	wantParameters := []entities.ApplicationParameter{
		{Name: "N8N_ENCRYPTION_KEY", Required: true, Secret: true},
		{Name: "N8N_HOST", Default: "localhost"},
		{Name: "N8N_LLM_KEY", Required: true, Secret: true, LLMKind: entities.ApplicationLLMKindKey},
		{Name: "N8N_LLM_ENDPOINT", Required: true, LLMKind: entities.ApplicationLLMKindEndpoint},
	}
	if len(app.Parameters) != len(wantParameters) {
		t.Fatalf("got %d parameters, want %d", len(app.Parameters), len(wantParameters))
	}
	for i, want := range wantParameters {
		if app.Parameters[i] != want {
			t.Errorf("parameters[%d] = %+v, want %+v", i, app.Parameters[i], want)
		}
	}

	// The minimal shape: every optional field is missing, and that must decode
	// to zero values rather than fail.
	bare := resp.Applications[1]
	if bare.Category != "" || bare.DocumentationURL != "" || bare.CredentialsMode != "" {
		t.Errorf("omitted strings must stay empty: %+v", bare)
	}
	if bare.RecommendedCPU != 0 || bare.RecommendedRamMB != 0 || bare.RecommendedStorageMB != 0 {
		t.Errorf("omitted recommendations must stay zero: %+v", bare)
	}
	if bare.LLMKeyEnabled {
		t.Error("llm_key_enabled = true for an application that is not offered a key")
	}
	if len(bare.Parameters) != 0 {
		t.Errorf("got %d parameters, want 0", len(bare.Parameters))
	}
	if len(bare.Images) != 1 || bare.Images[0] != "img-ubuntu-24" {
		t.Errorf("images = %v, want [img-ubuntu-24]", bare.Images)
	}
}

// The catalog of a project is read by location: the filter is a query parameter
// of the same endpoint, and without it the call stays a plain GET on the
// collection. A project with no catalog answers without the collection at all,
// and that must read as an empty list rather than fail.
func TestGetApplicationsRequest(t *testing.T) {
	var lastRequest atomic.Value

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastRequest.Store(r.Method + " " + r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("location_id") == "" {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		_, _ = w.Write([]byte(`{"applications":[
			{"id":"n8n","location_id":"dc-1","images":["img-ubuntu-24"],"llm_key_enabled":true,
			 "parameters":[{"name":"N8N_HOST","required":false,"default":"localhost"}]}
		]}`))
	}))
	ctx := context.Background()

	applications, err := client.GetApplications(ctx, "dc-1")
	if err != nil {
		t.Fatalf("GetApplications: %v", err)
	}
	if got, want := lastRequest.Load(), "GET /api/v1/applications?location_id=dc-1"; got != want {
		t.Errorf("filtered call sent %v, want %q", got, want)
	}
	if len(applications) != 1 {
		t.Fatalf("got %d applications, want 1", len(applications))
	}
	app := applications[0]
	if app.ID != "n8n" || app.LocationID != "dc-1" || !app.LLMKeyEnabled {
		t.Errorf("application = %+v", app)
	}
	if len(app.Parameters) != 1 {
		t.Fatalf("got %d parameters, want 1", len(app.Parameters))
	}
	if want := (entities.ApplicationParameter{Name: "N8N_HOST", Default: "localhost"}); app.Parameters[0] != want {
		t.Errorf("parameters[0] = %+v, want %+v", app.Parameters[0], want)
	}

	applications, err = client.GetApplications(ctx, "")
	if err != nil {
		t.Fatalf("GetApplications without a filter: %v", err)
	}
	if got, want := lastRequest.Load(), "GET /api/v1/applications"; got != want {
		t.Errorf("unfiltered call sent %v, want %q", got, want)
	}
	if len(applications) != 0 {
		t.Errorf("got %d applications, want 0", len(applications))
	}
}

// A refused catalog request must reach the caller as an error carrying the
// status and the code of the API — an empty catalog and "the request failed"
// are the same value otherwise.
func TestGetApplicationsAPIError(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"code":-12030,"message":"bad request"}]}`))
	}))

	applications, err := client.GetApplications(context.Background(), "dc-1")
	if err == nil {
		t.Fatal("a refused catalog request must return an error")
	}
	if applications != nil {
		t.Errorf("a refused catalog request must not return a catalog, got %+v", applications)
	}

	var re *RequestError
	if !errors.As(err, &re) {
		t.Fatalf("the API error must survive the wrapping: %v", err)
	}
	if re.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", re.StatusCode, http.StatusBadRequest)
	}
	if !HasAPICode(err, -12030) {
		t.Errorf("the code of the API must reach the caller: %v", err)
	}
}
