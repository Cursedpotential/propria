

📂 ROOT DIRECTORY

File: requirements.txt
(Dependencies including GUI, Database, and UUID support)

Plaintext

```
flet>=0.21.0
supabase>=2.0.0
python-dotenv>=1.0.0
pydantic>=2.0.0
pydantic-settings>=2.0.0
ijson>=3.2.0
httpx>=0.25.0
uuid6>=2024.1.12
psycopg2-binary
pyperclip

```

File: launch_app.py
(Self-checking bootstrapper to install dependencies and run the app)

Python

```
import sys
import subprocess
import importlib.util
import os

REQUIRED_PACKAGES = [
    ("flet", "flet"),
    ("supabase", "supabase"),
    ("dotenv", "python-dotenv"),
    ("pydantic", "pydantic"),
    ("pydantic_settings", "pydantic-settings"),
    ("ijson", "ijson"),
    ("uuid6", "uuid6"),
    ("psycopg2", "psycopg2-binary"),
    ("pyperclip", "pyperclip")
]

def check_prerequisites():
    print("🔍 SYSTEM CHECK: Verifying prerequisites...")
    missing = []
    
    for module_name, pip_name in REQUIRED_PACKAGES:
        if importlib.util.find_spec(module_name) is None:
            missing.append(pip_name)
            
    if missing:
        print(f"⚠️  Missing {len(missing)} packages: {', '.join(missing)}")
        print("⚙️  Auto-installing missing dependencies...")
        try:
            subprocess.check_call([sys.executable, "-m", "pip", "install", *missing])
            print("✅ Dependencies installed successfully.")
        except subprocess.CalledProcessError:
            print("❌ Failed to install dependencies. Please run 'pip install -r requirements.txt' manually.")
            sys.exit(1)
    else:
        print("✅ All libraries found.")

    # Check for .env file
    if not os.path.exists(".env"):
        print("⚠️  No .env file found. Creating empty template...")
        with open(".env", "w") as f:
            f.write("SUPABASE_URL=\nSUPABASE_ANON_KEY=\nSUPABASE_SERVICE_KEY=\nDB_CONNECTION_STRING=\n")
    
    print("🚀 Launching Application...")
    subprocess.run([sys.executable, "main.py"])

if __name__ == "__main__":
    check_prerequisites()

```

File: .env
(Configuration Template - Do not commit real keys)

Ini, TOML

```
SUPABASE_URL=
SUPABASE_ANON_KEY=
SUPABASE_SERVICE_KEY=
DB_CONNECTION_STRING=
ZEP_API_KEY=
ZEP_API_URL=https://api.getzep.com
DEFAULT_IMPORT_PATH=
LOG_LEVEL=INFO

```

File: main.py
(Application Entry Point with Routing and Health Checks)

Python

```
import flet as ft
from ui.theme import apply_theme
from config.settings import settings
from db.client import get_supabase
# Import Pages
from ui.pages.home import HomePage
from ui.pages.import_page import ImportPage
from ui.pages.browse_page import BrowsePage
from ui.pages.export_page import ExportPage
from ui.pages.settings_page import SettingsPage

def main(page: ft.Page):
    page.title = "Conversation Parser"
    apply_theme(page)

    # --- 1. SYSTEM HEALTH CHECK ---
    # Before loading the app, checking if we are ready
    is_configured = False
    try:
        client = get_supabase()
        # Fast check: does the platform table exist?
        client.table("platforms").select("count", count="exact").execute()
        is_configured = True
    except:
        is_configured = False

    # --- 2. NAVIGATION LOGIC ---
    def change_route(e):
        index = e.control.selected_index
        routes = ["/", "/import", "/browse", "/export", "/settings"]
        page.go(routes[index])

    def route_change(route):
        page.views.clear()
        
        # Define Navigation Rail
        nav_rail = ft.NavigationRail(
            selected_index=0,
            label_type=ft.NavigationRailLabelType.ALL,
            min_width=100,
            destinations=[
                ft.NavigationRailDestination(icon=ft.icons.DASHBOARD, label="Home"),
                ft.NavigationRailDestination(icon=ft.icons.FILE_DOWNLOAD, label="Import"),
                ft.NavigationRailDestination(icon=ft.icons.LIBRARY_BOOKS, label="Browse"),
                ft.NavigationRailDestination(icon=ft.icons.OUTPUT, label="Export"),
                ft.NavigationRailDestination(icon=ft.icons.SETTINGS, label="Settings"),
            ],
            on_change=change_route,
        )

        troute = ft.TemplateRoute(page.route)
        
        # Route Handling
        if troute.match("/"):
            nav_rail.selected_index = 0
            # IF NOT CONFIGURED -> Force User to Home with Warning
            if not is_configured:
                 content = ft.Column([
                     ft.Icon(ft.icons.WARNING, size=50, color=ft.colors.ORANGE),
                     ft.Text("System Not Configured", size=30),
                     ft.Text("Please go to Settings > Database Tools to initialize your tables."),
                     ft.ElevatedButton("Go to Settings", on_click=lambda _: page.go("/settings"))
                 ])
            else:
                content = HomePage(page)

        elif troute.match("/import"):
            nav_rail.selected_index = 1
            content = ImportPage(page)
        elif troute.match("/browse"):
            nav_rail.selected_index = 2
            content = BrowsePage(page)
        elif troute.match("/export"):
            nav_rail.selected_index = 3
            content = ExportPage(page)
        elif troute.match("/settings"):
            nav_rail.selected_index = 4
            content = SettingsPage(page)
        else:
            content = ft.Text("Page not found")

        # Build View
        page.views.append(
            ft.View(
                route,
                [
                    ft.Row(
                        [nav_rail, ft.VerticalDivider(width=1), ft.Container(content=content, expand=True, padding=10)],
                        expand=True,
                    )
                ],
                padding=0,
                bgcolor=ft.colors.BLACK
            )
        )
        page.update()

    page.on_route_change = route_change
    
    # If not configured, start at Settings. Otherwise Home.
    if not is_configured:
        page.go("/settings")
    else:
        page.go("/")

if __name__ == "__main__":
    ft.app(target=main)

```

