// Byline: Codex · GPT-5 · 2026-08-15 (case-scoped Knowledge MVP)
// Byline: Codex · GPT-5 · 2026-08-16 (canonical source/chunk inspector)
// Byline: Codex · GPT-5 · 2026-08-18 (authored/derived source catalog labels)
// Byline: Codex · GPT-5.6-Sol · 2026-08-29 (native evidence-only semantic search)
// Byline: Claude Code · Opus 5.5 · 2026-09-27 (DF-24: Graphiti memory pane removed; Graphiti is retired, D-070)
"use client";

import { FormEvent, useState } from "react";
import { BookOpen, Database, Eye, Loader2, Search } from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { AddToCaseDialog } from "@/components/matters/add-to-case-dialog";
import { KnowledgeItemDrawer } from "@/components/knowledge/knowledge-item-drawer";
import {
  ApiError,
  getKnowledgeContent,
  listKnowledgeContents,
  searchKnowledge,
} from "@/lib/api-client";
import type {
  EvidencePromotionResult,
  KnowledgeContentRow,
  KnowledgeItemDetail,
  KnowledgeLane,
  KnowledgeSearchHit,
  KnowledgeSourceRef,
  MatterDetail,
} from "@/lib/shared/types";

const PAGE_SIZE = 20;
const LANES = ["platform", "legal", "personal_history", "context", "evidence"] as const;
const EVIDENCE_SEARCH_LANE = "evidence" as const;
const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const SHA256_PATTERN = /^[0-9a-f]{64}$/i;

type View = "search" | "contents";

interface KnowledgeBrowserProps {
  matterContext?: {
    matter: MatterDetail;
    partitionKey: string;
    defaultCourtCaseId?: string;
    onEvidencePromoted: (result: EvidencePromotionResult) => void;
  };
}

function errorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback;
}

function metadataBadges(metadata?: Record<string, unknown> | null) {
  if (!metadata) return null;
  return ["knowledge_lane", "lane", "case_id", "source", "disclosure_tier"]
    .filter((key) => metadata[key] !== undefined && metadata[key] !== null)
    .map((key) => (
      <Badge key={key} variant="outline">
        {key}: {String(metadata[key])}
      </Badge>
    ));
}

function optionalString(value: unknown) {
  return typeof value === "string" && value.trim() ? value.trim() : undefined;
}

function sourceRefForHit(
  hit: KnowledgeSearchHit,
  fallbackLane: string,
  partitionKey: string,
): KnowledgeSourceRef | null {
  const metadata = hit.meta_data ?? {};
  const laneValue = optionalString(metadata.knowledge_lane) || optionalString(metadata.lane) || fallbackLane;
  const metadataPartition = optionalString(metadata.partition_key) || optionalString(metadata.case_id);
  const artifactId = optionalString(metadata.artifact_id);
  const sha256 = optionalString(metadata.sha256);
  if (
    laneValue !== "evidence" ||
    !partitionKey.trim() ||
    !hit.id?.trim() ||
    (metadataPartition !== undefined && metadataPartition !== partitionKey.trim()) ||
    !artifactId ||
    !UUID_PATTERN.test(artifactId) ||
    !sha256 ||
    !SHA256_PATTERN.test(sha256)
  ) {
    return null;
  }
  return {
    lane: laneValue as KnowledgeLane,
    partition_key: partitionKey.trim(),
    artifact_id: artifactId,
    sha256: sha256.toLowerCase(),
    conversation_id: optionalString(metadata.conversation_id),
    retrieval_ref: hit.id.trim(),
    content_ref: hit.content_id || undefined,
    chunk_ref: optionalString(metadata.chunk_ref) || optionalString(metadata.chunk_id),
  };
}

