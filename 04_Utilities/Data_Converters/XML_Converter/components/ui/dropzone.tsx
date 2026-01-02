import React, { createContext, useCallback, useContext } from 'react'
import { CheckCircle, File, Loader2, Upload, X } from 'lucide-react'
import { cn, formatBytes } from '../../lib/utils'
import { Button } from './button'

// Types mocked to match the provided code structure since we don't have the original hook source
export interface FileWithErrors {
    name: string
    size: number
    type: string
    preview?: string
    errors: { message: string }[]
}

interface DropzoneContextType {
    files: FileWithErrors[]
    setFiles: (files: FileWithErrors[]) => void
    onUpload: () => void
    loading: boolean
    successes: string[]
    errors: { name: string; message: string }[]
    maxFileSize: number
    maxFiles: number
    isSuccess: boolean
    isDragActive: boolean
    isDragReject: boolean
    inputRef: React.RefObject<HTMLInputElement>
}

const DropzoneContext = createContext<DropzoneContextType | undefined>(undefined)

// Modified to accept standard Dropzone props + context values
type DropzoneProps = DropzoneContextType & {
  className?: string
  getRootProps: (props?: any) => any
  getInputProps: (props?: any) => any
  children?: React.ReactNode
}

const Dropzone = ({
  className,
  children,
  getRootProps,
  getInputProps,
  ...contextValues
}: DropzoneProps) => {
  const { isSuccess, isDragActive, isDragReject, errors, files } = contextValues
  
  const isInvalid =
    (isDragActive && isDragReject) ||
    (errors.length > 0 && !isSuccess) ||
    files.some((file) => file.errors.length !== 0)

  return (
    <DropzoneContext.Provider value={contextValues}>
      <div
        {...getRootProps({
          className: cn(
            'border-2 border-gray-300 rounded-lg p-6 text-center bg-card transition-colors duration-300 text-foreground cursor-pointer',
            className,
            isSuccess ? 'border-solid border-green-500' : 'border-dashed border-zinc-700',
            isDragActive && 'border-primary bg-primary/10',
            isInvalid && 'border-destructive bg-destructive/10'
          ),
        })}
      >
        <input {...getInputProps()} />
        {children}
      </div>
    </DropzoneContext.Provider>
  )
}

const DropzoneContent = ({ className }: { className?: string }) => {
  const {
    files,
    setFiles,
    onUpload,
    loading,
    successes,
    errors,
    maxFileSize,
    maxFiles,
    isSuccess,
  } = useDropzoneContext()

  const exceedMaxFiles = files.length > maxFiles

  const handleRemoveFile = useCallback(
    (fileName: string) => {
      setFiles(files.filter((file) => file.name !== fileName))
    },
    [files, setFiles]
  )

  if (isSuccess) {
    return (
      <div className={cn('flex flex-row items-center gap-x-2 justify-center', className)}>
        <CheckCircle size={16} className="text-primary" />
        <p className="text-primary text-sm">
          Ready to process {files.length} file{files.length > 1 ? 's' : ''}
        </p>
      </div>
    )
  }

  return (
    <div className={cn('flex flex-col', className)}>
      {files.map((file, idx) => {
        const fileError = errors.find((e) => e.name === file.name)
        const isSuccessfullyUploaded = !!successes.find((e) => e === file.name)

        return (
          <div
            key={`${file.name}-${idx}`}
            className="flex items-center gap-x-4 border-b border-white/5 py-2 first:mt-4 last:mb-4 "
          >
            <div className="h-10 w-10 rounded border border-white/10 bg-muted flex items-center justify-center text-zinc-400">
                <File size={18} />
            </div>

            <div className="shrink grow flex flex-col items-start truncate">
              <p title={file.name} className="text-sm truncate max-w-full text-zinc-300">
                {file.name}
              </p>
              {file.errors.length > 0 ? (
                <p className="text-xs text-destructive">
                  {file.errors
                    .map((e) =>
                      e.message.startsWith('File is larger than')
                        ? `File is larger than ${formatBytes(maxFileSize, 2)} (Size: ${formatBytes(file.size, 2)})`
                        : e.message
                    )
                    .join(', ')}
                </p>
              ) : loading && !isSuccessfullyUploaded ? (
                <p className="text-xs text-muted-foreground">Reading file...</p>
              ) : !!fileError ? (
                <p className="text-xs text-destructive">Failed: {fileError.message}</p>
              ) : isSuccessfullyUploaded ? (
                <p className="text-xs text-primary">Ready</p>
              ) : (
                <p className="text-xs text-muted-foreground">{formatBytes(file.size, 2)}</p>
              )}
            </div>

            {!loading && !isSuccessfullyUploaded && (
              <Button
                size="icon"
                variant="link"
                className="shrink-0 justify-self-end text-muted-foreground hover:text-foreground"
                onClick={(e) => { e.stopPropagation(); handleRemoveFile(file.name); }}
              >
                <X size={16} />
              </Button>
            )}
          </div>
        )
      })}
      {exceedMaxFiles && (
        <p className="text-sm text-left mt-2 text-destructive">
          You may upload only up to {maxFiles} files, please remove {files.length - maxFiles} file
          {files.length - maxFiles > 1 ? 's' : ''}.
        </p>
      )}
    </div>
  )
}

const DropzoneEmptyState = ({ className }: { className?: string }) => {
  const { maxFiles, maxFileSize, inputRef, isSuccess } = useDropzoneContext()

  if (isSuccess) {
    return null
  }

  return (
    <div className={cn('flex flex-col items-center gap-y-2', className)}>
      <Upload size={20} className="text-muted-foreground" />
      <p className="text-sm text-zinc-300">
        Upload{!!maxFiles && maxFiles > 1 ? ` ${maxFiles}` : ''} file
        {!maxFiles || maxFiles > 1 ? 's' : ''}
      </p>
      <div className="flex flex-col items-center gap-y-1">
        <p className="text-xs text-muted-foreground">
          Drag and drop or select files
        </p>
        {maxFileSize !== Number.POSITIVE_INFINITY && (
          <p className="text-xs text-muted-foreground">
            Maximum file size: {formatBytes(maxFileSize, 2)}
          </p>
        )}
      </div>
    </div>
  )
}

const useDropzoneContext = () => {
  const context = useContext(DropzoneContext)

  if (!context) {
    throw new Error('useDropzoneContext must be used within a Dropzone')
  }

  return context
}

export { Dropzone, DropzoneContent, DropzoneEmptyState, useDropzoneContext }