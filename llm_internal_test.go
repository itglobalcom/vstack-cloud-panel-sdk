package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

func TestBuildLLMPaths(t *testing.T) {
	cases := map[string]string{
		buildLLMPath():                                "llm",
		buildLLMPath(llmConnectionPath):               "llm/connection",
		buildLLMPath(llmModelsPath, "7"):              "llm/models/7",
		buildLLMGroupPath(3, ""):                      "llm/groups/3",
		buildLLMGroupPath(3, llmModelsSubPath):        "llm/groups/3/models",
		buildLLMGroupPath(3, llmDisableSubPath):       "llm/groups/3/disable",
		buildLLMAccessKeyPath(3, 0, ""):               "llm/groups/3/access-keys",
		buildLLMAccessKeyPath(3, 9, ""):               "llm/groups/3/access-keys/9",
		buildLLMAccessKeyPath(3, 9, llmSecretSubPath): "llm/groups/3/access-keys/9/secret",
		buildLLMAccessKeyListPath(0):                  "llm/access-keys",
		buildLLMAccessKeyListPath(3):                  "llm/access-keys?group_id=3",
		buildLLMUsagePath(&entities.LLMUsageRequest{From: "2026-10-01", To: "2026-10-31"}): "llm/usage?from=2026-10-01&to=2026-10-31",
		buildLLMUsagePath(&entities.LLMUsageRequest{
			From: "2026-10-01", To: "2026-10-31", GroupBy: entities.LLMUsageGroupByAccessKey,
			GroupID: 3, AccessKeyID: 9, ModelID: 7,
		}): "llm/usage?access_key_id=9&from=2026-10-01&group_by=access_key&group_id=3&model_id=7&to=2026-10-31",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
	}
}

