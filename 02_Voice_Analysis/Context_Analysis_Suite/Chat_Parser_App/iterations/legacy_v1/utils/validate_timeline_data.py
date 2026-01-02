#!/usr/bin/env python3
"""
Validates timeline data for integrity issues like duplicate IDs,
missing data, and inconsistent formats.
"""
import pandas as pd
from pathlib import Path
import logging
import sys
from collections import Counter

# Configuration
PROJECT_ROOT = Path(__file__).resolve().parent.parent
OUTPUT_DIR = PROJECT_ROOT / "data/processed"

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

def validate_csv(csv_path):
    """Validate timeline CSV data for common issues."""
    logging.info(f"Validating CSV: {csv_path}")
    
    # Load data
    df = pd.read_csv(csv_path, low_memory=False)
    
    issues = []
    
    # Check for duplicate IDs
    id_counts = Counter(df['event_id'])
    duplicates = {id_: count for id_, count in id_counts.items() if count > 1}
    
    if duplicates:
        issues.append(f"Found {len(duplicates)} duplicate event IDs")
        logging.warning("Duplicate IDs found:")
        for id_, count in list(duplicates.items())[:10]:  # Show first 10
            logging.warning(f"  ID '{id_}' appears {count} times")
    
    # Check for missing required fields
    required_fields = ['event_id', 'event_type', 'start_time']
    for field in required_fields:
        missing_count = df[field].isna().sum()
        if missing_count > 0:
            issues.append(f"Missing {field}: {missing_count} records")
    
    # Validate timestamps
    df['start_time_parsed'] = pd.to_datetime(df['start_time'], errors='coerce')
    invalid_timestamps = df['start_time_parsed'].isna().sum()
    if invalid_timestamps > 0:
        issues.append(f"Invalid start_time format: {invalid_timestamps} records")
    
    # Validate coordinates
    def validate_coords(coord_str):
        if pd.isna(coord_str):
            return True
        try:
            parts = str(coord_str).split(',')
            if len(parts) != 2:
                return False
            lat, lng = float(parts[0]), float(parts[1])
            return -90 <= lat <= 90 and -180 <= lng <= 180
        except:
            return False
    
    coord_fields = ['start_latlng', 'end_latlng', 'point_latlng']
    for field in coord_fields:
        if field in df.columns:
            invalid_coords = (~df[field].isna() & ~df[field].apply(validate_coords)).sum()
            if invalid_coords > 0:
                issues.append(f"Invalid {field} coordinates: {invalid_coords} records")
    
    # Check event type distribution
    event_types = df['event_type'].value_counts()
    logging.info("Event type distribution:")
    for event_type, count in event_types.items():
        logging.info(f"  {event_type}: {count}")
    
    # Check for orphaned points (points without valid parent_id)
    points_df = df[df['event_type'] == 'point']
    if not points_df.empty:
        all_parent_ids = set(df[df['event_type'] != 'point']['event_id'])
        orphaned_points = points_df[~points_df['parent_id'].isin(all_parent_ids)]
        if not orphaned_points.empty:
            issues.append(f"Orphaned points (no parent): {len(orphaned_points)} records")
    
    # Summary
    if issues:
        logging.warning(f"\nValidation found {len(issues)} issue(s):")
        for issue in issues:
            logging.warning(f"  - {issue}")
    else:
        logging.info("✓ Validation passed - no issues found!")
    
    return len(issues) == 0

def get_latest_csv():
    """Find the most recent CSV file from Pass 1 or Pass 2."""
    all_dirs = sorted([d for d in OUTPUT_DIR.iterdir() if d.is_dir()], reverse=True)
    
    for dir_ in all_dirs:
        # Check for Pass 2 output first
        final_csv = dir_ / 'final_master_timeline.csv'
        if final_csv.exists():
            return final_csv
        
        # Then check for Pass 1 output
        pass1_csv = dir_ / 'pass_1_output.csv'
        if pass1_csv.exists():
            return pass1_csv
    
    return None

def main():
    if len(sys.argv) > 1:
        csv_path = Path(sys.argv[1])
    else:
        csv_path = get_latest_csv()
    
    if not csv_path or not csv_path.exists():
        logging.error("No CSV file found to validate")
        sys.exit(1)
    
    is_valid = validate_csv(csv_path)
    sys.exit(0 if is_valid else 1)

if __name__ == '__main__':
    main()
