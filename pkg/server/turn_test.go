package server

import (
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"voxlab/pkg/engine"
	"voxlab/pkg/statemachine"
)

// fakeEngine replaces only the recognition calls; everything else stays nil.
type fakeEngine struct {
	engine.SpeechEngine
	calls     atomic.Int32
	onProcess func()
	result    *engine.ASRResult
}

func (f *fakeEngine) ProcessASRChunk([]float32, bool) (*engine.ASRResult, error) {
	f.calls.Add(1)
	if f.onProcess != nil {
		f.onProcess()
	}
	return f.result, nil
}

func (f *fakeEngine) ResetASR() {}

func (f *fakeEngine) Name() string { return "fake" }

func listeningServer(t *testing.T, eng engine.SpeechEngine) *Server {
	t.Helper()
	srv := setupTestServer(t)
	srv.engine = eng
	srv.stateMachine.StartCommand()
	return srv
}

func TestASRWorkItemTurns(t *testing.T) {
	item := func(srv *Server) asrWorkItem {
		return asrWorkItem{turn: srv.turn.Load(), state: statemachine.StateCommandListen}
	}

	t.Run("current turn is recognized", func(t *testing.T) {
		eng := &fakeEngine{}
		srv := listeningServer(t, eng)
		srv.handleASRWorkItem(item(srv))
		if eng.calls.Load() != 1 {
			t.Fatalf("recognizer calls = %d, want 1", eng.calls.Load())
		}
	})

	// A cancel followed by activate returns to the same state, which the state check cannot see.
	t.Run("audio from an abandoned turn is dropped", func(t *testing.T) {
		eng := &fakeEngine{}
		srv := listeningServer(t, eng)
		stale := item(srv)
		srv.newTurn()
		srv.handleASRWorkItem(stale)
		if eng.calls.Load() != 0 {
			t.Fatalf("stale audio reached the recognizer")
		}
	})

	t.Run("result of a turn cancelled during recognition is dropped", func(t *testing.T) {
		eng := &fakeEngine{result: &engine.ASRResult{IsFinal: true, Transcript: "stop recording"}}
		srv := listeningServer(t, eng)
		eng.onProcess = func() { srv.turn.Add(1) } // what cancel does first
		srv.handleASRWorkItem(item(srv))
		if got := srv.stateMachine.Current(); got != statemachine.StateCommandListen {
			t.Fatalf("cancelled result was acted on: state %s", got)
		}
	})
}

// During a bench run a live mic must not mix into the harness audio.
func TestBenchRunExcludesOtherMics(t *testing.T) {
	srv := setupTestServer(t)
	if err := srv.webMicSource.Start(srv.ctx, srv.audioInChan); err != nil {
		t.Fatal(err)
	}
	srv.evalMu.Lock()
	srv.evalRunning = true
	srv.evalMu.Unlock()

	ts := httptest.NewServer(srv.routes())
	defer ts.Close()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	dial := func() *websocket.Conn {
		t.Helper()
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Close() })
		return conn
	}
	pcm := make([]byte, 960)

	mic := dial()
	if err := mic.WriteMessage(websocket.BinaryMessage, pcm); err != nil {
		t.Fatal(err)
	}
	harness := dial()
	if err := harness.WriteJSON(map[string]string{"action": "eval_session"}); err != nil {
		t.Fatal(err)
	}
	if err := harness.WriteMessage(websocket.BinaryMessage, pcm); err != nil {
		t.Fatal(err)
	}

	select {
	case <-srv.audioInChan:
	case <-time.After(2 * time.Second):
		t.Fatal("harness audio did not reach the pipeline")
	}
	time.Sleep(100 * time.Millisecond)
	if n := len(srv.audioInChan); n != 0 {
		t.Fatalf("%d chunks from another client reached the pipeline", n)
	}
}
