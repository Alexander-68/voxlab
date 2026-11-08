---
type: "query"
date: "2026-11-08T13:35:33.300051+00:00"
question: "Why does SherpaRunner connect Sherpa-ONNX Runner Integration to 8 subsystems?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["SherpaRunner", "WarmASR", "WarmTTS", "ASRStream"]
---

# Q: Why does SherpaRunner connect Sherpa-ONNX Runner Integration to 8 subsystems?

## Answer

SherpaRunner in pkg/engine/sherpa_runner.go:L37 is the concrete engine bridging streaming Zipformer ASR, offline Parakeet ASR, and Kokoro neural TTS, linking audio streaming, warm daemons, and model discovery.

## Outcome

- Signal: useful

## Source Nodes

- SherpaRunner
- WarmASR
- WarmTTS
- ASRStream