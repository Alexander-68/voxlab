package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"voxlab/pkg/evaluation"
)

// EvalState tracks active evaluation progress.
type EvalState struct {
	Running   bool                   `json:"running"`
	Tier      string                 `json:"tier,omitempty"`
	Current   int                    `json:"current"`
	Total     int                    `json:"total"`
	LastItem  *evaluation.ItemResult `json:"last_item,omitempty"`
	HasReport bool                   `json:"has_report"`
}

// StartEvaluation kicks off an evaluation suite in the background.
func (s *Server) StartEvaluation(tier string) error {
	s.evalMu.Lock()
	if s.evalRunning {
		s.evalMu.Unlock()
		return fmt.Errorf("evaluation is already running")
	}

	tier = strings.TrimSpace(strings.ToLower(tier))
	if tier == "" || tier == "2" || strings.Contains(tier, "tier-2") || strings.Contains(tier, "deep") {
		tier = "tier-2-deep-matrix"
	} else {
		tier = "tier-1-smoke"
	}

	manifestPath := filepath.Join("data", "test_samples", "eval_tier2", "manifest.json")
	if tier == "tier-1-smoke" {
		manifestPath = filepath.Join("data", "test_samples", "eval", "manifest.json")
	}

	manifest, err := evaluation.LoadManifest(manifestPath)
	if err != nil {
		s.evalMu.Unlock()
		return fmt.Errorf("failed loading manifest %s: %w", manifestPath, err)
	}

	evalCtx, evalCancel := context.WithCancel(context.Background())
	s.evalRunning = true
	s.evalTier = tier
	s.evalProgress = 0
	s.evalTotal = len(manifest.Items)
	s.evalCancel = evalCancel
	s.lastEvalItem = nil
	manifestDir := filepath.Dir(manifestPath)
	s.evalMu.Unlock()

	log.Printf("[Evaluation] Starting %s with %d items...", tier, len(manifest.Items))
	s.BroadcastJSON(map[string]interface{}{
		"event": "eval.started",
		"data": map[string]interface{}{
			"tier":  tier,
			"total": len(manifest.Items),
		},
	})

	go func() {
		restorePipeline := s.pinEvalPipeline()

		report, err := s.evalRunner.EvaluateSuite(evalCtx, manifest, manifestDir, func(res *evaluation.ItemResult, cur, tot int) {
			s.evalMu.Lock()
			s.evalProgress = cur
			s.evalTotal = tot
			s.lastEvalItem = res
			s.evalMu.Unlock()

			s.BroadcastJSON(map[string]interface{}{
				"event": "eval.progress",
				"data": map[string]interface{}{
					"tier":    tier,
					"current": cur,
					"total":   tot,
					"result":  res,
				},
			})
		})

		if err != nil && err != context.Canceled {
			log.Printf("[Evaluation] Suite completed with note: %v", err)
		}
		restorePipeline()

		// A run is over only here, so a new one cannot start while this one is stopping.
		cancelled := evalCtx.Err() != nil
		evalCancel()
		s.evalMu.Lock()
		s.evalRunning = false
		s.evalCancel = nil
		if !cancelled {
			s.lastReport = report
		}
		s.evalMu.Unlock()

		if cancelled {
			s.BroadcastJSON(map[string]interface{}{
				"event": "eval.cancelled",
				"data":  map[string]interface{}{},
			})
			log.Println("[Evaluation] Active evaluation cancelled by user request.")
			return
		}

		// Persist report JSON to data directory
		reportFile := filepath.Join("data", fmt.Sprintf("eval_report_%s.json", tier))
		if repData, err := json.MarshalIndent(report, "", "  "); err == nil {
			_ = os.WriteFile(reportFile, repData, 0644)
		}

		s.BroadcastJSON(map[string]interface{}{
			"event": "eval.completed",
			"data": map[string]interface{}{
				"tier":   tier,
				"report": report,
			},
		})
		log.Printf("[Evaluation] Completed %s: %d/%d passed (%.1f%%)",
			tier, report.Passed, report.TotalCases, report.PassRate*100.0)
	}()

	return nil
}

