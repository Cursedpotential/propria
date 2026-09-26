"""
SMS Backup & Restore XML Parser
Handles both SMS and MMS with proper base64 exclusion

For SMS Backup & Restore format from SyncTech:
https://www.synctech.com.au/sms-backup-restore/fields-in-xml-backup-files/
"""
from __future__ import annotations
import xml.etree.ElementTree as ET
import base64
import re
from datetime import datetime
from typing import Dict, Any, Iterator, Optional, List
from pathlib import Path


# SMS type mapping
SMS_TYPE = {
    1: "Received",
    2: "Sent", 
    3: "Draft",
    4: "Outbox",
    5: "Failed",
    6: "Queued"
}

# Call type mapping - CRITICAL for blocking evidence
CALL_TYPE = {
    1: "Incoming",
    2: "Outgoing",
    3: "Missed",
    4: "Voicemail",
    5: "Rejected",      # SHE REJECTED THE CALL
    6: "Refused_List"   # YOU'RE ON HER BLOCK LIST
}


def parse_java_timestamp(ts: str) -> Optional[datetime]:
    """Convert Java milliseconds timestamp to datetime."""
    if not ts:
        return None
    try:
        return datetime.fromtimestamp(int(ts) / 1000)
    except (ValueError, TypeError):
        return None


def decode_base64_text(data: str) -> Optional[str]:
    """Try to decode base64 as UTF-8 text. Returns None if not text."""
    if not data:
        return None
    try:
        decoded = base64.b64decode(data)
        # Try to decode as text
        return decoded.decode('utf-8', errors='strict')
    except Exception:
        return None


def is_base64_block(text: str) -> bool:
    """Check if a string looks like base64 encoded data."""
    if not text or len(text) < 20:
        return False
    # Base64 is alphanumeric + /+ with optional = padding
    # If >80% of chars are base64-valid and it's long, probably encoded
    b64_chars = set('ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=')
    ratio = sum(1 for c in text if c in b64_chars) / len(text)
    return ratio > 0.9 and len(text) > 50


def stream_sms_records(path: Path, source_name: str = "sms") -> Iterator[Dict[str, Any]]:
    """
    Stream SMS records from SMS Backup & Restore XML.
    Uses iterparse to handle massive files without loading into memory.
    
    Yields dicts with:
        - datetime: parsed timestamp
        - sender: contact name or phone number
        - message: body text (NEVER base64)
        - source: source identifier
        - direction: Sent/Received/etc
        - phone: raw phone number
        - status: delivery status
        - read: whether message was read
    """
    context = ET.iterparse(str(path), events=("end",))
    
    for event, elem in context:
        if elem.tag != "sms":
            continue
            
        body = elem.attrib.get("body", "")
        
        # SKIP if body looks like base64 (shouldn't happen for SMS but safety)
        if is_base64_block(body):
            elem.clear()
            continue
        
        ts = elem.attrib.get("date")
        dt = parse_java_timestamp(ts)
        
        msg_type = int(elem.attrib.get("type", 0))
        direction = SMS_TYPE.get(msg_type, "Unknown")
        
        sender = elem.attrib.get("contact_name") or elem.attrib.get("address") or ""
        phone = elem.attrib.get("address", "")
        
        # Status 64 = Failed (potentially blocked)
        status = int(elem.attrib.get("status", -1))
        read = elem.attrib.get("read", "0") == "1"
        
        yield {
            "datetime": dt,
            "sender": sender,
            "message": body.strip(),
            "source": source_name,
            "direction": direction,
            "phone": phone,
            "status": status,
            "read": read,
            "msg_type": msg_type
        }
        
        elem.clear()


