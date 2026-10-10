//go:build !windows

package engine

import "fmt"

type stubWarmTTS struct{}

func (s *stubWarmTTS) Synthesize(text string, sid int, speed float64) (*TTSResult, error) {
	return nil, fmt.Errorf("in-process warm TTS is not supported on this platform")
}

func (s *stubWarmTTS) SynthesizeStream(text string, sid int, speed float64, onChunk func(chunk TTSChunk) error) (*TTSResult, error) {
	return nil, fmt.Errorf("in-process warm TTS is not supported on this platform")
}

func (s *stubWarmTTS) IsWarm() bool {
	return false
}

func (s *stubWarmTTS) ModelKey() string {
	return ""
}

func (s *stubWarmTTS) Close() error {
	return nil
}

// newPlatformWarmTTS returns nil on non-Windows platforms.
func newPlatformWarmTTS(dllDir string, modelFile string, voicesPath string, tokensPath string, dataDirPath string, lexPath string, ruleFsts string, numThreads int, provider string) (WarmTTS, error) {
	return nil, fmt.Errorf("in-process warm TTS is not supported on this platform")
}
