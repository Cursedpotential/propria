// Byline: Claude Code · Opus 5.5 · 2026-09-24; schemes 2026-09-25
// Category strip: rows stacked on top of each other; each row holds blocks side by side; each block is split into
// colored category segments sized by weight. Categories and colors come from the data: any category gets the next
// color from the chosen scheme (schemes.ts) unless `colors` pins one. Feed it anything with `rowsFromRecords` (row, block, category, weight).
// Origin: the chart on the owner's bout review page (owner 2026-09-24: "pull in different categories, assign them
// colors and stack them on top of each other").

import type { KeyboardEvent } from "react";

import "./category-strip.css";

import { categoryTotals, type StripRow } from "./rows";
import { SCHEMES, schemeColors, schemeOrder, schemeValue, type Scheme } from "./schemes";

export interface CategoryStripProps {
  rows: StripRow[];
  /** How categories are colored and ordered (see schemes.ts); default: categorical, colors in order of appearance. */
  scheme?: Scheme;
  /** Fixed colors for categories that have a meaning (any CSS color); every other category gets a palette color. */
  colors?: Record<string, string>;
  /** Legend order; default: along the scheme's scale, else largest first. */
  order?: string[];
  /** Unit shown in the legend and row totals, e.g. "msgs". */
  unit?: string;
  description?: string;
  showLegend?: boolean;
  selectedRowId?: string | null;
  /** When set, each row is a button that reports its id. */
  onSelectRow?: (rowId: string) => void;
}

export function CategoryStrip({
  rows,
  scheme = SCHEMES.categorical,
  colors,
  order,
  unit = "",
  description,
  showLegend = true,
  selectedRowId = null,
  onSelectRow,
}: CategoryStripProps) {
  const bySize = categoryTotals(rows);
  const totals = categoryTotals(rows, order ?? schemeOrder(bySize.map(([c]) => c), scheme));
  const color = schemeColors(totals.map(([c]) => c), scheme, colors);
  const offScale = (c: string) => scheme.kind !== "categorical" && !colors?.[c] && schemeValue(scheme, c) === undefined;

  return (
    <section className="category-strip">
      {showLegend ? (
        <div className="category-strip-legend" aria-label="Categories">
          {totals.map(([category, weight]) => (
            <span key={category} className={offScale(category) ? "off-scale" : undefined}
                  title={offScale(category) ? `Not on the ${scheme.label} scale` : undefined}>
              <i style={{ background: color[category] }} />
              {category} · {weight.toLocaleString()}
              {unit ? ` ${unit}` : ""}
            </span>
          ))}
        </div>
      ) : null}
      {description ? <p className="category-strip-description">{description}</p> : null}
      <div className="category-strip-rows">
        {rows.map((row) => {
          const select = onSelectRow ? () => onSelectRow(row.id) : undefined;
          const onKeyDown = select
            ? (e: KeyboardEvent) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  select();
                }
              }
            : undefined;
          const total = row.total ?? row.blocks.reduce((sum, b) => sum + b.segments.reduce((s, x) => s + x.weight, 0), 0);
          return (
            <div
              key={row.id}
              className="category-strip-row"
              role={select ? "button" : undefined}
              tabIndex={select ? 0 : undefined}
              aria-pressed={select ? selectedRowId === row.id : undefined}
              onClick={select}
              onKeyDown={onKeyDown}
            >
              <span className="category-strip-label">{row.label}</span>
              <div className="category-strip-bar">
                {row.blocks.map((block) => (
                  <span
                    key={block.id}
                    className="category-strip-block"
                    style={{ flexGrow: Math.max(1, block.weight ?? block.segments.reduce((s, x) => s + x.weight, 0)) }}
                    title={block.title}
                  >
                    {block.segments.length ? (
                      block.segments.map((seg, i) => (
                        <i
                          key={i}
                          style={{ flexGrow: Math.max(1, seg.weight), background: color[seg.category] }}
                          title={seg.title ?? `${seg.category} · ${seg.weight}`}
                        />
                      ))
                    ) : (
                      <i className="category-strip-empty" />
                    )}
                  </span>
                ))}
              </div>
              <span className="category-strip-total">{total.toLocaleString()}</span>
            </div>
          );
        })}
      </div>
    </section>
  );
}
