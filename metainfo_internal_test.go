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
// collection. A response that carries no collection must read as an empty list
// rather than fail.
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

func TestBuildProjectConsumptionPath(t *testing.T) {
	cases := map[string]struct {
		sub     string
		period  *entities.ConsumptionPeriodRequest
		service entities.ConsumptionService
		want    string
	}{
		"defaults":             {"", nil, "", "project/consumption"},
		"empty period":         {"", &entities.ConsumptionPeriodRequest{}, "", "project/consumption"},
		"period":               {"", &entities.ConsumptionPeriodRequest{From: "2026-10-01", To: "2026-10-31"}, "", "project/consumption?from=2026-10-01&to=2026-10-31"},
		"resources filtered":   {"resources", &entities.ConsumptionPeriodRequest{From: "2026-10-01", To: "2026-10-31"}, entities.ConsumptionServiceVstack, "project/consumption/resources?from=2026-10-01&service=vstack&to=2026-10-31"},
		"resources unfiltered": {"resources", nil, "", "project/consumption/resources"},
		"daily to only":        {"daily", &entities.ConsumptionPeriodRequest{To: "2026-10-31"}, "", "project/consumption/daily?to=2026-10-31"},
	}
	for name, c := range cases {
		if got := buildProjectConsumptionPath(c.sub, c.period, c.service); got != c.want {
			t.Errorf("%s: buildProjectConsumptionPath(%q, %+v, %q) = %q, want %q", name, c.sub, c.period, c.service, got, c.want)
		}
	}
}

// The envelope fields sit flat next to the breakdown. Only id and name may be
// missing: a resource not addressable through the Public API and without a name
// of its own arrives without both and must decode to nil, not to "".
func TestParseProjectConsumptionResponses(t *testing.T) {
	envelope := `"from":"2026-10-01","to":"2026-10-31","time_zone":"Europe/Moscow","currency":"RUB","is_final":true,"total":1234.5678`
	wantTotals := entities.ConsumptionTotals{
		From: "2026-10-01", To: "2026-10-31", TimeZone: "Europe/Moscow", Currency: "RUB", IsFinal: true, Total: 1234.5678,
	}

	t.Run("services", func(t *testing.T) {
		var resp GetProjectConsumptionResponse
		body := `{"consumption":{` + envelope + `,"services":[
			{"service":"vstack","total":1000.5},
			{"service":"llm","total":234.0678}
		]}}`
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.Consumption == nil {
			t.Fatal("consumption = nil")
		}
		if resp.Consumption.ConsumptionTotals != wantTotals {
			t.Errorf("envelope = %+v, want %+v", resp.Consumption.ConsumptionTotals, wantTotals)
		}
		want := []entities.ServiceConsumption{
			{Service: entities.ConsumptionServiceVstack, Total: 1000.5},
			{Service: entities.ConsumptionServiceLLM, Total: 234.0678},
		}
		if len(resp.Consumption.Services) != len(want) {
			t.Fatalf("got %d services, want %d", len(resp.Consumption.Services), len(want))
		}
		for i := range want {
			if resp.Consumption.Services[i] != want[i] {
				t.Errorf("services[%d] = %+v, want %+v", i, resp.Consumption.Services[i], want[i])
			}
		}
	})

	t.Run("resources", func(t *testing.T) {
		var resp GetProjectConsumptionResourcesResponse
		body := `{"consumption":{` + envelope + `,"resources":[
			{"service":"vstack","type":"vstack_server","id":"l1s42","name":"web-01","total":900.5},
			{"service":"vstack","type":"vstack_edge","total":100}
		]}}`
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.Consumption == nil {
			t.Fatal("consumption = nil")
		}
		if resp.Consumption.ConsumptionTotals != wantTotals {
			t.Errorf("envelope = %+v, want %+v", resp.Consumption.ConsumptionTotals, wantTotals)
		}
		if len(resp.Consumption.Resources) != 2 {
			t.Fatalf("got %d resources, want 2", len(resp.Consumption.Resources))
		}

		named := resp.Consumption.Resources[0]
		if named.Service != entities.ConsumptionServiceVstack || named.Type != entities.ConsumptionResourceTypeVstackServer || named.Total != 900.5 {
			t.Errorf("resources[0] = %+v", named)
		}
		if named.ID == nil || *named.ID != "l1s42" {
			t.Errorf("resources[0].id = %v, want l1s42", named.ID)
		}
		if named.Name == nil || *named.Name != "web-01" {
			t.Errorf("resources[0].name = %v, want web-01", named.Name)
		}

		bare := resp.Consumption.Resources[1]
		if bare.Type != entities.ConsumptionResourceTypeVstackEdge || bare.Total != 100 {
			t.Errorf("resources[1] = %+v", bare)
		}
		if bare.ID != nil {
			t.Errorf("resources[1].id = %q, want nil", *bare.ID)
		}
		if bare.Name != nil {
			t.Errorf("resources[1].name = %q, want nil", *bare.Name)
		}
	})

	t.Run("daily", func(t *testing.T) {
		var resp GetProjectConsumptionDailyResponse
		body := `{"consumption":{` + envelope + `,"days":[
			{"date":"2026-10-01","total":1234.5678},
			{"date":"2026-10-02","total":0}
		]}}`
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.Consumption == nil {
			t.Fatal("consumption = nil")
		}
		if resp.Consumption.ConsumptionTotals != wantTotals {
			t.Errorf("envelope = %+v, want %+v", resp.Consumption.ConsumptionTotals, wantTotals)
		}
		want := []entities.DailyConsumption{{Date: "2026-10-01", Total: 1234.5678}, {Date: "2026-10-02"}}
		if len(resp.Consumption.Days) != len(want) {
			t.Fatalf("got %d days, want %d", len(resp.Consumption.Days), len(want))
		}
		for i := range want {
			if resp.Consumption.Days[i] != want[i] {
				t.Errorf("days[%d] = %+v, want %+v", i, resp.Consumption.Days[i], want[i])
			}
		}
	})
}

