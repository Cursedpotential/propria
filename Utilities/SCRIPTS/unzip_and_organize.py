import os
import shutil
import re
import sys
import zipfile
import tarfile
from pathlib import Path

# Fix unicode printing
sys.stdout.reconfigure(encoding='utf-8')

BASE_DIR = r"C:\Users\matts\Downloads"
STAGING_DIR = os.path.join(BASE_DIR, "_Unzipped_Staging")
TRASH_DIR = os.path.join(BASE_DIR, "_Trash_Zips")

# Destination Folders (Reuse from previous, but flattened for logic)
DEST_DIRS = {
    "Court_Legal": os.path.join(BASE_DIR, "_Court_Legal"),
    "Court_Evidence": os.path.join(BASE_DIR, "_Court_Legal", "Evidence_&_Timelines"),
    "Projects_AI": os.path.join(BASE_DIR, "_Projects", "AI_LLM_Agents"),
    "Projects_Forensics": os.path.join(BASE_DIR, "_Projects", "Forensics_Autopsy"),
    "Projects_General": os.path.join(BASE_DIR, "_Projects", "General_Code_&_Repos"),
    "Software": os.path.join(BASE_DIR, "_Software_Installers"),
    "Documents": os.path.join(BASE_DIR, "_Documents_Misc"),
    "Media": os.path.join(BASE_DIR, "_Media_Personal"),
    "Archives": os.path.join(BASE_DIR, "_Archives_Misc") # For things we can't extract or don't know what to do with
}

# Regex Patterns (Reused)
PATTERNS = {
    "Court_Legal": [r"(?i).*affidavit.*", r"(?i).*objection.*", r"(?i).*custody.*", r"(?i).*salem.*v.*kinzel.*", r"(?i).*court.*", r"(?i).*legal.*", r"(?i).*motion.*", r"(?i).*order.*", r"(?i).*subpoena.*", r"(?i).*attorney.*", r"(?i).*divorce.*", r"(?i).*litigation.*", r"(?i).*pleading.*", r"(?i).*discovery.*", r"(?i).*FOC.*", r"(?i).*hearing.*", r"(?i).*judicial.*"],
    "Court_Evidence": [r"(?i).*timeline.*", r"(?i).*evidence.*", r"(?i).*call_history.*", r"(?i).*sms.*", r"(?i).*chat-export.*", r"(?i).*takeout.*", r"(?i).*facebook.*", r"(?i).*whatsapp.*", r"(?i).*location.*history.*", r"(?i).*google.*maps.*", r"(?i).*forensic.*report.*"],
    "Projects_AI": [r"(?i).*gemini.*", r"(?i).*claude.*", r"(?i).*chatgpt.*", r"(?i).*llm.*", r"(?i).*ai.*agent.*", r"(?i).*skill.*", r"(?i).*prompt.*", r"(?i).*context.*", r"(?i).*ollama.*", r"(?i).*mcp.*", r"(?i).*openai.*", r"(?i).*anthropic.*", r"(?i).*neural.*", r"(?i).*transformer.*", r"(?i).*langchain.*", r"(?i).*vector.*", r"(?i).*rag.*", r"(?i).*assistant.*"],
    "Projects_Forensics": [r"(?i).*autopsy.*", r"(?i).*forensic.*", r"(?i).*volatility.*", r"(?i).*hashdb.*", r"(?i).*hex.*editor.*", r"(?i).*sleuth.*", r"(?i).*wireshark.*", r"(?i).*pcap.*", r"(?i).*exif.*", r"(?i).*metadata.*", r"(?i).*ufs.*explorer.*"],
    "Projects_General": [r"(?i).*-main", r"(?i).*-master", r"(?i).*repo.*", r"(?i).*script.*", r"(?i).*code.*", r"(?i).*project.*", r"(?i).*tool.*", r"(?i).*plugin.*", r"(?i).*extension.*"],
    "Software": [r"(?i).*installer.*", r"(?i).*setup.*", r"(?i).*driver.*", r"(?i).*portable.*"],
    "Media": [r"(?i).*photo.*", r"(?i).*video.*", r"(?i).*image.*"]
}

def get_category_by_name(name):
    for key, patterns in PATTERNS.items():
        if any(re.match(p, name) for p in patterns):
            return key
    return "Projects_General" # Default for zips that look like code but don't match AI/Forensics

def safe_extract(zip_path, extract_to):
    try:
        shutil.unpack_archive(zip_path, extract_to)
        return True
    except Exception as e:
        print(f"Failed to extract {zip_path}: {e}")
        return False

def main():
    if not os.path.exists(STAGING_DIR):
        os.makedirs(STAGING_DIR)
    if not os.path.exists(TRASH_DIR):
        os.makedirs(TRASH_DIR)

    # Walk through all folders to find zips (including the ones we just moved)
    archives_processed = 0
    
    for root, dirs, files in os.walk(BASE_DIR):
        # Skip our special folders to prevent recursion loops or processing trash
        if "_Trash_Zips" in root or "_Unzipped_Staging" in root:
            continue
            
        for file in files:
            if file.lower().endswith(('.zip', '.7z', '.tar.gz', '.rar')):
                file_path = os.path.join(root, file)
                file_name_no_ext = Path(file).stem
                
                # Logic: 
                # 1. Determine category based on Zip Name
                category = get_category_by_name(file)
                dest_base = DEST_DIRS.get(category, DEST_DIRS["Archives"])
                
                print(f"Processing: {file} -> Category: {category}")
                
                # 2. Extract to Staging
                staging_target = os.path.join(STAGING_DIR, file_name_no_ext)
                if os.path.exists(staging_target):
                    # Handle duplicates in staging
                    staging_target = f"{staging_target}_{re.sub(r'[^a-zA-Z0-9]', '', str(os.urandom(4)))}"
                
                if safe_extract(file_path, staging_target):
                    # 3. Analyze Structure
                    # If staging_target contains ONLY one folder, use that folder as the content
                    items = os.listdir(staging_target)
                    final_source = staging_target
                    if len(items) == 1 and os.path.isdir(os.path.join(staging_target, items[0])):
                        final_source = os.path.join(staging_target, items[0])
                        target_name = items[0]
                    else:
                        target_name = file_name_no_ext

                    # 4. Move to Final Destination
                    final_dest = os.path.join(dest_base, target_name)
                    
                    # Handle destination collision
                    if os.path.exists(final_dest):
                        final_dest = f"{final_dest}_extracted"
                        if os.path.exists(final_dest):
                             final_dest = f"{final_dest}_{re.sub(r'[^a-zA-Z0-9]', '', str(os.urandom(4)))}"
                    
                    try:
                        shutil.move(final_source, final_dest)
                        print(f"  -> Extracted to: {final_dest}")
                        
                        # 5. Move original Zip to Trash
                        trash_dest = os.path.join(TRASH_DIR, file)
                        if os.path.exists(trash_dest):
                             trash_dest = f"{trash_dest}_{re.sub(r'[^a-zA-Z0-9]', '', str(os.urandom(4)))}"
                        shutil.move(file_path, trash_dest)
                        archives_processed += 1
                        
                    except Exception as e:
                        print(f"  -> Error moving extracted content: {e}")
    
    # Cleanup Staging
    try:
        os.rmdir(STAGING_DIR) # Only if empty
    except:
        pass # Ignore

    print(f"\nProcessing Complete. {archives_processed} archives extracted and moved to _Trash_Zips.")

if __name__ == "__main__":
    main()
