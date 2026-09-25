package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/itglobalcom/vstack-cloud-panel-sdk/entities"
)

const backupTestServerID = "l1s42"

func testBackupSchedule() *entities.BackupSchedule {
	return &entities.BackupSchedule{
		Hour:   0,
		Minute: 30,
		Daily:  &entities.BackupRule{Keep: 7, BackupStorageID: 3},
		Weekly: &entities.BackupWeeklyRule{BackupRule: entities.BackupRule{Keep: 4, BackupStorageID: 3}, Weekday: 7},
	}
}

func TestBuildServerBackupPath(t *testing.T) {
	cases := map[string]string{
		buildServerBackupPath("l1s42"):                                         "servers/l1s42/backup",
		buildServerBackupPath("l1s42", backupStoragesPath):                     "servers/l1s42/backup/storages",
		buildServerBackupPath("l1s42", backupRestorePointsPath):                "servers/l1s42/backup/restore-points",
		buildServerBackupRestorePointPath("l1s42", 9):                          "servers/l1s42/backup/restore-points/9",
		buildServerBackupRestorePointPath("l1s42", 9, backupRestorePath):       "servers/l1s42/backup/restore-points/9/restore",
		buildServerBackupRestorePointPath("l1s42", 9, backupRestoreNearbyPath): "servers/l1s42/backup/restore-points/9/restore-nearby",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
	}
}

type recordedRequest struct {
	method string
	path   string
	query  string
	body   string
}

// recordingClient answers every request with response and records the last one.
func recordingClient(t *testing.T, response string) (*CloudClient, func() recordedRequest) {
	t.Helper()
	var (
		mu  sync.Mutex
		rec recordedRequest
	)
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		mu.Lock()
		rec = recordedRequest{method: r.Method, path: r.URL.Path, query: r.URL.RawQuery, body: string(raw)}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	return client, func() recordedRequest {
		mu.Lock()
		defer mu.Unlock()
		return rec
	}
}

