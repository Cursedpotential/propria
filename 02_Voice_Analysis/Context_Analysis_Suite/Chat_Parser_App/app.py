import os
import psycopg2
import re
import sqlparse
import json
import csv
import io
import logging
from datetime import datetime
from flask import Flask, render_template_string, request, jsonify, Response

app = Flask(__name__)

# ============================================================================
# LOGGING SETUP - Comprehensive logging for every action
# ============================================================================
os.makedirs('logs', exist_ok=True)
os.makedirs('schemas', exist_ok=True)
os.makedirs('uploads', exist_ok=True)

LOGFILE = 'logs/traceiq.log'
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler(LOGFILE, encoding='utf-8')
    ]
)
logger = logging.getLogger(__name__)

def log_action(action, details, status='INFO'):
    """Log every user action with timestamp and status"""
    message = f"[{status}] {action}: {details}"
    if status == 'ERROR':
        logger.error(message)
    elif status == 'WARNING':
        logger.warning(message)
    else:
        logger.info(message)

# ============================================================================
# DATABASE CONFIGURATION
# ============================================================================
DB_HOST = os.getenv('DB_HOST', 'postgres')
DB_PORT = os.getenv('DB_PORT', 5432)
DB_NAME = os.getenv('DB_NAME', 'traceiq')
DB_USER = os.getenv('DB_USER', 'traceiq')
DB_PASSWORD = os.getenv('DB_PASSWORD', 'forensic123')

def get_db_connection():
    """Establish PostgreSQL connection"""
    try:
        conn = psycopg2.connect(
            host=DB_HOST,
            port=DB_PORT,
            database=DB_NAME,
            user=DB_USER,
            password=DB_PASSWORD
        )
        return conn
    except Exception as e:
        logger.error(f"Database connection failed: {e}")
        log_action('DATABASE_CONNECT', f'Failed: {e}', 'ERROR')
        raise

# ============================================================================
# SQL EXTRACTION & CATEGORIZATION
# ============================================================================
def extract_sql_from_markdown(content):
    """Extract SQL from markdown - handles sql-tagged blocks separately"""
    logger.info("Starting markdown SQL extraction")
    log_action('EXTRACT_SQL', 'Starting extraction from markdown', 'INFO')

    sql_blocks = []

    # Capture fenced code blocks and accept only those explicitly labeled 'sql'
    # Match ```sql (with optional whitespace/newline after) through closing ```
    pattern = re.compile(r"```\s*sql\s*\n(.*?)```", re.DOTALL | re.IGNORECASE)
    for m in pattern.finditer(content):
        block = m.group(1).strip()
        if block:
            sql_blocks.append(block)

    logger.info(f"Found {len(sql_blocks)} SQL blocks")
    log_action('EXTRACT_SQL', f'Extracted {len(sql_blocks)} blocks', 'SUCCESS')
    return sql_blocks

def clean_sql_comments(sql_text):
    """Remove SQL line comments (--) while preserving code structure"""
    lines = sql_text.split('\n')
    cleaned = []
    for line in lines:
        # Normalize whitespace
        raw = line.rstrip()
        # Skip explicit FOREIGN KEY / CONSTRAINT / REFERENCES lines to avoid ordering problems
        if re.search(r"\bFOREIGN\s+KEY\b", raw, re.IGNORECASE) or re.search(r"\bCONSTRAINT\b", raw, re.IGNORECASE) or re.search(r"\bREFERENCES\b", raw, re.IGNORECASE):
            continue

        if '--' in raw:
            # Split on first -- and keep only the part before
            before_comment = raw.split('--', 1)[0].rstrip()
            if before_comment:
                cleaned.append(before_comment)
        else:
            cleaned.append(raw)

    # Remove consecutive blank lines and return
    out_lines = []
    for ln in cleaned:
        if not out_lines:
            out_lines.append(ln)
            continue
        if ln.strip() == '' and out_lines[-1].strip() == '':
            continue
        out_lines.append(ln)
    return '\n'.join(out_lines)

def categorize_sql_statements(sql_blocks_list):
    """Categorize SQL statements by type - process each block separately for robustness"""
    statements = {
        'tables': [],
        'indexes': [],
        'views': []
    }
    
    # Process each block separately to avoid cross-block splitting issues
    for block in sql_blocks_list:
        # Clean comments from this block
        clean_block = clean_sql_comments(block)
        
        # Split this block's statements
        raw_statements = [s.strip() for s in sqlparse.split(clean_block) if s.strip()]
        
        # Reattach fragments (statements that don't start with CREATE/ALTER)
        reattached = []
        for stmt in raw_statements:
            if stmt and not re.match(r'^\s*(CREATE|ALTER)', stmt, re.IGNORECASE):
                # Fragment - try to append to previous statement
                if reattached:
                    reattached[-1] += '\n' + stmt
                else:
                    # No previous statement; skip orphan fragment
                    continue
            else:
                reattached.append(stmt)
        
        # Now categorize the cleaned statements
        for stmt in reattached:
            if not stmt.strip():
                continue
            stmt_upper = stmt.upper()
            if 'CREATE TABLE' in stmt_upper:
                statements['tables'].append(stmt)
            elif 'CREATE INDEX' in stmt_upper or stmt_upper.startswith('CREATE INDEX'):
                statements['indexes'].append(stmt)
            elif 'CREATE VIEW' in stmt_upper:
                statements['views'].append(stmt)
            elif 'ALTER TABLE' in stmt_upper:
                statements['tables'].append(stmt)
    
    logger.info(f"Categorized {len(statements['tables'])} tables, {len(statements['indexes'])} indexes, {len(statements['views'])} views")
    log_action('CATEGORIZE_SQL', f'Tables: {len(statements["tables"])}, Indexes: {len(statements["indexes"])}, Views: {len(statements["views"])}', 'INFO')
    return statements

