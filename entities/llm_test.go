package entities

import (
	"strings"
	"testing"
)

// Every LLM id addresses a positive database key: zero and negatives are refused.
func TestLLMIDsValidate(t *testing.T) {
	validators := map[string]func(int) error{
		"LLMGroupID":     func(v int) error { return LLMGroupID(v).Validate() },
		"LLMAccessKeyID": func(v int) error { return LLMAccessKeyID(v).Validate() },
		"LLMModelID":     func(v int) error { return LLMModelID(v).Validate() },
	}
	for name, validate := range validators {
		if err := validate(1); err != nil {
			t.Errorf("%s(1).Validate() = %v, want nil", name, err)
		}
		for _, v := range []int{0, -1} {
			if err := validate(v); err == nil {
				t.Errorf("%s(%d).Validate() = nil, want an error", name, v)
			}
		}
	}
}

// A name is required and limited to LLMNameMaxLength UTF-16 code units once
// trimmed, the way the API counts it: Cyrillic is one unit per letter (two
// bytes), an emoji is a surrogate pair (two units).
func TestLLMNameRequestsValidate(t *testing.T) {
	validators := map[string]func(string) error{
		"CreateLLMGroupRequest":    func(n string) error { return (&CreateLLMGroupRequest{Name: n}).Validate() },
		"IssueLLMAccessKeyRequest": func(n string) error { return (&IssueLLMAccessKeyRequest{Name: n}).Validate() },
		"ConnectLLMRequest.GroupName": func(n string) error {
			return (&ConnectLLMRequest{GroupName: n, FirstKeyName: "key"}).Validate()
		},
		"ConnectLLMRequest.FirstKeyName": func(n string) error {
			return (&ConnectLLMRequest{GroupName: "group", FirstKeyName: n}).Validate()
		},
	}

	valid := map[string]string{
		"short":             "prod",
		"max ascii":         strings.Repeat("a", LLMNameMaxLength),
		"max ascii trimmed": "  " + strings.Repeat("a", LLMNameMaxLength) + "\t",
		"max cyrillic":      strings.Repeat("я", LLMNameMaxLength),
		"max surrogates":    strings.Repeat("😀", LLMNameMaxLength/2),
	}
	invalid := map[string]string{
		"empty":                 "",
		"blank":                 " \t ",
		"ascii over the limit":  strings.Repeat("a", LLMNameMaxLength+1),
		"surrogates over limit": strings.Repeat("😀", LLMNameMaxLength/2) + "a",
	}

	for name, validate := range validators {
		for c, value := range valid {
			if err := validate(value); err != nil {
				t.Errorf("%s: %s: Validate() = %v, want nil", name, c, err)
			}
		}
		for c, value := range invalid {
			if err := validate(value); err == nil {
				t.Errorf("%s: %s: Validate() = nil, want an error", name, c)
			}
		}
	}
}

// The models set replaces the whole set: nil is a forgotten field and refused,
// an empty list revokes every model, a non-positive id is refused.
func TestSetLLMGroupModelsRequestValidate(t *testing.T) {
	valid := map[string][]LLMModelID{
		"empty revokes all": {},
		"two models":        {1, 2},
	}
	for name, ids := range valid {
		if err := (&SetLLMGroupModelsRequest{ModelIDs: ids}).Validate(); err != nil {
			t.Errorf("%s: Validate(%v) = %v, want nil", name, ids, err)
		}
	}

	invalid := map[string][]LLMModelID{
		"nil":            nil,
		"zero after one": {1, 0},
		"negative":       {-3},
	}
	for name, ids := range invalid {
		if err := (&SetLLMGroupModelsRequest{ModelIDs: ids}).Validate(); err == nil {
			t.Errorf("%s: Validate(%v) = nil, want an error", name, ids)
		}
	}
}

// Tags replace the whole set: nil is refused, an empty list removes every tag.
func TestSetLLMTagsRequestValidate(t *testing.T) {
	if err := (&SetLLMTagsRequest{Tags: []string{}}).Validate(); err != nil {
		t.Errorf("Validate(empty) = %v, want nil", err)
	}
	if err := (&SetLLMTagsRequest{Tags: []string{"prod", "team-a"}}).Validate(); err != nil {
		t.Errorf("Validate(two tags) = %v, want nil", err)
	}
	if err := (&SetLLMTagsRequest{}).Validate(); err == nil {
		t.Error("Validate(nil tags) = nil, want an error")
	}
}