📂 CONFIG DIRECTORY

File: config/__init__.py

Python

```
# Init

```

File: config/settings.py
(Pydantic settings with Zep and Admin support)

Python

```
from pydantic_settings import BaseSettings, SettingsConfigDict

class Settings(BaseSettings):
    # Supabase
    supabase_url: str = ""
    supabase_anon_key: str = ""
    supabase_service_key: str = ""
    
    # Zep Integration
    zep_api_key: str = ""
    zep_api_url: str = "https://api.getzep.com"
    
    # App Config
    default_import_path: str = ""
    log_level: str = "INFO"
    
    # Admin (For Schema Deploy)
    db_connection_string: str = ""
    
    model_config = SettingsConfigDict(
        env_file=".env", 
        env_file_encoding="utf-8",
        extra="ignore"
    )

try:
    settings = Settings()
except:
    # Fallback if env file missing entirely
    settings = Settings() 

```

📂 DB DIRECTORY

File: db/__init__.py

Python

```
# Init

```

File: db/client.py
(Supabase Singleton)

Python

```
from supabase import create_client, Client
from config.settings import settings
import logging

logger = logging.getLogger(__name__)
_client: Client | None = None

def get_supabase() -> Client:
    global _client
    if _client is None:
        try:
            logger.info("Initializing Supabase client...")
            _client = create_client(
                settings.supabase_url, 
                settings.supabase_service_key
            )
        except Exception as e:
            logger.error(f"Failed to initialize Supabase client: {e}")
            raise e
    return _client

```

File: db/schema.py
(The Master Schema with Zep, Views, and UUIDv7)

Python

