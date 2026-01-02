# Enhanced Geocoding System - Usage Guide

## Overview
This comprehensive geocoding system combines Radar.io and Google Places APIs to provide enhanced location data with cross-validation, accuracy calculations, and automated processing.

## Setup Instructions

### 1. API Key Configuration
Replace the placeholder API keys in the CFG object:
```javascript
const CFG = {
  RADAR_API_KEY: 'your_actual_radar_key_here',
  GOOGLE_API_KEY: 'your_actual_google_key_here',
  // ... other settings
};
```

### 2. Menu System
Once installed, you'll see a "🗺️ Geocoding Tools" menu with:

#### Individual Tasks:
- **Process Next Empty Column**: Automatically geocodes coordinates in column A
- **Generate Map Links**: Creates Google Maps, Place ID, and Radar map links
- **Calculate GPS Accuracy**: Analyzes coordinate precision
- **Google Reverse Geocode**: Manual Google reverse geocoding

#### Analysis Tools:
- **Calculate Overnight Stays**: Calculates nights between check-in/check-out dates
- **Compare Radar vs Google**: Cross-validates results between services
- **Accuracy Report**: Generates GPS precision analysis

## Key Functions

### Primary Functions:
- `ENHANCED_GEOCODE(coords, placeId)` - Returns comprehensive geocoding data
- `GOOGLE_PLACE_LOOKUP(placeId)` - Gets Google Place details
- `GOOGLE_REVERSE_GEOCODE(coords)` - Manual Google reverse geocoding
- `GENERATE_MAP_LINKS(coords, placeId)` - Creates map URLs
- `CALCULATE_GPS_ACCURACY(coords)` - Analyzes coordinate precision

### Legacy Functions (Enhanced):
- `RADAR_REVERSE_GEOCODE(coords)` - Radar-only geocoding
- `RADAR_PLACE_NAME(coords)` - Place name or "Residence"
- `RADAR_BOTH(coords)` - Returns address and place name array

## Features

### ✅ Google Place ID Integration
- Automatic Place ID lookup when available
- Cross-validation between Radar and Google results
- Mismatch flagging with confidence scores

### ✅ Enhanced Data Structure
Returns structured object with:
- Primary address/name (Google preferred)
- Source attribution (Radar/Google)
- Place type (first in Google types array)
- Raw API responses (flattened JSON)
- Match status and confidence
- Residence detection
- GPS accuracy analysis
- Map links (Google, Radar static, Radar path)

### ✅ Smart Fallbacks
1. **Google Places** (if Place ID available) - Preferred
2. **Radar Place Layer** (business/venue names)
3. **Radar Address Layer** (street addresses)
4. **"Residence"** flag (when no place name found)

### ✅ Accuracy & Quality Control
- GPS coordinate precision analysis (decimal places)
- Estimated accuracy in meters
- Quality rating (Good/Poor based on threshold)
- Cross-service validation with similarity scoring

### ✅ Map Link Generation
- Google Maps coordinate links
- Google Place ID links (when available)
- Radar static map links
- Radar path map links

### ✅ Bulk Processing
- Process entire sheets automatically
- Find next empty column and populate
- Menu-driven operations for ease of use

### ✅ Analysis Tools
- Overnight stays calculation
- GPS accuracy reporting
- Service comparison analysis
- Mismatch detection and flagging

## Usage Examples

### Basic Geocoding:
```
=ENHANCED_GEOCODE("43.0125,-83.6875")
=ENHANCED_GEOCODE("43.0125,-83.6875", "ChIJOwg_06VPwokRYv534QaPC8g")
```

### Place ID Lookup:
```
=GOOGLE_PLACE_LOOKUP("ChIJOwg_06VPwokRYv534QaPC8g")
```

### Map Links:
```
=GENERATE_MAP_LINKS("43.0125,-83.6875", "ChIJOwg_06VPwokRYv534QaPC8g")
```

### GPS Accuracy:
```
=CALCULATE_GPS_ACCURACY("43.0125,-83.6875")
```

## Data Format

The enhanced system returns structured data including:
- **primary_address**: Best available address
- **primary_name**: Best available place name
- **place_type**: Google place type (first in array)
- **data_source**: "Google" or "Radar"
- **match_status**: "Match" or "⚠️ Mismatch"
- **confidence**: Similarity percentage
- **is_residence**: Boolean flag
- **gps_accuracy**: Precision analysis object
- **map_links**: All generated map URLs
- **radar_raw**: Complete Radar API response
- **google_raw**: Complete Google API response

## Cache Management
- 1-hour default cache expiry
- Automatic cache validation
- Manual cache clearing via menu
- Separate caching for each service

## Error Handling
- Comprehensive error catching and reporting
- API rate limit detection
- Invalid coordinate validation
- Missing API key detection
- Graceful degradation when services unavailable

## Michigan-Specific Features
Since you're in Flint, MI:
- MI state abbreviation is allowed by default
- Can easily add other states as needed
- Location-specific testing coordinates included