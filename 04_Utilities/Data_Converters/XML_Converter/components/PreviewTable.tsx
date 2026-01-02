import React from 'react';
import { ParsedMessage } from '../types';

interface Props {
  data: ParsedMessage[];
}

export const PreviewTable: React.FC<Props> = ({ data }) => {
  if (data.length === 0) return null;

  const headers = Object.keys(data[0]);

  return (
    <div className="border border-slate-700 rounded-lg overflow-hidden bg-slate-900 shadow-lg">
      <div className="overflow-x-auto max-h-[500px]">
        <table className="min-w-full divide-y divide-slate-700">
          <thead className="bg-slate-800 sticky top-0 z-10">
            <tr>
              {headers.map((h) => (
                <th
                  key={h}
                  scope="col"
                  className="px-6 py-3 text-left text-xs font-medium text-slate-300 uppercase tracking-wider whitespace-nowrap"
                >
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="bg-slate-900 divide-y divide-slate-800">
            {data.map((row, idx) => (
              <tr key={idx} className="hover:bg-slate-800/50 transition-colors">
                {headers.map((h) => {
                  let val = row[h] || '';
                  const isLong = val.length > 50;
                  const isBase64 = val.length > 100 && !val.includes(' ') && /^[A-Za-z0-9+/=]+$/.test(val);
                  
                  return (
                    <td key={`${idx}-${h}`} className="px-6 py-4 whitespace-nowrap text-sm text-slate-400">
                      <div className="max-w-xs overflow-hidden text-ellipsis" title={val}>
                        {isBase64 ? (
                          <span className="text-xs bg-slate-700 text-indigo-300 px-2 py-1 rounded">Base64 Data</span>
                        ) : (
                          isLong ? val.substring(0, 50) + '...' : val
                        )}
                      </div>
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="bg-slate-800 px-4 py-3 border-t border-slate-700 text-xs text-slate-400">
        Showing first {data.length} records found in stream.
      </div>
    </div>
  );
};