// Each read is a GET on its own route with the period and the filter as query
// and the key in X-API-KEY. A malformed period is refused before the request,
// and a response without the consumption envelope is ErrNotFound, not an empty
// consumption.
func TestGetProjectConsumptionRequests(t *testing.T) {
	period := &entities.ConsumptionPeriodRequest{From: "2026-10-01", To: "2026-10-31"}
	reversed := &entities.ConsumptionPeriodRequest{From: "2026-10-31", To: "2026-10-01"}
	envelope := `"from":"2026-10-01","to":"2026-10-31","time_zone":"Europe/Moscow","currency":"RUB","is_final":true,"total":10`

	type call func(ctx context.Context, c *CloudClient, p *entities.ConsumptionPeriodRequest) (float64, error)
	reads := map[string]struct {
		call call
		uri  string
		body string
	}{
		"services": {
			call: func(ctx context.Context, c *CloudClient, p *entities.ConsumptionPeriodRequest) (float64, error) {
				r, err := c.GetProjectConsumption(ctx, p)
				if r == nil {
					return 0, err
				}
				return r.Total, err
			},
			uri:  "/api/v1/project/consumption?from=2026-10-01&to=2026-10-31",
			body: `{"consumption":{` + envelope + `,"services":[{"service":"vstack","total":10}]}}`,
		},
		"resources": {
			call: func(ctx context.Context, c *CloudClient, p *entities.ConsumptionPeriodRequest) (float64, error) {
				r, err := c.GetProjectConsumptionResources(ctx, p, entities.ConsumptionServiceVstack)
				if r == nil {
					return 0, err
				}
				return r.Total, err
			},
			uri:  "/api/v1/project/consumption/resources?from=2026-10-01&service=vstack&to=2026-10-31",
			body: `{"consumption":{` + envelope + `,"resources":[{"service":"vstack","type":"vstack_edge","total":10}]}}`,
		},
		"daily": {
			call: func(ctx context.Context, c *CloudClient, p *entities.ConsumptionPeriodRequest) (float64, error) {
				r, err := c.GetProjectConsumptionDaily(ctx, p)
				if r == nil {
					return 0, err
				}
				return r.Total, err
			},
			uri:  "/api/v1/project/consumption/daily?from=2026-10-01&to=2026-10-31",
			body: `{"consumption":{` + envelope + `,"days":[{"date":"2026-10-01","total":10}]}}`,
		},
	}

	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			var calls int32
			var lastRequest, apiKey, respond atomic.Value
			respond.Store(read.body)
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&calls, 1)
				lastRequest.Store(r.Method + " " + r.URL.RequestURI())
				apiKey.Store(r.Header.Get("X-API-KEY"))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(respond.Load().(string)))
			}))
			ctx := context.Background()

			total, err := read.call(ctx, client, period)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if total != 10 {
				t.Errorf("total = %v, want 10", total)
			}
			if got, want := lastRequest.Load(), "GET "+read.uri; got != want {
				t.Errorf("sent %v, want %q", got, want)
			}
			if got := apiKey.Load(); got != "test-key" {
				t.Errorf("X-API-KEY = %v, want test-key", got)
			}

			before := atomic.LoadInt32(&calls)
			if _, err := read.call(ctx, client, reversed); err == nil {
				t.Error("a reversed period must be refused")
			}
			if after := atomic.LoadInt32(&calls); after != before {
				t.Errorf("a reversed period reached the API: %d calls, want %d", after, before)
			}

			respond.Store(`{}`)
			if _, err := read.call(ctx, client, nil); !errors.Is(err, ErrNotFound) {
				t.Errorf("a response without consumption: err = %v, want ErrNotFound", err)
			}
		})
	}
}
