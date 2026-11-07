# VoxLab TODO

Open issues and findings from the first whole-codebase review (2026-11-07, at `0a8a33f`) and the fixes that followed.
IDs (C1, I6, ...) are the review's. Finished work is in the git log and is not repeated here.

## Status (2026-11-07, second session)

- **Repeatability confirmed.** Two Tier 2 runs on the same corpus (`7c3c29b`, `c208fa9`) gave identical transcripts on all 104 items, 78/104 each.
- **No memory leak seen.** Sampled every 30 s across 8 minutes of a run: `eval_bench` 75 MB, streaming daemon 783 MB, second-pass daemon
  842 to 846 MB, all flat. The earlier kill was system memory pressure, not VoxLab.
- **The non-English corpus was invalid, now fixed (I17 of the language work).** Kokoro was never given a language, so es/pt/fr/it voices read
  their text as English. With `--kokoro-lang`, the same model transcribes a Spanish and a French sentence word for word. VoxLab now sets
  the language from the voice prefix (`e` es, `f` fr, `h` hi, `i` it, `p` pt-br) and drops the en/zh lexicons for those voices. `CMD_ES_EF` keeps
  `Lang: "en"`, as it is English read in a Spanish voice on purpose.
- **Tier 2 on the corrected corpus: 93/104 (89.4%)**, report in `data/eval_report_tier-2-deep-matrix.json`. The Tier 2 test's 80% bar now passes.
- **Corpus diff note:** three Mandarin perturbation clips changed although their bases did not. The committed ones used noise seeds one
  higher than the current generator (found by brute-force search), so they were stale. Regeneration is now idempotent (clips are
  decoded after writing). `T2_DICT_ZH_VITALS_XB_SNR15_PINK` passing is from that new noise, not the language fix.
  The sub-gate clips changed on purpose: they are now scaled to their labelled levels.

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
- [ ] `Resample` is anti-aliased since `4fe6c6d`, so regenerating the corpus changes every Kokoro clip (24 to 16 kHz). Re-baseline Tier 2 then.
- [ ] Fed at about 3x real time with 0.85 to 1.05 s pauses, the first pass lost whole phrases (2 of 12). Likely a faster-than-real-time
      artifact; check at real time before trusting short-pause results.