// The wire form of cloudmng: optional fields are left out, so a group still
// being created and a key still being issued come without id and state, and a
// detailed model embeds the catalog form at the top level of its JSON.
func TestLLMResponseLiterals(t *testing.T) {
	t.Run("groups with one in creation", func(t *testing.T) {
		body := []byte(`{"groups":[
			{"id":3,"name":"prod","state":"active","tags":["team-a"],
			 "models":[{"model_id":7,"tech_title":"qwen3","name":"Qwen 3","pending_operation":"grant"}],
			 "pending_operation":"change_models","is_read_only":false,"created_at":"2026-10-01T10:00:00Z"},
			{"name":"staging","created_at":"2026-11-01T10:00:00Z","task_id":"llm42"}
		]}`)
		var resp ListLLMGroupsResponse
		mustUnmarshal(t, body, &resp)
		if len(resp.Groups) != 2 {
			t.Fatalf("groups = %d, want 2", len(resp.Groups))
		}
		g := resp.Groups[0]
		if g.ID == nil || *g.ID != 3 || g.State == nil || *g.State != entities.LLMGroupStateActive {
			t.Errorf("active group = %+v", g)
		}
		if g.IsReadOnly == nil || *g.IsReadOnly {
			t.Errorf("is_read_only = %v, want present false", g.IsReadOnly)
		}
		if g.PendingOperation == nil || *g.PendingOperation != entities.LLMGroupOperationChangeModels {
			t.Errorf("pending_operation = %v, want change_models", g.PendingOperation)
		}
		if len(g.Models) != 1 || g.Models[0].ModelID != 7 || g.Models[0].PendingOperation == nil ||
			*g.Models[0].PendingOperation != entities.LLMGroupModelOperationGrant {
			t.Errorf("models = %+v", g.Models)
		}
		creating := resp.Groups[1]
		if creating.ID != nil || creating.State != nil || creating.IsReadOnly != nil {
			t.Errorf("group in creation must have no id, state, is_read_only: %+v", creating)
		}
		if creating.TaskID == nil || *creating.TaskID != "llm42" || creating.Name != "staging" {
			t.Errorf("group in creation = %+v, want name staging and task llm42", creating)
		}
	})

	t.Run("no groups", func(t *testing.T) {
		for _, body := range []string{`{"groups":[]}`, `{}`} {
			var resp ListLLMGroupsResponse
			mustUnmarshal(t, []byte(body), &resp)
			if len(resp.Groups) != 0 {
				t.Errorf("%s: groups = %v, want none", body, resp.Groups)
			}
		}
	})

	t.Run("group details embed the list form", func(t *testing.T) {
		body := []byte(`{"group":{"id":3,"name":"prod","state":"disabled","created_at":"2026-10-01T10:00:00Z",
			"deployment_state":"deployed",
			"spend":{"today":{"credits":1.5,"cost":3},"week":{"credits":10},"month":{"credits":40,"cost":80}},
			"total_spent_credits":1200,"updated_at":"2026-11-01T10:00:00Z"}}`)
		var resp GetLLMGroupResponse
		mustUnmarshal(t, body, &resp)
		g := resp.Group
		if g == nil || g.ID == nil || *g.ID != 3 || g.Name != "prod" || *g.State != entities.LLMGroupStateDisabled {
			t.Fatalf("embedded group fields = %+v", g)
		}
		if g.DeploymentState != entities.LLMGroupDeploymentStateDeployed || g.TotalSpentCredits != 1200 ||
			g.UpdatedAt != "2026-11-01T10:00:00Z" {
			t.Errorf("details = %+v", g)
		}
		if g.Spend == nil || g.Spend.Today.Credits != 1.5 || g.Spend.Today.Cost == nil || *g.Spend.Today.Cost != 3 {
			t.Errorf("spend.today = %+v", g.Spend)
		}
		if g.Spend.Week.Cost != nil {
			t.Errorf("unpriced spend.week.cost = %v, want nil", *g.Spend.Week.Cost)
		}
	})

	t.Run("access keys with one being issued", func(t *testing.T) {
		body := []byte(`{"access_keys":[
			{"id":9,"group_id":3,"group_name":"prod","name":"ci","state":"active",
			 "locks":{"by_client":true,"client_reason":"rotation","by_quota":false,"by_admin":false,"by_balance":true,"by_platform":false},
			 "tags":["ci"],"consumed_credits":0,"month_spend":{"credits":12.5},"expires_at":"2027-01-01T00:00:00Z",
			 "is_secret_available":true,"pending_operation":"lock_by_client","is_group_read_only":false,
			 "created_at":"2026-10-01T10:00:00Z"},
			{"group_id":3,"group_name":"prod","name":"new","created_at":"2026-11-01T10:00:00Z","task_id":"llm43"}
		]}`)
		var resp ListLLMAccessKeysResponse
		mustUnmarshal(t, body, &resp)
		if len(resp.AccessKeys) != 2 {
			t.Fatalf("keys = %d, want 2", len(resp.AccessKeys))
		}
		k := resp.AccessKeys[0]
		if k.ID == nil || *k.ID != 9 || k.GroupID != 3 || *k.GroupName != "prod" || *k.State != entities.LLMAccessKeyStateActive {
			t.Errorf("key = %+v", k)
		}
		if k.Locks == nil || !k.Locks.ByClient || !k.Locks.ByBalance || k.Locks.ByAdmin ||
			k.Locks.ClientReason == nil || *k.Locks.ClientReason != "rotation" {
			t.Errorf("locks = %+v", k.Locks)
		}
		if k.ConsumedCredits == nil || *k.ConsumedCredits != 0 {
			t.Errorf("consumed_credits = %v, want present 0", k.ConsumedCredits)
		}
		if k.MonthSpend == nil || k.MonthSpend.Credits != 12.5 || k.LastUsedAt != nil {
			t.Errorf("month_spend = %+v, last_used_at = %v", k.MonthSpend, k.LastUsedAt)
		}
		if *k.PendingOperation != entities.LLMAccessKeyOperationLockByClient || !*k.IsSecretAvailable {
			t.Errorf("key operation/secret = %+v", k)
		}
		issuing := resp.AccessKeys[1]
		if issuing.ID != nil || issuing.State != nil || issuing.Locks != nil || issuing.IsSecretAvailable != nil {
			t.Errorf("key being issued must have no id, state, locks, is_secret_available: %+v", issuing)
		}
		if issuing.GroupID != 3 || issuing.TaskID == nil || *issuing.TaskID != "llm43" {
			t.Errorf("key being issued = %+v, want group 3 and task llm43", issuing)
		}
	})

	t.Run("connection not connected", func(t *testing.T) {
		var resp GetLLMConnectionResponse
		mustUnmarshal(t, []byte(`{"connection":{"state":"not_connected"}}`), &resp)
		c := resp.Connection
		if c == nil || c.State != entities.LLMConnectionStateNotConnected {
			t.Fatalf("connection = %+v", c)
		}
		if c.TaskID != nil || c.GroupName != nil || c.FirstKeyName != nil {
			t.Errorf("not connected must carry no task or names: %+v", c)
		}

		mustUnmarshal(t, []byte(`{"connection":{"state":"connecting","task_id":"llm5","group_name":"prod","first_key_name":"ci"}}`), &resp)
		c = resp.Connection
		if c.State != entities.LLMConnectionStateConnecting || *c.TaskID != "llm5" || *c.GroupName != "prod" || *c.FirstKeyName != "ci" {
			t.Errorf("connecting = %+v", c)
		}
	})

	t.Run("model catalog", func(t *testing.T) {
		body := []byte(`{"models":[
			{"id":7,"tech_title":"qwen3","name":"Qwen 3","description":"Chat","vendor":"Alibaba","kind":"text",
			 "contour":"local","storage_countries":["RU"],"is_pii_protected":true,"context_size":32768,
			 "unit_kind":"tokens","endpoint":"https://llm.example/v1",
			 "starting_price":{"provider":"local","input":{"unit":"input_tokens","credits":0.5},
			   "output":{"unit":"output_tokens","variant":"default","credits":1.5,"money":3},"is_from_live_provider":false},
			 "is_available_to_groups":true,"prices_collected_at":"2026-11-01T00:00:00Z"},
			{"id":8,"tech_title":"whisper","name":"Whisper","description":"","vendor":"OpenAI","kind":"speech",
			 "contour":"external","is_pii_protected":false,"context_size":0,"unit_kind":"none",
			 "endpoint":"https://llm.example/v1","is_available_to_groups":false}
		],"credit_price":{"basis":"package","price_per_credit":2,"currency":"RUB","tax_label":"nds_excluded","package_name":"S"},
		"prices_collected_at":"2026-11-01T00:00:00Z"}`)
		var catalog entities.LLMModelCatalog
		mustUnmarshal(t, body, &catalog)
		if len(catalog.Models) != 2 {
			t.Fatalf("models = %d, want 2", len(catalog.Models))
		}
		m := catalog.Models[0]
		if m.ID != 7 || m.Kind != entities.LLMModelKindText || m.Contour != entities.LLMModelContourLocal ||
			m.UnitKind != entities.LLMModelUnitKindTokens || !m.IsPIIProtected || m.ContextSize != 32768 ||
			m.Endpoint != "https://llm.example/v1" || !m.IsAvailableToGroups {
			t.Errorf("model = %+v", m)
		}
		sp := m.StartingPrice
		if sp == nil || sp.Input.Unit != entities.LLMModelPriceUnitInputTokens || sp.Input.Credits != 0.5 ||
			sp.Input.Money != nil || sp.Output == nil || *sp.Output.Variant != "default" || *sp.Output.Money != 3 {
			t.Errorf("starting_price = %+v", sp)
		}
		bare := catalog.Models[1]
		if bare.StartingPrice != nil || bare.PricesCollectedAt != nil || bare.StorageCountries != nil {
			t.Errorf("model without prices = %+v", bare)
		}
		cp := catalog.CreditPrice
		if cp == nil || cp.Basis != entities.LLMCreditPriceBasisPackage || cp.PricePerCredit != 2 ||
			cp.Currency != "RUB" || *cp.TaxLabel != entities.LLMTaxLabelNDSExcluded || *cp.PackageName != "S" ||
			cp.ValidThrough != nil {
			t.Errorf("credit_price = %+v", cp)
		}

		var empty entities.LLMModelCatalog
		mustUnmarshal(t, []byte(`{}`), &empty)
		if empty.Models != nil || empty.CreditPrice != nil || empty.PricesCollectedAt != nil {
			t.Errorf("empty catalog = %+v", empty)
		}
	})

	t.Run("model details embed the catalog form", func(t *testing.T) {
		body := []byte(`{"model":{"id":7,"tech_title":"qwen3","name":"Qwen 3","vendor":"Alibaba","kind":"rag",
			"contour":"external","is_pii_protected":true,"context_size":8192,"unit_kind":"tokens",
			"endpoint":"https://llm.example/v1","is_available_to_groups":true,
			"default_provider":"together",
			"prices":[{"unit":"cache_read_tokens","credits":0.1}],
			"protected_price":{"model":[{"unit":"input_tokens","credits":1}],"recognition":{"unit":"characters","credits":0.2},
			  "total":[{"unit":"input_tokens","credits":1.2}]},
			"providers":[{"slug":"together","name":"Together","context_length":8192,"prices":[{"unit":"input_tokens","credits":1}],
			  "discount":0.1,"is_live":true,"supports_tools":true,"supports_reasoning":false,
			  "supports_structured_outputs":true,"supports_web_search":false}],
			"supports_streaming":true}}`)
		var resp GetLLMModelResponse
		mustUnmarshal(t, body, &resp)
		m := resp.Model
		if m == nil || m.ID != 7 || m.Kind != entities.LLMModelKindRAG || m.Contour != entities.LLMModelContourExternal {
			t.Fatalf("embedded model = %+v", m)
		}
		if m.DefaultProvider != "together" || len(m.Prices) != 1 || m.Prices[0].Unit != entities.LLMModelPriceUnitCacheReadTokens {
			t.Errorf("details = %+v", m)
		}
		if m.ProtectedPrice == nil || m.ProtectedPrice.Recognition.Unit != entities.LLMModelPriceUnitCharacters ||
			len(m.ProtectedPrice.Total) != 1 || m.ProtectedPrice.Total[0].Credits != 1.2 {
			t.Errorf("protected_price = %+v", m.ProtectedPrice)
		}
		if len(m.Providers) != 1 {
			t.Fatalf("providers = %+v", m.Providers)
		}
		p := m.Providers[0]
		if p.Slug != "together" || *p.ContextLength != 8192 || p.MaxCompletionTokens != nil || p.Variant != nil ||
			p.Discount != 0.1 || !p.IsLive || !p.SupportsTools || p.SupportsReasoning || !p.SupportsStructuredOutputs {
			t.Errorf("provider = %+v", p)
		}
		if m.SupportsStreaming == nil || !*m.SupportsStreaming || m.SupportsTools != nil || m.CreditPrice != nil {
			t.Errorf("capabilities = streaming %v, tools %v, credit_price %v", m.SupportsStreaming, m.SupportsTools, m.CreditPrice)
		}
	})

	t.Run("package without current package and stop", func(t *testing.T) {
		body := []byte(`{"package":{"usage":{"period_from":"2026-11-01","period_to":"2026-12-01","total_credits":5,
			"in_package_credits":0,"over_package_credits":5},"currency":"RUB","packages_disabled":false,
			"exhausted_package_mode":"continue_pay_as_you_go"}}`)
		var resp GetLLMPackageResponse
		mustUnmarshal(t, body, &resp)
		p := resp.Package
		if p == nil || p.Package != nil || p.Stop != nil || p.PricePerCredit != nil || p.TaxLabel != nil ||
			p.PayAsYouGoLimitCredits != nil || p.Usage.PackageVolumeCredits != nil {
			t.Fatalf("package without optionals = %+v", p)
		}
		if p.ExhaustedPackageMode != entities.LLMExhaustedPackageModeContinuePayAsYouGo ||
			p.Usage.PeriodTo != "2026-12-01" || p.Usage.OverPackageCredits != 5 || p.Currency != "RUB" {
			t.Errorf("package = %+v", p)
		}
	})

	t.Run("package with current package and stop", func(t *testing.T) {
		body := []byte(`{"package":{"package":{"id":2,"tech_title":"S","volume_credits":1000,"discount_percent":10,"price":900,
			"full_month_volume_credits":1000,"period_from":"2026-11-01","period_to":"2026-12-01","auto_renew":true,"state":"active"},
			"usage":{"period_from":"2026-11-01","period_to":"2026-12-01","total_credits":1000,"package_volume_credits":1000,
			"in_package_credits":1000,"over_package_credits":0},"price_per_credit":0.9,"currency":"RUB","tax_label":"vat_excluded",
			"packages_disabled":false,"exhausted_package_mode":"stop","pay_as_you_go_limit_credits":0,
			"stop":{"stopped_at":"2026-11-20T10:00:00Z","reason":"package_exhausted","state":"stopped"}}}`)
		var resp GetLLMPackageResponse
		mustUnmarshal(t, body, &resp)
		p := resp.Package
		if p.Package == nil || p.Package.ID != 2 || p.Package.State != entities.LLMPackageStateActive || !p.Package.AutoRenew ||
			p.Package.FullMonthVolumeCredits == nil || p.Package.FullMonthPrice != nil {
			t.Errorf("package.package = %+v", p.Package)
		}
		if p.Stop == nil || p.Stop.Reason != entities.LLMPackageStopReasonPackageExhausted || p.Stop.State != entities.LLMPackageStopStateStopped {
			t.Errorf("stop = %+v", p.Stop)
		}
		if p.PayAsYouGoLimitCredits == nil || *p.PayAsYouGoLimitCredits != 0 || *p.TaxLabel != entities.LLMTaxLabelVATExcluded {
			t.Errorf("limit/tax = %v/%v", p.PayAsYouGoLimitCredits, p.TaxLabel)
		}
	})

	t.Run("package plans", func(t *testing.T) {
		body := []byte(`{"package_plans":[{"id":1,"tech_title":"S","description":"Small","volume_credits":1000,
			"discount_percent":10,"price":900,"currency":"RUB",
			"current_month":{"period_from":"2026-11-07","period_to":"2026-12-01","volume_credits":800,"price":720}},
			{"id":2,"tech_title":"M","volume_credits":5000,"discount_percent":20,"price":4000,"currency":"RUB"}]}`)
		var resp ListLLMPackagePlansResponse
		mustUnmarshal(t, body, &resp)
		if len(resp.PackagePlans) != 2 {
			t.Fatalf("plans = %d, want 2", len(resp.PackagePlans))
		}
		if cm := resp.PackagePlans[0].CurrentMonth; cm == nil || cm.VolumeCredits != 800 || cm.Price != 720 {
			t.Errorf("current_month = %+v", cm)
		}
		if p := resp.PackagePlans[1]; p.CurrentMonth != nil || p.TaxLabel != nil || p.VolumeCredits != 5000 {
			t.Errorf("plan without optionals = %+v", p)
		}
	})

	t.Run("usage none has no rows", func(t *testing.T) {
		body := []byte(`{"usage":{"from":"2026-10-01","to":"2026-10-31","group_by":"none",
			"totals":{"credits":10,"local_credits":4,"external_credits":6,"in_tokens":100,"out_tokens":50,"cached_tokens":5},
			"collection":{"state":"not_collected"}}}`)
		var resp GetLLMUsageResponse
		mustUnmarshal(t, body, &resp)
		u := resp.Usage
		if u == nil || u.GroupBy != entities.LLMUsageGroupByNone || u.Items != nil || u.CreditPrice != nil {
			t.Fatalf("usage = %+v", u)
		}
		if u.Totals.Credits != 10 || u.Totals.LocalCredits != 4 || u.Totals.ExternalCredits != 6 ||
			u.Totals.Cost != nil || u.Totals.InTokens != 100 || u.Totals.OutTokens != 50 || u.Totals.CachedTokens != 5 {
			t.Errorf("totals = %+v", u.Totals)
		}
		if u.Collection.State != entities.LLMUsageCollectionStateNotCollected || u.Collection.CollectedAt != nil {
			t.Errorf("collection = %+v", u.Collection)
		}
	})

	t.Run("usage rows of every breakdown", func(t *testing.T) {
		sums := `"credits":2,"local_credits":1,"external_credits":1,"cost":4,"in_tokens":10,"out_tokens":5,"cached_tokens":0`
		body := []byte(`{"usage":{"from":"2026-10-01","to":"2026-10-31","group_by":"day",
			"totals":{` + sums + `},"items":[
			{` + sums + `,"date":"2026-10-01"},
			{` + sums + `,"model_id":7,"model_tech_title":"qwen3","model_name":"Qwen 3"},
			{` + sums + `,"group_id":3,"group_name":"prod"},
			{` + sums + `,"access_key_id":9,"access_key_name":"ci"}],
			"collection":{"state":"collected","collected_at":"2026-11-01T00:00:00Z"}}}`)
		var resp GetLLMUsageResponse
		mustUnmarshal(t, body, &resp)
		items := resp.Usage.Items
		if len(items) != 4 {
			t.Fatalf("items = %d, want 4", len(items))
		}
		for i, it := range items {
			if it.Credits != 2 || it.Cost == nil || *it.Cost != 4 || it.InTokens != 10 {
				t.Errorf("items[%d] embedded sums = %+v", i, it.LLMUsageAmounts)
			}
		}
		if d := items[0]; d.Date == nil || *d.Date != "2026-10-01" || d.ModelID != nil || d.GroupID != nil || d.AccessKeyID != nil {
			t.Errorf("day row = %+v", d)
		}
		if m := items[1]; m.ModelID == nil || *m.ModelID != 7 || *m.ModelTechTitle != "qwen3" || *m.ModelName != "Qwen 3" || m.Date != nil {
			t.Errorf("model row = %+v", m)
		}
		if g := items[2]; g.GroupID == nil || *g.GroupID != 3 || *g.GroupName != "prod" || g.ModelID != nil {
			t.Errorf("group row = %+v", g)
		}
		if k := items[3]; k.AccessKeyID == nil || *k.AccessKeyID != 9 || *k.AccessKeyName != "ci" || k.GroupID != nil {
			t.Errorf("access key row = %+v", k)
		}
		if c := resp.Usage.Collection; c.State != entities.LLMUsageCollectionStateCollected || c.CollectedAt == nil {
			t.Errorf("collection = %+v", c)
		}
	})

	t.Run("request log", func(t *testing.T) {
		body := []byte(`{"source_connected":true,"requests":[
			{"occurred_at":"2026-11-06T10:00:00Z","group_id":3,"access_key_id":9,"model_id":7,"model_tech_title":"qwen3",
			 "response_code":200,"duration_ms":850,"in_tokens":100,"out_tokens":50,"cached_tokens":10,"credits":0.75},
			{"occurred_at":"2026-11-06T10:01:00Z","group_id":3,"access_key_id":9,"model_tech_title":"gone",
			 "response_code":404,"duration_ms":3,"in_tokens":0,"out_tokens":0,"cached_tokens":0,"credits":0}]}`)
		var log entities.LLMRequestLog
		mustUnmarshal(t, body, &log)
		if !log.SourceConnected || len(log.Requests) != 2 {
			t.Fatalf("log = %+v", log)
		}
		r := log.Requests[0]
		if r.GroupID != 3 || r.AccessKeyID != 9 || r.ModelID == nil || *r.ModelID != 7 || r.ResponseCode != 200 ||
			r.DurationMs != 850 || r.CachedTokens != 10 || r.Credits != 0.75 {
			t.Errorf("request = %+v", r)
		}
		if log.Requests[1].ModelID != nil {
			t.Errorf("request of an unknown model: model_id = %v, want nil", *log.Requests[1].ModelID)
		}

		var off entities.LLMRequestLog
		mustUnmarshal(t, []byte(`{"source_connected":false}`), &off)
		if off.SourceConnected || off.Requests != nil {
			t.Errorf("source not connected = %+v", off)
		}
	})
}

