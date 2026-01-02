import os
import sys
import json
import sqlite3
import hashlib
import base64
import logging
from datetime import datetime
from functools import wraps
from flask import Flask, render_template, request, jsonify, send_file
from flask_cors import CORS
import ijson
import threading

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('data/logs/app.log'),
        logging.StreamHandler(sys.stdout)
    ]
)
logger = logging.getLogger('TraceIQ')

app = Flask(__name__)
CORS(app)

# Database configuration
DB_PATH = os.path.join(os.path.dirname(__file__), 'data', 'processed', 'timeline.db')
os.makedirs(os.path.dirname(DB_PATH), exist_ok=True)

# Processing status
processing_status = {
    'active': False,
    'current_file': None,
    'records_processed': 0,
    'errors': [],
    'warnings': [],
    'completed': False,
    'start_time': None,
    'end_time': None
}

def generate_short_uuid(source_string, length=8):
    """Generate deterministic short UUID"""
    try:
        hash_bytes = hashlib.sha256(source_string.encode()).digest()
        b64_hash = base64.b64encode(hash_bytes).decode('ascii')
        clean_hash = b64_hash.replace('+', '').replace('/', '').replace('=', '')
        return clean_hash[:length]
    except Exception as e:
        logger.error(f"UUID generation failed: {e}")
        return f"err_{length}"

def init_database(db_path):
    """Initialize SQLite database with complete schema"""
    try:
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()
        
        # Enable WAL mode for better concurrency
        cursor.execute("PRAGMA journal_mode=WAL;")
        cursor.execute("PRAGMA foreign_keys=ON;")
        
        # Read and execute schema
        schema_path = os.path.join(os.path.dirname(__file__), '..', 'schema_complete.sql')
        with open(schema_path, 'r') as f:
            schema = f.read()
            cursor.executescript(schema)
        
        conn.commit()
        conn.close()
        logger.info(f"Database initialized at {db_path}")
        
    except Exception as e:
        logger.error(f"Database initialization failed: {e}")
        raise

def process_timeline_json(json_path, db_path):
    """Process timeline with comprehensive error handling"""
    global processing_status
    
    processing_status = {
        'active': True,
        'current_file': os.path.basename(json_path),
        'records_processed': 0,
        'errors': [],
        'warnings': [],
        'completed': False,
        'start_time': datetime.now().isoformat(),
        'end_time': None
    }
    
    logger.info(f"Starting processing of {json_path}")
    
    try:
        # Ensure database exists
        if not os.path.exists(db_path):
            init_database(db_path)
        
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()
        
        # Enable foreign keys for this connection
        cursor.execute("PRAGMA foreign_keys=ON;")
        
        # Process file with streaming
        with open(json_path, 'rb') as f:
            objects = ijson.items(f, 'timelineObjects.item')
            
            for idx, obj in enumerate(objects):
                try:
                    if 'activitySegment' in obj:
                        process_activity_segment(obj['activitySegment'], cursor, idx)
                    elif 'placeVisit' in obj:
                        process_place_visit(obj['placeVisit'], cursor, idx)
                    
                    processing_status['records_processed'] += 1
                    
                    # Commit every 100 records
                    if idx % 100 == 0:
                        conn.commit()
                        logger.info(f"Processed {idx} records")
                        
                except Exception as e:
                    error_msg = f"Record {idx}: {str(e)}"
                    processing_status['errors'].append(error_msg)
                    logger.error(error_msg)
                    conn.rollback()
        
        conn.commit()
        conn.close()
        
        processing_status['completed'] = True
        processing_status['end_time'] = datetime.now().isoformat()
        logger.info(f"Processing complete: {processing_status['records_processed']} records")
        
    except Exception as e:
        error_msg = f"Fatal error: {str(e)}"
        processing_status['errors'].append(error_msg)
        processing_status['completed'] = True
        processing_status['end_time'] = datetime.now().isoformat()
        logger.error(error_msg)
    finally:
        processing_status['active'] = False

