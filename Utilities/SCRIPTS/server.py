# Salem MCP Gateway Server
# Runs on VPS, coordinates MCP tools, contains context/token usage
# Claude Desktop points here instead of running MCPs locally

from fastmcp import FastMCP
from typing import Any
import os
import httpx
import json
from datetime import datetime

# Initialize FastMCP server
mcp = FastMCP("Salem MCP Gateway")

# ===========================================
# CONFIGURATION
# ===========================================
SUPABASE_URL = os.environ.get("SUPABASE_URL")
SUPABASE_KEY = os.environ.get("SUPABASE_KEY")
NEO4J_URI = os.environ.get("NEO4J_URI")
NEO4J_USER = os.environ.get("NEO4J_USER", "neo4j")
NEO4J_PASSWORD = os.environ.get("NEO4J_PASSWORD")
QDRANT_URL = os.environ.get("QDRANT_URL")
QDRANT_API_KEY = os.environ.get("QDRANT_API_KEY")
R2_ENDPOINT = os.environ.get("R2_ENDPOINT")
R2_ACCESS_KEY = os.environ.get("R2_ACCESS_KEY")
R2_SECRET_KEY = os.environ.get("R2_SECRET_KEY")
R2_BUCKET = os.environ.get("R2_BUCKET", "salem-legal-evidence")

# Internal service URLs (within Docker network)
GRAPHITI_URL = "http://graphiti:8000"
MEM0_URL = "http://mem0:8000"
UNSTRUCTURED_URL = "http://unstructured:8000"
STIRLING_URL = "http://stirling-pdf:8080"
LITELLM_URL = "http://litellm:4000"
N8N_WEBHOOK_BASE = "https://n8n.mitechconsult.com/webhook"


# ===========================================
# SUPABASE TOOLS
# ===========================================
@mcp.tool()
async def query_timeline(
    case_id: str = None,
    start_date: str = None,
    end_date: str = None,
    category: str = None,
    significant_only: bool = False,
    limit: int = 50
) -> dict:
    """Query timeline events from the database.
    
    Args:
        case_id: Filter by case UUID
        start_date: Filter events after this date (ISO format)
        end_date: Filter events before this date (ISO format)
        category: Filter by category (e.g., 'childhood_memory', 'current_case_incident')
        significant_only: Only return events marked as significant
        limit: Maximum number of results
    """
    # Build query
    query = "SELECT * FROM timeline_events WHERE 1=1"
    params = []
    
    if case_id:
        query += " AND case_id = $1"
        params.append(case_id)
    if start_date:
        query += f" AND date_parsed >= ${len(params)+1}"
        params.append(start_date)
    if end_date:
        query += f" AND date_parsed <= ${len(params)+1}"
        params.append(end_date)
    if category:
        query += f" AND category = ${len(params)+1}"
        params.append(category)
    if significant_only:
        query += " AND is_significant = true"
    
    query += f" ORDER BY date_parsed DESC LIMIT ${len(params)+1}"
    params.append(limit)
    
    async with httpx.AsyncClient() as client:
        response = await client.post(
            f"{SUPABASE_URL}/rest/v1/rpc/raw_sql",
            headers={
                "apikey": SUPABASE_KEY,
                "Authorization": f"Bearer {SUPABASE_KEY}",
                "Content-Type": "application/json"
            },
            json={"query": query, "params": params}
        )
        return response.json()


@mcp.tool()
async def search_entities(
    name: str = None,
    entity_type: str = None,
    relationship_to_user: str = None
) -> dict:
    """Search for entities (people, places, organizations) in the database.
    
    Args:
        name: Search by name (partial match)
        entity_type: Filter by type ('person', 'place', 'organization')
        relationship_to_user: Filter by relationship (e.g., 'mother', 'employer')
    """
    query_parts = []
    
    if name:
        query_parts.append(f"canonical_name.ilike.*{name}*")
    if entity_type:
        query_parts.append(f"type.eq.{entity_type}")
    if relationship_to_user:
        query_parts.append(f"relationship_to_user.eq.{relationship_to_user}")
    
    query_string = "&".join(query_parts) if query_parts else ""
    
    async with httpx.AsyncClient() as client:
        response = await client.get(
            f"{SUPABASE_URL}/rest/v1/entities?{query_string}",
            headers={
                "apikey": SUPABASE_KEY,
                "Authorization": f"Bearer {SUPABASE_KEY}"
            }
        )
        return response.json()


