#!/usr/bin/env python3
import sqlite3
import sys

print("1. Testing database connection...")
try:
    conn = sqlite3.connect('./locations.db')
    print("✓ Connected to database")
    
    cursor = conn.cursor()
    print("✓ Created cursor")
    
    # Test simple query
    cursor.execute("SELECT name FROM sqlite_master WHERE type='table' LIMIT 1")
    result = cursor.fetchone()
    print(f"✓ Simple query works: {result}")
    
    # Test table creation
    cursor.execute("CREATE TABLE IF NOT EXISTS test_table (id INTEGER PRIMARY KEY)")
    print("✓ CREATE TABLE IF NOT EXISTS works")
    
    # Test dropping the test table
    cursor.execute("DROP TABLE IF EXISTS test_table")
    print("✓ DROP TABLE works")
    
    conn.close()
    print("✓ All tests passed!")
    
except Exception as e:
    print(f"✗ Error: {e}")
    sys.exit(1)
