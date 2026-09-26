// TraceIQ Frontend JavaScript
class TraceIQProcessor {
    constructor() {
        this.processingInterval = null;
        this.currentDbName = '';
    }
    
    init() {
        this.bindEvents();
    }
    
    bindEvents() {
        document.getElementById('timeline-file')?.addEventListener('change', (e) => {
            const fileName = e.target.files[0]?.name || 'No file selected';
            document.getElementById('file-name').textContent = fileName;
        });
    }
    
    startProcessing() {
        const fileInput = document.getElementById('timeline-file');
        const file = fileInput.files[0];
        
        if (!file) {
            alert('Please select a JSON file first.');
            return;
        }
        
        const dbName = document.getElementById('db-name').value || 'timeline.db';
        this.currentDbName = dbName;
        
        // Disable button
        document.getElementById('process-button').disabled = true;
        document.getElementById('process-button').textContent = 'Processing...';
        
        // Show progress
        document.getElementById('progress-container').style.display = 'block';
        document.getElementById('status-container').style.display = 'block';
        document.getElementById('download-section').style.display = 'none';
        
        // Reset stats
        document.getElementById('records-count').textContent = '0';
        document.getElementById('errors-count').textContent = '0';
        document.getElementById('warnings-count').textContent = '0';
        document.getElementById('status-indicator').textContent = 'Running';
        
        // Clear status
        document.getElementById('status-content').innerHTML = '';
        
        // Create form data
        const formData = new FormData();
        formData.append('file', file);
        formData.append('db_name', dbName);
        
        // Start upload and processing
        fetch('/api/upload', {
            method: 'POST',
            body: formData
        })
        .then(response => response.json())
        .then(data => {
            if (data.error) {
                alert('Error: ' + data.error);
                this.resetUI();
                return;
            }
            
            // Start monitoring status
            this.startStatusMonitoring();
        })
        .catch(error => {
            alert('Error: ' + error.message);
            this.resetUI();
        });
    }
    
    startStatusMonitoring() {
        this.processingInterval = setInterval(() => {
            fetch('/api/status')
                .then(response => response.json())
                .then(status => {
                    // Update stats
                    document.getElementById('records-count').textContent = status.records_processed;
                    document.getElementById('errors-count').textContent = status.errors.length;
                    document.getElementById('warnings-count').textContent = status.warnings.length;
                    
                    // Update progress
                    if (status.active) {
                        const progress = Math.min((status.records_processed / 1000) * 100, 90);
                        document.getElementById('progress-fill').style.width = progress + '%';
                        document.getElementById('progress-text').textContent = 
                            `Processing: ${status.records_processed} records...`;
                    }
                    
                    // Update status indicator
                    if (status.active) {
                        document.getElementById('status-indicator').textContent = 'Running';
                    } else if (status.completed) {
                        document.getElementById('status-indicator').textContent = 'Completed';
                        document.getElementById('progress-fill').style.width = '100%';
                        document.getElementById('progress-text').textContent = 'Complete!';
                        
                        // Show download
                        document.getElementById('download-section').style.display = 'block';
                        
                        // Stop monitoring
                        clearInterval(this.processingInterval);
                        
                        // Re-enable button
                        this.resetUI();
                        
                        // Run quality check
                        this.runQualityCheck();
                    }
                    
                    // Update status content
                    const statusContent = document.getElementById('status-content');
                    statusContent.innerHTML = '';
                    
                    if (status.errors.length > 0) {
                        status.errors.slice(-5).forEach(error => {
                            const div = document.createElement('div');
                            div.className = 'status-item error';
                            div.textContent = error;
                            statusContent.appendChild(div);
                        });
                    }
                    
                    if (status.warnings.length > 0) {
                        status.warnings.slice(-5).forEach(warning => {
                            const div = document.createElement('div');
                            div.className = 'status-item warning';
                            div.textContent = warning;
                            statusContent.appendChild(div);
                        });
                    }
                });
        }, 1000);
    }
    
    runQualityCheck() {
        fetch('/api/quality')
            .then(response => response.json())
            .then(data => {
                if (!data.error) {
                    console.log('Quality check:', data);
                    if (data.quality_score < 80) {
                        alert(`Warning: Data quality score is ${data.quality_score}%. Check logs.`);
                    }
                }
            });
    }
    
    resetUI() {
        document.getElementById('process-button').disabled = false;
        document.getElementById('process-button').textContent = 'Process Timeline';
        if (this.processingInterval) {
            clearInterval(this.processingInterval);
        }
    }
    
    downloadDatabase() {
        window.location.href = `/api/download/${this.currentDbName}`;
    }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    const processor = new TraceIQProcessor();
    processor.init();
});