// pinEvalPipeline routes host ASR to the harness for a run and returns a function restoring the
// previous setup: harness audio only flows from the web mic source, and browser ASR ignores it.
// ponytail: shares the live pipeline, so another tab's controls still reach it during a run.
// A separate pipeline instance per run would remove that.
func (s *Server) pinEvalPipeline() (restore func()) {
	prevSource, prevEngine := s.currentSource(), s.ASREngineMode()
	s.SwitchSource("web_ui_mic")
	if prevEngine == "browser" {
		mode := "sim"
		if s.cfg.Engine.Mode == "sherpa" {
			mode = "sherpa"
		}
		if err := s.SetASREngineMode(mode); err != nil {
			log.Printf("[Evaluation] Could not switch ASR engine: %v", err)
		}
	}
	return func() {
		s.SwitchSource(prevSource)
		if prevEngine == "browser" {
			_ = s.SetASREngineMode(prevEngine)
		}
	}
}

// evalRunConfig records the pipeline settings a report's numbers depend on.
func (s *Server) evalRunConfig() evaluation.RunConfig {
	cfg := evaluation.RunConfig{
		Engine:      s.ASREngineMode(),
		Feed:        "realtime",
		GateDBFS:    s.dsp.GateThreshold(),
		GateEnabled: s.dsp.IsGateEnabled(),
		AGC:         s.dsp.IsAGCEnabled(),
		HighPass:    s.dsp.IsHighPassEnabled(),
		Enhancer:    s.enhancerName(),
	}
	if info := s.asrModelInfo(); info != nil && cfg.Engine == "sherpa" {
		cfg.ASRModel, _ = info["asr_model"].(string)
		cfg.SecondPassModel, _ = info["second_pass_model"].(string)
		cfg.SecondPassCommand, _ = info["second_pass_command"].(bool)
		cfg.SecondPassDictation, _ = info["second_pass_dictation"].(bool)
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		dirty := ""
		for _, setting := range bi.Settings {
			switch {
			case setting.Key == "vcs.revision":
				cfg.Commit = setting.Value
			case setting.Key == "vcs.modified" && setting.Value == "true":
				dirty = "-dirty" // the revision alone would claim code this build does not match
			}
		}
		cfg.Commit += dirty
	}
	return cfg
}

// EvalRunner returns the harness, which stamps its reports with this server's configuration.
func (s *Server) EvalRunner() *evaluation.EvalRunner {
	return s.evalRunner
}

// CancelEvaluation asks an in-progress run to stop. The run announces eval.cancelled once stopped.
func (s *Server) CancelEvaluation() {
	s.evalMu.Lock()
	defer s.evalMu.Unlock()
	if s.evalCancel != nil {
		s.evalCancel()
	}
}

// EvalStatus returns the current state of the evaluation harness.
func (s *Server) EvalStatus() EvalState {
	s.evalMu.Lock()
	defer s.evalMu.Unlock()
	return EvalState{
		Running:   s.evalRunning,
		Tier:      s.evalTier,
		Current:   s.evalProgress,
		Total:     s.evalTotal,
		LastItem:  s.lastEvalItem,
		HasReport: s.lastReport != nil,
	}
}

// LastReport returns the latest completed SuiteReport.
func (s *Server) LastReport() *evaluation.SuiteReport {
	s.evalMu.Lock()
	defer s.evalMu.Unlock()
	return s.lastReport
}

// REST Handlers
func (s *Server) handleEvalStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.EvalStatus())
}

func (s *Server) handleEvalStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tier := r.URL.Query().Get("tier")
	if tier == "" {
		var body struct {
			Tier string `json:"tier"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil && body.Tier != "" {
			tier = body.Tier
		}
	}
	if tier == "" {
		tier = "tier-2-deep-matrix"
	}

	if err := s.StartEvaluation(tier); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "started",
		"tier":    tier,
		"message": "Evaluation suite initiated",
	})
}

func (s *Server) handleEvalCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.CancelEvaluation()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "cancelled",
		"message": "Evaluation cancelled",
	})
}

func (s *Server) handleEvalReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rep := s.LastReport()
	if rep != nil {
		_ = json.NewEncoder(w).Encode(rep)
		return
	}

	// Try reading from disk fallback. The tier becomes part of a path, so only known tiers pass.
	tier := r.URL.Query().Get("tier")
	if tier == "" {
		tier = "tier-2-deep-matrix"
	}
	if tier != "tier-1-smoke" && tier != "tier-2-deep-matrix" {
		http.Error(w, "unknown tier", http.StatusBadRequest)
		return
	}
	reportFile := filepath.Join("data", fmt.Sprintf("eval_report_%s.json", tier))
	if data, err := os.ReadFile(reportFile); err == nil {
		_, _ = w.Write(data)
		return
	}

	http.Error(w, "no report available", http.StatusNotFound)
}