# ============================================================================
# DATABASE INFO & SCHEMA QUERIES
# ============================================================================
def get_database_info():
    """Get comprehensive database statistics"""
    try:
        conn = get_db_connection()
        cursor = conn.cursor()
        
        # Get tables
        cursor.execute("""
            SELECT schemaname, tablename 
            FROM pg_catalog.pg_tables 
            WHERE schemaname = 'public' 
            AND tablename NOT IN ('geography_columns', 'geometry_columns', 'spatial_ref_sys')
            ORDER BY tablename
        """)
        tables_raw = cursor.fetchall()
        
        tables = []
        for schema, table in tables_raw:
            try:
                cursor.execute(f"SELECT COUNT(*) FROM {schema}.{table}")
                row_count = cursor.fetchone()[0]
                tables.append({'schema': schema, 'name': table, 'rows': row_count})
            except:
                tables.append({'schema': schema, 'name': table, 'rows': 0})
        
        # Get indexes
        cursor.execute("""
            SELECT indexname, tablename 
            FROM pg_indexes 
            WHERE schemaname = 'public'
            ORDER BY tablename, indexname
        """)
        indexes = cursor.fetchall()
        
        # Get views
        cursor.execute("""
            SELECT table_name FROM information_schema.views 
            WHERE table_schema = 'public' 
            ORDER BY table_name
        """)
        views = [row[0] for row in cursor.fetchall()]
        
        conn.close()
        
        info = {
            'tables': tables,
            'indexes': [{'name': idx[0], 'table': idx[1]} for idx in indexes],
            'views': views,
            'total_tables': len(tables),
            'total_indexes': len(indexes),
            'total_views': len(views),
            'total_rows': sum(t['rows'] for t in tables)
        }
        
        log_action('GET_STATS', f'Tables: {info["total_tables"]}, Indexes: {info["total_indexes"]}, Views: {info["total_views"]}, Rows: {info["total_rows"]}', 'SUCCESS')
        return info
    except Exception as e:
        logger.error(f"Database info error: {e}")
        log_action('GET_STATS', f'Failed: {e}', 'ERROR')
        return {
            'tables': [], 'indexes': [], 'views': [],
            'total_tables': 0, 'total_indexes': 0, 'total_views': 0, 'total_rows': 0
        }

# ============================================================================
# FLASK ROUTES - API ENDPOINTS
# ============================================================================
@app.route('/')
def index():
    """Main dashboard page"""
    log_action('PAGE_LOAD', 'Dashboard accessed', 'INFO')
    db_info = get_database_info()
    return render_template_string(HTML_TEMPLATE, db_info=db_info)

@app.route('/api/tables')
def get_tables_api():
    """API: Get all tables with stats"""
    log_action('API_CALL', 'Tables list requested', 'INFO')
    return jsonify(get_database_info())

@app.route('/api/export/<table_name>')
def export_table(table_name):
    """API: Export table data as CSV or JSON"""
    format_type = request.args.get('format', 'csv')
    log_action('EXPORT', f'Table: {table_name}, Format: {format_type}', 'INFO')
    
    try:
        conn = get_db_connection()
        cursor = conn.cursor()
        cursor.execute(f"SELECT * FROM {table_name} LIMIT 10000")
        
        columns = [desc[0] for desc in cursor.description]
        rows = cursor.fetchall()
        conn.close()
        
        if format_type == 'csv':
            output = io.StringIO()
            writer = csv.writer(output)
            writer.writerow(columns)
            writer.writerows(rows)
            response = Response(output.getvalue(), mimetype='text/csv')
            response.headers['Content-Disposition'] = f'attachment; filename={table_name}.csv'
            log_action('EXPORT', f'CSV export successful: {len(rows)} rows', 'SUCCESS')
            return response
        else:
            data = [dict(zip(columns, row)) for row in rows]
            response = Response(json.dumps(data, indent=2, default=str), mimetype='application/json')
            response.headers['Content-Disposition'] = f'attachment; filename={table_name}.json'
            log_action('EXPORT', f'JSON export successful: {len(rows)} rows', 'SUCCESS')
            return response
            
    except Exception as e:
        log_action('EXPORT', f'Failed: {e}', 'ERROR')
        return jsonify({'error': str(e)}), 500

@app.route('/api/export-schema')
def export_schema():
    """API: Export complete database schema as SQL"""
    log_action('EXPORT', 'Schema export requested', 'INFO')
    try:
        conn = get_db_connection()
        cursor = conn.cursor()
        
        cursor.execute("""
            SELECT table_name FROM information_schema.tables 
            WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
        """)
        tables = [row[0] for row in cursor.fetchall()]
        
        schema_sql = "-- TraceIQ Schema\n\n"
        for table in tables:
            cursor.execute(f"SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name = '{table}'")
            columns = cursor.fetchall()
            
            schema_sql += f"CREATE TABLE {table} (\n"
            schema_sql += ",\n".join([f"    {col[0]} {col[1]} {'NOT NULL' if col[2] == 'NO' else ''}" for col in columns])
            schema_sql += "\n);\n\n"
        
        conn.close()
        
        response = Response(schema_sql, mimetype='text/plain')
        response.headers['Content-Disposition'] = 'attachment; filename=schema.sql'
        log_action('EXPORT', 'Schema export successful', 'SUCCESS')
        return response
        
    except Exception as e:
        log_action('EXPORT', f'Failed: {e}', 'ERROR')
        return jsonify({'error': str(e)}), 500

