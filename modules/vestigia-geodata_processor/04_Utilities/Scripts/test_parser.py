"""
Test Suite for Google Timeline Parser
Tests against real 2024-Q2 data to validate parsing logic
"""

import json
import sqlite3
from pathlib import Path
from parser import (
    parse_latlng,
    haversine_distance,
    find_multi_device_duplicates,
    split_path_on_duplicates,
    parse_visit,
    parse_activity,
    parse_timeline_path,
    parse_semantic_segment,
    validate_event,
    validate_waypoint
)


def test_coordinate_parsing():
    """Test coordinate string parsing"""
    print("\n" + "="*70)
    print("TEST: Coordinate Parsing")
    print("="*70)
    
    test_cases = [
        ("43.1664629°, -83.7346893°", (43.1664629, -83.7346893)),
        ("43.119800°, -83.617738°", (43.119800, -83.617738)),
        ("43.0042428°, -83.5931585°", (43.0042428, -83.5931585)),
    ]
    
    for coord_str, expected in test_cases:
        result = parse_latlng(coord_str)
        assert abs(result[0] - expected[0]) < 0.000001, f"Latitude mismatch: {result[0]} != {expected[0]}"
        assert abs(result[1] - expected[1]) < 0.000001, f"Longitude mismatch: {result[1]} != {expected[1]}"
        print(f"  ✓ {coord_str} → {result}")
    
    print("  All coordinate parsing tests passed!")


def test_distance_calculation():
    """Test haversine distance calculation"""
    print("\n" + "="*70)
    print("TEST: Distance Calculation")
    print("="*70)
    
    # Known test case: ~405m apart (from real data)
    lat1, lon1 = 43.113125, -83.617342
    lat2, lon2 = 43.116687, -83.618398
    
    dist = haversine_distance(lat1, lon1, lat2, lon2)
    
    print(f"  Point 1: ({lat1}, {lon1})")
    print(f"  Point 2: ({lat2}, {lon2})")
    print(f"  Distance: {dist:.1f}m")
    
    # Should be around 405m
    assert 400 < dist < 410, f"Distance calculation error: {dist}"
    print("  ✓ Distance calculation correct!")


def test_multi_device_detection():
    """Test multi-device duplicate detection"""
    print("\n" + "="*70)
    print("TEST: Multi-Device Detection")
    print("="*70)
    
    # Case 1: No duplicates
    path_clean = [
        {'time': '2024-04-01T16:05:00.000-04:00', 'point': '43.008759°, -83.592431°'},
        {'time': '2024-04-01T16:10:00.000-04:00', 'point': '43.003938°, -83.593357°'},
    ]
    duplicates = find_multi_device_duplicates(path_clean)
    assert duplicates == [], f"Should have no duplicates, got {duplicates}"
    print("  ✓ Clean path: No duplicates detected")
    
    # Case 2: Duplicate timestamps with large separation (405m)
    path_multi = [
        {'time': '2024-04-01T16:41:00.000-04:00', 'point': '43.113125°, -83.617342°'},
        {'time': '2024-04-01T16:41:00.000-04:00', 'point': '43.116687°, -83.618398°'},
        {'time': '2024-04-01T16:49:00.000-04:00', 'point': '43.119041°, -83.619728°'},
    ]
    duplicates = find_multi_device_duplicates(path_multi)
    assert duplicates == [(0, 1)], f"Should detect duplicate at (0,1), got {duplicates}"
    print("  ✓ Multi-device path: Duplicate detected at indices (0, 1)")
    
    # Case 3: Duplicate timestamps with small separation (GPS noise, <100m)
    path_noise = [
        {'time': '2024-04-01T17:10:00.000-04:00', 'point': '43.119800°, -83.617738°'},
        {'time': '2024-04-01T17:10:00.000-04:00', 'point': '43.120342°, -83.617718°'},  # ~60m apart
    ]
    duplicates = find_multi_device_duplicates(path_noise)
    assert duplicates == [], f"Should ignore GPS noise duplicates, got {duplicates}"
    print("  ✓ GPS noise: Sub-100m duplicate ignored")


def test_path_splitting():
    """Test path splitting on duplicates"""
    print("\n" + "="*70)
    print("TEST: Path Splitting")
    print("="*70)
    
    # Build a path with duplicate at indices 9, 10
    path = []
    for i in range(14):
        if i == 9:
            path.append({'time': '2024-04-01T16:41:00.000-04:00', 'point': '43.113125°, -83.617342°'})
        elif i == 10:
            path.append({'time': '2024-04-01T16:41:00.000-04:00', 'point': '43.116687°, -83.618398°'})
        else:
            path.append({'time': f'2024-04-01T16:{i:02d}:00.000-04:00', 'point': '43.0°, -83.0°'})
    
    duplicates = [(9, 10)]
    device_paths = split_path_on_duplicates(path, duplicates)
    
    assert len(device_paths) == 2, f"Should split into 2 paths, got {len(device_paths)}"
    assert len(device_paths[0]) == 10, f"Device 0 should have 10 waypoints, got {len(device_paths[0])}"
    assert len(device_paths[1]) == 4, f"Device 1 should have 4 waypoints, got {len(device_paths[1])}"
    
    print(f"  ✓ Split 14 waypoints → Device 0: {len(device_paths[0])} waypoints")
    print(f"                        → Device 1: {len(device_paths[1])} waypoints")


