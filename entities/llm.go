package entities

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf16"
)

// LLMGroupID identifies an LLM group of the project.
type LLMGroupID int

// Validate reports an error unless the ID is positive.
func (id LLMGroupID) Validate() error {
	if id <= 0 {
		return fmt.Errorf("LLM group ID must be positive, got %d", id)
	}
	return nil
}

// LLMAccessKeyID identifies an LLM access key (version 2) of a group.
type LLMAccessKeyID int

// Validate reports an error unless the ID is positive.
func (id LLMAccessKeyID) Validate() error {
	if id <= 0 {
		return fmt.Errorf("LLM access key ID must be positive, got %d", id)
	}
	return nil
}

// LLMModelID identifies a model of the LLM catalog.
type LLMModelID int

// Validate reports an error unless the ID is positive.
func (id LLMModelID) Validate() error {
	if id <= 0 {
		return fmt.Errorf("LLM model ID must be positive, got %d", id)
	}
	return nil
}

// LLMNameMaxLength is the longest group or access key name the API accepts,
// counted after trimming the surrounding white space.
const LLMNameMaxLength = 128

// LLMUsageMaxDays is the longest usage period, both ends included.
const LLMUsageMaxDays = 366

// LLMDateLayout is the layout of the usage period dates (YYYY-MM-DD).
const LLMDateLayout = "2006-01-02"

// LLMConnectionState is the state of the LLM service connection of the project.
type LLMConnectionState string

const (
	LLMConnectionStateNotConnected LLMConnectionState = "not_connected"
	LLMConnectionStateConnecting   LLMConnectionState = "connecting"
	LLMConnectionStateConnected    LLMConnectionState = "connected"
)

// LLMGroupState is the state of an LLM group.
type LLMGroupState string

const (
	LLMGroupStateActive   LLMGroupState = "active"
	LLMGroupStateDisabled LLMGroupState = "disabled"
	LLMGroupStateDeleted  LLMGroupState = "deleted"
	// LLMGroupStateSuspendedByProjectQuota replaces active while the LLM service
	// of the project is suspended: the project quota is used up.
	LLMGroupStateSuspendedByProjectQuota LLMGroupState = "suspended_by_project_quota"
)

// LLMGroupOperation is the operation a group is waiting on.
type LLMGroupOperation string

const (
	LLMGroupOperationDisable      LLMGroupOperation = "disable"
	LLMGroupOperationChangeModels LLMGroupOperation = "change_models"
)

// LLMGroupModelOperation is the operation a model of a group is waiting on.
type LLMGroupModelOperation string

const (
	LLMGroupModelOperationGrant  LLMGroupModelOperation = "grant"
	LLMGroupModelOperationRevoke LLMGroupModelOperation = "revoke"
)

// LLMGroupDeploymentState is the state of the group deployment on the platform.
type LLMGroupDeploymentState string

const (
	LLMGroupDeploymentStateSendToDeploy LLMGroupDeploymentState = "send_to_deploy"
	LLMGroupDeploymentStateDeployed     LLMGroupDeploymentState = "deployed"
	LLMGroupDeploymentStateDestroyed    LLMGroupDeploymentState = "destroyed"
	LLMGroupDeploymentStateError        LLMGroupDeploymentState = "error"
)

// LLMAccessKeyState is the state of an LLM access key.
type LLMAccessKeyState string

const (
	LLMAccessKeyStateActive   LLMAccessKeyState = "active"
	LLMAccessKeyStateRevoking LLMAccessKeyState = "revoking"
	LLMAccessKeyStateRevoked  LLMAccessKeyState = "revoked"
	LLMAccessKeyStateRemoving LLMAccessKeyState = "removing"
	LLMAccessKeyStateRemoved  LLMAccessKeyState = "removed"
	// LLMAccessKeyStateSuspendedByProjectQuota replaces active while the LLM
	// service of the project is suspended: the project quota is used up.
	LLMAccessKeyStateSuspendedByProjectQuota LLMAccessKeyState = "suspended_by_project_quota"
)

