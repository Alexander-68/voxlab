package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"voxlab/pkg/statemachine"
)

func dialTestClient(t *testing.T, srv *Server) *websocket.Conn {
	t.Helper()
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func waitUntil(t *testing.T, what string, done func() bool) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); !done(); time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
	}
}

func TestDisconnectDuringPlaybackClearsEchoMute(t *testing.T) {
	srv := setupTestServer(t)
	conn := dialTestClient(t, srv)
	if err := conn.WriteJSON(map[string]interface{}{"action": "playback_status", "playing": true}); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "playback to start", srv.dsp.IsEchoMuted)
	conn.Close()
	waitUntil(t, "echo mute to clear", func() bool { return !srv.dsp.IsEchoMuted() })
	if got := srv.stateMachine.Current(); got != statemachine.StateIdle {
		t.Fatalf("state after disconnect = %s, want IDLE", got)
	}
}

// README: words spoken during dictation can never trigger system commands.
func TestDictationIsIsolatedFromOtherActions(t *testing.T) {
	srv := setupTestServer(t)
	conn := dialTestClient(t, srv)
	srv.stateMachine.StartDictation("target")

	for _, msg := range []map[string]interface{}{
		{"action": "confirm", "approved": true},
		{"action": "playback_status", "playing": true},
		{"action": "playback_status", "playing": false},
		{"action": "inject_text", "text": "stop recording", "mode": "command"},
	} {
		if err := conn.WriteJSON(msg); err != nil {
			t.Fatal(err)
		}
	}
	// A later message on the same connection proves the earlier ones were handled.
	if err := conn.WriteJSON(map[string]interface{}{"action": "set_draft", "text": "done"}); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "messages to be handled", func() bool {
		srv.dictMu.Lock()
		defer srv.dictMu.Unlock()
		return srv.dictationDraft == "done"
	})

	if got := srv.stateMachine.Current(); got != statemachine.StateAnnotationListen {
		t.Fatalf("dictation was interrupted: state %s", got)
	}
}

func TestEvalReportRejectsUnknownTiers(t *testing.T) {
	srv := setupTestServer(t)
	for _, tier := range []string{"../../../secret", `..\..\secret`, "tier-3"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/eval/report", nil)
		q := req.URL.Query()
		q.Set("tier", tier)
		req.URL.RawQuery = q.Encode()
		srv.handleEvalReport(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("tier %q: got %d, want 400", tier, w.Code)
		}
	}
}

func TestCancelledRunLeavesNoReportAndAllowsANewRun(t *testing.T) {
	srv := setupTestServer(t)
	srv.StartBackgroundLoops()
	t.Chdir(filepath.Join("..", ".."))

	for range 2 { // the second pass proves the first left nothing behind
		if err := srv.StartEvaluation("tier-1-smoke"); err != nil {
			t.Fatalf("start: %v", err)
		}
		srv.CancelEvaluation()
		waitUntil(t, "run to stop", func() bool { return !srv.EvalStatus().Running })
		if srv.LastReport() != nil {
			t.Fatal("cancelled run published a report")
		}
	}
}

func TestSaveDraftStoresTextAndClearsDraft(t *testing.T) {
	srv := setupTestServer(t)
	t.Chdir(t.TempDir())
	if err := os.Mkdir("data", 0755); err != nil {
		t.Fatal(err)
	}
	srv.dictationDraft = "unedited"
	srv.dictationTarget = "scope_1"

	srv.saveDraft("Edited text.")
	srv.saveDraft("Second note.")

	if srv.dictationDraft != "" {
		t.Errorf("draft not cleared: %q", srv.dictationDraft)
	}
	data, err := os.ReadFile(annotationLogPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d log lines, want 2", len(lines))
	}
	var entry struct {
		TargetID string `json:"target_id"`
		Text     string `json:"text"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatal(err)
	}
	if entry.Text != "Edited text." || entry.TargetID != "scope_1" {
		t.Errorf("unexpected entry: %+v", entry)
	}
}

// A failed save must keep the draft, or the dictated text is lost.
func TestFailedSaveKeepsDraft(t *testing.T) {
	srv := setupTestServer(t)
	t.Chdir(t.TempDir()) // no data directory
	srv.dictationDraft = "keep me"
	srv.saveDraft("keep me")
	if srv.dictationDraft != "keep me" {
		t.Fatalf("draft lost after failed save: %q", srv.dictationDraft)
	}
}