def test_visit_parsing():
    """Test visit segment parsing"""
    print("\n" + "="*70)
    print("TEST: Visit Parsing")
    print("="*70)
    
    segment = {
        'startTime': '2024-04-01T15:35:45.000-04:00',
        'endTime': '2024-04-01T15:41:42.000-04:00',
        'startTimeTimezoneUtcOffsetMinutes': -240,
        'endTimeTimezoneUtcOffsetMinutes': -240,
        'visit': {
            'hierarchyLevel': 0,
            'probability': '0.9800000190734863',
            'topCandidate': {
                'placeId': 'ChIJg-CPDKOHI4gRO7I4zq6fIxU',
                'semanticType': 'UNKNOWN',
                'probability': '0.6169196963310242',
                'placeLocation': {
                    'latLng': '43.12106°, -83.6173514°'
                }
            }
        }
    }
    
    event = parse_visit(segment, segment_index=5)
    
    assert event['event_id'] == 'visit_5'
    assert event['event_type'] == 'visit'
    assert event['visit_place_id'] == 'ChIJg-CPDKOHI4gRO7I4zq6fIxU'
    assert abs(event['visit_latitude'] - 43.12106) < 0.00001
    assert abs(event['visit_longitude'] - -83.6173514) < 0.00001
    
    errors = validate_event(event)
    assert errors == [], f"Visit validation failed: {errors}"
    
    print(f"  ✓ Visit parsed: {event['event_id']}")
    print(f"    Place: {event['visit_place_id']}")
    print(f"    Location: ({event['visit_latitude']:.6f}, {event['visit_longitude']:.6f})")


def test_activity_parsing():
    """Test activity segment parsing"""
    print("\n" + "="*70)
    print("TEST: Activity Parsing")
    print("="*70)
    
    segment = {
        'startTime': '2024-04-01T15:06:10.000-04:00',
        'endTime': '2024-04-01T15:35:45.000-04:00',
        'startTimeTimezoneUtcOffsetMinutes': -240,
        'endTimeTimezoneUtcOffsetMinutes': -240,
        'activity': {
            'start': {
                'latLng': '43.1649597°, -83.7319977°'
            },
            'end': {
                'latLng': '43.1210679°, -83.6181803°'
            },
            'distanceMeters': '11909.0',
            'topCandidate': {
                'type': 'IN_PASSENGER_VEHICLE',
                'probability': '0.0'
            }
        }
    }
    
    event = parse_activity(segment, segment_index=2)
    
    assert event['event_id'] == 'activity_2'
    assert event['event_type'] == 'activity'
    assert event['activity_type'] == 'IN_PASSENGER_VEHICLE'
    assert event['activity_distance_meters'] == 11909.0
    assert abs(event['activity_start_latitude'] - 43.1649597) < 0.00001
    
    errors = validate_event(event)
    assert errors == [], f"Activity validation failed: {errors}"
    
    print(f"  ✓ Activity parsed: {event['event_id']}")
    print(f"    Type: {event['activity_type']}")
    print(f"    Distance: {event['activity_distance_meters']:.1f}m")


def test_orphaned_path_parsing():
    """Test orphaned path (timelinePath only) parsing"""
    print("\n" + "="*70)
    print("TEST: Orphaned Path Parsing")
    print("="*70)
    
    segment = {
        'startTime': '2024-04-01T10:00:00.000-04:00',
        'endTime': '2024-04-01T12:00:00.000-04:00',
        'timelinePath': [
            {'point': '43.1664629°, -83.7346893°', 'time': '2024-04-01T11:01:00.000-04:00'},
            {'point': '43.164967°, -83.7315168°', 'time': '2024-04-01T11:02:00.000-04:00'}
        ]
    }
    
    event, waypoints = parse_semantic_segment(segment, segment_index=0)
    
    assert event is None, "Orphaned path should not create event"
    assert len(waypoints) == 2, f"Should create 2 waypoints, got {len(waypoints)}"
    assert waypoints[0]['parent_id'] == 'orphaned_path_0'
    assert waypoints[0]['parent_type'] == 'orphaned_path'
    assert waypoints[0]['sequence'] == 1
    assert waypoints[1]['sequence'] == 2
    
    for wp in waypoints:
        errors = validate_waypoint(wp)
        assert errors == [], f"Waypoint validation failed: {errors}"
    
    print(f"  ✓ Orphaned path parsed: {waypoints[0]['parent_id']}")
    print(f"    Waypoints: {len(waypoints)}")


