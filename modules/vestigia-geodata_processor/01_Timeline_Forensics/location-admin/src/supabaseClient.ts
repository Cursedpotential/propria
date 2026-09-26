import { createClient } from '@supabase/supabase-js'

const supabaseUrl = import.meta.env.VITE_SUPABASE_URL
const supabaseKey = import.meta.env.VITE_SUPABASE_KEY

if (!supabaseUrl || !supabaseKey) {
    console.error("Missing Supabase URL or Key in env vars")
}

export const supabase = createClient(supabaseUrl, supabaseKey)
