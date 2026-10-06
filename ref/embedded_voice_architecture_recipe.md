# Embedded Voice Pipeline Architecture for Custom Linux & WPE WebKit

A complete engineering specification for building an offline, low-latency, chatter-immune voice input (STT + Intent Routing) and voice output (TTS) system on embedded Linux running WPE WebKit.

---

## 1. System Architecture Overview

Attempting to run real-time neural speech models directly inside the browser runtime (via WebAssembly or WebGPU) introduces high memory consumption, garbage-collection pauses, and lack of hardware-specific optimizations (ARM NEON/NPU). 

This recipe uses the **Native Daemon Bridge Pattern**: native C++/Python daemons directly access the host audio layer and execute optimized ONNX runtimes. The WPE WebKit frontend communicates with this daemon over a high-throughput, low-latency loopback WebSocket or Unix Domain Socket.

```
 ┌────────────────────────────────────────────────────────────────────────┐
 │                              AUDIO HARDWARE                            │
 └──────────────┬──────────────────────────────────────────▲──────────────┘
                │ Raw Mic Input (16kHz Mono)               │ Audio Sink (ALSA/PipeWire)
                ▼                                          │
 ┌─────────────────────────────────────────────────────────┴──────────────┐
 │                      NATIVE HOST VOICE DAEMON                          │
 │                                                                        │
 │  [Stage 1: Energy & Silero VAD]                                        │
 │         │                                                              │
 │         ▼                                                              │
 │  [Stage 2: Wake-Word Detector] (openWakeWord / Sherpa-KWS)             │
 │         │ (Triggered)                                                  │
 │         ▼                                                              │
 │  [Stage 3: Streaming ASR] (Sherpa-ONNX Zipformer)                      │
 │         │ (Final Phrase)                                               │
 │         ▼                                                              │
 │  [Stage 4: Semantic Intent Router] ◄── Commands & Annotations Catalog  │
 │         │                             (MiniLM / Cosine Scoring)        │
 │         ▼                                       ▲                      │
 │     JSON Intent Event                           │ TTS Request Event    │
 └─────────┬───────────────────────────────────────┴──────────────────────┘
           │                                       ▲
           │  WebSocket / Unix Socket (ws://127.0.0.1:9001)
           ▼                                       │
 ┌─────────────────────────────────────────────────┴──────────────────────┐
 │                        WPE WEBKIT CLIENT                               │
 │                                                                        │
 │   - Receives clean action payloads (e.g. `NAV_SETTINGS`, score: 0.92)  │
 │   - Drives Web UI, displays confidence tags, and updates state         │
 │   - Dispatches speech synthesis events to the daemon                   │
 └────────────────────────────────────────────────────────────────────────┘
```

---

## 2. OS & Build Configuration (Yocto / Buildroot)

### Audio Server: PipeWire vs. Direct ALSA
* **Embedded Recommendation:** Use **PipeWire** (with `wireplumber`). PipeWire prevents device-locking collisions between the background recording daemon and system sounds or WPE WebKit playback.
* **Direct ALSA Alternative:** If using raw ALSA without a sound server, configure an `asound.conf` using the `dsnoop` (mic multiplexing) and `dmix` (speaker mixing) plugins.

### WPE WebKit Build Options (CMake)
Ensure your custom Linux image compiles WPE WebKit (`wpewebkit`) with media flags configured for minimal overhead:

```cmake
# CMake flags for WPE WebKit
-DENABLE_WEB_AUDIO=ON               # Needed if WPE plays UI chimes
-DENABLE_MEDIA_STREAM=OFF           # Not needed; native daemon records mic
-DENABLE_SPEECH_SYNTHESIS=OFF       # Bypass broken Flite/Spiel native paths
-DUSE_GSTREAMER_HOLEPUNCH=ON        # Optimize hardware video overlays
-DENABLE_ACCELERATED_2D_CANVAS=ON   # GPU UI acceleration
```

