// Directory Scanner Pro - Frontend JavaScript

let currentScanResults = null;
let currentAnalysisResults = null;
let currentDuplicates = null;
let isScanning = false;
let logVisible = false;
let maxLogEntries = 1000;

// Initialize application
document.addEventListener('DOMContentLoaded', async function() {
    console.log('Directory Scanner Pro - Frontend loaded');
    addLogEntry('Directory Scanner Pro started', 'success');
    
    // Initialize UI components
    initializeUI();
    addLogEntry('UI components initialized', 'info');
    
    // Load cloud drives
    addLogEntry('Detecting cloud drives...', 'info');
    await loadCloudDrives();
    
    // Load settings
    addLogEntry('Loading application settings...', 'info');
    await loadSettings();
    
    // Set up event handlers
    setupEventHandlers();
    addLogEntry('Event handlers configured', 'debug');
    
    // Subscribe to scan progress updates
    try {
        await window.go.main.App.SubscribeScanProgress();
        addLogEntry('Subscribed to scan progress updates', 'success');
    } catch (error) {
        addLogEntry('Failed to subscribe to scan progress: ' + error, 'error');
    }
    
    // Listen for scan progress events
    window.runtime.EventsOn('scanProgress', handleScanProgress);
    addLogEntry('Application ready for use', 'success');
});

// Initialize UI components
function initializeUI() {
    // Set default thread count based on system
    const threadCountInput = document.getElementById('threadCount');
    if (threadCountInput.value === '0') {
        threadCountInput.value = navigator.hardwareConcurrency || 4;
    }
}

// Logging system
function addLogEntry(message, type = 'info') {
    const logContent = document.getElementById('logContent');
    const timestamp = new Date().toLocaleTimeString();
    
    const logEntry = document.createElement('div');
    logEntry.className = `log-entry ${type}`;
    logEntry.innerHTML = `<span class="log-timestamp">[${timestamp}]</span>${message}`;
    
    logContent.appendChild(logEntry);
    
    // Auto-scroll to bottom
    logContent.scrollTop = logContent.scrollHeight;
    
    // Limit log entries
    const entries = logContent.querySelectorAll('.log-entry');
    if (entries.length > maxLogEntries) {
        entries[0].remove();
    }
    
    console.log(`[${type.toUpperCase()}] ${message}`);
}

function clearLog() {
    const logContent = document.getElementById('logContent');
    logContent.innerHTML = '';
    addLogEntry('Log cleared', 'info');
}

function toggleLogPanel() {
    const logPanel = document.getElementById('logPanel');
    const toggleBtn = document.getElementById('toggleLogBtn');
    
    logVisible = !logVisible;
    
    if (logVisible) {
        logPanel.classList.remove('hidden');
        toggleBtn.textContent = 'Hide Logs';
        addLogEntry('Log panel opened', 'debug');
    } else {
        logPanel.classList.add('hidden');
        toggleBtn.textContent = 'Show Logs';
    }
}

// Set up all event handlers
function setupEventHandlers() {
    // Browse button
    document.getElementById('browseBtn').addEventListener('click', async function() {
        try {
            const path = await window.go.main.App.Browse();
            if (path) {
                document.getElementById('scanPath').value = path;
            }
        } catch (error) {
            showToast('Failed to browse directories: ' + error, 'error');
        }
    });

    // Scan button
    document.getElementById('scanBtn').addEventListener('click', startScan);
    
    // Stop button
    document.getElementById('stopBtn').addEventListener('click', stopScan);
    
    // Export buttons
    document.getElementById('exportBtn').addEventListener('click', exportResults);
    document.getElementById('exportAnalysisBtn').addEventListener('click', exportAnalysis);
    
    // Refresh cloud drives
    document.getElementById('refreshCloudBtn').addEventListener('click', loadCloudDrives);
    
    // Analysis buttons
    document.getElementById('analyzeBtn').addEventListener('click', analyzeFiles);
    document.getElementById('findDuplicatesBtn').addEventListener('click', findDuplicates);
    
    // Log panel controls
    document.getElementById('toggleLogBtn').addEventListener('click', toggleLogPanel);
    document.getElementById('clearLogBtn').addEventListener('click', clearLog);
    document.getElementById('hideLogBtn').addEventListener('click', toggleLogPanel);
}

