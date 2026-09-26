// Byline: Grok · grok-4.6 · 2026-08-18
// Auth boundary: Codex · GPT-5 · 2026-09-12
// Browser and server components use the same-origin Next BFF. Authentik's
// identity JWT and the private API address never enter browser JavaScript.

export function legalApiBase(): string {
  if (typeof window === "undefined") {
    const webBase = process.env.LEGAL_WEB_INTERNAL_URL ?? "http://127.0.0.1:3000";
    return `${webBase.replace(/\/$/, "")}/api/legal`;
  }
  return "/api/legal";
}

export async function fetchHealth(): Promise<{
  status: string;
  service: string;
  evidence_platform: string;
}> {
  const response = await fetch(`${legalApiBase()}/health`, {
    cache: "no-store"
  });
  if (!response.ok) {
    throw new Error(`legal-api health ${response.status}`);
  }
  return response.json();
}

export async function fetchAgnoStatus(): Promise<{
  reachable: boolean;
  evidence_platform: string;
  matters_visible: boolean;
}> {
  const response = await fetch(`${legalApiBase()}/v1/agno/status`, {
    cache: "no-store"
  });
  if (!response.ok) {
    throw new Error(`legal-api agno status ${response.status}`);
  }
  return response.json();
}

export async function fetchMatter(): Promise<{
  matter: { display_name: string };
  court_case: { display_name: string; court: string | null };
  package_imported: boolean;
  accepted_item_count: number;
  omitted_item_ids: string[];
  issue: IssueNode;
  factor_count: number;
  draft_count: number;
  authority_count: number;
  strategy_count: number;
  redteam_count: number;
  open_todo_count: number;
  review_count: number;
  release_count: number;
  open_research_count: number;
  discovery_count: number;
  investigation_count: number;
  upcoming_event_count: number;
  upcoming_events: Array<{
    event_id: string;
    occurs_at: string;
    title: string;
    kind: string;
    location: string;
    confirmed: boolean;
  }>;
  agent_run_count: number;
  exhibit_count: number;
  audit_count?: number;
  judge_confirmed: boolean;
  foc_confirmed: boolean;
  next_surfaces: Array<{ path: string; label: string; help: string }>;
}> {
  const response = await fetch(`${legalApiBase()}/v1/matter`, {
    cache: "no-store"
  });
  if (!response.ok) {
    throw new Error(`legal-api matter ${response.status}`);
  }
  return response.json();
}

export type FactorSide = {
  for_parent: string[];
  citations: unknown[];
};

export type FactorRow = {
  letter: string;
  title: string;
  authority: string;
  petitioner: FactorSide;
  respondent: FactorSide;
  contradictions: string[];
  missing_proof: string[];
  both_parent_analysis: string | null;
};

export type FactorAnalysisView = {
  letter: string;
  petitioner: FactorSide;
  respondent: FactorSide;
  contradictions: string[];
  missing_proof: string[];
  both_parent_analysis: string;
};

export type IssueNode = {
  issue_id?: string;
  title: string;
  governing_authority: string;
  elements: Array<{
    element_id?: string;
    label: string;
    supporting: unknown[];
    contradicting: unknown[];
    missing_proof: string[];
  }>;
  children: IssueNode[];
};

function emptySide(): FactorSide {
  return { for_parent: [], citations: [] };
}

export function structuralBothParentPrompt(factor: FactorRow): string {
  return [
    "STRUCTURAL BOTH-PARENT PROMPT — not a finding.",
    "court_safe=false. Scratch only. Do not diagnose. Do not invent facts.",
    `Factor (${factor.letter}) ${factor.title} under ${factor.authority}.`,
    "Address BOTH sides — petitioner and respondent — including",
    `petitioner notes (${factor.petitioner.for_parent.length}) and citations (${factor.petitioner.citations.length}),`,
    `respondent notes (${factor.respondent.for_parent.length}) and citations (${factor.respondent.citations.length}).`,
    "Name contradictions and missing proof explicitly.",
    "Weighting stays an owner decision.",
  ].join(" ");
}

export async function fetchFactors(): Promise<FactorRow[]> {
  const response = await fetch(`${legalApiBase()}/v1/factors`, {
    cache: "no-store"
  });
  if (!response.ok) {
    throw new Error(`legal-api factors ${response.status}`);
  }
  const rows: FactorRow[] = await response.json();
  return rows.map((row) => ({
    ...row,
    petitioner: { ...emptySide(), ...row.petitioner },
    respondent: { ...emptySide(), ...row.respondent },
    contradictions: row.contradictions ?? [],
    missing_proof: row.missing_proof ?? [],
    both_parent_analysis: row.both_parent_analysis ?? null,
  }));
}

export async function fetchFactorAnalysis(letter: string): Promise<FactorAnalysisView | null> {
  const response = await fetch(`${legalApiBase()}/v1/factors/${letter}/analysis`, {
    cache: "no-store"
  });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    return null;
  }
  return response.json();
}

function normalizeIssue(raw: Partial<IssueNode> & { title: string; governing_authority: string }): IssueNode {
  return {
    issue_id: raw.issue_id,
    title: raw.title,
    governing_authority: raw.governing_authority,
    elements: (raw.elements ?? []).map((element) => ({
      element_id: element.element_id,
      label: element.label,
      supporting: element.supporting ?? [],
      contradicting: element.contradicting ?? [],
      missing_proof: element.missing_proof ?? [],
    })),
    children: (raw.children ?? []).map((child) => normalizeIssue(child)),
  };
}

export async function fetchIssueTree(): Promise<IssueNode> {
  const response = await fetch(`${legalApiBase()}/v1/issues`, {
    cache: "no-store"
  });
  if (response.ok) {
    return normalizeIssue(await response.json());
  }
  const matter = await fetchMatter();
  return normalizeIssue(matter.issue);
}

export type ProviderRow = {
  id: string;
  display_name: string;
  train: boolean;
  retain: string;
  confidential_eligible: boolean;
  role: string;
  notes: string;
  use: string;
};

export type ProviderGrid = {
  rows: ProviderRow[];
  eligible_confidential_models: string[];
  pacer: boolean;
  local_ollama_as_trust_posture: boolean;
  court_safe: boolean;
  legal_conclusion: boolean;
  source: string;
  disclaimer: string;
};

export async function fetchProviders(): Promise<ProviderGrid> {
  const response = await fetch(`${legalApiBase()}/v1/providers`, {
    cache: "no-store"
  });
  if (!response.ok) {
    throw new Error(`legal-api providers ${response.status}`);
  }
  return response.json();
}

export async function fetchDrafts(): Promise<
  Array<{
    section_id: string;
    heading: string;
    body: string;
    factor_letter: string | null;
    citation_count: number;
    unsupported: boolean;
    support: {
      unsupported_count: number;
      paragraphs: Array<{ index: number; text: string; state: string }>;
    } | null;
  }>
> {
  const response = await fetch(`${legalApiBase()}/v1/drafts`, {
    cache: "no-store"
  });
  if (!response.ok) {
    throw new Error(`legal-api drafts ${response.status}`);
  }
  return response.json();
}
