# Etl-Analysis-flow

Infrastructure for ingesting, tagging, analyzing, and storing message screenshots using private services.

## Setup

1. Copy `.env.example` to `.env`
2. Fill in all values
3. Run `bash scripts/setup.sh`

## Services

| Service     | URL                      |
|-------------|--------------------------|
| n8n         | http://n8n-vps:5678      |
| Filebrowser | http://n8n-vps:8080      |
| Neo4j       | http://n8n-vps:7474      |

