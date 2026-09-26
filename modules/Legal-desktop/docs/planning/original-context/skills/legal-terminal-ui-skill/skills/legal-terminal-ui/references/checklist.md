# Pre-Ship Design Checklist

- [ ] Was the design interview run and final_recommendation set before this screen was built?
- [ ] Does the layout conceal complexity behind sensible defaults?
- [ ] Are panels modular, dockable, and savable as named workspace views?
- [ ] Is the color system built from dark-first tokens with proper elevation and desaturated accents?
- [ ] Have all color pairs passed scripts/contrast_check.py (WCAG 2.2)?
- [ ] Is status color always paired with an icon or label?
- [ ] Does every chart match its data type, with tables as first-class dense components?
- [ ] Is ingestion/processing status persistently visible with drill-down and retry actions?
- [ ] Can users export any visualized/tabular data as CSV/JSON?
- [ ] Are legal relationships visualized, not just listed?
- [ ] Is keyboard-first navigation supported for core terminal functions?
- [ ] Does data provenance show for case facts and documents?
- [ ] If AI chat/agent features are present, is there a confirmation step before any write action?
- [ ] If database views are present, is per-source provenance labeled?
