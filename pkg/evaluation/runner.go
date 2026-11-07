package evaluation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
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
	Accent            string        `json:"accent,omitempty"`
	WavFile           string        `json:"wav_file,omitempty"`
	Passed            bool          `json:"passed"`
	Transcript        string        `json:"transcript"`
	IntentID          string        `json:"intent_id,omitempty"`
	Confidence        float64       `json:"confidence,omitempty"`
	Class             string        `json:"class,omitempty"`    // what the item expects: "ACCEPT", "REJECT", "SILENCE"
	Decision          string        `json:"decision,omitempty"` // what the pipeline did, same values
	Reason            string        `json:"reason,omitempty"`
	WER               float64       `json:"wer"`
	CER               float64       `json:"cer"`
	Duration          time.Duration `json:"duration"`
	TTFT              time.Duration `json:"ttft"`
	EndpointDelay     time.Duration `json:"endpoint_delay"`
	SecondPassLatency time.Duration `json:"second_pass_latency"`
	Error             string        `json:"error,omitempty"`
	KeyTermHit        bool          `json:"key_term_hit,omitempty"`
	DroppedChunks     int           `json:"dropped_chunks,omitempty"`
}

const (
	sampleRate    = 16000
	chunkSamples  = 480 // 30 ms
	chunkDuration = 30 * time.Millisecond

	// commandSettle bounds trailing silence for commands. The recognizer only reports results
	// while audio arrives, like a live mic, so silence keeps flowing until an outcome or this limit.
	commandSettle = 3 * time.Second
	// dictationTail is the pause before stopping, long enough for a segment to endpoint.
	dictationTail = 1500 * time.Millisecond
	// flushWait covers the recognizer's finish timeout plus a second pass.
	flushWait  = 15 * time.Second
	stateWait  = 3 * time.Second
	itemMargin = 25 * time.Second
)

// ItemTimeout bounds one item: audio is fed in real time, so it scales with the clip.
func ItemTimeout(item EvalItem) time.Duration {
	return time.Duration(item.DurationSec*float64(time.Second)) + itemMargin
}

// EvalRunner coordinates synthetic speech playback into the server pipeline and collects metrics.
type EvalRunner struct {
	handler http.HandlerFunc
	wsURL   string

	// Describe reports the pipeline configuration to stamp on reports.
	Describe func() RunConfig
	// OnItemStart announces the clip about to be fed, for anyone listening along.
	OnItemStart func(item EvalItem, current, total int)
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

	finish := func() *SuiteReport {
		report := ComputeSuiteReport(manifest.Tier, results, time.Since(t0))
		if r.Describe != nil {
			cfg := r.Describe()
			report.Config = &cfg
		}
		return report
	}

	total := len(manifest.Items)
	for i, item := range manifest.Items {
		if r.OnItemStart != nil {
			r.OnItemStart(item, i+1, total)
		}
		wavPath := ResolveWavPath(manifestDir, item)
		itemCtx, cancel := context.WithTimeout(ctx, ItemTimeout(item))
		res, err := r.EvaluateItem(itemCtx, item, wavPath)
		cancel()

		// A run stopped mid-item has no result for that item.
		if ctx.Err() != nil {
			return finish(), ctx.Err()
		}
		if err != nil {
			res.Error = err.Error()
		}
		results = append(results, *res)

		if onProgress != nil {
			onProgress(res, i+1, total)
		}
	}

	return finish(), nil
}

