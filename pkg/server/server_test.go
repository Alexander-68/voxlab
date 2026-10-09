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

func TestServerMonitorAudioStreaming(t *testing.T) {
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

	// Drain initial voice.state
	var initMsg map[string]interface{}
	_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := conn.ReadJSON(&initMsg); err != nil {
		t.Fatalf("failed reading initial WS message: %v", err)
	}

	// 1. Initially monitor is disabled
	if srv.IsMonitorEnabled() {
		t.Errorf("expected monitor to be disabled initially")
	}

	// 2. Enable monitor via WebSocket action
	if err := conn.WriteJSON(map[string]interface{}{
		"action":  "set_monitor",
		"enabled": true,
	}); err != nil {
		t.Fatalf("failed to send set_monitor action: %v", err)
	}

	time.Sleep(20 * time.Millisecond)
	if !srv.IsMonitorEnabled() {
		t.Errorf("expected monitor to be enabled after set_monitor: true")
	}

	// 3. Set active source to host_native_mic
	srv.sourceMu.Lock()
	srv.activeSource = "host_native_mic"
	srv.sourceMu.Unlock()

	// 4. Feed a synthetic audio chunk (16kHz sine wave, 480 samples = 30ms)
	chunk := make([]float32, 480)
	for i := range chunk {
		chunk[i] = 0.5 // above noise floor
	}
	srv.processIncomingAudio(chunk)

	// 5. Read binary audio chunk from WebSocket
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	foundBinary := false
	for i := 0; i < 5; i++ {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if msgType == websocket.BinaryMessage {
			foundBinary = true
			if len(data) != 480*2 {
				t.Errorf("expected 960 bytes (480 int16 samples), got %d bytes", len(data))
			}
			break
		}
	}

	if !foundBinary {
		t.Errorf("expected binary PCM chunk over WebSocket for headphone monitoring")
	}

	// 6. Disable monitor via WebSocket action
	if err := conn.WriteJSON(map[string]interface{}{
		"action":  "set_monitor",
		"enabled": false,
	}); err != nil {
		t.Fatalf("failed to send set_monitor action: %v", err)
	}

	time.Sleep(20 * time.Millisecond)
	if srv.IsMonitorEnabled() {
		t.Errorf("expected monitor to be disabled after set_monitor: false")
	}
}

