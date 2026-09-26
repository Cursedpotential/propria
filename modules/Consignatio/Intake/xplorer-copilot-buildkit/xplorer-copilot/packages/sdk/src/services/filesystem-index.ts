import { isTauri, transport } from '../transport';

/** Credentials and the actual service URL stay in the native process. */
export const search = async (
  query: string,
  mode: 'keyword' | 'hybrid',
  limit = 20,
): Promise<unknown> => {
  if (!isTauri())
    throw new Error('Filesystem index search requires the Intake desktop connection.');
  return transport('filesystem_index_search', { query, mode, limit });
};