@mcp.tool()
async def add_timeline_event(
    description: str,
    date_raw: str,
    category: str,
    case_id: str = None,
    location: str = None,
    witnesses: list[str] = None,
    is_significant: bool = False,
    manipulation_pattern: str = None,
    legal_factors: list[str] = None,
    child_present: bool = None,
    raw_quotes: list[str] = None,
    source_app: str = "mcp_gateway"
) -> dict:
    """Add a new timeline event to the database.
    
    Args:
        description: What happened
        date_raw: Date as spoken/written (e.g., "Summer 2005", "October 12, 2024")
        category: Event category
        case_id: Associated case UUID
        location: Where it happened
        witnesses: List of witness names
        is_significant: Mark as significant for the case
        manipulation_pattern: If applicable (DARVO, gaslighting, etc.)
        legal_factors: MCL 722.23 factors (K, J, G, etc.)
        child_present: Was the child present
        raw_quotes: Exact quotes to preserve
        source_app: Which app created this
    """
    event_data = {
        "description": description,
        "date_raw": date_raw,
        "category": category,
        "source_app": source_app,
    }
    
    if case_id:
        event_data["case_id"] = case_id
    if location:
        event_data["location"] = location
    if witnesses:
        event_data["witnesses"] = witnesses
    if is_significant:
        event_data["is_significant"] = is_significant
    if manipulation_pattern:
        event_data["manipulation_pattern"] = manipulation_pattern
    if legal_factors:
        event_data["legal_factors"] = legal_factors
    if child_present is not None:
        event_data["child_present"] = child_present
    if raw_quotes:
        event_data["raw_quotes"] = raw_quotes
    
    async with httpx.AsyncClient() as client:
        response = await client.post(
            f"{SUPABASE_URL}/rest/v1/timeline_events",
            headers={
                "apikey": SUPABASE_KEY,
                "Authorization": f"Bearer {SUPABASE_KEY}",
                "Content-Type": "application/json",
                "Prefer": "return=representation"
            },
            json=event_data
        )
        return response.json()


# ===========================================
# KNOWLEDGE GRAPH TOOLS (via Graphiti)
# ===========================================
@mcp.tool()
async def graph_add_entity(
    name: str,
    entity_type: str,
    properties: dict = None
) -> dict:
    """Add an entity node to the knowledge graph.
    
    Args:
        name: Entity name (will be canonicalized)
        entity_type: 'Person', 'Place', or 'Organization'
        properties: Additional properties for the node
    """
    async with httpx.AsyncClient() as client:
        response = await client.post(
            f"{GRAPHITI_URL}/nodes",
            json={
                "name": name,
                "type": entity_type,
                "properties": properties or {}
            }
        )
        return response.json()


@mcp.tool()
async def graph_add_relationship(
    source_name: str,
    target_name: str,
    relationship_type: str,
    properties: dict = None
) -> dict:
    """Add a relationship between two entities in the knowledge graph.
    
    Args:
        source_name: Source entity name
        target_name: Target entity name
        relationship_type: Type of relationship (e.g., 'PARENT_OF', 'MARRIED_TO', 'THREATENED')
        properties: Additional properties (start_date, end_date, etc.)
    """
    async with httpx.AsyncClient() as client:
        response = await client.post(
            f"{GRAPHITI_URL}/relationships",
            json={
                "source": source_name,
                "target": target_name,
                "type": relationship_type,
                "properties": properties or {}
            }
        )
        return response.json()


@mcp.tool()
async def graph_query(query: str) -> dict:
    """Run a natural language query against the knowledge graph.
    
    Args:
        query: Natural language question about entities and relationships
    """
    async with httpx.AsyncClient() as client:
        response = await client.post(
            f"{GRAPHITI_URL}/query",
            json={"query": query}
        )
        return response.json()


# ===========================================
# MEMORY TOOLS (via Mem0)
# ===========================================
@mcp.tool()
async def memory_add(
    content: str,
    user_id: str = "matt",
    metadata: dict = None
) -> dict:
    """Add a memory to the AI memory system.
    
    Args:
        content: The memory content to store
        user_id: User identifier
        metadata: Additional metadata
    """
    async with httpx.AsyncClient() as client:
        response = await client.post(
            f"{MEM0_URL}/v1/memories",
            json={
                "messages": [{"role": "user", "content": content}],
                "user_id": user_id,
                "metadata": metadata or {}
            }
        )
        return response.json()


@mcp.tool()
async def memory_search(
    query: str,
    user_id: str = "matt",
    limit: int = 10
) -> dict:
    """Search memories using semantic similarity.
    
    Args:
        query: Search query
        user_id: User identifier
        limit: Maximum results
    """
    async with httpx.AsyncClient() as client:
        response = await client.post(
            f"{MEM0_URL}/v1/memories/search",
            json={
                "query": query,
                "user_id": user_id,
                "limit": limit
            }
        )
        return response.json()


# ===========================================
# WORKFLOW TRIGGERS (via N8N)
# ===========================================
@mcp.tool()
async def trigger_workflow(
    workflow_name: str,
    payload: dict = None
) -> dict:
    """Trigger an N8N workflow.
    
    Args:
        workflow_name: Name of workflow (e.g., 'gdrive-sync', 'timeline', 'factor-tag')
        payload: Data to send to the workflow
    """
    async with httpx.AsyncClient() as client:
        response = await client.post(
            f"{N8N_WEBHOOK_BASE}/{workflow_name}",
            json=payload or {}
        )
        return {"status": response.status_code, "response": response.text}


