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
- **Voice Input Language Selector**: Dynamic language selector (`auto`, `en-US`, `en-GB`, `es-ES`, `fr-FR`, `de-DE`, `it-IT`, `pt-BR`, `zh-CN`, `hi-IN`) wired to Browser `SpeechRecognition.lang` only. The server stores the value but nothing reads it. Nemotron 3.5 does accept a per-stream language, but `sherpa-onnx-online-websocket-server` exposes no option for it; see `TODO.md`.
- **Synthetic Round-Trip Acoustic Evaluation (Tier 1 & Tier 2)**:
  - **Tier 1 Smoke Suite**: 5 golden test cases verifying zero-regression pipeline health (~25s). Runs on every `go test`; skips without models.
  - **Tier 2 Deep Acoustic Matrix**: On demand (`go run ./cmd/eval_bench -tier 2`, or `VOXLAB_TIER2=1 go test ./pkg/evaluation/ -timeout 40m`; about 10 minutes, audio is fed at real time). Full 104-clip acoustic benchmark matrix spanning 20+ Kokoro voices, British/American accents, and extended multilingual dictation suites across Spanish (`es-ES`), Brazilian Portuguese (`pt-BR`), French (`fr-FR`), Italian (`it-IT`), and Mandarin Chinese (`zh-CN` with 18 varied acoustic cases), speaking rates (0.8x - 1.3x), SNR degradation ladder (+20dB down to -5dB), mechanical typing clacks, room babble chatter, acoustic room reverb, and sub-gate noise suppression. (Japanese removed from supported matrix due to lack of native ASR dictionary in resident models).
  - **Measured Results** (Tier 2 at `c208fa9` plus this session's uncommitted fixes, so stamped `-dirty`; report and settings in `data/eval_report_tier-2-deep-matrix.json`): **93/104 passed (89.4%)**, above the Tier 2 test's 80% bar. 0 audio chunks dropped. Second pass costs about 350 ms per dictation segment.
    - A dictation item passes at an error rate of 0.35 or lower: CER for Chinese, WER otherwise. Items may list a written form (`alt_text`, e.g. "2 mg"), and the closer reference counts.
    - Run-to-run repeatability is confirmed: two runs on the old corpus gave identical transcripts on all 104 items.
    - The previous 78/104 used an invalid non-English corpus (Kokoro read es/pt/fr/it text as English), so it is not comparable.

    | Clean items | Passed | WER | CER |
    | --- | --- | --- | --- |
    | English commands (incl. Spanish-accented) | 12/12 | 0.00 | 0.00 |
    | English dictation | 4/4 | 0.11 | 0.05 |
    | Spanish (`es-ES`) | 4/4 | 0.06 | 0.02 |
    | Portuguese (`pt-BR`) | 4/4 | 0.06 | 0.03 |
    | Italian (`it-IT`) | 4/4 | 0.03 | 0.01 |
    | French (`fr-FR`) | 3/4 | 0.09 | 0.02 |
    | Mandarin (`zh-CN`) | 8/14 | 0.37 | 0.37 |

    - **Known limits on these numbers**: the corpus is synthetic Kokoro speech. Mandarin errors are mostly homophones, which points at Kokoro's tones as much as at the ASR (see `TODO.md`). "Total Turnaround" includes the real-time audio feed, so it mostly reflects clip length.
  - **Multilingual Tokenizer & Hallucination Guard**: Hanzi/CJK rune segmentation in `corpus.go` for standard Chinese WER & CER evaluation; CJK guard in `refineLocked` (`pkg/engine/sherpa_runner.go`) preventing English offline models from overwriting native streaming Hanzi transcripts.
  - **Metrics Engine**: Automated computation of WER, CER, Intent Accuracy, Rejection Precision, Gate Precision, SNR Degradation Curves, and acoustic latency percentiles (TTFT, silence endpoint delay, 2nd-pass refinement overhead, and round-trip turnaround).
  - **CLI & Web UI Harness**: On-demand CLI runner (`cmd/eval_bench`) and interactive Web UI Dashboard (`🧪 Acoustic Benchmark` tab) with live progress streaming, KPI cards, SNR degradation table, latency waterfall percentiles, and sample audio playback.
- **Engine**: Resident in-memory streaming daemon (`sherpa-onnx-online-websocket-server`) and offline second pass (`sherpa-onnx-offline-websocket-server`) running on 4 inference threads.
- **Audio Pipeline**: Real-time Go audio loop decoupled from ASR worker thread; continuous 30Hz metering; high-quality linear resampler.
- **Tests**: `go test -p 1 ./pkg/...` passes with models installed. Tier 2 and the multilingual test are on demand.
