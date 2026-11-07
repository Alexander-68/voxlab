# VoxLab TODO

Status, next-session topics and open issues. Review IDs (C1, I6, ...) are from the first whole-codebase review (2026-11-07, at `0a8a33f`).
Finished work is in the git log and is not repeated here.

## Current status (2026-11-07)

- **ASR Streaming Engine**: Upgraded to **Nemotron 3.5** (`nemotron-3.5-asr-streaming-0.6b-560ms-int8`), supporting 40+ multilingual languages and native PnC. Kroko Zipformer retained for lightweight embedded profiling with live UI switching.
- **ASR Second-Pass Engine**: Upgraded to **Parakeet Ultra** (`nemo-parakeet-tdt-0.6b-ultra-int8`).
- **Second-Pass Mode Policy**: Independent controls for Command mode (disabled by default for instant 0ms execution) and Dictation mode (enabled by default for full-utterance refinement).
- **Voice Input Language Selector**: Browser speech language selector (`en-US`, `en-GB`, `es-ES`, `fr-FR`, `de-DE`, `it-IT`, `pt-BR`, `zh-CN`, `hi-IN`) sets Browser `SpeechRecognition.lang` only. Server ASR always auto-detects. Nemotron 3.5 does accept a per-stream language, but `sherpa-onnx-online-websocket-server` exposes no option for it; see Language support below.
- **Synthetic Round-Trip Acoustic Evaluation (Tier 1 & Tier 2)**:
  - **Tier 1 Smoke Suite**: 5 golden test cases verifying zero-regression pipeline health (~25s). Runs on every `go test`; skips without models.
  - **Tier 2 Deep Acoustic Matrix**: On demand (`go run ./cmd/eval_bench -tier 2`, or `VOXLAB_TIER2=1 go test ./pkg/evaluation/ -timeout 40m`; about 10 minutes, audio is fed at real time). Full 104-clip acoustic benchmark matrix spanning 20+ Kokoro voices, British/American accents, and extended multilingual dictation suites across Spanish (`es-ES`), Brazilian Portuguese (`pt-BR`), French (`fr-FR`), Italian (`it-IT`), and Mandarin Chinese (`zh-CN` with 18 varied acoustic cases), speaking rates (0.8x - 1.3x), SNR degradation ladder (+20dB down to -5dB), mechanical typing clacks, room babble chatter, acoustic room reverb, and sub-gate noise suppression. (Japanese removed from supported matrix due to lack of native ASR dictionary in resident models).
  - **Measured Results** (Tier 2 re-baselined with anti-aliased sinc resampler; report and settings in `data/eval_report_tier-2-deep-matrix.json`): **92/104 passed (88.5%)**,
    above the Tier 2 test's 80% bar. 0 audio chunks dropped. Second pass costs about 331 ms per dictation segment.
    With anti-aliased sinc downsampling (24 to 16 kHz), French dictation now achieves 4/4 (100% pass, WER 0.06 / CER 0.01).
    Mandarin clean dictation is 7/14 passed (1 clip at CER 0.388 just missed the 0.35 threshold); Spanish-accented command at SNR 10dB (`T2_CMD_ES_EF_SNR10`) now passes.
    - A dictation item passes at an error rate of 0.35 or lower: CER for Chinese, WER otherwise. Items may list a written form (`alt_text`, e.g. "2 mg"), and the closer reference counts.
    - Run-to-run repeatability is confirmed: two runs on the old corpus gave identical transcripts on all 104 items.
    - The previous 78/104 used an invalid non-English corpus (Kokoro read es/pt/fr/it text as English), so it is not comparable.

    | Clean items | Passed | WER | CER |
    | --- | --- | --- | --- |
    | English commands (incl. Spanish-accented) | 12/12 | 0.00 | 0.00 |
    | English dictation | 4/4 | 0.11 | 0.05 |
    | Spanish (`es-ES`) | 4/4 | 0.08 | 0.03 |
    | Portuguese (`pt-BR`) | 4/4 | 0.06 | 0.03 |
    | Italian (`it-IT`) | 4/4 | 0.03 | 0.01 |
    | French (`fr-FR`) | 4/4 | 0.06 | 0.01 |
    | Mandarin (`zh-CN`) | 7/14 | 0.37 | 0.37 |

    - **Known limits on these numbers**: the corpus is synthetic Kokoro speech. Mandarin errors are mostly homophones, which points at Kokoro's tones as much as at the ASR (see Language support). "Total Turnaround" includes the real-time audio feed, so it mostly reflects clip length.
  - **Multilingual Tokenizer & Hallucination Guard**: Hanzi/CJK rune segmentation in `corpus.go` for standard Chinese WER & CER evaluation; CJK guard in `refineLocked` (`pkg/engine/sherpa_runner.go`) preventing English offline models from overwriting native streaming Hanzi transcripts.
  - **Metrics Engine**: Automated computation of WER, CER, Intent Accuracy, Rejection Precision, Gate Precision, SNR Degradation Curves, and acoustic latency percentiles (TTFT, silence endpoint delay, 2nd-pass refinement overhead, and round-trip turnaround).
  - **CLI & Web UI Harness**: On-demand CLI runner (`cmd/eval_bench`) and interactive Web UI Dashboard (`🧪 Acoustic Benchmark` tab) with live progress streaming, KPI cards, SNR degradation table, latency waterfall percentiles, and sample audio playback.
