/**
 * Convert PCM Int16Array chunks to a WAV Blob.
 */
export function pcmToWav(chunks: Int16Array[], sampleRate: number): Blob {
  const totalSamples = chunks.reduce((sum, c) => sum + c.length, 0);
  const buffer = new ArrayBuffer(44 + totalSamples * 2);
  const view = new DataView(buffer);

  // RIFF header
  writeString(view, 0, 'RIFF');
  view.setUint32(4, 36 + totalSamples * 2, true);
  writeString(view, 8, 'WAVE');

  // fmt chunk
  writeString(view, 12, 'fmt ');
  view.setUint32(16, 16, true);        // chunk size
  view.setUint16(20, 1, true);         // PCM format
  view.setUint16(22, 1, true);         // mono
  view.setUint32(24, sampleRate, true);
  view.setUint32(28, sampleRate * 2, true); // byte rate
  view.setUint16(32, 2, true);         // block align
  view.setUint16(34, 16, true);        // bits per sample

  // data chunk
  writeString(view, 36, 'data');
  view.setUint32(40, totalSamples * 2, true);

  let offset = 44;
  for (const chunk of chunks) {
    for (let i = 0; i < chunk.length; i++) {
      view.setInt16(offset, chunk[i], true);
      offset += 2;
    }
  }

  return new Blob([buffer], { type: 'audio/wav' });
}

function writeString(view: DataView, offset: number, str: string) {
  for (let i = 0; i < str.length; i++) {
    view.setUint8(offset + i, str.charCodeAt(i));
  }
}