def test_multi_device_path_parsing():
    """Test path parsing with multi-device splitting"""
    print("\n" + "="*70)
    print("TEST: Multi-Device Path Parsing")
    print("="*70)
    
    # Segment 6 from real data (has duplicate at indices 9, 10)
    segment = {
        'startTime': '2024-04-01T16:00:00.000-04:00',
        'endTime': '2024-04-01T18:00:00.000-04:00',
        'timelinePath': [
            {'point': '43.008759°, -83.592431°', 'time': '2024-04-01T16:05:00.000-04:00'},
            {'point': '43.003938°, -83.593357°', 'time': '2024-04-01T16:10:00.000-04:00'},
            {'point': '43.008759°, -83.592431°', 'time': '2024-04-01T16:12:00.000-04:00'},
            {'point': '43.002443°, -83.593096°', 'time': '2024-04-01T16:16:00.000-04:00'},
            {'point': '43.008759°, -83.592431°', 'time': '2024-04-01T16:18:00.000-04:00'},
            {'point': '43.004156°, -83.593259°', 'time': '2024-04-01T16:20:00.000-04:00'},
            {'point': '43.008759°, -83.592431°', 'time': '2024-04-01T16:23:00.000-04:00'},
            {'point': '43.003424°, -83.594555°', 'time': '2024-04-01T16:25:00.000-04:00'},
            {'point': '43.022221°, -83.589768°', 'time': '2024-04-01T16:27:00.000-04:00'},
            {'point': '43.113125°, -83.617342°', 'time': '2024-04-01T16:41:00.000-04:00'},  # Device A
            {'point': '43.116687°, -83.618398°', 'time': '2024-04-01T16:41:00.000-04:00'},  # Device B
            {'point': '43.119041°, -83.619728°', 'time': '2024-04-01T16:49:00.000-04:00'},
            {'point': '43.118801°, -83.619726°', 'time': '2024-04-01T17:02:00.000-04:00'},
            {'point': '43.116882°, -83.618380°', 'time': '2024-04-01T17:08:00.000-04:00'}
        ]
    }
    
    event, waypoints = parse_semantic_segment(segment, segment_index=6)
    
    assert event is None, "Orphaned path should not create event"
    
    # Should split into two device paths
    device0_waypoints = [wp for wp in waypoints if wp['device_index'] == 0]
    device1_waypoints = [wp for wp in waypoints if wp['device_index'] == 1]
    
    assert len(device0_waypoints) == 10, f"Device 0 should have 10 waypoints, got {len(device0_waypoints)}"
    assert len(device1_waypoints) == 4, f"Device 1 should have 4 waypoints, got {len(device1_waypoints)}"
    
    # Verify parent IDs
    assert all(wp['parent_id'] == 'orphaned_path_6_device0' for wp in device0_waypoints)
    assert all(wp['parent_id'] == 'orphaned_path_6_device1' for wp in device1_waypoints)
    
    # Verify multi_device_split flag
    assert all(wp['multi_device_split'] for wp in waypoints)
    assert all(wp['split_from_segment'] == 6 for wp in waypoints)
    
    print(f"  ✓ Multi-device path split successfully")
    print(f"    Original: 14 waypoints")
    print(f"    Device 0: {len(device0_waypoints)} waypoints (parent_id: {device0_waypoints[0]['parent_id']})")
    print(f"    Device 1: {len(device1_waypoints)} waypoints (parent_id: {device1_waypoints[0]['parent_id']})")


def run_all_tests():
    """Run complete test suite"""
    print("\n" + "="*70)
    print("GOOGLE TIMELINE PARSER TEST SUITE")
    print("="*70)
    
    try:
        test_coordinate_parsing()
        test_distance_calculation()
        test_multi_device_detection()
        test_path_splitting()
        test_visit_parsing()
        test_activity_parsing()
        test_orphaned_path_parsing()
        test_multi_device_path_parsing()
        
        print("\n" + "="*70)
        print("✓ ALL TESTS PASSED")
        print("="*70)
        return True
        
    except AssertionError as e:
        print(f"\n❌ TEST FAILED: {e}")
        return False
    except Exception as e:
        print(f"\n❌ UNEXPECTED ERROR: {e}")
        import traceback
        traceback.print_exc()
        return False


if __name__ == '__main__':
    success = run_all_tests()
    exit(0 if success else 1)