```
from dataclasses import dataclass, field
from typing import List

# --- SQL GENERATORS ---
def get_uuidv7_function_sql():
    return """
    create or replace function uuid_generate_v7() 
    returns uuid 
    as $$
    begin
      return encode(
        set_bit(
          set_bit(
            overlay(uuid_send(gen_random_uuid()) placing substring(int8send(floor(extract(epoch from clock_timestamp()) * 1000)::bigint) from 3) from 1 for 6),
            52, 1
          ),
          53, 1
        ),
        'hex')::uuid;
    end
    $$ language plpgsql;
    """

@dataclass
class Column:
    name: str
    type: str
    constraints: str = "" 

@dataclass
class Table:
    name: str
    columns: List[Column]
    indexes: List[str] = field(default_factory=list)
    
    def to_sql(self) -> str:
        cols = []
        for c in self.columns:
            line = f"    {c.name} {c.type}"
            if c.constraints:
                line += f" {c.constraints}"
            cols.append(line)
        
        sql = f"CREATE TABLE IF NOT EXISTS {self.name} (\n"
        sql += ",\n".join(cols)
        sql += "\n);"
        
        for idx_col in self.indexes:
            if "WHERE" in idx_col:
                pass # Handled manually in schema manager
            else:
                idx_name = f"idx_{self.name}_{idx_col.replace(',', '_').replace(' ', '')}"
                sql += f"\nCREATE INDEX IF NOT EXISTS {idx_name} ON {self.name}({idx_col});"
        return sql

# --- RESTORED VIEWS ---
VIEWS_SQL = """
CREATE OR REPLACE VIEW v_conversation_summary AS
SELECT 
    c.id,
    c.title,
    p.name as platform,
    c.message_count,
    c.platform_created_at,
    c.platform_updated_at,
    ib.source_path,
    COUNT(CASE WHEN m.zep_synced THEN 1 END) as messages_synced_to_zep,
    c.created_at as imported_at
FROM conversations c
JOIN platforms p ON c.platform_id = p.id
LEFT JOIN import_batches ib ON c.import_batch_id = ib.id
LEFT JOIN messages m ON m.conversation_id = c.id
GROUP BY c.id, p.name, ib.source_path;

CREATE OR REPLACE VIEW v_import_stats AS
SELECT 
    p.name as platform,
    COUNT(DISTINCT ib.id) as import_batches,
    COUNT(DISTINCT c.id) as total_conversations,
    SUM(c.message_count) as total_messages,
    MAX(ib.completed_at) as last_import
FROM platforms p
LEFT JOIN import_batches ib ON ib.platform_id = p.id
LEFT JOIN conversations c ON c.import_batch_id = ib.id
GROUP BY p.name;
"""

# --- THE COMPLETE MODULES ---
SCHEMA_MODULES = {
    "core": [
        Table("platforms", [
            Column("id", "UUID", "PRIMARY KEY DEFAULT uuid_generate_v7()"),
            Column("name", "TEXT", "UNIQUE NOT NULL"),
            Column("display_name", "TEXT", "NOT NULL"),
            Column("created_at", "TIMESTAMPTZ", "DEFAULT NOW()")
        ]),
        Table("import_batches", [
            Column("id", "UUID", "PRIMARY KEY DEFAULT uuid_generate_v7()"),
            Column("platform_id", "UUID", "REFERENCES platforms(id)"),
            Column("source_path", "TEXT", "NOT NULL"),
            Column("file_count", "INTEGER", "DEFAULT 0"),
            Column("conversation_count", "INTEGER", "DEFAULT 0"),
            Column("message_count", "INTEGER", "DEFAULT 0"),
            Column("status", "TEXT", "DEFAULT 'pending'"), 
            Column("error_message", "TEXT"),
            Column("started_at", "TIMESTAMPTZ", "DEFAULT NOW()"),
            Column("completed_at", "TIMESTAMPTZ"),
            Column("metadata", "JSONB", "DEFAULT '{}'")
        ])
    ],
    "messaging": [
        Table("conversations", [
            Column("id", "UUID", "PRIMARY KEY DEFAULT uuid_generate_v7()"),
            Column("platform_id", "UUID", "REFERENCES platforms(id) NOT NULL"),
            Column("import_batch_id", "UUID", "REFERENCES import_batches(id)"),
            Column("external_uuid", "TEXT"),
            Column("title", "TEXT"),
            Column("summary", "TEXT"),
            Column("source_file", "TEXT"),
            Column("message_count", "INTEGER", "DEFAULT 0"),
            Column("platform_created_at", "TIMESTAMPTZ"),
            Column("platform_updated_at", "TIMESTAMPTZ"),
            Column("created_at", "TIMESTAMPTZ", "DEFAULT NOW()"),
            Column("metadata", "JSONB", "DEFAULT '{}'"),
        ], indexes=["platform_id", "import_batch_id", "platform_created_at"]),
        
        Table("messages", [
            Column("id", "UUID", "PRIMARY KEY DEFAULT uuid_generate_v7()"),
            Column("conversation_id", "UUID", "REFERENCES conversations(id) ON DELETE CASCADE NOT NULL"),
            Column("external_uuid", "TEXT"),
            Column("role", "TEXT", "NOT NULL CHECK (role IN ('user', 'assistant'))"),
            Column("content", "TEXT", "NOT NULL"),
            Column("content_hash", "TEXT"),
            Column("sequence_num", "INTEGER"),
            Column("platform_timestamp", "TIMESTAMPTZ"),
            Column("created_at", "TIMESTAMPTZ", "DEFAULT NOW()"),
            Column("metadata", "JSONB", "DEFAULT '{}'"),
            Column("zep_synced", "BOOLEAN", "DEFAULT FALSE"),
            Column("zep_synced_at", "TIMESTAMPTZ"),
            Column("zep_message_id", "TEXT")
        ], indexes=["conversation_id", "role", "content_hash"])
    ],
    "zep_integration": [
        Table("zep_export_queue", [
            Column("id", "UUID", "PRIMARY KEY DEFAULT uuid_generate_v7()"),
            Column("message_id", "UUID", "REFERENCES messages(id) ON DELETE CASCADE"),
            Column("conversation_id", "UUID", "REFERENCES conversations(id) ON DELETE CASCADE"),
            Column("status", "TEXT", "DEFAULT 'pending'"), 
            Column("attempts", "INTEGER", "DEFAULT 0"),
            Column("last_error", "TEXT"),
            Column("created_at", "TIMESTAMPTZ", "DEFAULT NOW()"),
            Column("processed_at", "TIMESTAMPTZ")
        ], indexes=["status"]),

        Table("zep_sync_log", [
            Column("id", "UUID", "PRIMARY KEY DEFAULT uuid_generate_v7()"),
            Column("batch_size", "INTEGER"),
            Column("messages_synced", "INTEGER"),
            Column("errors", "INTEGER", "DEFAULT 0"),
            Column("started_at", "TIMESTAMPTZ", "DEFAULT NOW()"),
            Column("completed_at", "TIMESTAMPTZ"),
            Column("error_details", "JSONB")
        ])
    ],
    "content_intelligence": [
        Table("code_snippets", [
            Column("id", "UUID", "PRIMARY KEY DEFAULT uuid_generate_v7()"),
            Column("message_id", "UUID", "REFERENCES messages(id) ON DELETE CASCADE"),
            Column("language", "TEXT"),
            Column("code_content", "TEXT"),
            Column("created_at", "TIMESTAMPTZ", "DEFAULT NOW()")
        ], indexes=["language"]),
        
        Table("legal_references", [
            Column("id", "UUID", "PRIMARY KEY DEFAULT uuid_generate_v7()"),
            Column("message_id", "UUID", "REFERENCES messages(id) ON DELETE CASCADE"),
            Column("ref_type", "TEXT"),
            Column("citation", "TEXT"),
            Column("context_snippet", "TEXT"),
            Column("created_at", "TIMESTAMPTZ", "DEFAULT NOW()")
        ], indexes=["citation"]),
        
        Table("artifacts", [
            Column("id", "UUID", "PRIMARY KEY DEFAULT uuid_generate_v7()"),
            Column("message_id", "UUID", "REFERENCES messages(id) ON DELETE CASCADE"),
            Column("artifact_type", "TEXT"),
            Column("title", "TEXT"),
            Column("content", "TEXT"),
            Column("metadata", "JSONB", "DEFAULT '{}'")
        ], indexes=["artifact_type"])
    ]
}

```