// llmCall is one HTTP exchange the stub expects: the request line, the body and
// the answer.
type llmCall struct {
	method, uri, body, key string
}

// llmStub answers by "METHOD uri" and records every request it gets.
type llmStub struct {
	mu      sync.Mutex
	answers map[string]string
	calls   []llmCall
}

func (s *llmStub) handler(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		line := r.Method + " " + r.URL.RequestURI()
		s.mu.Lock()
		s.calls = append(s.calls, llmCall{r.Method, r.URL.RequestURI(), string(raw), r.Header.Get("X-API-KEY")})
		answer, ok := s.answers[line]
		s.mu.Unlock()
		if !ok {
			t.Errorf("unexpected request %s", line)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(answer))
	})
}

func (s *llmStub) snapshot() []llmCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]llmCall(nil), s.calls...)
}

// Every operation goes to its own route with its own method and body, carries
// the key in X-API-KEY and decodes its answer; a task operation returns the
// llm<n> task reference.
func TestLLMOperationsOnTheWire(t *testing.T) {
	const task = `{"task_id":"llm42"}`
	ctx := context.Background()

	type op struct {
		method, uri, body, answer string
		call                      func(c *CloudClient) (any, error)
		want                      any
	}
	ops := map[string]op{
		"GetLLMConnection": {
			"GET", "/api/v1/llm/connection", "", `{"connection":{"state":"connected"}}`,
			func(c *CloudClient) (any, error) {
				r, err := c.GetLLMConnection(ctx)
				return field(r, err, func() any { return r.State })
			}, entities.LLMConnectionStateConnected,
		},
		"ConnectLLM": {
			"POST", "/api/v1/llm/connection", `{"group_name":"prod","first_key_name":"ci"}`, task,
			func(c *CloudClient) (any, error) {
				r, err := c.ConnectLLM(ctx, &entities.ConnectLLMRequest{GroupName: "prod", FirstKeyName: "ci"})
				return field(r, err, func() any { return r.ID })
			}, "llm42",
		},
		"GetLLMGroupList": {
			"GET", "/api/v1/llm/groups", "", `{"groups":[{"id":3,"name":"prod","created_at":"x"}]}`,
			func(c *CloudClient) (any, error) {
				r, err := c.GetLLMGroupList(ctx)
				return len(r), err
			}, 1,
		},
		"GetLLMGroup": {
			"GET", "/api/v1/llm/groups/3", "", `{"group":{"id":3,"name":"prod","created_at":"x","deployment_state":"error"}}`,
			func(c *CloudClient) (any, error) {
				r, err := c.GetLLMGroup(ctx, 3)
				return field(r, err, func() any { return r.DeploymentState })
			}, entities.LLMGroupDeploymentStateError,
		},
		"CreateLLMGroup": {
			"POST", "/api/v1/llm/groups", `{"name":"prod"}`, task,
			func(c *CloudClient) (any, error) {
				r, err := c.CreateLLMGroup(ctx, &entities.CreateLLMGroupRequest{Name: "prod"})
				return field(r, err, func() any { return r.ID })
			}, "llm42",
		},
		"SetLLMGroupModels": {
			"PUT", "/api/v1/llm/groups/3/models", `{"model_ids":[7,8]}`, task,
			func(c *CloudClient) (any, error) {
				r, err := c.SetLLMGroupModels(ctx, 3, &entities.SetLLMGroupModelsRequest{ModelIDs: []entities.LLMModelID{7, 8}})
				return field(r, err, func() any { return r.ID })
			}, "llm42",
		},
		"SetLLMGroupModels empty": {
			"PUT", "/api/v1/llm/groups/4/models", `{"model_ids":[]}`, task,
			func(c *CloudClient) (any, error) {
				r, err := c.SetLLMGroupModels(ctx, 4, &entities.SetLLMGroupModelsRequest{ModelIDs: []entities.LLMModelID{}})
				return field(r, err, func() any { return r.ID })
			}, "llm42",
		},
		"SetLLMGroupTags": {
			"PUT", "/api/v1/llm/groups/3/tags", `{"tags":["a","b"]}`, "",
			func(c *CloudClient) (any, error) {
				return nil, c.SetLLMGroupTags(ctx, 3, &entities.SetLLMTagsRequest{Tags: []string{"a", "b"}})
			}, nil,
		},
		"DisableLLMGroup": {
			"POST", "/api/v1/llm/groups/3/disable", "", task,
			func(c *CloudClient) (any, error) {
				r, err := c.DisableLLMGroup(ctx, 3)
				return field(r, err, func() any { return r.ID })
			}, "llm42",
		},
		"GetLLMAccessKeyList": {
			"GET", "/api/v1/llm/access-keys", "", `{"access_keys":[{"id":9,"group_id":3,"name":"ci","created_at":"x"}]}`,
			func(c *CloudClient) (any, error) {
				r, err := c.GetLLMAccessKeyList(ctx, 0)
				return len(r), err
			}, 1,
		},
		"GetLLMAccessKeyList by group": {
			"GET", "/api/v1/llm/access-keys?group_id=3", "", `{}`,
			func(c *CloudClient) (any, error) {
				r, err := c.GetLLMAccessKeyList(ctx, 3)
				return len(r), err
			}, 0,
		},
		"GetLLMAccessKey": {
			"GET", "/api/v1/llm/groups/3/access-keys/9", "", `{"access_key":{"id":9,"group_id":3,"name":"ci","created_at":"x"}}`,
			func(c *CloudClient) (any, error) {
				r, err := c.GetLLMAccessKey(ctx, 3, 9)
				return field(r, err, func() any { return r.Name })
			}, "ci",
		},
		"IssueLLMAccessKey": {
			"POST", "/api/v1/llm/groups/3/access-keys", `{"name":"ci"}`, task,
			func(c *CloudClient) (any, error) {
				r, err := c.IssueLLMAccessKey(ctx, 3, &entities.IssueLLMAccessKeyRequest{Name: "ci"})
				return field(r, err, func() any { return r.ID })
			}, "llm42",
		},
		"RevealLLMAccessKeySecret": {
			"POST", "/api/v1/llm/groups/3/access-keys/9/secret", "", `{"secret":"sk-1"}`,
			func(c *CloudClient) (any, error) {
				return c.RevealLLMAccessKeySecret(ctx, 3, 9)
			}, "sk-1",
		},
		"LockLLMAccessKey with reason": {
			"POST", "/api/v1/llm/groups/3/access-keys/9/lock", `{"reason":"rotation"}`, task,
			func(c *CloudClient) (any, error) {
				r, err := c.LockLLMAccessKey(ctx, 3, 9, &entities.LockLLMAccessKeyRequest{Reason: "rotation"})
				return field(r, err, func() any { return r.ID })
			}, "llm42",
		},
		"LockLLMAccessKey empty reason": {
			"POST", "/api/v1/llm/groups/3/access-keys/10/lock", `{}`, task,
			func(c *CloudClient) (any, error) {
				r, err := c.LockLLMAccessKey(ctx, 3, 10, &entities.LockLLMAccessKeyRequest{})
				return field(r, err, func() any { return r.ID })
			}, "llm42",
		},
		"LockLLMAccessKey nil": {
			"POST", "/api/v1/llm/groups/3/access-keys/11/lock", "", task,
			func(c *CloudClient) (any, error) {
				r, err := c.LockLLMAccessKey(ctx, 3, 11, nil)
				return field(r, err, func() any { return r.ID })
			}, "llm42",
		},
		"UnlockLLMAccessKey": {
			"POST", "/api/v1/llm/groups/3/access-keys/9/unlock", "", task,
			func(c *CloudClient) (any, error) {
				r, err := c.UnlockLLMAccessKey(ctx, 3, 9)
				return field(r, err, func() any { return r.ID })
			}, "llm42",
		},
		"DeleteLLMAccessKey": {
			"DELETE", "/api/v1/llm/groups/3/access-keys/9", "", task,
			func(c *CloudClient) (any, error) {
				r, err := c.DeleteLLMAccessKey(ctx, 3, 9)
				return field(r, err, func() any { return r.ID })
			}, "llm42",
		},
		"SetLLMAccessKeyTags": {
			"PUT", "/api/v1/llm/groups/3/access-keys/9/tags", `{"tags":[]}`, "",
			func(c *CloudClient) (any, error) {
				return nil, c.SetLLMAccessKeyTags(ctx, 3, 9, &entities.SetLLMTagsRequest{Tags: []string{}})
			}, nil,
		},
		"GetLLMModelList": {
			"GET", "/api/v1/llm/models", "", `{"models":[{"id":7,"tech_title":"qwen3"}],"prices_collected_at":"x"}`,
			func(c *CloudClient) (any, error) {
				r, err := c.GetLLMModelList(ctx)
				return field(r, err, func() any { return len(r.Models) })
			}, 1,
		},
		"GetLLMModel": {
			"GET", "/api/v1/llm/models/7", "", `{"model":{"id":7,"tech_title":"qwen3","default_provider":"local"}}`,
			func(c *CloudClient) (any, error) {
				r, err := c.GetLLMModel(ctx, 7)
				return field(r, err, func() any { return r.DefaultProvider })
			}, "local",
		},
		"GetLLMPackage": {
			"GET", "/api/v1/llm/package", "", `{"package":{"currency":"RUB","exhausted_package_mode":"stop"}}`,
			func(c *CloudClient) (any, error) {
				r, err := c.GetLLMPackage(ctx)
				return field(r, err, func() any { return r.ExhaustedPackageMode })
			}, entities.LLMExhaustedPackageModeStop,
		},
		"GetLLMPackagePlanList": {
			"GET", "/api/v1/llm/package-plans", "", `{"package_plans":[{"id":1},{"id":2}]}`,
			func(c *CloudClient) (any, error) {
				r, err := c.GetLLMPackagePlanList(ctx)
				return len(r), err
			}, 2,
		},
		"GetLLMUsage": {
			"GET", "/api/v1/llm/usage?from=2026-10-01&group_by=model&model_id=7&to=2026-10-31", "",
			`{"usage":{"from":"2026-10-01","to":"2026-10-31","group_by":"model","collection":{"state":"failed"}}}`,
			func(c *CloudClient) (any, error) {
				r, err := c.GetLLMUsage(ctx, &entities.LLMUsageRequest{
					From: "2026-10-01", To: "2026-10-31", GroupBy: entities.LLMUsageGroupByModel, ModelID: 7,
				})
				return field(r, err, func() any { return r.Collection.State })
			}, entities.LLMUsageCollectionStateFailed,
		},
		"GetLLMRequestList": {
			"GET", "/api/v1/llm/requests", "", `{"source_connected":true}`,
			func(c *CloudClient) (any, error) {
				r, err := c.GetLLMRequestList(ctx)
				return field(r, err, func() any { return r.SourceConnected })
			}, true,
		},
	}

	for name, o := range ops {
		t.Run(name, func(t *testing.T) {
			stub := &llmStub{answers: map[string]string{o.method + " " + o.uri: o.answer}}
			client := newTestClient(t, stub.handler(t))

			got, err := o.call(client)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if got != o.want {
				t.Errorf("%s = %v, want %v", name, got, o.want)
			}
			calls := stub.snapshot()
			if len(calls) != 1 {
				t.Fatalf("sent %d requests, want 1: %+v", len(calls), calls)
			}
			c := calls[0]
			if c.method != o.method || c.uri != o.uri {
				t.Errorf("sent %s %s, want %s %s", c.method, c.uri, o.method, o.uri)
			}
			if c.body != o.body {
				t.Errorf("body = %q, want %q", c.body, o.body)
			}
			if c.key != "test-key" {
				t.Errorf("X-API-KEY = %q, want test-key", c.key)
			}
		})
	}
}