@app.route('/api/upload-schema', methods=['POST'])
def upload_schema():
    """API: Upload and execute schema with category selection"""
    log_action('SCHEMA_UPLOAD', 'Schema upload started', 'INFO')
    
    if 'schema' not in request.files:
        log_action('SCHEMA_UPLOAD', 'No file provided', 'ERROR')
        return jsonify({'error': 'No file provided'}), 400
    
    file = request.files['schema']
    mode = request.form.get('mode', 'all')
    filename = file.filename
    log_action('SCHEMA_UPLOAD', f'File: {filename}, Mode: {mode}', 'INFO')
    
    # Save uploaded file
    filepath = f'schemas/{filename}'
    os.makedirs('schemas', exist_ok=True)
    file.save(filepath)
    log_action('SCHEMA_UPLOAD', f'File saved: {filepath}', 'INFO')
    
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()
        
        # Extract SQL blocks
        if filename.endswith('.md'):
            sql_blocks = extract_sql_from_markdown(content)
            if not sql_blocks:
                log_action('SCHEMA_UPLOAD', 'No SQL blocks found in markdown', 'ERROR')
                return jsonify({'error': 'No SQL blocks found in markdown'}), 400
        else:
            # For non-markdown files, treat entire content as single block
            sql_blocks = [content]
        
        # Categorize statements (pass blocks list directly for per-block processing)
        categorized = categorize_sql_statements(sql_blocks)

        dry_run = request.form.get('dry', 'false').lower() == 'true'

        conn = get_db_connection()
        cursor = conn.cursor()

        executed = 0
        skipped = 0
        
        # Execute in correct order: tables first, then indexes, then views
        if mode in ['all', 'tables']:
            for i, stmt in enumerate(categorized['tables']):
                    sp_name = f"sp_tables_{i}"
                    try:
                        cursor.execute(f"SAVEPOINT {sp_name}")
                        cursor.execute(stmt)
                        cursor.execute(f"RELEASE SAVEPOINT {sp_name}")
                        executed += 1
                        log_action('SCHEMA_UPLOAD_TABLE', f'Executed: {stmt[:80]}...', 'SUCCESS')
                    except Exception as e:
                        skipped += 1
                        try:
                            cursor.execute(f"ROLLBACK TO SAVEPOINT {sp_name}")
                        except Exception:
                            pass
                        log_action('SCHEMA_UPLOAD_TABLE', f'Failed: {e}', 'ERROR')
        
        if mode in ['all', 'indexes']:
            for i, stmt in enumerate(categorized['indexes']):
                sp_name = f"sp_indexes_{i}"
                try:
                    cursor.execute(f"SAVEPOINT {sp_name}")
                    cursor.execute(stmt)
                    cursor.execute(f"RELEASE SAVEPOINT {sp_name}")
                    executed += 1
                    log_action('SCHEMA_UPLOAD_INDEX', f'Executed: {stmt[:80]}...', 'SUCCESS')
                except Exception as e:
                    skipped += 1
                    try:
                        cursor.execute(f"ROLLBACK TO SAVEPOINT {sp_name}")
                    except Exception:
                        pass
                    log_action('SCHEMA_UPLOAD_INDEX', f'Failed: {e}', 'ERROR')
        
        if mode in ['all', 'views']:
            for i, stmt in enumerate(categorized['views']):
                sp_name = f"sp_views_{i}"
                try:
                    cursor.execute(f"SAVEPOINT {sp_name}")
                    cursor.execute(stmt)
                    cursor.execute(f"RELEASE SAVEPOINT {sp_name}")
                    executed += 1
                    log_action('SCHEMA_UPLOAD_VIEW', f'Executed: {stmt[:80]}...', 'SUCCESS')
                except Exception as e:
                    skipped += 1
                    try:
                        cursor.execute(f"ROLLBACK TO SAVEPOINT {sp_name}")
                    except Exception:
                        pass
                    log_action('SCHEMA_UPLOAD_VIEW', f'Failed: {e}', 'ERROR')
        
        # If dry-run, rollback any changes so nothing is persisted
        if dry_run:
            try:
                conn.rollback()
            except Exception:
                pass
        else:
            conn.commit()
        conn.close()
        
        log_action('SCHEMA_UPLOAD', f'Complete: {executed} executed, {skipped} skipped', 'SUCCESS')
        
        return jsonify({
            'mode': mode,
            'dry_run': dry_run,
            'tables_found': len(categorized['tables']),
            'indexes_found': len(categorized['indexes']),
            'views_found': len(categorized['views']),
            'executed': executed,
            'skipped': skipped
        })
        
    except Exception as e:
        log_action('SCHEMA_UPLOAD', f'Fatal error: {e}', 'ERROR')
        return jsonify({'error': str(e)}), 500

