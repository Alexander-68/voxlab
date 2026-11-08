package metrics

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"
)

// TestCaseMode specifies whether audio is evaluated as a command or dictation.
type TestCaseMode string

const (
	ModeCommand   TestCaseMode = "command"
	ModeDictation TestCaseMode = "dictation"
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

// LatencyPercentiles summarizes response times across percentiles.
type LatencyPercentiles struct {
	MinMs float64 `json:"min_ms"`
	MaxMs float64 `json:"max_ms"`
	AvgMs float64 `json:"avg_ms"`
	P50Ms float64 `json:"p50_ms"`
	P90Ms float64 `json:"p90_ms"`
	P95Ms float64 `json:"p95_ms"`
	P99Ms float64 `json:"p99_ms"`
}

// LatencyStatistics aggregates TTFT, silence endpoint delay, refinement overhead, and total duration.
type LatencyStatistics struct {
	TTFT          LatencyPercentiles `json:"ttft"`
	EndpointDelay LatencyPercentiles `json:"endpoint_delay"`
	SecondPass    LatencyPercentiles `json:"second_pass"`
	TotalDuration LatencyPercentiles `json:"total_duration"`
}

// SNRCurvePoint measures performance degradation at a specific SNR level.
type SNRCurvePoint struct {
	SNR            *float64 `json:"snr_db,omitempty"`
	Label          string   `json:"label"` // e.g. "Clean", "+20 dB", "+10 dB", "0 dB", "-5 dB"
	Count          int      `json:"count"`
	Passed         int      `json:"passed"`
	PassRate       float64  `json:"pass_rate"`
	AvgWER         float64  `json:"avg_wer"`
	AvgCER         float64  `json:"avg_cer"`
	IntentAccuracy float64  `json:"intent_accuracy"`
	AvgLatencyMs   float64  `json:"avg_latency_ms"`
}

// DimensionGroup aggregates accuracy metrics for a specific dimension (voice, rate, noise type).
type DimensionGroup struct {
	Key      string  `json:"key"`
	Count    int     `json:"count"`
	Passed   int     `json:"passed"`
	PassRate float64 `json:"pass_rate"`
	AvgWER   float64 `json:"avg_wer"`
	AvgCER   float64 `json:"avg_cer"`
}

// RunConfig records what produced a report, so its numbers can be compared and reproduced.
type RunConfig struct {
	Engine              string  `json:"engine"`
	ASRModel            string  `json:"asr_model,omitempty"`
	SecondPassModel     string  `json:"second_pass_model,omitempty"`
	SecondPassCommand   bool    `json:"second_pass_command"`
	SecondPassDictation bool    `json:"second_pass_dictation"`
	Feed                string  `json:"feed"`
	GateDBFS            float64 `json:"gate_dbfs"`
	GateEnabled         bool    `json:"gate_enabled"`
	AGC                 bool    `json:"agc"`
	HighPass            bool    `json:"highpass"`
	Enhancer            string  `json:"enhancer"`
	Commit              string  `json:"commit,omitempty"` // empty when the binary has no VCS stamp
}

// SuiteReport contains aggregated evaluation statistics for a benchmark run.
type SuiteReport struct {
	Tier              string                    `json:"tier"` // "tier-1-smoke" or "tier-2-deep-matrix"
	Config            *RunConfig                `json:"config,omitempty"`
	DroppedChunks     int                       `json:"dropped_chunks"` // audio lost to ASR queue overflow, across all items
	TotalCases        int                       `json:"total_cases"`
	Passed            int                       `json:"passed"`
	Failed            int                       `json:"failed"`
	PassRate          float64                   `json:"pass_rate"`
	AvgWER            float64                   `json:"avg_wer"`
	AvgCER            float64                   `json:"avg_cer"`
	IntentAccuracy    *float64                  `json:"intent_accuracy"`    // null: no such items in this run
	RejectionAccuracy *float64                  `json:"rejection_accuracy"` // null: no such items in this run
	GatePrecision     *float64                  `json:"gate_precision"`     // null: no such items in this run
	Duration          time.Duration             `json:"duration"`
	LatencyStats      LatencyStatistics         `json:"latency_stats"`
	SNRCurve          []SNRCurvePoint           `json:"snr_curve"`
	VoiceBreakdown    map[string]DimensionGroup `json:"voice_breakdown"`
	RateBreakdown     map[string]DimensionGroup `json:"rate_breakdown"`
	NoiseBreakdown    map[string]DimensionGroup `json:"noise_breakdown"`
	Results           []ItemResult              `json:"results"`
}

// ComputePercentiles computes min, max, avg, and p50..p99 for a list of millisecond durations.
func ComputePercentiles(values []float64) LatencyPercentiles {
	if len(values) == 0 {
		return LatencyPercentiles{}
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	var sum float64
	for _, v := range sorted {
		sum += v
	}

	getPercentile := func(p float64) float64 {
		if len(sorted) == 1 {
			return sorted[0]
		}
		idx := int(math.Round(float64(len(sorted)-1) * p))
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		return sorted[idx]
	}

	return LatencyPercentiles{
		MinMs: math.Round(sorted[0]*10) / 10,
		MaxMs: math.Round(sorted[len(sorted)-1]*10) / 10,
		AvgMs: math.Round((sum/float64(len(sorted)))*10) / 10,
		P50Ms: math.Round(getPercentile(0.50)*10) / 10,
		P90Ms: math.Round(getPercentile(0.90)*10) / 10,
		P95Ms: math.Round(getPercentile(0.95)*10) / 10,
		P99Ms: math.Round(getPercentile(0.99)*10) / 10,
	}
}

// ComputeSuiteReport aggregates item results into a full evaluation report with SNR curve and percentiles.
func ComputeSuiteReport(tier string, results []ItemResult, suiteDuration time.Duration) *SuiteReport {
	report := &SuiteReport{
		Tier:           tier,
		TotalCases:     len(results),
		Results:        results,
		Duration:       suiteDuration,
		VoiceBreakdown: make(map[string]DimensionGroup),
		RateBreakdown:  make(map[string]DimensionGroup),
		NoiseBreakdown: make(map[string]DimensionGroup),
	}

	if len(results) == 0 {
		return report
	}

	var (
		totalWER          float64
		totalCER          float64
		werCount          int
		validCmdCount     int
		validCmdPassed    int
		adversarialCount  int
		adversarialPassed int
		gateCount         int
		gatePassed        int

		ttftVals       []float64
		endpointVals   []float64
		secondPassVals []float64
		totalDurVals   []float64
	)

	type snrAgg struct {
		snrVal   *float64
		count    int
		passed   int
		werSum   float64
		werCount int
		cerSum   float64
		cmdCount int
		cmdPass  int
		latSum   float64
	}
	snrMap := make(map[string]*snrAgg)

	snrKey := func(snr *float64, noiseType string) string {
		if snr == nil || noiseType == "clean" || noiseType == "" {
			return "Clean"
		}
		return fmt.Sprintf("%+.0f dB", *snr)
	}

	type dimAgg struct {
		count    int
		passed   int
		werSum   float64
		cerSum   float64
		werCount int
	}
	voiceMap := make(map[string]*dimAgg)
	rateMap := make(map[string]*dimAgg)
	noiseMap := make(map[string]*dimAgg)

	for _, res := range results {
		if res.Passed {
			report.Passed++
		} else {
			report.Failed++
		}
		report.DroppedChunks += res.DroppedChunks

		totalDurVals = append(totalDurVals, float64(res.Duration.Milliseconds()))
		if res.TTFT > 0 {
			ttftVals = append(ttftVals, float64(res.TTFT.Milliseconds()))
		}
		if res.EndpointDelay > 0 {
			endpointVals = append(endpointVals, float64(res.EndpointDelay.Milliseconds()))
		}
		if res.SecondPassLatency > 0 {
			secondPassVals = append(secondPassVals, float64(res.SecondPassLatency.Milliseconds()))
		}

		if res.Class == "ACCEPT" {
			totalWER += res.WER
			totalCER += res.CER
			werCount++
		}

		if res.Mode == ModeCommand && res.Class == "ACCEPT" {
			validCmdCount++
			if res.Passed {
				validCmdPassed++
			}
		} else if res.Class == "REJECT" {
			adversarialCount++
			if res.Passed {
				adversarialPassed++
			}
		} else if res.Class == "SILENCE" {
			gateCount++
			if res.Passed {
				gatePassed++
			}
		}

		sKey := snrKey(res.SNR, res.NoiseType)
		sEntry, ok := snrMap[sKey]
		if !ok {
			sEntry = &snrAgg{snrVal: res.SNR}
			snrMap[sKey] = sEntry
		}
		sEntry.count++
		if res.Passed {
			sEntry.passed++
		}
		if res.Class == "ACCEPT" {
			sEntry.werSum += res.WER
			sEntry.cerSum += res.CER
			sEntry.werCount++
		}
		if res.Mode == ModeCommand && res.Class == "ACCEPT" {
			sEntry.cmdCount++
			if res.Passed {
				sEntry.cmdPass++
			}
		}
		sEntry.latSum += float64(res.Duration.Milliseconds())

		if res.Voice != "" {
			vEntry, exists := voiceMap[res.Voice]
			if !exists {
				vEntry = &dimAgg{}
				voiceMap[res.Voice] = vEntry
			}
			vEntry.count++
			if res.Passed {
				vEntry.passed++
			}
			if res.Class == "ACCEPT" {
				vEntry.werSum += res.WER
				vEntry.cerSum += res.CER
				vEntry.werCount++
			}
		}

		if res.SpeakingRate > 0 {
			rKey := fmt.Sprintf("%.1fx", res.SpeakingRate)
			rEntry, exists := rateMap[rKey]
			if !exists {
				rEntry = &dimAgg{}
				rateMap[rKey] = rEntry
			}
			rEntry.count++
			if res.Passed {
				rEntry.passed++
			}
			if res.Class == "ACCEPT" {
				rEntry.werSum += res.WER
				rEntry.cerSum += res.CER
				rEntry.werCount++
			}
		}

		nKey := res.NoiseType
		if nKey == "" {
			nKey = "clean"
		}
		nEntry, exists := noiseMap[nKey]
		if !exists {
			nEntry = &dimAgg{}
			noiseMap[nKey] = nEntry
		}
		nEntry.count++
		if res.Passed {
			nEntry.passed++
		}
		if res.Class == "ACCEPT" {
			nEntry.werSum += res.WER
			nEntry.cerSum += res.CER
			nEntry.werCount++
		}
	}

	report.PassRate = float64(report.Passed) / float64(report.TotalCases)
	if werCount > 0 {
		report.AvgWER = totalWER / float64(werCount)
		report.AvgCER = totalCER / float64(werCount)
	}
	report.IntentAccuracy = passRatio(validCmdPassed, validCmdCount)
	report.RejectionAccuracy = passRatio(adversarialPassed, adversarialCount)
	report.GatePrecision = passRatio(gatePassed, gateCount)

	report.LatencyStats = LatencyStatistics{
		TTFT:          ComputePercentiles(ttftVals),
		EndpointDelay: ComputePercentiles(endpointVals),
		SecondPass:    ComputePercentiles(secondPassVals),
		TotalDuration: ComputePercentiles(totalDurVals),
	}

	snrOrder := []string{"Clean", "+20 dB", "+15 dB", "+10 dB", "+5 dB", "+0 dB", "-5 dB"}
	for _, key := range snrOrder {
		agg, ok := snrMap[key]
		if !ok {
			continue
		}
		pt := SNRCurvePoint{
			SNR:          agg.snrVal,
			Label:        key,
			Count:        agg.count,
			Passed:       agg.passed,
			PassRate:     float64(agg.passed) / float64(agg.count),
			AvgLatencyMs: math.Round((agg.latSum/float64(agg.count))*10) / 10,
		}
		if agg.werCount > 0 {
			pt.AvgWER = math.Round((agg.werSum/float64(agg.werCount))*1000) / 1000
			pt.AvgCER = math.Round((agg.cerSum/float64(agg.werCount))*1000) / 1000
		}
		if agg.cmdCount > 0 {
			pt.IntentAccuracy = math.Round((float64(agg.cmdPass)/float64(agg.cmdCount))*1000) / 1000
		}
		report.SNRCurve = append(report.SNRCurve, pt)
		delete(snrMap, key)
	}

	for key, agg := range snrMap {
		pt := SNRCurvePoint{
			SNR:          agg.snrVal,
			Label:        key,
			Count:        agg.count,
			Passed:       agg.passed,
			PassRate:     float64(agg.passed) / float64(agg.count),
			AvgLatencyMs: math.Round((agg.latSum/float64(agg.count))*10) / 10,
		}
		if agg.werCount > 0 {
			pt.AvgWER = math.Round((agg.werSum/float64(agg.werCount))*1000) / 1000
			pt.AvgCER = math.Round((agg.cerSum/float64(agg.werCount))*1000) / 1000
		}
		report.SNRCurve = append(report.SNRCurve, pt)
	}

	for k, agg := range voiceMap {
		dg := DimensionGroup{
			Key:      k,
			Count:    agg.count,
			Passed:   agg.passed,
			PassRate: float64(agg.passed) / float64(agg.count),
		}
		if agg.werCount > 0 {
			dg.AvgWER = math.Round((agg.werSum/float64(agg.werCount))*1000) / 1000
			dg.AvgCER = math.Round((agg.cerSum/float64(agg.werCount))*1000) / 1000
		}
		report.VoiceBreakdown[k] = dg
	}

	for k, agg := range rateMap {
		dg := DimensionGroup{
			Key:      k,
			Count:    agg.count,
			Passed:   agg.passed,
			PassRate: float64(agg.passed) / float64(agg.count),
		}
		if agg.werCount > 0 {
			dg.AvgWER = math.Round((agg.werSum/float64(agg.werCount))*1000) / 1000
			dg.AvgCER = math.Round((agg.cerSum/float64(agg.werCount))*1000) / 1000
		}
		report.RateBreakdown[k] = dg
	}

	for k, agg := range noiseMap {
		dg := DimensionGroup{
			Key:      k,
			Count:    agg.count,
			Passed:   agg.passed,
			PassRate: float64(agg.passed) / float64(agg.count),
		}
		if agg.werCount > 0 {
			dg.AvgWER = math.Round((agg.werSum/float64(agg.werCount))*1000) / 1000
			dg.AvgCER = math.Round((agg.cerSum/float64(agg.werCount))*1000) / 1000
		}
		report.NoiseBreakdown[k] = dg
	}

	return report
}

func passRatio(passed, count int) *float64 {
	if count == 0 {
		return nil
	}
	r := float64(passed) / float64(count)
	return &r
}

// CalculateWER computes Word Error Rate between a reference string and hypothesis string.
func CalculateWER(reference, hypothesis string) float64 {
	refWords := normalizeAndSplitWords(reference)
	hypWords := normalizeAndSplitWords(hypothesis)

	if len(refWords) == 0 {
		if len(hypWords) == 0 {
			return 0.0
		}
		return 1.0
	}

	d := make([][]int, len(refWords)+1)
	for i := range d {
		d[i] = make([]int, len(hypWords)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}

	for i := 1; i <= len(refWords); i++ {
		for j := 1; j <= len(hypWords); j++ {
			cost := 1
			if refWords[i-1] == hypWords[j-1] {
				cost = 0
			}
			d[i][j] = min(
				d[i-1][j]+1,      // deletion
				d[i][j-1]+1,      // insertion
				d[i-1][j-1]+cost, // substitution
			)
		}
	}

	return float64(d[len(refWords)][len(hypWords)]) / float64(len(refWords))
}

// CalculateCER computes Character Error Rate between a reference string and hypothesis string.
func CalculateCER(reference, hypothesis string) float64 {
	refRunes := normalizeAndFilterRunes(reference)
	hypRunes := normalizeAndFilterRunes(hypothesis)

	if len(refRunes) == 0 {
		if len(hypRunes) == 0 {
			return 0.0
		}
		return 1.0
	}

	d := make([][]int, len(refRunes)+1)
	for i := range d {
		d[i] = make([]int, len(hypRunes)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}

	for i := 1; i <= len(refRunes); i++ {
		for j := 1; j <= len(hypRunes); j++ {
			cost := 1
			if refRunes[i-1] == hypRunes[j-1] {
				cost = 0
			}
			d[i][j] = min(
				d[i-1][j]+1,      // deletion
				d[i][j-1]+1,      // insertion
				d[i-1][j-1]+cost, // substitution
			)
		}
	}

	return float64(d[len(refRunes)][len(hypRunes)]) / float64(len(refRunes))
}

// IsCJK returns true if the rune is within Chinese, Japanese, or Korean script blocks.
func IsCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r)
}

func normalizeAndSplitWords(s string) []string {
	var clean strings.Builder
	for _, r := range strings.ToLower(s) {
		if IsCJK(r) {
			clean.WriteRune(' ')
			clean.WriteRune(r)
			clean.WriteRune(' ')
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			clean.WriteRune(r)
		}
	}
	return strings.Fields(clean.String())
}

func normalizeAndFilterRunes(s string) []rune {
	var out []rune
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out = append(out, r)
		}
	}
	return out
}
