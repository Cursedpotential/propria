
import React, { useState, useMemo, useRef } from 'react';
import { StagedVideo, AppSettings } from '../types';
import { EditableSpeaker } from './EditableSpeaker';
import { useVideoAnalysis } from '../hooks/useVideoAnalysis';
import { Loader } from './Loader';
import { BrainIcon } from './icons/BrainIcon';
import { InfoIcon } from './icons/InfoIcon';

const parseTimestamp = (timestamp: string): number => {
    return parseFloat(timestamp.replace('s', ''));
}

const EditableField: React.FC<{label: string, value: string, onChange: (value: string) => void, type?: 'input' | 'textarea'}> = ({ label, value, onChange, type = 'input' }) => (
    <div>
        <label className="block text-sm font-medium text-slate-300 mb-1">{label}</label>
        {type === 'input' ? (
            <input type="text" value={value} onChange={e => onChange(e.target.value)} className="w-full bg-slate-900 border-slate-600 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm" />
        ) : (
            <textarea value={value} onChange={e => onChange(e.target.value)} rows={4} className="w-full bg-slate-900 border-slate-600 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm" />
        )}
    </div>
);

const SelectField: React.FC<{label: string, value: string, onChange: (value: any) => void, options: string[]}> = ({ label, value, onChange, options }) => (
    <div>
        <label className="block text-sm font-medium text-slate-300 mb-1">{label}</label>
        <select value={value} onChange={e => onChange(e.target.value)} className="w-full bg-slate-900 border-slate-600 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm">
            {options.map(opt => <option key={opt} value={opt}>{opt}</option>)}
        </select>
    </div>
);

