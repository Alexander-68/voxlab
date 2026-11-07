# VoxLab - Next Session Roadmap

## Starting Topics

### 1. Test Updated Voice Recognition Flow
- [ ] **Live Partial Streaming**: Verify interim neural hypotheses stream smoothly every ~330ms during continuous speech in Web UI.
- [ ] **Natural Sentence Boundary Endpointing (~720ms)**: Test that natural speaking pauses (0.7–0.8s) endpoint sentences reliably, automatically append a period (`.`), and capitalize the next phrase without merging consecutive thoughts into one paragraph.
- [ ] **Typing Noise / Clack Rejection**: Confirm that ambient noise, breath, or keyboard typing cleanly resets back to the ready listening state (`Listening... speak to dictate` / `Ready`) without getting stuck in `(recognizing speech...)`.
- [ ] **Command vs. Dictation Modes**: Test snappy command execution (~600ms endpoint) vs. fluid dictation annotation.

---

### 2. Domain Term Correction (medical annotation) — planned
Goal: medical terms (drugs, conditions, acronyms) appear correctly spelled in dictation.

**Decision (2026-10-17): list-based post-correction of the final text, not sherpa hotwords.**
- Sherpa hotwords (`--hotwords-file`, `modified_beam_search`) were tested on Parakeet TDT v2: score ≤6 had no effect,
  score 10 forced the term but corrupted the rest of the phrase. Parakeet ships no `bpe.vocab`, so hotwords
  need a generated one. Biasing the Kroko first pass is pointless: the second pass overwrites its text.
- MedASR (medical-trained model) was considered and rejected.

**Plan**
- [ ] Term list `data/terms.txt` (user provides): one term per line in the desired spelling (`metoprolol`, `AFib`, `CHF`),
      optional known mishearings: `a fib => AFib`.
- [ ] After the second pass, match words/n-grams of the final text against the list (explicit mishearings first,
      then fuzzy spelling/phonetic similarity with a conservative threshold) and replace.
- [ ] Log each correction in the event log (`raw -> corrected`), like `transcript.refined`.
- [ ] Reload the list at runtime without a restart.
- [ ] Measure on recorded real dictation with the user's terms (false corrections matter more than missed ones).

---

## Current Status (Session 2026-11-07)
- **ASR Streaming Engine**: Upgraded to **Nemotron 3.5** (`nemotron-3.5-asr-streaming-0.6b-560ms-int8`), supporting 40+ multilingual languages and native PnC. Kroko Zipformer retained for lightweight embedded profiling with live UI switching.
- **ASR Second-Pass Engine**: Upgraded to **Parakeet Ultra** (`nemo-parakeet-tdt-0.6b-ultra-int8`).
- **Second-Pass Mode Policy**: Independent controls for Command mode (disabled by default for instant 0ms execution) and Dictation mode (enabled by default for full-utterance refinement).
- **Voice Input Language Selector**: Dynamic language selector (`auto`, `en-US`, `en-GB`, `es-ES`, `fr-FR`, `de-DE`, `it-IT`, `pt-BR`, `ja-JP`, `zh-CN`, `hi-IN`) wired to both Browser `SpeechRecognition.lang` and server ASR pipelines.
- **Synthetic Round-Trip Acoustic Evaluation (Tier 1 & Tier 2)**:
  - **Tier 1 Smoke Suite**: 5 golden test cases verifying zero-regression pipeline health (~17s).
  - **Tier 2 Deep Acoustic Matrix**: Full 66-clip acoustic benchmark matrix spanning 12+ Kokoro voices, British/American/Multilingual accents, speaking rates (0.8x - 1.3x), SNR degradation ladder (+20dB down to -5dB), mechanical typing clacks, room babble chatter, acoustic room reverb, and sub-gate noise suppression.
  - **Metrics Engine**: Automated computation of WER, CER, Intent Accuracy, Rejection Precision, Gate Precision, SNR Degradation Curves, and acoustic latency percentiles (TTFT, silence endpoint delay, 2nd-pass refinement overhead, and round-trip turnaround).
  - **CLI & Web UI Harness**: On-demand CLI runner (`cmd/eval_bench`) and interactive Web UI Dashboard (`🧪 Acoustic Benchmark` tab) with live progress streaming, KPI cards, SNR degradation table, latency waterfall percentiles, and sample audio playback.
- **Engine**: Resident in-memory streaming daemon (`sherpa-onnx-online-websocket-server`) and offline second pass (`sherpa-onnx-offline-websocket-server`) running on 4 inference threads.
- **Audio Pipeline**: Real-time Go audio loop decoupled from ASR worker thread; continuous 30Hz metering; high-quality linear resampler.
- **Tests**: 100% of repository tests passing (`go test ./pkg/...`), including the new `pkg/evaluation` suite.