def stream_mms_records(path: Path, source_name: str = "mms") -> Iterator[Dict[str, Any]]:
    """
    Stream MMS records from SMS Backup & Restore XML.
    Properly extracts text from parts, ignoring base64 image data.
    
    MMS structure:
    <mms>
        <parts>
            <part ct="text/plain" text="actual message" />
            <part ct="image/jpeg" data="BASE64_GARBAGE_HERE" />
        </parts>
        <addrs>
            <addr address="+1234567890" type="137" />  (137=from, 151=to)
        </addrs>
    </mms>
    """
    context = ET.iterparse(str(path), events=("end",))
    
    for event, elem in context:
        if elem.tag != "mms":
            continue
        
        ts = elem.attrib.get("date")
        dt = parse_java_timestamp(ts)
        
        # Extract text from parts - ONLY text/plain, skip images
        text_parts = []
        parts_elem = elem.find("parts")
        if parts_elem is not None:
            for part in parts_elem.findall("part"):
                content_type = part.attrib.get("ct", "")
                
                # Only process text content
                if "text" in content_type.lower():
                    # Try 'text' attribute first (newer format)
                    text = part.attrib.get("text", "")
                    
                    # If empty, try 'data' but decode it
                    if not text:
                        data = part.attrib.get("data", "")
                        if data and not is_base64_block(data):
                            text = data
                        elif data:
                            # Try to decode base64 as text
                            decoded = decode_base64_text(data)
                            if decoded:
                                text = decoded
                    
                    if text and text.strip():
                        text_parts.append(text.strip())
        
        # If no text parts, skip this MMS
        if not text_parts:
            elem.clear()
            continue
        
        # Extract addresses
        sender = ""
        recipients = []
        addrs_elem = elem.find("addrs")
        if addrs_elem is not None:
            for addr in addrs_elem.findall("addr"):
                addr_type = addr.attrib.get("type", "")
                address = addr.attrib.get("address", "")
                # type 137 = from, 151 = to (BCC), 130 = to
                if addr_type == "137":
                    sender = address
                else:
                    recipients.append(address)
        
        msg_type = int(elem.attrib.get("msg_box", 0))
        # msg_box: 1=received, 2=sent
        direction = "Received" if msg_type == 1 else "Sent" if msg_type == 2 else "Unknown"
        
        yield {
            "datetime": dt,
            "sender": sender,
            "recipients": recipients,
            "message": " ".join(text_parts),
            "source": source_name,
            "direction": direction,
            "msg_type": msg_type,
            "has_attachments": len(list(parts_elem.findall("part"))) > len(text_parts) if parts_elem else False
        }
        
        elem.clear()


def stream_call_log(path: Path, source_name: str = "calls") -> Iterator[Dict[str, Any]]:
    """
    Stream call log records. CRITICAL for proving blocking.
    
    Key evidence:
    - type=5 (Rejected): She actively rejected your call
    - type=6 (Refused_List): You're on her block list
    - duration=0 on outgoing: Call didn't connect (blocked or declined)
    """
    context = ET.iterparse(str(path), events=("end",))
    
    for event, elem in context:
        if elem.tag != "call":
            continue
        
        ts = elem.attrib.get("date")
        dt = parse_java_timestamp(ts)
        
        call_type = int(elem.attrib.get("type", 0))
        duration = int(elem.attrib.get("duration", 0))
        
        # Blocking indicators
        is_blocked_indicator = False
        block_evidence = []
        
        if call_type == 5:
            is_blocked_indicator = True
            block_evidence.append("Call actively rejected")
        if call_type == 6:
            is_blocked_indicator = True
            block_evidence.append("Number on refuse/block list")
        if call_type == 2 and duration == 0:
            is_blocked_indicator = True
            block_evidence.append("Outgoing call with 0 duration - did not connect")
        
        yield {
            "datetime": dt,
            "phone": elem.attrib.get("number", ""),
            "contact_name": elem.attrib.get("contact_name", ""),
            "duration": duration,
            "call_type": call_type,
            "call_type_name": CALL_TYPE.get(call_type, "Unknown"),
            "source": source_name,
            "is_blocked_indicator": is_blocked_indicator,
            "block_evidence": block_evidence
        }
        
        elem.clear()