// LLMAccessKeyOperation is the operation an access key is waiting on.
type LLMAccessKeyOperation string

const (
	LLMAccessKeyOperationLockByClient   LLMAccessKeyOperation = "lock_by_client"
	LLMAccessKeyOperationUnlockByClient LLMAccessKeyOperation = "unlock_by_client"
	LLMAccessKeyOperationLockByAdmin    LLMAccessKeyOperation = "lock_by_admin"
	LLMAccessKeyOperationUnlockByAdmin  LLMAccessKeyOperation = "unlock_by_admin"
	LLMAccessKeyOperationRemove         LLMAccessKeyOperation = "remove"
)

// LLMModelKind is the kind of a catalog model.
type LLMModelKind string

const (
	LLMModelKindText   LLMModelKind = "text"
	LLMModelKindRAG    LLMModelKind = "rag"
	LLMModelKindSpeech LLMModelKind = "speech"
	LLMModelKindImage  LLMModelKind = "image"
	LLMModelKindVideo  LLMModelKind = "video"
)

// LLMModelContour tells a model served in the local contour from an external provider.
type LLMModelContour string

const (
	LLMModelContourLocal    LLMModelContour = "local"
	LLMModelContourExternal LLMModelContour = "external"
)

// LLMModelUnitKind is the unit a model is billed in.
type LLMModelUnitKind string

const (
	LLMModelUnitKindTokens LLMModelUnitKind = "tokens"
	LLMModelUnitKindUnits  LLMModelUnitKind = "units"
	LLMModelUnitKindBytes  LLMModelUnitKind = "bytes"
	LLMModelUnitKindNone   LLMModelUnitKind = "none"
)

// LLMModelPriceUnit is the unit a model price is quoted for.
type LLMModelPriceUnit string

const (
	LLMModelPriceUnitInputTokens          LLMModelPriceUnit = "input_tokens"
	LLMModelPriceUnitOutputTokens         LLMModelPriceUnit = "output_tokens"
	LLMModelPriceUnitCacheReadTokens      LLMModelPriceUnit = "cache_read_tokens"
	LLMModelPriceUnitCacheWriteTokens     LLMModelPriceUnit = "cache_write_tokens"
	LLMModelPriceUnitImageOutputTokens    LLMModelPriceUnit = "image_output_tokens"
	LLMModelPriceUnitWebSearchRequest     LLMModelPriceUnit = "web_search_request"
	LLMModelPriceUnitRequest              LLMModelPriceUnit = "request"
	LLMModelPriceUnitCharacters           LLMModelPriceUnit = "characters"
	LLMModelPriceUnitAudioMinute          LLMModelPriceUnit = "audio_minute"
	LLMModelPriceUnitRequestMegabyte      LLMModelPriceUnit = "request_megabyte"
	LLMModelPriceUnitResponseMegabyte     LLMModelPriceUnit = "response_megabyte"
	LLMModelPriceUnitVideoSecond          LLMModelPriceUnit = "video_second"
	LLMModelPriceUnitVideoTokens          LLMModelPriceUnit = "video_tokens"
	LLMModelPriceUnitVideoMegapixelSecond LLMModelPriceUnit = "video_megapixel_second"
	LLMModelPriceUnitVideoImageInput      LLMModelPriceUnit = "video_image_input"
	LLMModelPriceUnitVideoReferenceImage  LLMModelPriceUnit = "video_reference_image"
	LLMModelPriceUnitVideoReferenceSecond LLMModelPriceUnit = "video_reference_second"
	LLMModelPriceUnitVideoMinimum         LLMModelPriceUnit = "video_minimum"
)

// LLMCreditPriceBasis is where the price of a credit comes from.
type LLMCreditPriceBasis string