- **Engine**: Resident in-memory streaming daemon (`sherpa-onnx-online-websocket-server`) and offline second pass (`sherpa-onnx-offline-websocket-server`) running on 4 inference threads.
- **Audio Pipeline**: Real-time Go audio loop decoupled from ASR worker thread; continuous 30Hz metering; windowed-sinc resampler (anti-aliased when downsampling).
- **Tests**: `go test -p 1 ./pkg/...` passes with models installed. Tier 2 and the multilingual test are on demand.

### Findings from the review sessions

- **Anti-aliased corpus re-baseline complete.** After regenerating all 104 Tier-2 test audio samples using the Hann-windowed sinc resampler introduced in `4fe6c6d` (24 to 16 kHz downsampling), Tier 2 achieves 92/104 (88.5% pass rate), comfortably above the 80% bar with 0 dropped audio chunks. French clean dictation reached 4/4 (100% pass); overall Intent Accuracy is 88.2%, Rejection Precision 100%, and Gate Precision 100%.
- **Repeatability confirmed.** Two Tier 2 runs on the same corpus (`7c3c29b`, `c208fa9`) gave identical transcripts on all 104 items, 78/104 each.
- **No memory leak seen.** Sampled every 30 s across 8 minutes of a run: `eval_bench` 75 MB, streaming daemon 783 MB, second-pass daemon
  842 to 846 MB, all flat. The earlier kill was system memory pressure, not VoxLab.
- **The non-English corpus was invalid, now fixed (I17 of the language work).** Kokoro was never given a language, so es/pt/fr/it voices read
  their text as English. With `--kokoro-lang`, the same model transcribes a Spanish and a French sentence word for word. VoxLab now sets
  the language from the voice prefix (`e` es, `f` fr, `h` hi, `i` it, `p` pt-br) and drops the en/zh lexicons for those voices. `CMD_ES_EF` keeps
  `Lang: "en"`, as it is English read in a Spanish voice on purpose.
- **Corpus diff note:** three Mandarin perturbation clips changed although their bases did not. The committed ones used noise seeds one
  higher than the current generator (found by brute-force search), so they were stale. Regeneration is now idempotent (clips are
  decoded after writing). `T2_DICT_ZH_VITALS_XB_SNR15_PINK` passing is from that new noise, not the language fix.
  The sub-gate clips changed on purpose: they are now scaled to their labelled levels.

## Next session

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

## Language support

### Findings

| Model | Role | Language input? |
| --- | --- | --- |
| Nemotron 3.5 streaming | First pass (default) | **Yes.** The encoder has a `prompt_index` input and `prompt_dictionary` metadata; `auto` = 101. Its README says to use per-stream language strings. |
| Parakeet Ultra | Second pass | **Unverified.** `tokens.txt` has `<\|fr\|>`, `<\|predict_lang\|>` and similar, and `SherpaOnnxOfflineStreamSetOption` exists, but nothing has been tried. |
| Nemotron Speech EN | First pass (alternative) | No. English only, no language tokens. |
| Kroko Zipformer | First pass (alternative) | No. English only. |

- **Codes:** use full locales. Bare codes can map elsewhere: `es` is `es-US` (3), while `es-ES` is 2; `pt` is `pt-PT` (13), while `pt-BR` is 12.
  There is no bare `zh`, only `zh-CN` (4) and `zh-TW` (5). `ja` also has no bare form. These ID mappings are read from the encoder metadata.
- **Selector values:** `en-US`, `en-GB`, `es-ES`, `fr-FR`, `de-DE`, `it-IT`, `pt-BR`, `zh-CN` and `hi-IN` are all in the map. `auto` is too.
- **The blocker:** our path is `sherpa-onnx-online-websocket-server`, whose flags include no language option. `--language` exists only on the file-based
  `sherpa-onnx.exe`. Whether the websocket protocol can set it is unverified; nothing in `--help` suggests so.
- **Likely route:** `SherpaOnnxOnlineStreamSetOption` is exported by `bin/sherpa-onnx-c-api.dll`, which we already load for TTS. That means running the
  streaming recognizer in-process instead of through the websocket daemon. It is a sizeable change: new C structs, and the 1.13.8 version coupling noted in I9.

### Measured effect of a hint (valid corpus)

`sherpa-onnx.exe`, first pass only, no second pass. Each language's Tier 2 dictation texts in every installed voice of that language, clean and
with pink noise at +5 dB. WER, CER for Mandarin; lower is better. "Better/worse" counts clips.

