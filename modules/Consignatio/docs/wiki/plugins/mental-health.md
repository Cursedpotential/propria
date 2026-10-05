---
title: "mental-health"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# mental-health

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Personal mental-health reference: ADHD CBT techniques, psychoeducation on conditions, therapies, coping skills, medications, and a guidance persona (educational only). One progressive-disclosure entry skill (`mental-health:mental-health`) routing to 3 member skills loaded on demand.

Source: `E:/AI_Workspace/plugins/plugins/mental-health`. Version: `1.1.1`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `mh-adhd-cbt`

(mental-health) This skill should be used when the user asks for ADHD-focused Cognitive Behavioral Therapy (CBT) support. An evidence-based ADHD CBT companion built on the Safren (Harvard/MGH), Solanto (NYU), and Ramsay & Rostain protocols, integrating third-wave CBT techniques (ACT, DBT, mindfulness). Triggers when the user mentions ADHD CBT, cognitive behavioral therapy, thought records, automatic thoughts, cognitive restructuring, procrastination, attention management, executive function training, ADHD anxiety, ADHD emotional regulation, Coach B, distractibility delay, trouble starting tasks, ADHD self-management, ADHD treatment, thinking errors, adaptive thinking, ADHD coping, an ADHD session, or a CBT session. Also applies when the user describes ADHD-related struggles without naming CBT — e.g., 'my inner critic won't shut up,' 'I procrastinated again,' 'I can't get anything done,' 'I can't focus at all,' or 'I'm falling apart.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/mental-health/skills/adhd-cbt/SKILL.md:1>) · SHA-256 `5bea038f14ebed151373778401ad760714fa7f2cfaff4578b8a3d5dfef5a5d70`

### `mental-health`

(mental-health) Personal mental-health reference: ADHD CBT techniques, psychoeducation on conditions, therapies, coping skills, medications, and a guidance persona (educational only). Entry point / router — read this first, then load one member from references/. Triggers: adhd, cbt, anxiety, depression, therapy, coping, psychoeducation, mental health, guidance. Members: adhd-cbt, mental-health-psychoeducation, provide-guidance.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/mental-health/skills/mental-health/SKILL.md:1>) · SHA-256 `51cc358b53af1f60417d4e01722046f644de48aedc826c8d6a8d2c0c92d1d8e5`

### `mh-mental-health-psychoeducation`

(mental-health) Comprehensive psychoeducation on mental health conditions, therapy modalities, evidence-based coping techniques, psychiatric medications, and self-assessment frameworks. Educational resource only — not medical advice, diagnosis, or treatment. Use when learning about mental health concepts, understanding therapy options, exploring coping strategies, or recognizing when to seek professional help. Trigger on 'mental health', 'therapy types', 'coping strategies', 'anxiety', 'depression', 'ADHD', 'psychiatric medication', 'when should I see a therapist'.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/mental-health/skills/mental-health-psychoeducation/SKILL.md:1>) · SHA-256 `67a3ea7c52424d178eab2c06ecf31db387238c142b11c6b47e1b135f73da2f7e`

### `mh-provide-guidance`

(mental-health) You are an all-knowing psychiatrist, psychologist, and life coach and you provide honest and concise advice to people based on the question asked combined with the context provided.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/mental-health/skills/provide-guidance/SKILL.md:1>) · SHA-256 `ed9e54f00f178cff2b146e0e461893693896dd5023fff2d875febd49bed19288`

## Agents

No entries found in the inspected declarations.

## Cli Entries

No entries found in the inspected declarations.

## Scripts

No entries found in the inspected declarations.

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.


Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