@mcp.tool()
async def sync_google_drive() -> dict:
    """Trigger Google Drive sync workflow to pull new files."""
    return await trigger_workflow("gdrive-sync")


@mcp.tool()
async def run_deduplication() -> dict:
    """Trigger deduplication workflow to find and archive duplicates."""
    return await trigger_workflow("dedup")


@mcp.tool()
async def generate_timeline(
    start_date: str = None,
    end_date: str = None,
    format: str = "markdown"
) -> dict:
    """Generate a timeline report.
    
    Args:
        start_date: Start of date range
        end_date: End of date range
        format: Output format ('markdown', 'json', 'mcr_exhibit')
    """
    return await trigger_workflow("timeline", {
        "start_date": start_date,
        "end_date": end_date,
        "format": format
    })


@mcp.tool()
async def tag_legal_factors(event_ids: list[str] = None) -> dict:
    """Run MCL 722.23 factor tagging on timeline events.
    
    Args:
        event_ids: Specific events to tag, or None for all untagged
    """
    return await trigger_workflow("factor-tag", {"event_ids": event_ids})


# ===========================================
# DOCUMENT PROCESSING TOOLS
# ===========================================
@mcp.tool()
async def parse_document(
    file_path: str,
    extract_tables: bool = True,
    extract_images: bool = False
) -> dict:
    """Parse a document using Unstructured.
    
    Args:
        file_path: Path to file (in /mnt/r2 or accessible path)
        extract_tables: Whether to extract table data
        extract_images: Whether to extract embedded images
    """
    async with httpx.AsyncClient(timeout=300) as client:
        with open(file_path, 'rb') as f:
            response = await client.post(
                f"{UNSTRUCTURED_URL}/general/v0/general",
                files={"files": f},
                data={
                    "strategy": "auto",
                    "extract_tables": str(extract_tables).lower(),
                    "extract_images": str(extract_images).lower()
                }
            )
        return response.json()


@mcp.tool()
async def ocr_image(file_path: str) -> dict:
    """Run OCR on an image file.
    
    Args:
        file_path: Path to image file
    """
    async with httpx.AsyncClient(timeout=120) as client:
        with open(file_path, 'rb') as f:
            response = await client.post(
                f"{UNSTRUCTURED_URL}/general/v0/general",
                files={"files": f},
                data={"strategy": "ocr_only"}
            )
        return response.json()


# ===========================================
# UTILITY TOOLS
# ===========================================
@mcp.tool()
async def get_case_summary(case_id: str = None) -> dict:
    """Get a summary of the current case status.
    
    Args:
        case_id: Case UUID, or None for primary case
    """
    # Get counts from various tables
    async with httpx.AsyncClient() as client:
        # Timeline events count
        events_resp = await client.get(
            f"{SUPABASE_URL}/rest/v1/timeline_events?select=count",
            headers={
                "apikey": SUPABASE_KEY,
                "Authorization": f"Bearer {SUPABASE_KEY}",
                "Prefer": "count=exact"
            }
        )
        
        # Entities count
        entities_resp = await client.get(
            f"{SUPABASE_URL}/rest/v1/entities?select=count",
            headers={
                "apikey": SUPABASE_KEY,
                "Authorization": f"Bearer {SUPABASE_KEY}",
                "Prefer": "count=exact"
            }
        )
        
        # Files count
        files_resp = await client.get(
            f"{SUPABASE_URL}/rest/v1/files?select=count",
            headers={
                "apikey": SUPABASE_KEY,
                "Authorization": f"Bearer {SUPABASE_KEY}",
                "Prefer": "count=exact"
            }
        )
        
        return {
            "timeline_events": events_resp.headers.get("content-range", "0"),
            "entities": entities_resp.headers.get("content-range", "0"),
            "files": files_resp.headers.get("content-range", "0"),
            "generated_at": datetime.utcnow().isoformat()
        }


@mcp.tool()
async def log_command(
    command: str,
    source: str = "mcp_gateway",
    agent: str = None,
    status: str = "complete",
    response: str = None
) -> dict:
    """Log a command execution to the audit trail.
    
    Args:
        command: What was executed
        source: Source system
        agent: Which agent ran it
        status: Execution status
        response: Result or error message
    """
    async with httpx.AsyncClient() as client:
        response = await client.post(
            f"{SUPABASE_URL}/rest/v1/command_log",
            headers={
                "apikey": SUPABASE_KEY,
                "Authorization": f"Bearer {SUPABASE_KEY}",
                "Content-Type": "application/json"
            },
            json={
                "command": command,
                "source": source,
                "agent": agent,
                "status": status,
                "response": response
            }
        )
        return {"logged": True}


# ===========================================
# SERVER ENTRY POINT
# ===========================================
# For running with uvicorn (used in docker-compose)
app = mcp.get_app()

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8080)
