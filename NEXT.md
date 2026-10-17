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

## Current Status (End of Session 2026-10-10)
- **Engine**: Resident in-memory streaming daemon (`sherpa-onnx-online-websocket-server`) running on 4 inference threads.
- **Audio Pipeline**: Real-time Go audio loop decoupled from ASR worker thread; continuous 30Hz metering.
- **Visualizer**: Optional Live Oscilloscope & Audio Meter toggle on frontend reduces CPU load to 0% when disabled.
- **Tests**: All repository unit and integration tests passing (`go test ./...`).