// field returns get() unless the call failed or returned a nil pointer.
func field[T any](r *T, err error, get func() any) (any, error) {
	if err != nil || r == nil {
		return nil, err
	}
	return get(), nil
}

// A 200 without the envelope is ErrNotFound, not a zero value: the caller must
// not take a missing group or an empty secret for a real one.
func TestLLMReadsWithoutEnvelope(t *testing.T) {
	ctx := context.Background()
	usage := &entities.LLMUsageRequest{From: "2026-10-01", To: "2026-10-31"}
	reads := map[string]func(c *CloudClient) error{
		"GetLLMConnection": func(c *CloudClient) error { _, err := c.GetLLMConnection(ctx); return err },
		"GetLLMGroup":      func(c *CloudClient) error { _, err := c.GetLLMGroup(ctx, 3); return err },
		"GetLLMAccessKey":  func(c *CloudClient) error { _, err := c.GetLLMAccessKey(ctx, 3, 9); return err },
		"RevealLLMAccessKeySecret": func(c *CloudClient) error {
			_, err := c.RevealLLMAccessKeySecret(ctx, 3, 9)
			return err
		},
		"GetLLMModel":   func(c *CloudClient) error { _, err := c.GetLLMModel(ctx, 7); return err },
		"GetLLMPackage": func(c *CloudClient) error { _, err := c.GetLLMPackage(ctx); return err },
		"GetLLMUsage":   func(c *CloudClient) error { _, err := c.GetLLMUsage(ctx, usage); return err },
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	for name, read := range reads {
		if err := read(client); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s on {}: err = %v, want ErrNotFound", name, err)
		}
	}
}