const (
	LLMCreditPriceBasisPackage         LLMCreditPriceBasis = "package"
	LLMCreditPriceBasisProjectDiscount LLMCreditPriceBasis = "project_discount"
	LLMCreditPriceBasisIndividual      LLMCreditPriceBasis = "individual"
	LLMCreditPriceBasisPayAsYouGo      LLMCreditPriceBasis = "pay_as_you_go"
)

// LLMTaxLabel is the tax caption of a price.
type LLMTaxLabel string

const (
	LLMTaxLabelNDSExcluded LLMTaxLabel = "nds_excluded"
	LLMTaxLabelVATExcluded LLMTaxLabel = "vat_excluded"
)

// LLMPackageSource tells a package of the partner tariff from an individual
// package of the project.
type LLMPackageSource string

const (
	LLMPackageSourceCatalog    LLMPackageSource = "catalog"
	LLMPackageSourceIndividual LLMPackageSource = "individual"
)

// LLMPackageState is the state of a credit package.
type LLMPackageState string

const (
	LLMPackageStateScheduled LLMPackageState = "scheduled"
	LLMPackageStateActive    LLMPackageState = "active"
	LLMPackageStateFinished  LLMPackageState = "finished"
	LLMPackageStateReplaced  LLMPackageState = "replaced"
	LLMPackageStateCancelled LLMPackageState = "cancelled"
)

// LLMExhaustedPackageMode is what happens once the package is used up.
type LLMExhaustedPackageMode string

const (
	LLMExhaustedPackageModeStop               LLMExhaustedPackageMode = "stop"
	LLMExhaustedPackageModeContinuePayAsYouGo LLMExhaustedPackageMode = "continue_pay_as_you_go"
)

// LLMPackageStopReason is why the service of the project was stopped.
type LLMPackageStopReason string

const (
	LLMPackageStopReasonPackageExhausted         LLMPackageStopReason = "package_exhausted"
	LLMPackageStopReasonPayAsYouGoLimitExhausted LLMPackageStopReason = "pay_as_you_go_limit_exhausted"
	LLMPackageStopReasonProjectQuotaExhausted    LLMPackageStopReason = "project_quota_exhausted"
)

// LLMPackageStopState is the state of a stop of the service.
type LLMPackageStopState string

const (
	LLMPackageStopStateStopped  LLMPackageStopState = "stopped"
	LLMPackageStopStateStarting LLMPackageStopState = "starting"
	LLMPackageStopStateStarted  LLMPackageStopState = "started"
)

// LLMUsageGroupBy is the breakdown of the usage rows.
type LLMUsageGroupBy string

const (
	LLMUsageGroupByNone      LLMUsageGroupBy = "none"
	LLMUsageGroupByDay       LLMUsageGroupBy = "day"
	LLMUsageGroupByModel     LLMUsageGroupBy = "model"
	LLMUsageGroupByGroup     LLMUsageGroupBy = "group"
	LLMUsageGroupByAccessKey LLMUsageGroupBy = "access_key"
)

// LLMUsageCollectionState tells "not collected yet" from "no usage".
type LLMUsageCollectionState string

const (
	LLMUsageCollectionStateNotCollected LLMUsageCollectionState = "not_collected"
	LLMUsageCollectionStateCollected    LLMUsageCollectionState = "collected"
	LLMUsageCollectionStateFailed       LLMUsageCollectionState = "failed"
)

// LLMConnection is the LLM service connection of the project.
type LLMConnection struct {
	State LLMConnectionState `json:"state"`
	// TaskID is the connection task while State is connecting.
	TaskID       *string `json:"task_id,omitempty"`
	GroupName    *string `json:"group_name,omitempty"`
	FirstKeyName *string `json:"first_key_name,omitempty"`
}

// LLMSpendAmount is an amount spent in credits and, where priced, in money.
type LLMSpendAmount struct {
	Credits float64  `json:"credits"`
	Cost    *float64 `json:"cost,omitempty"`
}

// LLMGroupModel is a catalog model granted to a group.
type LLMGroupModel struct {
	ModelID          LLMModelID              `json:"model_id"`
	TechTitle        string                  `json:"tech_title"`
	Name             string                  `json:"name"`
	PendingOperation *LLMGroupModelOperation `json:"pending_operation,omitempty"`
}

