// Byline amendment: Codex · GPT-5 · 2026-09-12 (Propria semantic token adoption)
import { PrivilegeScanForm } from "@/components/PrivilegeScanForm";
import { ProviderTermsGrid } from "@/components/ProviderTermsGrid";
import { fetchDrafts, fetchProviders } from "@/lib/api/client";
import type { ProviderGrid } from "@/lib/api/client";

export default async function PrivilegePage() {
  let drafts: Awaited<ReturnType<typeof fetchDrafts>> = [];
  let error: string | null = null;
  let grid: ProviderGrid | null = null;
  let gridError: string | null = null;
  try {
    drafts = await fetchDrafts();
  } catch (exc) {
    error = exc instanceof Error ? exc.message : "legal-api unreachable";
  }
  try {
    grid = await fetchProviders();
  } catch (exc) {
    gridError = exc instanceof Error ? exc.message : "legal-api providers unreachable";
  }

  return (
    <>
      <h1 className="legal">Confidentiality check</h1>
      <p>
        Select a saved draft or paste text to find phrases associated with attorney
        communications, litigation preparation, case strategy, medical information,
        and child-identifying details. The keyword scan lists each match with its
        category and surrounding text so you can review it before sharing.
      </p>
      {error ? <p>{error}</p> : null}
      <PrivilegeScanForm drafts={drafts} />
      <ProviderTermsGrid grid={grid} error={gridError} />
    </>
  );
}
