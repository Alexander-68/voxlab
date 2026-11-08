---
type: "query"
date: "2026-11-08T13:35:33.028424+00:00"
question: "Should Speech Denoiser & Audio Generation be split?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["cDenoisedAudio", "eval_bench/main.go", "gen_samples/main.go"]
---

# Q: Should Speech Denoiser & Audio Generation be split?

## Answer

Yes. Cohesion is 0.127 because it mixes CGO denoiser bindings (GTCRN, DPDFNet) with sample generation scripts (cmd/gen_samples) and evaluation runners.

## Outcome

- Signal: useful

## Source Nodes

- cDenoisedAudio
- eval_bench/main.go
- gen_samples/main.go