/**
 * Byline: Claude Code · Sonnet 5 · 2026-09-14
 *
 * Shared tabular results component for the native metadata panel (rows = the
 * selected files) and the native search surface (rows = hits). Owner decision
 * 2026-09-14 22:16 EDT: implement BOTH Glide Data Grid and AG Grid Community
 * behind a user-selectable toggle so they can be compared side by side -- this
 * is that toggle. The choice is remembered per browser (localStorage) as a
 * lightweight per-viewer convenience, not shared state.
 */
import { useMemo, useState } from 'react';
import DataEditor, {
  GridCellKind,
  type GridCell,
  type GridColumn,
  type Item,
} from '@glideapps/glide-data-grid';
import '@glideapps/glide-data-grid/dist/index.css';
import { AgGridReact } from 'ag-grid-react';
import { ModuleRegistry, AllCommunityModule, themeQuartz, type ColDef } from 'ag-grid-community';

ModuleRegistry.registerModules([AllCommunityModule]);

export type CellValue = string | number | boolean | null | undefined;
export type GridRow = Record<string, CellValue>;

export interface ResultsGridColumn {
  key: string;
  title: string;
  width?: number;
}

interface ResultsGridProps {
  columns: ResultsGridColumn[];
  rows: GridRow[];
  height?: number;
  storageKey?: string;
  onRowClick?: (row: GridRow, index: number) => void;
  emptyMessage: string;
}

type Engine = 'glide' | 'ag-grid';

const readStoredEngine = (storageKey: string): Engine => {
  try {
    const stored = window.localStorage.getItem(storageKey);
    return stored === 'ag-grid' ? 'ag-grid' : 'glide';
  } catch {
    return 'glide';
  }
};

const cellToText = (value: CellValue): string => {
  if (value === null || value === undefined) return '';
  return typeof value === 'boolean' ? (value ? 'true' : 'false') : String(value);
};

const GlideView = ({
  columns,
  rows,
  height,
  onRowClick,
}: Pick<ResultsGridProps, 'columns' | 'rows' | 'height' | 'onRowClick'>) => {
  const gridColumns: GridColumn[] = useMemo(
    () =>
      columns.map((column) => ({
        title: column.title,
        id: column.key,
        width: column.width ?? 160,
      })),
    [columns],
  );
  const getCellContent = ([columnIndex, rowIndex]: Item): GridCell => {
    const column = columns[columnIndex];
    const value = cellToText(rows[rowIndex]?.[column.key]);
    return {
      kind: GridCellKind.Text,
      data: value,
      displayData: value,
      allowOverlay: false,
    };
  };
  return (
    <DataEditor
      columns={gridColumns}
      getCellContent={getCellContent}
      rows={rows.length}
      height={height ?? 360}
      onCellClicked={([, rowIndex]) => onRowClick?.(rows[rowIndex], rowIndex)}
      smoothScrollX
      smoothScrollY
    />
  );
};

const AgGridView = ({
  columns,
  rows,
  height,
  onRowClick,
}: Pick<ResultsGridProps, 'columns' | 'rows' | 'height' | 'onRowClick'>) => {
  const columnDefs: ColDef[] = useMemo(
    () =>
      columns.map((column) => ({
        field: column.key,
        headerName: column.title,
        width: column.width,
        valueFormatter: (params) => cellToText(params.value as CellValue),
      })),
    [columns],
  );
  return (
    <div style={{ height: height ?? 360, width: '100%' }}>
      <AgGridReact
        theme={themeQuartz}
        rowData={rows}
        columnDefs={columnDefs}
        onRowClicked={(event) => {
          if (event.data) onRowClick?.(event.data as GridRow, event.rowIndex ?? -1);
        }}
      />
    </div>
  );
};

const ResultsGrid = ({
  columns,
  rows,
  height,
  storageKey = 'intake-results-grid-engine',
  onRowClick,
  emptyMessage,
}: ResultsGridProps) => {
  const [engine, setEngine] = useState<Engine>(() => readStoredEngine(storageKey));

  const setAndPersist = (next: Engine) => {
    setEngine(next);
    try {
      window.localStorage.setItem(storageKey, next);
    } catch {
      // Per-viewer convenience only; a blocked localStorage is not an error.
    }
  };

  if (rows.length === 0) {
    return <p className="text-xp-text-muted text-xs">{emptyMessage}</p>;
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-1 text-xs" role="radiogroup" aria-label="Grid engine">
        {(['glide', 'ag-grid'] as const).map((option) => (
          <button
            key={option}
            type="button"
            role="radio"
            aria-checked={engine === option}
            onClick={() => setAndPersist(option)}
            className={`border-xp-border rounded border px-2 py-1 ${
              engine === option ? 'bg-xp-blue text-white' : 'hover:bg-xp-surface-light'
            }`}
          >
            {option === 'glide' ? 'Glide Data Grid' : 'AG Grid Community'}
          </button>
        ))}
      </div>
      {engine === 'glide' ? (
        <GlideView columns={columns} rows={rows} height={height} onRowClick={onRowClick} />
      ) : (
        <AgGridView columns={columns} rows={rows} height={height} onRowClick={onRowClick} />
      )}
    </div>
  );
};

export default ResultsGrid;
