import type { TTSRequest } from '../hooks';

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
  onPlaybackStart?: () => void;
  onPlaybackEnd?: () => void;
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

    const workletUrl = new URL('/worklets/tts-playback-processor.js', window.location.origin).href;
    await this.audioContext.audioWorklet.addModule(workletUrl);

    this.workletNode = new AudioWorkletNode(this.audioContext, 'tts-playback-processor');
    this.workletNode.connect(this.audioContext.destination);
  }

  async startStream(url: string, payload: TTSRequest, signal?: AbortSignal): Promise<void> {
    if (!this.audioContext || !this.workletNode) {
      throw new Error('Player not initialized. Call init() first.');
    }

    // 清空上一次的数据
    this.pcmChunks.length = 0;
    this.workletNode.port.postMessage({ type: 'clear' });
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

      let buffer = new Uint8Array();
      let hasFirstChunk = false;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        if (!hasFirstChunk) {
          this.firstChunkTime = performance.now();
          const ttfp = this.firstChunkTime - this.startTime;
          hasFirstChunk = true;
          this.setState('playing');
          this.onPlaybackStart?.();
          this.updateMetrics({ ttfp: Math.round(ttfp) });
        }

        // 合并到 buffer
        const newBuffer = new Uint8Array(buffer.length + value.length);
        newBuffer.set(buffer);
        newBuffer.set(value, buffer.length);
        buffer = newBuffer;

        // 按 2 字节对齐切分 Int16Array
        const alignedLength = Math.floor(buffer.length / 2) * 2;
        if (alignedLength >= 2) {
          const pcm = new Int16Array(buffer.buffer, buffer.byteOffset, alignedLength / 2);
          const chunk = new Int16Array(pcm);
          this.pcmChunks.push(chunk);
          this.workletNode!.port.postMessage({ pcm: chunk });

          // 保留未对齐的尾部字节
          const remaining = buffer.length - alignedLength;
          if (remaining > 0) {
            const leftover = new Uint8Array(remaining);
            leftover.set(buffer.subarray(alignedLength));
            buffer = leftover;
          } else {
            buffer = new Uint8Array();
          }
        }

        this.updateMetrics({ bytesReceived: this._metrics.bytesReceived + value.length });
      }

      // 处理尾部字节
      if (buffer.length >= 2) {
        const pcm = new Int16Array(buffer.buffer, buffer.byteOffset, Math.floor(buffer.length / 2));
        const chunk = new Int16Array(pcm);
        this.pcmChunks.push(chunk);
        this.workletNode!.port.postMessage({ pcm: chunk });
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

      // 等待播放结束
      const estimatedPlayTime = audioDuration * 1000;
      const playStart = this.firstChunkTime || this.startTime;
      const elapsed = performance.now() - playStart;
      const remaining = Math.max(0, estimatedPlayTime - elapsed);
      setTimeout(() => {
        if (this._state === 'completed') {
          this.onPlaybackEnd?.();
        }
      }, remaining + 200);
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
    this.workletNode?.port.postMessage({ type: 'clear' });
    this.setState('idle');
  }

  destroy(): void {
    this.stop();
    this.workletNode?.disconnect();
    this.audioContext?.close();
    this.audioContext = null;
    this.workletNode = null;
  }
}
