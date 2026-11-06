package evaluation

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"voxlab/pkg/audio"
	"voxlab/pkg/server"
)

// ItemResult contains the evaluation verdict and metrics for one test item.
type ItemResult struct {
	ID          string        `json:"id"`
	Description string        `json:"description"`
	Mode        TestCaseMode  `json:"mode"`
	Passed      bool          `json:"passed"`
	Transcript  string        `json:"transcript"`
	IntentID    string        `json:"intent_id,omitempty"`
	Confidence  float64       `json:"confidence,omitempty"`
	Decision    string        `json:"decision,omitempty"` // "ACCEPT", "REJECT", "SILENCE"
	Reason      string        `json:"reason,omitempty"`
	WER         float64       `json:"wer"`
	Duration    time.Duration `json:"duration"`
	Error       string        `json:"error,omitempty"`
}

// SuiteReport contains aggregated evaluation statistics.
type SuiteReport struct {
	TotalCases int          `json:"total_cases"`
	Passed     int          `json:"passed"`
	Failed     int          `json:"failed"`
	PassRate   float64      `json:"pass_rate"`
	AvgWER     float64      `json:"avg_wer"`
	Duration   time.Duration `json:"duration"`
	Results    []ItemResult `json:"results"`
}

// EvalRunner coordinates synthetic speech playback into the server pipeline and collects metrics.
type EvalRunner struct {
	server *server.Server
}

// NewEvalRunner creates a new evaluation runner attached to a server.
func NewEvalRunner(srv *server.Server) *EvalRunner {
	return &EvalRunner{server: srv}
}

// EvaluateItem streams a single test item's WAV audio into the pipeline and evaluates the result.
func (r *EvalRunner) EvaluateItem(ctx context.Context, item EvalItem, wavPath string) (*ItemResult, error) {
	t0 := time.Now()
	res := &ItemResult{
		ID:          item.ID,
		Description: item.Description,
		Mode:        item.Mode,
	}

	wavData, err := os.ReadFile(wavPath)
	if err != nil {
		res.Error = fmt.Sprintf("cannot read wav: %v", err)
		return res, err
	}

	samples, sampleRate, err := audio.DecodeWAV(bytes.NewReader(wavData))
	if err != nil {
		res.Error = fmt.Sprintf("cannot decode wav: %v", err)
		return res, err
	}
	if sampleRate != 16000 {
		samples = audio.Resample(samples, sampleRate, 16000)
	}

	// Spin up test HTTP/WS server for isolated test connection
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", r.server.HandleWebSocket)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		res.Error = fmt.Sprintf("ws dial failed: %v", err)
		return res, err
	}
	defer conn.Close()

	// Drain messages in background
	var (
		mu              sync.Mutex
		receivedIntent  string
		receivedConf    float64
		receivedReject  string
		finalTranscript string
		receivedDraft   string
	)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			var msg struct {
				Event string                 `json:"event"`
				Data  map[string]interface{} `json:"data"`
			}
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			mu.Lock()
			switch msg.Event {
			case "intent_matched":
				if id, ok := msg.Data["intent_id"].(string); ok {
					receivedIntent = id
				}
				if c, ok := msg.Data["confidence"].(float64); ok {
					receivedConf = c
				}
			case "utterance_rejected":
				if r, ok := msg.Data["reason"].(string); ok {
					receivedReject = r
				}
				if t, ok := msg.Data["raw_transcript"].(string); ok && finalTranscript == "" {
					finalTranscript = t
				}
			case "transcript.final":
				if t, ok := msg.Data["transcript"].(string); ok {
					finalTranscript = t
				}
			case "annotation.draft":
				if d, ok := msg.Data["draft_text"].(string); ok {
					receivedDraft = d
				}
			}
			mu.Unlock()
		}
	}()

	// 1. Reset state to IDLE and activate mode
	_ = conn.WriteJSON(map[string]interface{}{"action": "cancel"})
	time.Sleep(50 * time.Millisecond)

	if item.Mode == ModeDictation {
		_ = conn.WriteJSON(map[string]interface{}{
			"action":    "activate",
			"mode":      "dictation",
			"target_id": "eval_scope",
		})
	} else {
		_ = conn.WriteJSON(map[string]interface{}{
			"action": "activate",
			"mode":   "command",
		})
	}

	time.Sleep(50 * time.Millisecond)

	// 2. Stream audio chunks
	chunkSize := 480 // 30ms @ 16kHz
	for i := 0; i < len(samples); i += chunkSize {
		end := min(i+chunkSize, len(samples))
		chunk := samples[i:end]
		pcm := audio.Float32ToBytesPCM(chunk)
		if err := conn.WriteMessage(websocket.BinaryMessage, pcm); err != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	// 3. Follow with post-speech silence chunks to trigger ASR trailing-silence endpointing
	silencePCM := make([]byte, chunkSize*2) // 480 int16 samples of zeroes
	for k := 0; k < 30; k++ {               // ~900ms additional silence
		_ = conn.WriteMessage(websocket.BinaryMessage, silencePCM)
		time.Sleep(5 * time.Millisecond)
	}

	// 4. For dictation, stop listening to trigger final flush
	if item.Mode == ModeDictation {
		time.Sleep(100 * time.Millisecond)
		_ = conn.WriteJSON(map[string]interface{}{
			"action": "stop",
			"mode":   "dictation",
		})
	}

	// 4. Wait for processing or timeout
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		hasOutcome := false
		if item.Mode == ModeDictation {
			hasOutcome = receivedDraft != ""
		} else if strings.HasPrefix(item.Expected, "REJECT") {
			hasOutcome = receivedReject != ""
		} else if item.Expected == "DROPPED_BY_GATE" {
			// Expect nothing triggered
			hasOutcome = false
		} else {
			hasOutcome = receivedIntent != ""
		}
		mu.Unlock()

		if hasOutcome {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	res.Duration = time.Since(t0)

	mu.Lock()
	defer mu.Unlock()

	res.IntentID = receivedIntent
	res.Confidence = receivedConf
	res.Reason = receivedReject
	if item.Mode == ModeDictation {
		res.Transcript = receivedDraft
	} else {
		res.Transcript = finalTranscript
	}

	// Check pass / fail conditions
	switch {
	case item.Expected == "DROPPED_BY_GATE":
		// Must not trigger intent or create dictation draft
		res.Passed = (receivedIntent == "" && receivedDraft == "")
		res.Decision = "SILENCE"

	case strings.HasPrefix(item.Expected, "REJECT:"):
		expectedReason := strings.TrimPrefix(item.Expected, "REJECT:")
		res.Passed = (receivedReject == expectedReason || (receivedReject != "" && expectedReason == ""))
		res.Decision = "REJECT"

	case item.Mode == ModeDictation:
		expectedSubstr := strings.ToLower(item.Expected)
		res.Passed = strings.Contains(strings.ToLower(receivedDraft), expectedSubstr)
		res.WER = CalculateWER(item.Text, receivedDraft)
		res.Decision = "ACCEPT"

	default: // Command intent match
		res.Passed = (receivedIntent == item.Expected)
		res.WER = CalculateWER(item.Text, finalTranscript)
		res.Decision = "ACCEPT"
	}

	return res, nil
}