// LLMGroup is an LLM group of the project.
//
// A group still being created carries only Name, CreatedAt and TaskID: ID and
// State are nil until its task completes.
type LLMGroup struct {
	ID               *LLMGroupID        `json:"id,omitempty"`
	Name             string             `json:"name"`
	State            *LLMGroupState     `json:"state,omitempty"`
	Tags             []string           `json:"tags,omitempty"`
	Models           []LLMGroupModel    `json:"models,omitempty"`
	PendingOperation *LLMGroupOperation `json:"pending_operation,omitempty"`
	IsReadOnly       *bool              `json:"is_read_only,omitempty"`
	CreatedAt        string             `json:"created_at"`
	TaskID           *string            `json:"task_id,omitempty"`
}

// LLMGroupSpend is the spend of a group over the current day, week and month.
type LLMGroupSpend struct {
	Today *LLMSpendAmount `json:"today,omitempty"`
	Week  *LLMSpendAmount `json:"week,omitempty"`
	Month *LLMSpendAmount `json:"month,omitempty"`
}

// LLMGroupDetails is a group read on its own: the list form plus the
// deployment state and the spend.
type LLMGroupDetails struct {
	LLMGroup
	DeploymentState   LLMGroupDeploymentState `json:"deployment_state"`
	Spend             *LLMGroupSpend          `json:"spend,omitempty"`
	TotalSpentCredits int64                   `json:"total_spent_credits"`
	UpdatedAt         string                  `json:"updated_at"`
}

// LLMAccessKeyLocks are the locks of an access key by reason.
type LLMAccessKeyLocks struct {
	ByClient     bool    `json:"by_client"`
	ClientReason *string `json:"client_reason,omitempty"`
	ByQuota      bool    `json:"by_quota"`
	ByAdmin      bool    `json:"by_admin"`
	ByBalance    bool    `json:"by_balance"`
	ByPlatform   bool    `json:"by_platform"`
}

