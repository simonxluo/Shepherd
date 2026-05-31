import type { TTSRequest } from '../types';

export interface TTSStreamMetrics {
  ttfp: number | null;              // 首包延迟 (ms)
  rtf: number | null;               // 实时率
  audioDuration: number;            // 音频时长 (s)
  speedMultiplier: number;          // 速度倍率
  bytesReceived: number;            // 已接收字节数
}

export type StreamState = 'idle' | 'streaming' | 'playing' | 'completed' | 'error';

export class StreamAudioPlayer {
  private audioContext: AudioContext | null = null;
  private workletNode: AudioWorkletNode | null = null;
  private scriptProcessorNode: ScriptProcessorNode | null = null;
  private useWorklet = true;
  // ScriptProcessor fallback 内部队列
  private _fallbackQueue: Int16Array[] = [];
  private _fallbackReadOffset = 0;
  private abortController: AbortController | null = null;
  private sampleRate: number;
  private startTime = 0;
  private firstChunkTime = 0;
  private _state: StreamState = 'idle';
  private _metrics: TTSStreamMetrics = {
    ttfp: null,
    rtf: null,
    audioDuration: 0,
    speedMultiplier: 0,
    bytesReceived: 0,
  };

  readonly pcmChunks: Int16Array[] = [];

  onMetricsUpdate?: (metrics: TTSStreamMetrics) => void;
  onError?: (error: Error) => void;
  onStateChange?: (state: StreamState) => void;

  constructor(sampleRate = 48000) {
    this.sampleRate = sampleRate;
  }

  get state() { return this._state; }
  get metrics() { return this._metrics; }

  private setState(state: StreamState) {
    this._state = state;
    this.onStateChange?.(state);
  }

  private updateMetrics(partial: Partial<TTSStreamMetrics>) {
    this._metrics = { ...this._metrics, ...partial };
    this.onMetricsUpdate?.(this._metrics);
  }

  async init(): Promise<void> {
    this.audioContext = new AudioContext({ sampleRate: this.sampleRate });

    // 恢复可能被浏览器策略暂停的 AudioContext
    if (this.audioContext.state === 'suspended') {
      await this.audioContext.resume();
    }

    // 检测 AudioWorklet 支持并加载 worklet 模块
    if (this.audioContext.audioWorklet) {
      try {
        const workletUrl = new URL('/worklets/tts-playback-processor.js', window.location.origin).href;
        await this.audioContext.audioWorklet.addModule(workletUrl);
        this.workletNode = new AudioWorkletNode(this.audioContext, 'tts-playback-processor');
        this.workletNode.connect(this.audioContext.destination);
        this.useWorklet = true;
        return;
      } catch {
        // worklet 加载失败，降级到 ScriptProcessorNode
      }
    }

    // AudioWorklet 不可用，使用 ScriptProcessorNode 降级方案
    this.useWorklet = false;
    this.scriptProcessorNode = this.audioContext.createScriptProcessor(4096, 0, 1);
    this.scriptProcessorNode.onaudioprocess = (event: AudioProcessingEvent) => {
      const output = event.outputBuffer.getChannelData(0);
      let written = 0;

      while (written < output.length) {
        if (this._fallbackQueue.length === 0) break;

        const chunk = this._fallbackQueue[0];
        const available = chunk.length - this._fallbackReadOffset;
        const needed = output.length - written;
        const toCopy = Math.min(available, needed);

        for (let i = 0; i < toCopy; i++) {
          output[written + i] = chunk[this._fallbackReadOffset + i] / 32768;
        }

        written += toCopy;
        this._fallbackReadOffset += toCopy;

        if (this._fallbackReadOffset >= chunk.length) {
          this._fallbackQueue.shift();
          this._fallbackReadOffset = 0;
        }
      }

      for (let i = written; i < output.length; i++) {
        output[i] = 0;
      }
    };
    this.scriptProcessorNode.connect(this.audioContext.destination);
  }

  private sendChunk(pcm: Int16Array): void {
    if (this.useWorklet && this.workletNode) {
      this.workletNode.port.postMessage({ pcm });
    } else {
      this._fallbackQueue.push(new Int16Array(pcm));
    }
  }

  private clearBuffer(): void {
    if (this.useWorklet && this.workletNode) {
      this.workletNode.port.postMessage({ type: 'clear' });
    } else {
      this._fallbackQueue.length = 0;
      this._fallbackReadOffset = 0;
    }
  }