| Language | Clips | Clean auto | Clean hinted | +5 dB auto | +5 dB hinted | Better / worse |
| --- | --- | --- | --- | --- | --- | --- |
| Spanish | 12 + 12 | 0.131 | 0.131 | 0.255 | 0.222 | 4 / 2 |
| Portuguese | 12 + 12 | 0.010 | 0.035 | 0.079 | 0.083 | 1 / 3 |
| Italian | 8 + 8 | 0.125 | 0.125 | 0.148 | 0.148 | 0 / 0 |
| French | 4 + 4 | 0.056 | 0.102 | 0.164 | 0.164 | 0 / 1 |
| Mandarin | 28 + 28 | 0.324 | 0.321 | 0.538 | 0.478 | 14 / 6 |

- Clean speech gains nothing. Portuguese and French get slightly worse, by one or two clips.
- In noise the hint helps Mandarin clearly (CER -0.06, 9 clips better, 4 worse) and Spanish a little. Elsewhere it changes nothing.
- French has one voice, so 8 clips; treat it as a hint only.

### Next steps

- **Decided:** keep server ASR on `auto`. The selector now sets only the browser recognizer; `set_language` and `input_language` are gone.
  Revisit in-process streaming ASR if noisy Mandarin matters.
- [ ] Mandarin stays at 8/14 clean (CER 0.37), while the model reads its own `zh.wav` almost perfectly. Errors are homophones
      ("咪达唑仑两毫克" heard as "米打左轮量好课"), so suspect Kokoro's Mandarin tones before the ASR. Check by ear or with a real recording.
- [ ] Try the same for Parakeet Ultra, and revisit the CJK guard in `refineLocked`.

## Bench accuracy

- [ ] **I14** SNR curve mixes noise types in one row, varies utterances per level, and computes SNR over padding too.
- [ ] **I19** No tests for reconnects or `pkg/config`.
- [ ] Add spoken negation cases to the corpus. §13 is covered at text level only, so ASR plus matcher are untested together.
- [ ] `res.Duration` includes real-time audio feed, so "Total Turnaround" and SNR-row latency now mostly reflect clip length.
- [ ] Isolated pipeline per run. During a run another tab's controls still reach the live pipeline (`ponytail:` note in `pinEvalPipeline`).
      Seen: TTS playing in an open page echo-mutes the mic, so a Tier 1 item heard silence and failed.
- [ ] Second-pass latency is now measured (about 350 ms per segment). It runs inside the ASR worker (I2), so it adds to every dictation turn.

## Robustness

- [ ] **I2** `asrMu` is held across network I/O, and the second pass runs synchronously in the ASR worker, though the README says background. `activate` and `cancel` can
      stall for seconds, and the new `asrTurnMu` waits on the same work.
- [ ] **I5** Unsynchronized shared state: `cfg.KWS.*`, `cfg.Engine.Kokoro*`, `s.wavSource`, `s.httpServer`, and check-then-act guards in `StateMachine`. Run `-race` once a
      C toolchain is installed; it cannot run here.
- [ ] **I4 remainder** Disconnect cancels playback but not a listening turn (recipe §11). The TTS failsafe timer can unmute during later playback.
- [ ] LAN mode (`-host 0.0.0.0`) has no authentication, only a warning.
- [ ] Daemons are reaped and killed with VoxLab through a job object on Windows only. Elsewhere a crash still orphans them.

## Features missing against README and plan

Confirm whether each is intended before building.

- [ ] **I20** Wake word is simulated; `set_kws_keyword` has no effect.
- [ ] **I21** Pre-roll ring buffer is written but never read.
- [ ] **I23** `Confirmation` and `Preconditions` are never enforced, and confirming after CLARIFY emits no intent.
- [ ] **I24** Spec'd timeouts are missing (5 s speech start, 30 s max duration, CONFIRM expiry). `Fault()` is never called.
- [ ] **I25** No host speaker output, no Silero VAD, one endpoint value for both modes.

## Install, UI and smaller items

- [ ] **I27** `download_models.sh` glob fails silently on Linux.
- [ ] **I29** Sherpa daemons start even in simulator mode, about 1.4 GB.
- [ ] New dictation clears the server draft, so unsaved text in the box is replaced by the first new phrase.
- [ ] Second pass may drop speech: once, "Findings colon clear lungs" was refined to "colon clear lungs common". Not reproduced in 36
      Kokoro segments. Suspected cause, now fixed: the segment cut dropped the end pad, so a close next phrase lost its start pad, and the
      0.6 s start pad was shorter than the measured first-token lag (up to 0.76 s). Pads are now 0.9 s / 0.3 s. Close if it does not recur.
- [x] `Resample` is anti-aliased since `4fe6c6d`, so regenerating the corpus changes every Kokoro clip (24 to 16 kHz). Re-baseline Tier 2 then.
- [ ] Fed at about 3x real time with 0.85 to 1.05 s pauses, the first pass lost whole phrases (2 of 12). Likely a faster-than-real-time
      artifact; check at real time before trusting short-pause results.