---

## 3. Voice Input Pipeline: Chatter-Immune Architecture

To ensure room chatter does not trigger false system actions, the input pipeline passes audio through four sequential filters:

### Stage 1: Energy Gating & Voice Activity Detection (VAD)
1. **Root-Mean-Square (RMS) Floor:** Audio below a calibrated noise floor (e.g., $<-42\text{ dBFS}$) is immediately dropped to ignore distant chatter.
2. **Silero VAD (ONNX):** Evaluates 30 ms chunks. Returns voice probability $P_{\text{voice}} \in [0.0, 1.0]$. Only passes chunks where $P_{\text{voice}} \ge 0.70$.

### Stage 2: Activation (Wake Word Engine)
* **Engine:** `openWakeWord` or `sherpa-onnx-kws`.
* **Execution:** Runs continuously against the VAD-cleared audio buffer ($< 3\%\text{ CPU}$ on ARM Cortex-A53).
* **Behavior:** When the target phrase (e.g., *"Hey Console"*) is verified, opens an ASR capture window for $T_{\text{active}} = 4.0\text{ seconds}$ or until $800\text{ ms}$ of trailing silence is detected.

### Stage 3: Streaming ASR (Sherpa-ONNX)
* **Model:** Streaming Zipformer-RNN-T (pre-trained, 16 kHz).
* **Execution:** Decodes incrementally in realtime. Sends interim transcriptions to WPE if live dictation feedback is desired, or holds until endpointing triggers a final transcript.

### Stage 4: Semantic Intent Router & Distractor Suppression
Instead of naive string matching, use a dual-tier matching strategy against your command catalog and annotations:

1. **Pre-compute Command Vectors:** Encode every canonical command and human annotation offline using a tiny sentence transformer (`all-MiniLM-L6-v2` or `bge-small-en` exported to ONNX). Store as an $N \times D$ matrix.
2. **Distractor/Out-of-Domain (OOD) Anchors:** Include common conversational embeddings in the matrix (e.g., *"what are you doing"*, *"great weather"*, *"let us order food"*).
3. **Runtime Vector Scoring:**
   Compute cosine similarity between the transcribed speech vector $\mathbf{u}$ and command vectors $\mathbf{v}_i$:
   $$\text{score}_i = \frac{\mathbf{u} \cdot \mathbf{v}_i}{\|\mathbf{u}\|_2 \|\mathbf{v}_i\|_2}$$
4. **Decision Boundary:**
   * **Accept & Execute:** $\text{score}_{\text{top1}} \ge 0.80$ AND $(\text{score}_{\text{top1}} - \text{score}_{\text{top2}}) \ge 0.12$.
   * **Disambiguation Prompt:** $0.65 \le \text{score}_{\text{top1}} < 0.80$ (UI prompts user).
   * **Chatter Rejection:** $\text{score}_{\text{top1}} < 0.65$ or closest match is an OOD distractor. Drop quietly.

---

## 4. Voice Output Pipeline (TTS)

* **Engine:** **Piper TTS** (VITS architecture via ONNX Runtime).
* **Advantages:** Human-quality phoneme synthesis, runs at $> 10\times$ faster than real-time on ARM64, and individual voice models require only $\sim30\text{ MB}$ to $\sim60\text{ MB}$ of storage.
* **Routing:** When the WPE frontend requests speech output, the daemon renders the PCM waveform directly to the default ALSA/PipeWire sink.

---

## 5. IPC Protocol Specification (WebSocket)

The daemon runs an asynchronous loopback server on `ws://127.0.0.1:9001`.

### A. Events Dispatched to WPE WebKit

#### Wake-Word Triggered:
```json
{
  "event": "wake_word_detected",
  "data": { "phrase": "hey_console", "confidence": 0.94 }
}
```

