-- Byline: Grok · grok-4.6 · 2026-08-18
-- Local SQLite stand-in for legal_core.app_settings.
-- Live PostgreSQL apply of 0001_legal_os_bootstrap.sql stays Type 1 HOLD.
-- WAL. No CREATE ROLE. Table prefix replaces PG schemas.

PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS legal_core_app_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  theme TEXT NOT NULL DEFAULT 'dark',
  display_timezone TEXT NOT NULL DEFAULT 'America/New_York',
  confidential_mode INTEGER NOT NULL DEFAULT 0,
  case_phase TEXT NOT NULL DEFAULT 'Discovery',
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT OR IGNORE INTO legal_core_app_settings (id) VALUES (1);
