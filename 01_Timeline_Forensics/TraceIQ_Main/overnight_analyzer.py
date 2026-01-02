#!/usr/bin/env python3
"""
ANALYSIS MODULE: Overnight Stay Detector
"""
import pandas as pd
from pathlib import Path
import logging
import sys
import sqlite3
from datetime import time, timedelta

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

def parse_offset(offset_str):
    if not isinstance(offset_str, str) or offset_str == 'Z': return timedelta(hours=0)
    try:
        sign = -1 if offset_str.startswith('-') else 1
        h, m = map(int, offset_str.replace('+', '').replace('-', '').split(':'))
        return sign * timedelta(hours=h, minutes=m)
    except:
        return timedelta(hours=0)

def is_overnight(start_dt_utc, end_dt_utc, offset):
    if not pd.notna(start_dt_utc) or not pd.notna(end_dt_utc): return False
    start_local = start_dt_utc + offset
    end_local = end_dt_utc + offset
    return start_local.date() != end_local.date()

def main(db_path_str):
    db_path = Path(db_path_str)
    if not db_path.exists():
        logging.error(f"Database not found at {db_path}"); return

    logging.info(f"Running Overnight Stay Analyzer on {db_path.name}...")
    conn = None
    try:
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()
        query = "SELECT custom_id, start_time, end_time, start_timezone_offset, start_latlng, place_id FROM timeline_events WHERE event_type = 'visit'"
        df = pd.read_sql_query(query, conn, parse_dates={'start_time': {'utc': True}, 'end_time': {'utc': True}})
        
        if df.empty:
            logging.info("No visit events to analyze for overnight stays."); return

        df['offset'] = df['start_timezone_offset'].apply(parse_offset)
        df['is_overnight_stay'] = df.apply(lambda row: is_overnight(row['start_time'], row['end_time'], row['offset']), axis=1)
        
        overnight_df = df[df['is_overnight_stay']].copy()
        
        if overnight_df.empty:
            logging.info("Analysis complete. No new overnight stays were detected."); return

        logging.info(f"Found {len(overnight_df)} overnight stays. Updating database...")
        
        ids_to_flag = overnight_df['custom_id'].tolist()
        cursor.executemany("UPDATE timeline_events SET is_overnight = 1 WHERE custom_id = ? ", [(id,) for id in ids_to_flag])
        
        overnight_events = [
            (row['custom_id'], row['start_time'].strftime('%Y-%m-%d'), row['start_latlng'], row['place_id'], f"Overnight stay from {row['start_time']} to {row['end_time']}")
            for _, row in overnight_df.iterrows()
        ]
        
        cursor.executemany("INSERT OR IGNORE INTO overnight_stays (timeline_custom_id, stay_date, location_coords, location_place_id, notes) VALUES (?, ?, ?, ?, ?)", overnight_events)
        
        conn.commit()
        logging.info(f"Successfully flagged {len(ids_to_flag)} events and inserted/ignored {len(overnight_events)} records into 'overnight_stays'.")
    except Exception as e:
        if conn: conn.rollback()
        logging.error(f"An error occurred during overnight analysis: {e}", exc_info=True)
    finally:
        if conn: conn.close()

if __name__ == '__main__':
    if len(sys.argv) < 2: sys.exit("Usage: python overnight_analyzer.py <path_to_database>")
    main(sys.argv[1])