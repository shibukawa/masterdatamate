---
name: spec-compiler
description: Initialize, author, validate, compile, search, query, audit, and maintain AI-friendly Markdown specification corpora. Use when Codex works with canonical target specs in specs/, shared specs in specs/shared/, generated artifacts in .speccompiler/generated/, project assets in spec-assets/, BACKLOG.md, speccompiler-style frontmatter facts, JSONL knowledge bases, backlinks, context bundles, impact analysis, corpus feedback, initial wizard blueprints, or Markdown backlog progress tracking.
---

# Spec Compiler

## Overview

Use this skill to manage a Markdown-first specification corpus as a compiled, AI-friendly knowledge base. Keep canonical detail in source Markdown documents, use deterministic scripts for structure-sensitive work, and use AI judgment for intent interpretation, concise authoring, audits, and recommendations.

The bundled helper script is a starter implementation of the workflow surface from the repo specs:

```bash
python <skill-dir>/scripts/speccompiler.py --help
```

Prefer project-local assets over skill assets. Never modify installed skill assets for project customization; materialize overrides under the configured project-local asset folders.

## Default Project Locations

Use these default locations unless `speccompiler.json` configures different paths:

- `specs/`: canonical target documentation folder. Create and edit source-of-truth specification documents here.
- `specs/shared/`: canonical shared or common-detail specs reused across document sets.
- `.speccompiler/generated/`: generated documentation and machine-readable artifacts, including JSONL records, backlinks, indexes, and relation data.
- `BACKLOG.md`: project-local Markdown backlog for documentation gaps, implementation tasks, decisions, validation findings, audits, and cleanup.
- `spec-assets/`: project-local mutable assets, including templates, type definitions, retrieval recipes, and feedback rules.
- `speccompiler.json`: project-local configuration that may override these defaults.

When editing resulting specs, inspect `specs/`, `specs/shared/`, `.speccompiler/generated/`, and relevant backlog items first so the current compiled corpus informs the edit. Treat `specs/` and `specs/shared/` as canonical source. Treat `.speccompiler/generated/` as derived context that may guide edits but should not become the source of truth.

## Core Workflow

1. Inspect existing project material before asking questions.
2. Initialize or locate the spec project configuration:
   ```bash
   python <skill-dir>/scripts/speccompiler.py init --project .
   ```
3. Create or update canonical Markdown documents in the configured target documentation folder, defaulting to `specs/`, from resolved templates:
   ```bash
   python <skill-dir>/scripts/speccompiler.py create-doc --project . --type api --title "Refresh access token"
   ```
4. Validate before and after edits:
   ```bash
   python <skill-dir>/scripts/speccompiler.py validate --project .
   ```
5. Compile generated artifacts after valid changes:
   ```bash
   python <skill-dir>/scripts/speccompiler.py compile --project .
   ```
6. Use search, impact, and context commands to load only the smallest useful slices:
   ```bash
   python <skill-dir>/scripts/speccompiler.py search --project . "token refresh"
   python <skill-dir>/scripts/speccompiler.py impact --project . auth-api
   python <skill-dir>/scripts/speccompiler.py context --project . "implement token refresh"
   ```
7. Track deferred documentation, implementation, decisions, validation findings, audits, and cleanup in the project backlog:
   ```bash
   python <skill-dir>/scripts/speccompiler.py backlog add --project . --title "Document token expiry rules" --type documentation
   ```

## Authoring Rules

- Treat each detailed document as one canonical concern.
- Require frontmatter with stable `id`, `type`, and `title`.
- Use ordinary Markdown links for relations; infer relation semantics from section/table context.
- Keep sections compact enough to retrieve independently.
- Put reusable shared facts in common-detail documents and link to them instead of duplicating.
- Use optional native-language summaries only as human explanation, not as canonical truth.
- When a document is intentionally incomplete, make it a blueprint and create backlog items for missing facts, decisions, or validation follow-ups.

## Asset Resolution

Use this precedence for templates, type definitions, retrieval recipes, and feedback rules:

1. project-local asset override from the configured project asset folders,
2. skill-provided baseline asset from `assets/`,
3. generic fallback template only when no specialized template exists.

Customization workflow:

1. Resolve the effective asset.
2. If the requested change targets a skill asset, copy it into the matching project-local asset folder first.
3. Edit the project-local copy.
4. Validate or dry-run affected documents.
5. Compile affected outputs when practical.

## Initial Wizard

For uncertain projects, use the wizard workflow before implementation:

```bash
python <skill-dir>/scripts/speccompiler.py wizard --project . --write
```

Inspect source directories, manifests, tests, docs, and user intent. Classify the entry point as legacy source, incomplete POC, scratch idea, inspired project, docs-only project, active-codebase change, migration, or rewrite. Ask only unresolved questions that affect partitioning, first runnable slice, source inheritance, local execution, or blockers.

Default to modular monolith boundaries unless independent deployment, ownership, scaling, runtime isolation, integration, security, or compliance constraints justify microservices.

Wizard output should include detected entry point, architecture recommendation, first runnable slice, initial blueprint documents, inherited/ignored/uncertain source areas, backlog path, open decisions, blockers, and handoff context.

## Validation And Repair

Run deterministic validation for metadata, duplicate IDs, malformed frontmatter, dangling links, invalid declared facts, stale generated outputs, and cross-set references. Apply only safe structural repairs automatically:

- convert resolvable bare IDs to Markdown links,
- refresh link captions from target titles,
- rewrite inbound paths after moves while preserving document IDs.

Meaning-changing edits, inferred facts, or semantic restructures require user or calling-agent acceptance.

## Query And Context

Compile before graph or context work. Prefer section-level search records and typed relation expansion to full-document loading. A context bundle should include IDs, titles, selected section text, key facts, nearby relations, estimated token cost, sufficiency signal, and omitted-but-relevant context when budget prevents inclusion.

Include active backlog items when they affect the current task.

## Audits And Feedback

Keep deterministic compiler findings separate from AI advisory findings. Use explicit AI audits for spec/code drift, component responsibility drift, API implementation mismatch, and likely missing relationships. Report evidence, confidence, affected specs, and implementation paths; do not silently rewrite canonical semantics.

For corpus feedback, rank recommendations by evidence and likely retrieval/answerability benefit. Classify each recommendation as safe repair, proposed semantic change, or aggressive-mode candidate. Backlog recommendations that are not applied immediately.

## References

Read only the relevant reference file when more detail is needed:

- `references/product-requirements.md` for core product model, authoring model, compiler outputs, and validation requirements.
- `references/skill-workflow-requirements.md` for required skill actions and workflow diagrams.
- `references/document-type-requirements.md` for extensible document type behavior and baseline types.
- `references/initial-workflow-requirements.md` for initial wizard, blueprint, backlog, and first implementation requirements.
