package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestDefaultBindIsLoopback(t *testing.T) {
	if host := setupTestServer(t).cfg.Host; !isLoopbackName(host) {
		t.Fatalf("default bind must be loopback, got %q", host)
	}
}

func TestWebSocketRejectsForeignOrigins(t *testing.T) {
	ts := httptest.NewServer(setupTestServer(t).routes())
	defer ts.Close()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	tests := []struct {
		name    string
		headers http.Header
		want    int
	}{
		{"same origin", http.Header{"Origin": {ts.URL}}, http.StatusSwitchingProtocols},
		{"non-browser client without Origin", nil, http.StatusSwitchingProtocols},
		{"foreign origin", http.Header{"Origin": {"http://evil.example"}}, http.StatusForbidden},
		{"DNS rebinding", http.Header{"Origin": {"http://evil.example"}, "Host": {"evil.example"}}, http.StatusForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, resp, err := websocket.DefaultDialer.Dial(wsURL, tc.headers)
			if conn != nil {
				defer conn.Close()
			}
			if resp == nil {
				t.Fatalf("no handshake response: %v", err)
			}
			if resp.StatusCode != tc.want {
				t.Fatalf("got status %d, want %d", resp.StatusCode, tc.want)
			}
		})
	}
}

func TestRESTRejectsCrossSiteRequests(t *testing.T) {
	ts := httptest.NewServer(setupTestServer(t).routes())
	defer ts.Close()

	do := func(method, path string, headers map[string]string) int {
		t.Helper()
		req, err := http.NewRequest(method, ts.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range headers {
			if k == "Host" {
				req.Host = v
			} else {
				req.Header.Set(k, v)
			}
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}

	if got := do(http.MethodGet, "/api/eval/start", nil); got != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/eval/start: got %d, want 405", got)
	}
	if got := do(http.MethodGet, "/api/eval/cancel", nil); got != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/eval/cancel: got %d, want 405", got)
	}
	if got := do(http.MethodPost, "/api/eval/cancel", map[string]string{"Origin": "http://evil.example", "Sec-Fetch-Site": "cross-site"}); got != http.StatusForbidden {
		t.Errorf("cross-site POST: got %d, want 403", got)
	}
	if got := do(http.MethodGet, "/api/config", map[string]string{"Host": "evil.example"}); got != http.StatusForbidden {
		t.Errorf("rebinding Host: got %d, want 403", got)
	}
	if got := do(http.MethodGet, "/api/config", map[string]string{"Host": "localhost:8080"}); got != http.StatusOK {
		t.Errorf("localhost Host: got %d, want 200", got)
	}
}
