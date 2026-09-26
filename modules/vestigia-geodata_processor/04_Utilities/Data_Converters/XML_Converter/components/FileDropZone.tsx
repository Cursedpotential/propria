
import React, { useCallback, useRef, useState } from 'react';
import { useDropzone } from 'react-dropzone';
import { Dropzone, DropzoneContent, DropzoneEmptyState, FileWithErrors } from './ui/dropzone';
import { GoogleDriveService } from '../services/googleDriveService';
import { StreamSource } from '../types';

interface Props {
  onFileSelect: (source: StreamSource) => void;
  currentFile: File | null;
  currentSourceName?: string;
  sourceLabel: string;
  onSourceLabelChange: (label: string) => void;
  googleDriveClientId?: string;
}

export const FileDropZone: React.FC<Props> = ({ onFileSelect, currentFile, currentSourceName, sourceLabel, onSourceLabelChange, googleDriveClientId }) => {
  const [files, setFiles] = useState<FileWithErrors[]>([]);
  const inputRef = useRef<HTMLInputElement>(null);
  const driveService = useRef(new GoogleDriveService());

  const onDrop = useCallback((acceptedFiles: File[]) => {
    // Map to the internal FileWithErrors type expected by the UI component
    const mappedFiles = acceptedFiles.map(f => ({
        name: f.name,
        size: f.size,
        type: f.type,
        errors: []
    }));
    setFiles(mappedFiles);

    if (acceptedFiles.length > 0) {
        const f = acceptedFiles[0];
        onFileSelect({ type: 'FILE', file: f });
        if (!sourceLabel) onSourceLabelChange(f.name.split('.')[0]);
    }
  }, [onFileSelect, sourceLabel, onSourceLabelChange]);

  const { getRootProps, getInputProps, isDragActive, isDragReject } = useDropzone({
      onDrop,
      maxFiles: 1,
      accept: { 'text/xml': ['.xml'] }
  });

  const handleGoogleDrive = async () => {
      if (!googleDriveClientId) {
          alert("Please configure Google Drive Client ID in Settings first.");
          return;
      }
      try {
          const result = await driveService.current.createPicker(googleDriveClientId);
          if (result) {
              // Update UI state
              setFiles([{
                  name: result.name,
                  size: result.size,
                  type: 'text/xml',
                  errors: []
              }]);
              
              onFileSelect({ 
                  type: 'DRIVE', 
                  url: result.id, 
                  name: result.name, 
                  token: result.token, 
                  size: result.size 
              });
              
              if (!sourceLabel) onSourceLabelChange(result.name.split('.')[0]);
          }
      } catch (e: any) {
          console.error(e);
          alert(`Google Drive Error: ${e.toString()}`);
      }
  };

  return (
    <div className="glass-panel rounded-xl p-6 border border-white/10 shadow-2xl space-y-4">
        {/* Source Label Input */}
        <div className="flex justify-between items-end">
            <div className="flex-1 mr-4">
                <label className="block text-xs font-bold text-violet-300 uppercase tracking-wider mb-2">Data Source Label</label>
                <input 
                    type="text" 
                    value={sourceLabel}
                    onChange={(e) => onSourceLabelChange(e.target.value)}
                    placeholder="e.g. 'iPhone 13 Backup' or 'Cloud Export 2023'"
                    className="w-full bg-black/40 border border-white/10 rounded-lg px-4 py-3 text-sm text-white focus:border-violet-500 focus:ring-1 focus:ring-violet-500 outline-none transition-all placeholder:text-zinc-600"
                />
                <p className="text-[10px] text-zinc-500 mt-1.5">
                    Required for deduplication. Same content from different sources will be preserved.
                </p>
            </div>
             <button 
                onClick={handleGoogleDrive}
                className="mb-6 px-3 py-2 bg-white/5 hover:bg-white/10 border border-white/10 rounded-lg flex items-center gap-2 transition-colors text-xs font-medium text-zinc-300"
             >
                <svg viewBox="0 0 87.3 78" className="w-4 h-4">
                    <path d="m6.6 66.85 3.85 6.65c.8 1.4 1.95 2.5 3.3 3.3l13.75-23.8h-27.5c0 1.55.4 3.1 1.2 4.5z" fill="#0066da"/>
                    <path d="m43.65 25-13.75-23.8c-1.35.8-2.5 1.9-3.3 3.3l-25.4 44a9.06 9.06 0 0 0 -1.2 4.5h27.5z" fill="#00ac47"/>
                    <path d="m73.55 76.8c1.35-.8 2.5-1.9 3.3-3.3l1.6-2.75 7.65-13.25c.8-1.4 1.2-2.95 1.2-4.5h-27.502l5.852 11.5z" fill="#ea4335"/>
                    <path d="m43.65 25 13.75-23.8c-1.35-.8-2.9-1.2-4.5-1.2h-18.5c-1.6 0-3.15.45-4.5 1.2z" fill="#00832d"/>
                    <path d="m59.8 53h-27.5l-13.75 23.8c1.35.8 2.9 1.2 4.5 1.2h50.5c1.6 0 3.15-.45 4.5-1.2z" fill="#2684fc"/>
                    <path d="m73.4 26.5-12.7-22c-.8-1.4-1.95-2.5-3.3-3.3l-13.75 23.8h27.5c0-1.55-.4-3.1-1.2-4.5z" fill="#ffba00"/>
                </svg>
                Google Drive
            </button>
        </div>

        {/* New Dropzone Component Integration */}
        <Dropzone 
            files={files}
            setFiles={setFiles}
            onUpload={() => {}} // No-op for local processing
            loading={false}
            successes={currentFile || currentSourceName ? [currentSourceName || currentFile!.name] : []}
            errors={[]}
            maxFileSize={1024 * 1024 * 1024 * 50} // 50GB
            maxFiles={1}
            isSuccess={!!currentFile || !!currentSourceName}
            isDragActive={isDragActive}
            isDragReject={isDragReject}
            inputRef={inputRef}
            getRootProps={getRootProps}
            getInputProps={getInputProps}
            className="min-h-[200px] flex flex-col items-center justify-center bg-black/20"
        >
            <DropzoneEmptyState />
            <DropzoneContent />
        </Dropzone>
    </div>
  );
};
