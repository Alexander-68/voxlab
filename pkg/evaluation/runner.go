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
)

// ItemResult contains the evaluation verdict and metrics for one test item.
type ItemResult struct {
	ID                string        `json:"id"`
	Description       string        `json:"description"`
	Mode              TestCaseMode  `json:"mode"`
	Voice             string        `json:"voice,omitempty"`
	SpeakingRate      float64       `json:"speaking_rate,omitempty"`
	SNR               *float64      `json:"snr_db,omitempty"`
	NoiseType         string        `json:"noise_type,omitempty"`
	Passed            bool          `json:"passed"`
	Transcript        string        `json:"transcript"`
	IntentID          string        `json:"intent_id,omitempty"`
	Confidence        float64       `json:"confidence,omitempty"`
	Decision          string        `json:"decision,omitempty"` // "ACCEPT", "REJECT", "SILENCE"
	Reason            string        `json:"reason,omitempty"`
	WER               float64       `json:"wer"`
	CER               float64       `json:"cer"`
	Duration          time.Duration `json:"duration"`
	TTFT              time.Duration `json:"ttft"`
	EndpointDelay     time.Duration `json:"endpoint_delay"`
	SecondPassLatency time.Duration `json:"second_pass_latency"`
	Error             string        `json:"error,omitempty"`
}

// EvalRunner coordinates synthetic speech playback into the server pipeline and collects metrics.
type EvalRunner struct {
	handler http.HandlerFunc
	wsURL   string
}

// NewEvalRunner creates a new evaluation runner attached to a WebSocket handler function.
func NewEvalRunner(handler http.HandlerFunc) *EvalRunner {
	return &EvalRunner{handler: handler}
}

// NewEvalRunnerWithURL creates a new evaluation runner connecting directly to an active WebSocket URL.
func NewEvalRunnerWithURL(wsURL string) *EvalRunner {
	return &EvalRunner{wsURL: wsURL}
}

// EvaluateSuite evaluates all items in a manifest, reporting item-by-item progress.
func (r *EvalRunner) EvaluateSuite(ctx context.Context, manifest *EvalManifest, manifestDir string, onProgress func(res *ItemResult, current, total int)) (*SuiteReport, error) {
	t0 := time.Now()
	var results []ItemResult

	total := len(manifest.Items)
	for i, item := range manifest.Items {
		select {
		case <-ctx.Done():
			return ComputeSuiteReport(manifest.Tier, results, time.Since(t0)), ctx.Err()
		default:
		}

		wavPath := ResolveWavPath(manifestDir, item)
		itemCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
		res, err := r.EvaluateItem(itemCtx, item, wavPath)
		cancel()

		if err != nil {
			res.Error = err.Error()
		}
		results = append(results, *res)

		if onProgress != nil {
			onProgress(res, i+1, total)
		}
	}

	report := ComputeSuiteReport(manifest.Tier, results, time.Since(t0))
	return report, nil
}