export function KnowledgeBrowser({ matterContext }: KnowledgeBrowserProps = {}) {
  const [view, setView] = useState<View>("search");
  const [caseId, setCaseId] = useState("primary");
  const [catalogLane, setCatalogLane] = useState("");
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(false);
  const [searchHits, setSearchHits] = useState<KnowledgeSearchHit[]>([]);
  const [searchCount, setSearchCount] = useState<number | null>(null);
  const [searchScope, setSearchScope] = useState<{ partitionKey: string; lane: string } | null>(null);

  const [contents, setContents] = useState<KnowledgeContentRow[]>([]);
  const [contentOffset, setContentOffset] = useState(0);
  const [contentTotal, setContentTotal] = useState(0);
  const [contentDetail, setContentDetail] = useState<KnowledgeItemDetail | null>(null);
  const [contentDetailLoading, setContentDetailLoading] = useState(false);
  const [contentDetailOpen, setContentDetailOpen] = useState(false);

  const activePartition = matterContext?.partitionKey ?? caseId;

  async function loadContents(targetOffset: number) {
    const selectedCase = activePartition.trim();
    if (!selectedCase) {
      toast.error("Case ID is required");
      return;
    }
    if (!catalogLane) {
      toast.error("Select a knowledge lane before loading sources");
      return;
    }
    setContents([]);
    setContentTotal(0);
    setLoading(true);
    try {
      const response = await listKnowledgeContents({
        caseId: selectedCase,
        lane: catalogLane,
        limit: PAGE_SIZE,
        offset: targetOffset,
      });
      setContents(response.data ?? []);
      setContentTotal(response.meta?.total_count ?? 0);
      setContentOffset(targetOffset);
    } catch (error) {
      setContents([]);
      setContentTotal(0);
      toast.error(errorMessage(error, "Failed to load knowledge contents"));
    } finally {
      setLoading(false);
    }
  }

  async function openContent(row: KnowledgeContentRow) {
    const selectedCase = activePartition.trim();
    if (!selectedCase) {
      toast.error("Case ID is required");
      return;
    }
    setContentDetail(null);
    setContentDetailOpen(true);
    setContentDetailLoading(true);
    try {
      setContentDetail(await getKnowledgeContent(row.id, selectedCase));
    } catch (error) {
      setContentDetailOpen(false);
      toast.error(errorMessage(error, "Failed to load canonical records"));
    } finally {
      setContentDetailLoading(false);
    }
  }

  async function runProjectionSearch(event?: FormEvent) {
    event?.preventDefault();
    if (!query.trim()) {
      toast.error("Enter a knowledge question or search phrase");
      return;
    }
    if (!activePartition.trim()) {
      toast.error("Case ID is required");
      return;
    }
    const requestedScope = { partitionKey: activePartition.trim(), lane: EVIDENCE_SEARCH_LANE };
    setSearchHits([]);
    setSearchCount(null);
    setSearchScope(null);
    setLoading(true);
    try {
      const response = await searchKnowledge(query.trim(), {
        caseId: requestedScope.partitionKey,
        lane: requestedScope.lane,
        limit: 20,
      });
      setSearchHits(response.data ?? []);
      setSearchCount(response.meta?.total_count ?? response.data?.length ?? 0);
      setSearchScope(requestedScope);
    } catch (error) {
      setSearchHits([]);
      setSearchScope(null);
      toast.error(errorMessage(error, "Knowledge search failed"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-4">
      {!matterContext && <div className="flex flex-wrap gap-2" role="tablist" aria-label="Knowledge views">
        <Button
          id="knowledge-search-tab"
          type="button"
          role="tab"
          aria-controls="knowledge-search-panel"
          aria-selected={view === "search"}
          variant={view === "search" ? "default" : "outline"}
          onClick={() => setView("search")}
        >
          <Search /> Projection search
        </Button>
        <Button
          id="knowledge-contents-tab"
          type="button"
          role="tab"
          aria-controls="knowledge-contents-panel"
          aria-selected={view === "contents"}
          variant={view === "contents" ? "default" : "outline"}
          onClick={() => setView("contents")}
        >
          <Database /> Sources
        </Button>
      </div>}

      {matterContext && (
        <aside className="rounded-md border border-blue-300 bg-blue-50 p-3 text-sm text-blue-950">
          <strong>Matter-bound knowledge projection.</strong> Searches are prefiltered to partition <code>{activePartition}</code> before vector ranking.
        </aside>
      )}

      {view === "search" && (
        <Card
          id="knowledge-search-panel"
          role={matterContext ? undefined : "tabpanel"}
          aria-labelledby={matterContext ? undefined : "knowledge-search-tab"}
          aria-busy={loading}
        >
          <CardHeader>
            <CardTitle>Weaviate projection search</CardTitle>
            <CardDescription>
              {matterContext
                ? "Projection results are prefiltered to this Matter's explicit Knowledge partition before vector ranking. PostgreSQL remains canonical and evidence promotion remains custody-gated."
                : "This is supplemental evidence-vector recall, not the authored store. Results are prefiltered by case before ranking; other lanes remain available in the canonical source browser."}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            <form className={`grid gap-3 lg:items-end ${matterContext ? "lg:grid-cols-[13rem_13rem_1fr_auto]" : "lg:grid-cols-[12rem_13rem_1fr_auto]"}`} onSubmit={runProjectionSearch}>
              {matterContext ? (
                <div className="space-y-1">
                  <Label>Knowledge scope</Label>
                  <div className="rounded-md border bg-muted/40 px-3 py-2 text-sm">
                    <span className="font-medium">{matterContext.matter.title}</span>
                    <span className="block font-mono text-xs text-muted-foreground">partition: {activePartition}</span>
                  </div>
                </div>
              ) : <div className="space-y-1">
                <Label htmlFor="knowledge-case">Case ID</Label>
                <Input id="knowledge-case" value={caseId} onChange={(e) => setCaseId(e.target.value)} />
              </div>}
              <div className="space-y-1">
                <Label htmlFor="knowledge-lane">Knowledge lane</Label>
                <select
                  id="knowledge-lane"
                  className="h-9 w-full rounded-md border border-input bg-muted px-3 text-sm"
                  value={EVIDENCE_SEARCH_LANE}
                  disabled
                >
                  <option value={EVIDENCE_SEARCH_LANE}>evidence (native)</option>
                </select>
              </div>
              <div className="space-y-1">
                <Label htmlFor="knowledge-query">Question or search phrase</Label>
                <Input
                  id="knowledge-query"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="What did we decide about the evidence workflow?"
                />
              </div>
              <Button type="submit" disabled={loading}>
                {loading ? <Loader2 className="animate-spin" aria-hidden="true" /> : <Search aria-hidden="true" />} Search
              </Button>
            </form>

            {loading && searchHits.length === 0 ? (
              <div className="space-y-2">{Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-28" />)}</div>
            ) : searchCount === null ? (
              <p className="py-10 text-center text-sm text-muted-foreground">Search the selected case to inspect grounded knowledge.</p>
            ) : searchHits.length === 0 ? (
              <p className="py-10 text-center text-sm text-muted-foreground">No matching knowledge found.</p>
            ) : (
              <div className="space-y-3">
                <p className="text-xs text-muted-foreground">
                  {searchCount} result{searchCount === 1 ? "" : "s"}
                  {searchScope && ` · scope ${searchScope.partitionKey} / ${searchScope.lane}`}
                </p>
                {searchHits.map((hit, index) => {
                  const source = searchScope
                    ? sourceRefForHit(hit, searchScope.lane, searchScope.partitionKey)
                    : null;
                  return (
                    <article key={hit.id || `${hit.content_id}-${index}`} className="rounded-lg border p-4">
                      <div className="mb-2 flex flex-wrap items-start justify-between gap-3">
                        <div className="flex flex-wrap items-center gap-2">
                          <BookOpen className="h-4 w-4 text-muted-foreground" />
                          <span className="font-medium">{hit.name || hit.content_origin || "Knowledge result"}</span>
                          {hit.reranking_score !== null && hit.reranking_score !== undefined && <Badge variant="secondary">score {hit.reranking_score.toFixed(3)}</Badge>}
                          {metadataBadges(hit.meta_data)}
                        </div>
                        {source ? (
                          <AddToCaseDialog
                            source={source}
                            defaultTitle={hit.name || hit.content_origin || "Knowledge-derived evidence"}
                            boundMatter={matterContext?.matter}
                            defaultCourtCaseId={matterContext?.defaultCourtCaseId}
                            onPromoted={matterContext?.onEvidencePromoted}
                          />
                        ) : (
                          <Badge variant="destructive">Not promotable: evidence custody metadata required</Badge>
                        )}
                      </div>
                      <p className="whitespace-pre-wrap text-sm leading-6">{hit.content}</p>
                    </article>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {!matterContext && view === "contents" && (
        <Card
          id="knowledge-contents-panel"
          role="tabpanel"
          aria-labelledby="knowledge-contents-tab"
          aria-busy={loading}
        >
          <CardHeader>
            <CardTitle>Knowledge sources</CardTitle>
            <CardDescription>
              Authored normalized sources with separately labeled, rebuildable derived chunks. Open a source to inspect lineage and acquisition context.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <form
              className="grid gap-3 sm:grid-cols-[12rem_13rem_auto] sm:items-end"
              onSubmit={(event) => {
                event.preventDefault();
                void loadContents(0);
              }}
            >
              <div className="space-y-1">
                <Label htmlFor="sources-case">Case ID</Label>
                <Input
                  id="sources-case"
                  value={caseId}
                  onChange={(event) => {
                    setCaseId(event.target.value);
                    setContents([]);
                    setContentTotal(0);
                    setContentDetail(null);
                    setContentDetailOpen(false);
                  }}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="sources-lane">Knowledge lane</Label>
                <select
                  id="sources-lane"
                  required
                  className="h-9 w-full rounded-md border border-input bg-transparent px-3 text-sm"
                  value={catalogLane}
                  onChange={(event) => {
                    setCatalogLane(event.target.value);
                    setContents([]);
                    setContentTotal(0);
                    setContentDetail(null);
                    setContentDetailOpen(false);
                  }}
                >
                  <option value="">Select a lane</option>
                  {LANES.map((item) => (
                    <option key={item} value={item}>{item}</option>
                  ))}
                </select>
              </div>
              <Button type="submit" disabled={loading || !catalogLane}>
                {loading ? <Loader2 className="animate-spin" aria-hidden="true" /> : <Database aria-hidden="true" />}
                Load sources
              </Button>
            </form>
            {loading && contents.length === 0 ? (
              <div className="space-y-2">{Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-16" />)}</div>
            ) : contents.length === 0 ? (
              <p className="py-10 text-center text-sm text-muted-foreground">No knowledge sources found.</p>
            ) : (
              <div className="divide-y rounded-lg border">
                {contents.map((row) => (
                  <div key={row.id} className="space-y-2 p-4">
                    <div className="flex flex-wrap items-start justify-between gap-3">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-medium">{row.name || row.id}</span>
                        {row.type && <Badge variant="outline">{row.type}</Badge>}
                        {row.status && <Badge variant={row.status === "completed" ? "secondary" : "outline"}>{row.status}</Badge>}
                      </div>
                      <Button type="button" size="sm" variant="outline" onClick={() => void openContent(row)}>
                        <Eye className="size-4" aria-hidden="true" /> Inspect source
                      </Button>
                    </div>
                    {row.description && <p className="text-sm text-muted-foreground">{row.description}</p>}
                    <div className="flex flex-wrap gap-2">
                      {metadataBadges(row.metadata)}
                      {row.metadata?.parser_id !== undefined && row.metadata.parser_id !== null && <Badge variant="secondary">parser: {String(row.metadata.parser_id)}</Badge>}
                      {row.metadata?.chunker_id !== undefined && row.metadata.chunker_id !== null && <Badge variant="secondary">chunker: {String(row.metadata.chunker_id)}</Badge>}
                      {row.metadata?.record_count !== undefined && <Badge variant="outline">{String(row.metadata.record_count)} authored records</Badge>}
                      {row.metadata?.chunk_count !== undefined && <Badge variant="outline">{String(row.metadata.chunk_count)} derived chunks</Badge>}
                    </div>
                    {row.metadata?.source_sha256 !== undefined && row.metadata.source_sha256 !== null && (
                      <p className="break-all font-mono text-xs text-muted-foreground">
                        SHA-256 {String(row.metadata.source_sha256)}
                      </p>
                    )}
                  </div>
                ))}
              </div>
            )}
            <div className="flex items-center justify-between">
              <span className="text-xs text-muted-foreground">
                {contentTotal ? `${contentOffset + 1}–${Math.min(contentOffset + PAGE_SIZE, contentTotal)} of ${contentTotal}` : "0 sources"}
              </span>
              <div className="flex gap-2">
                <Button variant="outline" size="sm" disabled={contentOffset === 0 || loading} onClick={() => void loadContents(Math.max(0, contentOffset - PAGE_SIZE))}>Previous</Button>
                <Button variant="outline" size="sm" disabled={contentOffset + PAGE_SIZE >= contentTotal || loading} onClick={() => void loadContents(contentOffset + PAGE_SIZE)}>Next</Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <KnowledgeItemDrawer
        detail={contentDetail}
        loading={contentDetailLoading}
        open={contentDetailOpen}
        onOpenChange={(open) => {
          setContentDetailOpen(open);
          if (!open && !contentDetailLoading) setContentDetail(null);
        }}
      />
    </div>
  );
}