File: db/queries.py
(DB Operations helpers)

Python

```
from db.client import get_supabase
import logging

logger = logging.getLogger(__name__)

def get_platform_id(name: str) -> str:
    client = get_supabase()
    resp = client.table("platforms").select("id").eq("name", name).execute()
    if resp.data:
        return resp.data[0]['id']
    raise ValueError(f"Platform '{name}' not found. Run SQL setup.")

def create_import_batch(platform_id: str, source_path: str) -> str:
    client = get_supabase()
    data = {
        "platform_id": platform_id,
        "source_path": source_path,
        "status": "processing"
    }
    resp = client.table("import_batches").insert(data).execute()
    return resp.data[0]['id']

def complete_import_batch(batch_id: str, stats: dict):
    client = get_supabase()
    data = {
        "status": "completed",
        "completed_at": "now()",
        "conversation_count": stats.get('success', 0),
        "metadata": stats
    }
    client.table("import_batches").update(data).eq("id", batch_id).execute()

```

📂 PARSERS DIRECTORY

File: parsers/__init__.py

Python

```
# Init

```

File: parsers/base.py

Python

```
from abc import ABC, abstractmethod
from typing import Generator

class BaseParser(ABC):
    @abstractmethod
    def parse(self, file_path: str) -> Generator[dict, None, None]:
        pass

```

File: parsers/models.py
(Dataclasses for type hints)

Python

```
from dataclasses import dataclass, field
from typing import List, Optional
from datetime import datetime

@dataclass
class Message:
    role: str
    content: str
    external_uuid: Optional[str] = None
    platform_timestamp: Optional[str] = None
    content_hash: Optional[str] = None
    sequence_num: Optional[int] = 0

@dataclass
class Conversation:
    external_uuid: str
    title: str
    messages: List[Message]
    platform_created_at: Optional[str] = None
    platform_updated_at: Optional[str] = None

```

File: parsers/claude_parser.py

Python

```
import ijson
from .base import BaseParser

class ClaudeParser(BaseParser):
    def parse(self, file_path: str):
        with open(file_path, 'rb') as f:
            conversations = ijson.items(f, 'item')
            for conv_data in conversations:
                yield self._transform_conversation(conv_data)

    def _transform_conversation(self, data: dict) -> dict:
        created_at = data.get('create_time')
        updated_at = data.get('update_time')
        messages = []
        raw_mappings = data.get('mapping', {})
        
        for node_id, node_data in raw_mappings.items():
            message_obj = node_data.get('message')
            if message_obj and message_obj.get('content'):
                role = message_obj.get('author', {}).get('role')
                if role in ['user', 'assistant']:
                    parts = message_obj.get('content', {}).get('parts', [])
                    text_content = "".join([str(p) for p in parts if isinstance(p, str)])
                    if text_content:
                        messages.append({
                            "external_uuid": message_obj.get('id'),
                            "role": role,
                            "content": text_content,
                            "platform_timestamp": message_obj.get('create_time')
                        })
        
        messages.sort(key=lambda x: x.get('platform_timestamp') or 0)
        return {
            "external_uuid": data.get('uuid'),
            "title": data.get('name', 'Untitled'),
            "platform_created_at": created_at,
            "platform_updated_at": updated_at,
            "messages": messages
        }

```

File: parsers/gemini_parser.py

Python

```
from .base import BaseParser

class GeminiParser(BaseParser):
    def parse(self, file_path: str):
        raise NotImplementedError("Gemini CSV parsing pending implementation.")

```

File: parsers/content_extractor.py
(The Logic to extract Code and Legal Citations)

Python

