# **Timeline Processor (Rebuilt)**

## **1\. Project Overview**

This project is a modular, multi-pass pipeline designed to process, clean, enrich, and analyze Google Timeline JSON exports. It is built to be robust, memory-efficient, and handle inconsistencies in the source data gracefully. The core philosophy is to break down the complex task of timeline processing into a series of simple, single-purpose modules that can be individually tested, modified, and improved without breaking the entire workflow.

The entire pipeline is managed by the run\_timeline\_processor.py script, which provides a user-friendly command-line interface with real-time progress updates for each pass.

## **2\. Core Workflow & Logic**

The system operates as a sequential pipeline, where the output of one module becomes the input for the next.

### **Pass 1: Parsing (modules/pass\_1\_parser.py)**

* **Goal:** Convert the raw, potentially unordered Timeline.json into two clean, structured CSV files.  
* **Process:**  
  1. **Extraction (1a):** Streams the source JSON, writing each raw object to an intermediate .jsonl file to minimize memory usage.  
  2. **Sort & Process (1b):** Loads the intermediate file, performs a full chronological sort, and then processes the ordered data.  
* **Key Logic:**  
  * **ID Generation:** Creates sortable YYMMDDX IDs for top-level events and nested YYMMDDX.X IDs for path points.  
  * **Full Data Capture:** Extracts **every** field from the source JSON, storing nested objects as JSON strings to ensure no data is lost.  
  * **Anomaly Flagging:** Identifies and flags "orphan paths" (anomaly\_flag \= True).  
* **Outputs:**  
  * data/processed/raw\_timeline\_events.csv  
  * data/processed/unique\_locations.csv

### **Pass 2: Timestamp Normalization (modules/pass\_2\_timestamp\_normalizer.py)**

* **Goal:** Correct the inaccurate, rounded startTime and endTime of parent timelinePath container events.  
* **Process:**  
  1. Reads the raw\_timeline\_events.csv.  
  2. Identifies path container events.  
  3. For each path, it looks ahead to the next activitySegment in the chronologically sorted file.  
  4. It uses the startTime and endTime from that *activity* to overwrite the path's original, inaccurate timestamps.  
* **Key Logic:**  
  * This pass **does not** alter the timestamps of individual pathPoint records.  
  * If no subsequent activity is found, it uses the first and last point times as a fallback.  
* **Output:**  
  * data/processed/normalized\_timeline\_events.csv

### **Pass 3: Master Cache Building (modules/pass\_3\_cache\_builder.py)**

* **Goal:** Consolidate all historical API responses into two master cache files.  
* **Process:**  
  1. Scans all JSON files in data/raw\_api\_responses/.  
  2. Identifies and extracts data from Radar (geocoding) and Google Places responses.  
* **Outputs:**  
  * data/processed/radar\_geocoding\_cache.json  
  * data/processed/google\_place\_id\_cache.json

## **3\. Final Output Schema (The Goal)**

The ultimate goal of the enrichment and analysis passes is to produce a single, master CSV file with the following structure:

| Field | Description | Source Pass |
| :---- | :---- | :---- |
| Event ID | Unique, sortable ID (YYMMDDX or YYMMDDX.X) | Pass 1 |
| Type | e.g., placeVisit, activitySegment, pathPoint | Pass 1 |
| Subtype | e.g., IN\_PASSENGER\_VEHICLE | Pass 1 |
| start\_day | Day of the week (e.g., "Wednesday") | Pass 6 (Analysis) |
| start\_date | Formatted date (e.g., "10/04/2017") | Pass 6 (Analysis) |
| start\_time | Formatted time (e.g., "03:31:00 PM") | Pass 6 (Analysis) |
| end\_day | Day of the week | Pass 6 (Analysis) |
| end\_date | Formatted date | Pass 6 (Analysis) |
| end\_time | Formatted time | Pass 6 (Analysis) |
| duration | Calculated duration (e.g., "0h 13m") | Pass 6 (Analysis) |
| overnight\_flag | True if start/end dates differ | Pass 6 (Analysis) |
| start\_address | Full address from enrichment | Pass 5 (Merging) |
| start\_place\_name | Place name from enrichment | Pass 5 (Merging) |
| Place Type | Category from enrichment | Pass 5 (Merging) |
| Google Maps Link (lat,lang based) | URL based on coordinates | Pass 5 (Merging) |
| Start Place Flag | Custom flag for analysis | Pass 6 (Analysis) |
| end\_address | Full address from enrichment | Pass 5 (Merging) |
| end\_place\_name | Place name from enrichment | Pass 5 (Merging) |
| End place type | Category from enrichment | Pass 5 (Merging) |
| Google Maps Link (lat,lang based) | URL based on coordinates | Pass 5 (Merging) |
| End Place Flag | Custom flag for analysis | Pass 6 (Analysis) |
| distance | Distance in miles | Pass 6 (Analysis) |
| Expected Distance | Calculated straight-line distance | Pass 6 (Analysis) |
| anomaly\_flag | True for orphan paths, etc. | Pass 1 |
| start\_lat\_lng | Raw start coordinates | Pass 1 |
| Start Google Maps Link (google id) | URL based on Place ID | Pass 5 (Merging) |
| Start Full place types | JSON array of all types | Pass 5 (Merging) |
| end\_lat\_lng | Raw end coordinates | Pass 1 |
| End Google Maps Link (google id) | URL based on Place ID | Pass 5 (Merging) |
| End Full place types | JSON array of all types | Pass 5 (Merging) |
| confidence | Primary confidence score | Pass 1 |
| raw\_visit\_confidence | Raw confidence score | Pass 1 |
| Links to Radar Static Maps | URL to a static map image | Pass 5 (Merging) |
| Source | ID for the API call or cache hit | Pass 5 (Merging) |
| Radar API Time | Timestamp of Radar API call | Pass 5 (Merging) |
| Google API Retrieval Time | Timestamp of Google API call | Pass 5 (Merging) |

