/**
 * AudioCapture: Handles browser microphone capture, 16kHz resampling,
 * WebRTC constraints, and real-time canvas visualizer.
 */
class AudioCapture {
  constructor(onChunkCallback) {
    this.onChunk = onChunkCallback;
    this.audioCtx = null;
    this.mediaStream = null;
    this.sourceNode = null;
    this.processorNode = null;
    this.analyserNode = null;
    this.isRecording = false;

    this.sampleRate = 16000;
    this.chunkSize = 480; // 30ms at 16kHz

    // Visualizer settings
    this.canvas = null;
    this.canvasCtx = null;
    this.animId = null;
    this.remoteAnimId = null;
    this.isRemote = false;
    this.remoteBuffer = new Float32Array(256);
  }

  setCanvas(canvasElement) {
    this.canvas = canvasElement;
    if (this.canvas) {
      this.canvasCtx = this.canvas.getContext('2d');
    }
  }

  async start(options = {}) {
    if (this.isRecording) return;
    this.setRemoteMode(false);

    const constraints = {
      audio: {
        channelCount: 1,
        sampleRate: 16000,
        echoCancellation: options.echoCancellation !== false,
        noiseSuppression: options.noiseSuppression !== false,
        autoGainControl: options.autoGainControl !== false,
      }
    };

    try {
      this.mediaStream = await navigator.mediaDevices.getUserMedia(constraints);
      this.audioCtx = new (window.AudioContext || window.webkitAudioContext)({
        sampleRate: 16000
      });

      this.sourceNode = this.audioCtx.createMediaStreamSource(this.mediaStream);
      this.analyserNode = this.audioCtx.createAnalyser();
      this.analyserNode.fftSize = 512;
      this.sourceNode.connect(this.analyserNode);

      // Create ScriptProcessor for raw PCM chunking
      this.processorNode = this.audioCtx.createScriptProcessor(512, 1, 1);
      this.analyserNode.connect(this.processorNode);
      this.processorNode.connect(this.audioCtx.destination);

      let bufferAccumulator = [];

      this.processorNode.onaudioprocess = (e) => {
        if (!this.isRecording) return;
        const inputData = e.inputBuffer.getChannelData(0);

        // Optional headphone monitoring for audio testing
        const outData = e.outputBuffer.getChannelData(0);
        if (this.isMonitoring) {
          for (let i = 0; i < inputData.length; i++) {
            outData[i] = inputData[i];
          }
        } else {
          outData.fill(0);
        }

        for (let i = 0; i < inputData.length; i++) {
          bufferAccumulator.push(inputData[i]);
          if (bufferAccumulator.length >= this.chunkSize) {
            const chunk = bufferAccumulator.slice(0, this.chunkSize);
            bufferAccumulator = bufferAccumulator.slice(this.chunkSize);

            // Convert Float32 [-1, 1] to Int16 PCM Little Endian
            const int16Buffer = new Int16Array(chunk.length);
            for (let j = 0; j < chunk.length; j++) {
              let s = Math.max(-1, Math.min(1, chunk[j]));
              int16Buffer[j] = s < 0 ? s * 0x8000 : s * 0x7FFF;
            }

            if (this.onChunk) {
              this.onChunk(int16Buffer.buffer);
            }
          }
        }
      };

      this.isRecording = true;
      this.startVisualizer();
      return true;
    } catch (err) {
      console.error('[AudioCapture] Mic access error:', err);
      throw err;
    }
  }

  stop() {
    this.isRecording = false;
    if (this.animId) {
      cancelAnimationFrame(this.animId);
      this.animId = null;
    }
    if (this.processorNode) {
      this.processorNode.disconnect();
      this.processorNode = null;
    }
    if (this.sourceNode) {
      this.sourceNode.disconnect();
      this.sourceNode = null;
    }
    if (this.mediaStream) {
      this.mediaStream.getTracks().forEach(t => t.stop());
      this.mediaStream = null;
    }
    if (this.audioCtx) {
      this.audioCtx.close();
      this.audioCtx = null;
    }
    if (!this.isRemote) {
      this.clearVisualizer();
    }
  }

  setWebRtcNoiseSuppression(enabled) {
    if (this.mediaStream) {
      this.mediaStream.getAudioTracks().forEach((track) => {
        track.applyConstraints({ noiseSuppression: enabled }).catch((err) => {
          console.warn('[AudioCapture] Could not update noiseSuppression constraint:', err);
        });
      });
    }
  }

  setMonitoring(enabled) {
    this.isMonitoring = enabled;
  }

