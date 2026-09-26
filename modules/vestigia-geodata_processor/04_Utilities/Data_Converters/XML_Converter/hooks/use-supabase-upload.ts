import { useState, useRef, useCallback } from 'react';
import { useDropzone, type DropzoneOptions, type FileRejection } from 'react-dropzone';

export interface SupabaseUploadConfig {
  bucketName: string;
  path: string;
  allowedMimeTypes?: string[];
  maxFiles?: number;
  maxFileSize?: number;
}

export interface UploadedFile {
  name: string;
  size: number;
  errors: string[];
}

export interface UseSupabaseUploadReturn {
  files: UploadedFile[];
  loading: boolean;
  isSuccess: boolean;
  errors: string[];
  getRootProps: ReturnType<typeof useDropzone>['getRootProps'];
  getInputProps: ReturnType<typeof useDropzone>['getInputProps'];
  isDragActive: boolean;
  isDragReject: boolean;
  onUpload: () => Promise<void>;
  inputRef: React.RefObject<HTMLInputElement>;
}

export function useSupabaseUpload(config: SupabaseUploadConfig): UseSupabaseUploadReturn {
  const [files, setFiles] = useState<UploadedFile[]>([]);
  const [loading, setLoading] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const [errors, setErrors] = useState<string[]>([]);
  const inputRef = useRef<HTMLInputElement>(null);

  const onDrop = useCallback((acceptedFiles: File[], fileRejections: FileRejection[]) => {
    setIsSuccess(false);
    setErrors([]);

    const processedFiles: UploadedFile[] = acceptedFiles.map(file => ({
      name: file.name,
      size: file.size,
      errors: []
    }));

    const rejectedErrors = fileRejections.map(rejection => {
      const errors = rejection.errors.map(e => e.message).join(', ');
      return `${rejection.file.name}: ${errors}`;
    });

    setFiles(processedFiles);
    setErrors(rejectedErrors);
  }, []);

  const dropzoneOptions: DropzoneOptions = {
    onDrop,
    accept: config.allowedMimeTypes
      ? config.allowedMimeTypes.reduce((acc, type) => ({...acc, [type]: []}), {})
      : undefined,
    maxFiles: config.maxFiles,
    maxSize: config.maxFileSize,
  };

  const { getRootProps, getInputProps, isDragActive, isDragReject } = useDropzone(dropzoneOptions);

  const onUpload = async () => {
    if (files.length === 0) return;

    setLoading(true);
    setErrors([]);

    try {
      // For this XML converter, we don't actually upload to Supabase here
      // Instead, we just signal success and let the parent component handle the file
      // This is a placeholder implementation
      await new Promise(resolve => setTimeout(resolve, 500));
      setIsSuccess(true);
    } catch (error) {
      setErrors([error instanceof Error ? error.message : 'Upload failed']);
    } finally {
      setLoading(false);
    }
  };

  return {
    files,
    loading,
    isSuccess,
    errors,
    getRootProps,
    getInputProps,
    isDragActive,
    isDragReject,
    onUpload,
    inputRef,
  };
}
