require('dotenv').config({ override: true });
const { Pool } = require('pg');

const pool = new Pool({
  host: process.env.DB_HOST,
  port: parseInt(process.env.DB_PORT || '6543'),
  database: process.env.DB_NAME,
  user: process.env.DB_USER,
  password: process.env.DB_PASSWORD,
  ssl: { rejectUnauthorized: false }
});

async function runQuery() {
  const sql = process.argv[2];
  const paramsJson = process.argv[3];

  if (!sql) {
    console.error(JSON.stringify({ error: 'Usage: node db_query_tool.js "SELECT * FROM table" [params_json]' }));
    process.exit(1);
  }

  let params = [];
  if (paramsJson) {
    try {
      params = JSON.parse(paramsJson);
    } catch (e) {
      console.error(JSON.stringify({ error: 'Invalid params JSON' }));
      process.exit(1);
    }
  }

  try {
    const res = await pool.query(sql, params);
    console.log(JSON.stringify({ 
      success: true, 
      rowCount: res.rowCount,
      rows: res.rows 
    }, null, 2));
  } catch (err) {
    console.error(JSON.stringify({ 
      success: false, 
      error: err.message,
      detail: err.detail
    }, null, 2));
  } finally {
    await pool.end();
  }
}

runQuery();