  startVisualizer() {
    if (!this.canvas || !this.analyserNode) return;

    const bufferLength = this.analyserNode.frequencyBinCount;
    const timeData = new Uint8Array(bufferLength);
    const freqData = new Uint8Array(bufferLength);

    const draw = () => {
      this.animId = requestAnimationFrame(draw);
      this.analyserNode.getByteTimeDomainData(timeData);
      this.analyserNode.getByteFrequencyData(freqData);

      const ctx = this.canvasCtx;
      const width = this.canvas.width;
      const height = this.canvas.height;

      // Dark background with slight decay
      ctx.fillStyle = '#06090d';
      ctx.fillRect(0, 0, width, height);

      // Frequency spectrum bars (bottom half)
      const barWidth = (width / bufferLength) * 2.5;
      let barX = 0;
      for (let i = 0; i < bufferLength; i++) {
        const barHeight = (freqData[i] / 255) * (height / 2);
        ctx.fillStyle = 'rgba(6, 182, 212, 0.35)';
        ctx.fillRect(barX, height - barHeight, barWidth, barHeight);
        barX += barWidth + 1;
      }

      // Time-domain oscilloscope wave (center)
      ctx.lineWidth = 2;
      ctx.strokeStyle = '#38bdf8';
      ctx.beginPath();

      const sliceWidth = width / bufferLength;
      let x = 0;

      for (let i = 0; i < bufferLength; i++) {
        const v = timeData[i] / 128.0;
        const y = (v * height) / 2;

        if (i === 0) {
          ctx.moveTo(x, y);
        } else {
          ctx.lineTo(x, y);
        }
        x += sliceWidth;
      }

      ctx.lineTo(width, height / 2);
      ctx.stroke();
    };

    draw();
  }

  setRemoteMode(enabled) {
    this.isRemote = enabled;
    if (enabled) {
      this.startRemoteVisualizer();
    } else {
      if (this.remoteAnimId) {
        cancelAnimationFrame(this.remoteAnimId);
        this.remoteAnimId = null;
      }
      if (!this.isRecording) {
        this.clearVisualizer();
      }
    }
  }

  pushRemoteWave(samples, passedGate = true) {
    if (!this.remoteBuffer) {
      this.remoteBuffer = new Float32Array(256);
    }
    this.remoteGatePassed = passedGate;
    const n = Math.min(samples.length, 256);
    this.remoteBuffer.copyWithin(0, n);
    for (let i = 0; i < n; i++) {
      this.remoteBuffer[256 - n + i] = samples[i];
    }
  }

  startRemoteVisualizer() {
    if (this.remoteAnimId) {
      cancelAnimationFrame(this.remoteAnimId);
    }
    if (!this.canvas) return;

    const draw = () => {
      this.remoteAnimId = requestAnimationFrame(draw);
      const ctx = this.canvasCtx;
      const width = this.canvas.width;
      const height = this.canvas.height;

      ctx.fillStyle = '#06090d';
      ctx.fillRect(0, 0, width, height);

      const buf = this.remoteBuffer;
      const len = buf.length;
      const isGateOpen = this.remoteGatePassed !== false;

      // Frequency spectrum bars (bottom half)
      const numBars = 32;
      const barWidth = width / numBars;
      for (let b = 0; b < numBars; b++) {
        const start = Math.floor((b / numBars) * len);
        const end = Math.floor(((b + 1) / numBars) * len);
        let energy = 0;
        for (let k = start; k < end; k++) {
          energy += Math.abs(buf[k]);
        }
        energy = (energy / Math.max(1, end - start)) * 4.0;
        if (energy > 1) energy = 1;

        const barHeight = energy * (height * 0.45);
        ctx.fillStyle = isGateOpen ? 'rgba(6, 182, 212, 0.45)' : 'rgba(100, 116, 139, 0.25)';
        ctx.fillRect(b * barWidth, height - barHeight, barWidth - 1, barHeight);
      }

      // Time-domain wave (center)
      ctx.lineWidth = 2;
      ctx.strokeStyle = isGateOpen ? '#38bdf8' : '#64748b'; // Cyan when Gate Open, Muted Slate when Gated
      ctx.beginPath();
      const sliceWidth = width / len;
      let x = 0;

      for (let i = 0; i < len; i++) {
        const v = (buf[i] + 1.0) / 2.0;
        const y = v * height;

        if (i === 0) {
          ctx.moveTo(x, y);
        } else {
          ctx.lineTo(x, y);
        }
        x += sliceWidth;
      }
      ctx.stroke();
    };

    draw();
  }

  clearVisualizer() {
    if (!this.canvasCtx || !this.canvas) return;
    this.canvasCtx.fillStyle = '#06090d';
    this.canvasCtx.fillRect(0, 0, this.canvas.width, this.canvas.height);
    this.canvasCtx.lineWidth = 1;
    this.canvasCtx.strokeStyle = '#1e293b';
    this.canvasCtx.beginPath();
    this.canvasCtx.moveTo(0, this.canvas.height / 2);
    this.canvasCtx.lineTo(this.canvas.width, this.canvas.height / 2);
    this.canvasCtx.stroke();
  }
}