func TestBackupOperationRequests(t *testing.T) {
	ctx := context.Background()
	const taskRef = `{"task_id":"l1t5"}`

	cases := []struct {
		name      string
		response  string
		call      func(c *CloudClient) (any, error)
		wantVerb  string
		wantPath  string
		wantQuery string
		wantBody  string
	}{
		{
			name:     "server storages",
			response: `{"storages":[],"limits":{}}`,
			call: func(c *CloudClient) (any, error) {
				return c.GetServerBackupStorages(ctx, backupTestServerID)
			},
			wantVerb: http.MethodGet,
			wantPath: "/api/v1/servers/l1s42/backup/storages",
		},
		{
			name:     "order storages with image",
			response: `{}`,
			call: func(c *CloudClient) (any, error) {
				return c.GetBackupStorageList(ctx, "msk-1", "l1i7")
			},
			wantVerb:  http.MethodGet,
			wantPath:  "/api/v1/backup/storages",
			wantQuery: "image_id=l1i7&location_id=msk-1",
		},
		{
			name:     "order storages without image",
			response: `{}`,
			call: func(c *CloudClient) (any, error) {
				return c.GetBackupStorageList(ctx, "msk-1", "")
			},
			wantVerb:  http.MethodGet,
			wantPath:  "/api/v1/backup/storages",
			wantQuery: "location_id=msk-1",
		},
		{
			name:     "service state",
			response: `{"enabled":false}`,
			call: func(c *CloudClient) (any, error) {
				return c.GetServerBackup(ctx, backupTestServerID)
			},
			wantVerb: http.MethodGet,
			wantPath: "/api/v1/servers/l1s42/backup",
		},
		{
			name:     "enable",
			response: taskRef,
			call: func(c *CloudClient) (any, error) {
				return c.EnableServerBackup(ctx, backupTestServerID, testBackupSchedule())
			},
			wantVerb: http.MethodPost,
			wantPath: "/api/v1/servers/l1s42/backup",
			wantBody: `{"hour":0,"minute":30,"daily":{"keep":7,"backup_storage_id":3},"weekly":{"keep":4,"backup_storage_id":3,"weekday":7}}`,
		},
		{
			name:     "update",
			response: taskRef,
			call: func(c *CloudClient) (any, error) {
				return c.UpdateServerBackup(ctx, backupTestServerID, testBackupSchedule())
			},
			wantVerb: http.MethodPut,
			wantPath: "/api/v1/servers/l1s42/backup",
			wantBody: `{"hour":0,"minute":30,"daily":{"keep":7,"backup_storage_id":3},"weekly":{"keep":4,"backup_storage_id":3,"weekday":7}}`,
		},
		{
			name:     "disable",
			response: taskRef,
			call: func(c *CloudClient) (any, error) {
				return c.DisableServerBackup(ctx, backupTestServerID)
			},
			wantVerb: http.MethodDelete,
			wantPath: "/api/v1/servers/l1s42/backup",
		},
		{
			name:     "restore points",
			response: `{}`,
			call: func(c *CloudClient) (any, error) {
				return c.GetServerBackupRestorePoints(ctx, backupTestServerID)
			},
			wantVerb: http.MethodGet,
			wantPath: "/api/v1/servers/l1s42/backup/restore-points",
		},
		{
			name:     "manual copy with name",
			response: taskRef,
			call: func(c *CloudClient) (any, error) {
				return c.CreateServerBackupRestorePoint(ctx, backupTestServerID, &entities.CreateBackupRestorePointRequest{Name: "before-upgrade"})
			},
			wantVerb: http.MethodPost,
			wantPath: "/api/v1/servers/l1s42/backup/restore-points",
			wantBody: `{"name":"before-upgrade"}`,
		},
		{
			name:     "manual copy without request",
			response: taskRef,
			call: func(c *CloudClient) (any, error) {
				return c.CreateServerBackupRestorePoint(ctx, backupTestServerID, nil)
			},
			wantVerb: http.MethodPost,
			wantPath: "/api/v1/servers/l1s42/backup/restore-points",
			wantBody: `{}`,
		},
		{
			name:     "restore over",
			response: taskRef,
			call: func(c *CloudClient) (any, error) {
				return c.RestoreServerBackupRestorePoint(ctx, backupTestServerID, 9)
			},
			wantVerb: http.MethodPost,
			wantPath: "/api/v1/servers/l1s42/backup/restore-points/9/restore",
		},
		{
			name:     "restore nearby into location",
			response: taskRef,
			call: func(c *CloudClient) (any, error) {
				return c.RestoreServerBackupRestorePointNearby(ctx, backupTestServerID, 9, &entities.RestoreBackupRestorePointNearbyRequest{LocationID: "spb-1"})
			},
			wantVerb: http.MethodPost,
			wantPath: "/api/v1/servers/l1s42/backup/restore-points/9/restore-nearby",
			wantBody: `{"location_id":"spb-1"}`,
		},
		{
			name:     "restore nearby without request",
			response: taskRef,
			call: func(c *CloudClient) (any, error) {
				return c.RestoreServerBackupRestorePointNearby(ctx, backupTestServerID, 9, nil)
			},
			wantVerb: http.MethodPost,
			wantPath: "/api/v1/servers/l1s42/backup/restore-points/9/restore-nearby",
			wantBody: `{}`,
		},
		{
			name:     "delete restore point",
			response: taskRef,
			call: func(c *CloudClient) (any, error) {
				return c.DeleteServerBackupRestorePoint(ctx, backupTestServerID, 9)
			},
			wantVerb:  http.MethodDelete,
			wantPath:  "/api/v1/servers/l1s42/backup/restore-points/9",
			wantQuery: "return_task=true",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, last := recordingClient(t, tc.response)

			got, err := tc.call(client)
			if err != nil {
				t.Fatalf("call: %v", err)
			}
			rec := last()
			if rec.method != tc.wantVerb {
				t.Errorf("method = %q, want %q", rec.method, tc.wantVerb)
			}
			if rec.path != tc.wantPath {
				t.Errorf("path = %q, want %q", rec.path, tc.wantPath)
			}
			if rec.query != tc.wantQuery {
				t.Errorf("query = %q, want %q", rec.query, tc.wantQuery)
			}
			if rec.body != tc.wantBody {
				t.Errorf("body = %q, want %q", rec.body, tc.wantBody)
			}
			if task, ok := got.(*TaskID); ok && (task == nil || task.ID != "l1t5") {
				t.Errorf("task = %v, want l1t5", task)
			}
		})
	}
}

