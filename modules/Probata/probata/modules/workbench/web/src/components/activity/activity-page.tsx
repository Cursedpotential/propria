import { Activity, FileClock } from "lucide-react";

import { useFixedCase } from "@/lib/fixed-case-context";
import { useAppNavigate, useBrowserSearchParams } from "@/lib/router-compat";

import { ActivityOperationsLedger } from "./activity-operations-ledger";
import { BatchActivityPanel } from "./batch-activity-panel";

/** Present durable single-source operations and folder batches on one Activity page. */
export function ActivityPage() {
  const { mode } = useFixedCase();
  const searchParams = useBrowserSearchParams();
  const navigate = useAppNavigate();
  const batchId = searchParams.get("batch") ?? "";

  function closeBatch() {
    const next = new URLSearchParams(searchParams);
    next.delete("batch");
    const query = next.toString();
    void navigate.replace(`/activity${query ? `?${query}` : ""}`);
  }

  return (
    <main className="space-y-6 pb-10">
      <header className="border-b px-5 pb-5 pt-6 lg:px-8">
        <div className="flex items-start gap-3">
          <span className="mt-0.5 inline-flex size-9 shrink-0 items-center justify-center border bg-primary/10 text-primary"><Activity className="size-4" aria-hidden="true" /></span>
          <div className="max-w-2xl">
            <p className="text-sm font-medium text-muted-foreground">{mode === "DEV" ? "Dev case" : "Live case"}</p>
            <h1 className="mt-1 text-2xl font-semibold tracking-tight">Activity</h1>
            <p className="mt-1 text-sm leading-6 text-muted-foreground">Follow each source import from its first request to the latest recorded result. Open an import to review its status, make a decision, or start a safe new attempt.</p>
          </div>
        </div>
      </header>

      {batchId ? <BatchActivityPanel batchId={batchId} mode={mode} onClose={closeBatch} /> : (
        <div className="mx-5 flex items-start gap-2 border-l-2 border-primary/40 bg-muted/30 px-3 py-2 text-xs text-muted-foreground lg:mx-8">
          <FileClock className="mt-0.5 size-3.5 shrink-0" aria-hidden="true" />
          <p>Folder imports appear here by their saved batch reference. Their item states come from the durable batch record.</p>
        </div>
      )}

      <div className="px-5 lg:px-8">
        <ActivityOperationsLedger />
      </div>
    </main>
  );
}
