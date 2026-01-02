"""
Google Timeline Data Parser - Core Parsing Logic
Validated against: 2018-Q1 and 2024-Q2 Google Takeout exports
Version: 1.0 (Multi-device aware)

This module handles the parsing of Google Timeline semanticSegments into
structured database records with multi-device detection and splitting.
"""

import json
from datetime import datetime
from math import radians, cos, sin, asin, sqrt
from typing import Dict, List, Tuple, Optional, Any


# ============================================================================
# CONSTANTS
# ============================================================================

MULTI_DEVICE_THRESHOLD_METERS = 100  # Distance threshold for multi-device detection


# ============================================================================
# COORDINATE PARSING
# ============================================================================

def parse_latlng(latlng_str: str) -> Tuple[float, float]:
    """
    Parse Google's coordinate format: '43.1234°, -83.5678°'
    
    Args:
        latlng_str: Coordinate string with degree symbols
        
    Returns:
        Tuple of (latitude, longitude) as floats
        
    Example:
        >>> parse_latlng("43.1234°, -83.5678°")
        (43.1234, -83.5678)
    """
    parts = latlng_str.replace('°', '').split(',')
    lat = float(parts[0].strip())
    lon = float(parts[1].strip())
    return lat, lon


def haversine_distance(lat1: float, lon1: float, lat2: float, lon2: float) -> float:
    """
    Calculate distance between two points in meters using Haversine formula.
    
    Args:
        lat1, lon1: First point coordinates
        lat2, lon2: Second point coordinates
        
    Returns:
        Distance in meters
    """
    # Convert to radians
    lon1, lat1, lon2, lat2 = map(radians, [lon1, lat1, lon2, lat2])
    
    # Haversine formula
    dlon = lon2 - lon1
    dlat = lat2 - lat1
    a = sin(dlat/2)**2 + cos(lat1) * cos(lat2) * sin(dlon/2)**2
    c = 2 * asin(sqrt(a))
    r = 6371000  # Radius of Earth in meters
    
    return c * r


# ============================================================================
# MULTI-DEVICE DETECTION
# ============================================================================

def find_multi_device_duplicates(
    path: List[Dict[str, str]], 
    threshold: float = MULTI_DEVICE_THRESHOLD_METERS
) -> List[Tuple[int, int]]:
    """
    Detect duplicate timestamps with significant spatial separation.
    
    This indicates multiple devices recording simultaneously.
    
    Args:
        path: List of waypoint dicts with 'time' and 'point' keys
        threshold: Distance threshold in meters (default: 100m)
        
    Returns:
        List of (idx1, idx2) tuples where duplicates occur
        
    Example:
        >>> path = [
        ...     {'time': '2024-04-01T16:41:00.000-04:00', 'point': '43.113125°, -83.617342°'},
        ...     {'time': '2024-04-01T16:41:00.000-04:00', 'point': '43.116687°, -83.618398°'}
        ... ]
        >>> find_multi_device_duplicates(path)
        [(0, 1)]  # 405m apart - definitely two devices
    """
    duplicates = []
    
    for i in range(len(path) - 1):
        if path[i]['time'] == path[i+1]['time']:
            # Same timestamp - check distance
            lat1, lon1 = parse_latlng(path[i]['point'])
            lat2, lon2 = parse_latlng(path[i+1]['point'])
            
            dist = haversine_distance(lat1, lon1, lat2, lon2)
            
            if dist > threshold:
                duplicates.append((i, i+1))
    
    return duplicates


def split_path_on_duplicates(
    path: List[Dict[str, str]], 
    duplicates: List[Tuple[int, int]]
) -> List[List[Dict[str, str]]]:
    """
    Split a path into device-specific sub-paths at duplicate timestamps.
    
    Strategy: Split at first duplicate only (conservative approach)
    - Device 0: waypoints [0 to first_duplicate_idx]
    - Device 1: waypoints [first_duplicate_idx+1 to end]
    
    Args:
        path: Original waypoint path
        duplicates: List of duplicate index pairs from find_multi_device_duplicates()
        
    Returns:
        List of device paths (2 paths if split, 1 path if no split)
        
    Example:
        >>> path = [wp0, wp1, ..., wp9, wp10, wp11, wp12]  # duplicate at (9, 10)
        >>> split_path_on_duplicates(path, [(9, 10)])
        [[wp0, ..., wp9], [wp10, wp11, wp12]]  # Two device paths
    """
    if not duplicates:
        return [path]
    
    # Split at first duplicate only
    split_idx = duplicates[0][0]
    
    device0_path = path[:split_idx + 1]  # Include first duplicate point
    device1_path = path[split_idx + 1:]  # Start from second duplicate point
    
    return [device0_path, device1_path]


# ============================================================================
# TIMELINE EVENT PARSING
# ============================================================================