// Arguments are checked before the request is built: none of these reaches the API.
func TestLLMInvalidArgumentsSendNothing(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t, noRequestHandler(t))
	okKey := &entities.IssueLLMAccessKeyRequest{Name: "ci"}
	okTags := &entities.SetLLMTagsRequest{Tags: []string{}}
	okModels := &entities.SetLLMGroupModelsRequest{ModelIDs: []entities.LLMModelID{7}}

	calls := map[string]func() error{
		"ConnectLLM nil": func() error { _, err := client.ConnectLLM(ctx, nil); return err },
		"ConnectLLM blank key": func() error {
			_, err := client.ConnectLLM(ctx, &entities.ConnectLLMRequest{GroupName: "prod"})
			return err
		},
		"ConnectLLMAndWait bad": func() error { _, err := client.ConnectLLMAndWait(ctx, &entities.ConnectLLMRequest{}); return err },
		"GetLLMGroup zero":      func() error { _, err := client.GetLLMGroup(ctx, 0); return err },
		"CreateLLMGroup nil":    func() error { _, err := client.CreateLLMGroup(ctx, nil); return err },
		"CreateLLMGroup blank": func() error {
			_, err := client.CreateLLMGroup(ctx, &entities.CreateLLMGroupRequest{Name: " "})
			return err
		},
		"CreateLLMGroupAndWait blank": func() error {
			_, err := client.CreateLLMGroupAndWait(ctx, &entities.CreateLLMGroupRequest{})
			return err
		},
		"SetLLMGroupModels zero group": func() error { _, err := client.SetLLMGroupModels(ctx, 0, okModels); return err },
		"SetLLMGroupModels nil":        func() error { _, err := client.SetLLMGroupModels(ctx, 3, nil); return err },
		"SetLLMGroupModels nil ids": func() error {
			_, err := client.SetLLMGroupModels(ctx, 3, &entities.SetLLMGroupModelsRequest{})
			return err
		},
		"SetLLMGroupModels zero model": func() error {
			_, err := client.SetLLMGroupModels(ctx, 3, &entities.SetLLMGroupModelsRequest{ModelIDs: []entities.LLMModelID{0}})
			return err
		},
		"SetLLMGroupTags zero group":   func() error { return client.SetLLMGroupTags(ctx, 0, okTags) },
		"SetLLMGroupTags nil":          func() error { return client.SetLLMGroupTags(ctx, 3, nil) },
		"SetLLMGroupTags nil tags":     func() error { return client.SetLLMGroupTags(ctx, 3, &entities.SetLLMTagsRequest{}) },
		"DisableLLMGroup negative":     func() error { _, err := client.DisableLLMGroup(ctx, -1); return err },
		"GetLLMAccessKeyList negative": func() error { _, err := client.GetLLMAccessKeyList(ctx, -1); return err },
		"GetLLMAccessKey zero group":   func() error { _, err := client.GetLLMAccessKey(ctx, 0, 9); return err },
		"GetLLMAccessKey zero key":     func() error { _, err := client.GetLLMAccessKey(ctx, 3, 0); return err },
		"IssueLLMAccessKey zero group": func() error { _, err := client.IssueLLMAccessKey(ctx, 0, okKey); return err },
		"IssueLLMAccessKey nil":        func() error { _, err := client.IssueLLMAccessKey(ctx, 3, nil); return err },
		"IssueLLMAccessKey long": func() error {
			_, err := client.IssueLLMAccessKey(ctx, 3, &entities.IssueLLMAccessKeyRequest{Name: strings.Repeat("a", entities.LLMNameMaxLength+1)})
			return err
		},
		"RevealLLMAccessKeySecret zero key": func() error { _, err := client.RevealLLMAccessKeySecret(ctx, 3, 0); return err },
		"LockLLMAccessKey zero key":         func() error { _, err := client.LockLLMAccessKey(ctx, 3, 0, nil); return err },
		"UnlockLLMAccessKey zero group":     func() error { _, err := client.UnlockLLMAccessKey(ctx, 0, 9); return err },
		"DeleteLLMAccessKey zero key":       func() error { _, err := client.DeleteLLMAccessKey(ctx, 3, 0); return err },
		"SetLLMAccessKeyTags zero key":      func() error { return client.SetLLMAccessKeyTags(ctx, 3, 0, okTags) },
		"SetLLMAccessKeyTags nil":           func() error { return client.SetLLMAccessKeyTags(ctx, 3, 9, nil) },
		"GetLLMModel zero":                  func() error { _, err := client.GetLLMModel(ctx, 0); return err },
		"GetLLMUsage nil":                   func() error { _, err := client.GetLLMUsage(ctx, nil); return err },
		"GetLLMUsage reversed": func() error {
			_, err := client.GetLLMUsage(ctx, &entities.LLMUsageRequest{From: "2026-10-31", To: "2026-10-01"})
			return err
		},
	}
	for name, call := range calls {
		if err := call(); err == nil {
			t.Errorf("%s: err = nil, want an error", name)
		}
	}
}