#### Final Intent Decoded:
```json
{
  "event": "intent_matched",
  "data": {
    "intent_id": "DISPLAY_SET_BRIGHTNESS",
    "target": "display",
    "action": "decrease",
    "confidence": 0.89,
    "raw_transcript": "dim the screen a little bit"
  }
}
```

#### Ambient Chatter Ignored (Debug Mode):
```json
{
  "event": "utterance_rejected",
  "data": {
    "reason": "OUT_OF_DOMAIN",
    "confidence": 0.42,
    "raw_transcript": "so I told him to go to the store"
  }
}
```

### B. Events Sent by WPE WebKit to Daemon

#### Speak Text (TTS):
```json
{
  "action": "speak",
  "payload": {
    "text": "Settings updated successfully.",
    "voice": "en_US-lessac-medium",
    "interrupt": true
  }
}
```

---

## 6. Reference Implementation

### Host Voice Daemon (`voice_daemon.py`)

```python
#!/usr/bin/env python3
import asyncio
import json
import numpy as np
import websockets
from numpy.linalg import norm

# Configuration and Thresholds
HOST = "127.0.0.1"
PORT = 9001
CONFIDENCE_THRESHOLD = 0.80
MARGIN_THRESHOLD = 0.12

# Command Registry with Annotations
COMMAND_CATALOG = [
    {
        "id": "NAV_SETTINGS",
        "phrases": ["open settings", "go to preferences", "show configuration", "system setup"]
    },
    {
        "id": "MEDIA_NEXT",
        "phrases": ["next song", "skip track", "play next", "skip forward"]
    },
    {
        "id": "DISPLAY_DARK",
        "phrases": ["dim screen", "lower brightness", "make it darker", "turn down lights"]
    },
    {
        "id": "__DISTRACTOR__",
        "phrases": ["how are you doing", "what are you eating", "see you later", "let's go outside"]
    }
]

class MockIntentScorer:
    """
    Simulates local ONNX sentence transformer embedding & cosine scoring.
    In production, substitute with onnxruntime running all-MiniLM-L6-v2.
    """
    def __init__(self, catalog):
        self.catalog = catalog

    def score(self, text: str):
        text_lower = text.lower()
        best_id = None
        best_score = 0.0
        second_score = 0.0

        for item in self.catalog:
            for phrase in item["phrases"]:
                # Simple token overlap proxy for demonstration
                tokens = set(phrase.split())
                input_tokens = set(text_lower.split())
                overlap = len(tokens & input_tokens)
                sim = overlap / max(len(tokens | input_tokens), 1)

                if sim > best_score:
                    second_score = best_score
                    best_score = sim
                    best_id = item["id"]
                elif sim > second_score:
                    second_score = sim

        return best_id, best_score, (best_score - second_score)

class VoiceDaemon:
    def __init__(self):
        self.clients = set()
        self.scorer = MockIntentScorer(COMMAND_CATALOG)

    async def register(self, websocket):
        self.clients.add(websocket)
        try:
            async for message in websocket:
                data = json.loads(message)
                if data.get("action") == "speak":
                    await self.handle_tts(data["payload"])
        finally:
            self.clients.remove(websocket)

    async def broadcast(self, payload: dict):
        if self.clients:
            msg = json.dumps(payload)
            await asyncio.gather(*[client.send(msg) for client in self.clients])

    async def handle_tts(self, payload: dict):
        text = payload.get("text", "")
        print(f"[TTS Pipeline] Synthesizing via Piper: '{text}'")
        # System call invocation for Piper TTS:
        # echo "$text" | piper --model voice.onnx --output-raw | aplay -r 22050 -f S16_LE -t raw
        await asyncio.sleep(0.05)

    async def simulate_asr_event(self, phrase: str):
        """Simulates incoming speech recognized by Sherpa-ONNX"""
        intent_id, score, margin = self.scorer.score(phrase)

        if intent_id == "__DISTRACTOR__" or score < CONFIDENCE_THRESHOLD or margin < MARGIN_THRESHOLD:
            await self.broadcast({
                "event": "utterance_rejected",
                "data": { "raw": phrase, "confidence": score, "reason": "CHATTER_OR_AMBIGUOUS" }
            })
            return

        await self.broadcast({
            "event": "intent_matched",
            "data": { "intent_id": intent_id, "confidence": score, "raw": phrase }
        })

async def main():
    daemon = VoiceDaemon()
    server = await websockets.serve(daemon.register, HOST, PORT)
    print(f"Voice daemon operational on ws://{HOST}:{PORT}")
    
    # Run server forever
    await server.wait_closed()

if __name__ == "__main__":
    asyncio.run(main())
```

