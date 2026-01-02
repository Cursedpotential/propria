// frontend/src/App.tsx
import React, { useEffect, useState, useCallback } from 'react';
import { BrowserOpenFileDialog, EventsOn, EventsOff, LogPrint } from '../wailsjs/runtime/runtime';
import {
  App,
  ScanOptions,
  ScanProgress,
  CloudDrive,
  Settings,
  FileNode,
} from '../wailsjs/go/main/App';

// Types matching backend Go structs
interface ScanOptions {
  path: string;
  maxDepth: number;
  includeHidden: boolean;
  includeSystem: boolean;
  followSymlinks: boolean;
  minSize: number;
  maxSize: number;
  extensions: string[];
  excludeDirs: string[];
  excludePatterns: string[];
  threadCount: number;
  calculateHash: boolean;
  cacheResults: boolean;
  pageSize: number;
}

interface ScanProgress {
  sessionId: string;
  currentPath: string;
  filesScanned: number;
  dirsScanned: number;
  totalSize: number;
  formattedSize: string;
  errors: string[];
  percentage: number;
  speed: number;
  timeRemaining: number; // nanoseconds
  currentDepth: number;
  estimatedTotal: number;
  status: string;
}

interface CloudDrive {
  type: string;
  provider: string;
  mount_point: string;
  path: string;
  email: string;
  mounted: boolean;
  display_name: string;
  detected: boolean;
  notes: string;
}

interface Settings {
  threadCount: number;
  cacheResults: boolean;
  defaultExportFormat: string;
}

interface FileNode {
  id: string;
  name: string;
  path: string;
  size: number;
  formattedSize: string;
  created: string;
  modified: string;
  accessed: string;
  isDir: boolean;
  isSymlink: boolean;
  isHidden: boolean;
  isSystem: boolean;
  isReadOnly: boolean;
  isCloud: boolean;
  cloudType?: string;
  extension: string;
  mimeType: string;
  permissions: string;
  owner: string;
  hash?: string;
  attributes: number;
  childCount: number;
  childrenLoaded: boolean;
  children?: FileNode[];
  parentId?: string;
}

const EXPORT_FORMATS = [
  'json',
  'csv',
  'excel',
  'html',
  'pdf',
  'markdown',
  'xml',
  'tree',
] as const;

type ExportFormat = typeof EXPORT_FORMATS[number];