// Start scan
async function startScan() {
    const path = document.getElementById('scanPath').value.trim();
    if (!path) {
        addLogEntry('Scan failed: No path specified', 'error');
        showToast('Please enter or select a path to scan', 'error');
        return;
    }
    
    addLogEntry(`Starting scan of: ${path}`, 'info');

    const options = {
        path: path,
        maxDepth: parseInt(document.getElementById('maxDepth').value) || 0,
        includeHidden: document.getElementById('includeHidden').checked,
        includeSystem: document.getElementById('includeSystem').checked,
        followSymlinks: document.getElementById('followSymlinks').checked,
        calculateHash: document.getElementById('calculateHash').checked,
        threadCount: parseInt(document.getElementById('threadCount').value) || 0,
        minSize: 0,
        maxSize: 0,
        extensions: [],
        excludeDirs: [],
        excludePatterns: [],
        cacheResults: true,
        pageSize: 1000
    };

    try {
        isScanning = true;
        updateScanUI(true);
        addLogEntry(`Scan options: depth=${options.maxDepth}, threads=${options.threadCount}, hash=${options.calculateHash}`, 'debug');
        
        // Clear previous results
        currentScanResults = null;
        currentAnalysisResults = null;
        currentDuplicates = null;
        
        // Update UI
        document.getElementById('resultsContainer').innerHTML = '<p style="text-align: center; color: var(--text-secondary);">Scanning...</p>';
        updateStatusBar('Scanning...', 0, 0, '0 B', 0);
        
        // Trigger scan start event
        window.dispatchEvent(new CustomEvent('scan:start'));
        
        const result = await window.go.main.App.StartScan(options);
        addLogEntry('Scan command sent to backend successfully', 'success');
        
    } catch (error) {
        addLogEntry('Scan failed: ' + error, 'error');
        showToast('Failed to start scan: ' + error, 'error');
        isScanning = false;
        updateScanUI(false);
    }
}

// Stop scan
async function stopScan() {
    try {
        await window.go.main.App.StopScan();
        isScanning = false;
        updateScanUI(false);
        showToast('Scan stopped', 'success');
    } catch (error) {
        console.error('Stop scan error:', error);
        showToast('Failed to stop scan: ' + error, 'error');
    }
}

// Handle scan progress updates
function handleScanProgress(progress) {
    console.log('Scan progress:', progress);
    
    // Log progress details
    if (progress.currentPath && progress.currentPath !== 'Scanning...') {
        addLogEntry(`Scanning: ${progress.currentPath}`, 'debug');
    }
    
    // Update progress bar
    const percentage = progress.percentage || 0;
    const progressBar = document.getElementById('progressBar');
    progressBar.style.width = percentage + '%';
    progressBar.textContent = Math.round(percentage) + '%';
    
    // Update current path
    const currentPathEl = document.getElementById('currentPath');
    if (progress.currentPath) {
        currentPathEl.textContent = 'Scanning: ' + progress.currentPath;
    }
    
    // Update status bar
    updateStatusBar(
        progress.status || 'Scanning...',
        progress.filesScanned || 0,
        progress.dirsScanned || 0,
        progress.formattedSize || '0 B',
        progress.speed || 0
    );
    
    // Check if scan is complete
    if (progress.status === 'completed') {
        isScanning = false;
        updateScanUI(false);
        addLogEntry(`Scan completed! Files: ${progress.filesScanned}, Dirs: ${progress.dirsScanned}, Size: ${progress.formattedSize}`, 'success');
        loadScanResults();
        showToast('Scan completed successfully!', 'success');
        
        // Trigger scan complete event
        window.dispatchEvent(new CustomEvent('scan:complete'));
    }
}