func TestBackupRejectsBadArguments(t *testing.T) {
	ctx := context.Background()
	var requests int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))

	cases := map[string]func() error{
		"order storages without location": func() error { _, err := client.GetBackupStorageList(ctx, "", "l1i7"); return err },
		"server storages without server":  func() error { _, err := client.GetServerBackupStorages(ctx, ""); return err },
		"state without server":            func() error { _, err := client.GetServerBackup(ctx, ""); return err },
		"enable without server":           func() error { _, err := client.EnableServerBackup(ctx, "", testBackupSchedule()); return err },
		"enable without schedule":         func() error { _, err := client.EnableServerBackup(ctx, backupTestServerID, nil); return err },
		"enable with invalid schedule": func() error {
			_, err := client.EnableServerBackup(ctx, backupTestServerID, &entities.BackupSchedule{Hour: 24, Daily: &entities.BackupRule{Keep: 1, BackupStorageID: 3}})
			return err
		},
		"update without schedule": func() error { _, err := client.UpdateServerBackup(ctx, backupTestServerID, nil); return err },
		"disable without server":  func() error { _, err := client.DisableServerBackup(ctx, ""); return err },
		"points without server":   func() error { _, err := client.GetServerBackupRestorePoints(ctx, ""); return err },
		"copy without server":     func() error { _, err := client.CreateServerBackupRestorePoint(ctx, "", nil); return err },
		"restore without point":   func() error { _, err := client.RestoreServerBackupRestorePoint(ctx, backupTestServerID, 0); return err },
		"restore nearby without point": func() error {
			_, err := client.RestoreServerBackupRestorePointNearby(ctx, backupTestServerID, 0, nil)
			return err
		},
		"delete point without point":  func() error { _, err := client.DeleteServerBackupRestorePoint(ctx, backupTestServerID, 0); return err },
		"delete point without server": func() error { _, err := client.DeleteServerBackupRestorePoint(ctx, "", 9); return err },
	}
	for name, call := range cases {
		if err := call(); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if got := atomic.LoadInt32(&requests); got != 0 {
		t.Errorf("%d request(s) sent for invalid arguments, want 0", got)
	}
}

// Null fields are omitted on the wire: a planned copy has no storage, no created
// time and no retention.
func TestParseBackupRestorePointsResponse(t *testing.T) {
	raw := `{"restore_points":[
		{"id":9,"server_id":"l1s42","name":"daily","size_mb":2048,"backup_storage_id":3,"state":"active",
		 "created":"2026-09-20T02:00:00","retained_by":["daily","weekly"],"keep_until":"2026-09-27T02:00:00","is_manual":false},
		{"id":10,"server_id":"l1s42","name":"manual","size_mb":0,"state":"in_progress","retained_by":[],"is_manual":true}]}`
	var resp ListBackupRestorePointsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.RestorePoints) != 2 {
		t.Fatalf("restore points = %d, want 2", len(resp.RestorePoints))
	}
	active := resp.RestorePoints[0]
	if active.ID != 9 || active.ServerID != "l1s42" || active.SizeMB != 2048 || active.State != entities.BackupRestorePointStateActive {
		t.Errorf("active point = %+v", active)
	}
	if active.BackupStorageID == nil || *active.BackupStorageID != 3 {
		t.Errorf("backup storage = %v, want 3", active.BackupStorageID)
	}
	if len(active.RetainedBy) != 2 || active.RetainedBy[1] != entities.BackupRetentionRuleWeekly {
		t.Errorf("retained by = %v, want [daily weekly]", active.RetainedBy)
	}
	if active.Created == "" || active.KeepUntil == "" || active.IsManual {
		t.Errorf("active point = %+v, want created, keep_until and scheduled", active)
	}
	planned := resp.RestorePoints[1]
	if planned.State != entities.BackupRestorePointStateInProgress || !planned.IsManual {
		t.Errorf("planned point = %+v, want manual in_progress", planned)
	}
	if planned.BackupStorageID != nil || planned.Created != "" || planned.KeepUntil != "" {
		t.Errorf("planned point = %+v, want omitted fields left empty", planned)
	}

	var empty ListBackupRestorePointsResponse
	if err := json.Unmarshal([]byte(`{}`), &empty); err != nil {
		t.Fatalf("unmarshal empty: %v", err)
	}
	if empty.RestorePoints != nil {
		t.Errorf("restore points = %v, want nil", empty.RestorePoints)
	}
}

