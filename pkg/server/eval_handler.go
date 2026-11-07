package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
		defer func() {
			s.evalMu.Lock()
			s.evalRunning = false
			s.evalCancel = nil
			s.evalMu.Unlock()
		}()

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

		s.evalMu.Lock()
		s.lastReport = report
		s.evalMu.Unlock()

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

// CancelEvaluation terminates any in-progress benchmark run.
func (s *Server) CancelEvaluation() {
	s.evalMu.Lock()
	if s.evalRunning && s.evalCancel != nil {
		s.evalCancel()
		s.evalRunning = false
	}
	s.evalMu.Unlock()

	s.BroadcastJSON(map[string]interface{}{
		"event": "eval.cancelled",
		"data":  map[string]interface{}{},
	})
	log.Println("[Evaluation] Active evaluation cancelled by user request.")
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

	// Try reading from disk fallback
	tier := r.URL.Query().Get("tier")
	if tier == "" {
		tier = "tier-2-deep-matrix"
	}
	reportFile := filepath.Join("data", fmt.Sprintf("eval_report_%s.json", tier))
	if data, err := os.ReadFile(reportFile); err == nil {
		_, _ = w.Write(data)
		return
	}

	http.Error(w, "no report available", http.StatusNotFound)
}
