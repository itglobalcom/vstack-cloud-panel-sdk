package entities

import "fmt"

// Backup restore point states. These are the values returned in
// BackupRestorePoint.State.
const (
	// BackupRestorePointStateInProgress is a copy that is planned or being taken.
	BackupRestorePointStateInProgress = "in_progress"
	// BackupRestorePointStateActive is a finished copy the server can be restored from.
	BackupRestorePointStateActive = "active"
	// BackupRestorePointStateFailed is a copy the platform failed to take.
	BackupRestorePointStateFailed = "failed"
)

// Backup retention rules. These are the values returned in
// BackupRestorePoint.RetainedBy.
const (
	BackupRetentionRuleDaily   = "daily"
	BackupRetentionRuleWeekly  = "weekly"
	BackupRetentionRuleMonthly = "monthly"
)

// BackupDayOfMonthLast is the BackupMonthlyRule.DayOfMonth value for the last
// day of the month.
const BackupDayOfMonthLast = "last"

// BackupStorageCatalog lists the backup storages available to a server (or to a
// server order) together with the partner's schedule limits.
type BackupStorageCatalog struct {
	Storages []BackupStorage `json:"storages,omitempty"`
	Limits   *BackupLimits   `json:"limits,omitempty"`
}

// BackupStorage is a backup storage a schedule rule can put copies into.
type BackupStorage struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	SortOrder   int    `json:"sort_order"`
	// PricePerGB is the price of storing one gigabyte of copies in the project
	// tariff.
	PricePerGB float64 `json:"price_per_gb"`
	// LocationID is the techTitle of the location the storage lives in.
	LocationID string `json:"location_id"`
	// RestoreLocationIDs are the techTitles of the locations a copy from this
	// storage can be restored nearby into.
	RestoreLocationIDs []string `json:"restore_location_ids,omitempty"`
}

// BackupLimits are the partner's limits a backup schedule must fit into.
type BackupLimits struct {
	// ScheduleWindowFromHour and ScheduleWindowToHour bound the hour the
	// copies may start at.
	ScheduleWindowFromHour int               `json:"schedule_window_from_hour"`
	ScheduleWindowToHour   int               `json:"schedule_window_to_hour"`
	Daily                  *BackupRuleLimits `json:"daily,omitempty"`
	Weekly                 *BackupRuleLimits `json:"weekly,omitempty"`
	Monthly                *BackupRuleLimits `json:"monthly,omitempty"`
}

// BackupRuleLimits bound the number of copies a single schedule rule keeps.
type BackupRuleLimits struct {
	MaxKeep     int `json:"max_keep"`
	DefaultKeep int `json:"default_keep"`
}

// ServerBackup is the state of the backup service of a server.
type ServerBackup struct {
	Enabled bool `json:"enabled"`
	// Schedule is nil while the service is not enabled, and also for an enabled
	// service of a server that has no schedule (the legacy backup model).
	Schedule *BackupSchedule `json:"schedule,omitempty"`
}

// BackupSchedule is the backup schedule of a server: the time copies are taken
// at and up to three rules. It is the body of enabling and updating the backup
// service and of the backup block of a server order. A nil rule is disabled.
type BackupSchedule struct {
	// Hour and Minute are the time the copies start at.
	Hour    int                `json:"hour"`
	Minute  int                `json:"minute"`
	Daily   *BackupRule        `json:"daily,omitempty"`
	Weekly  *BackupWeeklyRule  `json:"weekly,omitempty"`
	Monthly *BackupMonthlyRule `json:"monthly,omitempty"`
}

// BackupRule is a daily schedule rule; it is embedded in the weekly and monthly
// rules.
type BackupRule struct {
	// Keep is the number of copies the rule retains.
	Keep            int `json:"keep"`
	BackupStorageID int `json:"backup_storage_id"`
}

// BackupWeeklyRule is a weekly schedule rule.
type BackupWeeklyRule struct {
	BackupRule
	// Weekday is the ISO day of week: 1 is Monday, 7 is Sunday.
	Weekday int `json:"weekday"`
}

// BackupMonthlyRule is a monthly schedule rule.
type BackupMonthlyRule struct {
	BackupRule
	// DayOfMonth is the day of month the copy is taken on: "1" to "28" or
	// BackupDayOfMonthLast.
	DayOfMonth string `json:"day_of_month"`
}

// Validate validates the backup schedule
func (r *BackupSchedule) Validate() error {
	if r.Hour < 0 || r.Hour > 23 {
		return fmt.Errorf("backup hour must be between 0 and 23")
	}
	if r.Minute < 0 || r.Minute > 59 {
		return fmt.Errorf("backup minute must be between 0 and 59")
	}
	if r.Daily == nil && r.Weekly == nil && r.Monthly == nil {
		return fmt.Errorf("at least one backup rule is required")
	}
	if r.Daily != nil && r.Daily.BackupStorageID <= 0 {
		return fmt.Errorf("daily backup storage ID must be greater than 0")
	}
	if r.Weekly != nil {
		if r.Weekly.BackupStorageID <= 0 {
			return fmt.Errorf("weekly backup storage ID must be greater than 0")
		}
		if r.Weekly.Weekday < 1 || r.Weekly.Weekday > 7 {
			return fmt.Errorf("weekly backup weekday must be between 1 and 7")
		}
	}
	if r.Monthly != nil {
		if r.Monthly.BackupStorageID <= 0 {
			return fmt.Errorf("monthly backup storage ID must be greater than 0")
		}
		if r.Monthly.DayOfMonth == "" {
			return fmt.Errorf("monthly backup day of month is required")
		}
	}
	return nil
}

// BackupRestorePoint is a copy of a server.
type BackupRestorePoint struct {
	ID       int    `json:"id"`
	ServerID string `json:"server_id"`
	Name     string `json:"name"`
	SizeMB   int    `json:"size_mb"`
	// BackupStorageID is nil for a copy that is not bound to a storage.
	BackupStorageID *int `json:"backup_storage_id,omitempty"`
	// State is one of the BackupRestorePointState* values.
	State   string `json:"state"`
	Created string `json:"created,omitempty"`
	// RetainedBy lists the schedule rules (BackupRetentionRule* values) that
	// keep the copy.
	RetainedBy []string `json:"retained_by,omitempty"`
	KeepUntil  string   `json:"keep_until,omitempty"`
	IsManual   bool     `json:"is_manual"`
}

// CreateBackupRestorePointRequest represents a request to take a manual copy of
// a server. An empty Name lets the API name the copy.
type CreateBackupRestorePointRequest struct {
	Name string `json:"name,omitempty"`
}

// RestoreBackupRestorePointNearbyRequest represents a request to restore a copy
// into a new server. An empty LocationID restores into the server's location.
type RestoreBackupRestorePointNearbyRequest struct {
	// LocationID is the techTitle of the target location.
	LocationID string `json:"location_id,omitempty"`
}
