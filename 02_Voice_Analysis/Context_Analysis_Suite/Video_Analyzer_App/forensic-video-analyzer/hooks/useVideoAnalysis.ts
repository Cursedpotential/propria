
import { useState, useCallback } from 'react';
import { GoogleGenAI, Type } from '@google/genai';
import { AnalysisReport, LoadingPhase, StagedVideo, ForensicAnalysis, AppSettings } from '../types';
import { extractAudio, toBase64 } from '../utils/audio';

const FRAME_COUNT = 8; // Number of frames to extract

const extractFrames = (videoFile: File, onProgress: (phase: LoadingPhase) => void): Promise<{ frames: string[]; timestamps: number[] }> => {
    return new Promise((resolve, reject) => {
      onProgress('extractingFrames');
      const video = document.createElement('video');
      video.preload = 'metadata';
      video.src = URL.createObjectURL(videoFile);
      const canvas = document.createElement('canvas');
      const context = canvas.getContext('2d');

      const frames: string[] = [];
      const timestamps: number[] = [];
      let framesExtracted = 0;

      video.onloadedmetadata = () => {
        canvas.width = video.videoWidth;
        canvas.height = video.videoHeight;
        const duration = video.duration;
        if (duration < 1) {
            reject(new Error("Video is too short for analysis."));
            return;
        }

        const captureFrame = (frameIndex: number) => {
          const time = (duration / (FRAME_COUNT + 1)) * (frameIndex + 1);
          video.currentTime = time;
        };

        video.onseeked = () => {
          if (framesExtracted < FRAME_COUNT && context) {
            context.drawImage(video, 0, 0, video.videoWidth, video.videoHeight);
            const dataUrl = canvas.toDataURL('image/jpeg', 0.8);
            frames.push(dataUrl.split(',')[1]);
            timestamps.push(video.currentTime);
            framesExtracted++;
            if (framesExtracted < FRAME_COUNT) {
              captureFrame(framesExtracted);
            } else {
              URL.revokeObjectURL(video.src);
              resolve({ frames, timestamps });
            }
          }
        };
        
        captureFrame(0);
      };

      video.onerror = () => {
        reject(new Error('Failed to load or process video file.'));
        URL.revokeObjectURL(video.src);
      };
    });
  };