def parse_visit(segment: Dict[str, Any], segment_index: int) -> Dict[str, Any]:
    """
    Parse a 'visit' segment into timeline_events table format.
    
    Args:
        segment: semanticSegment dict with 'visit' key
        segment_index: Position in semanticSegments array
        
    Returns:
        Dict ready for insertion into timeline_events table
    """
    visit = segment['visit']
    top_candidate = visit.get('topCandidate', {})
    place_location = top_candidate.get('placeLocation', {})
    
    # Parse coordinates if available
    lat, lon = None, None
    if 'latLng' in place_location:
        lat, lon = parse_latlng(place_location['latLng'])
    
    return {
        'event_id': f"visit_{segment_index}",
        'event_type': 'visit',
        'segment_index': segment_index,
        'start_time': segment['startTime'],
        'end_time': segment['endTime'],
        'start_tz_offset': segment.get('startTimeTimezoneUtcOffsetMinutes'),
        'end_tz_offset': segment.get('endTimeTimezoneUtcOffsetMinutes'),
        
        # Visit-specific fields
        'visit_hierarchy_level': visit.get('hierarchyLevel'),
        'visit_probability': float(visit.get('probability', 0)),
        'visit_place_id': top_candidate.get('placeId'),
        'visit_place_name': None,  # Can be enriched later with geocoding
        'visit_semantic_type': top_candidate.get('semanticType'),
        'visit_place_probability': float(top_candidate.get('probability', 0)),
        'visit_latitude': lat,
        'visit_longitude': lon,
        
        # Activity fields (NULL for visits)
        'activity_type': None,
        'activity_probability': None,
        'activity_distance_meters': None,
        'activity_start_latitude': None,
        'activity_start_longitude': None,
        'activity_end_latitude': None,
        'activity_end_longitude': None,
        
        # Split tracking
        'is_split': False,
        'split_reason': None
    }


def parse_activity(segment: Dict[str, Any], segment_index: int) -> Dict[str, Any]:
    """
    Parse an 'activity' segment into timeline_events table format.
    
    Args:
        segment: semanticSegment dict with 'activity' key
        segment_index: Position in semanticSegments array
        
    Returns:
        Dict ready for insertion into timeline_events table
    """
    activity = segment['activity']
    top_candidate = activity.get('topCandidate', {})
    
    # Parse start/end coordinates
    start_lat, start_lon = None, None
    end_lat, end_lon = None, None
    
    if 'start' in activity and 'latLng' in activity['start']:
        start_lat, start_lon = parse_latlng(activity['start']['latLng'])
    
    if 'end' in activity and 'latLng' in activity['end']:
        end_lat, end_lon = parse_latlng(activity['end']['latLng'])
    
    return {
        'event_id': f"activity_{segment_index}",
        'event_type': 'activity',
        'segment_index': segment_index,
        'start_time': segment['startTime'],
        'end_time': segment['endTime'],
        'start_tz_offset': segment.get('startTimeTimezoneUtcOffsetMinutes'),
        'end_tz_offset': segment.get('endTimeTimezoneUtcOffsetMinutes'),
        
        # Visit fields (NULL for activities)
        'visit_hierarchy_level': None,
        'visit_probability': None,
        'visit_place_id': None,
        'visit_place_name': None,
        'visit_semantic_type': None,
        'visit_place_probability': None,
        'visit_latitude': None,
        'visit_longitude': None,
        
        # Activity-specific fields
        'activity_type': top_candidate.get('type'),
        'activity_probability': float(top_candidate.get('probability', 0)),
        'activity_distance_meters': float(activity.get('distanceMeters', 0)),
        'activity_start_latitude': start_lat,
        'activity_start_longitude': start_lon,
        'activity_end_latitude': end_lat,
        'activity_end_longitude': end_lon,
        
        # Split tracking
        'is_split': False,
        'split_reason': None
    }


# ============================================================================
# WAYPOINT PARSING (with Multi-Device Support)
# ============================================================================

def create_waypoints(
    path: List[Dict[str, str]],
    parent_id: str,
    parent_type: str,
    segment_index: int,
    multi_device_split: bool = False,
    device_index: Optional[int] = None,
    split_from_segment: Optional[int] = None
) -> List[Dict[str, Any]]:
    """
    Convert a timelinePath into individual waypoint records.
    
    Args:
        path: List of waypoint dicts with 'time' and 'point' keys
        parent_id: Event or orphaned path ID this path belongs to
        parent_type: 'visit', 'activity', or 'orphaned_path'
        segment_index: Position in semanticSegments array
        multi_device_split: True if this path was split due to multi-device
        device_index: 0 or 1 for device-specific paths, None otherwise
        split_from_segment: Original segment_index if split
        
    Returns:
        List of dicts ready for insertion into waypoints table
    """
    waypoints = []
    
    for sequence, wp in enumerate(path, start=1):
        lat, lon = parse_latlng(wp['point'])
        
        waypoints.append({
            'parent_id': parent_id,
            'parent_type': parent_type,
            'sequence': sequence,
            'timestamp': wp['time'],
            'latitude': lat,
            'longitude': lon,
            'multi_device_split': multi_device_split,
            'device_index': device_index,
            'split_from_segment': split_from_segment,
            'segment_index': segment_index
        })
    
    return waypoints