@app.route('/api/upload-data', methods=['POST'])
def upload_data():
    """API: Import data from JSON files"""
    log_action('DATA_UPLOAD', 'Data upload started', 'INFO')
    
    if 'json' not in request.files:
        log_action('DATA_UPLOAD', 'No file provided', 'ERROR')
        return jsonify({'error': 'No JSON file provided'}), 400
    
    file = request.files['json']
    log_action('DATA_UPLOAD', f'File: {file.filename}', 'INFO')
    
    try:
        data = json.load(file)
        log_action('DATA_UPLOAD', f'Loaded {len(data)} records', 'INFO')
        
        if not isinstance(data, list) or len(data) == 0:
            log_action('DATA_UPLOAD', 'Invalid JSON format', 'ERROR')
            return jsonify({'error': 'Invalid JSON format'}), 400
        
        # Determine target table
        target_table = 'imported_data'
        if hasattr(data[0], 'keys'):
            sample_keys = list(data[0].keys())
            if 'eventid' in sample_keys or 'serialid' in sample_keys:
                target_table = 'timeline_enriched'
        
        log_action('DATA_UPLOAD', f'Target table: {target_table}', 'INFO')
        
        conn = get_db_connection()
        cursor = conn.cursor()
        
        # Get table columns
        cursor.execute("""
            SELECT column_name FROM information_schema.columns 
            WHERE table_name = %s AND table_schema = 'public'
        """, (target_table,))
        columns = [row[0] for row in cursor.fetchall()]
        
        inserted = 0
        for i, record in enumerate(data):
            record_keys = [k for k in record.keys() if k in columns]
            if not record_keys:
                continue
            
            placeholders = ', '.join(['%s'] * len(record_keys))
            columns_str = ', '.join(record_keys)
            
            sql = f"INSERT INTO {target_table} ({columns_str}) VALUES ({placeholders})"
            
            try:
                cursor.execute(sql, [record[k] for k in record_keys])
                inserted += 1
                if i % 100 == 0:
                    log_action('DATA_UPLOAD', f'Inserted {i}/{len(data)} records', 'INFO')
            except Exception as e:
                log_action('DATA_UPLOAD', f'Failed record {i}: {e}', 'WARNING')
        
        conn.commit()
        conn.close()
        
        log_action('DATA_UPLOAD', f'Complete: {inserted} records into {target_table}', 'SUCCESS')
        return jsonify({
            'records': inserted,
            'target_table': target_table
        })
        
    except Exception as e:
        log_action('DATA_UPLOAD', f'Failed: {e}', 'ERROR')
        return jsonify({'error': str(e)}), 500

@app.route('/api/create-view', methods=['POST'])
def create_view():
    """API: Create database view"""
    log_action('CREATE_VIEW', 'View creation requested', 'INFO')
    
    data = request.json
    view_name = data.get('name')
    view_sql = data.get('sql')
    
    if not view_name or not view_sql:
        log_action('CREATE_VIEW', 'Missing name or SQL', 'ERROR')
        return jsonify({'error': 'View name and SQL required'}), 400
    
    try:
        conn = get_db_connection()
        cursor = conn.cursor()
        cursor.execute(f"CREATE VIEW {view_name} AS {view_sql}")
        conn.commit()
        conn.close()
        
        log_action('CREATE_VIEW', f'View created: {view_name}', 'SUCCESS')
        return jsonify({'success': True})
    except Exception as e:
        log_action('CREATE_VIEW', f'Failed: {e}', 'ERROR')
        return jsonify({'error': str(e)}), 500

@app.route('/api/run-query', methods=['POST'])
def run_query():
    """API: Execute custom SQL query"""
    log_action('CUSTOM_QUERY', 'Custom query execution requested', 'INFO')
    
    data = request.json
    query = data.get('query')
    
    if not query:
        log_action('CUSTOM_QUERY', 'No query provided', 'ERROR')
        return jsonify({'error': 'Query required'}), 400
    
    try:
        conn = get_db_connection()
        cursor = conn.cursor()
        cursor.execute(query)
        
        if cursor.description:
            columns = [desc[0] for desc in cursor.description]
            rows = cursor.fetchall()
        else:
            columns = []
            rows = []
        
        conn.close()
        
        log_action('CUSTOM_QUERY', f'Executed: {len(rows)} rows returned', 'SUCCESS')
        return jsonify({'columns': columns, 'rows': rows})
    except Exception as e:
        log_action('CUSTOM_QUERY', f'Failed: {e}', 'ERROR')
        return jsonify({'error': str(e)}), 500

@app.route('/api/logs')
def get_logs():
    """API: Get application logs"""
    log_action('API_CALL', 'Logs API requested', 'INFO')
    try:
        with open(LOGFILE, 'r', encoding='utf-8') as f:
            logs = f.read()
        return jsonify({'logs': logs})
    except:
        return jsonify({'logs': 'No logs yet'})

@app.route('/logs')
def log_viewer():
    """Log viewer page"""
    log_action('PAGE_LOAD', 'Log viewer accessed', 'INFO')
    return LOG_VIEWER_HTML

@app.route('/api/clear-logs', methods=['POST'])
def clear_logs():
    """API: Clear log file"""
    open(LOGFILE, 'w').close()
    log_action('LOGS', 'Logs cleared by user', 'INFO')
    return jsonify({'success': True})

