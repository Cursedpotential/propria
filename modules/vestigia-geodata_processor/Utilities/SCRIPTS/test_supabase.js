const { Client } = require('pg');

const client = new Client({
  connectionString: 'postgresql://postgres.oflqpddqaecotsdsxbzp:%40%40Kailah2020%21%23@aws-0-us-east-2.pooler.supabase.com:5432/postgres',
  ssl: { rejectUnauthorized: false }
});

async function test() {
  try {
    await client.connect();
    const res = await client.query('SELECT version()');
    console.log('Connected! PostgreSQL version:', res.rows[0].version);
    
    const tables = await client.query(`
      SELECT table_name 
      FROM information_schema.tables 
      WHERE table_schema = 'public' 
      ORDER BY table_name
    `);
    console.log('\nTables in public schema:');
    tables.rows.forEach(r => console.log(' -', r.table_name));
  } catch (err) {
    console.error('Error:', err.message);
  } finally {
    await client.end();
  }
}

test();
