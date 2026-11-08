package evaluation

import (
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

var (
	ComputePercentiles = metrics.ComputePercentiles
	ComputeSuiteReport = metrics.ComputeSuiteReport
	CalculateWER       = metrics.CalculateWER
	CalculateCER       = metrics.CalculateCER
)

// Re-export corpus types and functions
type TestCaseMode = corpus.TestCaseMode

const (
	ModeCommand   = corpus.ModeCommand
	ModeDictation = corpus.ModeDictation
)

type EvalItem = corpus.EvalItem
type EvalManifest = corpus.EvalManifest

const DefaultMaxErrorRate = corpus.DefaultMaxErrorRate

var (
	ScoreDictation        = corpus.ScoreDictation
	BestReference         = corpus.BestReference
	LoadManifest          = corpus.LoadManifest
	ResolveWavPath        = corpus.ResolveWavPath
	DefaultBaseUtterances = corpus.DefaultBaseUtterances
	GenerateTier1Corpus   = corpus.GenerateTier1Corpus
	GenerateTier2Corpus   = corpus.GenerateTier2Corpus
	LoadOrSynthesizeClip  = corpus.LoadOrSynthesizeClip
)

type BaseUtterance = corpus.BaseUtterance
type MatrixVariation = corpus.MatrixVariation
