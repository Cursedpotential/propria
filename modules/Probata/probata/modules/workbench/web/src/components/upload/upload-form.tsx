// Byline: Claude Code · Sonnet (agent) · 2026-07-19
// Byline amendment: Codex · GPT-6.1-Sol · 2026-10-05 (canonical acquisition upload journey).
"use client";

import { useCallback, useState } from "react";
import { toast } from "sonner";
import type { FileRejection } from "react-dropzone";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Dropzone } from "./dropzone";
import { UploadProgress, type UploadItem } from "./upload-progress";
import { uploadProfferSource, ApiError } from "@/lib/api-client";
import { useOperatingMode } from "@/lib/fixed-case-context";
import { humanizeBytes } from "@/lib/utils";
import { useRefresh } from "@/lib/refresh-context";
import type { ProfferUploadResponse } from "@/lib/shared/types";

/** Upload selected originals with progress and their actual acquisition receipts.
 * Inputs: local files and current policy context. Output: queue and receipt UI.
 * Effects: canonical upload POSTs and refresh after success; no staged identities.
 * Pick for standalone uploads; New Run additionally authors context and starts intake.
 */
export function UploadForm() {
  const mode = useOperatingMode();
  const [items, setItems] = useState<UploadItem[]>([]);
  const [uploading, setUploading] = useState(false);
  const [results, setResults] = useState<(ProfferUploadResponse & { name: string; mime: string })[]>([]);
  const { triggerRefresh } = useRefresh();

  const handleFilesRejected = useCallback((rejections: FileRejection[]) => {
    for (const rejection of rejections) {
      const name = rejection.file.name;
      const errors = rejection.errors.map((e) => {
        if (e.code === "file-too-large") {
          return `exceeds 100MB limit (${humanizeBytes(rejection.file.size)})`;
        }
        return e.message;
      });
      toast.error(`${name}: ${errors.join(", ")}`);
    }
  }, []);

  const handleFilesSelected = useCallback(
    (files: File[]) => {
      const newItems: UploadItem[] = files.map((file) => ({
        id: `${file.name}-${Date.now()}-${Math.random()}`,
        file,
        progress: 0,
        status: "uploading" as const,
      }));
      setItems((prev) => [...prev, ...newItems]);
      setUploading(true);

      const uploadQueue = async () => {
        let anySuccess = false;
        for (const item of newItems) {
          try {
            const result = await uploadProfferSource(item.file, mode, (percent) => {
              setItems((prev) =>
                prev.map((i) => (i.id === item.id ? { ...i, progress: percent } : i)),
              );
            });

            setItems((prev) =>
              prev.map((i) =>
                i.id === item.id ? { ...i, status: "complete", progress: 100 } : i,
              ),
            );
            setResults((prev) => [...prev, { ...result, name: item.file.name, mime: item.file.type || "application/octet-stream" }]);
            toast.success(`${item.file.name} uploaded`);
            anySuccess = true;
          } catch (err) {
            const message = err instanceof ApiError ? err.message : "Upload failed";
            setItems((prev) =>
              prev.map((i) =>
                i.id === item.id ? { ...i, status: "error", error: message } : i,
              ),
            );
            toast.error(`Failed to upload ${item.file.name}: ${message}`);
          }
        }
        setUploading(false);
        if (anySuccess) triggerRefresh();
      };

      uploadQueue().catch(console.error);
    },
    [triggerRefresh, mode],
  );

  const clearCompleted = useCallback(() => {
    setItems((prev) => prev.filter((i) => i.status === "uploading"));
    setResults([]);
  }, []);

  const hasCompleted = items.some(
    (i) => i.status === "complete" || i.status === "error",
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle>Upload Files</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <Dropzone
          onFilesSelected={handleFilesSelected}
          onFilesRejected={handleFilesRejected}
          disabled={uploading}
        />
        <UploadProgress items={items} />
        {results.length > 0 && (
          <div className="space-y-2">
            {results.map((r, i) => (
              <div key={`${r.sha256}-${i}`} className="rounded-md border p-3 text-sm">
                <p>{r.name} uploaded · {humanizeBytes(r.byte_length)} · {r.mime}</p>
                <details className="mt-1 text-xs text-muted-foreground">
                  <summary>Acquisition receipt</summary>
                  <p className="break-all">{r.acquisition_ref}</p>
                  <p className="break-all">SHA-256: {r.sha256}</p>
                  <p>Operating policy: {r.matter_mode}</p>
                </details>
              </div>
            ))}
          </div>
        )}
        {hasCompleted && !uploading && (
          <div className="flex justify-end">
            <Button variant="outline" size="sm" onClick={clearCompleted}>
              Clear completed
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