// manualCopyStub serves a server whose restore point list gains newPoint once
// the manual copy is requested; the copy task completes on its first read.
func manualCopyStub(t *testing.T, newPoint string, taskReads *int32) *CloudClient {
	t.Helper()
	var copied int32
	existing := `{"id":1,"server_id":"l1s42","name":"daily","size_mb":10,"state":"active","is_manual":false}`

	return newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/servers/l1s42/backup/restore-points":
			atomic.StoreInt32(&copied, 1)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"task_id":"l1t77"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/servers/l1s42/backup/restore-points":
			points := existing
			if atomic.LoadInt32(&copied) == 1 {
				points += "," + newPoint
			}
			_, _ = w.Write([]byte(`{"restore_points":[` + points + `]}`))
		case r.URL.Path == "/api/v1/tasks/l1t77":
			atomic.AddInt32(taskReads, 1)
			_, _ = w.Write([]byte(`{"task":{"id":"l1t77","is_completed":"Completed"}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestCreateServerBackupRestorePointAndWait(t *testing.T) {
	ctx := context.Background()

	t.Run("active point is returned", func(t *testing.T) {
		var taskReads int32
		client := manualCopyStub(t,
			`{"id":2,"server_id":"l1s42","name":"before-upgrade","size_mb":20,"state":"active","is_manual":true}`, &taskReads)

		point, err := client.CreateServerBackupRestorePointAndWait(ctx, backupTestServerID,
			&entities.CreateBackupRestorePointRequest{Name: "before-upgrade"})
		if err != nil {
			t.Fatalf("CreateServerBackupRestorePointAndWait: %v", err)
		}
		if point.ID != 2 || point.State != entities.BackupRestorePointStateActive {
			t.Errorf("point = %+v, want active point 2", point)
		}
		if atomic.LoadInt32(&taskReads) == 0 {
			t.Error("the copy task was not awaited")
		}
	})

	t.Run("point without requested name is found", func(t *testing.T) {
		var taskReads int32
		client := manualCopyStub(t,
			`{"id":2,"server_id":"l1s42","name":"generated","size_mb":20,"state":"active","is_manual":true}`, &taskReads)

		point, err := client.CreateServerBackupRestorePointAndWait(ctx, backupTestServerID, nil)
		if err != nil {
			t.Fatalf("CreateServerBackupRestorePointAndWait: %v", err)
		}
		if point.ID != 2 {
			t.Errorf("point = %+v, want point 2", point)
		}
	})

	t.Run("failed point is an error", func(t *testing.T) {
		var taskReads int32
		client := manualCopyStub(t,
			`{"id":2,"server_id":"l1s42","name":"before-upgrade","size_mb":0,"state":"failed","is_manual":true}`, &taskReads)

		point, err := client.CreateServerBackupRestorePointAndWait(ctx, backupTestServerID,
			&entities.CreateBackupRestorePointRequest{Name: "before-upgrade"})
		if err == nil {
			t.Fatalf("point = %+v, want an error for a failed copy", point)
		}
		if !IsBackupRestorePointFailed(err) {
			t.Errorf("IsBackupRestorePointFailed = false: %v", err)
		}
		if !strings.Contains(err.Error(), "restore point 2") {
			t.Errorf("error %q does not name the failed point", err)
		}
	})
}

func TestDeleteServerBackupRestorePointAndWait(t *testing.T) {
	t.Run("task is awaited", func(t *testing.T) {
		var (
			query     atomic.Value
			taskReads int32
		)
		client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch {
			case r.Method == http.MethodDelete:
				query.Store(r.URL.RawQuery)
				_, _ = w.Write([]byte(`{"task_id":"l1t78"}`))
			case r.URL.Path == "/api/v1/tasks/l1t78":
				state := entities.TaskStateInProgress
				if atomic.AddInt32(&taskReads, 1) >= 2 {
					state = entities.TaskStateCompleted
				}
				_, _ = w.Write([]byte(`{"task":{"id":"l1t78","is_completed":"` + state + `"}}`))
			default:
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			}
		}))

		if err := client.DeleteServerBackupRestorePointAndWait(context.Background(), backupTestServerID, 9); err != nil {
			t.Fatalf("DeleteServerBackupRestorePointAndWait: %v", err)
		}
		if got, _ := query.Load().(string); got != "return_task=true" {
			t.Errorf("query = %q, want return_task=true", got)
		}
		if got := atomic.LoadInt32(&taskReads); got < 2 {
			t.Errorf("task read %d time(s), want polling until Completed", got)
		}
	})

	t.Run("failed task is an error", func(t *testing.T) {
		client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodDelete {
				_, _ = w.Write([]byte(`{"task_id":"l1t78"}`))
				return
			}
			_, _ = w.Write([]byte(`{"task":{"id":"l1t78","is_completed":"Failed"}}`))
		}))

		err := client.DeleteServerBackupRestorePointAndWait(context.Background(), backupTestServerID, 9)
		if !IsTaskFailed(err) {
			t.Errorf("IsTaskFailed = false: %v", err)
		}
	})
}

// The schedule update that keeps the storages answers the synthetic task; the
// wait must not read it.
func TestUpdateServerBackupAndWaitAlreadyCompleted(t *testing.T) {
	var taskReads int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut:
			_, _ = w.Write([]byte(`{"task_id":"` + AlreadyCompletedTaskID + `"}`))
		case strings.HasPrefix(r.URL.Path, "/api/v1/tasks/"):
			atomic.AddInt32(&taskReads, 1)
			_, _ = w.Write([]byte(`{"task":{"id":"x","is_completed":"Completed"}}`))
		default:
			_, _ = w.Write([]byte(`{"enabled":true,"schedule":{"hour":0,"minute":30,"daily":{"keep":7,"backup_storage_id":3}}}`))
		}
	}))

	backup, err := client.UpdateServerBackupAndWait(context.Background(), backupTestServerID, testBackupSchedule())
	if err != nil {
		t.Fatalf("UpdateServerBackupAndWait: %v", err)
	}
	if !backup.Enabled || backup.Schedule == nil || backup.Schedule.Minute != 30 {
		t.Errorf("backup = %+v, want the re-read schedule", backup)
	}
	if got := atomic.LoadInt32(&taskReads); got != 0 {
		t.Errorf("synthetic task read %d time(s), want 0", got)
	}
}

func TestBackupServerNotFound(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	cases := map[string]func() error{
		"state":    func() error { _, err := client.GetServerBackup(ctx, backupTestServerID); return err },
		"storages": func() error { _, err := client.GetServerBackupStorages(ctx, backupTestServerID); return err },
		"enable": func() error {
			_, err := client.EnableServerBackupAndWait(ctx, backupTestServerID, testBackupSchedule())
			return err
		},
		"disable":        func() error { return client.DisableServerBackupAndWait(ctx, backupTestServerID) },
		"restore points": func() error { _, err := client.GetServerBackupRestorePoints(ctx, backupTestServerID); return err },
		"manual copy": func() error {
			_, err := client.CreateServerBackupRestorePointAndWait(ctx, backupTestServerID, nil)
			return err
		},
		"restore": func() error {
			_, err := client.RestoreServerBackupRestorePointAndWait(ctx, backupTestServerID, 9)
			return err
		},
		"restore nearby": func() error {
			return client.RestoreServerBackupRestorePointNearbyAndWait(ctx, backupTestServerID, 9, nil)
		},
		"delete point": func() error { return client.DeleteServerBackupRestorePointAndWait(ctx, backupTestServerID, 9) },
	}
	for name, call := range cases {
		err := call()
		if !IsNotFound(err) {
			t.Errorf("%s: IsNotFound = false: %v", name, err)
		}
	}
}

func TestIsBackupRestorePointFailed(t *testing.T) {
	wrapped := fmt.Errorf("backup restore point 2 on server l1s42: %w", ErrBackupRestorePointFailed)
	if !IsBackupRestorePointFailed(wrapped) {
		t.Errorf("IsBackupRestorePointFailed(%q) = false, want true", wrapped)
	}
	for _, err := range []error{nil, ErrTaskFailed, ErrNotFound, errors.New("backup restore point failed")} {
		if IsBackupRestorePointFailed(err) {
			t.Errorf("IsBackupRestorePointFailed(%v) = true, want false", err)
		}
	}
}
