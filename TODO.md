# VoxLab TODO

Open issues and findings from the first whole-codebase review (2026-11-07, at `0a8a33f`) and the fixes that followed.
IDs (C1, I6, ...) are the review's. Finished work is in the git log and is not repeated here.

## After the reboot

1. **Check free memory first.** Tier 2 plus its daemons need a few GB.
2. **Rerun Tier 2 and compare with the first run.** `go run ./cmd/eval_bench -tier 2 -out <file>`
   - Only one full run exists (`data/eval_report_tier2.json`, commit `7c3c29b`, 78/104). The second was killed for low system
     memory before finishing an item, so the bench's run-to-run repeatability is **still unconfirmed**.
   - That run predates `eec9ec0` and `1bc287c`. Those commits did not intend to change scoring, but the numbers are from older code.
3. **Log memory during the run** (see Memory below), so a leak can be confirmed or ruled out.
4. **Decide the language work below.** It needs the corpus fix first, or the effect cannot be measured.

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

### Measured effect of a hint

`sherpa-onnx.exe`, first pass only, no second pass, 4 clean Tier 2 clips per language, `--language=<locale>` against none. Lower is better.

| Language | Metric | Auto | Hinted |
| --- | --- | --- | --- |
| Spanish | WER | 0.585 | 0.619 |
| Portuguese | WER | 0.439 | 0.501 |
| French | WER | 0.883 | 0.852 |
| Italian | WER | 0.671 | 0.632 |
| Mandarin | CER | 0.423 | 0.423 |

The hint is live, since it changes the output, but on these clips it shows no clear gain: two better, two worse, one unchanged. Four clips is too few to
separate that from noise, and see the next section for why these clips are the wrong test anyway.

### The non-English corpus is probably invalid

The same model transcribes its own bundled real-speech samples almost perfectly (no reference text, judged by reading):

- `es.wav`: "No preguntes qué puede hacer tu país por ti..."
- `fr.wav`: "Ne vous demandez pas ce que votre pays peut faire pour vous..."
- `zh.wav`: "不要问你的国家能为你做什么..."

Our Kokoro clips come out like English phonemes reading foreign text: "Los signos vitales" becomes "Los neus Vairs", and "El sitio de la incisión" becomes
"El city of Lincion". This fits review item I17: Kokoro's `Lang` is never set and only en/zh lexicons load. Three samples are not proof, but until this is
resolved, **treat every non-English figure in `NEXT.md` as a corpus measurement, not an ASR measurement.** Mandarin is less clear, as zh has a lexicon, and
still scores 0.42 here.

### Next steps

- [ ] Listen to a few `base_dict_es_*` and `base_dict_fr_*` clips to confirm the mispronunciation. The new Listen toggle or Play button both work.
- [ ] Fix the corpus first: set Kokoro's language per voice, or use real recordings such as Common Voice.
      `GenerateTier2Corpus` reuses existing base WAVs, so they must be deleted to regenerate (I17).
- [ ] Re-measure hint against auto on the valid corpus, with more than 4 clips per language.
- [ ] Only if a hint helps: move streaming ASR in-process and wire the selector. If not, keep `auto` and make the selector browser-only.
- [ ] Try the same for Parakeet Ultra, and revisit the CJK guard in `refineLocked`.

## Memory

The Tier 2 rerun was killed by the harness for low system memory.

- **At that point:** no `voxlab`, `eval_bench` or `sherpa-onnx` process was left, including daemons, so they were not holding memory after the kill.
- **Biggest user:** Chrome, 49 processes with about 10 GB private, running since 2026-10-15. One process alone held 3.8 GB. RAM free was 2.8 of 15.6 GB.
- **So:** the evidence points at long-running Chrome more than at VoxLab, but this does not clear VoxLab. Nothing measured its memory during a run.
- [ ] During the next run, sample the working set of `eval_bench` and both daemons every 30 s. Flat means no leak; steady growth across items means a leak.
- [ ] If it grows, suspects are per-item state in the harness, `segAudio`, and the daemons themselves. Child processes are also never reaped (I10).

## Bench accuracy

