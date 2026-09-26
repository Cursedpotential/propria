
// Helper function to convert ArrayBuffer to Base64
export const toBase64 = (buffer: ArrayBuffer): Promise<string> => {
  return new Promise((resolve, reject) => {
    const blob = new Blob([buffer], { type: 'audio/wav' });
    const reader = new FileReader();
    reader.onload = () => {
      const dataUrl = reader.result as string;
      // remove the data URL prefix
      const base64 = dataUrl.split(',')[1];
      resolve(base64);
    };
    reader.onerror = (error) => reject(error);
    reader.readAsDataURL(blob);
  });
};

// Helper function to encode AudioBuffer to WAV format
const toWav = (audioBuffer: AudioBuffer): ArrayBuffer => {
  const numOfChan = audioBuffer.numberOfChannels;
  const
   len = audioBuffer.length * numOfChan * 2;
  const buffer = new ArrayBuffer(44 + len);
  const view = new DataView(buffer);
  const channels = [];
  let i;
  let sample;
  let offset = 0;
  let pos = 0;

  // write WAVE header
  setUint32(0x46464952); // "RIFF"
  setUint32(36 + len); // file length - 8
  setUint32(0x45564157); // "WAVE"

  setUint32(0x20746d66); // "fmt " chunk
  setUint32(16); // length = 16
  setUint16(1); // PCM (uncompressed)
  setUint16(numOfChan);
  setUint32(audioBuffer.sampleRate);
  setUint32(audioBuffer.sampleRate * 2 * numOfChan); // avg. bytes/sec
  setUint16(numOfChan * 2); // block-align
  setUint16(16); // 16-bit (hardcoded in this demo)

  setUint32(0x61746164); // "data" - chunk
  setUint32(len); // chunk length

  // write interleaved data
  for (i = 0; i < audioBuffer.numberOfChannels; i++) {
    channels.push(audioBuffer.getChannelData(i));
  }

  while (pos < audioBuffer.length) {
    for (i = 0; i < numOfChan; i++) {
      // interleave channels
      sample = Math.max(-1, Math.min(1, channels[i][pos])); // clamp
      sample = (0.5 + sample < 0 ? sample * 32768 : sample * 32767) | 0; // scale to 16-bit signed int
      view.setInt16(44 + offset, sample, true); // write 16-bit sample
      offset += 2;
    }
    pos++;
  }

  function setUint16(data: number) {
    view.setUint16(pos, data, true);
    pos += 2;
  }

  function setUint32(data: number) {
    view.setUint32(pos, data, true);
    pos += 4;
  }

  return buffer;
};

// Main function to extract audio from a video file
export const extractAudio = (
  videoFile: File,
  setLoadingPhase: (phase: string) => void
): Promise<{ audioBuffer: ArrayBuffer; mimeType: string }> => {
  return new Promise((resolve, reject) => {
    setLoadingPhase('extractingAudio');
    const reader = new FileReader();

    reader.onload = async (event) => {
      try {
        const audioContext = new (window.AudioContext || (window as any).webkitAudioContext)();
        const videoFileAsBuffer = event.target?.result as ArrayBuffer;
        
        const decodedAudioBuffer = await audioContext.decodeAudioData(videoFileAsBuffer);
        
        // For simplicity, we can resample to a common rate if needed, e.g., 16000
        // But for now, we'll use the original sample rate.
        
        const wavBuffer = toWav(decodedAudioBuffer);
        resolve({ audioBuffer: wavBuffer, mimeType: 'audio/wav' });

      } catch (e) {
        console.error("Web Audio API error:", e);
        reject(new Error("Could not decode audio from video file. The file may be corrupt or in an unsupported format."));
      }
    };

    reader.onerror = (error) => {
      reject(new Error("Failed to read video file for audio extraction: " + error));
    };

    reader.readAsArrayBuffer(videoFile);
  });
};