// The period is both ends inclusive and at most LLMUsageMaxDays long; 2024 is a
// leap year, so its whole year is exactly the limit. An empty group_by takes
// the API default; zero filters apply no filter.
func TestLLMUsageRequestValidate(t *testing.T) {
	valid := map[string]LLMUsageRequest{
		"single day":       {From: "2026-10-15", To: "2026-10-15"},
		"366 days":         {From: "2024-01-01", To: "2024-12-31"},
		"group_by none":    {From: "2026-10-01", To: "2026-10-31", GroupBy: LLMUsageGroupByNone},
		"group_by day":     {From: "2026-10-01", To: "2026-10-31", GroupBy: LLMUsageGroupByDay},
		"group_by model":   {From: "2026-10-01", To: "2026-10-31", GroupBy: LLMUsageGroupByModel},
		"group_by group":   {From: "2026-10-01", To: "2026-10-31", GroupBy: LLMUsageGroupByGroup},
		"group_by key":     {From: "2026-10-01", To: "2026-10-31", GroupBy: LLMUsageGroupByAccessKey},
		"positive filters": {From: "2026-10-01", To: "2026-10-31", GroupID: 3, AccessKeyID: 5, ModelID: 7},
	}
	for name, r := range valid {
		if err := r.Validate(); err != nil {
			t.Errorf("%s: Validate(%+v) = %v, want nil", name, r, err)
		}
	}

	invalid := map[string]LLMUsageRequest{
		"no from":            {To: "2026-10-31"},
		"no to":              {From: "2026-10-01"},
		"dotted from":        {From: "01.10.2026", To: "2026-10-31"},
		"bad month in to":    {From: "2026-10-01", To: "2026-13-01"},
		"reversed":           {From: "2026-10-31", To: "2026-10-01"},
		"367 days":           {From: "2024-01-01", To: "2025-01-01"},
		"unknown group_by":   {From: "2026-10-01", To: "2026-10-31", GroupBy: "week"},
		"cased group_by":     {From: "2026-10-01", To: "2026-10-31", GroupBy: "Day"},
		"negative group":     {From: "2026-10-01", To: "2026-10-31", GroupID: -1},
		"negative key":       {From: "2026-10-01", To: "2026-10-31", AccessKeyID: -1},
		"negative model":     {From: "2026-10-01", To: "2026-10-31", ModelID: -1},
		"date-time in range": {From: "2026-10-01T00:00:00Z", To: "2026-10-31"},
	}
	for name, r := range invalid {
		if err := r.Validate(); err == nil {
			t.Errorf("%s: Validate(%+v) = nil, want an error", name, r)
		}
	}
}