def search_for_pattern(path: Path, pattern: str, case_insensitive: bool = True) -> Iterator[Dict[str, Any]]:
    """
    Search SMS/MMS XML for a specific pattern.
    Skips base64 content to avoid false positives.
    
    Use this to find specific statements like "comply".
    """
    flags = re.IGNORECASE if case_insensitive else 0
    regex = re.compile(pattern, flags)
    
    # Search SMS
    for record in stream_sms_records(path):
        if regex.search(record["message"]):
            record["match_type"] = "sms"
            record["pattern_matched"] = pattern
            yield record
    
    # Search MMS
    for record in stream_mms_records(path):
        if regex.search(record["message"]):
            record["match_type"] = "mms"
            record["pattern_matched"] = pattern
            yield record


def find_blocking_evidence(path: Path, target_phone: str = None) -> Iterator[Dict[str, Any]]:
    """
    Find all evidence of call blocking/rejection.
    
    If target_phone provided, filters to only that number.
    Normalizes phone numbers for comparison.
    """
    def normalize_phone(phone: str) -> str:
        return re.sub(r'[^\d]', '', phone)[-10:]  # Last 10 digits
    
    target_normalized = normalize_phone(target_phone) if target_phone else None
    
    for record in stream_call_log(path):
        if record["is_blocked_indicator"]:
            if target_normalized:
                if normalize_phone(record["phone"]) == target_normalized:
                    yield record
            else:
                yield record


# ============================================
# MAIN ANALYSIS FUNCTIONS
# ============================================

def analyze_sms_file(path: Path, output_csv: Path = None) -> Dict[str, Any]:
    """
    Full analysis of an SMS Backup & Restore XML file.
    Returns summary statistics and optionally writes to CSV.
    """
    import csv
    
    stats = {
        "total_sms": 0,
        "total_mms": 0,
        "total_calls": 0,
        "blocked_calls": 0,
        "by_contact": {},
        "by_direction": {"Sent": 0, "Received": 0}
    }
    
    all_records = []
    
    # Process SMS
    for rec in stream_sms_records(path):
        stats["total_sms"] += 1
        direction = rec.get("direction", "Unknown")
        if direction in stats["by_direction"]:
            stats["by_direction"][direction] += 1
        
        contact = rec.get("sender", "Unknown")
        stats["by_contact"][contact] = stats["by_contact"].get(contact, 0) + 1
        all_records.append(rec)
    
    # Process MMS
    for rec in stream_mms_records(path):
        stats["total_mms"] += 1
        all_records.append(rec)
    
    # Process calls
    for rec in stream_call_log(path):
        stats["total_calls"] += 1
        if rec["is_blocked_indicator"]:
            stats["blocked_calls"] += 1
    
    # Write CSV if requested
    if output_csv and all_records:
        with open(output_csv, 'w', newline='', encoding='utf-8') as f:
            writer = csv.DictWriter(f, fieldnames=all_records[0].keys())
            writer.writeheader()
            writer.writerows(all_records)
    
    return stats


if __name__ == "__main__":
    import sys
    
    if len(sys.argv) < 2:
        print("Usage: python sms_backup_parser.py <xml_file> [search_pattern]")
        sys.exit(1)
    
    xml_path = Path(sys.argv[1])
    
    if len(sys.argv) > 2:
        # Search mode
        pattern = sys.argv[2]
        print(f"Searching for: {pattern}")
        for match in search_for_pattern(xml_path, pattern):
            print(f"\n{match['datetime']} | {match['direction']} | {match['sender']}")
            print(f"  {match['message'][:200]}...")
    else:
        # Analysis mode
        print(f"Analyzing: {xml_path}")
        stats = analyze_sms_file(xml_path)
        print(f"\nSMS: {stats['total_sms']}")
        print(f"MMS: {stats['total_mms']}")
        print(f"Calls: {stats['total_calls']}")
        print(f"Blocked call indicators: {stats['blocked_calls']}")