# ============================================================================
# COMPLETE HTML TEMPLATE - ALL PANELS INCLUDED
# ============================================================================
HTML_TEMPLATE = """
<!DOCTYPE html>
<html>
<head>
    <title>TraceIQ Enhanced</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: 'Segoe UI', Arial, sans-serif; background: #f0f2f5; }
        .layout { display: flex; height: 100vh; }
        .sidebar { width: 280px; background: #2c3e50; color: white; overflow-y: auto; }
        .sidebar-header { background: #34495e; padding: 20px; }
        .sidebar-header h2 { font-size: 1.3em; }
        .sidebar-section { padding: 15px; border-bottom: 1px solid #34495e; }
        .sidebar-section h3 { font-size: 0.85em; text-transform: uppercase; opacity: 0.7; margin-bottom: 10px; }
        .sidebar-item { padding: 10px 15px; border-radius: 5px; cursor: pointer; margin: 5px 0; transition: 0.2s; }
        .sidebar-item:hover { background: #34495e; }
        .sidebar-item.active { background: #3498db; }
        .stat-badge { background: #e74c3c; color: white; padding: 2px 8px; border-radius: 10px; font-size: 0.75em; float: right; }
        .main-content { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
        .top-bar { background: white; border-bottom: 1px solid #e0e0e0; padding: 20px 30px; display: flex; justify-content: space-between; align-items: center; }
        .top-bar h1 { color: #2c3e50; font-size: 1.8em; }
        .content-area { flex: 1; overflow-y: auto; padding: 30px; }
        .card { background: white; border-radius: 10px; padding: 25px; margin-bottom: 20px; box-shadow: 0 2px 5px rgba(0,0,0,0.1); }
        .card h2 { color: #2c3e50; margin-bottom: 15px; padding-bottom: 10px; border-bottom: 2px solid #3498db; }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; margin: 20px 0; }
        .stat-card { background: linear-gradient(135deg, #3498db 0%, #2980b9 100%); color: white; padding: 20px; border-radius: 10px; text-align: center; cursor: pointer; transition: 0.3s; }
        .stat-card:hover { transform: translateY(-5px); box-shadow: 0 10px 20px rgba(0,0,0,0.2); }
        .stat-number { font-size: 2.5em; font-weight: bold; }
        .stat-label { font-size: 0.9em; opacity: 0.9; margin-top: 5px; }
        .upload-zone { border: 3px dashed #3498db; border-radius: 10px; padding: 40px; text-align: center; background: #ecf0f1; cursor: pointer; transition: 0.3s; }
        .upload-zone:hover { background: #d5dbdb; transform: translateY(-2px); }
        .file-input { display: none; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th { background: #34495e; color: white; padding: 12px; text-align: left; }
        td { padding: 10px; border-bottom: 1px solid #e0e0e0; }
        tr:hover { background: #ecf0f1; }
        .button { background: linear-gradient(135deg, #27ae60, #229954); color: white; border: none; padding: 12px 25px; border-radius: 8px; cursor: pointer; font-size: 14px; font-weight: bold; transition: 0.3s; margin: 5px; display: inline-block; text-decoration: none; }
        .button:hover { transform: translateY(-2px); box-shadow: 0 5px 15px rgba(0,0,0,0.2); }
        .button.secondary { background: linear-gradient(135deg, #95a5a6, #7f8c8d); }
        .status { padding: 15px; border-radius: 8px; margin: 15px 0; }
        .status.success { background: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
        .status.error { background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
        .status.info { background: #d1ecf1; color: #0c5460; border: 1px solid #bee5eb; }
        .section { display: none; }
        .section.active { display: block; animation: fadeIn 0.3s; }
        @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
        select, input, textarea { padding: 10px; width: 100%; margin: 10px 0; border-radius: 5px; border: 1px solid #bdc3c7; font-size: 14px; }
        .sql-block { background: #34495e; color: #ecf0f1; padding: 15px; margin: 10px; border-radius: 5px; cursor: grab; display: inline-block; }
        .sql-block:hover { background: #2c3e50; }
        .query-builder { min-height: 200px; border: 2px dashed #bdc3c7; border-radius: 10px; padding: 20px; background: #ecf0f1; }
        .regex-tester { background: #fff3cd; padding: 20px; border-radius: 10px; margin: 15px 0; }
        .category-info { background: #e8f4f8; padding: 15px; border-radius: 8px; margin: 10px 0; }
        .category-count { font-size: 1.5em; font-weight: bold; color: #2c3e50; }
    </style>
</head>
<body>
    <div class="layout">
        <div class="sidebar">
            <div class="sidebar-header">
                <h2>TraceIQ Enhanced</h2>
                <p>Production-Ready Manager</p>
            </div>
            <div class="sidebar-section">
                <h3>Navigation</h3>
                <div class="sidebar-item active" onclick="showSection('dashboard')">Dashboard</div>
                <div class="sidebar-item" onclick="showSection('import-schema')">Import Schema</div>
                <div class="sidebar-item" onclick="showSection('import-data')">Import Data</div>
                <div class="sidebar-item" onclick="showSection('tables')">Tables <span class="stat-badge" id="stat-tables">{{ db_info.total_tables }}</span></div>
                <div class="sidebar-item" onclick="showSection('indexes')">Indexes <span class="stat-badge" id="stat-indexes">{{ db_info.total_indexes }}</span></div>
                <div class="sidebar-item" onclick="showSection('views')">Views <span class="stat-badge" id="stat-views">{{ db_info.total_views }}</span></div>
                <div class="sidebar-item" onclick="showSection('sql-builder')">SQL Builder</div>
                <div class="sidebar-item" onclick="showSection('regex-helper')">Regex Helper</div>
                <div class="sidebar-item" onclick="showSection('export')">Export</div>
                <div class="sidebar-item" onclick="window.open('/logs', '_blank')">Logs</div>
            </div>
            <div class="sidebar-section">
                <h3>Stats</h3>
                <div style="padding: 10px; font-size: 0.9em;">
                    <div>Tables: <strong>{{ db_info.total_tables }}</strong></div>
                    <div>Indexes: <strong>{{ db_info.total_indexes }}</strong></div>
                    <div>Views: <strong>{{ db_info.total_views }}</strong></div>
                    <div>Rows: <strong>{{ "{:,}".format(db_info.total_rows) }}</strong></div>
                </div>
            </div>
        </div>
        
        <div class="main-content">
            <div class="top-bar">
                <h1 id="page-title">Dashboard</h1>
                <button class="button" onclick="window.location.reload()">Refresh</button>
            </div>
            
            <div class="content-area">
                <!-- DASHBOARD SECTION -->
                <div id="dashboard-section" class="section active">
                    <div class="card">
                        <h2>System Overview</h2>
                        <div class="stats-grid">
                            <div class="stat-card" onclick="showSection('tables')">
                                <div class="stat-number">{{ db_info.total_tables }}</div>
                                <div class="stat-label">Tables</div>
                            </div>
                            <div class="stat-card" onclick="showSection('indexes')">
                                <div class="stat-number">{{ db_info.total_indexes }}</div>
                                <div class="stat-label">Indexes</div>
                            </div>
                            <div class="stat-card" onclick="showSection('views')">
                                <div class="stat-number">{{ db_info.total_views }}</div>
                                <div class="stat-label">Views</div>
                            </div>
                            <div class="stat-card" onclick="showSection('export')">
                                <div class="stat-number">{{ "{:,}".format(db_info.total_rows) }}</div>
                                <div class="stat-label">Total Rows</div>
                            </div>
                        </div>
                    </div>
                </div>
                
                <!-- IMPORT SCHEMA SECTION -->
                <div id="import-schema-section" class="section">
                    <div class="card">
                        <h2>Import Schema</h2>
                        <p>Upload SQL or Markdown files. Select category to import:</p>
                        <select id="schema-mode">
                            <option value="all">All (Recommended - Tables First)</option>
                            <option value="tables">Tables Only</option>
                            <option value="indexes">Indexes Only</option>
                            <option value="views">Views Only</option>
                        </select>
                        <div class="upload-zone" onclick="document.getElementById('schema-file').click()">
                            <p style="font-size: 1.3em;">📁 Click to Upload Schema (.sql or .md)</p>
                        </div>
                        <input type="file" id="schema-file" class="file-input" accept=".sql,.md" onchange="uploadSchema()">
                        <div id="import-schema-status"></div>
                    </div>
                </div>
                
                <!-- IMPORT DATA SECTION -->
                <div id="import-data-section" class="section">
                    <div class="card">
                        <h2>Import Data (JSON)</h2>
                        <div class="upload-zone" onclick="document.getElementById('data-file').click()">
                            <p style="font-size: 1.3em;">📁 Click to Upload JSON Data</p>
                        </div>
                        <input type="file" id="data-file" class="file-input" accept=".json" onchange="uploadData()">
                        <div id="import-data-status"></div>
                    </div>
                </div>
                
                <!-- TABLES SECTION -->
                <div id="tables-section" class="section">
                    <div class="card">
                        <h2>Tables</h2>
                        <table>
                            <thead><tr><th>Table</th><th>Rows</th><th>Actions</th></tr></thead>
                            <tbody id="tables-tbody">
                                {% for table in db_info.tables %}
                                <tr>
                                    <td><strong>{{ table.name }}</strong></td>
                                    <td>{{ "{:,}".format(table.rows) }}</td>
                                    <td>
                                        <a href="/api/export/{{ table.name }}?format=csv" class="button">CSV</a>
                                        <a href="/api/export/{{ table.name }}?format=json" class="button">JSON</a>
                                    </td>
                                </tr>
                                {% endfor %}
                            </tbody>
                        </table>
                    </div>
                </div>
                
                <!-- INDEXES SECTION -->
                <div id="indexes-section" class="section">
                    <div class="card">
                        <h2>Indexes</h2>
                        <div id="indexes-list">
                            {% if db_info.indexes %}
                            <table>
                                <thead><tr><th>Index</th><th>Table</th></tr></thead>
                                <tbody>
                                    {% for idx in db_info.indexes %}
                                    <tr><td>{{ idx.name }}</td><td>{{ idx.table }}</td></tr>
                                    {% endfor %}
                                </tbody>
                            </table>
                            {% else %}
                            <p>No indexes found</p>
                            {% endif %}
                        </div>
                    </div>
                </div>
                
                <!-- VIEWS SECTION -->
                <div id="views-section" class="section">
                    <div class="card">
                        <h2>Views</h2>
                        <div id="views-list">
                            {% if db_info.views %}
                            <ul style="list-style: none; padding: 0;">
                                {% for view in db_info.views %}
                                <li style="padding: 10px; margin: 5px 0; background: #ecf0f1; border-radius: 5px;">{{ view }}</li>
                                {% endfor %}
                            </ul>
                            {% else %}
                            <p>No views found</p>
                            {% endif %}
                        </div>
                        <h3 style="margin-top: 30px;">Create New View</h3>
                        <input type="text" id="view-name" placeholder="View name">
                        <textarea id="view-sql" rows="8" placeholder="SELECT ..."></textarea>
                        <button class="button" onclick="createView()">Create View</button>
                        <div id="view-status"></div>
                    </div>
                </div>
                
                <!-- SQL BUILDER SECTION -->
                <div id="sql-builder-section" class="section">
                    <div class="card">
                        <h2>SQL Builder</h2>
                        <div style="margin-bottom: 20px;">
                            <div class="sql-block" draggable="true">SELECT</div>
                            <div class="sql-block" draggable="true">FROM tablename</div>
                            <div class="sql-block" draggable="true">WHERE column = value</div>
                            <div class="sql-block" draggable="true">ORDER BY column</div>
                            <div class="sql-block" draggable="true">LIMIT 100</div>
                        </div>
                        <textarea id="built-query" class="query-builder" rows="10" placeholder="Build your query here..."></textarea>
                        <button class="button" onclick="runQuery()">Run Query</button>
                        <button class="button secondary" onclick="clearQuery()">Clear</button>
                        <div id="query-result"></div>
                    </div>
                </div>
                
                <!-- REGEX HELPER SECTION -->
                <div id="regex-helper-section" class="section">
                    <div class="card">
                        <h2>Regex Helper</h2>
                        <button class="button" onclick="useRegex('\\d+')">Numbers [0-9]</button>
                        <button class="button" onclick="useRegex('[A-Za-z]+')">Letters [A-Za-z]</button>
                        <button class="button" onclick="useRegex('\\d{3}-\\d{3}-\\d{4}')">Phone (3-3-4)</button>
                        <button class="button" onclick="useRegex('[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}')">Email</button>
                        <input type="text" id="regex-pattern" placeholder="Enter regex pattern">
                        <textarea id="regex-test-text" rows="5" placeholder="Enter test text..."></textarea>
                        <button class="button" onclick="testRegex()">Test Regex</button>
                        <div class="regex-tester" id="regex-result"></div>
                    </div>
                </div>
                
                <!-- EXPORT SECTION -->
                <div id="export-section" class="section">
                    <div class="card">
                        <h2>Export Data</h2>
                        <select id="export-select">
                            <option value="">-- Select Table --</option>
                            {% for table in db_info.tables %}
                            <option value="{{ table.name }}">{{ table.name }}</option>
                            {% endfor %}
                        </select>
                        <button class="button" onclick="exportData('csv')">Export CSV</button>
                        <button class="button" onclick="exportData('json')">Export JSON</button>
                        <button class="button secondary" onclick="exportSchema()">Export Schema</button>
                    </div>
                </div>
            </div>
        </div>
    </div>
    
    <script>
        // Navigation
        function showSection(section) {
            document.querySelectorAll('.section').forEach(s => s.classList.remove('active'));
            document.getElementById(section + '-section').classList.add('active');
            document.querySelectorAll('.sidebar-item').forEach(item => item.classList.remove('active'));
            event.target.classList.add('active');
            document.getElementById('page-title').textContent = section.replace('-', ' ').replace(/\\b\\w/g, function(l){ return l.toUpperCase(); });
        }
        
        // Import Schema
        function uploadSchema() {
            const file = document.getElementById('schema-file').files[0];
            if (!file) return;
            
            const mode = document.getElementById('schema-mode').value;
            const formData = new FormData();
            formData.append('schema', file);
            formData.append('mode', mode);
            document.getElementById('import-schema-status').innerHTML = '<div class="status info">Processing...</div>';
            
            fetch('/api/upload-schema', { method: 'POST', body: formData })
                .then(r => r.json())
                .then(data => {
                    if (data.error) {
                        document.getElementById('import-schema-status').innerHTML = `<div class="status error">${data.error}</div>`;
                    } else {
                        document.getElementById('import-schema-status').innerHTML = `
                            <div class="status success">
                                Success! Mode: ${data.mode}<br>
                                Tables: ${data.tables_found}, Indexes: ${data.indexes_found}, Views: ${data.views_found}<br>
                                Executed: ${data.executed}, Skipped: ${data.skipped}
                            </div>`;
                        setTimeout(() => window.location.reload(), 2000);
                    }
                });
        }
        
        // Import Data
        function uploadData() {
            const file = document.getElementById('data-file').files[0];
            if (!file) return;
            
            const formData = new FormData();
            formData.append('json', file);
            document.getElementById('import-data-status').innerHTML = '<div class="status info">Analyzing...</div>';
            
            fetch('/api/upload-data', { method: 'POST', body: formData })
                .then(r => r.json())
                .then(data => {
                    if (data.error) {
                        document.getElementById('import-data-status').innerHTML = `<div class="status error">${data.error}</div>`;
                    } else {
                        document.getElementById('import-data-status').innerHTML = `
                            <div class="status success">Imported ${data.records} records into ${data.target_table}</div>`;
                    }
                });
        }
        
        // Create View
        function createView() {
            const name = document.getElementById('view-name').value;
            const sql = document.getElementById('view-sql').value;
            if (!name || !sql) {
                alert('Enter view name and SQL');
                return;
            }
            
            fetch('/api/create-view', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({name: name, sql: sql})
            })
            .then(r => r.json())
            .then(data => {
                document.getElementById('view-status').innerHTML = data.error ? 
                    `<div class="status error">${data.error}</div>` : 
                    '<div class="status success">View created</div>';
                if (!data.error) window.location.reload();
            });
        }
        
        // Run Query
        function runQuery() {
            const query = document.getElementById('built-query').value;
            if (!query) {
                alert('Enter query');
                return;
            }
            
            fetch('/api/run-query', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({query: query})
            })
            .then(r => r.json())
            .then(data => {
                if (data.error) {
                    document.getElementById('query-result').innerHTML = `<div class="status error">${data.error}</div>`;
                } else {
                    let html = '<table><thead><tr>';
                    data.columns.forEach(col => html += `<th>${col}</th>`);
                    html += '</tr></thead><tbody>';
                    data.rows.forEach(row => {
                        html += '<tr>';
                        row.forEach(cell => html += `<td>${cell}</td>`);
                        html += '</tr>';
                    });
                    html += '</tbody></table>';
                    document.getElementById('query-result').innerHTML = html;
                }
            });
        }
        
        function clearQuery() {
            document.getElementById('built-query').value = '';
            document.getElementById('query-result').innerHTML = '';
        }
        
        // Regex Helper
        function useRegex(pattern) {
            document.getElementById('regex-pattern').value = pattern;
        }
        
        function testRegex() {
            const pattern = document.getElementById('regex-pattern').value;
            const text = document.getElementById('regex-test-text').value;
            try {
                const regex = new RegExp(pattern, 'g');
                const matches = text.match(regex);
                document.getElementById('regex-result').innerHTML = matches ? 
                    `<strong>${matches.length} matches</strong><br>${matches.map(m => `<code>${m}</code>`).join(', ')}` : 
                    '<strong>No matches</strong>';
            } catch (e) {
                document.getElementById('regex-result').innerHTML = `<strong style="color: red">Invalid: ${e.message}</strong>`;
            }
        }
        
        // Export
        function exportData(format) {
            const table = document.getElementById('export-select').value;
            if (!table) {
                alert('Select table');
                return;
            }
            window.location.href = `/api/export/${table}?format=${format}`;
        }
        
        function exportSchema() {
            window.location.href = '/api/export-schema';
        }
        
        // Drag & Drop for SQL Builder
        document.addEventListener('DOMContentLoaded', function() {
            const blocks = document.querySelectorAll('.sql-block');
            const builder = document.getElementById('built-query');
            
            blocks.forEach(block => {
                block.addEventListener('dragstart', e => {
                    e.dataTransfer.setData('text', block.textContent);
                });
            });
            
            if (builder) {
                builder.addEventListener('dragover', e => e.preventDefault());
                builder.addEventListener('drop', e => {
                    e.preventDefault();
                    builder.value += e.dataTransfer.getData('text') + ' ';
                });
            }
        });
    </script>
</body>
</html>
"""

