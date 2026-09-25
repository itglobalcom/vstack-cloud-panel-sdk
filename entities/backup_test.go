package entities

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBackupScheduleValidate(t *testing.T) {
	daily := &BackupRule{Keep: 7, BackupStorageID: 3}
	cases := []struct {
		name    string
		req     BackupSchedule
		wantErr bool
	}{
		{"daily at midnight", BackupSchedule{Hour: 0, Minute: 0, Daily: daily}, false},
		{"all rules", BackupSchedule{Hour: 23, Minute: 59, Daily: daily,
			Weekly:  &BackupWeeklyRule{BackupRule: BackupRule{Keep: 4, BackupStorageID: 3}, Weekday: 1},
			Monthly: &BackupMonthlyRule{BackupRule: BackupRule{Keep: 12, BackupStorageID: 4}, DayOfMonth: "1"}}, false},
		{"hour out of range", BackupSchedule{Hour: 24, Daily: daily}, true},
		{"negative minute", BackupSchedule{Minute: -1, Daily: daily}, true},
		{"minute out of range", BackupSchedule{Minute: 60, Daily: daily}, true},
		{"no rules", BackupSchedule{Hour: 2}, true},
		{"daily without storage", BackupSchedule{Daily: &BackupRule{Keep: 7}}, true},
		{"weekly without storage", BackupSchedule{Weekly: &BackupWeeklyRule{BackupRule: BackupRule{Keep: 4}, Weekday: 1}}, true},
		{"weekday zero", BackupSchedule{Weekly: &BackupWeeklyRule{BackupRule: BackupRule{Keep: 4, BackupStorageID: 3}}}, true},
		{"weekday eight", BackupSchedule{Weekly: &BackupWeeklyRule{BackupRule: BackupRule{Keep: 4, BackupStorageID: 3}, Weekday: 8}}, true},
		{"monthly without storage", BackupSchedule{Monthly: &BackupMonthlyRule{BackupRule: BackupRule{Keep: 12}, DayOfMonth: "1"}}, true},
		{"monthly without day", BackupSchedule{Monthly: &BackupMonthlyRule{BackupRule: BackupRule{Keep: 12, BackupStorageID: 4}}}, true},
	}
	for _, tc := range cases {
		err := tc.req.Validate()
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: Validate() = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
	}
}

func TestCreateServerRequestBackup(t *testing.T) {
	req := CreateServerRequest{
		LocationID: "msk-1", ImageID: "l1i7", CPU: 1, RamMB: 1024, Name: "web",
		Volumes: []VolumeSpec{{Name: "boot", SizeMB: 10240}},
	}

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), `"backup"`) {
		t.Errorf("body = %s, want no backup block for a server without backup", raw)
	}

	req.Backup = &BackupSchedule{Hour: 2, Daily: &BackupRule{Keep: 7, BackupStorageID: 3}}
	if err := req.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
	raw, err = json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if want := `"backup":{"hour":2,"minute":0,"daily":{"keep":7,"backup_storage_id":3}}`; !strings.Contains(string(raw), want) {
		t.Errorf("body = %s, want %s", raw, want)
	}

	req.Backup = &BackupSchedule{Hour: 2}
	if err := req.Validate(); err == nil {
		t.Error("Validate() = nil, want an error for a backup block without rules")
	}
}

func TestParseServerBackupResponse(t *testing.T) {
	enabled := `{"enabled":true,"schedule":{"hour":2,"minute":0,
		"daily":{"keep":7,"backup_storage_id":3},
		"monthly":{"keep":12,"backup_storage_id":4,"day_of_month":"last"}}}`
	var backup ServerBackup
	if err := json.Unmarshal([]byte(enabled), &backup); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !backup.Enabled || backup.Schedule == nil {
		t.Fatalf("backup = %+v, want enabled with a schedule", backup)
	}
	if backup.Schedule.Daily == nil || backup.Schedule.Daily.Keep != 7 || backup.Schedule.Daily.BackupStorageID != 3 {
		t.Errorf("daily = %+v, want keep 7 on storage 3", backup.Schedule.Daily)
	}
	if backup.Schedule.Weekly != nil {
		t.Errorf("weekly = %+v, want nil for an omitted rule", backup.Schedule.Weekly)
	}
	if m := backup.Schedule.Monthly; m == nil || m.DayOfMonth != BackupDayOfMonthLast || m.BackupStorageID != 4 || m.Keep != 12 {
		t.Errorf("monthly = %+v, want keep 12 on storage 4 on day last", m)
	}

	var legacy ServerBackup
	if err := json.Unmarshal([]byte(`{"enabled":true}`), &legacy); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !legacy.Enabled || legacy.Schedule != nil {
		t.Errorf("backup = %+v, want enabled without a schedule", legacy)
	}

	var disabled ServerBackup
	if err := json.Unmarshal([]byte(`{"enabled":false}`), &disabled); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if disabled.Enabled || disabled.Schedule != nil {
		t.Errorf("backup = %+v, want disabled without a schedule", disabled)
	}
}

func TestParseBackupStorageCatalog(t *testing.T) {
	raw := `{"storages":[{"id":3,"name":"S3 MSK","description":"object storage","sort_order":1,
		"price_per_gb":1.25,"location_id":"msk-1","restore_location_ids":["msk-1","spb-1"]}],
		"limits":{"schedule_window_from_hour":1,"schedule_window_to_hour":6,
		"daily":{"max_keep":30,"default_keep":7},"weekly":{"max_keep":12,"default_keep":4},
		"monthly":{"max_keep":12,"default_keep":0}}}`
	var catalog BackupStorageCatalog
	if err := json.Unmarshal([]byte(raw), &catalog); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(catalog.Storages) != 1 {
		t.Fatalf("storages = %d, want 1", len(catalog.Storages))
	}
	s := catalog.Storages[0]
	if s.ID != 3 || s.Name != "S3 MSK" || s.SortOrder != 1 || s.PricePerGB != 1.25 || s.LocationID != "msk-1" {
		t.Errorf("storage = %+v", s)
	}
	if len(s.RestoreLocationIDs) != 2 || s.RestoreLocationIDs[1] != "spb-1" {
		t.Errorf("restore locations = %v, want [msk-1 spb-1]", s.RestoreLocationIDs)
	}
	l := catalog.Limits
	if l == nil || l.ScheduleWindowFromHour != 1 || l.ScheduleWindowToHour != 6 {
		t.Fatalf("limits = %+v", l)
	}
	if l.Daily == nil || l.Daily.MaxKeep != 30 || l.Daily.DefaultKeep != 7 {
		t.Errorf("daily limits = %+v, want max 30 default 7", l.Daily)
	}
	if l.Monthly == nil || l.Monthly.MaxKeep != 12 {
		t.Errorf("monthly limits = %+v, want max 12", l.Monthly)
	}

	var empty BackupStorageCatalog
	if err := json.Unmarshal([]byte(`{}`), &empty); err != nil {
		t.Fatalf("unmarshal empty: %v", err)
	}
	if empty.Storages != nil || empty.Limits != nil {
		t.Errorf("empty catalog = %+v, want no storages and no limits", empty)
	}
}
