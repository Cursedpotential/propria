// Byline: Claude Code · Opus 5.5 · 2026-09-26
// The repair builder's editable step list. Steps come only from the engine's tool list
// (GET /api/proffer/repair/tools); each can be added, removed and moved, and its settings
// are edited from its tool's params_schema. Whether the chain is valid is the engine's call
// (Validate), never decided here.
"use client";

import { ArrowDown, ArrowUp, Plus, Trash2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  MAX_REPAIR_STEPS,
  addStep,
  moveStep,
  numberParam,
  paramFields,
  removeStep,
  setParam,
  stepId,
  toolLabel,
  updateStepParams,
  type RepairDraftStep,
  type RepairParamField,
} from "@/lib/repair-plan";
import type { ProfferRepairParamValue, ProfferRepairTool } from "@/lib/shared/types";

function types(tool: ProfferRepairTool) {
  return `${tool.input_types.join(" | ")} → ${tool.output_types.join(" | ")}`;
}

function ParamField({
  stepKey,
  field,
  value,
  disabled,
  onChange,
}: {
  stepKey: string;
  field: RepairParamField;
  value: ProfferRepairParamValue | undefined;
  disabled: boolean;
  onChange: (value: ProfferRepairParamValue | undefined) => void;
}) {
  const id = `${stepKey}-${field.name}`;
  const label = field.name.replaceAll("_", " ");
  const row = "flex min-w-0 items-center justify-between gap-2";
  if (field.kind === "boolean") {
    const checked = typeof value === "boolean" ? value : field.defaultValue === true;
    return (
      <div className={row} title={field.description || undefined}>
        <Label htmlFor={id} className="text-[11px] font-normal">{label}</Label>
        <Switch id={id} checked={checked} disabled={disabled} onCheckedChange={(next) => onChange(next)} />
      </div>
    );
  }
  if (field.kind === "choice") {
    const current = value ?? field.defaultValue;
    return (
      <div className={row} title={field.description || undefined}>
        <span className="text-[11px]">{label}</span>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button type="button" variant="outline" size="xs" disabled={disabled} aria-label={label}>
              {current === undefined ? "choose" : String(current)}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuRadioGroup
              value={current === undefined ? "" : String(current)}
              onValueChange={(next) => onChange(field.choices?.find((choice) => String(choice) === next))}
            >
              {field.choices?.map((choice) => (
                <DropdownMenuRadioItem key={String(choice)} value={String(choice)}>{String(choice)}</DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    );
  }
  const numeric = field.kind === "integer" || field.kind === "number";
  return (
    <div className={row} title={field.description || undefined}>
      <Label htmlFor={id} className="text-[11px] font-normal">{label}</Label>
      <Input
        id={id}
        type={numeric ? "number" : "text"}
        inputMode={field.kind === "integer" ? "numeric" : undefined}
        min={field.minimum}
        max={field.maximum}
        step={field.kind === "integer" ? 1 : field.kind === "number" ? "any" : undefined}
        placeholder={field.defaultValue === undefined ? "" : String(field.defaultValue)}
        value={value === undefined ? "" : String(value)}
        disabled={disabled}
        className="h-7 w-24 px-2 text-xs md:text-xs"
        onChange={(event) => onChange(numeric ? numberParam(event.target.value) : event.target.value || undefined)}
      />
    </div>
  );
}

function StepRow({
  step,
  index,
  count,
  tool,
  toolsLoaded,
  disabled,
  onMove,
  onRemove,
  onParams,
}: {
  step: RepairDraftStep;
  index: number;
  count: number;
  tool: ProfferRepairTool | undefined;
  toolsLoaded: boolean;
  disabled: boolean;
  onMove: (delta: -1 | 1) => void;
  onRemove: () => void;
  onParams: (params: Record<string, ProfferRepairParamValue>) => void;
}) {
  const id = stepId(index);
  return (
    <li className="space-y-1 border bg-background/60 p-1.5" data-activity={step.activity}>
      <div className="flex min-w-0 items-center gap-1">
        <span className="font-mono text-[10px] text-muted-foreground">{id}</span>
        <span className="min-w-0 flex-1 truncate text-xs font-medium" title={tool?.description || step.activity}>
          {toolLabel(step.activity)}
        </span>
        {tool?.needs_n8n && <Badge variant="outline" className="px-1 py-0 text-[9px]">n8n</Badge>}
        <Button type="button" variant="ghost" size="icon-xs" aria-label={`Move ${id} up`} disabled={disabled || index === 0} onClick={() => onMove(-1)}>
          <ArrowUp />
        </Button>
        <Button type="button" variant="ghost" size="icon-xs" aria-label={`Move ${id} down`} disabled={disabled || index === count - 1} onClick={() => onMove(1)}>
          <ArrowDown />
        </Button>
        <Button type="button" variant="ghost" size="icon-xs" aria-label={`Remove ${id}`} disabled={disabled} onClick={onRemove}>
          <Trash2 />
        </Button>
      </div>
      {tool && (
        <p className="truncate font-mono text-[10px] text-muted-foreground" title={types(tool)}>
          {types(tool)} · {tool.writes === "derived" ? "writes a derived copy" : "reads only"}
        </p>
      )}
      {toolsLoaded && !tool && <p className="text-[11px] text-destructive">Not in the engine&apos;s tool list</p>}
      {paramFields(tool?.params_schema).map((field) => (
        <ParamField
          key={field.name}
          stepKey={step.key}
          field={field}
          value={step.params[field.name]}
          disabled={disabled}
          onChange={(value) => onParams(setParam(step.params, field.name, value))}
        />
      ))}
    </li>
  );
}

/** The tool list, shown in full while the plan is empty: every tool with what it does. */
function ToolList({ tools, disabled, onAdd }: { tools: readonly ProfferRepairTool[]; disabled: boolean; onAdd: (activity: string) => void }) {
  return (
    <ul className="space-y-1" aria-label="Repair tools" data-testid="repair-tool-list">
      {tools.map((tool) => (
        <li key={tool.id} className="space-y-0.5 border p-1.5">
          <div className="flex min-w-0 items-center gap-1">
            <span className="min-w-0 flex-1 truncate text-xs font-medium" title={tool.id}>{toolLabel(tool.id)}</span>
            {tool.needs_n8n && <Badge variant="outline" className="px-1 py-0 text-[9px]">n8n</Badge>}
            <Button type="button" variant="outline" size="xs" disabled={disabled} onClick={() => onAdd(tool.id)}>
              <Plus /> Add
            </Button>
          </div>
          <p className="text-[11px] leading-4 text-muted-foreground">{tool.description}</p>
          <p className="truncate font-mono text-[10px] text-muted-foreground">{types(tool)}</p>
        </li>
      ))}
    </ul>
  );
}

export function RepairStepEditor({
  steps,
  tools,
  toolsLoading,
  toolsError,
  disabled,
  onChange,
}: {
  steps: readonly RepairDraftStep[];
  tools: readonly ProfferRepairTool[];
  toolsLoading: boolean;
  toolsError: string | null;
  disabled: boolean;
  onChange: (steps: RepairDraftStep[]) => void;
}) {
  const byId = new Map(tools.map((tool) => [tool.id, tool]));
  const toolsLoaded = !toolsLoading && !toolsError;
  const full = steps.length >= MAX_REPAIR_STEPS;
  return (
    <div className="space-y-1.5" data-testid="repair-step-editor">
      {steps.length > 0 && (
        <ol className="space-y-1" aria-label="Plan steps">
          {steps.map((step, index) => (
            <StepRow
              key={step.key}
              step={step}
              index={index}
              count={steps.length}
              tool={byId.get(step.activity)}
              toolsLoaded={toolsLoaded}
              disabled={disabled}
              onMove={(delta) => onChange(moveStep(steps, index, delta))}
              onRemove={() => onChange(removeStep(steps, index))}
              onParams={(params) => onChange(updateStepParams(steps, index, params))}
            />
          ))}
        </ol>
      )}
      {toolsLoading && <p className="text-[11px] text-muted-foreground" role="status">Reading the tool list…</p>}
      {toolsError && <p className="text-[11px] text-destructive" role="status">{toolsError}</p>}
      {toolsLoaded && steps.length === 0 && <ToolList tools={tools} disabled={disabled} onAdd={(activity) => onChange(addStep(steps, activity))} />}
      {toolsLoaded && steps.length > 0 && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              type="button"
              variant="outline"
              size="xs"
              disabled={disabled || full}
              title={full ? `A plan has at most ${MAX_REPAIR_STEPS} steps` : undefined}
            >
              <Plus /> Add step
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="max-w-[20rem]">
            {tools.map((tool) => (
              <DropdownMenuItem key={tool.id} className="flex-col items-start gap-0.5" onSelect={() => onChange(addStep(steps, tool.id))}>
                <span className="font-medium">{toolLabel(tool.id)}</span>
                <span className="font-mono text-[10px] text-muted-foreground">{types(tool)}</span>
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  );
}