// EvaluateItem streams a single test item's WAV audio into the pipeline and evaluates the result.
func (r *EvalRunner) EvaluateItem(ctx context.Context, item EvalItem, wavPath string) (*ItemResult, error) {
	t0 := time.Now()
	res := &ItemResult{
		ID:           item.ID,
		Description:  item.Description,
		Mode:         item.Mode,
		Voice:        item.Voice,
		SpeakingRate: item.SpeakingRate,
		SNR:          item.SNR,
		NoiseType:    item.NoiseType,
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

	var (
		conn *websocket.Conn
		ts   *httptest.Server
	)
	if r.wsURL != "" {
		c, _, err := websocket.DefaultDialer.Dial(r.wsURL, nil)
		if err != nil {
			res.Error = fmt.Sprintf("ws dial failed: %v", err)
			return res, err
		}
		conn = c
	} else if r.handler != nil {
		mux := http.NewServeMux()
		mux.HandleFunc("/ws", r.handler)
		ts = httptest.NewServer(mux)
		defer ts.Close()

		wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
		c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			res.Error = fmt.Sprintf("ws dial failed: %v", err)
			return res, err
		}
		conn = c
	} else {
		res.Error = "no websocket handler or URL configured"
		return res, fmt.Errorf("no websocket handler or URL configured")
	}
	defer conn.Close()

	// Drain messages in background
	var (
		mu                 sync.Mutex
		receivedIntent     string
		receivedConf       float64
		receivedReject     string
		finalTranscript    string
		receivedDraft      string
		firstPartialTime   time.Time
		finalRecvTime      time.Time
		secondPassRecvTime time.Time
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
			now := time.Now()
			switch msg.Event {
			case "transcript.partial":
				if firstPartialTime.IsZero() {
					firstPartialTime = now
				}
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
				if finalRecvTime.IsZero() {
					finalRecvTime = now
				}
			case "transcript.final":
				if t, ok := msg.Data["transcript"].(string); ok {
					finalTranscript = t
				}
				if finalRecvTime.IsZero() {
					finalRecvTime = now
				}
			case "transcript.refined":
				if secondPassRecvTime.IsZero() {
					secondPassRecvTime = now
				}
			case "annotation.draft":
				if d, ok := msg.Data["draft_text"].(string); ok {
					receivedDraft = d
				}
				if secondPassRecvTime.IsZero() {
					secondPassRecvTime = now
				}
			}
			mu.Unlock()
		}
	}()

	// 1. Reset state to IDLE and activate mode
	_ = conn.WriteJSON(map[string]interface{}{"action": "cancel"})
	time.Sleep(40 * time.Millisecond)

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

	time.Sleep(40 * time.Millisecond)

	// 2. Stream audio chunks
	streamStartTime := time.Now()
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
	speechEndTime := time.Now()

	// 3. Follow with post-speech silence chunks to trigger ASR trailing-silence endpointing
	silencePCM := make([]byte, chunkSize*2) // 480 int16 samples of zeroes
	for k := 0; k < 45; k++ {               // ~1.35s additional silence to guarantee Sherpa rule2 endpointing
		_ = conn.WriteMessage(websocket.BinaryMessage, silencePCM)
		time.Sleep(10 * time.Millisecond)
	}

	// 4. For dictation, stop listening to trigger final flush
	if item.Mode == ModeDictation {
		time.Sleep(100 * time.Millisecond)
		_ = conn.WriteJSON(map[string]interface{}{
			"action": "stop",
			"mode":   "dictation",
		})
	}

	// 5. Wait for processing or timeout
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		hasOutcome := false
		if item.Mode == ModeDictation {
			hasOutcome = receivedDraft != ""
		} else if strings.HasPrefix(item.Expected, "REJECT") {
			hasOutcome = receivedReject != ""
		} else if item.Expected == "DROPPED_BY_GATE" {
			hasOutcome = false
		} else {
			hasOutcome = receivedIntent != ""
		}
		mu.Unlock()

		if hasOutcome {
			break
		}
		time.Sleep(60 * time.Millisecond)
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

	// Compute latencies
	if !firstPartialTime.IsZero() && firstPartialTime.After(streamStartTime) {
		res.TTFT = firstPartialTime.Sub(streamStartTime)
	}
	if !finalRecvTime.IsZero() && finalRecvTime.After(speechEndTime) {
		res.EndpointDelay = finalRecvTime.Sub(speechEndTime)
	}
	if !secondPassRecvTime.IsZero() && !finalRecvTime.IsZero() && secondPassRecvTime.After(finalRecvTime) {
		res.SecondPassLatency = secondPassRecvTime.Sub(finalRecvTime)
	}

	// Check pass / fail conditions
	switch {
	case item.Expected == "DROPPED_BY_GATE":
		// Must not trigger intent or create dictation draft
		res.Passed = (receivedIntent == "" && receivedDraft == "")
		res.Decision = "SILENCE"
		res.WER = 0.0
		res.CER = 0.0

	case strings.HasPrefix(item.Expected, "REJECT:"):
		expectedReason := strings.TrimPrefix(item.Expected, "REJECT:")
		res.Passed = (receivedReject == expectedReason || (receivedReject != "" && expectedReason == ""))
		res.Decision = "REJECT"
		res.WER = CalculateWER(item.Text, finalTranscript)
		res.CER = CalculateCER(item.Text, finalTranscript)

	case item.Mode == ModeDictation:
		expectedSubstr := strings.ToLower(item.Expected)
		res.WER = CalculateWER(item.Text, receivedDraft)
		res.CER = CalculateCER(item.Text, receivedDraft)
		res.Passed = (expectedSubstr != "" && strings.Contains(strings.ToLower(receivedDraft), expectedSubstr)) || (res.WER <= 0.35 && len(receivedDraft) > 0)
		res.Decision = "ACCEPT"

	default: // Command intent match
		res.Passed = (receivedIntent == item.Expected)
		res.WER = CalculateWER(item.Text, finalTranscript)
		res.CER = CalculateCER(item.Text, finalTranscript)
		res.Decision = "ACCEPT"
	}

	return res, nil
}