// LLMAccessKey is an LLM access key (version 2) of a group.
//
// A key still being issued carries only GroupID, GroupName, Name, CreatedAt and
// TaskID: ID is nil until its task completes. GroupName and MonthSpend come in
// the list only.
type LLMAccessKey struct {
	ID                *LLMAccessKeyID        `json:"id,omitempty"`
	GroupID           LLMGroupID             `json:"group_id"`
	GroupName         *string                `json:"group_name,omitempty"`
	Name              string                 `json:"name"`
	State             *LLMAccessKeyState     `json:"state,omitempty"`
	Locks             *LLMAccessKeyLocks     `json:"locks,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	ConsumedCredits   *int64                 `json:"consumed_credits,omitempty"`
	MonthSpend        *LLMSpendAmount        `json:"month_spend,omitempty"`
	ExpiresAt         *string                `json:"expires_at,omitempty"`
	LastUsedAt        *string                `json:"last_used_at,omitempty"`
	IsSecretAvailable *bool                  `json:"is_secret_available,omitempty"`
	PendingOperation  *LLMAccessKeyOperation `json:"pending_operation,omitempty"`
	IsGroupReadOnly   *bool                  `json:"is_group_read_only,omitempty"`
	CreatedAt         string                 `json:"created_at"`
	TaskID            *string                `json:"task_id,omitempty"`
}

// LLMModelPrice is the price of one unit of a model, in credits and, where
// priced, in money.
type LLMModelPrice struct {
	Unit    LLMModelPriceUnit `json:"unit"`
	Variant *string           `json:"variant,omitempty"`
	Credits float64           `json:"credits"`
	Money   *float64          `json:"money,omitempty"`
}

// LLMModelStartingPrice is the lowest price a model starts from.
type LLMModelStartingPrice struct {
	Provider           string         `json:"provider"`
	Input              LLMModelPrice  `json:"input"`
	Output             *LLMModelPrice `json:"output,omitempty"`
	IsFromLiveProvider bool           `json:"is_from_live_provider"`
}

// LLMCreditPrice is the price of one credit for the project.
type LLMCreditPrice struct {
	Basis          LLMCreditPriceBasis `json:"basis"`
	PricePerCredit float64             `json:"price_per_credit"`
	// Currency is the currency code, for example RUB.
	Currency     string       `json:"currency"`
	TaxLabel     *LLMTaxLabel `json:"tax_label,omitempty"`
	PackageName  *string      `json:"package_name,omitempty"`
	ValidThrough *string      `json:"valid_through,omitempty"`
}

// LLMModel is a model of the project catalog. Name and Description are in English.
type LLMModel struct {
	ID               LLMModelID       `json:"id"`
	TechTitle        string           `json:"tech_title"`
	Name             string           `json:"name"`
	Description      string           `json:"description,omitempty"`
	Vendor           string           `json:"vendor"`
	Kind             LLMModelKind     `json:"kind"`
	Contour          LLMModelContour  `json:"contour"`
	StorageCountries []string         `json:"storage_countries,omitempty"`
	IsPIIProtected   bool             `json:"is_pii_protected"`
	ContextSize      int              `json:"context_size"`
	UnitKind         LLMModelUnitKind `json:"unit_kind"`
	// Endpoint is the inference address of the model.
	Endpoint            string                 `json:"endpoint"`
	StartingPrice       *LLMModelStartingPrice `json:"starting_price,omitempty"`
	IsAvailableToGroups bool                   `json:"is_available_to_groups"`
	PricesCollectedAt   *string                `json:"prices_collected_at,omitempty"`
}

// LLMModelProtectedPrice is the price of a model with personal data protection:
// the model, the recognition and their total.
type LLMModelProtectedPrice struct {
	Model       []LLMModelPrice `json:"model,omitempty"`
	Recognition LLMModelPrice   `json:"recognition"`
	Total       []LLMModelPrice `json:"total,omitempty"`
}

// LLMModelProvider is a provider serving a model.
type LLMModelProvider struct {
	Slug                      string          `json:"slug"`
	Name                      string          `json:"name"`
	Variant                   *string         `json:"variant,omitempty"`
	ContextLength             *int            `json:"context_length,omitempty"`
	MaxCompletionTokens       *int            `json:"max_completion_tokens,omitempty"`
	Prices                    []LLMModelPrice `json:"prices,omitempty"`
	Discount                  float64         `json:"discount"`
	IsLive                    bool            `json:"is_live"`
	SupportsTools             bool            `json:"supports_tools"`
	SupportsReasoning         bool            `json:"supports_reasoning"`
	SupportsStructuredOutputs bool            `json:"supports_structured_outputs"`
	SupportsWebSearch         bool            `json:"supports_web_search"`
}

// LLMModelDetails is a model read on its own: the catalog form plus the prices
// by unit and variant, the providers and the capabilities.
type LLMModelDetails struct {
	LLMModel
	CreditPrice       *LLMCreditPrice         `json:"credit_price,omitempty"`
	DefaultProvider   string                  `json:"default_provider"`
	Prices            []LLMModelPrice         `json:"prices,omitempty"`
	ProtectedPrice    *LLMModelProtectedPrice `json:"protected_price,omitempty"`
	Providers         []LLMModelProvider      `json:"providers,omitempty"`
	SupportsStreaming *bool                   `json:"supports_streaming,omitempty"`
	SupportsTools     *bool                   `json:"supports_tools,omitempty"`
}

// LLMModelCatalog is the model catalog of the project with the credit price and
// the time the prices were collected.
type LLMModelCatalog struct {
	Models            []LLMModel      `json:"models,omitempty"`
	CreditPrice       *LLMCreditPrice `json:"credit_price,omitempty"`
	PricesCollectedAt *string         `json:"prices_collected_at,omitempty"`
}

// LLMPackage is a credit package of the project with the choice of what
// happens once it is used up.
type LLMPackage struct {
	ID            int     `json:"id"`
	TechTitle     string  `json:"tech_title"`
	VolumeCredits float64 `json:"volume_credits"`
	// DiscountPercent is nil when the package has no price.
	DiscountPercent *float64 `json:"discount_percent,omitempty"`
	// Price is nil when the package has no price: usage in credits only.
	Price                  *float64                `json:"price,omitempty"`
	FullMonthVolumeCredits *float64                `json:"full_month_volume_credits,omitempty"`
	FullMonthPrice         *float64                `json:"full_month_price,omitempty"`
	PeriodFrom             string                  `json:"period_from"`
	PeriodTo               string                  `json:"period_to"`
	AutoRenew              bool                    `json:"auto_renew"`
	ExhaustedPackageMode   LLMExhaustedPackageMode `json:"exhausted_package_mode"`
	PayAsYouGoLimitCredits *float64                `json:"pay_as_you_go_limit_credits,omitempty"`
	State                  LLMPackageState         `json:"state"`
}

// LLMPackageUsage is the spend of the open package period. PeriodTo is exclusive.
type LLMPackageUsage struct {
	PeriodFrom           string   `json:"period_from"`
	PeriodTo             string   `json:"period_to"`
	TotalCredits         float64  `json:"total_credits"`
	PackageVolumeCredits *float64 `json:"package_volume_credits,omitempty"`
	InPackageCredits     float64  `json:"in_package_credits"`
	OverPackageCredits   float64  `json:"over_package_credits"`
}

// LLMPackageStop is a stop of the service by the package, the pay-as-you-go
// limit or the project quota.
type LLMPackageStop struct {
	StoppedAt string               `json:"stopped_at"`
	Reason    LLMPackageStopReason `json:"reason"`
	State     LLMPackageStopState  `json:"state"`
}

// LLMCurrentPackage is the package state of the project. Package is nil when
// the project has no current package. PricePerCredit, Currency and TaxLabel
// are nil for a project without the Finances option: usage in credits only.
type LLMCurrentPackage struct {
	Package          *LLMPackage     `json:"package,omitempty"`
	Usage            LLMPackageUsage `json:"usage"`
	PricePerCredit   *float64        `json:"price_per_credit,omitempty"`
	Currency         *string         `json:"currency,omitempty"`
	TaxLabel         *LLMTaxLabel    `json:"tax_label,omitempty"`
	PackagesDisabled bool            `json:"packages_disabled"`
	Stop             *LLMPackageStop `json:"stop,omitempty"`
}

// LLMPackagePlanMonthTerms are the terms of a plan for the rest of the current
// month. Price is nil when the plan has no price.
type LLMPackagePlanMonthTerms struct {
	PeriodFrom    string   `json:"period_from"`
	PeriodTo      string   `json:"period_to"`
	VolumeCredits float64  `json:"volume_credits"`
	Price         *float64 `json:"price,omitempty"`
}

// LLMPackagePlan is a package available to the project: a package of the
// partner tariff or an individual package of the project. Description is in
// English. Price, DiscountPercent, Currency and TaxLabel are nil for a project
// without the Finances option: usage in credits only.
type LLMPackagePlan struct {
	ID              int                       `json:"id"`
	Source          LLMPackageSource          `json:"source"`
	TechTitle       string                    `json:"tech_title"`
	Description     string                    `json:"description,omitempty"`
	VolumeCredits   float64                   `json:"volume_credits"`
	DiscountPercent *float64                  `json:"discount_percent,omitempty"`
	Price           *float64                  `json:"price,omitempty"`
	Currency        *string                   `json:"currency,omitempty"`
	TaxLabel        *LLMTaxLabel              `json:"tax_label,omitempty"`
	CurrentMonth    *LLMPackagePlanMonthTerms `json:"current_month,omitempty"`
}

// LLMUsageAmounts are the usage sums of the totals and of every row.
type LLMUsageAmounts struct {
	Credits         float64  `json:"credits"`
	LocalCredits    float64  `json:"local_credits"`
	ExternalCredits float64  `json:"external_credits"`
	Cost            *float64 `json:"cost,omitempty"`
	InTokens        int64    `json:"in_tokens"`
	OutTokens       int64    `json:"out_tokens"`
	CachedTokens    int64    `json:"cached_tokens"`
}

// LLMUsageItem is a usage row: the sums plus the fields of its breakdown —
// Date for day, the Model* fields for model, the Group* fields for group, the
// AccessKey* fields for access_key.
type LLMUsageItem struct {
	LLMUsageAmounts
	Date           *string         `json:"date,omitempty"`
	ModelID        *LLMModelID     `json:"model_id,omitempty"`
	ModelTechTitle *string         `json:"model_tech_title,omitempty"`
	ModelName      *string         `json:"model_name,omitempty"`
	GroupID        *LLMGroupID     `json:"group_id,omitempty"`
	GroupName      *string         `json:"group_name,omitempty"`
	AccessKeyID    *LLMAccessKeyID `json:"access_key_id,omitempty"`
	AccessKeyName  *string         `json:"access_key_name,omitempty"`
}

// LLMUsageCollection is the state of the usage collection.
type LLMUsageCollection struct {
	State       LLMUsageCollectionState `json:"state"`
	CollectedAt *string                 `json:"collected_at,omitempty"`
	AttemptedAt *string                 `json:"attempted_at,omitempty"`
}

// LLMUsage is the usage of the project over a period. Items is empty for the
// none breakdown; day rows go by date ascending, the others by credits descending.
type LLMUsage struct {
	From        string             `json:"from"`
	To          string             `json:"to"`
	GroupBy     LLMUsageGroupBy    `json:"group_by"`
	Totals      LLMUsageAmounts    `json:"totals"`
	Items       []LLMUsageItem     `json:"items,omitempty"`
	CreditPrice *LLMCreditPrice    `json:"credit_price,omitempty"`
	Collection  LLMUsageCollection `json:"collection"`
}

// LLMRequestLogItem is a request made with an access key of the project.
type LLMRequestLogItem struct {
	// OccurredAt is in UTC.
	OccurredAt     string         `json:"occurred_at"`
	GroupID        LLMGroupID     `json:"group_id"`
	AccessKeyID    LLMAccessKeyID `json:"access_key_id"`
	ModelID        *LLMModelID    `json:"model_id,omitempty"`
	ModelTechTitle string         `json:"model_tech_title"`
	ResponseCode   int            `json:"response_code"`
	DurationMs     int            `json:"duration_ms"`
	InTokens       int64          `json:"in_tokens"`
	OutTokens      int64          `json:"out_tokens"`
	CachedTokens   int64          `json:"cached_tokens"`
	Credits        float64        `json:"credits"`
}

// LLMRequestLog is the request log of the project access keys over the last 7
// days, up to 200 rows. SourceConnected is false while the log source is not
// connected.
type LLMRequestLog struct {
	SourceConnected bool                `json:"source_connected"`
	Requests        []LLMRequestLogItem `json:"requests,omitempty"`
}

// validateLLMName checks a group or access key name the way the API does:
// required and at most LLMNameMaxLength UTF-16 code units once trimmed.
func validateLLMName(field, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("%s is required", field)
	}
	if n := len(utf16.Encode([]rune(trimmed))); n > LLMNameMaxLength {
		return fmt.Errorf("%s must be at most %d characters, got %d", field, LLMNameMaxLength, n)
	}
	return nil
}

// ConnectLLMRequest connects the LLM service: the first group and its first access key.
type ConnectLLMRequest struct {
	GroupName    string `json:"group_name"`
	FirstKeyName string `json:"first_key_name"`
}

// Validate checks both names.
func (r *ConnectLLMRequest) Validate() error {
	if err := validateLLMName("group name", r.GroupName); err != nil {
		return err
	}
	return validateLLMName("first key name", r.FirstKeyName)
}

// CreateLLMGroupRequest creates an LLM group.
type CreateLLMGroupRequest struct {
	Name string `json:"name"`
}

// Validate checks the name.
func (r *CreateLLMGroupRequest) Validate() error {
	return validateLLMName("group name", r.Name)
}

// SetLLMGroupModelsRequest replaces the whole set of the group models.
// An empty non-nil ModelIDs revokes every model.
type SetLLMGroupModelsRequest struct {
	ModelIDs []LLMModelID `json:"model_ids"`
}

// Validate requires the set and positive model IDs.
func (r *SetLLMGroupModelsRequest) Validate() error {
	if r.ModelIDs == nil {
		return fmt.Errorf("model IDs are required; an empty list revokes every model")
	}
	for _, id := range r.ModelIDs {
		if err := id.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// SetLLMTagsRequest replaces the whole tag set of a group or of an access key.
// An empty non-nil Tags removes every tag.
type SetLLMTagsRequest struct {
	Tags []string `json:"tags"`
}

// Validate requires the set.
func (r *SetLLMTagsRequest) Validate() error {
	if r.Tags == nil {
		return fmt.Errorf("tags are required; an empty list removes every tag")
	}
	return nil
}

// IssueLLMAccessKeyRequest issues an access key in a group.
type IssueLLMAccessKeyRequest struct {
	Name string `json:"name"`
}

// Validate checks the name.
func (r *IssueLLMAccessKeyRequest) Validate() error {
	return validateLLMName("access key name", r.Name)
}

// LockLLMAccessKeyRequest locks an access key by the client; Reason is optional.
type LockLLMAccessKeyRequest struct {
	Reason string `json:"reason,omitempty"`
}

// LLMUsageRequest selects the usage read: the period, the breakdown and the filters.
//
// From and To are required YYYY-MM-DD dates, both inclusive. An empty GroupBy
// takes the API default, none. A zero GroupID, AccessKeyID or ModelID applies
// no filter.
type LLMUsageRequest struct {
	From        string
	To          string
	GroupBy     LLMUsageGroupBy
	GroupID     LLMGroupID
	AccessKeyID LLMAccessKeyID
	ModelID     LLMModelID
}

// Validate checks the period, the breakdown and the filters.
func (r *LLMUsageRequest) Validate() error {
	if r.From == "" {
		return fmt.Errorf("from is required")
	}
	if r.To == "" {
		return fmt.Errorf("to is required")
	}
	from, err := time.Parse(LLMDateLayout, r.From)
	if err != nil {
		return fmt.Errorf("from %q is not a YYYY-MM-DD date", r.From)
	}
	to, err := time.Parse(LLMDateLayout, r.To)
	if err != nil {
		return fmt.Errorf("to %q is not a YYYY-MM-DD date", r.To)
	}
	if to.Before(from) {
		return fmt.Errorf("to %s is earlier than from %s", r.To, r.From)
	}
	if days := int(to.Sub(from).Hours()/24) + 1; days > LLMUsageMaxDays {
		return fmt.Errorf("the period is %d days, at most %d are allowed", days, LLMUsageMaxDays)
	}
	switch r.GroupBy {
	case "", LLMUsageGroupByNone, LLMUsageGroupByDay, LLMUsageGroupByModel, LLMUsageGroupByGroup, LLMUsageGroupByAccessKey:
	default:
		return fmt.Errorf("unsupported group_by %q", r.GroupBy)
	}
	if r.GroupID < 0 {
		return fmt.Errorf("group ID filter must not be negative, got %d", r.GroupID)
	}
	if r.AccessKeyID < 0 {
		return fmt.Errorf("access key ID filter must not be negative, got %d", r.AccessKeyID)
	}
	if r.ModelID < 0 {
		return fmt.Errorf("model ID filter must not be negative, got %d", r.ModelID)
	}
	return nil
}
