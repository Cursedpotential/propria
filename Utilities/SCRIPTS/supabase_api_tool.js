require('dotenv').config({ override: true });
const { createClient } = require('@supabase/supabase-js');

// Config
const SUPABASE_URL = process.env.SUPABASE_URL;
const SUPABASE_KEY = process.env.SUPABASE_SERVICE_ROLE_KEY || process.env.SUPABASE_ANON_KEY;

if (!SUPABASE_URL || !SUPABASE_KEY) {
  console.error(JSON.stringify({ error: 'Missing SUPABASE_URL or keys in .env' }));
  process.exit(1);
}

const supabase = createClient(SUPABASE_URL, SUPABASE_KEY, {
  auth: {
    persistSession: false // For backend scripts, we often don't need file-system persistence
  }
});

async function main() {
  const [,, action, table, payload] = process.argv;

  if (!action || !table) {
    console.error(JSON.stringify({ error: 'Usage: node supabase_api_tool.js <action> <table> [json_payload/query]' }));
    process.exit(1);
  }

  try {
    let response;
    let data;
    
    // Parse payload if provided
    let params = {};
    if (payload) {
      try {
        params = JSON.parse(payload);
      } catch (e) {
        // If not JSON, treat as string (e.g., for select columns)
        params = payload; 
      }
    }

    switch (action) {
      case 'select':
        // Usage: node tool.js select my_table "*" OR '{"columns": "id, name", "filter": {"status": "active"}}'
        let query = supabase.from(table).select(typeof params === 'string' ? params : (params.columns || '*'));
        
        if (typeof params === 'object' && params.filter) {
          Object.entries(params.filter).forEach(([key, value]) => {
            query = query.eq(key, value);
          });
        }
        if (typeof params === 'object' && params.limit) {
            query = query.limit(params.limit);
        }
        
        response = await query;
        break;

      case 'insert':
        // Usage: node tool.js insert my_table '[{"name": "foo"}]'
        response = await supabase.from(table).insert(params).select();
        break;

      case 'upsert':
        // Usage: node tool.js upsert my_table '[{"id": 1, "name": "bar"}]'
        response = await supabase.from(table).upsert(params).select();
        break;
        
      case 'delete':
         // Usage: node tool.js delete my_table '{"id": 1}'
         let delQuery = supabase.from(table).delete();
         Object.entries(params).forEach(([key, value]) => {
            delQuery = delQuery.eq(key, value);
         });
         response = await delQuery.select();
         break;

      default:
        throw new Error(`Unknown action: ${action}`);
    }

    if (response.error) {
      throw response.error;
    }

    console.log(JSON.stringify({ success: true, data: response.data }, null, 2));

  } catch (error) {
    console.error(JSON.stringify({ success: false, error: error.message || error }));
    process.exit(1);
  }
}

main();