# ============================================================================
# LOG VIEWER HTML
# ============================================================================
LOG_VIEWER_HTML = """
<!DOCTYPE html>
<html>
<head>
    <title>TraceIQ Logs</title>
    <style>
        body { font-family: monospace; background: #1e1e1e; color: #d4d4d4; padding: 20px; }
        .logs { background: #0d1117; padding: 20px; border-radius: 8px; height: 80vh; overflow-y: auto; white-space: pre-wrap; font-size: 14px; }
        .refresh { background: #238636; color: white; padding: 10px 20px; border: none; border-radius: 5px; cursor: pointer; margin-bottom: 10px; margin-right: 10px; }
        .error { color: #f85149; }
        .success { color: #238636; }
        .info { color: #58a6ff; }
    </style>
</head>
<body>
    <button class="refresh" onclick="loadLogs()">Refresh Logs</button>
    <button class="refresh" onclick="clearLogs()">Clear Logs</button>
    <button class="refresh" onclick="window.close()">Close</button>
    <div id="logs">Loading logs...</div>
    <script>
        function loadLogs() {
            fetch('/api/logs')
                .then(r => r.json())
                .then(data => {
                    document.getElementById('logs').textContent = data.logs;
                    document.getElementById('logs').scrollTop = document.getElementById('logs').scrollHeight;
                });
        }
        
        function clearLogs() {
            fetch('/api/clear-logs', {method: 'POST'})
                .then(() => loadLogs());
        }
        
        loadLogs();
        setInterval(loadLogs, 5000);
    </script>
</body>
</html>
"""

# ============================================================================
# APPLICATION ENTRY POINT
# ============================================================================
if __name__ == '__main__':
    logger.info("=" * 60)
    logger.info("TRACEIQ APPLICATION STARTING")
    logger.info(f"Database: {DB_HOST}:{DB_PORT}/{DB_NAME}")
    logger.info(f"Log file: {LOGFILE}")
    logger.info("=" * 60)
    
    # Ensure directories exist
    os.makedirs('schemas', exist_ok=True)
    os.makedirs('uploads', exist_ok=True)
    os.makedirs('logs', exist_ok=True)
    
    app.run(host='0.0.0.0', port=5000, debug=False)