---

### WPE WebKit JavaScript Integration (`app.js`)

```javascript
// Connect to the local loopback voice daemon
const voiceSocket = new WebSocket('ws://127.0.0.1:9001');

voiceSocket.onopen = () => {
  console.log('[Voice Bridge] Connected to voice daemon');
};

voiceSocket.onmessage = (event) => {
  const message = JSON.parse(event.data);

  switch (message.event) {
    case 'wake_word_detected':
      displayVoiceStateIndicator('LISTENING');
      break;

    case 'intent_matched':
      displayVoiceStateIndicator('ACTIVE');
      executeIntent(message.data.intent_id, message.data.raw);
      break;

    case 'utterance_rejected':
      displayVoiceStateIndicator('IDLE');
      console.log(`[Noise Filter] Ignored speech: "${message.data.raw}"`);
      break;

    default:
      break;
  }
};

function executeIntent(intentId, transcript) {
  console.log(`Executing ${intentId} from prompt: "${transcript}"`);

  if (intentId === 'NAV_SETTINGS') {
    window.location.hash = '#/settings';
    speakReply("Opening system settings.");
  } else if (intentId === 'DISPLAY_DARK') {
    document.body.classList.add('dark-mode');
    speakReply("Display brightness lowered.");
  }
}

function speakReply(textToSpeak) {
  voiceSocket.send(JSON.stringify({
    action: 'speak',
    payload: {
      text: textToSpeak,
      voice: 'en_US-lessac-medium'
    }
  }));
}

function displayVoiceStateIndicator(state) {
  const el = document.getElementById('voice-status');
  if (el) el.innerText = state;
}
```

---

## 7. Performance Benchmarks & Edge Tuning

| Pipeline Segment | Target Execution Latency (ARM64 / 4-core) | Primary Strategy |
| :--- | :--- | :--- |
| **VAD Gating** | $< 2\text{ ms}$ per chunk | Silero VAD (ONNX quantized) |
| **Wake-Word Spotting** | $15\text{–}25\text{ ms}$ continuous evaluation | openWakeWord (16 kHz mono) |
| **ASR Output (Sherpa)** | $< 180\text{ ms}$ token delay | Zipformer Streaming RNN-T |
| **Semantic Routing** | $< 15\text{ ms}$ post-transcription | Cosine matrix dot product |
| **IPC Roundtrip** | $< 0.5\text{ ms}$ | Loopback WebSocket (`127.0.0.1`) |
| **TTS Generation (Piper)**| $< 120\text{ ms}$ time-to-first-audio | VITS ONNX model (FP16/NEON) |

### Hardening Checklist
1. **CPU Pinning (cgroups / `taskset`):** Pin WPE WebKit's `WPENetworkProcess` and `WPEWebProcess` to cores 0–2. Bind the Voice Daemon to core 3 to guarantee audio buffers never under-run during heavy DOM rendering.
2. **Audio Buffer Tuning:** Set PipeWire quantum to 512 samples ($32\text{ ms}$ at 16 kHz) to prevent audio capture lag without causing CPU interrupt thrashing.
3. **Model Quantization:** Convert all ONNX models (ASR, VAD, Embeddings, TTS) to **INT8** or **FP16** to keep total resident memory footprint below $200\text{ MB}$.