// llmTaskAnswer is a task read a task read with the state and the resources.
func llmTaskAnswer(id, state, resources string) string {
	return `{"task":{"id":"` + id + `","type":"CreateLlmGroup","progress_percent":100,"is_completed":"` + state +
		`","created":"2026-11-01T10:00:00Z","resources":[` + resources + `]}}`
}

// A creating operation waits for its llm<n> task and then reads the resource the
// task reports: the group from llm_group, the key from llm_access_key.
func TestLLMCreateAndWait(t *testing.T) {
	ctx := context.Background()

	t.Run("group", func(t *testing.T) {
		stub := &llmStub{answers: map[string]string{
			"POST /api/v1/llm/groups":   `{"task_id":"llm42"}`,
			"GET /api/v1/tasks/llm42":   llmTaskAnswer("llm42", entities.TaskStateCompleted, `{"type":"llm_access_key","id":"99"},{"type":"llm_group","id":"15"}`),
			"GET /api/v1/llm/groups/15": `{"group":{"id":15,"name":"prod","created_at":"x","deployment_state":"deployed"}}`,
		}}
		client := newTestClient(t, stub.handler(t))

		g, err := client.CreateLLMGroupAndWait(ctx, &entities.CreateLLMGroupRequest{Name: "prod"})
		if err != nil {
			t.Fatalf("CreateLLMGroupAndWait: %v", err)
		}
		if g.ID == nil || *g.ID != 15 {
			t.Errorf("group id = %v, want 15", g.ID)
		}
		if calls := stub.snapshot(); calls[len(calls)-1].uri != "/api/v1/llm/groups/15" {
			t.Errorf("last request = %+v, want the group read", calls[len(calls)-1])
		}
	})

	t.Run("access key", func(t *testing.T) {
		stub := &llmStub{answers: map[string]string{
			"POST /api/v1/llm/groups/3/access-keys":   `{"task_id":"llm43"}`,
			"GET /api/v1/tasks/llm43":                 llmTaskAnswer("llm43", entities.TaskStateCompleted, `{"type":"llm_group","id":"3"},{"type":"llm_access_key","id":"21"}`),
			"GET /api/v1/llm/groups/3/access-keys/21": `{"access_key":{"id":21,"group_id":3,"name":"ci","created_at":"x"}}`,
		}}
		client := newTestClient(t, stub.handler(t))

		k, err := client.IssueLLMAccessKeyAndWait(ctx, 3, &entities.IssueLLMAccessKeyRequest{Name: "ci"})
		if err != nil {
			t.Fatalf("IssueLLMAccessKeyAndWait: %v", err)
		}
		if k.ID == nil || *k.ID != 21 {
			t.Errorf("key id = %v, want 21", k.ID)
		}
	})

	t.Run("no resource", func(t *testing.T) {
		stub := &llmStub{answers: map[string]string{
			"POST /api/v1/llm/groups":               `{"task_id":"llm42"}`,
			"GET /api/v1/tasks/llm42":               llmTaskAnswer("llm42", entities.TaskStateCompleted, ``),
			"POST /api/v1/llm/groups/3/access-keys": `{"task_id":"llm43"}`,
			"GET /api/v1/tasks/llm43":               llmTaskAnswer("llm43", entities.TaskStateCompleted, `{"type":"llm_group","id":"3"}`),
		}}
		client := newTestClient(t, stub.handler(t))

		if _, err := client.CreateLLMGroupAndWait(ctx, &entities.CreateLLMGroupRequest{Name: "prod"}); !errors.Is(err, ErrNotFound) {
			t.Errorf("group task without llm_group: err = %v, want ErrNotFound", err)
		}
		if _, err := client.IssueLLMAccessKeyAndWait(ctx, 3, &entities.IssueLLMAccessKeyRequest{Name: "ci"}); !errors.Is(err, ErrNotFound) {
			t.Errorf("key task without llm_access_key: err = %v, want ErrNotFound", err)
		}
	})

	t.Run("failed task", func(t *testing.T) {
		stub := &llmStub{answers: map[string]string{
			"POST /api/v1/llm/groups": `{"task_id":"llm42"}`,
			"GET /api/v1/tasks/llm42": llmTaskAnswer("llm42", entities.TaskStateFailed, `{"type":"llm_group","id":"15"}`),
		}}
		client := newTestClient(t, stub.handler(t))

		if _, err := client.CreateLLMGroupAndWait(ctx, &entities.CreateLLMGroupRequest{Name: "prod"}); !errors.Is(err, ErrTaskFailed) {
			t.Errorf("failed task: err = %v, want ErrTaskFailed", err)
		}
	})
}

func mustUnmarshal(t *testing.T, body []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}