export const ReviewAndVerify: React.FC<{
    stagedVideos: StagedVideo[];
    setStagedVideos: React.Dispatch<React.SetStateAction<StagedVideo[]>>;
    speakerMap: Record<string, string>;
    setSpeakerMap: React.Dispatch<React.SetStateAction<Record<string, string>>>;
    settings: AppSettings;
}> = ({ stagedVideos, setStagedVideos, speakerMap, setSpeakerMap, settings }) => {
    const [selectedVideoId, setSelectedVideoId] = useState<string | null>(null);
    const videoRef = useRef<HTMLVideoElement>(null);
    const { isProcessing, currentPhase, runAdvancedAnalysis } = useVideoAnalysis(settings);

    const analyzableVideos = useMemo(() => stagedVideos.filter(v => v.status === 'Analyzed' || v.status === 'Verified'), [stagedVideos]);
    const selectedVideo = useMemo(() => stagedVideos.find(v => v.id === selectedVideoId), [selectedVideoId, stagedVideos]);

    const handleSelectVideo = (video: StagedVideo) => {
        setSelectedVideoId(video.id);
    };

    const updateSelectedVideo = (field: keyof StagedVideo, value: any) => {
        setStagedVideos(prev => prev.map(v => v.id === selectedVideoId ? { ...v, [field]: value } : v));
    };
    
    const handleTranscriptChange = (index: number, newText: string) => {
        if (selectedVideo && selectedVideo.report) {
            const newReport = JSON.parse(JSON.stringify(selectedVideo.report));
            newReport.transcript[index].text = newText;
            updateSelectedVideo('report', newReport);
        }
    };

    const handleApprove = () => {
        if (!selectedVideoId) return;
        updateSelectedVideo('status', 'Verified');
    };

    const handleRunAdvanced = async () => {
        if (!selectedVideo) return;
        const result = await runAdvancedAnalysis(selectedVideo, speakerMap);
        updateSelectedVideo('forensicAnalysis', result);
    };

    const handleTimestampClick = (timeInSeconds: number) => {
        if (videoRef.current) {
            videoRef.current.currentTime = timeInSeconds;
            videoRef.current.play();
        }
    };

    return (
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 h-full">
            <div className="lg:col-span-3 bg-slate-700/50 p-4 rounded-lg border border-slate-600 flex flex-col">
                <h2 className="text-xl font-bold text-slate-100 mb-4">Videos to Verify</h2>
                <div className="flex-grow overflow-y-auto space-y-2 pr-2">
                    {analyzableVideos.length > 0 ? analyzableVideos.map(video => (
                        <div key={video.id} onClick={() => handleSelectVideo(video)} className={`p-3 rounded-md cursor-pointer transition-colors ${selectedVideoId === video.id ? 'bg-cyan-900/50' : 'bg-slate-600/50 hover:bg-slate-600'}`}>
                            <p className="text-sm font-medium text-slate-100 truncate">{video.file.name}</p>
                            <p className={`text-xs font-semibold ${video.status === 'Verified' ? 'text-cyan-400' : 'text-green-400'}`}>{video.status}</p>
                        </div>
                    )) : <p className="text-slate-400 text-center mt-4">No analyzed videos available.</p>}
                </div>
            </div>

            <div className="lg:col-span-9 overflow-y-auto">
                {selectedVideo && selectedVideo.report ? (
                    <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
                        <div className="space-y-4">
                            <video ref={videoRef} key={selectedVideo.id} src={selectedVideo.videoUrl} controls className="w-full aspect-video bg-black rounded-lg" />
                            <div className="bg-slate-700/50 p-4 rounded-lg border border-slate-600 space-y-4">
                                <h3 className="text-lg font-semibold text-cyan-300">Event Details</h3>
                                <div className="flex items-center gap-2 bg-slate-800 p-2 rounded-md">
                                    <p className="text-sm font-medium text-slate-300">Initial Apparent Sentiment:</p>
                                    <span className="text-sm font-semibold text-slate-100">{selectedVideo.report.moodAnalysis.overallMood}</span>
                                    <div className="group relative flex items-center">
                                        <InfoIcon className="w-4 h-4 text-slate-400" />
                                        <div className="absolute bottom-full mb-2 hidden w-64 rounded-md bg-slate-900 p-2 text-xs text-slate-200 shadow-lg group-hover:block border border-slate-700 z-10">
                                            This is the AI's first-pass, surface-level assessment of the emotion being projected. Use this to compare against the actual context and the Advanced Analysis results.
                                        </div>
                                    </div>
                                </div>
                                <EditableField label="Event Description" value={selectedVideo.description} onChange={val => updateSelectedVideo('description', val)} type="textarea" />
                                <div className="grid grid-cols-2 gap-4">
                                    <EditableField label="Location" value={selectedVideo.location} onChange={val => updateSelectedVideo('location', val)} />
                                    <EditableField label="Category" value={selectedVideo.category} onChange={val => updateSelectedVideo('category', val)} />
                                    <SelectField label="Evidence Strength" value={selectedVideo.evidenceStrength} onChange={val => updateSelectedVideo('evidenceStrength', val)} options={['Weak', 'Moderate', 'Strong', 'Conclusive']} />
                                    <SelectField label="Is Significant Event?" value={String(selectedVideo.isSignificant)} onChange={val => updateSelectedVideo('isSignificant', val === 'true')} options={['false', 'true']} />
                                </div>
                                <EditableField label="Manipulation Pattern" value={selectedVideo.manipulationPattern} onChange={val => updateSelectedVideo('manipulationPattern', val)} />
                            </div>
                            {selectedVideo.forensicAnalysis && (
                                <div className="bg-slate-700/50 p-4 rounded-lg border border-purple-600 space-y-2">
                                    <h3 className="text-lg font-semibold text-purple-300">Advanced Analysis Results</h3>
                                    <p className="text-sm"><strong>True Sentiment:</strong> <span className="text-slate-100">{selectedVideo.forensicAnalysis.sentiment}</span></p>
                                    <p className="text-sm"><strong>Deception Score:</strong> <span className="text-slate-100">{selectedVideo.forensicAnalysis.deceptionScore} / 100</span></p>
                                    {selectedVideo.forensicAnalysis.microExpressions && selectedVideo.forensicAnalysis.microExpressions.length > 0 && <p className="text-sm"><strong>Tactics/Expressions:</strong> <span className="text-slate-200">{selectedVideo.forensicAnalysis.microExpressions.join(', ')}</span></p>}
                                    <p className="text-sm"><strong>Key Excerpt:</strong> <em className="text-slate-300">"{selectedVideo.forensicAnalysis.transcriptExcerpt}"</em></p>
                                </div>
                            )}
                        </div>
                        <div className="space-y-4 max-h-[calc(100vh-250px)] flex flex-col">
                            <div className="bg-slate-700/50 p-4 rounded-lg border border-slate-600 flex-grow flex flex-col">
                                <h3 className="text-lg font-semibold text-cyan-300 mb-3">Editable Transcript</h3>
                                <div className="flex-grow overflow-y-auto pr-2 space-y-4">
                                    {selectedVideo.report.transcript?.map((entry, index) => (
                                        <div key={index} className="flex items-start gap-3 text-sm">
                                            <div className="w-28 flex-shrink-0">
                                                <EditableSpeaker originalSpeaker={entry.speaker} speakerMap={speakerMap} onUpdate={(original, newName) => setSpeakerMap(prev => ({...prev, [original]: newName}))} />
                                                <button onClick={() => handleTimestampClick(parseTimestamp(entry.timestamp))} className="text-xs text-cyan-400 hover:underline mt-1">{entry.timestamp}</button>
                                            </div>
                                            <textarea value={entry.text} onChange={e => handleTranscriptChange(index, e.target.value)} rows={2} className="w-full bg-slate-800 border-slate-600 rounded-md shadow-sm py-1 px-2 focus:outline-none focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm" />
                                        </div>
                                    ))}
                                </div>
                            </div>
                            {isProcessing ? <Loader phase={currentPhase} /> : (
                                <div className="flex gap-4">
                                    {selectedVideo.status === 'Analyzed' && <button onClick={handleApprove} className="w-full bg-green-600 hover:bg-green-500 text-white font-bold py-3 px-4 rounded-lg">Approve & Save Changes</button>}
                                    {selectedVideo.status === 'Verified' && <button onClick={handleRunAdvanced} className="w-full bg-purple-600 hover:bg-purple-500 text-white font-bold py-3 px-4 rounded-lg flex items-center justify-center gap-2"><BrainIcon className="w-5 h-5"/>Run Advanced Analysis</button>}
                                </div>
                            )}
                        </div>
                    </div>
                ) : (
                    <div className="flex flex-col items-center justify-center h-full text-center text-slate-400">
                        <h3 className="text-lg font-semibold text-slate-200">Review & Verify</h3>
                        <p className="mt-2">Select an 'Analyzed' video from the list to review, edit, and verify its AI-generated report.</p>
                    </div>
                )}
            </div>
        </div>
    );
};
