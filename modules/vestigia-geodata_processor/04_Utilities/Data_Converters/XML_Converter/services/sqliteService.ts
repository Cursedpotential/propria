
import { ParsedMessage } from '../types';
import initSqlJs from 'sql.js';

export class SqliteService {
    private db: any = null;
    private columns: string[] = [];
    private tableName: string = 'messages';
    
    // Batch buffer
    private insertBuffer: any[][] = [];
    private batchSize: number = 500;

    constructor() {}

    async init(columns: string[], tableName: string) {
        // Initialize SQL.js using the bundled WASM file
        const SQL = await initSqlJs({
            // Vite/Webpack will bundle this correctly if configured, 
            // or we might need to copy the wasm file to public assets.
            // For simplicity in this setup, we'll try the default locator or a relative path if needed.
            locateFile: file => `https://cdnjs.cloudflare.com/ajax/libs/sql.js/1.8.0/${file}` 
        });
        
        this.db = new SQL.Database();
        this.columns = columns;
        this.tableName = tableName;

        this.createSchema();
    }

    private createSchema() {
        if (!this.db) return;

        // Map specific known columns to types, default to TEXT
        const colDefs = this.columns.map(c => {
            if (c === 'uuid_v7') return `"${c}" TEXT PRIMARY KEY`;
            if (c.includes('date') && c.includes('iso')) return `"${c}" TEXT`; // SQLite stores dates as text
            if (c === 'record_hash') return `"${c}" TEXT`;
            if (c === 'type' || c === 'read') return `"${c}" INTEGER`;
            return `"${c}" TEXT`;
        });

        // Add Full Text Search Virtual Table
        const ftsCols = this.columns.join(', ');

        const sql = `
            CREATE TABLE "${this.tableName}" (${colDefs.join(', ')});
            CREATE INDEX idx_date ON "${this.tableName}" (date_iso);
            CREATE INDEX idx_source ON "${this.tableName}" (source_id);
            
            -- Enable FTS5 for full text search locally
            CREATE VIRTUAL TABLE "${this.tableName}_fts" USING fts5(${ftsCols});
            CREATE TRIGGER "${this.tableName}_ai" AFTER INSERT ON "${this.tableName}" BEGIN
              INSERT INTO "${this.tableName}_fts" (${ftsCols}) VALUES (${this.columns.map(c => `new."${c}"`).join(', ')});
            END;
        `;

        this.db.run(sql);
    }

    insert(msg: ParsedMessage) {
        if (!this.db) return;

        const row = this.columns.map(col => {
            const val = msg[col];
            return val === undefined || val === null ? null : String(val);
        });
        
        this.insertBuffer.push(row);

        if (this.insertBuffer.length >= this.batchSize) {
            this.flush();
        }
    }

    flush() {
        if (!this.db || this.insertBuffer.length === 0) return;

        // Use a transaction for speed
        this.db.run("BEGIN TRANSACTION");
        
        const placeholders = `(${this.columns.map(() => '?').join(',')})`;
        const stmt = this.db.prepare(`INSERT INTO "${this.tableName}" VALUES ${placeholders}`);
        
        for (const row of this.insertBuffer) {
            stmt.run(row);
        }
        stmt.free();
        
        this.db.run("COMMIT");
        this.insertBuffer = [];
    }

    export(): Uint8Array {
        this.flush(); // Ensure remaining items are written
        return this.db.export();
    }

    close() {
        if (this.db) {
            this.db.close();
            this.db = null;
        }
    }
}