```
import re
from dataclasses import dataclass, field
from typing import List, Dict

@dataclass
class ExtractedContent:
    clean_text: str
    code_blocks: List[Dict[str, str]] = field(default_factory=list)
    legal_refs: List[Dict[str, str]] = field(default_factory=list)
    artifacts: List[Dict[str, str]] = field(default_factory=list)

class ContentExtractor:
    CODE_BLOCK_PATTERN = re.compile(r'```(\w+)?\n(.*?)```', re.DOTALL)
    LEGAL_PATTERNS = [
        (r'MCL\s+\d+(\.\d+)?', 'statute'),
        (r'MCR\s+\d+(\.\d+)?', 'court_rule'),
        (r'Best Interest Factor\s+[a-zA-Z]', 'legal_factor'),
        (r'(?i)motion to \w+', 'motion_type')
    ]

    @staticmethod
    def process(raw_text: str) -> ExtractedContent:
        if not raw_text:
            return ExtractedContent(clean_text="")

        code_blocks = []
        artifacts = []
        legal_refs = []

        # Extract Code
        for match in ContentExtractor.CODE_BLOCK_PATTERN.finditer(raw_text):
            lang = match.group(1) or "text"
            content = match.group(2)
            code_blocks.append({
                "language": lang.strip(),
                "code_content": content,
                "description": "Extracted code block"
            })

        # Extract Legal
        for pattern, ref_type in ContentExtractor.LEGAL_PATTERNS:
            for match in re.finditer(pattern, raw_text):
                start = max(0, match.start() - 50)
                end = min(len(raw_text), match.end() + 50)
                context = raw_text[start:end].replace('\n', ' ').strip()
                legal_refs.append({
                    "ref_type": ref_type,
                    "citation": match.group(0),
                    "context_snippet": context
                })

        # Extract Artifacts (Simple Table Check)
        if '|' in raw_text and '---' in raw_text:
            artifacts.append({
                "artifact_type": "table",
                "title": "Embedded Markdown Table",
                "content": "Table detected in message content" 
            })

        return ExtractedContent(
            clean_text=raw_text,
            code_blocks=code_blocks,
            legal_refs=legal_refs,
            artifacts=artifacts
        )

```

📂 SERVICES DIRECTORY

File: services/__init__.py

Python

```
# Init

```

File: services/import_service.py
(The core engine: Hashing + UUIDv7 + Extraction + Insert)

Python

```
import asyncio
import traceback
import uuid6
from parsers.claude_parser import ClaudeParser
from parsers.content_extractor import ContentExtractor
from db.client import get_supabase
from db.queries import (
    get_platform_id, 
    create_import_batch, 
    complete_import_batch
)
from utils.hashing import hash_content

class ImportStats:
    def __init__(self):
        self.total_read = 0
        self.success = 0
        self.duplicates = 0
        self.errors = 0

async def process_import_batch(
    file_path: str, 
    platform_name: str, 
    batch_size: int, 
    delay_seconds: float, 
    log_callback, 
    progress_callback,
    stats_callback
):
    stats = ImportStats()
    log_callback(f"🚀 INITIALIZING IMPORT: {platform_name.upper()}", "blue")
    
    try:
        log_callback("Checking database connection...", "grey")
        platform_id = get_platform_id(platform_name)
        batch_id = create_import_batch(platform_id, file_path)
        log_callback(f"Batch ID created: {batch_id}", "green")

        if platform_name == 'claude':
            parser = ClaudeParser()
            generator = parser.parse(file_path) 
        else:
            raise NotImplementedError("Gemini parser awaiting integration.")

        conv_batch = []
        msg_batch = []
        
        for conv_data in generator:
            stats.total_read += 1
            conv_data['platform_id'] = platform_id
            conv_data['import_batch_id'] = batch_id
            
            msgs = conv_data.pop('messages', [])
            conv_batch.append(conv_data)
            
            for m in msgs:
                m['temp_conv_uuid'] = conv_data['external_uuid']
            msg_batch.extend(msgs)

            if len(conv_batch) >= batch_size:
                await _process_db_batch(conv_batch, msg_batch, platform_id, stats, log_callback)
                conv_batch = []
                msg_batch = []
                
                progress_callback(stats.total_read)
                stats_callback(stats.total_read, stats.success, stats.errors)
                
                if delay_seconds > 0:
                    await asyncio.sleep(delay_seconds)

        if conv_batch:
            await _process_db_batch(conv_batch, msg_batch, platform_id, stats, log_callback)
        
        complete_import_batch(batch_id, {
            "total": stats.total_read,
            "success": stats.success,
            "errors": stats.errors
        })
        
        log_callback("✅ IMPORT COMPLETED", "green")

    except Exception as e:
        log_callback(f"🛑 CRITICAL FAILURE: {str(e)}", "red")
        traceback.print_exc()

async def _process_db_batch(convs, msgs, platform_id, stats, log_callback):
    try:
        client = get_supabase()
        
        # 1. Upsert Conversations
        res = client.table("conversations").upsert(
            convs, on_conflict="platform_id, external_uuid"
        ).execute()
        
        inserted_convs = res.data
        id_map = {row['external_uuid']: row['id'] for row in inserted_convs}
        
        final_msgs = []
        batch_code = []
        batch_legal = []
        batch_artifacts = []

        for m in msgs:
            ext_id = m.pop('temp_conv_uuid', None)
            if ext_id in id_map:
                # UUIDv7 Generation
                msg_uuid = str(uuid6.uuid7())
                m['id'] = msg_uuid 
                m['conversation_id'] = id_map[ext_id]
                
                # Content Hash
                m['content_hash'] = hash_content(m.get('content', ''))
                m['zep_synced'] = False
                
                # Extraction
                extracted = ContentExtractor.process(m.get('content', ''))
                
                for cb in extracted.code_blocks:
                    cb['message_id'] = msg_uuid
                    batch_code.append(cb)
                    
                for lr in extracted.legal_refs:
                    lr['message_id'] = msg_uuid
                    batch_legal.append(lr)
                    
                for art in extracted.artifacts:
                    art['message_id'] = msg_uuid
                    batch_artifacts.append(art)
                
                final_msgs.append(m)

        # Bulk Insert
        if final_msgs:
            client.table("messages").upsert(final_msgs).execute()
        
        if batch_code:
            client.table("code_snippets").insert(batch_code).execute()
            
        if batch_legal:
            client.table("legal_references").insert(batch_legal).execute()
            
        if batch_artifacts:
            client.table("artifacts").insert(batch_artifacts).execute()
            
        stats.success += len(inserted_convs)
        
    except Exception as e:
        stats.errors += len(convs)
        log_callback(f"⚠️ Batch Failed: {str(e)}", "red")

```

