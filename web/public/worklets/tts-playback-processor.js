/**
 * AudioWorklet Processor for real-time PCM playback.
 * Consumes Int16Array chunks from a FIFO queue, converts to Float32 for output.
 */
class TTSPlaybackProcessor extends AudioWorkletProcessor {
  constructor() {
    super();
    this._queue = [];
    this._readOffset = 0;

    this.port.onmessage = (event) => {
      if (event.data.type === 'clear') {
        this._queue.length = 0;
        this._readOffset = 0;
      } else if (event.data.pcm) {
        this._queue.push(event.data.pcm);
      }
    };
  }

  process(_inputs, outputs) {
    const output = outputs[0];
    if (!output || !output[0]) return true;

    const channel = output[0];
    let written = 0;

    while (written < channel.length) {
      if (this._queue.length === 0) break;

      const chunk = this._queue[0];
      const available = chunk.length - this._readOffset;
      const needed = channel.length - written;
      const toCopy = Math.min(available, needed);

      for (let i = 0; i < toCopy; i++) {
        // Int16 -> Float32 conversion
        channel[written + i] = chunk[this._readOffset + i] / 32768;
      }

      written += toCopy;
      this._readOffset += toCopy;

      if (this._readOffset >= chunk.length) {
        this._queue.shift();
        this._readOffset = 0;
      }
    }

    // Fill remaining with silence
    for (let i = written; i < channel.length; i++) {
      channel[i] = 0;
    }

    return true;
  }
}

registerProcessor('tts-playback-processor', TTSPlaybackProcessor);