// The closed sets go on the wire as these snake_case strings; the spellings are
// fixed by the contract.
func TestLLMClosedSetsOnTheWire(t *testing.T) {
	fixed := map[string]string{
		string(LLMConnectionStateNotConnected):               "not_connected",
		string(LLMConnectionStateConnecting):                 "connecting",
		string(LLMConnectionStateConnected):                  "connected",
		string(LLMGroupStateActive):                          "active",
		string(LLMGroupStateDisabled):                        "disabled",
		string(LLMGroupStateDeleted):                         "deleted",
		string(LLMGroupOperationDisable):                     "disable",
		string(LLMGroupOperationChangeModels):                "change_models",
		string(LLMGroupModelOperationGrant):                  "grant",
		string(LLMGroupModelOperationRevoke):                 "revoke",
		string(LLMGroupDeploymentStateSendToDeploy):          "send_to_deploy",
		string(LLMGroupDeploymentStateDeployed):              "deployed",
		string(LLMGroupDeploymentStateDestroyed):             "destroyed",
		string(LLMGroupDeploymentStateError):                 "error",
		string(LLMAccessKeyStateRevoking):                    "revoking",
		string(LLMAccessKeyStateRevoked):                     "revoked",
		string(LLMAccessKeyStateRemoving):                    "removing",
		string(LLMAccessKeyStateRemoved):                     "removed",
		string(LLMAccessKeyOperationLockByClient):            "lock_by_client",
		string(LLMAccessKeyOperationUnlockByClient):          "unlock_by_client",
		string(LLMAccessKeyOperationLockByAdmin):             "lock_by_admin",
		string(LLMAccessKeyOperationUnlockByAdmin):           "unlock_by_admin",
		string(LLMAccessKeyOperationRemove):                  "remove",
		string(LLMModelKindText):                             "text",
		string(LLMModelKindRAG):                              "rag",
		string(LLMModelKindSpeech):                           "speech",
		string(LLMModelKindImage):                            "image",
		string(LLMModelKindVideo):                            "video",
		string(LLMModelContourLocal):                         "local",
		string(LLMModelContourExternal):                      "external",
		string(LLMModelUnitKindTokens):                       "tokens",
		string(LLMModelUnitKindUnits):                        "units",
		string(LLMModelUnitKindBytes):                        "bytes",
		string(LLMModelUnitKindNone):                         "none",
		string(LLMModelPriceUnitInputTokens):                 "input_tokens",
		string(LLMModelPriceUnitOutputTokens):                "output_tokens",
		string(LLMModelPriceUnitCacheReadTokens):             "cache_read_tokens",
		string(LLMModelPriceUnitCacheWriteTokens):            "cache_write_tokens",
		string(LLMModelPriceUnitImageOutputTokens):           "image_output_tokens",
		string(LLMModelPriceUnitWebSearchRequest):            "web_search_request",
		string(LLMModelPriceUnitRequest):                     "request",
		string(LLMModelPriceUnitCharacters):                  "characters",
		string(LLMModelPriceUnitAudioMinute):                 "audio_minute",
		string(LLMModelPriceUnitRequestMegabyte):             "request_megabyte",
		string(LLMModelPriceUnitResponseMegabyte):            "response_megabyte",
		string(LLMModelPriceUnitVideoSecond):                 "video_second",
		string(LLMModelPriceUnitVideoTokens):                 "video_tokens",
		string(LLMModelPriceUnitVideoMegapixelSecond):        "video_megapixel_second",
		string(LLMModelPriceUnitVideoImageInput):             "video_image_input",
		string(LLMModelPriceUnitVideoReferenceImage):         "video_reference_image",
		string(LLMModelPriceUnitVideoReferenceSecond):        "video_reference_second",
		string(LLMModelPriceUnitVideoMinimum):                "video_minimum",
		string(LLMCreditPriceBasisPackage):                   "package",
		string(LLMCreditPriceBasisProjectDiscount):           "project_discount",
		string(LLMCreditPriceBasisIndividual):                "individual",
		string(LLMCreditPriceBasisPayAsYouGo):                "pay_as_you_go",
		string(LLMTaxLabelNDSExcluded):                       "nds_excluded",
		string(LLMTaxLabelVATExcluded):                       "vat_excluded",
		string(LLMPackageStateScheduled):                     "scheduled",
		string(LLMPackageStateFinished):                      "finished",
		string(LLMPackageStateReplaced):                      "replaced",
		string(LLMPackageStateCancelled):                     "cancelled",
		string(LLMExhaustedPackageModeStop):                  "stop",
		string(LLMExhaustedPackageModeContinuePayAsYouGo):    "continue_pay_as_you_go",
		string(LLMPackageStopReasonPackageExhausted):         "package_exhausted",
		string(LLMPackageStopReasonPayAsYouGoLimitExhausted): "pay_as_you_go_limit_exhausted",
		string(LLMPackageStopStateStopped):                   "stopped",
		string(LLMPackageStopStateStarting):                  "starting",
		string(LLMPackageStopStateStarted):                   "started",
		string(LLMUsageGroupByAccessKey):                     "access_key",
		string(LLMUsageCollectionStateNotCollected):          "not_collected",
		string(LLMUsageCollectionStateCollected):             "collected",
		string(LLMUsageCollectionStateFailed):                "failed",
	}
	for got, want := range fixed {
		if got != want {
			t.Errorf("wire value = %q, want %q", got, want)
		}
	}

	// Same spelling in different sets: a map keyed by the value cannot hold both.
	same := map[string]struct{ got, want string }{
		"LLMAccessKeyStateActive": {string(LLMAccessKeyStateActive), "active"},
		"LLMPackageStateActive":   {string(LLMPackageStateActive), "active"},
		"LLMUsageGroupByNone":     {string(LLMUsageGroupByNone), "none"},
		"LLMUsageGroupByDay":      {string(LLMUsageGroupByDay), "day"},
		"LLMUsageGroupByModel":    {string(LLMUsageGroupByModel), "model"},
		"LLMUsageGroupByGroup":    {string(LLMUsageGroupByGroup), "group"},
	}
	for name, c := range same {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", name, c.got, c.want)
		}
	}
}
