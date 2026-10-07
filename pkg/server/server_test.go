package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"voxlab/pkg/config"
	"voxlab/pkg/engine"
	"voxlab/pkg/tts"
)

func setupTestServer(t *testing.T) *Server {
	cfg := config.DefaultConfig()
	cfg.StaticDir = "../../web"
	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	return srv
}

func TestServerRESTEndpoints(t *testing.T) {
	srv := setupTestServer(t)

	// 1. GET /api/config
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()
	srv.handleConfig(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/config, got %d", w.Code)
	}
	var cfg config.AppConfig
	if err := json.NewDecoder(w.Body).Decode(&cfg); err != nil {
		t.Errorf("failed to decode config JSON: %v", err)
	}
	if cfg.Version != config.Version {
		t.Errorf("expected version %s, got %s", config.Version, cfg.Version)
	}

	// 2. GET /api/catalog
	reqCat := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	wCat := httptest.NewRecorder()
	srv.handleCatalog(wCat, reqCat)
	if wCat.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/catalog, got %d", wCat.Code)
	}

	// 3. GET /api/voices
	reqVoices := httptest.NewRequest(http.MethodGet, "/api/voices", nil)
	wVoices := httptest.NewRecorder()
	srv.handleVoices(wVoices, reqVoices)
	if wVoices.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/voices, got %d", wVoices.Code)
	}
	var voicesResp struct {
		Model    string             `json:"model"`
		IsNeural bool               `json:"is_neural"`
		Voices   []tts.VoiceProfile `json:"voices"`
	}
	if err := json.NewDecoder(wVoices.Body).Decode(&voicesResp); err != nil {
		t.Errorf("failed to decode voices JSON: %v", err)
	}
	if voicesResp.Model == "" {
		t.Errorf("expected non-empty model name in /api/voices response")
	}

	// 4. POST /api/tts
	ttsReq := engine.TTSRequest{
		Text:       "Testing Kokoro TTS via REST endpoint",
		Voice:      "af_heart",
		Speed:      1.0,
		SampleRate: 24000,
	}
	body, _ := json.Marshal(ttsReq)
	reqTTS := httptest.NewRequest(http.MethodPost, "/api/tts", bytes.NewReader(body))
	wTTS := httptest.NewRecorder()
	srv.handleTTS(wTTS, reqTTS)
	if wTTS.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/tts, got %d", wTTS.Code)
	}
	if !strings.HasPrefix(wTTS.Header().Get("Content-Type"), "audio/wav") {
		t.Errorf("expected audio/wav Content-Type, got %s", wTTS.Header().Get("Content-Type"))
	}
	if wTTS.Body.Len() < 100 {
		t.Errorf("expected non-trivial WAV payload, got %d bytes", wTTS.Body.Len())
	}
}

func TestServerWebSocketHandshakeAndAction(t *testing.T) {
	srv := setupTestServer(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", srv.handleWebSocket)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close()

	// Read initial voice.state event
	var initMsg map[string]interface{}
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err := conn.ReadJSON(&initMsg); err != nil {
		t.Fatalf("failed reading initial WS message: %v", err)
	}
	if initMsg["event"] != "voice.state" {
		t.Errorf("expected initial event voice.state, got %v", initMsg["event"])
	}

	// Send an inject_text action for a command
	cmdAction := map[string]interface{}{
		"action": "inject_text",
		"text":   "open settings",
		"mode":   "command",
	}
	if err := conn.WriteJSON(cmdAction); err != nil {
		t.Fatalf("failed to send action: %v", err)
	}

	// Wait for intent_matched event
	foundIntentMatched := false
	for i := 0; i < 5; i++ {
		var reply map[string]interface{}
		_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		if err := conn.ReadJSON(&reply); err != nil {
			break
		}
		if reply["event"] == "intent_matched" {
			foundIntentMatched = true
			data, _ := reply["data"].(map[string]interface{})
			if data["intent_id"] != "NAV_SETTINGS" {
				t.Errorf("expected intent_id NAV_SETTINGS, got %v", data["intent_id"])
			}
			break
		}
	}

	if !foundIntentMatched {
		t.Errorf("did not receive expected intent_matched event over WebSocket")
	}
}

func TestServerSwitchSource(t *testing.T) {
	srv := setupTestServer(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", srv.handleWebSocket)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close()

	// Read initial state message
	var initMsg map[string]interface{}
	_ = conn.ReadJSON(&initMsg)

	// Send set_source action for host_native_mic
	switchHostAction := map[string]interface{}{
		"action": "set_source",
		"source": "host_native_mic",
	}
	if err := conn.WriteJSON(switchHostAction); err != nil {
		t.Fatalf("failed sending set_source: %v", err)
	}

	foundSourceChanged := false
	for i := 0; i < 5; i++ {
		var msg map[string]interface{}
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		if msg["event"] == "audio.source_changed" {
			foundSourceChanged = true
			data, _ := msg["data"].(map[string]interface{})
			if data["source"] != "host_native_mic" {
				t.Errorf("expected source host_native_mic, got %v", data["source"])
			}
			break
		}
	}

	if !foundSourceChanged {
		t.Errorf("did not receive audio.source_changed event for host_native_mic")
	}
}

func TestServerPlaybackStatusEchoSuppression(t *testing.T) {
	srv := setupTestServer(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", srv.handleWebSocket)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close()

	// Initially not muted
	if srv.dsp.IsEchoMuted() {
		t.Errorf("expected echo to be unmuted initially")
	}

	// 1. Send playback_status: true (speaker playback starts)
	playAction := map[string]interface{}{
		"action":  "playback_status",
		"playing": true,
	}
	if err := conn.WriteJSON(playAction); err != nil {
		t.Fatalf("failed sending playback_status true: %v", err)
	}

	time.Sleep(20 * time.Millisecond)
	if !srv.dsp.IsEchoMuted() {
		t.Errorf("expected echo to be MUTED when playback_status: true is sent")
	}

	// 2. Send playback_status: false (speaker playback finishes)
	stopAction := map[string]interface{}{
		"action":  "playback_status",
		"playing": false,
	}
	if err := conn.WriteJSON(stopAction); err != nil {
		t.Fatalf("failed sending playback_status false: %v", err)
	}

	// Immediately after stop, tail delay should still keep it muted
	if !srv.dsp.IsEchoMuted() {
		t.Errorf("expected echo to remain muted immediately after playback ends (tail delay)")
	}

	// Wait for tail delay to clear (default 150ms)
	time.Sleep(200 * time.Millisecond)
	if srv.dsp.IsEchoMuted() {
		t.Errorf("expected echo to be UNMUTED after tail delay has passed")
	}
}

