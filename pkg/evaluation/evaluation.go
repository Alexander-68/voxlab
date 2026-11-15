package evaluation

import (
	"time"

	"voxlab/pkg/engine"
	"voxlab/pkg/evaluation/corpus"
	"voxlab/pkg/evaluation/metrics"
)

// Re-export metrics types and functions
type ItemResult = metrics.ItemResult
type SuiteReport = metrics.SuiteReport
type RunConfig = metrics.RunConfig
type LatencyStatistics = metrics.LatencyStatistics
type LatencyPercentiles = metrics.LatencyPercentiles
type SNRCurvePoint = metrics.SNRCurvePoint
type DimensionGroup = metrics.DimensionGroup

// ComputeSuiteReport: see metrics.ComputeSuiteReport.
func ComputeSuiteReport(tier string, results []ItemResult, suiteDuration time.Duration) *SuiteReport {
	return metrics.ComputeSuiteReport(tier, results, suiteDuration)
}

// CalculateWER: see metrics.CalculateWER.
func CalculateWER(reference, hypothesis string) float64 {
	return metrics.CalculateWER(reference, hypothesis)
}

// CalculateCER: see metrics.CalculateCER.
func CalculateCER(reference, hypothesis string) float64 {
	return metrics.CalculateCER(reference, hypothesis)
}

// Re-export corpus types and functions
type TestCaseMode = corpus.TestCaseMode

const (
	ModeCommand   = corpus.ModeCommand
	ModeDictation = corpus.ModeDictation
)

type EvalItem = corpus.EvalItem
type EvalManifest = corpus.EvalManifest

const DefaultMaxErrorRate = corpus.DefaultMaxErrorRate

// ScoreDictation: see corpus.ScoreDictation.
func ScoreDictation(item EvalItem, draft string) (errorRate float64, passed, keyTermHit bool) {
	return corpus.ScoreDictation(item, draft)
}

// BestReference: see corpus.BestReference.
func BestReference(item EvalItem, hypothesis string) string {
	return corpus.BestReference(item, hypothesis)
}

// LoadManifest: see corpus.LoadManifest.
func LoadManifest(path string) (*EvalManifest, error) { return corpus.LoadManifest(path) }

// ResolveWavPath: see corpus.ResolveWavPath.
func ResolveWavPath(manifestDir string, item EvalItem) string {
	return corpus.ResolveWavPath(manifestDir, item)
}

// GenerateTier1Corpus: see corpus.GenerateTier1Corpus.
func GenerateTier1Corpus(outDir string, runner engine.SpeechEngine) (*EvalManifest, error) {
	return corpus.GenerateTier1Corpus(outDir, runner)
}

// GenerateTier2Corpus: see corpus.GenerateTier2Corpus.
func GenerateTier2Corpus(outDir string, runner engine.SpeechEngine) (*EvalManifest, error) {
	return corpus.GenerateTier2Corpus(outDir, runner)
}

type BaseUtterance = corpus.BaseUtterance
type MatrixVariation = corpus.MatrixVariation