📂 UI DIRECTORY

File: ui/__init__.py

Python

```
# Init

```

File: ui/theme.py

Python

```
import flet as ft

def apply_theme(page: ft.Page):
    page.theme_mode = ft.ThemeMode.DARK
    page.theme = ft.Theme(
        color_scheme=ft.ColorScheme(
            primary=ft.colors.BLUE_400,
            primary_container=ft.colors.BLUE_900,
            secondary=ft.colors.TEAL_400,
            surface=ft.colors.GREY_900,
            background=ft.colors.BLACK,
            error=ft.colors.RED_400,
        ),
        visual_density=ft.VisualDensity.COMFORTABLE,
    )
    page.padding = 0
    page.window_width = 1200
    page.window_height = 800

```

File: ui/pages/home.py

Python

```
import flet as ft
def HomePage(page: ft.Page):
    return ft.Container(
        content=ft.Column([
            ft.Text("Dashboard", size=30, weight="bold"),
            ft.Divider(),
            ft.Text("Welcome to the Conversation Parser."),
            ft.Text("Select 'Import' on the left to get started."),
        ]),
        padding=20
    )

```

File: ui/pages/browse_page.py

Python

```
import flet as ft
def BrowsePage(page: ft.Page):
    return ft.Container(content=ft.Text("Browse Interface (Coming Soon)", size=20), alignment=ft.alignment.center)

```

File: ui/pages/export_page.py

Python

```
import flet as ft
def ExportPage(page: ft.Page):
    return ft.Container(content=ft.Text("Export Tools (Coming Soon)", size=20), alignment=ft.alignment.center)

```

File: ui/pages/settings_page.py
(Includes Direct DB Deployment Logic)

Python

```
import flet as ft
import psycopg2
import os
from config.settings import settings
from utils.env_manager import update_env_file
from utils.schema_manager import generate_full_schema_script

def SettingsPage(page: ft.Page):
    url_field = ft.TextField(label="Supabase URL", value=settings.supabase_url)
    anon_key_field = ft.TextField(label="Anon Key", value=settings.supabase_anon_key, password=True, can_reveal_password=True)
    service_key_field = ft.TextField(label="Service Key (Admin)", value=settings.supabase_service_key, password=True, can_reveal_password=True)
    
    db_string_field = ft.TextField(
        label="DB Connection String (URI)", 
        value=os.getenv("DB_CONNECTION_STRING", ""),
        password=True, 
        can_reveal_password=True,
        hint_text="postgresql://postgres.[ref]:[password]@aws-0-us-east-1.pooler.supabase.com:6543/postgres"
    )
    
    status_text = ft.Text("Ready", size=14)

    def save_settings(e):
        update_env_file("SUPABASE_URL", url_field.value)
        update_env_file("SUPABASE_ANON_KEY", anon_key_field.value)
        update_env_file("SUPABASE_SERVICE_KEY", service_key_field.value)
        update_env_file("DB_CONNECTION_STRING", db_string_field.value)
        settings.supabase_url = url_field.value
        settings.supabase_anon_key = anon_key_field.value
        settings.supabase_service_key = service_key_field.value
        page.snack_bar = ft.SnackBar(ft.Text("Settings Saved!"))
        page.snack_bar.open = True
        page.update()

    def deploy_schema_directly(e):
        if not db_string_field.value:
            status_text.value = "Error: Connection String required for Direct Deploy."
            status_text.color = ft.colors.RED
            page.update()
            return

        status_text.value = "Generating Schema & Connecting to DB..."
        status_text.color = ft.colors.BLUE
        page.update()
        
        sql_script = generate_full_schema_script()
        if not sql_script:
            status_text.value = "Schema Generation Failed (Check Logs)"
            return

        try:
            conn = psycopg2.connect(db_string_field.value)
            cur = conn.cursor()
            cur.execute('CREATE EXTENSION IF NOT EXISTS "uuid-ossp";')
            cur.execute(sql_script)
            conn.commit()
            cur.close()
            conn.close()
            status_text.value = "✅ SUCCESS: Schema Deployed Successfully!"
            status_text.color = ft.colors.GREEN
        except Exception as ex:
            status_text.value = f"Deployment Failed: {str(ex)}"
            status_text.color = ft.colors.RED
        page.update()

    return ft.Column(
        controls=[
            ft.Text("Configuration & Deployment", size=24, weight="bold"),
            ft.Divider(),
            url_field,
            anon_key_field,
            service_key_field,
            ft.Container(height=20),
            ft.Text("Database Admin (For Schema Deployment)", weight="bold"),
            db_string_field,
            ft.Row([
                ft.ElevatedButton("Save Credentials", icon=ft.icons.SAVE, on_click=save_settings),
                ft.ElevatedButton("DEPLOY SCHEMA NOW", icon=ft.icons.CLOUD_UPLOAD, bgcolor=ft.colors.RED_900, color=ft.colors.WHITE, on_click=deploy_schema_directly),
            ]),
            status_text
        ],
        scroll=ft.ScrollMode.AUTO
    )

```

