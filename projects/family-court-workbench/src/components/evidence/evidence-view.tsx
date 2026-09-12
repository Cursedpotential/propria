// Byline: Claude Code · Sonnet 5 · 2026-09-07
import "@glideapps/glide-data-grid/dist/index.css";
import { DataEditor, GridCellKind, type GridCell, type GridColumn, type Item } from "@glideapps/glide-data-grid";
import * as React from "react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { EvidenceResponse } from "@/types/store";

const COLUMNS: GridColumn[] = [
  { title: "Table", id: "table", width: 90 },
  { title: "Snippet", id: "snippet", width: 380 },
  { title: "Occurred", id: "occurred_at", width: 160 },
  { title: "Known", id: "known_at", width: 160 },
  { title: "Score", id: "score", width: 80 },
];

export function EvidenceView({ data }: { data: EvidenceResponse }) {
  const exhibits = React.useMemo(() => ("hits" in data.exhibits ? data.exhibits.hits : []), [data.exhibits]);
  const searchError = "error" in data.exhibits ? data.exhibits.error : null;

  const getCellContent = React.useCallback(
    ([col, row]: Item): GridCell => {
      const hit = exhibits[row];
      const text = (() => {
        switch (col) {
          case 0:
            return hit.table;
          case 1:
            return hit.snippet;
          case 2:
            return hit.occurred_at ?? "—";
          case 3:
            return hit.known_at ?? "—";
          case 4:
            return hit.score.toFixed(2);
          default:
            return "";
        }
      })();
      return { kind: GridCellKind.Text, data: text, displayData: text, allowOverlay: false };
    },
    [exhibits],
  );

  return (
    <div className="flex h-full flex-col gap-3">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            Exhibits <Badge tone="neutral">{exhibits.length}</Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          {searchError ? (
            <p className="text-sm text-critical-text">{searchError}</p>
          ) : (
            <div className="overflow-hidden rounded-[var(--radius-md)] border border-border">
              <DataEditor
                columns={COLUMNS}
                rows={exhibits.length}
                getCellContent={getCellContent}
                smoothScrollX
                smoothScrollY
                rowHeight={32}
                headerHeight={32}
                width="100%"
                height={Math.min(420, 32 + exhibits.length * 32 + 4)}
              />
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Evidence log</CardTitle>
        </CardHeader>
        <CardContent>
          <ul className="space-y-1 text-sm">
            {data.evidence_log.entries.length === 0 && <p className="text-sm text-text-tertiary">No evidence log entries yet.</p>}
            {data.evidence_log.entries.map((e) => (
              <li key={e.id} className="flex items-center justify-between gap-2 border-b border-border py-1 last:border-0">
                <span>
                  <Badge tone="neutral" className="mr-1.5">
                    {e.action}
                  </Badge>
                  {e.notes ?? "—"}
                </span>
                <span className="font-mono text-xs tabular-nums text-text-tertiary">{e.logged_at}</span>
              </li>
            ))}
          </ul>
        </CardContent>
      </Card>
    </div>
  );
}