def process_activity_segment(segment, cursor, idx):
    """Process activity segment with validation"""
    try:
        # Extract timestamps
        start_time = segment['duration']['startTimestamp']
        end_time = segment['duration']['endTimestamp']
        
        # Validate timestamps
        if not start_time or not end_time:
            raise ValueError("Missing timestamps in segment")
        
        # Generate IDs
        activity_id = generate_short_uuid(f"activity_{start_time}_{end_time}_{idx}")
        event_serial_id = datetime.fromisoformat(start_time.replace('Z', '+00:00')).strftime('%y%m%d%H%M%S') + f".{idx}"
        
        # Extract location data
        start_lat = segment.get('startLocation', {}).get('latitudeE7', 0) / 1e7
        start_lng = segment.get('startLocation', {}).get('longitudeE7', 0) / 1e7
        end_lat = segment.get('endLocation', {}).get('latitudeE7', 0) / 1e7
        end_lng = segment.get('endLocation', {}).get('longitudeE7', 0) / 1e7
        
        start_geopair = f"{start_lat},{start_lng}"
        end_geopair = f"{end_lat},{end_lng}"
        
        # Duration
        duration_seconds = (datetime.fromisoformat(end_time.replace('Z', '+00:00')) - 
                           datetime.fromisoformat(start_time.replace('Z', '+00:00'))).total_seconds()
        
        # Distance
        distance_meters = segment.get('distance', 0)
        
        # Activity type
        activity_type = segment.get('activityType', 'UNKNOWN')
        
        # Insert
        cursor.execute("""
            INSERT OR REPLACE INTO activities (
                activity_id, event_serial_id, start_timestamp_raw, end_timestamp_raw,
                start_timestamp_utc, end_timestamp_utc, duration_seconds, activity_type,
                distance_meters, activity_start_geopair, activity_end_geopair,
                processed_at, data_source, meta_json
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """, (
            activity_id, event_serial_id, start_time, end_time,
            start_time, end_time, int(duration_seconds), activity_type,
            distance_meters, start_geopair, end_geopair,
            datetime.now().isoformat(), 'google_timeline', json.dumps(segment)
        ))
        
    except Exception as e:
        logger.warning(f"Activity segment {idx} skipped: {str(e)}")
        processing_status['warnings'].append(f"Activity {idx}: {str(e)}")

def process_place_visit(place_visit, cursor, idx):
    """Process place visit with validation"""
    try:
        # Extract timestamps
        start_time = place_visit['duration']['startTimestamp']
        end_time = place_visit['duration']['endTimestamp']
        
        if not start_time or not end_time:
            raise ValueError("Missing timestamps in visit")
        
        # Generate IDs
        visit_id = generate_short_uuid(f"visit_{start_time}_{end_time}_{idx}")
        event_serial_id = datetime.fromisoformat(start_time.replace('Z', '+00:00')).strftime('%y%m%d%H%M%S') + f".{idx}"
        
        # Location
        location = place_visit.get('location', {})
        lat = location.get('latitudeE7', 0) / 1e7
        lng = location.get('longitudeE7', 0) / 1e7
        geopair = f"{lat},{lng}"
        
        # Place info
        place_id = location.get('placeId', '')
        place_name = location.get('name', '')
        
        # Duration
        duration_seconds = (datetime.fromisoformat(end_time.replace('Z', '+00:00')) - 
                           datetime.fromisoformat(start_time.replace('Z', '+00:00'))).total_seconds()
        
        # Insert
        cursor.execute("""
            INSERT OR REPLACE INTO visits (
                visit_id, event_serial_id, start_timestamp_raw, end_timestamp_raw,
                start_timestamp_utc, end_timestamp_utc, duration_seconds, visit_place_id,
                visit_geopair, processed_at, data_source, meta_json
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """, (
            visit_id, event_serial_id, start_time, end_time,
            start_time, end_time, int(duration_seconds), place_id,
            geopair, datetime.now().isoformat(), 'google_timeline', json.dumps(place_visit)
        ))
        
    except Exception as e:
        logger.warning(f"Place visit {idx} skipped: {str(e)}")
        processing_status['warnings'].append(f"Visit {idx}: {str(e)}")

