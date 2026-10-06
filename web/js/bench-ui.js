/**
 * BenchUI: Glues UI controls, AudioCapture, and VoiceClient together.
 */
document.addEventListener('DOMContentLoaded', () => {
  // DOM Elements
  const wsStatus = document.getElementById('ws-status');
  const voiceState = document.getElementById('voice-state');
  const echoStatus = document.getElementById('echo-status');
  const btnCancel = document.getElementById('btn-cancel');

  // Input & DSP
  const audioSourceSelect = document.getElementById('audio-source');
  const sourceStatusText = document.getElementById('source-status-text');
  const btnMicToggle = document.getElementById('btn-mic-toggle');
  const scopeCanvas = document.getElementById('scope-canvas');
  const vuBar = document.getElementById('vu-bar');
  const meterDbfs = document.getElementById('meter-dbfs');
  const gateIndicator = document.getElementById('gate-indicator');
  const noiseGateSlider = document.getElementById('noise-gate-slider');
  const noiseGateVal = document.getElementById('noise-gate-val');
  const chkHighpass = document.getElementById('chk-highpass');
  const chkAgc = document.getElementById('chk-agc');
  const chkWebrtcNs = document.getElementById('chk-webrtc-ns');

  // KWS
  const chkKwsEnable = document.getElementById('chk-kws-enable');
  const kwsKeywordSelect = document.getElementById('kws-keyword');
  const kwsThreshSlider = document.getElementById('kws-thresh-slider');
  const kwsThreshVal = document.getElementById('kws-thresh-val');

  // Tabs
  const tabBtnCommands = document.getElementById('tab-btn-commands');
  const tabBtnDictation = document.getElementById('tab-btn-dictation');
  const tabContentCommands = document.getElementById('tab-content-commands');
  const tabContentDictation = document.getElementById('tab-content-dictation');

  // Commands Tab
  const btnPttCommand = document.getElementById('btn-ptt-command');
  const btnSimCommand = document.getElementById('btn-sim-command');
  const commandTranscriptBox = document.getElementById('command-transcript-box');
  const asrStatusBadge = document.getElementById('asr-status-badge');
  const matchedIntentId = document.getElementById('matched-intent-id');
  const matchedConfidence = document.getElementById('matched-confidence');
  const matchedMargin = document.getElementById('matched-margin');
  const intentDecisionBadge = document.getElementById('intent-decision-badge');
  const slotsTags = document.getElementById('slots-tags');
  const rejectionBox = document.getElementById('rejection-box');
  const rejectionReason = document.getElementById('rejection-reason');

  // Dictation Tab
  const btnStartDictation = document.getElementById('btn-start-dictation');
  const btnStopDictation = document.getElementById('btn-stop-dictation');
  const dictationLiveBox = document.getElementById('dictation-live-box');
  const dictationStatusBadge = document.getElementById('dictation-status-badge');
  const dictationDraftText = document.getElementById('dictation-draft-text');
  const draftWordCount = document.getElementById('draft-word-count');
  const btnSaveDraft = document.getElementById('btn-save-draft');
  const btnCopyDraft = document.getElementById('btn-copy-draft');
  const btnClearDraft = document.getElementById('btn-clear-draft');

  // Kokoro TTS
  const ttsVoiceSelect = document.getElementById('tts-voice-select');
  const ttsSpeedSlider = document.getElementById('tts-speed-slider');
  const ttsSpeedVal = document.getElementById('tts-speed-val');
  const ttsInputText = document.getElementById('tts-input-text');
  const chkAutoSpeak = document.getElementById('chk-auto-speak');
  const btnSynthesize = document.getElementById('btn-synthesize');
  const ttsAudioPlayer = document.getElementById('tts-audio-player');
  const metricTtsLatency = document.getElementById('metric-tts-latency');
  const metricTtsDur = document.getElementById('metric-tts-dur');
  const metricIntentLatency = document.getElementById('metric-intent-latency');
  const eventLog = document.getElementById('event-log');
  const btnClearLog = document.getElementById('btn-clear-log');

  // Instantiate Voice Client and Audio Capture
  const client = new VoiceClient();
  const capture = new AudioCapture((chunk) => {
    client.sendAudioChunk(chunk);
  });
  capture.setCanvas(scopeCanvas);
  capture.clearVisualizer();

  // --- WebSocket Event Handlers ---
  client.on('connection', ({ connected }) => {
    if (connected) {
      wsStatus.className = 'badge badge-connected';
      wsStatus.textContent = 'CONNECTED';
    } else {
      wsStatus.className = 'badge badge-disconnected';
      wsStatus.textContent = 'DISCONNECTED';
    }
  });

  client.on('voice.state', (data) => {
    const state = data.to_state || data.state;
    voiceState.textContent = state;
    voiceState.className = 'badge ' + getStateBadgeClass(state);

    if (state === 'SPEAKING') {
      echoStatus.textContent = 'MUTED (HALF-DUPLEX)';
      echoStatus.className = 'badge badge-muted';
    } else {
      echoStatus.textContent = 'READY';
      echoStatus.className = 'badge badge-ready';
    }

    logEvent('voice.state', `${data.from_state || ''} -> ${state} (${data.trigger || ''})`);
  });

  client.on('audio.meter', (data) => {
    const dbfs = Math.round(data.dbfs);
    meterDbfs.textContent = `${dbfs} dBFS`;

    // Calculate meter width: -60 dBFS = 0%, 0 dBFS = 100%
    const norm = Math.max(0, Math.min(100, ((dbfs + 60) / 60) * 100));
    vuBar.style.width = `${norm}%`;

    if (data.echo_muted) {
      gateIndicator.className = 'gate-tag gate-closed';
      gateIndicator.textContent = 'ECHO MUTED';
    } else if (data.passed_gate) {
      gateIndicator.className = 'gate-tag gate-open';
      gateIndicator.textContent = 'GATE OPEN';
    } else {
      gateIndicator.className = 'gate-tag gate-closed';
      gateIndicator.textContent = 'NOISE GATED';
    }

    // Feed remote waveform into continuous 60 FPS visualizer
    if (data.wave && data.wave.length > 0 && (!capture.isRecording || data.source !== 'web_ui_mic')) {
      capture.pushRemoteWave(data.wave);
    }
  });

  client.on('audio.source_changed', (data) => {
    logEvent('audio.source_changed', `Source: ${data.source} (${data.device_name || 'active'})`);
    if (sourceStatusText) {
      sourceStatusText.textContent = `Active: ${data.source} (${data.device_name || 'ready'})`;
    }
    if (data.source === 'host_native_mic' || data.source === 'wav_file_injection') {
      if (capture.isRecording) capture.stop();
      capture.setRemoteMode(true);
      const label = data.source === 'host_native_mic' ? 'Host Active' : 'WAV Streaming';
      btnMicToggle.textContent = `Web Mic Idle (${label})`;
      btnMicToggle.disabled = true;
      btnMicToggle.className = 'btn btn-secondary btn-block';
    } else {
      capture.setRemoteMode(false);
      btnMicToggle.disabled = false;
      btnMicToggle.textContent = 'Start Web Mic';
      btnMicToggle.className = 'btn btn-primary btn-block';
    }
  });

  client.on('wake_word_detected', (data) => {
    logEvent('wake_word_detected', `Keyword: "${data.phrase}" (Conf: ${data.confidence.toFixed(2)})`);
    commandTranscriptBox.innerHTML = `<span style="color:#38bdf8;">[Wake word: "${data.phrase}"] Listening...</span>`;
    asrStatusBadge.textContent = 'Listening';
    asrStatusBadge.className = 'badge badge-listen';
  });

  client.on('transcript.partial', (data) => {
    if (data.mode === 'dictation') {
      dictationLiveBox.innerHTML = `<strong>${escapeHtml(data.transcript)}</strong> <span class="badge badge-listen">...</span>`;
      dictationStatusBadge.textContent = 'Transcribing';
      dictationStatusBadge.className = 'badge badge-listen';
    } else {
      commandTranscriptBox.innerHTML = `<strong>${escapeHtml(data.transcript)}</strong> <span class="badge badge-listen">...</span>`;
      asrStatusBadge.textContent = 'Decoding';
      asrStatusBadge.className = 'badge badge-listen';
    }
  });

  client.on('transcript.final', (data) => {
    logEvent('transcript.final', `"${data.transcript}"`);
    commandTranscriptBox.innerHTML = `<strong>${escapeHtml(data.transcript)}</strong>`;
    asrStatusBadge.textContent = 'Finalized';
    asrStatusBadge.className = 'badge badge-ready';
  });

  client.on('intent_matched', (data) => {
    logEvent('intent_matched', `${data.intent_id} (Conf: ${(data.confidence * 100).toFixed(1)}%, Margin: ${(data.margin * 100).toFixed(1)}%)`);
    matchedIntentId.textContent = data.intent_id;
    matchedConfidence.textContent = `${(data.confidence * 100).toFixed(0)}%`;
    matchedMargin.textContent = `+${(data.margin * 100).toFixed(0)}%`;
    intentDecisionBadge.textContent = 'MATCH ACCEPTED';
    intentDecisionBadge.className = 'badge badge-connected';
    rejectionBox.classList.add('hidden');

    // Display slots
    slotsTags.innerHTML = '';
    if (data.slots && Object.keys(data.slots).length > 0) {
      for (const [k, v] of Object.entries(data.slots)) {
        const tag = document.createElement('span');
        tag.className = 'badge badge-info';
        tag.textContent = `${k}: ${v}`;
        slotsTags.appendChild(tag);
      }
    } else {
      slotsTags.innerHTML = '<span class="badge badge-subtle">No parameters required</span>';
    }

    // Auto speak feedback if enabled
    if (chkAutoSpeak.checked) {
      let prompt = `Command ${data.intent_id.replace('_', ' ')} executed.`;
      if (data.intent_id === 'NAV_SETTINGS') prompt = 'Opening system settings.';
      else if (data.intent_id === 'START_RECORDING') prompt = 'Recording started on main channel.';
      else if (data.intent_id === 'STOP_RECORDING') prompt = 'Recording stopped and saved.';
      else if (data.intent_id === 'DISPLAY_DARK') prompt = 'Display brightness lowered.';
      client.speak(prompt, ttsVoiceSelect.value, parseFloat(ttsSpeedSlider.value));
    }
  });

  client.on('utterance_rejected', (data) => {
    logEvent('utterance_rejected', `Reason: ${data.reason} (Raw: "${data.raw_transcript}")`);
    matchedIntentId.textContent = '--';
    matchedConfidence.textContent = data.confidence ? `${(data.confidence * 100).toFixed(0)}%` : '--';
    matchedMargin.textContent = data.margin ? `${(data.margin * 100).toFixed(0)}%` : '--';
    intentDecisionBadge.textContent = 'REJECTED';
    intentDecisionBadge.className = 'badge badge-disconnected';

    rejectionBox.classList.remove('hidden');
    rejectionReason.textContent = `Ignored (${data.reason}): "${data.raw_transcript}"`;
  });

  client.on('annotation.draft', (data) => {
    logEvent('annotation.draft', `Draft received (${data.word_count} words)`);
    dictationDraftText.value = data.draft_text;
    draftWordCount.textContent = data.word_count;
    dictationLiveBox.innerHTML = `<span>Utterance finished. Draft ready for review.</span>`;
    dictationStatusBadge.textContent = 'Draft Ready';
    dictationStatusBadge.className = 'badge badge-ready';
  });

  client.on('tts.started', (data) => {
    logEvent('tts.started', `"${data.text}" (${data.voice})`);
  });

  client.on('tts.finished', (data) => {
    logEvent('tts.finished', `Duration: ${data.duration_sec.toFixed(2)}s, TTFA: ${data.latency_ms}ms`);
    metricTtsLatency.textContent = `${data.latency_ms} ms`;
    metricTtsDur.textContent = `${data.duration_sec.toFixed(2)} s`;

    if (data.wav_base64) {
      const audioUrl = `data:audio/wav;base64,${data.wav_base64}`;
      ttsAudioPlayer.src = audioUrl;
      ttsAudioPlayer.play().catch((e) => console.log('Autoplay policy caught:', e));
    }
  });

  client.on('telemetry.benchmark', (data) => {
    if (data.intent_latency_ms !== undefined) {
      metricIntentLatency.textContent = `${data.intent_latency_ms} ms`;
    }
  });

  // --- UI Event Listeners ---

  // Mic Toggle
  btnMicToggle.addEventListener('click', async () => {
    if (capture.isRecording) {
      capture.stop();
      btnMicToggle.textContent = 'Start Web Mic';
      btnMicToggle.className = 'btn btn-primary btn-block';
    } else {
      try {
        await capture.start({
          noiseSuppression: chkWebrtcNs.checked,
          autoGainControl: chkAgc.checked,
          echoCancellation: true,
        });
        btnMicToggle.textContent = 'Stop Web Mic';
        btnMicToggle.className = 'btn btn-danger btn-block';
      } catch (e) {
        alert('Could not start microphone: ' + e.message);
      }
    }
  });

  // Source selection
  audioSourceSelect.addEventListener('change', (e) => {
    const src = e.target.value;
    client.setSource(src);
    logEvent('source_change', `Switching audio capture to ${src}...`);
    if (src === 'host_native_mic') {
      if (capture.isRecording) capture.stop();
      capture.setRemoteMode(true);
      btnMicToggle.textContent = 'Web Mic Idle (Host Active)';
      btnMicToggle.disabled = true;
      btnMicToggle.className = 'btn btn-secondary btn-block';
      if (sourceStatusText) sourceStatusText.textContent = 'Connecting to Host Microphone...';
    } else if (src === 'wav_file_injection') {
      if (capture.isRecording) capture.stop();
      capture.setRemoteMode(true);
      btnMicToggle.textContent = 'Web Mic Idle (WAV Streaming)';
      btnMicToggle.disabled = true;
      btnMicToggle.className = 'btn btn-secondary btn-block';
      if (sourceStatusText) sourceStatusText.textContent = 'Injecting voice_test.wav in real-time...';
    } else {
      capture.setRemoteMode(false);
      btnMicToggle.disabled = false;
      btnMicToggle.textContent = 'Start Web Mic';
      btnMicToggle.className = 'btn btn-primary btn-block';
      if (sourceStatusText) sourceStatusText.textContent = 'Active: Browser Web Audio Mic';
    }
  });

  // Sliders
  noiseGateSlider.addEventListener('input', (e) => {
    noiseGateVal.textContent = `${e.target.value} dBFS`;
  });

  kwsThreshSlider.addEventListener('input', (e) => {
    kwsThreshVal.textContent = parseFloat(e.target.value).toFixed(2);
  });

  ttsSpeedSlider.addEventListener('input', (e) => {
    ttsSpeedVal.textContent = `${parseFloat(e.target.value).toFixed(1)}x`;
  });

  // Buttons
  btnCancel.addEventListener('click', () => {
    client.cancelTurn();
    commandTranscriptBox.innerHTML = '<span class="placeholder-text">Session cancelled. Returned to IDLE.</span>';
    asrStatusBadge.textContent = 'Idle';
    asrStatusBadge.className = 'badge badge-subtle';
  });

  btnPttCommand.addEventListener('click', () => {
    client.activateCommand();
    commandTranscriptBox.innerHTML = '<span style="color:#60a5fa;">[Manual Push-to-Talk] Listening for command...</span>';
    asrStatusBadge.textContent = 'Listening';
    asrStatusBadge.className = 'badge badge-listen';
  });

  btnSimCommand.addEventListener('click', () => {
    client.injectText('open settings', 'command');
  });

  btnStartDictation.addEventListener('click', () => {
    client.startDictation('scope_inspection_1');
    dictationLiveBox.innerHTML = '<span style="color:#60a5fa;">[Dictation Session Open] Speak your notes...</span>';
    dictationStatusBadge.textContent = 'Listening';
    dictationStatusBadge.className = 'badge badge-listen';
  });

  btnStopDictation.addEventListener('click', () => {
    client.cancelTurn();
    dictationLiveBox.innerHTML = '<span class="placeholder-text">Dictation ended.</span>';
    dictationStatusBadge.textContent = 'Idle';
    dictationStatusBadge.className = 'badge badge-subtle';
  });

  btnSaveDraft.addEventListener('click', () => {
    alert('Annotation draft approved and persisted to procedure log!');
    dictationDraftText.value = '';
    draftWordCount.textContent = '0';
  });

  btnCopyDraft.addEventListener('click', () => {
    navigator.clipboard.writeText(dictationDraftText.value);
    alert('Copied to clipboard!');
  });

  btnClearDraft.addEventListener('click', () => {
    dictationDraftText.value = '';
    draftWordCount.textContent = '0';
  });

  // Kokoro Synthesize
  btnSynthesize.addEventListener('click', () => {
    const text = ttsInputText.value.trim();
    if (!text) return;
    client.speak(text, ttsVoiceSelect.value, parseFloat(ttsSpeedSlider.value));
  });

  // Quick Injections
  document.querySelectorAll('.btn-inject').forEach((btn) => {
    btn.addEventListener('click', () => {
      const text = btn.getAttribute('data-text');
      client.injectText(text, 'command');
    });
  });

  // Quick Prompts
  document.querySelectorAll('.btn-prompt').forEach((btn) => {
    btn.addEventListener('click', () => {
      const text = btn.getAttribute('data-text');
      ttsInputText.value = text;
      client.speak(text, ttsVoiceSelect.value, parseFloat(ttsSpeedSlider.value));
    });
  });

  // Tabs
  tabBtnCommands.addEventListener('click', () => {
    tabBtnCommands.classList.add('active');
    tabBtnDictation.classList.remove('active');
    tabContentCommands.classList.add('active');
    tabContentDictation.classList.remove('active');
  });

  tabBtnDictation.addEventListener('click', () => {
    tabBtnDictation.classList.add('active');
    tabBtnCommands.classList.remove('active');
    tabContentDictation.classList.add('active');
    tabContentCommands.classList.remove('active');
  });

  btnClearLog.addEventListener('click', () => {
    eventLog.innerHTML = '';
  });

  // Helper Functions
  function getStateBadgeClass(state) {
    switch (state) {
      case 'IDLE': return 'badge-idle';
      case 'COMMAND_LISTEN':
      case 'ANNOTATION_LISTEN': return 'badge-listen';
      case 'SPEAKING': return 'badge-speak';
      case 'PROCESS': return 'badge-info';
      case 'CONFIRM': return 'badge-muted';
      default: return 'badge-idle';
    }
  }

  function logEvent(event, detail) {
    const timeStr = new Date().toLocaleTimeString();
    const div = document.createElement('div');
    div.className = 'log-entry';
    div.innerHTML = `<span class="log-time">[${timeStr}]</span> <span class="log-event">${event}:</span> ${escapeHtml(detail)}`;
    eventLog.prepend(div);
  }

  function escapeHtml(str) {
    if (!str) return '';
    return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }

  // Connect WebSocket
  client.connect();
});