def parse_timeline_path(
    segment: Dict[str, Any], 
    segment_index: int,
    parent_id: Optional[str] = None,
    parent_type: Optional[str] = None
) -> List[Dict[str, Any]]:
    """
    Parse a timelinePath segment with multi-device detection and splitting.
    
    This handles:
    1. Orphaned paths (no visit/activity)
    2. Paths attached to visits/activities
    3. Multi-device splitting when duplicate timestamps detected
    
    Args:
        segment: semanticSegment dict with 'timelinePath' key
        segment_index: Position in semanticSegments array
        parent_id: Event ID if attached to visit/activity, None for orphaned
        parent_type: 'visit', 'activity', or None for orphaned
        
    Returns:
        List of waypoint dicts ready for insertion into waypoints table
    """
    if 'timelinePath' not in segment or not segment['timelinePath']:
        return []
    
    path = segment['timelinePath']
    
    # Determine parent
    if parent_id is None:
        parent_id_base = f"orphaned_path_{segment_index}"
        parent_type = 'orphaned_path'
    else:
        parent_id_base = parent_id
    
    # Detect multi-device duplicates
    duplicates = find_multi_device_duplicates(path)
    
    if not duplicates:
        # Simple case: No multi-device splitting needed
        return create_waypoints(
            path, 
            parent_id_base, 
            parent_type, 
            segment_index
        )
    
    # Multi-device case: Split path
    device_paths = split_path_on_duplicates(path, duplicates)
    
    all_waypoints = []
    for device_idx, device_path in enumerate(device_paths):
        device_parent_id = f"{parent_id_base}_device{device_idx}"
        
        waypoints = create_waypoints(
            device_path,
            device_parent_id,
            parent_type,
            segment_index,
            multi_device_split=True,
            device_index=device_idx,
            split_from_segment=segment_index
        )
        
        all_waypoints.extend(waypoints)
    
    return all_waypoints


# ============================================================================
# MAIN SEGMENT PARSER
# ============================================================================

def parse_semantic_segment(
    segment: Dict[str, Any], 
    segment_index: int
) -> Tuple[Optional[Dict[str, Any]], List[Dict[str, Any]]]:
    """
    Parse a single semanticSegment into event + waypoints.
    
    Args:
        segment: Single element from semanticSegments array
        segment_index: Position in array (0-based)
        
    Returns:
        Tuple of (event_dict or None, waypoints_list)
        - event_dict is None for orphaned paths
        - waypoints_list is empty if no timelinePath
    """
    event = None
    waypoints = []
    
    # Parse based on segment type
    if 'visit' in segment:
        event = parse_visit(segment, segment_index)
        
        # Parse timelinePath if attached to visit
        if 'timelinePath' in segment:
            waypoints = parse_timeline_path(
                segment, 
                segment_index,
                parent_id=event['event_id'],
                parent_type='visit'
            )
    
    elif 'activity' in segment:
        event = parse_activity(segment, segment_index)
        
        # Parse timelinePath if attached to activity
        if 'timelinePath' in segment:
            waypoints = parse_timeline_path(
                segment,
                segment_index,
                parent_id=event['event_id'],
                parent_type='activity'
            )
    
    elif 'timelinePath' in segment:
        # Orphaned path (no visit or activity)
        waypoints = parse_timeline_path(segment, segment_index)
    
    return event, waypoints


# ============================================================================
# VALIDATION
# ============================================================================

def validate_event(event: Dict[str, Any]) -> List[str]:
    """
    Validate a parsed event record.
    
    Returns:
        List of error messages (empty if valid)
    """
    errors = []
    
    # Required fields
    if not event.get('event_id'):
        errors.append("Missing event_id")
    if not event.get('event_type'):
        errors.append("Missing event_type")
    if not event.get('start_time'):
        errors.append("Missing start_time")
    if not event.get('end_time'):
        errors.append("Missing end_time")
    
    # Type-specific validation
    if event['event_type'] == 'visit':
        if not event.get('visit_place_id'):
            errors.append("Visit missing place_id")
    elif event['event_type'] == 'activity':
        if not event.get('activity_type'):
            errors.append("Activity missing activity_type")
    
    return errors


def validate_waypoint(waypoint: Dict[str, Any]) -> List[str]:
    """
    Validate a parsed waypoint record.
    
    Returns:
        List of error messages (empty if valid)
    """
    errors = []
    
    # Required fields
    if not waypoint.get('parent_id'):
        errors.append("Missing parent_id")
    if not waypoint.get('timestamp'):
        errors.append("Missing timestamp")
    
    # Coordinate validation
    lat = waypoint.get('latitude')
    lon = waypoint.get('longitude')
    
    if lat is None:
        errors.append("Missing latitude")
    elif not (-90 <= lat <= 90):
        errors.append(f"Invalid latitude: {lat}")
    
    if lon is None:
        errors.append("Missing longitude")
    elif not (-180 <= lon <= 180):
        errors.append(f"Invalid longitude: {lon}")
    
    # Sequence validation
    seq = waypoint.get('sequence')
    if seq is None:
        errors.append("Missing sequence")
    elif seq < 1:
        errors.append(f"Invalid sequence: {seq}")
    
    return errors
