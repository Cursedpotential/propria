
import React, { useCallback, useState } from 'react';
import { StagedVideo, VideoStatus, AnalysisReport, AppSettings } from '../types';
import { useVideoAnalysis } from '../hooks/useVideoAnalysis';
import { FileDropzone } from './FileDropzone';
import { Loader } from './Loader';
import { TrashIcon } from './icons/TrashIcon';
import { CloudIcon } from './icons/CloudIcon';
import { UploadIcon } from './icons/UploadIcon';

interface VideoProcessorProps {
    stagedVideos: StagedVideo[];
    setStagedVideos: React.Dispatch<React.SetStateAction<StagedVideo[]>>;
    settings: AppSettings;
}

type IngestionMode = 'local' | 'cloud';

export const VideoProcessor: React.FC<VideoProcessorProps> = ({ stagedVideos, setStagedVideos, settings }) => {
  const { isProcessing, currentPhase, analyzeSingleVideo, processVideoViaBackend } = useVideoAnalysis(settings);
  const [ingestionMode, setIngestionMode] = useState<IngestionMode>('local');
  const [cloudUrls, setCloudUrls] = useState('');

  const handleFilesSelect = useCallback((files: File[]) => {
    const newVideos: StagedVideo[] = files
      .filter(file => file.type.startsWith('video/'))
      .map(file => ({
        id: `${file.name}-${Date.now()}`,
        file,
        videoUrl: URL.createObjectURL(file),
        status: 'Pending',
        description: '',
        category: 'Current Case Incident',
        location: 'Unknown',
        isSignificant: false,
        manipulationPattern: 'N/A',
        evidenceStrength: 'Moderate',
      }));
    setStagedVideos(prev => [...prev, ...newVideos]);
  }, [setStagedVideos]);

  const updateVideoStatus = (id: string, status: VideoStatus, report?: AnalysisReport) => {
    setStagedVideos(prev => prev.map(v => {
        if (v.id === id) {
            const updatedVideo = { ...v, status, report };
            if (report) {
                updatedVideo.description = report.summary;
            }
            return updatedVideo;
        }
        return v;
    }));
  };

  const handleRunLocalAnalysis = async () => {
    const pendingVideos = stagedVideos.filter(v => v.status === 'Pending');
    for (const video of pendingVideos) {
      try {
        updateVideoStatus(video.id, 'Processing');
        const report = await analyzeSingleVideo(video.file);
        updateVideoStatus(video.id, 'Analyzed', report);
      } catch (e: any) {
        console.error(`Failed to analyze ${video.file.name}:`, e);
        updateVideoStatus(video.id, 'Error');
      }
    }
  };

  const handleRunCloudAnalysis = async () => {
    const urls = cloudUrls.split('\n').filter(url => url.trim() !== '');
    setCloudUrls(''); // Clear textarea
    for (const url of urls) {
        const fakeFileName = url.substring(url.lastIndexOf('/') + 1);
        const videoId = `${fakeFileName}-${Date.now()}`;
        
        const placeholderVideo: StagedVideo = {
            id: videoId,
            file: new File([], fakeFileName),
            videoUrl: url,
            status: 'Processing',
            description: `Processing from cloud: ${fakeFileName}`,
            category: 'Current Case Incident',
            location: 'Unknown',
            isSignificant: false,
            manipulationPattern: 'N/A',
            evidenceStrength: 'Moderate',
        };
        setStagedVideos(prev => [...prev, placeholderVideo]);

        try {
            const report = await processVideoViaBackend(url);
            updateVideoStatus(videoId, 'Analyzed', report);
        } catch (e: any) {
            console.error(`Failed to analyze ${fakeFileName}:`, e);
            updateVideoStatus(videoId, 'Error');
        }
    }
  };

  const handleClearAll = () => {
    if (window.confirm("Are you sure you want to clear the entire ingestion list? This will not affect verified data.")) {
        stagedVideos.forEach(v => {
            if (v.videoUrl.startsWith('blob:')) {
                URL.revokeObjectURL(v.videoUrl);
            }
        });
        setStagedVideos([]);
    }
  };

  const pendingCount = stagedVideos.filter(v => v.status === 'Pending').length;

  if (stagedVideos.length === 0) {
    return <FileDropzone onFilesSelect={handleFilesSelect} />;
  }

  return (
    <div className="max-w-4xl mx-auto">
        <div className="flex justify-between items-center mb-4">
            <h2 className="text-xl font-bold text-slate-100">Ingestion Workbench</h2>
            <button onClick={handleClearAll} title="Clear All" className="p-2 text-slate-300 hover:text-white hover:bg-slate-700 rounded-md"><TrashIcon className="w-5 h-5" /></button>
        </div>

        <div className="flex border-b border-slate-600 mb-6">
            <button onClick={() => setIngestionMode('local')} className={`flex items-center gap-2 px-4 py-2 text-sm font-medium border-b-2 ${ingestionMode === 'local' ? 'border-cyan-400 text-white' : 'border-transparent text-slate-300 hover:text-white'}`}>
                <UploadIcon className="w-5 h-5" /> Local Upload
            </button>
            <button onClick={() => setIngestionMode('cloud')} className={`flex items-center gap-2 px-4 py-2 text-sm font-medium border-b-2 ${ingestionMode === 'cloud' ? 'border-cyan-400 text-white' : 'border-transparent text-slate-300 hover:text-white'}`}>
                <CloudIcon className="w-5 h-5" /> Cloud Ingestion
            </button>
        </div>

        {ingestionMode === 'local' && <FileDropzone onFilesSelect={handleFilesSelect} />}
        
        {ingestionMode === 'cloud' && (
            <div className="space-y-4">
                <p className="text-sm text-slate-300">Enter direct URLs to video files stored in a cloud bucket (e.g., GCS, R2), one URL per line. The app will simulate a backend process to analyze them.</p>
                <textarea 
                    value={cloudUrls}
                    onChange={e => setCloudUrls(e.target.value)}
                    rows={5}
                    placeholder="https://your-bucket.com/video1.mp4&#10;https://your-bucket.com/video2.mp4"
                    className="w-full bg-slate-900 border-slate-600 rounded-md shadow-sm p-2 focus:outline-none focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm"
                />
                <button onClick={handleRunCloudAnalysis} disabled={isProcessing || !cloudUrls.trim()} className="w-full bg-blue-600 hover:bg-blue-500 text-white font-bold py-3 px-4 rounded-lg disabled:bg-slate-600">
                    Process Cloud Videos
                </button>
            </div>
        )}

        <div className="mt-6">
            <h3 className="text-lg font-semibold text-slate-200 mb-3">Processing Queue</h3>
            <div className="space-y-3 max-h-60 overflow-y-auto pr-2">
                {stagedVideos.map((video) => (
                    <div key={video.id} className="flex items-center gap-4 p-3 rounded-md bg-slate-700/50">
                        <div className="flex-grow">
                            <p className="text-sm font-medium text-slate-100 truncate">{video.file.name}</p>
                            <p className={`text-xs font-semibold ${
                                video.status === 'Analyzed' ? 'text-green-400' : 
                                video.status === 'Verified' ? 'text-cyan-400' :
                                video.status === 'Error' ? 'text-red-400' : 
                                'text-amber-400'
                            }`}>{video.status}</p>
                        </div>
                    </div>
                ))}
            </div>
            {isProcessing && <div className="mt-4"><Loader phase={currentPhase} /></div>}
            <div className="mt-6">
                {pendingCount > 0 && (
                    <button onClick={handleRunLocalAnalysis} disabled={isProcessing} className="w-full bg-cyan-600 hover:bg-cyan-500 text-white font-bold py-3 px-4 rounded-lg disabled:bg-slate-600">
                        Analyze {pendingCount} Pending Local Videos
                    </button>
                )}
                {stagedVideos.length > 0 && pendingCount === 0 && !isProcessing && (
                    <p className="text-center text-slate-400">All videos have been processed. Go to the "Review & Verify" tab to continue.</p>
                )}
            </div>
        </div>
    </div>
  );
};
