import os
import subprocess
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import List, Optional

app = FastAPI()

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

LUA_FILTERS_PATH = r"C:\Users\matts\AI Workspace\pandoc-lua-filters"

class CommandRequest(BaseModel):
    command: str
    cwd: Optional[str] = None

@app.get("/api/filters")
def get_filters():
    filters = []
    if os.path.exists(LUA_FILTERS_PATH):
        for root, dirs, files in os.walk(LUA_FILTERS_PATH):
            for file in files:
                if file.endswith(".lua"):
                    full_path = os.path.join(root, file)
                    filters.append({
                        "name": file,
                        "path": full_path,
                        "category": os.path.basename(root) if root != LUA_FILTERS_PATH else "General"
                    })
    return filters

@app.post("/api/run")
def run_command(req: CommandRequest):
    try:
        result = subprocess.run(
            req.command, 
            shell=True, 
            capture_output=True, 
            text=True, 
            cwd=req.cwd
        )
        if result.returncode == 0:
            return {"status": "success", "stdout": result.stdout, "stderr": result.stderr}
        else:
            return {"status": "error", "stdout": result.stdout, "stderr": result.stderr}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/api/browse")
def browse_fs(path: Optional[str] = None):
    start_path = path if path and os.path.exists(path) else os.getcwd()
    if os.path.isfile(start_path):
        start_path = os.path.dirname(start_path)
        
    entries = []
    try:
        with os.scandir(start_path) as it:
            for entry in it:
                entries.append({
                    "name": entry.name,
                    "is_dir": entry.is_dir(),
                    "path": entry.path
                })
    except PermissionError:
        pass
        
    return {
        "current_path": start_path,
        "parent_path": os.path.dirname(start_path),
        "entries": sorted(entries, key=lambda x: (not x['is_dir'], x['name']))
    }

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
