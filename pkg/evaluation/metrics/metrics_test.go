package metrics

import (
	"testing"
	"time"
)

func TestCalculateWER(t *testing.T) {
	tests := []struct {
		ref  string
		hyp  string
		want float64
	}{
		{"", "", 0.0},
		{"hello", "", 1.0},
		{"", "hello", 1.0},
		{"hello world", "hello world", 0.0},
		{"hello world", "hello there world", 0.5},
		{"open settings", "open the settings", 0.5},
		{"start recording now", "start recording", 1.0 / 3.0},
	}

	for _, tt := range tests {
		got := CalculateWER(tt.ref, tt.hyp)
		diff := got - tt.want
		if diff < -0.01 || diff > 0.01 {
			t.Errorf("CalculateWER(%q, %q) = %f, want %f", tt.ref, tt.hyp, got, tt.want)
		}
	}
}

func TestCalculateCER(t *testing.T) {
	tests := []struct {
		ref  string
		hyp  string
		want float64
	}{
		{"你好世界", "你好世界", 0.0},
		{"你好世界", "你好", 0.5},
		{"你好", "你好世界", 1.0},
	}

	for _, tt := range tests {
		got := CalculateCER(tt.ref, tt.hyp)
		diff := got - tt.want
		if diff < -0.01 || diff > 0.01 {
			t.Errorf("CalculateCER(%q, %q) = %f, want %f", tt.ref, tt.hyp, got, tt.want)
		}
	}
}

func TestComputePercentiles(t *testing.T) {
	vals := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	p := ComputePercentiles(vals)
	if p.MinMs != 10 || p.MaxMs != 100 || p.AvgMs != 55 {
		t.Errorf("unexpected percentiles: %+v", p)
	}
	if p.P50Ms < 45 || p.P50Ms > 65 {
		t.Errorf("unexpected P50: %f", p.P50Ms)
	}
}

func TestComputeSuiteReport(t *testing.T) {
	snr := 10.0
	results := []ItemResult{
		{
			ID:           "test1",
			Mode:         "command",
			Class:        "ACCEPT",
			Passed:       true,
			Duration:     100 * time.Millisecond,
			TTFT:         50 * time.Millisecond,
			WER:          0.0,
			Voice:        "af_heart",
			SpeakingRate: 1.0,
			SNR:          &snr,
			NoiseType:    "babble",
		},
		{
			ID:           "test2",
			Mode:         "command",
			Class:        "ACCEPT",
			Passed:       false,
			Duration:     200 * time.Millisecond,
			TTFT:         80 * time.Millisecond,
			WER:          0.5,
			Voice:        "af_heart",
			SpeakingRate: 1.0,
			SNR:          &snr,
			NoiseType:    "babble",
		},
	}

	report := ComputeSuiteReport("tier-1-smoke", results, 300*time.Millisecond)
	if report.TotalCases != 2 || report.Passed != 1 || report.Failed != 1 {
		t.Errorf("unexpected report summary: %+v", report)
	}
	if report.PassRate != 0.5 {
		t.Errorf("expected pass rate 0.5, got %f", report.PassRate)
	}
	if len(report.SNRCurve) != 1 {
		t.Errorf("expected 1 SNR point, got %d", len(report.SNRCurve))
	}
}