- [ ] **Tier 2 test fails.** 75% is under its 80% threshold, which was deliberately not retuned. Revisit once the non-English corpus is valid.
- [ ] **I16** WER does not normalize numbers: "2 mg" against "two milligrams" fails `T2_CLEAN_DICT_MED_BF`, a false failure.
- [ ] **I12** Second-pass latency reads 0 for dictation. `latency_ms` from `transcript.refined` is ignored.
- [ ] **I14** SNR curve mixes noise types in one row, varies utterances per level, and computes SNR over padding too.
- [ ] **I15** Gate test is vacuous: clips measure below their labels (hiss -61 dBFS, not -48), and pass ignores partials. `Decision` records the expected class, not
      the outcome, and accuracy KPIs default to 1.0 with no data.
- [ ] **I17** `gen_eval_samples -tier 1` writes nothing, though `eval_bench` points users to it.
- [ ] **I18** The CLI writes `eval_report_tier2.json` but the server reads `eval_report_tier-2-deep-matrix.json`. The CLI always exits 0.
- [ ] **I19** No tests for reconnects or `pkg/config`.
- [ ] Add spoken negation cases to the corpus. §13 is covered at text level only, so ASR plus matcher are untested together.
- [ ] `res.Duration` includes real-time audio feed, so "Total Turnaround" and SNR-row latency now mostly reflect clip length.
- [ ] Isolated pipeline per run. During a run another tab's controls still reach the live pipeline (`ponytail:` note in `pinEvalPipeline`).

## Robustness

- [ ] **I2** `asrMu` is held across network I/O, and the second pass runs synchronously in the ASR worker, though the README says background. `activate` and `cancel` can
      stall for seconds, and the new `asrTurnMu` waits on the same work.
- [ ] **I5** Unsynchronized shared state: `cfg.KWS.*`, `cfg.Engine.Kokoro*`, `s.wavSource`, `s.httpServer`, and check-then-act guards in `StateMachine`. Run `-race` once a
      C toolchain is installed; it cannot run here.
- [ ] **I8** No `SetReadLimit`, `MaxBytesReader` or HTTP timeouts. `speak` spawns unbounded goroutines. `SwitchSource` accepts any string.
- [ ] **I10** No `cmd.Wait`, and daemon stderr is discarded, so failures are undiagnosable. Daemons outlive a crash (no job object).
- [ ] **I4 remainder** Disconnect cancels playback but not a listening turn (recipe §11). The TTS failsafe timer can unmute during later playback.
- [ ] LAN mode (`-host 0.0.0.0`) has no authentication, only a warning.
- [ ] `go vet` reports 5 `unsafe.Pointer` warnings, probably false positives on C-owned pointers.

## Features missing against README and plan

Confirm whether each is intended before building.

- [ ] **I20** Wake word is simulated; `set_kws_keyword` has no effect.
- [ ] **I21** Pre-roll ring buffer is written but never read.
- [ ] **I23** `Confirmation` and `Preconditions` are never enforced, and confirming after CLARIFY emits no intent.
- [ ] **I24** Spec'd timeouts are missing (5 s speech start, 30 s max duration, CONFIRM expiry). `Fault()` is never called.
- [ ] **I25** No host speaker output, no Silero VAD, one endpoint value for both modes.

## Install, UI and smaller items

- [ ] **I27** `download_models.sh` glob fails silently on Linux.
- [ ] **I28** A bind failure still prints "VoxLab is active".
- [ ] **I29** Sherpa daemons start even in simulator mode, about 1.4 GB.
- [ ] **I31** TTS and `eval.start` failures reach no client, leaving the UI stuck. Eval status is not restored on reload.
- [ ] **I32** `capture.isRemoteMode` does not exist. T1 Play URLs 404 for two items. `ws://` is hard-coded.
- [ ] New dictation clears the server draft, so unsaved text in the box is replaced by the first new phrase.
- [ ] Second pass may drop speech: in a probe, "Findings colon clear lungs" was refined to "colon clear lungs common". Seen once, not investigated.
- [ ] Formatter exists twice, in `dictation.go` and `bench-ui.js`, and must be kept in step. Chinese gets `。.` and word counts of 0 (`Tokenize` is ASCII-only).
- [ ] `ref/` still describes a 5-panel UI and Zipformer, and the plan's `0.0.0.0` bind.
- [ ] Remaining review minors: WAV parser edge cases, `Resample` has no anti-aliasing, `-port 8080` cannot override config, `go.mod` marks gorilla `// indirect`.
