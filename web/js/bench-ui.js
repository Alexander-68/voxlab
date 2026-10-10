/**
 * BenchUI: Glues UI controls, AudioCapture, and VoiceClient together.
 */
document.addEventListener('DOMContentLoaded', () => {
  // DOM Elements
  const engineStatus = document.getElementById('engine-status');
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
  const agcVal = document.getElementById('agc-val');
  const chkWebrtcNs = document.getElementById('chk-webrtc-ns');
  const webrtcNsLabel = document.getElementById('webrtc-ns-label');
  const chkMonitor = document.getElementById('chk-monitor');
  const dspSettingsBox = document.getElementById('dsp-settings-box');
  const dspBypassedBadge = document.getElementById('dsp-bypassed-badge');

  // KWS
  const chkKwsEnable = document.getElementById('chk-kws-enable');
  const kwsKeywordInput = document.getElementById('kws-keyword');
  const kwsThreshSlider = document.getElementById('kws-thresh-slider');
  const kwsThreshVal = document.getElementById('kws-thresh-val');
  const kwsSettingsBox = document.getElementById('kws-settings-box');
  const kwsDisabledBadge = document.getElementById('kws-disabled-badge');
  const kwsAcousticControls = document.getElementById('kws-acoustic-controls');

  // Tabs
  const tabBtnCommands = document.getElementById('tab-btn-commands');
  const tabBtnDictation = document.getElementById('tab-btn-dictation');
  const tabContentCommands = document.getElementById('tab-content-commands');
  const tabContentDictation = document.getElementById('tab-content-dictation');

  // ASR Engine Selector
  const asrEngineSelect = document.getElementById('asr-engine-select');
  const asrEngineName = document.getElementById('asr-engine-name');

  // Commands Tab
  const btnPttCommand = document.getElementById('btn-ptt-command');
  const commandTranscriptBox = document.getElementById('command-transcript-box');
  const asrStatusBadge = document.getElementById('asr-status-badge');
  const matchedIntentId = document.getElementById('matched-intent-id');
  const matchedConfidence = document.getElementById('matched-confidence');
  const matchedMargin = document.getElementById('matched-margin');
  const intentDecisionBadge = document.getElementById('intent-decision-badge');
  const slotsTags = document.getElementById('slots-tags');
  const rejectionBox = document.getElementById('rejection-box');
  const rejectionReason = document.getElementById('rejection-reason');
  const customCmdInput = document.getElementById('custom-cmd-input');
  const btnCustomInject = document.getElementById('btn-custom-inject');

  // Dictation Tab
  const btnStartDictation = document.getElementById('btn-start-dictation');
  const btnStopDictation = document.getElementById('btn-stop-dictation');
  const customDictInput = document.getElementById('custom-dict-input');
  const btnCustomDictInject = document.getElementById('btn-custom-dict-inject');
  const dictationLiveBox = document.getElementById('dictation-live-box');
  const dictationStatusBadge = document.getElementById('dictation-status-badge');
  const dictationDraftText = document.getElementById('dictation-draft-text');
  const draftWordCount = document.getElementById('draft-word-count');
  const btnSaveDraft = document.getElementById('btn-save-draft');
  const btnCopyDraft = document.getElementById('btn-copy-draft');
  const btnClearDraft = document.getElementById('btn-clear-draft');

  // Kokoro TTS
  const ttsModelSelect = document.getElementById('tts-model-select');
  const ttsVoiceSelect = document.getElementById('tts-voice-select');
  const ttsVoiceCount = document.getElementById('tts-voice-count');
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

  const ttsModelBadge = document.getElementById('tts-model-badge');
  const ttsModelName = document.getElementById('tts-model-name');
  const ttsPlayerModel = document.getElementById('tts-player-model');

  // Instantiate Voice Client and Audio Capture
  const client = new VoiceClient();
  const capture = new AudioCapture((chunk) => {
    client.sendAudioChunk(chunk);
  });
  capture.setCanvas(scopeCanvas);
  capture.clearVisualizer();

  // --- Dictation Punctuation & Formatting Helpers ---
  function formatDictationPhrase(rawText) {
    if (!rawText) return '';
    let text = rawText.trim();
    if (!text) return '';

    const punctuationMap = [
      { regex: /\b(period|full stop)\b/gi, replacement: '.' },
      { regex: /\bcomma\b/gi, replacement: ',' },
      { regex: /\bquestion mark\b/gi, replacement: '?' },
      { regex: /\bexclamation (?:mark|point)\b/gi, replacement: '!' },
      { regex: /\bsemicolon\b/gi, replacement: ';' },
      { regex: /\bcolon\b/gi, replacement: ':' },
      { regex: /\bnew paragraph\b/gi, replacement: '\n\n' },
      { regex: /\bnew line\b/gi, replacement: '\n' },
    ];

    for (const { regex, replacement } of punctuationMap) {
      text = text.replace(regex, replacement);
    }

    // Adjust spacing around punctuation and clean newlines
    text = text.replace(/[ \t]+([.,;:?!])/g, '$1');
    text = text.replace(/([.,;:?!])([a-zA-Z0-9])/g, '$1 $2');
    text = text.replace(/[ \t]*\n[ \t]*/g, '\n');
    text = text.replace(/\n{3,}/g, '\n\n');
    text = text.replace(/[ \t]{2,}/g, ' ');
    text = text.replace(/^[ \t]+|[ \t]+$/g, '');

    if (!text) return '';

    // Append semicolon if no punctuation was spoken at the end of the phrase
    if (!/[.;,!?:\n]$/.test(text)) {
      text += ';';
    }

    // Capitalize first letter of the phrase (accounting for leading newlines)
    text = text.replace(/^(\s*)([a-z])/i, (_, p1, p2) => p1 + p2.toUpperCase());
    // Capitalize after sentence-ending punctuation (. ? !)
    text = text.replace(/([.?!]\s+)([a-z])/g, (_, p1, p2) => p1 + p2.toUpperCase());
    // Capitalize after newlines
    text = text.replace(/(\n+\s*)([a-z])/g, (_, p1, p2) => p1 + p2.toUpperCase());

    return text;
  }

  function appendDictationPhrase(currentText, newPhrase) {
    const formatted = formatDictationPhrase(newPhrase);
    if (!formatted) return currentText || '';
    if (!currentText || !currentText.trim()) return formatted;
    const current = currentText.trimEnd();
    if (formatted.startsWith('\n')) {
      return current + formatted;
    }
    return current + ' ' + formatted;
  }

  // --- Browser Web Speech API Support ---
  const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
  let browserRecognizer = null;
  let isBrowserRecognizing = false;
  let isDictationActive = false;
  let isPttActive = false;
  let isBrowserWakeActive = false;
  let browserWakeResetTimer = null;
  let browserTargetMode = 'command';

  if (SpeechRecognition) {
    try {
      browserRecognizer = new SpeechRecognition();
      browserRecognizer.continuous = false;
      browserRecognizer.interimResults = true;
      browserRecognizer.maxAlternatives = 1;
      browserRecognizer.lang = 'en-US';

      browserRecognizer.onstart = () => {
        isBrowserRecognizing = true;
        if (browserTargetMode === 'command') {
          const isKws = chkKwsEnable && chkKwsEnable.checked && (kwsKeywordInput ? kwsKeywordInput.value.trim().length > 0 : false);
          asrStatusBadge.textContent = isPttActive ? 'Listening (PTT)' : (isKws ? 'Listening (KWS)' : 'Listening (Browser)');
          asrStatusBadge.className = 'badge badge-listen';
          if (!isPttActive && isKws && !isBrowserWakeActive) {
            commandTranscriptBox.innerHTML = `<span class="placeholder-text">Awaiting wake phrase ("${escapeHtml(kwsKeywordInput.value.trim())}") or Push to talk...</span>`;
          } else if (isPttActive) {
            commandTranscriptBox.innerHTML = '<span style="color:#60a5fa;">[Push to talk] Listening for command...</span>';
          }
        } else {
          dictationStatusBadge.textContent = 'Listening (Browser)';
          dictationStatusBadge.className = 'badge badge-listen';
          dictationLiveBox.innerHTML = '<span style="color:var(--cyan);">[Browser Web Speech] Listening...</span>';
        }
        logEvent('browser_asr.start', 'Browser SpeechRecognition started');
      };

      browserRecognizer.onresult = (event) => {
        let interim = '';
        let finalStr = '';
        for (let i = event.resultIndex; i < event.results.length; ++i) {
          const item = event.results[i];
          if (item.isFinal) {
            finalStr += item[0].transcript;
          } else {
            interim += item[0].transcript;
          }
        }

        const display = (finalStr || interim).trim();
        if (browserTargetMode === 'command') {
          const wakePhrase = (kwsKeywordInput ? kwsKeywordInput.value.trim().toLowerCase() : 'hey voxlab');
          const isKwsEnabled = chkKwsEnable && chkKwsEnable.checked && wakePhrase.length > 0;

          // Case 1: Manual Push-to-Talk active -> directly transcribe and execute command
          if (isPttActive || !isKwsEnabled) {
            if (display) {
              commandTranscriptBox.innerHTML = `<strong>${escapeHtml(display)}</strong>`;
            }
            if (finalStr.trim()) {
              logEvent('browser_asr.final', `"${finalStr.trim()}"`);
              client.injectText(finalStr.trim(), 'command');
              isPttActive = false;
            }
            return;
          }

          // Case 2: Option A - Continuous Wake-Word Detection active in Browser Mode
          const lowDisplay = display.toLowerCase();
          const hasWakePhrase = wakePhrase && lowDisplay.includes(wakePhrase);

          if (hasWakePhrase) {
            // Wake phrase spotted in speech!
            asrStatusBadge.textContent = 'Listening';
            asrStatusBadge.className = 'badge badge-listen';

            const wakeIdx = lowDisplay.indexOf(wakePhrase);
            const remainder = display.slice(wakeIdx + wakePhrase.length).replace(/^[,\s.:;?!]+/, '').trim();

            if (!remainder) {
              // User spoke wake phrase alone ("Hey VoxLab")
              isBrowserWakeActive = true;
              commandTranscriptBox.innerHTML = `<span style="color:#38bdf8;">[Wake word: "${escapeHtml(wakePhrase)}"]</span> Listening for command...`;
              logEvent('browser_kws.detected', `Wake phrase spotted: "${wakePhrase}"`);
              if (browserWakeResetTimer) clearTimeout(browserWakeResetTimer);
              browserWakeResetTimer = setTimeout(() => {
                isBrowserWakeActive = false;
                if (!isPttActive && asrStatusBadge.textContent.includes('Listening')) {
                  asrStatusBadge.textContent = 'Listening (KWS)';
                  asrStatusBadge.className = 'badge badge-subtle';
                  commandTranscriptBox.innerHTML = `<span class="placeholder-text">Awaiting wake phrase ("${escapeHtml(wakePhrase)}") or Push to talk...</span>`;
                }
              }, 6000);
            } else {
              // User spoke wake phrase + command in one breath ("Hey VoxLab open settings")
              commandTranscriptBox.innerHTML = `<span style="color:#38bdf8;">[Wake word: "${escapeHtml(wakePhrase)}"]</span> <strong>${escapeHtml(remainder)}</strong>`;
              if (finalStr.trim()) {
                const finalLow = finalStr.toLowerCase();
                const fWakeIdx = finalLow.indexOf(wakePhrase);
                const finalCmd = finalStr.slice(fWakeIdx + wakePhrase.length).replace(/^[,\s.:;?!]+/, '').trim();
                if (finalCmd) {
                  logEvent('browser_kws.final', `Wake command executed: "${finalCmd}"`);
                  client.injectText(finalCmd, 'command');
                  isBrowserWakeActive = false;
                  if (browserWakeResetTimer) clearTimeout(browserWakeResetTimer);
                }
              }
            }
          } else if (isBrowserWakeActive) {
            // Wake phrase was spotted in previous turn, this utterance is the command!
            if (display) {
              commandTranscriptBox.innerHTML = `<span style="color:#38bdf8;">[Command]:</span> <strong>${escapeHtml(display)}</strong>`;
            }
            if (finalStr.trim()) {
              logEvent('browser_kws.followup', `Command executed: "${finalStr.trim()}"`);
              client.injectText(finalStr.trim(), 'command');
              isBrowserWakeActive = false;
              if (browserWakeResetTimer) clearTimeout(browserWakeResetTimer);
            }
          } else {
            // Conversational speech / chatter without wake phrase -> ignore
            commandTranscriptBox.innerHTML = `<span class="placeholder-text" style="color:var(--text-muted);">Awaiting wake phrase ("${escapeHtml(wakePhrase)}") or Push to talk...</span>`;
          }
        } else {
          // Dictation mode
          if (display) {
            dictationLiveBox.innerHTML = `<strong>${escapeHtml(display)}</strong>`;
          }
          if (finalStr.trim()) {
            const formatted = formatDictationPhrase(finalStr);
            dictationDraftText.value = appendDictationPhrase(dictationDraftText.value, finalStr);
            dictationLiveBox.innerHTML = `<strong>${escapeHtml(formatted)}</strong>`;
            draftWordCount.textContent = dictationDraftText.value.trim().split(/\s+/).filter(Boolean).length;
            dictationStatusBadge.textContent = 'Dictating';
            dictationStatusBadge.className = 'badge badge-ready';
            logEvent('browser_asr.dictation', `Draft text: "${dictationDraftText.value}"`);
          }
        }
      };

      browserRecognizer.onerror = (event) => {
        isBrowserRecognizing = false;
        const err = event.error || 'unknown';
        logEvent('browser_asr.error', `Web Speech API error: ${err}`);
        if (browserTargetMode === 'command') {
          asrStatusBadge.textContent = `Error: ${err}`;
          asrStatusBadge.className = 'badge badge-disconnected';
        } else {
          dictationStatusBadge.textContent = `Error: ${err}`;
          dictationStatusBadge.className = 'badge badge-disconnected';
        }
      };

      browserRecognizer.onend = () => {
        isBrowserRecognizing = false;
        // Continuous dictation: continue until "Stop dictation" is clicked
        if (browserTargetMode === 'dictation' && isDictationActive) {
          try {
            browserRecognizer.continuous = true;
            browserRecognizer.start();
            isBrowserRecognizing = true;
            return;
          } catch (e) {
            console.warn('Could not auto-restart browser dictation recognizer:', e);
          }
        }

        // Continuous wake-word detection in browser mode: continue listening for wake phrase
        const currAsr = asrEngineSelect ? asrEngineSelect.value : 'sherpa';
        const isKwsEnabled = chkKwsEnable && chkKwsEnable.checked && (kwsKeywordInput ? kwsKeywordInput.value.trim().length > 0 : false);
        if (currAsr === 'browser' && browserTargetMode === 'command' && isKwsEnabled && tabBtnCommands.classList.contains('active')) {
          try {
            browserRecognizer.continuous = true;
            browserRecognizer.start();
            isBrowserRecognizing = true;
            return;
          } catch (e) {
            console.warn('Could not auto-restart browser KWS recognizer:', e);
          }
        }

        if (browserTargetMode === 'command') {
          if (asrStatusBadge.textContent.includes('Listening')) {
            asrStatusBadge.textContent = isKwsEnabled ? 'Listening (KWS)' : 'Ready';
            asrStatusBadge.className = 'badge badge-subtle';
          }
        } else {
          if (!isDictationActive && dictationStatusBadge.textContent.includes('Listening')) {
            dictationStatusBadge.textContent = 'Idle';
            dictationStatusBadge.className = 'badge badge-subtle';
          }
        }
      };
    } catch (e) {
      console.warn('[VoxLab] Web Speech API init failed:', e);
    }
  }

  function updateAsrEngineDisplay(engine) {
    if (!engine) return;
    let label = 'Sherpa-ONNX';
    let badgeText = 'SHERPA NEURAL';
    let badgeClass = 'badge badge-connected';

    if (engine === 'browser') {
      label = 'Browser Web Speech API';
      badgeText = 'WEB SPEECH API';
      badgeClass = 'badge badge-info';
    } else if (engine === 'sim') {
      label = 'Simulator';
      badgeText = 'SIMULATOR';
      badgeClass = 'badge badge-muted';
    }

    if (asrEngineName) {
      asrEngineName.textContent = label;
    }
    if (asrEngineSelect && asrEngineSelect.value !== engine) {
      asrEngineSelect.value = engine;
    }
    if (engineStatus) {
      engineStatus.textContent = badgeText;
      engineStatus.className = badgeClass;
      engineStatus.title = `Active ASR Engine: ${label}`;
    }

    // In browser mode: visibly disable panel "Noise Cancelling & Filters", fix "Audio Capture Source" to web_ui_mic, hide acoustic controls
    if (engine === 'browser') {
      if (audioSourceSelect) {
        audioSourceSelect.value = 'web_ui_mic';
        audioSourceSelect.disabled = true;
      }
      if (sourceStatusText) {
        sourceStatusText.textContent = 'Active: Browser Web Audio Mic (Fixed for Browser ASR)';
      }
      if (capture && capture.isRemoteMode) {
        capture.setRemoteMode(false);
        if (btnMicToggle) {
          btnMicToggle.disabled = false;
          btnMicToggle.textContent = capture.isRecording ? 'Stop Web Mic' : 'Start Web Mic';
          btnMicToggle.className = capture.isRecording ? 'btn btn-danger btn-block' : 'btn btn-primary btn-block';
        }
        client.setSource('web_ui_mic');
      }
      if (dspSettingsBox) {
        dspSettingsBox.classList.add('section-disabled');
        dspSettingsBox.querySelectorAll('input, select, button').forEach((el) => { el.disabled = true; });
      }
      if (dspBypassedBadge) {
        dspBypassedBadge.classList.remove('hidden');
      }
      if (kwsAcousticControls) {
        kwsAcousticControls.style.display = 'none';
      }
      // If continuous wake-word detection is enabled, arm browser recognizer for wake phrase spotting
      if (chkKwsEnable && chkKwsEnable.checked && (kwsKeywordInput ? kwsKeywordInput.value.trim().length > 0 : false) && tabBtnCommands.classList.contains('active')) {
        browserTargetMode = 'command';
        isPttActive = false;
        try {
          if (!isBrowserRecognizing && browserRecognizer) {
            browserRecognizer.continuous = true;
            browserRecognizer.start();
          }
        } catch (_) {}
      }
    } else {
      if (audioSourceSelect) {
        audioSourceSelect.disabled = false;
      }
      if (sourceStatusText && audioSourceSelect) {
        const src = audioSourceSelect.value;
        sourceStatusText.textContent = src === 'web_ui_mic' ? 'Active: Browser Web Audio Mic' :
          (src === 'host_native_mic' ? 'Connecting to Host Microphone...' : 'Injecting voice_test.wav in real-time...');
      }
      if (dspSettingsBox) {
        dspSettingsBox.classList.remove('section-disabled');
        dspSettingsBox.querySelectorAll('input, select, button').forEach((el) => { el.disabled = false; });
      }
      if (dspBypassedBadge) {
        dspBypassedBadge.classList.add('hidden');
      }
      if (kwsAcousticControls) {
        kwsAcousticControls.style.display = '';
      }
      if (!isDictationActive && isBrowserRecognizing && browserRecognizer) {
        try { browserRecognizer.stop(); } catch (_) {}
      }
      if (audioSourceSelect) {
        updateSourceControls(audioSourceSelect.value);
      }
    }
  }

  // --- WebSocket Event Handlers ---
  client.on('connection', ({ connected }) => {
    if (connected) {
      wsStatus.className = 'badge badge-connected';
      wsStatus.textContent = 'CONNECTED';
      if (chkMonitor && chkMonitor.checked) {
        client.setMonitor(true);
      }
    } else {
      wsStatus.className = 'badge badge-disconnected';
      wsStatus.textContent = 'DISCONNECTED';
    }
  });

  function updateSourceControls(src) {
    if (src === 'host_native_mic' || src === 'wav_file_injection') {
      chkWebrtcNs.disabled = true;
      if (webrtcNsLabel) webrtcNsLabel.textContent = 'Browser WebRTC Noise Suppression (Web Mic only)';
    } else {
      chkWebrtcNs.disabled = false;
      if (webrtcNsLabel) webrtcNsLabel.textContent = 'Browser WebRTC Noise Suppression (Web Mic)';
    }
  }

  function updateTtsModelDisplay(model, isNeural) {
    if (!model) return;
    if (ttsModelBadge) {
      ttsModelBadge.textContent = model.toUpperCase();
      ttsModelBadge.className = isNeural ? 'badge badge-connected' : 'badge badge-muted';
      ttsModelBadge.title = `Active TTS Model: ${model}`;
    }
    if (ttsModelName) {
      ttsModelName.textContent = model;
    }
    if (ttsPlayerModel) {
      ttsPlayerModel.textContent = model;
    }
    if (ttsModelSelect) {
      let matched = false;
      const lowModel = model.toLowerCase();
      // 1. Exact match
      for (const opt of ttsModelSelect.options) {
        if (opt.value && opt.value.toLowerCase() === lowModel) {
          ttsModelSelect.value = opt.value;
          matched = true;
          break;
        }
      }
      // 2. Specific variant match (FP16 or INT8)
      if (!matched && lowModel.includes('fp16')) {
        for (const opt of ttsModelSelect.options) {
          if (opt.value && opt.value.toLowerCase().includes('fp16')) {
            ttsModelSelect.value = opt.value;
            matched = true;
            break;
          }
        }
      }
      if (!matched && lowModel.includes('int8')) {
        for (const opt of ttsModelSelect.options) {
          if (opt.value && opt.value.toLowerCase().includes('int8')) {
            ttsModelSelect.value = opt.value;
            matched = true;
            break;
          }
        }
      }
      // 3. Fallback contains match (skipping FP16/INT8 options if active model is standard FP32)
      if (!matched) {
        for (const opt of ttsModelSelect.options) {
          const optLow = (opt.value || '').toLowerCase();
          const isVariantOpt = optLow.includes('fp16') || optLow.includes('int8');
          if (opt.value && !isVariantOpt && lowModel.includes(optLow)) {
            ttsModelSelect.value = opt.value;
            matched = true;
            break;
          }
        }
      }
    }
  }

  client.on('voice.state', (data) => {
    const state = data.to_state || data.state;
    voiceState.textContent = state;
    voiceState.className = 'badge ' + getStateBadgeClass(state);

    if (data.tts_model) {
      updateTtsModelDisplay(data.tts_model, data.tts_is_neural);
    }

    if (data.asr_engine) {
      updateAsrEngineDisplay(data.asr_engine);
    } else if (data.engine_mode && engineStatus) {
      if (data.engine_mode.includes('sherpa')) {
        updateAsrEngineDisplay('sherpa');
      } else {
        updateAsrEngineDisplay('sim');
      }
    }

    if (state === 'SPEAKING') {
      echoStatus.textContent = 'MUTED (HALF-DUPLEX)';
      echoStatus.className = 'badge badge-muted';
    } else {
      echoStatus.textContent = 'READY';
      echoStatus.className = 'badge badge-ready';
    }

    logEvent('voice.state', `${data.from_state || ''} -> ${state} (${data.trigger || ''})`);
  });

  client.on('asr.engine_changed', (data) => {
    logEvent('asr.engine_changed', `ASR Engine: ${data.engine} (${data.name || ''})`);
    updateAsrEngineDisplay(data.engine);
  });

  client.on('audio.meter', (data) => {
    const dbfs = Math.round(data.dbfs);
    meterDbfs.textContent = `${dbfs} dBFS`;

    // Calculate meter width: -60 dBFS = 0%, 0 dBFS = 100%
    const norm = Math.max(0, Math.min(100, ((dbfs + 60) / 60) * 100));
    vuBar.style.width = `${norm}%`;

    if (agcVal && data.agc_gain !== undefined) {
      agcVal.textContent = `${data.agc_gain.toFixed(1)}x`;
    }

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
      capture.pushRemoteWave(data.wave, data.passed_gate);
    }
  });

  client.on('audio.source_changed', (data) => {
    logEvent('audio.source_changed', `Source: ${data.source} (${data.device_name || 'active'})`);
    updateSourceControls(data.source);
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
    if (chkMonitor && chkMonitor.checked) {
      client.setMonitor(true);
    }
  });

  client.on('audio.chunk', (buf) => {
    capture.playRemotePCM(buf);
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
    dictationLiveBox.innerHTML = `<span>Dictation finished. Draft ready for review.</span>`;
    dictationStatusBadge.textContent = 'Draft Ready';
    dictationStatusBadge.className = 'badge badge-ready';
  });

  client.on('tts.started', (data) => {
    const modelTag = data.model ? ` [${data.model}]` : '';
    const speedTag = data.speed ? ` @ ${data.speed}x` : '';
    logEvent('tts.started', `"${data.text}" (${data.voice}${speedTag})${modelTag}`);
    if (data.model) {
      updateTtsModelDisplay(data.model, data.is_neural);
    }
    btnSynthesize.disabled = true;
    btnSynthesize.textContent = '⏳ Synthesizing Audio...';
  });

  client.on('tts.finished', (data) => {
    const modelTag = data.model ? ` [${data.model}]` : '';
    logEvent('tts.finished', `Duration: ${data.duration_sec.toFixed(2)}s, TTFA: ${data.latency_ms}ms${modelTag}`);
    metricTtsLatency.textContent = `${data.latency_ms} ms`;
    metricTtsDur.textContent = `${data.duration_sec.toFixed(2)} s`;
    if (ttsPlayerModel && data.model) {
      ttsPlayerModel.textContent = data.model;
    }
    btnSynthesize.disabled = false;
    btnSynthesize.textContent = '🔊 Synthesize & Speak';

    if (data.wav_base64) {
      const audioUrl = `data:audio/wav;base64,${data.wav_base64}`;
      ttsAudioPlayer.src = audioUrl;
      ttsAudioPlayer.play().catch((e) => {
        console.warn('Autoplay prevented or failed:', e);
        client.sendPlaybackStatus(false);
      });
    }
  });

  client.on('tts.model_changed', (data) => {
    logEvent('tts.model_changed', `Model changed: ${data.model} (${data.voices ? data.voices.length : 0} voices)`);
    updateTtsModelDisplay(data.model, data.is_neural);
    if (data.installed_models) {
      populateModelsDropdown(data.installed_models, data.model);
    }
    if (data.voices) {
      populateVoicesDropdown(data.voices);
    }
  });

  // Half-duplex echo suppression: coordinate physical speaker output with server echo gate
  ttsAudioPlayer.addEventListener('play', () => {
    logEvent('playback.start', 'Audio playback started -> Echo Gate MUTED');
    client.sendPlaybackStatus(true);
  });

  ttsAudioPlayer.addEventListener('playing', () => {
    client.sendPlaybackStatus(true);
  });

  ttsAudioPlayer.addEventListener('ended', () => {
    logEvent('playback.end', 'Audio playback completed -> Echo Gate OPEN');
    client.sendPlaybackStatus(false);
  });

  ttsAudioPlayer.addEventListener('pause', () => {
    if (ttsAudioPlayer.paused || ttsAudioPlayer.currentTime >= ttsAudioPlayer.duration) {
      client.sendPlaybackStatus(false);
    }
  });

  ttsAudioPlayer.addEventListener('error', (e) => {
    console.warn('[TTS] Audio player error:', e);
    client.sendPlaybackStatus(false);
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
    updateSourceControls(src);
    client.setSource(src);
    if (chkMonitor && chkMonitor.checked) {
      client.setMonitor(true);
    }
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
    const val = parseFloat(e.target.value);
    noiseGateVal.textContent = `${val} dBFS`;
    client.setNoiseGate(val);
  });

  kwsThreshSlider.addEventListener('input', (e) => {
    const val = parseFloat(e.target.value);
    kwsThreshVal.textContent = val.toFixed(2);
    client.setKwsThreshold(val);
  });

  // KWS controls & validation
  function updateKwsValidation() {
    const kw = kwsKeywordInput ? kwsKeywordInput.value.trim() : '';
    const hasKeyword = kw.length > 0;
    if (chkKwsEnable) {
      if (!hasKeyword) {
        chkKwsEnable.disabled = true;
        if (chkKwsEnable.checked) {
          chkKwsEnable.checked = false;
          handleKwsToggle(false);
        }
      } else {
        if (!kwsSettingsBox || !kwsSettingsBox.classList.contains('section-disabled')) {
          chkKwsEnable.disabled = false;
        }
      }
    }
  }

  function handleKwsToggle(enabled) {
    client.setKwsEnabled(enabled);
    logEvent('kws.toggle', `Continuous Wake-Word Detection: ${enabled ? 'ENABLED' : 'DISABLED'}`);
    const currAsr = asrEngineSelect ? asrEngineSelect.value : 'sherpa';
    if (currAsr === 'browser') {
      if (enabled && tabBtnCommands.classList.contains('active')) {
        browserTargetMode = 'command';
        isPttActive = false;
        isBrowserWakeActive = false;
        try {
          if (!isBrowserRecognizing && browserRecognizer) {
            browserRecognizer.continuous = true;
            browserRecognizer.start();
          }
        } catch (_) {}
        commandTranscriptBox.innerHTML = `<span class="placeholder-text">Awaiting wake phrase ("${escapeHtml(kwsKeywordInput ? kwsKeywordInput.value.trim() : '')}") or Push to talk...</span>`;
        asrStatusBadge.textContent = 'Listening (KWS)';
        asrStatusBadge.className = 'badge badge-subtle';
      } else {
        if (!isDictationActive && isBrowserRecognizing && browserRecognizer) {
          try { browserRecognizer.stop(); } catch (_) {}
        }
        if (!isDictationActive) {
          asrStatusBadge.textContent = 'Ready';
          asrStatusBadge.className = 'badge badge-subtle';
        }
      }
    }
  }

  if (kwsKeywordInput) {
    kwsKeywordInput.addEventListener('input', () => {
      updateKwsValidation();
      const kw = kwsKeywordInput.value.trim();
      if (kw) {
        client.setKwsKeyword(kw);
      }
    });
  }

  if (chkKwsEnable) {
    chkKwsEnable.addEventListener('change', (e) => {
      handleKwsToggle(e.target.checked);
    });
  }

  chkHighpass.addEventListener('change', (e) => {
    client.setHighPass(e.target.checked);
    logEvent('dsp.highpass', `80Hz Highpass De-rumble: ${e.target.checked ? 'ENABLED' : 'BYPASSED'}`);
  });

  chkAgc.addEventListener('change', (e) => {
    client.setAgc(e.target.checked);
    logEvent('dsp.agc', `Software AGC: ${e.target.checked ? 'ENABLED' : 'DISABLED'}`);
  });

  chkWebrtcNs.addEventListener('change', (e) => {
    capture.setWebRtcNoiseSuppression(e.target.checked);
    logEvent('dsp.webrtc_ns', `WebRTC Noise Suppression: ${e.target.checked ? 'ENABLED' : 'DISABLED'}`);
  });

  if (chkMonitor) {
    chkMonitor.addEventListener('change', (e) => {
      const enabled = e.target.checked;
      capture.setMonitoring(enabled);
      client.setMonitor(enabled);
      logEvent('dsp.monitor', `Headphone Audio Monitor: ${enabled ? 'ON (Live Ear)' : 'MUTED'}`);
    });
  }

  ttsSpeedSlider.addEventListener('input', (e) => {
    ttsSpeedVal.textContent = `${parseFloat(e.target.value).toFixed(1)}x`;
  });

  // Buttons
  btnCancel.addEventListener('click', () => {
    isDictationActive = false;
    isPttActive = false;
    isBrowserWakeActive = false;
    if (browserWakeResetTimer) clearTimeout(browserWakeResetTimer);
    if (isBrowserRecognizing && browserRecognizer) {
      try { browserRecognizer.stop(); } catch (_) {}
    }
    if (!ttsAudioPlayer.paused) {
      ttsAudioPlayer.pause();
      ttsAudioPlayer.currentTime = 0;
      client.sendPlaybackStatus(false);
    }
    client.cancelTurn();
    commandTranscriptBox.innerHTML = '<span class="placeholder-text">Session cancelled. Returned to IDLE.</span>';
    asrStatusBadge.textContent = 'Idle';
    asrStatusBadge.className = 'badge badge-subtle';
  });

  btnPttCommand.addEventListener('click', () => {
    isDictationActive = false;
    isPttActive = true;
    isBrowserWakeActive = false;
    if (browserWakeResetTimer) clearTimeout(browserWakeResetTimer);
    if (!ttsAudioPlayer.paused) {
      ttsAudioPlayer.pause();
      ttsAudioPlayer.currentTime = 0;
      client.sendPlaybackStatus(false);
    }
    if (!tabBtnCommands.classList.contains('active')) {
      tabBtnCommands.click();
    }

    const currAsr = asrEngineSelect ? asrEngineSelect.value : 'sherpa';
    if (currAsr === 'browser') {
      if (!SpeechRecognition || !browserRecognizer) {
        alert('Browser Web Speech API is not supported in this browser runtime (requires Chrome, Edge, or Safari).');
        return;
      }
      browserTargetMode = 'command';
      commandTranscriptBox.innerHTML = '<span style="color:#60a5fa;">[Push to talk] Listening for command...</span>';
      asrStatusBadge.textContent = 'Listening (PTT)';
      asrStatusBadge.className = 'badge badge-listen';
      const isContinuous = chkKwsEnable && chkKwsEnable.checked && (kwsKeywordInput ? kwsKeywordInput.value.trim().length > 0 : false);
      browserRecognizer.continuous = isContinuous;
      try {
        if (!isBrowserRecognizing) {
          browserRecognizer.start();
        }
      } catch (e) {
        console.warn('SpeechRecognition start error:', e);
      }
    } else {
      client.activateCommand();
      commandTranscriptBox.innerHTML = '<span style="color:#60a5fa;">[Push to talk] Listening for command...</span>';
      asrStatusBadge.textContent = 'Listening';
      asrStatusBadge.className = 'badge badge-listen';
    }
  });

  btnStartDictation.addEventListener('click', () => {
    isDictationActive = true;
    const currAsr = asrEngineSelect ? asrEngineSelect.value : 'sherpa';
    if (currAsr === 'browser') {
      if (!SpeechRecognition || !browserRecognizer) {
        alert('Browser Web Speech API is not supported in this browser runtime (requires Chrome, Edge, or Safari).');
        isDictationActive = false;
        return;
      }
      browserTargetMode = 'dictation';
      browserRecognizer.continuous = true;
      try {
        if (isBrowserRecognizing) {
          browserRecognizer.stop();
        }
        browserRecognizer.start();
      } catch (e) {
        console.warn('SpeechRecognition start error:', e);
      }
    } else {
      client.startDictation('scope_inspection_1');
      dictationLiveBox.innerHTML = '<span style="color:#60a5fa;">[Dictation Session Open] Speak your notes...</span>';
      dictationStatusBadge.textContent = 'Listening';
      dictationStatusBadge.className = 'badge badge-listen';
    }
  });

  btnStopDictation.addEventListener('click', () => {
    isDictationActive = false;
    if (isBrowserRecognizing && browserRecognizer) {
      try { browserRecognizer.stop(); } catch (_) {}
    }
    client.cancelTurn();
    dictationLiveBox.innerHTML = '<span class="placeholder-text">Dictation stopped.</span>';
    dictationStatusBadge.textContent = 'Idle';
    dictationStatusBadge.className = 'badge badge-subtle';
  });

  if (asrEngineSelect) {
    asrEngineSelect.addEventListener('change', (e) => {
      isDictationActive = false;
      const mode = e.target.value;
      if (mode === 'browser' && !SpeechRecognition) {
        alert('Browser Web Speech API is not supported in this browser runtime. Please use Chrome, Edge, or Safari.');
        asrEngineSelect.value = 'sherpa';
        return;
      }
      if (isBrowserRecognizing && browserRecognizer) {
        try { browserRecognizer.stop(); } catch (_) {}
      }
      client.setAsrEngine(mode);
      updateAsrEngineDisplay(mode);
      logEvent('asr.engine_select', `ASR Engine switched to: ${mode}`);
    });
  }

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
    client.injectText('', 'dictation');
  });

  // Kokoro Synthesize
  btnSynthesize.addEventListener('click', () => {
    const text = ttsInputText.value.trim();
    if (!text) return;
    client.speak(text, ttsVoiceSelect.value, parseFloat(ttsSpeedSlider.value));
  });

  // Custom Command Injections
  function handleCustomInject() {
    if (!customCmdInput) return;
    const text = customCmdInput.value.trim();
    if (!text) return;
    client.injectText(text, 'command');
  }

  if (btnCustomInject) {
    btnCustomInject.addEventListener('click', handleCustomInject);
  }

  if (customCmdInput) {
    customCmdInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        handleCustomInject();
      }
    });
  }

  // Quick Command Injections
  document.querySelectorAll('.btn-inject').forEach((btn) => {
    btn.addEventListener('click', () => {
      const text = btn.getAttribute('data-text');
      if (customCmdInput) {
        customCmdInput.value = text;
      }
      client.injectText(text, 'command');
    });
  });

  // Custom Dictation Injections
  function handleCustomDictInject(textToInject) {
    const raw = (textToInject !== undefined ? textToInject : (customDictInput ? customDictInput.value : '')).trim();
    if (!raw) return;
    const formatted = formatDictationPhrase(raw);
    dictationDraftText.value = appendDictationPhrase(dictationDraftText.value, raw);
    dictationLiveBox.innerHTML = `<strong>${escapeHtml(formatted)}</strong>`;
    draftWordCount.textContent = dictationDraftText.value.trim().split(/\s+/).filter(Boolean).length;
    dictationStatusBadge.textContent = 'Dictating';
    dictationStatusBadge.className = 'badge badge-ready';
    logEvent('dictation.inject', `Injected: "${formatted}"`);
    client.injectText(raw, 'dictation');
    if (customDictInput && textToInject === undefined) {
      customDictInput.value = '';
    }
  }

  if (btnCustomDictInject) {
    btnCustomDictInject.addEventListener('click', () => handleCustomDictInject());
  }

  if (customDictInput) {
    customDictInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        handleCustomDictInject();
      }
    });
  }

  // Quick Dictation Buttons
  document.querySelectorAll('.btn-inject-dict').forEach((btn) => {
    btn.addEventListener('click', () => {
      const text = btn.getAttribute('data-text');
      if (customDictInput) {
        customDictInput.value = text;
      }
      handleCustomDictInject(text);
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
    if (btnPttCommand) btnPttCommand.style.display = '';
    // Re-enable Wake-Word Detector (KWS) in commands mode
    if (kwsSettingsBox) {
      kwsSettingsBox.classList.remove('section-disabled');
      kwsSettingsBox.querySelectorAll('input, select, button').forEach((el) => { el.disabled = false; });
      updateKwsValidation();
    }
    if (kwsDisabledBadge) {
      kwsDisabledBadge.classList.add('hidden');
    }
    // If browser mode and KWS is enabled, resume continuous wake word spotting
    const currAsr = asrEngineSelect ? asrEngineSelect.value : 'sherpa';
    if (currAsr === 'browser' && chkKwsEnable && chkKwsEnable.checked && (kwsKeywordInput ? kwsKeywordInput.value.trim().length > 0 : false)) {
      browserTargetMode = 'command';
      isPttActive = false;
      try {
        if (!isBrowserRecognizing && browserRecognizer) {
          browserRecognizer.continuous = true;
          browserRecognizer.start();
        }
      } catch (_) {}
    }
  });

  tabBtnDictation.addEventListener('click', () => {
    tabBtnDictation.classList.add('active');
    tabBtnCommands.classList.remove('active');
    tabContentDictation.classList.add('active');
    tabContentCommands.classList.remove('active');
    if (btnPttCommand) btnPttCommand.style.display = 'none';
    // Visibly disable Wake-Word Detector (KWS) in dictation mode
    if (kwsSettingsBox) {
      kwsSettingsBox.classList.add('section-disabled');
      kwsSettingsBox.querySelectorAll('input, select, button').forEach((el) => { el.disabled = true; });
    }
    if (kwsDisabledBadge) {
      kwsDisabledBadge.classList.remove('hidden');
    }
    // Stop continuous command wake-word recognizer if running
    if (browserTargetMode === 'command' && isBrowserRecognizing && browserRecognizer && !isDictationActive) {
      try { browserRecognizer.stop(); } catch (_) {}
    }
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

  // Initialize source UI controls
  updateSourceControls(audioSourceSelect.value);

  const LANG_NAMES = {
    'en-us': 'American English (US)',
    'en-gb': 'British English (GB)',
    'es': 'Spanish (Español)',
    'fr': 'French (Français)',
    'hi': 'Hindi (हिन्दी)',
    'it': 'Italian (Italiano)',
    'ja': 'Japanese (日本語)',
    'pt-br': 'Portuguese (Brasil)',
    'zh': 'Mandarin Chinese (中文)',
  };

  function getLanguageGroup(v) {
    if (v.language && LANG_NAMES[v.language.toLowerCase()]) {
      return LANG_NAMES[v.language.toLowerCase()];
    }
    const prefix = v.id.slice(0, 2).toLowerCase();
    switch (prefix) {
      case 'af':
      case 'am': return 'American English (US)';
      case 'bf':
      case 'bm': return 'British English (GB)';
      case 'ef':
      case 'em': return 'Spanish (Español)';
      case 'ff': return 'French (Français)';
      case 'hf':
      case 'hm': return 'Hindi (हिन्दी)';
      case 'if':
      case 'im': return 'Italian (Italiano)';
      case 'jf':
      case 'jm': return 'Japanese (日本語)';
      case 'pf':
      case 'pm': return 'Portuguese (Brasil)';
      case 'zf':
      case 'zm': return 'Mandarin Chinese (中文)';
      default: return 'Other Voices';
    }
  }

  function populateModelsDropdown(models, activeModel) {
    if (!ttsModelSelect || !models || !models.length) return;
    const curVal = ttsModelSelect.value;
    ttsModelSelect.innerHTML = '';
    models.forEach((m) => {
      const opt = document.createElement('option');
      opt.value = m;
      let label = m;
      if (m.includes('v1_1')) label += ' (103 voices, Multi-lang)';
      else if (m.includes('v1_0')) label += ' (54 voices, Multi-lang)';
      else if (m.includes('v0_19')) label += ' (11 voices, English)';
      opt.textContent = label;
      ttsModelSelect.appendChild(opt);
    });

    if (activeModel) {
      let matched = false;
      const lowActive = activeModel.toLowerCase();
      // 1. Exact match
      for (const opt of ttsModelSelect.options) {
        if (opt.value && opt.value.toLowerCase() === lowActive) {
          ttsModelSelect.value = opt.value;
          matched = true;
          break;
        }
      }
      // 2. Specific variant match (FP16 or INT8)
      if (!matched && lowActive.includes('fp16')) {
        for (const opt of ttsModelSelect.options) {
          if (opt.value && opt.value.toLowerCase().includes('fp16')) {
            ttsModelSelect.value = opt.value;
            matched = true;
            break;
          }
        }
      }
      if (!matched && lowActive.includes('int8')) {
        for (const opt of ttsModelSelect.options) {
          if (opt.value && opt.value.toLowerCase().includes('int8')) {
            ttsModelSelect.value = opt.value;
            matched = true;
            break;
          }
        }
      }
      // 3. Fallback contains match (skipping FP16/INT8 options if active model is standard FP32)
      if (!matched) {
        for (const opt of ttsModelSelect.options) {
          const optLow = (opt.value || '').toLowerCase();
          const isVariantOpt = optLow.includes('fp16') || optLow.includes('int8');
          if (opt.value && !isVariantOpt && lowActive.includes(optLow)) {
            ttsModelSelect.value = opt.value;
            matched = true;
            break;
          }
        }
      }
    } else if (curVal && Array.from(ttsModelSelect.options).some(o => o.value === curVal)) {
      ttsModelSelect.value = curVal;
    }
  }

  function populateVoicesDropdown(voices) {
    if (!ttsVoiceSelect) return;
    if (ttsVoiceCount) {
      ttsVoiceCount.textContent = `${voices ? voices.length : 0}`;
    }
    if (!voices || !voices.length) return;
    const curVal = ttsVoiceSelect.value;
    ttsVoiceSelect.innerHTML = '';

    // Group voices by language / region
    const groups = new Map();
    voices.forEach((v) => {
      const groupName = getLanguageGroup(v);
      if (!groups.has(groupName)) {
        groups.set(groupName, []);
      }
      groups.get(groupName).push(v);
    });

    if (groups.size > 1) {
      for (const [groupName, groupVoices] of groups.entries()) {
        const optGroup = document.createElement('optgroup');
        optGroup.label = `${groupName} (${groupVoices.length})`;
        groupVoices.forEach((v) => {
          const opt = document.createElement('option');
          opt.value = v.id;
          const dot = v.gender === 'male' ? '🔵' : '🟣';
          opt.textContent = `${dot} ${v.id} - ${v.name}`;
          optGroup.appendChild(opt);
        });
        ttsVoiceSelect.appendChild(optGroup);
      }
    } else {
      voices.forEach((v) => {
        const opt = document.createElement('option');
        opt.value = v.id;
        const dot = v.gender === 'male' ? '🔵' : '🟣';
        opt.textContent = `${dot} ${v.id} - ${v.name}`;
        ttsVoiceSelect.appendChild(opt);
      });
    }

    if (curVal && Array.from(ttsVoiceSelect.options).some(o => o.value === curVal)) {
      ttsVoiceSelect.value = curVal;
    } else {
      // Find default flagship voice (e.g. af_heart or af_maple)
      const defaultOpt = Array.from(ttsVoiceSelect.options).find(o => o.textContent.includes('Default'));
      if (defaultOpt) {
        ttsVoiceSelect.value = defaultOpt.value;
      }
    }
  }

  if (ttsModelSelect) {
    ttsModelSelect.addEventListener('change', (e) => {
      const chosenModel = e.target.value;
      if (!chosenModel) return;
      client.setTtsModel(chosenModel);
      logEvent('tts.model_select', `Switched model -> ${chosenModel}`);

      // Optimistically fetch voices for the selected model
      fetch(`/api/voices?model=${encodeURIComponent(chosenModel)}`)
        .then(r => r.json())
        .then(data => {
          if (data && data.voices) {
            populateVoicesDropdown(data.voices);
          }
        })
        .catch(() => {});
    });
  }

  // Initial TTS model info fetch
  fetch('/api/voices')
    .then(r => r.json())
    .then(data => {
      if (data && data.model) {
        updateTtsModelDisplay(data.model, data.is_neural);
      }
      if (data && data.installed_models) {
        populateModelsDropdown(data.installed_models, data.model);
      }
      if (data && data.voices) {
        populateVoicesDropdown(data.voices);
      }
    })
    .catch(() => {});

  // Initialize KWS validation state
  updateKwsValidation();

  // Connect WebSocket
  client.connect();
});
