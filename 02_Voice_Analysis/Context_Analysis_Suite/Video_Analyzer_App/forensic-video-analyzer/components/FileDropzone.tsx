
import React, { useState, useCallback } from 'react';
import { UploadIcon } from './icons/UploadIcon';

interface FileDropzoneProps {
  onFilesSelect: (files: File[]) => void;
}

export const FileDropzone: React.FC<FileDropzoneProps> = ({ onFilesSelect }) => {
  const [isDragging, setIsDragging] = useState(false);

  const handleDragEnter = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(true);
  };

  const handleDragLeave = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
  };

  const handleDragOver = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
  };

  const handleDrop = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      onFilesSelect(Array.from(e.dataTransfer.files));
      e.dataTransfer.clearData();
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      onFilesSelect(Array.from(e.target.files));
    }
  };

  return (
    <div
      onDragEnter={handleDragEnter}
      onDragLeave={handleDragLeave}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
      className={`relative flex flex-col items-center justify-center w-full max-w-3xl mx-auto p-12 border-2 border-dashed rounded-xl transition-all duration-300 ${
        isDragging ? 'border-cyan-400 bg-slate-700' : 'border-slate-600 hover:border-slate-500 bg-slate-700/50'
      }`}
    >
      <input
        type="file"
        id="file-upload"
        className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
        onChange={handleFileChange}
        accept="video/*"
        multiple
      />
      <div className="text-center pointer-events-none">
        <UploadIcon className="w-16 h-16 mx-auto text-slate-400 mb-4" />
        <p className="text-xl font-semibold text-slate-200">
          Drop your video file(s) here or <span className="text-cyan-300">click to browse</span>
        </p>
        <p className="text-slate-400 mt-2">Supports MP4, WebM, Ogg (20-60 seconds recommended)</p>
        <div className="text-xs text-slate-400 mt-8 bg-slate-800/50 p-3 rounded-md border border-slate-600 max-w-md mx-auto">
            <p><span className="font-bold text-slate-300">Privacy First:</span> For local processing, videos are never uploaded. Only extracted data is sent for analysis.</p>
        </div>
      </div>
    </div>
  );
};
