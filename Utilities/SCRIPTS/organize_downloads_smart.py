import os
import shutil
import re
import sys
from pathlib import Path

# Fix unicode printing on Windows
sys.stdout.reconfigure(encoding='utf-8')

# Base Directory
BASE_DIR = r"C:\Users\matts\Downloads"
DRY_RUN = False  # Set to False to actually move files

# Destination Folders
DEST_DIRS = {
    "Court_Legal": os.path.join(BASE_DIR, "_Court_Legal"),
    "Court_Evidence": os.path.join(BASE_DIR, "_Court_Legal", "Evidence_&_Timelines"),
    "Projects_AI": os.path.join(BASE_DIR, "_Projects", "AI_LLM_Agents"),
    "Projects_Forensics": os.path.join(BASE_DIR, "_Projects", "Forensics_Autopsy"),
    "Projects_General": os.path.join(BASE_DIR, "_Projects", "General_Code_&_Repos"),
    "Software": os.path.join(BASE_DIR, "_Software_Installers"),
    "Documents": os.path.join(BASE_DIR, "_Documents_Misc"),
    "Duplicates": os.path.join(BASE_DIR, "_Duplicates_Versions"),
    "Media": os.path.join(BASE_DIR, "_Media_Personal"),
    "Archives": os.path.join(BASE_DIR, "_Archives_Misc")
}

# Regex Patterns for Classification
PATTERNS = {
    "Duplicates": [r".* \(\d+\)\..*", r"^Copy of .*"],
    
    "Court_Legal": [
        r"(?i).*affidavit.*", r"(?i).*objection.*", r"(?i).*custody.*", r"(?i).*salem.*v.*kinzel.*", 
        r"(?i).*court.*", r"(?i).*legal.*", r"(?i).*motion.*", r"(?i).*order.*", r"(?i).*subpoena.*",
        r"(?i).*attorney.*", r"(?i).*divorce.*", r"(?i).*litigation.*", r"(?i).*pleading.*",
        r"(?i).*discovery.*", r"(?i).*FOC.*", r"(?i).*hearing.*", r"(?i).*judicial.*"
    ],
    
    "Court_Evidence": [
        r"(?i).*timeline.*", r"(?i).*evidence.*", r"(?i).*call_history.*", r"(?i).*sms.*", 
        r"(?i).*chat-export.*", r"(?i).*takeout.*", r"(?i).*facebook.*", r"(?i).*whatsapp.*",
        r"(?i).*location.*history.*", r"(?i).*google.*maps.*", r"(?i).*forensic.*report.*"
    ],

    "Projects_AI": [
        r"(?i).*gemini.*", r"(?i).*claude.*", r"(?i).*chatgpt.*", r"(?i).*llm.*", r"(?i).*ai.*agent.*", 
        r"(?i).*skill.*", r"(?i).*prompt.*", r"(?i).*context.*", r"(?i).*ollama.*", r"(?i).*mcp.*",
        r"(?i).*openai.*", r"(?i).*anthropic.*", r"(?i).*neural.*", r"(?i).*transformer.*",
        r"(?i).*langchain.*", r"(?i).*vector.*", r"(?i).*rag.*", r"(?i).*assistant.*"
    ],

    "Projects_Forensics": [
        r"(?i).*autopsy.*", r"(?i).*forensic.*", r"(?i).*volatility.*", r"(?i).*hashdb.*",
        r"(?i).*hex.*editor.*", r"(?i).*sleuth.*", r"(?i).*wireshark.*", r"(?i).*pcap.*",
        r"(?i).*exif.*", r"(?i).*metadata.*", r"(?i).*ufs.*explorer.*"
    ],

    "Projects_General": [
        r"(?i).*-main\.zip", r"(?i).*-master\.zip", r"(?i).*repo.*", r"(?i).*script.*",
        r"(?i).*\.py$", r"(?i).*\.js$", r"(?i).*\.ts$", r"(?i).*\.java$", r"(?i).*\.cpp$", 
        r"(?i).*\.c$", r"(?i).*\.h$", r"(?i).*\.cs$", r"(?i).*\.go$", r"(?i).*\.rb$", r"(?i).*\.php$",
        r"(?i).*source.*code.*"
    ],

    "Software": [
        r"(?i).*\.exe$", r"(?i).*\.msi$", r"(?i).*\.iso$", r"(?i).*\.dmg$", r"(?i).*\.apk$", 
        r"(?i).*installer.*", r"(?i).*setup.*", r"(?i).*portable.*", r"(?i).*driver.*"
    ],
    
    "Media": [
        r"(?i).*\.mp4$", r"(?i).*\.mp3$", r"(?i).*\.mov$", r"(?i).*\.avi$", r"(?i).*\.mkv$", 
        r"(?i).*\.jpg$", r"(?i).*\.png$", r"(?i).*\.jpeg$", r"(?i).*\.gif$", r"(?i).*\.webp$"
    ],

    "Documents": [
        r"(?i).*\.pdf$", r"(?i).*\.docx$", r"(?i).*\.doc$", r"(?i).*\.xlsx$", r"(?i).*\.xls$", 
        r"(?i).*\.pptx$", r"(?i).*\.ppt$", r"(?i).*\.txt$", r"(?i).*\.md$", r"(?i).*\.csv$"
    ]
}