@app.route('/')
def index():
    return render_template('index.html')

@app.route('/api/upload', methods=['POST'])
def upload_file():
    if 'file' not in request.files:
        logger.error("No file in request")
        return jsonify({'error': 'No file provided'}), 400
    
    file = request.files['file']
    if file.filename == '':
        logger.error("Empty filename")
        return jsonify({'error': 'No file selected'}), 400
    
    if not file.filename.endswith('.json'):
        logger.error(f"Invalid file type: {file.filename}")
        return jsonify({'error': 'File must be JSON format'}), 400
    
    # Save file
    temp_path = os.path.join('/tmp', file.filename)
    file.save(temp_path)
    logger.info(f"File saved to {temp_path}")
    
    # Get options
    db_name = request.form.get('db_name', 'timeline.db')
    db_path = os.path.join(os.path.dirname(__file__), '..', 'data', 'processed', db_name)
    
    # Start processing
    thread = threading.Thread(target=process_timeline_json, args=(temp_path, db_path))
    thread.daemon = True
    thread.start()
    
    logger.info(f"Processing started for {db_path}")
    
    return jsonify({
        'message': 'Processing started',
        'db_path': db_path,
        'status_url': '/api/status'
    })

@app.route('/api/status')
def get_status():
    return jsonify(processing_status)

@app.route('/api/download/<filename>')
def download_file(filename):
    file_path = os.path.join(os.path.dirname(__file__), '..', 'data', 'processed', filename)
    if os.path.exists(file_path):
        logger.info(f"Download requested: {file_path}")
        return send_file(file_path, as_attachment=True)
    logger.error(f"File not found: {file_path}")
    return jsonify({'error': 'File not found'}), 404

@app.route('/api/quality')
def quality_check():
    """Run quality checks on processed data"""
    db_path = os.path.join(os.path.dirname(__file__), '..', 'data', 'processed', 'timeline.db')
    
    if not os.path.exists(db_path):
        return jsonify({'error': 'Database not found'}), 404
    
    try:
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()
        
        # Check 1: Missing coordinates
        cursor.execute("SELECT COUNT(*) FROM timeline_enriched WHERE location_geopair IS NULL")
        missing_coords = cursor.fetchone()[0]
        
        # Check 2: Impossible speeds
        cursor.execute("""
            SELECT COUNT(*) FROM timeline_enriched 
            WHERE delta_distance_meters > 0 AND delta_duration_seconds > 0 
            AND (delta_distance_meters / delta_duration_seconds) > 50
        """)
        impossible_speeds = cursor.fetchone()[0]
        
        # Check 3: Low confidence
        cursor.execute("SELECT COUNT(*) FROM timeline_enriched WHERE probability < 0.5")
        low_confidence = cursor.fetchone()[0]
        
        # Check 4: Total records
        cursor.execute("SELECT COUNT(*) FROM timeline_enriched")
        total_records = cursor.fetchone()[0]
        
        conn.close()
        
        quality_score = 100.0
        if total_records > 0:
            quality_score -= (missing_coords / total_records) * 20
            quality_score -= (impossible_speeds / total_records) * 30
            quality_score -= (low_confidence / total_records) * 10
        
        return jsonify({
            'total_records': total_records,
            'missing_coordinates': missing_coords,
            'impossible_speeds': impossible_speeds,
            'low_confidence': low_confidence,
            'quality_score': round(quality_score, 1)
        })
        
    except Exception as e:
        logger.error(f"Quality check failed: {str(e)}")
        return jsonify({'error': str(e)}), 500

@app.route('/api/export/<format>')
def export_data(format):
    """Export data in various formats"""
    if format not in ['csv', 'json', 'parquet']:
        return jsonify({'error': 'Invalid format'}), 400
    
    # Implementation would export from SQLite
    # For now, placeholder
    return jsonify({'message': 'Export functionality coming soon'})

if __name__ == '__main__':
    logger.info("Starting TraceIQ Timeline Processor")
    app.run(debug=True, host='0.0.0.0', port=5000)