// Load scan results
async function loadScanResults() {
    try {
        const progress = await window.go.main.App.GetCurrentProgress();
        if (progress && progress.filesScanned > 0) {
            // Enable export buttons
            document.getElementById('exportBtn').disabled = false;
            
            // Show basic results info
            const resultsHtml = `
                <div style="text-align: center; padding: 20px;">
                    <h3>Scan Completed</h3>
                    <p><strong>Files scanned:</strong> ${progress.filesScanned}</p>
                    <p><strong>Directories scanned:</strong> ${progress.dirsScanned}</p>
                    <p><strong>Total size:</strong> ${progress.formattedSize || humanizeBytes(progress.totalSize || 0)}</p>
                    <p style="margin-top: 16px; color: var(--text-secondary);">
                        Use the export buttons above to save detailed results.
                    </p>
                </div>
            `;
            document.getElementById('resultsContainer').innerHTML = resultsHtml;
        }
    } catch (error) {
        console.error('Failed to load scan results:', error);
    }
}

// Update scan UI state
function updateScanUI(scanning) {
    const scanBtn = document.getElementById('scanBtn');
    const stopBtn = document.getElementById('stopBtn');
    const progressSection = document.getElementById('progressSection');
    
    scanBtn.disabled = scanning;
    stopBtn.disabled = !scanning;
    
    if (scanning) {
        scanBtn.textContent = 'Scanning...';
        progressSection.style.display = 'block';
    } else {
        scanBtn.textContent = 'Start Scan';
        // Keep progress visible for a moment after completion
        setTimeout(() => {
            if (!isScanning) {
                progressSection.style.display = 'none';
            }
        }, 2000);
    }
}

// Update status bar
function updateStatusBar(status, files, dirs, size, speed) {
    document.getElementById('statusText').textContent = status;
    document.getElementById('filesScanned').textContent = files.toLocaleString();
    document.getElementById('dirsScanned').textContent = dirs.toLocaleString();
    document.getElementById('totalSize').textContent = size;
    document.getElementById('scanSpeed').textContent = Math.round(speed) + ' files/sec';
}

// Export results
async function exportResults() {
    const format = document.getElementById('exportFormat').value;
    
    try {
        const outputPath = await window.go.main.App.BrowseForExport(format);
        if (!outputPath) return;
        
        await window.go.main.App.ExportResults(format, outputPath);
        showToast('Results exported successfully!', 'success');
        
    } catch (error) {
        console.error('Export error:', error);
        showToast('Failed to export results: ' + error, 'error');
    }
}

// Export analysis
async function exportAnalysis() {
    if (!currentAnalysisResults) {
        showToast('No analysis results available. Run analysis first.', 'error');
        return;
    }
    
    const format = document.getElementById('exportFormat').value;
    
    try {
        const outputPath = await window.go.main.App.BrowseForExport(format);
        if (!outputPath) return;
        
        // This would need to be implemented in the Go backend
        showToast('Analysis export not yet implemented', 'error');
        
    } catch (error) {
        console.error('Export analysis error:', error);
        showToast('Failed to export analysis: ' + error, 'error');
    }
}

// Analyze files
async function analyzeFiles() {
    showToast('File analysis feature not yet implemented', 'error');
}

// Find duplicates
async function findDuplicates() {
    showToast('Duplicate detection feature not yet implemented', 'error');
}