def ensure_dirs():
    for folder in DEST_DIRS.values():
        if not os.path.exists(folder):
            print(f"Creating directory: {folder}")
            if not DRY_RUN:
                os.makedirs(folder, exist_ok=True)

def get_category(filename):
    # 1. Check Duplicates FIRST (Priority)
    for pattern in PATTERNS["Duplicates"]:
        if re.match(pattern, filename):
            return "Duplicates"

    # 2. Check Specific Project/Topic Categories
    if any(re.match(p, filename) for p in PATTERNS["Court_Evidence"]):
        return "Court_Evidence" # Check before general court to catch evidence files
    if any(re.match(p, filename) for p in PATTERNS["Court_Legal"]):
        return "Court_Legal"
    
    if any(re.match(p, filename) for p in PATTERNS["Projects_Forensics"]):
        return "Projects_Forensics"
    if any(re.match(p, filename) for p in PATTERNS["Projects_AI"]):
        return "Projects_AI"
    if any(re.match(p, filename) for p in PATTERNS["Projects_General"]):
        return "Projects_General"

    # 3. Check General File Types
    if any(re.match(p, filename) for p in PATTERNS["Software"]):
        return "Software"
    if any(re.match(p, filename) for p in PATTERNS["Media"]):
        return "Media"
    
    # 4. Fallback for Archives (if not a project zip)
    if filename.lower().endswith((".zip", ".rar", ".7z", ".tar", ".gz")):
         return "Archives"
         
    # 5. Fallback for Documents (if not legal/project doc)
    if any(re.match(p, filename) for p in PATTERNS["Documents"]):
        return "Documents"

    return None

def main():
    print(f"Starting organization of {BASE_DIR}...")
    ensure_dirs()
    
    moved_count = 0
    skipped_count = 0

    for item in os.listdir(BASE_DIR):
        item_path = os.path.join(BASE_DIR, item)
        
        # Skip directories themselves (unless we want to organize them too? 
        # For now, let's focus on files and Zips that act like folders)
        # But wait, many projects are folders. Let's try to organize folders if they match patterns.
        # We will skip the destination folders themselves.
        if item_path in DEST_DIRS.values() or item.startswith("_"):
            continue

        category_key = get_category(item)
        
        if category_key:
            dest_dir = DEST_DIRS[category_key]
            dest_path = os.path.join(dest_dir, item)
            
            try:
                print(f"Moving '{item}' -> {category_key}")
                if not DRY_RUN:
                    shutil.move(item_path, dest_path)
                moved_count += 1
            except Exception as e:
                print(f"Error moving {item}: {e}")
                skipped_count += 1
        else:
            # print(f"Skipping '{item}' (No category match)")
            skipped_count += 1

    print(f"\nDone. Moved {moved_count} items. Skipped {skipped_count}.")

if __name__ == "__main__":
    main()
