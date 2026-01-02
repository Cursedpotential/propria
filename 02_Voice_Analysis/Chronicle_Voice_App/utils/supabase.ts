
import { createClient } from '@supabase/supabase-js';

// Credentials provided by user. 
// NOTE: We use the ANON key (public), not the SERVICE_ROLE key (admin/secret).
const supabaseUrl = process.env.SUPABASE_URL || 'https://oflqpddqaecotsdsxbzp.supabase.co';
const supabaseKey = process.env.SUPABASE_ANON_KEY || 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6Im9mbHFwZGRxYWVjb3RzZHN4YnpwIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NTIxMTk1MzgsImV4cCI6MjA2NzY5NTUzOH0.2eOlLobklUawA2GTga7b7dLjgMPBeFBTL8s8AOgZqsc';

export const supabase = (supabaseUrl && supabaseKey) 
  ? createClient(supabaseUrl, supabaseKey) 
  : null;

/**
 * REQUIRED SQL SETUP (Run this in Supabase SQL Editor):
 * 
 * create table chronicle_cases (
 *   user_id uuid references auth.users not null primary key,
 *   case_data jsonb,
 *   updated_at timestamp with time zone default timezone('utc'::text, now()) not null
 * );
 * 
 * alter table chronicle_cases enable row level security;
 * 
 * create policy "Users can manage their own cases" 
 * on chronicle_cases for all 
 * using (auth.uid() = user_id);
 */