File: ui/pages/import_page.py
(The Master Console with Live Health Checks)

Python

```
import flet as ft
from services.import_service import process_import_batch
from db.client import get_supabase

def ImportPage(page: ft.Page):
    # Status Indicators
    icon_conn = ft.Icon(name=ft.icons.CIRCLE, color=ft.colors.GREY, size=14)
    text_conn = ft.Text("Checking API...", size=12)
    icon_schema = ft.Icon(name=ft.icons.CIRCLE, color=ft.colors.GREY, size=14)
    text_schema = ft.Text("Checking Tables...", size=12)
    
    status_card = ft.Container(
        content=ft.Row([
            ft.Row([icon_conn, text_conn]),
            ft.VerticalDivider(width=10),
            ft.Row([icon_schema, text_schema]),
        ]),
        padding=10, bgcolor=ft.colors.GREY_900, border_radius=5
    )

    def check_system_health():
        try:
            client = get_supabase()
            client.table("platforms").select("count", count="exact").execute()
            icon_conn.name = ft.icons.CHECK_CIRCLE
            icon_conn.color = ft.colors.GREEN
            text_conn.value = "API Connected"
            
            client.table("conversations").select("count", count="exact").execute()
            icon_schema.name = ft.icons.CHECK_CIRCLE
            icon_schema.color = ft.colors.GREEN
            text_schema.value = "Schema Valid"
            btn_start.disabled = False
        except Exception as e:
            icon_conn.name = ft.icons.ERROR
            icon_conn.color = ft.colors.RED
            text_conn.value = "Connection Failed"
            icon_schema.name = ft.icons.WARNING
            icon_schema.color = ft.colors.ORANGE
            text_schema.value = "Schema Missing"
            btn_start.disabled = True
        page.update()

    # Report Dialog
    report_dialog = ft.AlertDialog(
        title=ft.Text("Import Complete"),
        content=ft.Column([
            ft.Text("Summary Report", weight="bold"),
            ft.Divider(),
            ft.Row([ft.Text("Total Processed:"), ft.Text(value="0", ref=ft.Ref(), weight="bold")]),
            ft.Row([ft.Text("Successfully Saved:"), ft.Text(value="0", ref=ft.Ref(), color=ft.colors.GREEN)]),
            ft.Row([ft.Text("Errors:"), ft.Text(value="0", ref=ft.Ref(), color=ft.colors.RED)]),
        ], height=150),
        actions=[ft.TextButton("Close", on_click=lambda e: setattr(report_dialog, 'open', False) or page.update())],
    )

    def show_report(total, success, errors):
        report_dialog.content.controls[2].controls[1].value = str(total)
        report_dialog.content.controls[3].controls[1].value = str(success)
        report_dialog.content.controls[4].controls[1].value = str(errors)
        page.dialog = report_dialog
        report_dialog.open = True
        page.update()

    # Main UI
    file_path = ft.TextField(label="Selected File", read_only=True, expand=True)
    progress_bar = ft.ProgressBar(value=0, visible=False)
    txt_total = ft.Text("-", size=20, weight="bold")
    txt_success = ft.Text("-", size=20, color=ft.colors.GREEN)
    txt_errors = ft.Text("-", size=20, color=ft.colors.RED)
    log_column = ft.Column(scroll=ft.ScrollMode.ALWAYS, auto_scroll=True)
    
    def log(msg, color="white"):
        log_column.controls.append(ft.Text(f"> {msg}", color=color, font_family="Consolas", size=12))
        page.update()

    def update_stats(total, success, errors):
        txt_total.value = str(total)
        txt_success.value = str(success)
        txt_errors.value = str(errors)
        page.update()

    async def run_import(e):
        if not file_path.value:
            return
        btn_start.disabled = True
        progress_bar.visible = True
        progress_bar.value = None
        log_column.controls.clear()
        
        await process_import_batch(
            file_path=file_path.value,
            platform_name="claude",
            batch_size=50,
            delay_seconds=0.1,
            log_callback=log,
            progress_callback=lambda x: None,
            stats_callback=update_stats
        )
        progress_bar.value = 1
        show_report(int(txt_total.value), int(txt_success.value), int(txt_errors.value))
        btn_start.disabled = False
        page.update()

    file_picker = ft.FilePicker(on_result=lambda e: [setattr(file_path, 'value', e.files[0].path), page.update()] if e.files else None)
    page.overlay.append(file_picker)

    btn_start = ft.ElevatedButton("START IMPORT", icon=ft.icons.PLAY_ARROW, on_click=run_import, disabled=True)
    check_system_health()

    return ft.Column(
        controls=[
            ft.Row([ft.Text("Import Console", size=24, weight="bold"), ft.Container(expand=True), status_card]),
            ft.Divider(),
            ft.Row([file_path, ft.IconButton(ft.icons.FOLDER_OPEN, on_click=lambda _: file_picker.pick_files())]),
            btn_start,
            progress_bar,
            ft.Row([ft.Column([ft.Text("READ"), txt_total]), ft.Column([ft.Text("SAVED"), txt_success]), ft.Column([ft.Text("ERRORS"), txt_errors])], alignment=ft.MainAxisAlignment.SPACE_EVENLY),
            ft.Container(content=log_column, height=300, bgcolor=ft.colors.BLACK, border=ft.border.all(1, ft.colors.GREY_800), padding=10)
        ], scroll=ft.ScrollMode.AUTO, expand=True
    )

```