export const useVideoAnalysis = (settings: AppSettings) => {
  const [isProcessing, setIsProcessing] = useState(false);
  const [currentPhase, setCurrentPhase] = useState<LoadingPhase>('idle');

  const analyzeSingleVideo = useCallback(async (videoFile: File): Promise<AnalysisReport> => {
    if (!settings.apiKeys.google) throw new Error('Google API Key is not set in Settings.');
    
    const ai = new GoogleGenAI({ apiKey: settings.apiKeys.google, vertexai: true });
    
    setIsProcessing(true);
    setCurrentPhase('initializing');

    try {
      const [frameData, audioData] = await Promise.all([
        extractFrames(videoFile, setCurrentPhase),
        extractAudio(videoFile, (phase) => setCurrentPhase(phase as LoadingPhase))
      ]);
      
      const { frames, timestamps } = frameData;
      const { audioBuffer, mimeType } = audioData;
      const audioBase64 = await toBase64(audioBuffer);

      setCurrentPhase('analyzing');
      
      const prompt = `Perform a "surface-level" analysis...`; // Prompt remains the same

      const response = await ai.models.generateContent({
        model: settings.modelPreferences.basic, // Use selected basic model
        contents: { role: 'user', parts: [
          ...frames.map(data => ({ inlineData: { mimeType: 'image/jpeg', data } })),
          { inlineData: { mimeType, data: audioBase64 } },
          { text: prompt },
        ]},
        config: {
          systemInstruction: 'You are a naive observer AI...', // System instruction remains the same
          responseMimeType: 'application/json',
          responseSchema: { /* Schema remains the same */
            type: Type.OBJECT,
            properties: {
              summary: { type: Type.STRING }, entities: { type: Type.ARRAY, items: { type: Type.OBJECT, properties: { name: { type: Type.STRING }, description: { type: Type.STRING } } } },
              moodAnalysis: { type: Type.OBJECT, properties: { overallMood: { type: Type.STRING }, confidence: { type: Type.NUMBER }, bodyLanguage: { type: Type.STRING }, emotionalConsistency: { type: Type.STRING } } },
              events: { type: Type.ARRAY, items: { type: Type.OBJECT, properties: { timestamp: { type: Type.STRING }, description: { type: Type.STRING } } } },
              transcript: { type: Type.ARRAY, items: { type: Type.OBJECT, properties: { speaker: { type: Type.STRING }, text: { type: Type.STRING }, timestamp: { type: Type.STRING } } } }
            },
          },
        },
      });

      setCurrentPhase('compilingReport');
      return JSON.parse(response.text.trim()) as AnalysisReport;
    } finally {
      setIsProcessing(false);
      setCurrentPhase('done');
    }
  }, [settings]);

  const processVideoViaBackend = useCallback(async (videoUrl: string): Promise<AnalysisReport> => {
    // This function would use settings.apiKeys.openai or other keys
    // to make a call to your Vercel backend.
    console.log("Simulating backend call with settings:", settings);
    setIsProcessing(true);
    setCurrentPhase('analyzing');
    await new Promise(resolve => setTimeout(resolve, 3000));
    const mockReport: AnalysisReport = { /* ... mock data ... */ };
    setIsProcessing(false);
    setCurrentPhase('done');
    return mockReport;
  }, [settings]);

  const runAdvancedAnalysis = useCallback(async (video: StagedVideo, speakerMap: Record<string, string>): Promise<ForensicAnalysis> => {
    if (!settings.apiKeys.google) throw new Error('Google API Key is not set in Settings.');
    if (!video.report) throw new Error('Cannot run advanced analysis without a basic report.');

    const ai = new GoogleGenAI({ apiKey: settings.apiKeys.google, vertexai: true });

    setIsProcessing(true);
    setCurrentPhase('advancedAnalysis');

    const context = `CONTEXT FOR FORENSIC ANALYSIS...`; // Context prompt remains the same
    const prompt = `Based on the provided verified context...`; // Advanced prompt remains the same

    try {
        const response = await ai.models.generateContent({
            model: settings.modelPreferences.advanced, // Use selected advanced model
            contents: { role: 'user', parts: [{ text: context }, { text: prompt }] },
            config: {
                systemInstruction: 'You are a senior forensic psychologist...', // System instruction remains the same
                responseMimeType: 'application/json',
                responseSchema: { /* Schema remains the same */
                    type: Type.OBJECT,
                    properties: {
                        sourceType: { type: Type.STRING, enum: ['Surveillance Video'] }, sourceFileRef: { type: Type.STRING },
                        deceptionScore: { type: Type.NUMBER }, sentiment: { type: Type.STRING, enum: ['Hostile', 'Neutral', 'Supportive', 'Distressed', 'Deceptive'] },
                        microExpressions: { type: Type.ARRAY, items: { type: Type.STRING } }, voiceStressAnalysis: { type: Type.STRING },
                        transcriptExcerpt: { type: Type.STRING }, processedAt: { type: Type.NUMBER }
                    },
                    required: ['sourceType', 'deceptionScore', 'sentiment', 'processedAt']
                }
            }
        });
        const result = JSON.parse(response.text.trim()) as ForensicAnalysis;
        result.sourceFileRef = video.file.name;
        result.processedAt = Date.now();
        return result;
    } finally {
        setIsProcessing(false);
        setCurrentPhase('done');
    }
  }, [settings]);

  return { isProcessing, currentPhase, analyzeSingleVideo, runAdvancedAnalysis, processVideoViaBackend };
};
