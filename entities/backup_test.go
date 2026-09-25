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
