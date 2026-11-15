/**
 * VoiceClient: WebSocket bridge for VoxLab IPC protocol.
 */
class VoiceClient {
  constructor(url) {
    this.url = url || `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}/ws`;
    this.ws = null;
    this.listeners = new Map();
    this.reconnectTimer = null;
  }

  connect() {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return;
    }

    try {
      const socket = new WebSocket(this.url);
      this.ws = socket;
      socket.binaryType = 'arraybuffer';

      // Handlers ignore events from a superseded socket.
      socket.onopen = () => {
        if (this.ws !== socket) return;
        this.emit('connection', { connected: true });
        console.log('[VoiceClient] Connected to VoxLab host daemon');
      };

      socket.onclose = () => {
        if (this.ws !== socket) return;
        this.emit('connection', { connected: false });
        console.warn('[VoiceClient] Disconnected. Reconnecting in 2s...');
        this.scheduleReconnect();
      };

      socket.onerror = (err) => {
        console.error('[VoiceClient] WebSocket error:', err);
      };

      socket.onmessage = (event) => {
        if (this.ws !== socket) return;
        if (typeof event.data === 'string') {
          try {
            const msg = JSON.parse(event.data);
            if (msg.event) {
              this.emit(msg.event, msg.data);
            }
          } catch (e) {
            console.error('[VoiceClient] Failed to parse JSON message:', e);
          }
        } else if (event.data instanceof ArrayBuffer) {
          this.emit('audio.chunk', event.data);
        }
      };
    } catch (e) {
      console.error('[VoiceClient] Connection attempt failed:', e);
      this.scheduleReconnect();
    }
  }

  scheduleReconnect() {
    if (this.reconnectTimer) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, 2000);
  }

  on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, []);
    }
    this.listeners.get(event).push(callback);
  }

  emit(event, data) {
    const list = this.listeners.get(event);
    if (!list) return;
    list.forEach((cb) => {
      try {
        cb(data);
      } catch (e) {
        console.error(`[VoiceClient] Listener for "${event}" threw:`, e);
      }
    });
  }

  sendAudioChunk(arrayBuffer) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(arrayBuffer);
    }
  }

  sendAction(action, payload = {}) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn('[VoiceClient] Cannot send action, WS not open:', action);
      return false;
    }
    try {
      this.ws.send(JSON.stringify({ ...payload, action }));
      return true;
    } catch (e) {
      console.error('[VoiceClient] Failed to send action:', action, e);
      return false;
    }
  }

  // High-level RPC methods
  activateCommand() {
    this.sendAction('activate', { mode: 'command' });
  }

  startDictation(targetId) {
    this.sendAction('activate', { mode: 'dictation', target_id: targetId });
  }

  // Ends dictation, keeping the last phrase that is still being recognized.
  stopDictation() {
    this.sendAction('stop');
  }

  cancelTurn() {
    this.sendAction('cancel');
  }

  confirmAction(approved) {
    this.sendAction('confirm', { approved });
  }

  speak(text, voice, speed = 1.0, streaming = true) {
    this.sendAction('speak', { payload: { text, voice, speed, streaming } });
  }

  setSource(sourceName) {
    this.sendAction('set_source', { source: sourceName });
  }

  setNoiseGate(thresholdDbfs) {
    this.sendAction('set_noise_gate', { threshold_dbfs: thresholdDbfs });
  }

  setAsrModel(model) {
    this.sendAction('set_asr_model', { model: model });
  }

  setSecondPass(enabled, mode) {
    this.sendAction('set_second_pass', { enabled: !!enabled, mode: mode || '' });
  }

  setGateHold(holdMs) {
    this.sendAction('set_gate_hold', { hold_ms: holdMs });
  }

  setGateFloor(floorDb) {
    this.sendAction('set_gate_floor', { floor_db: floorDb });
  }

  setKwsThreshold(threshold) {
    this.sendAction('set_kws_threshold', { threshold: threshold });
  }

  setKwsEnabled(enabled) {
    this.sendAction('set_kws_enable', { enabled: !!enabled });
  }

  setKwsKeyword(keyword) {
    this.sendAction('set_kws_keyword', { keyword: keyword });
  }

  setAgc(enabled) {
    this.sendAction('set_agc', { enabled: !!enabled });
  }

  setHighPass(enabled) {
    this.sendAction('set_highpass', { enabled: !!enabled });
  }

  setMonitor(enabled) {
    this.sendAction('set_monitor', { enabled: !!enabled });
  }

  sendPlaybackStatus(playing) {
    this.sendAction('playback_status', { playing });
  }

  setDraft(text) {
    this.sendAction('set_draft', { text });
  }

  saveDraft(text) {
    this.sendAction('save_draft', { text });
  }

  injectText(text, mode = 'command') {
    this.sendAction('inject_text', { text, mode });
  }

  setTtsModel(modelName) {
    this.sendAction('set_tts_model', { model: modelName });
  }

  setSpeechEnhancer(enabled, model = '') {
    this.sendAction('set_speech_enhancer', { enabled: !!enabled, model: model });
  }

  setAsrEngine(engine) {
    this.sendAction('set_asr_engine', { engine });
  }

  startEvaluation(tier) {
    this.sendAction('eval.start', { tier });
  }

  cancelEvaluation() {
    this.sendAction('eval.cancel');
  }
}
