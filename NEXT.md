# VoxLab - Next Session Roadmap

## Starting Topics

### 1. Test Updated Voice Recognition Flow
- [ ] **Live Partial Streaming**: Verify interim neural hypotheses stream smoothly every ~650ms during continuous speech in Web UI.
- [ ] **Natural Sentence Boundary Endpointing (~720ms)**: Test that natural speaking pauses (0.7–0.8s) endpoint sentences reliably, automatically append a period (`.`), and capitalize the next phrase without merging consecutive thoughts into one paragraph.
- [ ] **Typing Noise / Clack Rejection**: Confirm that ambient noise, breath, or keyboard typing cleanly resets back to the ready listening state (`Listening... speak to dictate` / `Ready`) without getting stuck in `(recognizing speech...)`.
- [ ] **Command vs. Dictation Modes**: Test snappy command execution (~600ms endpoint) vs. fluid dictation annotation.

---

### 2. Build Hotwords (Contextual Biasing) for Sherpa-ONNX
- [ ] **Create Hotwords Catalog (`data/hotwords.txt`)**:
  - Add domain keywords, brand terms, and technical phrases prone to phonetic misrecognition:
    ```text
    SHERPA
    VOXLAB
    API
    WEB SPEECH
    GOOD EVENING
    HOW ARE YOU TODAY
    ```
- [ ] **Integrate Hotwords into Sherpa Runner & Daemon**:
  - Connect `--hotwords-file=data/hotwords.txt` and `--hotwords-score=2.0` (configurable 1.5–2.5).
  - Enable `--decoding-method=modified_beam_search`.
  - Wire configuration options into [`pkg/config/config.go`](file:///C:/Alex/voxlab/pkg/config/config.go), [`pkg/engine/sherpa_runner.go`](file:///C:/Alex/voxlab/pkg/engine/sherpa_runner.go), and [`pkg/engine/warm_asr.go`](file:///C:/Alex/voxlab/pkg/engine/warm_asr.go).
- [ ] **Evaluate Accuracy Improvements**:
  - Test phonetic disambiguation:
    - *"scarpa"* / *"shtirpa"* $\rightarrow$ **"sherpa"**
    - *"web a pia"* $\rightarrow$ **"Web API"**
  - Add UI controls or config setting to inspect and reload active hotwords at runtime.

---

## Current Status (End of Session 2026-10-10)
- **Engine**: Resident in-memory streaming daemon (`sherpa-onnx-online-websocket-server`) running on 4 inference threads.
- **Audio Pipeline**: Real-time Go audio loop decoupled from ASR worker thread; continuous 30Hz metering.
- **Visualizer**: Optional Live Oscilloscope & Audio Meter toggle on frontend reduces CPU load to 0% when disabled.
- **Tests**: All repository unit and integration tests passing (`go test ./...`).
