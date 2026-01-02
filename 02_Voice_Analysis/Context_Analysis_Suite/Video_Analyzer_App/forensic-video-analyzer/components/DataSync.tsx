
import React, { useState, useEffect } from 'react';
import { createClient, SupabaseClient } from '@supabase/supabase-js';
import { StagedVideo } from '../types';
import { transformToDbSchema } from '../utils/dataTransformer';
import { useLocalStorage } from '../hooks/useLocalStorage';

interface DataSyncProps {
  stagedVideos: StagedVideo[];
  speakerMap: Record<string, string>;
}

export const DataSync: React.FC<DataSyncProps> = ({ stagedVideos, speakerMap }) => {
  const [supabaseUrl, setSupabaseUrl] = useLocalStorage('supabaseUrl', '');
  const [supabaseAnonKey, setSupabaseAnonKey] = useLocalStorage('supabaseAnonKey', '');
  const [caseId, setCaseId] = useLocalStorage('caseId', '');
  const [status, setStatus] = useState<'idle' | 'syncing' | 'success' | 'error'>('idle');
  const [message, setMessage] = useState('');
  const [supabase, setSupabase] = useState<SupabaseClient | null>(null);

  useEffect(() => {
    if (supabaseUrl && supabaseAnonKey) {
      try {
        setSupabase(createClient(supabaseUrl, supabaseAnonKey));
        setMessage('');
      } catch (e) {
        setMessage('Invalid Supabase credentials.');
        setSupabase(null);
      }
    } else {
      setSupabase(null);
    }
  }, [supabaseUrl, supabaseAnonKey]);

  const handleSync = async () => {
    if (!supabase) {
      setMessage('Supabase client is not initialized. Please check your credentials.');
      setStatus('error');
      return;
    }
    if (!caseId) {
      setMessage('Please provide a Case ID to sync data.');
      setStatus('error');
      return;
    }

    setStatus('syncing');
    setMessage('Starting sync...');
    const verifiedVideos = stagedVideos.filter(v => v.status === 'Verified');

    try {
      // 1. Fetch existing data
      setMessage('Fetching existing case data from Supabase...');
      const { data: existingCase, error: fetchError } = await supabase
        .from('chronicle_cases')
        .select('case_data')
        .eq('id', caseId)
        .single();

      if (fetchError && fetchError.code !== 'PGRST116') { // PGRST116 = 'Not a single row' (i.e., not found)
        throw fetchError;
      }

      const existingCaseData = existingCase ? existingCase.case_data : null;
      setMessage('Transforming and merging new analysis data...');

      // 2. Transform new data and merge
      const newCaseData = transformToDbSchema(verifiedVideos, speakerMap, existingCaseData);

      // 3. Upsert the merged data
      setMessage('Uploading merged data to Supabase...');
      const { error: upsertError } = await supabase
        .from('chronicle_cases')
        .upsert({ id: caseId, case_data: newCaseData, user_id: '00000000-0000-0000-0000-000000000000' }); // user_id placeholder

      if (upsertError) {
        throw upsertError;
      }

      setStatus('success');
      setMessage(`Sync successful! ${newCaseData.events.length} total events now in case ${caseId}.`);
    } catch (e: any) {
      setStatus('error');
      setMessage(`Sync failed: ${e.message}`);
      console.error(e);
    }
  };

  const verifiedCount = stagedVideos.filter(v => v.status === 'Verified').length;

  return (
    <div className="max-w-2xl mx-auto">
      <h2 className="text-2xl font-bold text-slate-100 mb-6">Supabase Synchronization</h2>
      <p className="text-slate-300 mb-6">
        Configure your Supabase backend to persist analysis data. This will sync all <span className="font-bold text-cyan-400">Verified</span> videos from this session with the cloud.
      </p>
      <div className="space-y-4">
        <div>
          <label htmlFor="caseId" className="block text-sm font-medium text-slate-200">Case ID</label>
          <input type="text" id="caseId" value={caseId} onChange={e => setCaseId(e.target.value)} placeholder="Enter a unique ID for this case" className="mt-1 block w-full bg-slate-900 border-slate-600 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm" />
        </div>
        <div>
          <label htmlFor="supabaseUrl" className="block text-sm font-medium text-slate-200">Supabase Project URL</label>
          <input type="text" id="supabaseUrl" value={supabaseUrl} onChange={e => setSupabaseUrl(e.target.value)} placeholder="https://<project-id>.supabase.co" className="mt-1 block w-full bg-slate-900 border-slate-600 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm" />
        </div>
        <div>
          <label htmlFor="supabaseAnonKey" className="block text-sm font-medium text-slate-200">Supabase Anon (Public) Key</label>
          <input type="password" id="supabaseAnonKey" value={supabaseAnonKey} onChange={e => setSupabaseAnonKey(e.target.value)} placeholder="ey..." className="mt-1 block w-full bg-slate-900 border-slate-600 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm" />
        </div>
      </div>
      <div className="mt-8">
        <button onClick={handleSync} disabled={status === 'syncing' || !supabase || verifiedCount === 0} className="w-full bg-purple-600 hover:bg-purple-500 text-white font-bold py-3 px-4 rounded-lg disabled:bg-slate-600 disabled:cursor-not-allowed flex items-center justify-center gap-2">
          {status === 'syncing' ? 'Syncing...' : `Sync ${verifiedCount} Verified Videos`}
        </button>
      </div>
      {message && (
        <div className={`mt-4 p-3 rounded-lg text-sm ${status === 'error' ? 'bg-red-900/50 text-red-300' : status === 'success' ? 'bg-green-900/50 text-green-300' : 'bg-slate-700/50 text-slate-200'}`}>
          {message}
        </div>
      )}
    </div>
  );
};