// Load cloud drives
async function loadCloudDrives() {
    const cloudDrivesContainer = document.getElementById('cloud-drives');
    
    try {
        // Show loading state
        cloudDrivesContainer.innerHTML = '<div class="no-drives">Detecting cloud drives...</div>';
        
        // Detect cloud drives
        addLogEntry('Detecting cloud storage providers...', 'info');
        await window.go.main.App.DetectCloudDrives();
        
        // Get the detected drives
        const drives = await window.go.main.App.GetCloudDrives();
        addLogEntry(`Found ${drives ? drives.length : 0} cloud drive(s)`, 'info');
        
        if (drives && drives.length > 0) {
            // Group drives by provider
            const grouped = {};
            drives.forEach(drive => {
                if (!grouped[drive.provider]) {
                    grouped[drive.provider] = [];
                }
                grouped[drive.provider].push(drive);
            });
            
            let html = '';
            Object.keys(grouped).forEach(provider => {
                addLogEntry(`${provider}: ${grouped[provider].length} account(s)`, 'info');
                html += `
                    <div class="drive-group">
                        <h4>${provider}</h4>
                        <ul class="drive-list">
                `;
                
                grouped[provider].forEach(drive => {
                    addLogEntry(`  - ${drive.display_name || drive.type}: ${drive.path} (${drive.mounted ? 'mounted' : 'not mounted'})`, 'debug');
                    const mountedClass = drive.mounted ? 'mounted' : '';
                    const scanButtonDisabled = !drive.mounted ? 'disabled' : '';
                    
                    html += `
                        <li class="drive-item ${mountedClass}">
                            <div class="drive-info">
                                <div class="drive-name">${drive.display_name || drive.type}</div>
                                ${drive.path ? `<div class="drive-path">${drive.path}</div>` : ''}
                                ${drive.email ? `<div class="drive-email">${drive.email}</div>` : ''}
                            </div>
                            <div class="drive-actions">
                                ${drive.mounted 
                                    ? `<button class="btn btn-small" onclick="selectCloudDrive('${drive.path.replace(/\\/g, '\\\\')}')" ${scanButtonDisabled}>Scan</button>`
                                    : `<span class="unmounted">Not mounted</span>`
                                }
                            </div>
                        </li>
                    `;
                });
                
                html += `
                        </ul>
                    </div>
                `;
            });
            
            cloudDrivesContainer.innerHTML = html;
            addLogEntry('Cloud drive detection completed successfully', 'success');
        } else {
            cloudDrivesContainer.innerHTML = '<div class="no-drives">No cloud drives detected</div>';
            addLogEntry('No cloud drives detected on this system', 'warning');
        }
        
    } catch (error) {
        addLogEntry('Failed to detect cloud drives: ' + error, 'error');
        cloudDrivesContainer.innerHTML = '<div class="no-drives">Failed to detect cloud drives</div>';
    }
}

// Select cloud drive for scanning
function selectCloudDrive(path) {
    document.getElementById('scanPath').value = path;
    showToast('Cloud drive path selected', 'success');
}

// Load settings
async function loadSettings() {
    try {
        const settings = await window.go.main.App.GetSettings();
        
        // Apply settings to UI
        if (settings.threadCount > 0) {
            document.getElementById('threadCount').value = settings.threadCount;
        }
        
        if (settings.defaultExportFormat) {
            document.getElementById('exportFormat').value = settings.defaultExportFormat;
        }
        
    } catch (error) {
        console.error('Failed to load settings:', error);
    }
}

// Show toast notification
function showToast(message, type = 'success') {
    // Remove existing toasts
    const existingToasts = document.querySelectorAll('.toast');
    existingToasts.forEach(toast => toast.remove());
    
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.textContent = message;
    
    document.body.appendChild(toast);
    
    // Auto remove after 4 seconds
    setTimeout(() => {
        if (toast.parentNode) {
            toast.parentNode.removeChild(toast);
        }
    }, 4000);
}

// Utility function to humanize bytes
function humanizeBytes(bytes) {
    if (bytes === 0) return '0 B';
    
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    
    return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i];
}

// Global functions for HTML onclick handlers
window.selectCloudDrive = selectCloudDrive;