func TestServerModelManagement(t *testing.T) {
	srv := setupTestServer(t)

	// 1. GET /api/voices with model query param
	reqV019 := httptest.NewRequest(http.MethodGet, "/api/voices?model=kokoro-en-v0_19", nil)
	wV019 := httptest.NewRecorder()
	srv.handleVoices(wV019, reqV019)
	if wV019.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/voices?model=kokoro-en-v0_19, got %d", wV019.Code)
	}
	var respV019 struct {
		InstalledModels []string           `json:"installed_models"`
		Voices          []tts.VoiceProfile `json:"voices"`
	}
	if err := json.NewDecoder(wV019.Body).Decode(&respV019); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(respV019.Voices) != 11 {
		t.Errorf("expected 11 voices for v0_19 query, got %d", len(respV019.Voices))
	}

	// 1b. GET /api/voices for kokoro-multi-lang-v1_0
	reqV10 := httptest.NewRequest(http.MethodGet, "/api/voices?model=kokoro-multi-lang-v1_0", nil)
	wV10 := httptest.NewRecorder()
	srv.handleVoices(wV10, reqV10)
	if wV10.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/voices?model=kokoro-multi-lang-v1_0, got %d", wV10.Code)
	}
	var respV10 struct {
		Voices []tts.VoiceProfile `json:"voices"`
	}
	if err := json.NewDecoder(wV10.Body).Decode(&respV10); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(respV10.Voices) != 54 {
		t.Errorf("expected 54 voices for v1_0 query, got %d", len(respV10.Voices))
	}

	// 2. POST /api/models to switch model
	body, _ := json.Marshal(map[string]string{"model": "kokoro-en-v0_19"})
	reqPostModel := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewReader(body))
	wPostModel := httptest.NewRecorder()
	srv.handleModels(wPostModel, reqPostModel)
	if wPostModel.Code != http.StatusOK {
		t.Errorf("expected 200 OK for POST /api/models, got %d", wPostModel.Code)
	}

	activeModel, _ := srv.ttsMgr.ActiveModel()
	if !strings.Contains(activeModel, "kokoro-en-v0_19") {
		t.Errorf("expected active model kokoro-en-v0_19, got %s", activeModel)
	}

	// 3. Test WebSocket set_tts_model action
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

	if err := conn.WriteJSON(map[string]interface{}{
		"action": "set_tts_model",
		"model":  "kokoro-multi-lang-v1_1",
	}); err != nil {
		t.Fatalf("failed to send set_tts_model action: %v", err)
	}

	foundModelChanged := false
	for i := 0; i < 5; i++ {
		var msg map[string]interface{}
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		if msg["event"] == "tts.model_changed" {
			foundModelChanged = true
			data, _ := msg["data"].(map[string]interface{})
			if data["model_id"] != "kokoro-multi-lang-v1_1" {
				t.Errorf("expected model_id kokoro-multi-lang-v1_1, got %v", data["model_id"])
			}
			break
		}
	}
	if !foundModelChanged {
		t.Errorf("expected tts.model_changed event over WebSocket")
	}

	// 4. Test GET /api/voices and POST /api/models with FP16
	reqFP16 := httptest.NewRequest(http.MethodGet, "/api/voices?model=kokoro-multi-lang-v1_0%20(FP16)", nil)
	wFP16 := httptest.NewRecorder()
	srv.handleVoices(wFP16, reqFP16)
	if wFP16.Code != http.StatusOK {
		t.Errorf("expected 200 OK for FP16 voices query, got %d", wFP16.Code)
	}
	var respFP16 struct {
		Voices []tts.VoiceProfile `json:"voices"`
	}
	if err := json.NewDecoder(wFP16.Body).Decode(&respFP16); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(respFP16.Voices) != 54 {
		t.Errorf("expected 54 voices for FP16 query, got %d", len(respFP16.Voices))
	}

	bodyFP16, _ := json.Marshal(map[string]string{"model": "kokoro-multi-lang-v1_0 (FP16)"})
	reqPostFP16 := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewReader(bodyFP16))
	wPostFP16 := httptest.NewRecorder()
	srv.handleModels(wPostFP16, reqPostFP16)
	if wPostFP16.Code != http.StatusOK {
		t.Errorf("expected 200 OK for POST /api/models with FP16, got %d", wPostFP16.Code)
	}
	activeFP16, _ := srv.ttsMgr.ActiveModel()
	if !strings.Contains(activeFP16, "FP16") {
		t.Errorf("expected active model to contain FP16, got %s", activeFP16)
	}

	// 5. Test GET /api/voices and POST /api/models with INT8
	reqINT8 := httptest.NewRequest(http.MethodGet, "/api/voices?model=kokoro-multi-lang-v1_1%20(INT8)", nil)
	wINT8 := httptest.NewRecorder()
	srv.handleVoices(wINT8, reqINT8)
	if wINT8.Code != http.StatusOK {
		t.Errorf("expected 200 OK for INT8 voices query, got %d", wINT8.Code)
	}
	var respINT8 struct {
		Voices []tts.VoiceProfile `json:"voices"`
	}
	if err := json.NewDecoder(wINT8.Body).Decode(&respINT8); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(respINT8.Voices) != 103 {
		t.Errorf("expected 103 voices for INT8 query, got %d", len(respINT8.Voices))
	}

	bodyINT8, _ := json.Marshal(map[string]string{"model": "kokoro-multi-lang-v1_1 (INT8)"})
	reqPostINT8 := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewReader(bodyINT8))
	wPostINT8 := httptest.NewRecorder()
	srv.handleModels(wPostINT8, reqPostINT8)
	if wPostINT8.Code != http.StatusOK {
		t.Errorf("expected 200 OK for POST /api/models with INT8, got %d", wPostINT8.Code)
	}
	activeINT8, _ := srv.ttsMgr.ActiveModel()
	if !strings.Contains(activeINT8, "INT8") {
		t.Errorf("expected active model to contain INT8, got %s", activeINT8)
	}
}