// speechEnd locates the last chunk with speech-level energy; clips end in padding silence.
// ponytail: energy relative to the loudest chunk, so at low SNR this drifts toward the clip end.
func speechEnd(samples []float32) time.Duration {
	var rms []float64
	for i := 0; i < len(samples); i += chunkSamples {
		var sum float64
		chunk := samples[i:min(i+chunkSamples, len(samples))]
		for _, v := range chunk {
			sum += float64(v) * float64(v)
		}
		rms = append(rms, math.Sqrt(sum/float64(len(chunk))))
	}
	var peak float64
	for _, v := range rms {
		peak = max(peak, v)
	}
	for i := len(rms) - 1; i >= 0; i-- {
		if rms[i] >= 0.1*peak {
			return time.Duration(i+1) * chunkDuration
		}
	}
	return 0
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
		Accent:       item.Accent,
		WavFile:      item.WavFile,
	}

	wavData, err := os.ReadFile(wavPath)
	if err != nil {
		res.Error = fmt.Sprintf("cannot read wav: %v", err)
		return res, err
	}

	samples, wavRate, err := audio.DecodeWAV(bytes.NewReader(wavData))
	if err != nil {
		res.Error = fmt.Sprintf("cannot decode wav: %v", err)
		return res, err
	}
	if wavRate != sampleRate {
		samples = audio.Resample(samples, wavRate, sampleRate)
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
		mu               sync.Mutex
		state            string
		receivedIntent   string
		receivedConf     float64
		receivedReject   string
		receivedClarify  string
		finalTranscript  string
		receivedDraft    string
		droppedChunks    int
		feeding          bool // this item's audio is flowing, so meter events describe it
		gateOpened       bool
		firstPartialTime time.Time
		finalRecvTime    time.Time
		secondPassMs     float64 // summed: a dictation can refine several segments
	)

	go func() {
		for {
			kind, payload, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if kind != websocket.TextMessage { // monitor audio
				continue
			}
			var msg struct {
				Event string                 `json:"event"`
				Data  map[string]interface{} `json:"data"`
			}
			if json.Unmarshal(payload, &msg) != nil {
				continue
			}
			mu.Lock()
			now := time.Now()
			switch msg.Event {
			case "voice.state":
				if to, ok := msg.Data["to_state"].(string); ok {
					state = to
				}
			case "audio.meter":
				if open, _ := msg.Data["passed_gate"].(bool); open && feeding {
					gateOpened = true
				}
			case "asr.queue_overflow":
				droppedChunks++
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
			case "intent_clarification_required":
				if r, ok := msg.Data["reason"].(string); ok {
					receivedClarify = r
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
				if ms, ok := msg.Data["latency_ms"].(float64); ok {
					secondPassMs += ms
				}
			case "annotation.draft":
				if d, ok := msg.Data["draft_text"].(string); ok {
					receivedDraft = d
				}
			}
			mu.Unlock()
		}
	}()

	inState := func(want string) func() bool {
		return func() bool {
			mu.Lock()
			defer mu.Unlock()
			return state == want
		}
	}
	commandDone := func() bool {
		mu.Lock()
		defer mu.Unlock()
		return receivedIntent != "" || receivedReject != "" || receivedClarify != ""
	}
	waitFor := func(done func() bool, limit time.Duration) error {
		deadline := time.Now().Add(limit)
		for !done() {
			if time.Now().After(deadline) {
				return fmt.Errorf("timed out after %v", limit)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Millisecond):
			}
		}
		return nil
	}

	// 1. Claim the pipeline, return to IDLE and activate. Waiting for each state keeps the
	// item's audio inside its own turn.
	_ = conn.WriteJSON(map[string]interface{}{"action": "eval_session"})
	_ = conn.WriteJSON(map[string]interface{}{"action": "cancel"})
	defer func() { _ = conn.WriteJSON(map[string]interface{}{"action": "cancel"}) }()
	if err := waitFor(inState("IDLE"), stateWait); err != nil {
		return res, fmt.Errorf("waiting for IDLE: %w", err)
	}

	listenState := "COMMAND_LISTEN"
	if item.Mode == ModeDictation {
		listenState = "ANNOTATION_LISTEN"
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
	if err := waitFor(inState(listenState), stateWait); err != nil {
		return res, fmt.Errorf("waiting for %s: %w", listenState, err)
	}

	// 2. Stream audio at real time, as a microphone would. Latencies are only meaningful at
	// this pace, and faster feeds overflow the ASR queue during a second pass.
	streamStartTime := time.Now()
	mu.Lock()
	feeding = true
	mu.Unlock()
	nextChunk := streamStartTime
	send := func(pcm []byte) error {
		if err := conn.WriteMessage(websocket.BinaryMessage, pcm); err != nil {
			return err
		}
		nextChunk = nextChunk.Add(chunkDuration)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Until(nextChunk)):
			return nil
		}
	}

	isGateItem := item.Expected == "DROPPED_BY_GATE"
	stopEarly := func() bool { return item.Mode != ModeDictation && commandDone() }
	for i := 0; i < len(samples) && !stopEarly(); i += chunkSamples {
		if err := send(audio.Float32ToBytesPCM(samples[i:min(i+chunkSamples, len(samples))])); err != nil {
			return res, err
		}
	}
	speechEndTime := streamStartTime.Add(speechEnd(samples))

	// 3. Follow with silence so the recognizer endpoints and reports its results.
	silenceLimit := commandSettle
	if item.Mode == ModeDictation || isGateItem {
		silenceLimit = dictationTail
	}
	silencePCM := make([]byte, chunkSamples*2)
	for end := time.Now().Add(silenceLimit); time.Now().Before(end) && !stopEarly(); {
		if err := send(silencePCM); err != nil {
			return res, err
		}
	}

	// 4. For dictation, stop and wait for the flush to finish: the state returns to IDLE only
	// after the last segment reached the draft.
	if item.Mode == ModeDictation {
		_ = conn.WriteJSON(map[string]interface{}{
			"action": "stop",
			"mode":   "dictation",
		})
		if err := waitFor(inState("IDLE"), flushWait); err != nil {
			return res, fmt.Errorf("waiting for dictation flush: %w", err)
		}
	}

	res.Duration = time.Since(t0)

	mu.Lock()
	defer mu.Unlock()

	res.IntentID = receivedIntent
	res.Confidence = receivedConf
	res.Reason = receivedReject
	if res.Reason == "" {
		res.Reason = receivedClarify
	}
	res.DroppedChunks = droppedChunks
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
	res.SecondPassLatency = time.Duration(secondPassMs * float64(time.Millisecond))

	switch {
	case receivedReject != "" || receivedClarify != "":
		res.Decision = "REJECT"
	case receivedIntent != "" || receivedDraft != "":
		res.Decision = "ACCEPT"
	default:
		res.Decision = "SILENCE"
	}

	// Check pass / fail conditions
	switch {
	case item.Expected == "DROPPED_BY_GATE":
		// The gate must hold for the whole clip and nothing may reach recognition. ASR alone is no
		// proof: it hears nothing in faint noise whether or not the gate is doing its job.
		res.Class = "SILENCE"
		res.Passed = res.Decision == "SILENCE" && !gateOpened && firstPartialTime.IsZero() && finalTranscript == ""
		res.WER = 0.0
		res.CER = 0.0

	case strings.HasPrefix(item.Expected, "REJECT:"):
		expectedReason := strings.TrimPrefix(item.Expected, "REJECT:")
		res.Passed = (receivedReject == expectedReason || (receivedReject != "" && expectedReason == ""))
		res.Class = "REJECT"
		res.WER = CalculateWER(item.Text, finalTranscript)
		res.CER = CalculateCER(item.Text, finalTranscript)

	case item.Mode == ModeDictation:
		ref := BestReference(item, receivedDraft)
		res.WER = CalculateWER(ref, receivedDraft)
		res.CER = CalculateCER(ref, receivedDraft)
		_, res.Passed, res.KeyTermHit = ScoreDictation(item, receivedDraft)
		res.Class = "ACCEPT"

	default: // Command intent match
		res.Passed = (receivedIntent == item.Expected)
		res.WER = CalculateWER(item.Text, finalTranscript)
		res.CER = CalculateCER(item.Text, finalTranscript)
		res.Class = "ACCEPT"
	}

	return res, nil
}