  async startStream(url: string, payload: TTSRequest, signal?: AbortSignal): Promise<void> {
    if (!this.audioContext || (this.useWorklet && !this.workletNode) || (!this.useWorklet && !this.scriptProcessorNode)) {
      throw new Error('Player not initialized. Call init() first.');
    }

    // 清空上一次的数据
    this.pcmChunks.length = 0;
    this.clearBuffer();
    this.updateMetrics({
      ttfp: null, rtf: null, audioDuration: 0,
      speedMultiplier: 0, bytesReceived: 0,
    });

    this.abortController = new AbortController();
    const combinedSignal = signal
      ? AbortSignal.any([this.abortController.signal, signal])
      : this.abortController.signal;

    this.setState('streaming');
    this.startTime = performance.now();

    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
        signal: combinedSignal,
      });

      if (!response.ok) {
        const err = await response.json().catch(() => ({ error: { message: response.statusText } }));
        throw new Error(err.error?.message || `TTS 请求失败 (${response.status})`);
      }

      const reader = response.body?.getReader();
      if (!reader) throw new Error('无法获取响应流');

      let chunks: Uint8Array[] = [];
      let totalBytes = 0;
      let leftover = new Uint8Array(0);
      let leftoverBytes = 0;
      let hasFirstChunk = false;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        if (!hasFirstChunk) {
          this.firstChunkTime = performance.now();
          const ttfp = this.firstChunkTime - this.startTime;
          hasFirstChunk = true;
          this.setState('playing');
          this.updateMetrics({ ttfp: Math.round(ttfp) });
        }

        chunks.push(value);
        totalBytes += value.length;

        // 处理累积数据：合并 leftover + 所有 chunks，按 2 字节对齐
        const allBytes = leftoverBytes + totalBytes;
        const alignedLength = Math.floor(allBytes / 2) * 2;

        if (alignedLength >= 2 && alignedLength > leftoverBytes) {
          const merged = new Uint8Array(allBytes);
          let offset = 0;
          merged.set(leftover, offset);
          offset += leftoverBytes;
          for (const chunk of chunks) {
            merged.set(chunk, offset);
            offset += chunk.length;
          }

          const pcm = new Int16Array(merged.buffer, merged.byteOffset, alignedLength / 2);
          const pcmCopy = new Int16Array(pcm);
          this.pcmChunks.push(pcmCopy);
          this.sendChunk(pcmCopy);

          // 保留未对齐的尾部
          const remaining = allBytes - alignedLength;
          if (remaining > 0) {
            leftover = new Uint8Array(remaining);
            leftover.set(merged.subarray(alignedLength));
          } else {
            leftover = new Uint8Array(0);
          }
          leftoverBytes = remaining;
          chunks = [];
          totalBytes = 0;
        }

        this.updateMetrics({ bytesReceived: this._metrics.bytesReceived + value.length });
      }

      // 处理最终 leftover
      if (leftoverBytes >= 2) {
        const pcm = new Int16Array(leftover.buffer, leftover.byteOffset, Math.floor(leftoverBytes / 2));
        const pcmCopy = new Int16Array(pcm);
        this.pcmChunks.push(pcmCopy);
        this.sendChunk(pcmCopy);
      }

      // 计算最终指标
      const totalSamples = this.pcmChunks.reduce((sum, c) => sum + c.length, 0);
      const audioDuration = totalSamples / this.sampleRate;
      const wallTime = (performance.now() - this.startTime) / 1000;
      const rtf = wallTime > 0 ? audioDuration / wallTime : null;

      this.updateMetrics({
        audioDuration: Math.round(audioDuration * 100) / 100,
        rtf: rtf !== null ? Math.round(rtf * 100) / 100 : null,
        speedMultiplier: wallTime > 0 ? Math.round((audioDuration / wallTime) * 100) / 100 : 0,
      });

      this.setState('completed');
    } catch (err) {
      if ((err as Error).name === 'AbortError') {
        this.setState('idle');
        return;
      }
      this.setState('error');
      this.onError?.(err as Error);
    }
  }

  stop(): void {
    this.abortController?.abort();
    this.clearBuffer();
    this.setState('idle');
  }

  destroy(): void {
    this.stop();
    if (this.useWorklet) {
      this.workletNode?.disconnect();
    } else {
      this.scriptProcessorNode?.disconnect();
    }
    this.audioContext?.close();
    this.audioContext = null;
    this.workletNode = null;
    this.scriptProcessorNode = null;
  }
}