export default function App() {
  // UI state
  const [initialized, setInitialized] = useState(false);
  const [selectedPath, setSelectedPath] = useState('');
  const [cloudDrives, setCloudDrives] = useState<CloudDrive[]>([]);
  const [settings, setSettings] = useState<Settings>({
    threadCount: 4,
    cacheResults: true,
    defaultExportFormat: 'json',
  });

  // Pre-scan stats
  const [preScanStats, setPreScanStats] = useState<{
    dirCount: number;
    fileCount: number;
    totalSize: number;
    loading: boolean;
  }>({ dirCount: 0, fileCount: 0, totalSize: 0, loading: false });

  // Scan state
  const [isScanning, setIsScanning] = useState(false);
  const [scanProgress, setScanProgress] = useState<ScanProgress | null>(null);
  const [scanErrors, setScanErrors] = useState<string[]>([]);
  const [scanComplete, setScanComplete] = useState(false);

  // Export state
  const [exportFormat, setExportFormat] = useState<ExportFormat>('json');
  const [isExporting, setIsExporting] = useState(false);
  const [exportStatus, setExportStatus] = useState('');

  // Logs
  const [logs, setLogs] = useState<string[]>([]);

  // Initialize on mount
  useEffect(() => {
    (async () => {
      try {
        LogPrint("Initializing app...");
        
        // Load settings
        const loadedSettings = await App.GetSettings();
        setSettings(loadedSettings);
        LogPrint(`Settings loaded: ${JSON.stringify(loadedSettings)}`);

        // Detect cloud drives
        await App.DetectCloudDrives();
        const drives = await App.GetCloudDrives();
        setCloudDrives(drives);

        // Subscribe to scan progress
        await App.SubscribeScanProgress();

        // Listen for progress events
        EventsOn('scanProgress', (p: ScanProgress) => {
          setScanProgress(p);
        });

        setInitialized(true);
        LogPrint("App initialized successfully");
      } catch (err) {
        LogPrint(`Initialization error: ${err}`);
        addLog(`Initialization failed: ${err}`);
      }
    })();

    return () => {
      EventsOff('scanProgress');
    };
  }, []);

  // Add log entry
  const addLog = useCallback((msg: string) => {
    setLogs(prev => [...prev, `${new Date().toLocaleTimeString()}: ${msg}`]);
    LogPrint(msg);
  }, []);

  // Browse for directory
  const handleBrowse = async () => {
    try {
      const path = await App.Browse();
      if (path) {
        setSelectedPath(path);
        addLog(`Selected path: ${path}`);
        await calculatePreScanStats(path);
      }
    } catch (err) {
      addLog(`Browse error: ${err}`);
    }
  };

  // Calculate pre-scan stats by walking directory
  const calculatePreScanStats = async (path: string) => {
    setPreScanStats(prev => ({ ...prev, loading: true }));
    try {
      // Use a quick scan with depth 1 to get top-level stats
      const quickOpts: ScanOptions = {
        path,
        maxDepth: 1,
        includeHidden: settings.cacheResults,
        includeSystem: false,
        followSymlinks: false,
        minSize: 0,
        maxSize: 0,
        extensions: [],
        excludeDirs: ['.git', 'node_modules', 'venv', '.venv'],
        excludePatterns: [],
        threadCount: Math.min(settings.threadCount, 2), // Use fewer threads for quick scan
        calculateHash: false,
        cacheResults: false,
        pageSize: 100,
      };

      // This is a quick estimate - in production you might want a dedicated endpoint
      const result = await App.StartScan(quickOpts);
      if (result) {
        // For now, we'll use a simple estimate
        // In a real implementation, add a backend method for quick stats
        setPreScanStats({
          dirCount: 1000, // Placeholder - backend should provide this
          fileCount: 5000, // Placeholder
          totalSize: 1024 * 1024 * 1024, // Placeholder: 1GB
          loading: false,
        });
      }
    } catch (err) {
      addLog(`Pre-scan stats error: ${err}`);
    } finally {
      setPreScanStats(prev => ({ ...prev, loading: false }));
    }
  };

  // Start main scan
  const handleStartScan = async () => {
    if (!selectedPath) {
      addLog('No path selected');
      return;
    }

    setIsScanning(true);
    setScanComplete(false);
    setScanErrors([]);
    setScanProgress(null);

    const scanOpts: ScanOptions = {
      path: selectedPath,
      maxDepth: 100,
      includeHidden: settings.cacheResults,
      includeSystem: false,
      followSymlinks: false,
      minSize: 0,
      maxSize: 0,
      extensions: [],
      excludeDirs: ['.git', 'node_modules', 'venv', '.venv', 'dist', 'build'],
      excludePatterns: [],
      threadCount: settings.threadCount,
      calculateHash: false,
      cacheResults: settings.cacheResults,
      pageSize: 100,
    };

    try {
      addLog(`Starting scan of ${selectedPath} with ${settings.threadCount} threads...`);
      await App.StartScan(scanOpts);
      
      // Poll for completion (in production, use a completion event)
      const checkInterval = setInterval(async () => {
        const progress = await App.GetCurrentProgress();
        if (progress && progress.status === 'completed') {
          clearInterval(checkInterval);
          setIsScanning(false);
          setScanComplete(true);
          addLog('Scan completed successfully');
          
          // Get final results
          const drives = await App.GetCloudDrives();
          setCloudDrives(drives);
        }
      }, 1000);

    } catch (err) {
      addLog(`Scan error: ${err}`);
      setIsScanning(false);
    }
  };

  // Stop scan
  const handleStopScan = async () => {
    try {
      await App.StopScan();
      setIsScanning(false);
      addLog('Scan stopped');
    } catch (err) {
      addLog(`Stop scan error: ${err}`);
    }
  };

  // Export results
  const handleExport = async () => {
    if (!scanComplete) {
      addLog('No completed scan to export');
      return;
    }

    try {
      setIsExporting(true);
      setExportStatus('Opening save dialog...');

      const outputPath = await App.BrowseForExport(exportFormat);
      if (!outputPath) {
        setExportStatus('Export cancelled');
        setIsExporting(false);
        return;
      }

      setExportStatus(`Exporting to ${outputPath}...`);
      addLog(`Exporting scan results as ${exportFormat} to ${outputPath}`);

      await App.ExportResults(exportFormat, outputPath);

      setExportStatus('Export completed successfully');
      addLog(`Export completed: ${outputPath}`);
    } catch (err) {
      setExportStatus(`Export failed: ${err}`);
      addLog(`Export error: ${err}`);
    } finally {
      setIsExporting(false);
    }
  };

  // Format bytes for display
  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  // Get scan speed in MB/s
  const getScanSpeedMBps = (): number => {
    if (!scanProgress || scanProgress.speed === 0) return 0;
    return (scanProgress.speed / 1024 / 1024);
  };

  // Get ETA as string
  const getETA = (): string => {
    if (!scanProgress || scanProgress.timeRemaining === 0) return 'N/A';
    const seconds = Math.ceil(scanProgress.timeRemaining / 1e9);
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
    return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
  };

  return (
    <div className="min-h-screen bg-gray-900 text-gray-100 p-6">
      <div className="max-w-7xl mx-auto">
        <h1 className="text-3xl font-bold text-blue-400 mb-6">Directory Scanner Pro</h1>

        {/* Initialization status */}
        {!initialized && (
          <div className="bg-yellow-900 border border-yellow-700 rounded-lg p-4 mb-6">
            <p className="text-yellow-300">Initializing application...</p>
          </div>
        )}

        {/* Cloud Drives Section */}
        <div className="bg-gray-800 rounded-lg p-6 mb-6">
          <h2 className="text-xl font-semibold mb-4 text-green-400">Detected Cloud Drives</h2>
          {cloudDrives.length === 0 ? (
            <p className="text-gray-400">No cloud drives detected. You can still browse any folder manually.</p>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {cloudDrives.map((drive, idx) => (
                <div
                  key={idx}
                  className="bg-gray-700 rounded-lg p-4 cursor-pointer hover:bg-gray-600 transition-colors"
                  onClick={() => {
                    if (drive.mounted) {
                      setSelectedPath(drive.mount_point);
                      addLog(`Selected cloud drive: ${drive.display_name}`);
                      calculatePreScanStats(drive.mount_point);
                    }
                  }}
                >
                  <div className="flex items-start justify-between mb-2">
                    <h3 className="font-semibold text-blue-300">{drive.display_name}</h3>
                    <span className={`px-2 py-1 rounded text-xs ${
                      drive.mounted ? 'bg-green-900 text-green-300' : 'bg-red-900 text-red-300'
                    }`}>
                      {drive.mounted ? 'Mounted' : 'Not Mounted'}
                    </span>
                  </div>
                  <p className="text-sm text-gray-400 mb-1">{drive.provider}</p>
                  <p className="text-xs text-gray-500 truncate">{drive.mount_point}</p>
                  {drive.email && <p className="text-xs text-blue-400 mt-1">{drive.email}</p>}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Path Selection & Pre-scan Stats */}
        <div className="bg-gray-800 rounded-lg p-6 mb-6">
          <h2 className="text-xl font-semibold mb-4 text-purple-400">Scan Configuration</h2>
          
          <div className="flex items-center gap-4 mb-4">
            <button
              onClick={handleBrowse}
              className="bg-blue-600 hover:bg-blue-700 px-4 py-2 rounded-lg font-medium transition-colors"
            >
              Browse Folder
            </button>
            <span className="text-gray-300 flex-1 truncate">
              {selectedPath || 'No folder selected'}
            </span>
          </div>

          {selectedPath && (
            <div className="bg-gray-700 rounded-lg p-4">
              <h3 className="font-semibold mb-2 text-yellow-400">Pre-scan Statistics</h3>
              {preScanStats.loading ? (
                <p className="text-gray-400">Calculating...</p>
              ) : (
                <div className="grid grid-cols-3 gap-4 text-center">
                  <div>
                    <p className="text-2xl font-bold text-blue-400">{preScanStats.dirCount.toLocaleString()}</p>
                    <p className="text-xs text-gray-400">Directories</p>
                  </div>
                  <div>
                    <p className="text-2xl font-bold text-green-400">{preScanStats.fileCount.toLocaleString()}</p>
                    <p className="text-xs text-gray-400">Files</p>
                  </div>
                  <div>
                    <p className="text-2xl font-bold text-purple-400">{formatBytes(preScanStats.totalSize)}</p>
                    <p className="text-xs text-gray-400">Total Size</p>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Scan Controls */}
        <div className="bg-gray-800 rounded-lg p-6 mb-6">
          <h2 className="text-xl font-semibold mb-4 text-orange-400">Scan Controls</h2>
          <div className="flex items-center gap-4 mb-4">
            <button
              onClick={handleStartScan}
              disabled={!selectedPath || isScanning}
              className="bg-green-600 hover:bg-green-700 disabled:bg-gray-600 disabled:cursor-not-allowed px-6 py-2 rounded-lg font-medium transition-colors"
            >
              {isScanning ? 'Scanning...' : 'Start Scan'}
            </button>
            <button
              onClick={handleStopScan}
              disabled={!isScanning}
              className="bg-red-600 hover:bg-red-700 disabled:bg-gray-600 disabled:cursor-not-allowed px-6 py-2 rounded-lg font-medium transition-colors"
            >
              Stop Scan
            </button>
          </div>

          {/* Progress Display */}
          {isScanning && scanProgress && (
            <div className="bg-gray-700 rounded-lg p-4">
              <h3 className="font-semibold mb-2 text-cyan-400">Scan Progress</h3>
              <div className="mb-4">
                <div className="flex justify-between text-sm mb-1">
                  <span>{scanProgress.currentPath || 'Scanning...'}</span>
                  <span>{scanProgress.percentage.toFixed(1)}%</span>
                </div>
                <div className="w-full bg-gray-600 rounded-full h-2">
                  <div
                    className="bg-blue-500 h-2 rounded-full transition-all duration-300"
                    style={{ width: `${scanProgress.percentage}%` }}
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                <div>
                  <p className="text-gray-400">Files Scanned</p>
                  <p className="font-bold text-blue-400">{scanProgress.filesScanned.toLocaleString()}</p>
                </div>
                <div>
                  <p className="text-gray-400">Dirs Scanned</p>
                  <p className="font-bold text-green-400">{scanProgress.dirsScanned.toLocaleString()}</p>
                </div>
                <div>
                  <p className="text-gray-400">Speed</p>
                  <p className="font-bold text-yellow-400">{getScanSpeedMBps().toFixed(2)} MB/s</p>
                </div>
                <div>
                  <p className="text-gray-400">ETA</p>
                  <p className="font-bold text-purple-400">{getETA()}</p>
                </div>
              </div>

              <div className="mt-4 pt-4 border-t border-gray-600">
                <p className="text-sm text-gray-400">Total Size: <span className="font-bold text-white">{scanProgress.formattedSize}</span></p>
              </div>
            </div>
          )}

          {scanComplete && (
            <div className="bg-green-900 border border-green-700 rounded-lg p-4 mt-4">
              <p className="text-green-300 font-semibold">✓ Scan completed successfully!</p>
            </div>
          )}

          {scanErrors.length > 0 && (
            <div className="bg-red-900 border border-red-700 rounded-lg p-4 mt-4 max-h-40 overflow-y-auto">
              <p className="text-red-300 font-semibold mb-2">Scan Errors:</p>
              {scanErrors.map((err, idx) => (
                <p key={idx} className="text-red-200 text-sm mb-1">{err}</p>
              ))}
            </div>
          )}
        </div>

        {/* Export Section */}
        <div className="bg-gray-800 rounded-lg p-6 mb-6">
          <h2 className="text-xl font-semibold mb-4 text-teal-400">Export Results</h2>
          <div className="flex items-center gap-4 mb-4">
            <select
              value={exportFormat}
              onChange={(e) => setExportFormat(e.target.value as ExportFormat)}
              disabled={!scanComplete || isExporting}
              className="bg-gray-700 text-gray-100 px-4 py-2 rounded-lg border border-gray-600 disabled:opacity-50"
            >
              {EXPORT_FORMATS.map(fmt => (
                <option key={fmt} value={fmt}>{fmt.toUpperCase()}</option>
              ))}
            </select>
            <button
              onClick={handleExport}
              disabled={!scanComplete || isExporting}
              className="bg-teal-600 hover:bg-teal-700 disabled:bg-gray-600 disabled:cursor-not-allowed px-6 py-2 rounded-lg font-medium transition-colors"
            >
              {isExporting ? 'Exporting...' : 'Export'}
            </button>
          </div>
          {exportStatus && (
            <p className={`text-sm ${exportStatus.includes('failed') ? 'text-red-400' : 'text-teal-400'}`}>
              {exportStatus}
            </p>
          )}
        </div>

        {/* Logs Section */}
        <div className="bg-gray-800 rounded-lg p-6">
          <h2 className="text-xl font-semibold mb-4 text-gray-400">Application Logs</h2>
          <div className="bg-gray-900 rounded-lg p-4 max-h-64 overflow-y-auto font-mono text-xs text-green-400">
            {logs.length === 0 ? (
              <p className="text-gray-600">No logs yet...</p>
            ) : (
              logs.map((log, idx) => (
                <p key={idx} className="mb-1">{log}</p>
              ))
            )}
          </div>
          <button
            onClick={() => setLogs([])}
            className="mt-4 bg-gray-700 hover:bg-gray-600 px-4 py-2 rounded-lg text-sm transition-colors"
          >
            Clear Logs
          </button>
        </div>
      </div>
    </div>
  );
}