📂 UTILS DIRECTORY

File: utils/__init__.py

Python

```
# Init

```

File: utils/env_manager.py

Python

```
import os

def update_env_file(key: str, value: str, env_path: str = ".env"):
    lines = []
    if os.path.exists(env_path):
        with open(env_path, "r", encoding="utf-8") as f:
            lines = f.readlines()

    key_found = False
    new_lines = []
    for line in lines:
        if line.strip().startswith(f"{key}="):
            new_lines.append(f"{key}={value}\n")
            key_found = True
        else:
            new_lines.append(line)
            
    if not key_found:
        if new_lines and not new_lines[-1].endswith('\n'):
            new_lines.append('\n')
        new_lines.append(f"{key}={value}\n")
        
    with open(env_path, "w", encoding="utf-8") as f:
        f.writelines(new_lines)
    os.environ[key] = value

```

File: utils/hashing.py

Python

```
import hashlib

def hash_content(content: str) -> str:
    if not content:
        return ""
    return hashlib.sha256(content.encode('utf-8')).hexdigest()

```

File: utils/schema_manager.py
(Generates the full SQL with Zep tables, Views, and Partial Indexes)

Python

```
import re
from db.schema import SCHEMA_MODULES, get_uuidv7_function_sql, VIEWS_SQL

class SchemaValidator:
    VALID_NAME_REGEX = re.compile(r'^[a-z_][a-z0-9_]*$')
    @staticmethod
    def validate_table(table):
        errors = []
        if not SchemaValidator.VALID_NAME_REGEX.match(table.name):
            errors.append(f"Invalid table name: {table.name}")
        for col in table.columns:
            if not SchemaValidator.VALID_NAME_REGEX.match(col.name):
                errors.append(f"Table '{table.name}': Invalid column name '{col.name}'")
        return errors

def generate_full_schema_script():
    sql_output = []
    sql_output.append("-- EXTENSIONS")
    sql_output.append('CREATE EXTENSION IF NOT EXISTS "uuid-ossp";')
    sql_output.append(get_uuidv7_function_sql())
    
    all_errors = []
    for module, tables in SCHEMA_MODULES.items():
        sql_output.append(f"\n-- MODULE: {module.upper()}")
        for table in tables:
            errs = SchemaValidator.validate_table(table)
            if errs:
                all_errors.extend(errs)
                continue
            sql_output.append(table.to_sql())
            # Partial Indexes Handled Manually here
            if table.name == "messages":
                 sql_output.append("CREATE INDEX IF NOT EXISTS idx_messages_zep_synced ON messages(zep_synced) WHERE zep_synced = FALSE;")
            if table.name == "zep_export_queue":
                 sql_output.append("CREATE INDEX IF NOT EXISTS idx_zep_queue_status ON zep_export_queue(status) WHERE status = 'pending';")

    sql_output.append("\n-- VIEWS")
    sql_output.append(VIEWS_SQL)
    sql_output.append("\n-- SEED DATA")
    sql_output.append("INSERT INTO platforms (name, display_name) VALUES ('claude', 'Claude (Anthropic)'), ('gemini', 'Gemini (Google)') ON CONFLICT (name) DO NOTHING;")

    if all_errors:
        return f"-- ERRORS FOUND: {all_errors}"
    return "\n".join(sql_output)

if __name__ == "__main__":
    print(generate_full_schema_script())

```