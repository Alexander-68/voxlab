/**
 * VoiceClient: WebSocket bridge for VoxLab IPC protocol.
 */
class VoiceClient {
  constructor(url) {
    this.url = url || `ws://${window.location.host}/ws`;
    this.ws = null;
    this.listeners = new Map();
    this.reconnectTimer = null;
    this.isConnected = false;
  }

  connect() {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return;
    }

    try {
      this.ws = new WebSocket(this.url);
      this.ws.binaryType = 'arraybuffer';

      this.ws.onopen = () => {
        this.isConnected = true;
        this.emit('connection', { connected: true });
        console.log('[VoiceClient] Connected to VoxLab host daemon');
      };

      this.ws.onclose = () => {
        this.isConnected = false;
        this.emit('connection', { connected: false });
        console.warn('[VoiceClient] Disconnected. Reconnecting in 2s...');
        this.scheduleReconnect();
      };

      this.ws.onerror = (err) => {
        console.error('[VoiceClient] WebSocket error:', err);
      };

      this.ws.onmessage = (event) => {
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
    if (list) {
      list.forEach((cb) => cb(data));
    }
  }

  sendAudioChunk(arrayBuffer) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(arrayBuffer);
    }
  }

  sendAction(action, payload = {}) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ action, ...payload }));
    } else {
      console.warn('[VoiceClient] Cannot send action, WS not open:', action);
    }
  }

  // High-level RPC methods
  activateCommand() {
    this.sendAction('activate', { mode: 'command' });
  }

  startDictation(targetId = 'scope_inspection_1') {
    this.sendAction('activate', { mode: 'dictation', target_id: targetId });
  }

  cancelTurn() {
    this.sendAction('cancel');
  }

  confirmAction(approved) {
    this.sendAction('confirm', { approved });
  }

  speak(text, voice = 'af_heart', speed = 1.0, streaming = true) {
    this.sendAction('speak', { payload: { text, voice, speed, streaming } });
  }

  setSource(sourceName) {
    this.sendAction('set_source', { source: sourceName });
  }

  setNoiseGate(thresholdDbfs) {
    this.sendAction('set_noise_gate', { threshold_dbfs: thresholdDbfs });
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
    this.sendAction('set_agc', { enabled: enabled });
  }

  setHighPass(enabled) {
    this.sendAction('set_highpass', { enabled: enabled });
  }

  setMonitor(enabled) {
    this.sendAction('set_monitor', { enabled: !!enabled });
  }

  sendPlaybackStatus(playing) {
    this.sendAction('playback_status', { playing });
  }

  injectText(text, mode = 'command') {
    this.sendAction('inject_text', { text, mode });
  }

  setTtsModel(modelName) {
    this.sendAction('set_tts_model', { model: modelName });
  }

  setAsrEngine(engine) {
    this.sendAction('set_asr_engine', { engine });
  }
}
