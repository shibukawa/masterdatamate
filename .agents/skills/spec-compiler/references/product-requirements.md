# AI-Friendly Spec Compiler — Product Requirements

## 1. Purpose

Large software systems need specification documents that are readable by humans, usable by AI agents, and consistent across many files over time. Plain Markdown is easy to author and search, but it is not a compiled language: links can drift, duplicated facts can diverge, and large corpora become expensive for AI systems to load into context.

This product provides a **skill-delivered, AI-first context compiler** for AI-assisted software teams. Authors write human-readable Markdown documents with light structured metadata; the system compiles them into a validated, searchable, graph-aware knowledge base that AI agents can load incrementally under a context budget.

The primary interaction model is AI-mediated discovery, explanation, impact analysis, and editing. Direct Markdown reading remains important as a fallback, review, and debugging surface, but the product is optimized for humans and agents to ask the corpus questions instead of manually reading large documentation sets end to end.

The product should feel like a **CLI wiki plus compiler**:

- humans and AI can author documents naturally in Markdown,
- important facts are declared in machine-checkable form,
- the compiler verifies structure and consistency,
- generated indexes, backlinks, search records, and diagrams keep the corpus navigable at scale,
- AI agents can retrieve only the smallest relevant slices, expand through typed links when needed, and stop before loading entire documents or the whole repository.

## 2. Goals

### 2.1 Primary goals

1. **Make large specification corpora safe to evolve** through deterministic validation of IDs, links, and declared facts.
2. **Reduce AI context consumption** by compiling documents into compact searchable slices and structured indexes.
3. **Preserve Markdown-first authoring** so documents remain easy for humans and AI agents to write, review, and diff.
4. **Support whole-system understanding** across architecture, database, API, and UI concerns for monoliths, modular monoliths, and microservice systems.
5. **Provide bidirectional editing workflows** so users can express intent from either detailed documents or generated index views.
6. **Generate useful visualizations** such as C4-style architecture diagrams, ERDs, and DFDs from declared relations.
7. **Orchestrate incremental AI context loading** through token-aware search, graph expansion, and task-specific context recipes.
8. **Continuously improve corpus structure** through feedback reports that identify weak document granularity, missing links, and inefficient retrieval paths.
9. **Reduce first-implementation overhead** by creating thin blueprint documents, explicit backlog tasks, and focused handoff context instead of requiring complete documentation before coding begins.

### 2.2 Non-goals for v1

1. Full natural-language contradiction detection over arbitrary prose.
2. A mandatory hosted service or always-on backend.
3. Full multilingual duplication of all canonical content.
4. Replacing source control, code review, or project management systems.
5. Locking exact command names or final CLI syntax before the skill workflows are validated.
6. Automatically applying meaning-changing document restructures without explicit opt-in.

## 3. Target Users

### 3.1 Primary users

- **AI-assisted software teams** maintaining medium to large systems.
- Human engineers, architects, and product owners who need readable specifications.
- AI coding/document agents that need reliable, low-token access to the current system model.

### 3.2 User needs

Users need to:

- find the correct specification quickly,
- understand how a concept connects to other concepts,
- detect when two documents disagree,
- update the system from either a detailed view or an overview/index view,
- visualize architecture and data flow without manually redrawing diagrams,
- give AI agents only the context required for a task.

## 4. Product Principles

1. **Markdown first, not Markdown only**  
   Human-readable Markdown remains the authoring surface, while structured metadata and declared facts provide compile-time guarantees.

2. **Canonical detail, generated overview**  
   Detailed documents are the persisted source of truth. Index documents are generated views, but users may initiate edits from them through AI-assisted workflows.

3. **Deterministic where possible, AI-assisted where useful**  
   The compiler is responsible for deterministic parsing, validation, indexing, and derivation. AI assists with authoring, intent interpretation, prose improvement, and likely-but-not-provable issues.

4. **Context is a scarce resource**  
   Search and retrieval should prefer the smallest semantically useful section over whole-document loading, estimate context cost, and expand only when the task requires more evidence.

5. **Markdown-native navigation, durable internal identity**  
   Authors navigate with ordinary Markdown links, while documents keep opaque stable IDs in frontmatter so compiled artifacts can preserve durable identity across file moves and renames.

6. **Human-friendly explanation without duplicating truth**  
   Native-language content is a concise explanatory summary for people, not a required parallel copy of the canonical content.

7. **AI-first discovery, human-readable source**  
   Humans should usually find and understand information through AI-mediated search, answers, and impact analysis, while the underlying Markdown remains readable for review, fallback navigation, and trust.

8. **Feedback improves the corpus**  
   The system should use deterministic diagnostics and retrieval/query usage signals to recommend better document boundaries, links, templates, type definitions, and retrieval recipes over time.

## 5. Core Concepts

### 5.1 Document

A document is the smallest canonical unit of specification. Each detailed document covers one concern only.

Initial built-in document coverage in v1 includes:

- system
- component
- requirement / feature
- API
- database object
- UI screen or flow
- architectural decision
- glossary term

This list describes the initial product coverage, not a closed compiler taxonomy. Detailed built-in specializations, document-type assets, and user-defined type extension requirements are defined in [Extensible Document Type Framework — Product Requirements](./DOCUMENT_TYPE_REQUIREMENTS.md).

Each document must contain:

- an opaque stable ID in frontmatter,
- document type,
- title,
- optional aliases and tags,
- document-level declared facts,
- compact canonical sections,
- ordinary Markdown links to related documents in the body,
- optional native-language summary.

### 5.2 Declared fact

A declared fact is a machine-checkable claim embedded in a document. Examples include:

- ownership,
- lifecycle status,
- API method and route,
- database field name and type,
- component dependency,
- screen-to-API relation,
- requirement-to-component relation.

Declared facts are the basis for deterministic contradiction detection.

### 5.3 Link

A link is an ordinary Markdown link in the document body, for example `[Auth API](../api/auth.md)`. This keeps navigation natural for humans and lets agentic AI follow links without learning a custom syntax.

The compiler must verify that each target path resolves, map linked documents to their durable frontmatter IDs, generate reverse links automatically, and derive relation types from the surrounding section or table context.

### 5.4 Index view

An index view is a generated, curated overview over the canonical corpus, such as:

- component catalog,
- API map,
- feature matrix,
- database table index,
- UI navigation map,
- glossary index.

Indexes are compiler-managed views. Users may request edits from an index view, but the skill must update the canonical detailed documents and then regenerate the affected indexes.

### 5.5 Compiled knowledge base

The compiled knowledge base is the machine-readable representation derived from the Markdown corpus. It includes:

- document records,
- section records,
- fact records,
- link records,
- relation records,
- generated indexes,
- generated diagrams,
- search/query-ready tables.

### 5.6 Document set

A **document set** is a coherent folder-level group of canonical documents within a larger system corpus.

The product must support:

- a single document set for a monolith,
- multiple document sets for modular monoliths or microservice systems,
- one or more shared/common-information folders referenced by multiple document sets.

Document sets may represent bounded contexts, modules, services, or other team-defined partitions. The compiler must be able to compile each set in context while still producing a system-level knowledge graph across all configured sets and shared folders.

### 5.7 Asset layers

An **asset** is a reusable compiler or skill resource that shapes document authoring, validation, retrieval, feedback, or generated outputs. Assets include templates, document type definitions, retrieval recipes, context expansion rules, feedback diagnostic rules, and related references.

The product must distinguish between two asset layers:

1. **Skill asset bundle**  
   The installed skill ships read-only baseline assets. These built-in assets provide default document types, starter templates, context recipes, and feedback rules. Feedback workflows must never modify installed skill assets.

2. **Project-local assets**  
   Each project may define mutable local assets that override or extend the skill defaults. These assets are owned by the project, versioned with the project where appropriate, and are the only valid target for project-specific template, type-definition, retrieval-recipe, or feedback-rule updates.

Asset resolution must prefer project-local assets over skill defaults. If no project-local override exists, the compiler must fall back to the skill-provided baseline asset.

Project initialization should create the local configuration and asset-folder structure required for overrides, but it should not eagerly copy every built-in asset. A built-in asset should be materialized into the project only when the project customizes it or a workflow explicitly requests a local editable copy.

### 5.8 Project backlog

A **project backlog** is a project-local Markdown work list that tracks documentation tasks, implementation tasks, decisions, validation follow-ups, audit findings, and deferred gaps.

The backlog allows documents to grow incrementally with source code. Initial documents may be thin blueprints, while unfinished sections, unresolved design choices, and implementation follow-ups remain visible as actionable work rather than hidden as incomplete prose.

Backlog details for the initial project experience are defined in [Initial Workflow, Backlog, and First Implementation Experience — Product Requirements](./INITIAL_WORKFLOW_EXPERIENCE_REQUIREMENTS.md).

## 6. Authoring Model

### 6.1 Source format

The source format is Markdown with frontmatter.

Frontmatter must support document-level metadata and facts, at minimum:

```yaml
id: "opaque-stable-id"
type: "api"
title: "Refresh access token"
aliases:
  - "token refresh"
tags:
  - "auth"
native_summary: "アクセストークン更新 API の要約"
facts:
  lifecycle.status: "active"
  owner: "identity-platform"
```

Links are written in normal Markdown body content, for example:

```markdown
## Dependencies

- [Auth component](../components/auth.md)

## Reads

- [User sessions](../database/user-sessions.md)
```

The exact final fact schema may evolve, but the authoring model must preserve these capabilities:

- durable document identity,
- classification,
- search aliases,
- Markdown-native links,
- machine-checkable facts from both frontmatter and body sections.

### 6.2 Canonical section pattern

Each document should prefer compact, predictable sections so AI systems can retrieve them independently. A typical document may include:

- Summary
- Responsibilities / Scope
- Interfaces or Fields
- Rules / Constraints
- Dependencies / Reads / Invoked APIs / Related requirements
- Native-language summary

The system should not require identical headings for every document type if a specialized template is more useful, but every template must expose compact canonical sections that can be independently indexed.

Relation semantics should be inferred from the surrounding section or table context rather than encoded through custom link syntax. For example, a Markdown link inside `## Dependencies` becomes a dependency relation, while a link inside `## Reads` becomes a read relation.

### 6.3 Native-language summary

A document may include an optional concise explanation in a native language such as Japanese. This section exists to help human readers; it is not the canonical truth source and does not need to mirror every canonical section one-to-one.

### 6.4 Granularity

One detailed document should describe one concern. Large mixed documents are discouraged because they reduce retrieval quality, make contradictions harder to localize, and increase unnecessary context loading.

### 6.5 Common detail documents

Some facts intentionally apply to many documents of the same type or family. The system must support **common detail documents** that hold shared canonical facts and are linked from the documents that reuse them.

For example, a common ERD detail document may define system-wide audit columns such as `created_by`, `created_at`, `updated_by`, and `updated_at`; individual ERD or data-model documents can link to that common detail document rather than duplicate those fields.

Common detail documents are canonical source documents, not generated boilerplate. The compiler must preserve:

- the explicit links from specific documents to the shared detail document,
- relation records showing reuse or application of common detail,
- searchability of the common detail itself,
- validation and contradiction checks across inherited or referenced shared facts where the relation model supports them.

## 7. Skill User Experience

The system is delivered as a skill with Python-based scripts and supporting references/assets as needed.

The detailed workflow-level skill surface is defined in [Spec Compiler Skills — Product Requirements](./SKILL_WORKFLOW_REQUIREMENTS.md). The initial wizard and backlog-driven first implementation experience are defined in [Initial Workflow, Backlog, and First Implementation Experience — Product Requirements](./INITIAL_WORKFLOW_EXPERIENCE_REQUIREMENTS.md).

### 7.1 Required user workflows

The skill must support workflows for:

1. **Initialization and configuration**
   - initialize a project-local spec configuration,
   - create project-local asset folders for templates, type definitions, retrieval recipes, and feedback rules,
   - configure document sets, shared folders, and generated-output locations,
   - reference skill defaults without copying all built-in assets,
   - create or locate a project-local Markdown backlog.

2. **Initial wizard to first implementation**
   - inspect existing project material before asking questions,
   - classify the project entry point,
   - recommend modular monolith or microservice boundaries,
   - create thin blueprint documents,
   - select a first standalone runnable slice,
   - produce compact handoff context for the first implementation task.

3. **Backlog management**
   - track documentation, implementation, decision, validation, audit, and cleanup tasks,
   - use simple Kanban states for progress,
   - link tasks to documents, source paths, document sets, components, and implementation slices,
   - let humans and AI agents update progress.

4. **Authoring**
   - create a new document from the appropriate template,
   - update a detailed document,
   - add frontmatter facts, body-declared domain facts, Markdown links, aliases, and summaries,
   - create backlog tasks for intentionally deferred sections or unresolved decisions.

5. **Editing from an index view**
   - let a user express intent from a generated overview,
   - identify the affected canonical documents,
   - update those documents through AI assistance,
   - regenerate the index view.

6. **Compilation and validation**
   - parse the corpus,
   - validate metadata and declared facts,
   - emit compiled outputs,
   - report errors and warnings clearly,
   - allow unresolved findings to become backlog tasks when immediate repair is not required.

7. **Search, query, and context orchestration**
   - text-search relevant document slices,
   - graph-query relationships and impact paths,
   - return low-context results suitable for AI consumption,
   - expand context incrementally through typed relations under a token budget,
   - include active backlog items when they are relevant to the current task.

8. **Explanation and visualization**
   - explain a selected slice or relation set,
   - generate architecture and data-flow diagrams from compiled relations.

9. **Reference maintenance**
   - repair safe link/path drift automatically,
   - preserve durable identity across document moves and renames,
   - keep link captions aligned with the current target titles.

10. **AI-centric quality review**
   - run explicit, on-demand audits against implementation artifacts where deterministic compilation is insufficient,
   - report likely drift with evidence and confidence without treating AI findings as compiler errors,
   - create or propose backlog tasks for findings that are not fixed immediately.

11. **Corpus feedback**
   - analyze static corpus diagnostics and retrieval/query usage signals,
   - recommend improvements to document structure, links, summaries, aliases, templates, type definitions, and retrieval recipes,
   - apply safe structural repairs where policy allows,
   - propose semantic restructures for acceptance by default,
   - create or propose backlog tasks for recommendations that require later work.

### 7.2 Behavioral command surface

The final exact command syntax is intentionally unspecified in v1 requirements, but the skill must expose capabilities equivalent to:

- author / update
- init / configure
- initial wizard / first implementation
- backlog / progress
- customize asset
- compile / validate
- search / query
- context / pack
- visualize / explain
- repair / refactor
- audit
- feedback

### 7.3 AI-assisted behavior

AI assistance is expected for:

- interpreting natural-language edit intent,
- suggesting document updates,
- helping reconcile higher-level changes,
- generating or refining concise summaries,
- surfacing likely prose-level inconsistencies that are not deterministically provable.

The deterministic compiler remains the final authority for structural validation.

## 8. Compiler Requirements

### 8.1 Inputs

The compiler consumes:

- canonical Markdown specification documents,
- one or more configured document sets,
- optional shared/common-information folders referenced across document sets,
- document templates, document type definitions, retrieval recipes, and feedback rules from the resolved asset layers,
- optional local retrieval/query usage signals for feedback analysis,
- optional configuration defining repositories, generated-output paths, and validation rules.

### 8.2 Project configuration

Each project must be able to define a compiler configuration that declares:

- document sets representing systems, modules, bounded contexts, or services,
- shared/common-information folders visible to one or more document sets,
- skill asset bundle references and project-local asset folders containing templates, document-type definitions, retrieval recipes, and feedback rules,
- generated-output locations for local and system-level artifacts,
- enabled validations,
- repair policy for safe structural fixes versus semantic changes,
- feedback policy for recommendation-only, safe-repair, and explicit aggressive modes,
- optional implementation-source roots used by explicit AI audits.

The configuration format is not fixed by these requirements, but the compiler must treat configuration as the authoritative source for corpus boundaries and generated-output placement.

### 8.3 Required validation

The compiler must detect:

- missing required metadata,
- duplicate IDs,
- malformed Markdown where it prevents reliable parsing or link extraction,
- invalid or dangling Markdown links,
- missing backlink targets,
- document IDs referenced in prose where the context requires a Markdown link,
- malformed declared facts in frontmatter or body sections,
- contradictions among declared facts from frontmatter and body sections,
- stale generated outputs when applicable,
- invalid references across configured document sets or common-information folders.

### 8.4 Safe automatic repairs

The compiler must be able to apply safe structural repairs that preserve document meaning, including:

- converting resolvable bare document-ID references into ordinary Markdown links,
- using the target document title as the canonical link caption,
- updating stale or incorrect link captions after a target title changes,
- rewriting safe inbound link paths after a linked document is moved or renamed while preserving the target document ID.

Safe repairs may run automatically. Changes that alter semantics, introduce inferred facts, or reinterpret prose must be proposed to the user or calling agent rather than applied silently.

### 8.5 Declared-fact contradiction checks

v1 must support deterministic checks for conflicts such as:

- two documents claiming incompatible ownership for the same entity,
- duplicate or conflicting API method/route definitions,
- mismatched field names or types for the same database concept,
- incompatible lifecycle/status declarations,
- relations that contradict declared ownership or dependency rules,
- references to entities that do not exist.

The product may allow AI to suggest possible prose contradictions, but such suggestions are advisory and must not be confused with deterministic compiler errors.

### 8.6 Explicit AI audits

The product must provide explicit, user-invoked AI audits for quality checks that cannot be fully proven through deterministic compilation, including:

- API specifications versus current implementation,
- authored or generated DFD/data-flow models versus current implementation,
- component responsibility claims versus current code behavior,
- likely missing or stale cross-document relationships.

AI audits may consume substantial tokens and may inspect implementation sources configured for the project. Audit results must:

- remain separate from deterministic compiler errors and warnings,
- include supporting evidence and confidence where practical,
- identify the affected specs and implementation areas,
- never silently rewrite canonical document semantics.

### 8.7 Required derived outputs

The compiler must derive:

- reverse links / backlinks,
- curated index documents,
- section-level search records,
- document/entity/fact/relation tables,
- Mermaid diagrams,
- machine-readable artifacts suitable for DuckDB querying.

## 9. Compiled Data Model

### 9.1 JSONL artifacts

The compiler should emit JSONL records for at least:

- documents,
- sections,
- facts,
- links,
- relations,
- document-set / module membership.

These artifacts should be easy to inspect, stream, version, and load into other tools.

For a multi-set corpus, the compiler must emit:

- local JSONL artifacts for each configured document set, and
- a global merged artifact set spanning all configured document sets plus shared/common-information folders.

### 9.2 DuckDB support

The system should provide a DuckDB-compatible schema or load path over the generated JSONL so users can perform flexible local queries without a custom database server.

The queryable model must support questions such as:

- which APIs depend on this component?
- which screens invoke this API?
- which specs are impacted by this database table?
- which requirements are linked to this architectural decision?
- which entities have conflicting ownership or lifecycle declarations?

### 9.3 Search records

Search records should be section-oriented rather than document-only. Each searchable unit should expose enough metadata to support ranked retrieval:

- document ID,
- title,
- aliases,
- type,
- section name,
- compact text,
- declared facts,
- nearby links / relations,
- tags,
- estimated token cost,
- section role for retrieval,
- retrieval priority where declared or derived,
- recommended relation types for context expansion.

## 10. Search and Navigation

### 10.1 Text search

v1 must support deterministic text search over indexed document sections.

Text search should:

- rank compact relevant slices,
- include document identity and local context,
- avoid forcing whole-document retrieval when a smaller section is sufficient,
- support aliases and tags.

This is the baseline retrieval mode for RAG-like AI workflows.

### 10.2 Graph query

v1 must support relation-oriented queries over compiled links and facts.

Graph-style queries should make it easy to answer:

- dependency questions,
- impact-analysis questions,
- ownership questions,
- traceability questions across requirement, component, API, database, and UI layers.

### 10.3 Future semantic retrieval

Embedding-based or semantic search may be added later, but it is not required for v1. The baseline system must remain useful through deterministic text search and graph queries alone.

### 10.4 Context orchestration

The product must support AI-oriented context orchestration rather than only returning independent search hits.

Context orchestration should:

- accept a task description, question, or starting entity,
- begin with the smallest high-confidence section slices,
- estimate and respect an explicit or default token budget,
- expand through typed graph relations only when additional evidence is useful,
- prefer task-specific context recipes where available,
- explain why each selected slice was included,
- report useful omitted context when the budget prevents inclusion,
- stop when the question is answerable or when no useful budget-respecting expansion remains.

Task-specific context recipes should describe the preferred slice types and relation-expansion order for common work such as feature implementation, API changes, data-model changes, impact analysis, and drift audits.

AI-facing answers produced from orchestrated context should identify the supporting source slices and distinguish direct evidence from AI inference. If the compiled corpus does not provide enough evidence, the system should report insufficient support instead of presenting an unsupported answer as certain.

### 10.5 Corpus feedback loop

The product must provide a lightweight v1 feedback loop that improves the corpus as an AI-consumable context system.

The feedback loop should combine:

- **static corpus diagnostics**, such as oversized sections, mixed-concern documents, weakly connected documents, missing or low-quality links, high-hop answer paths, duplicate reusable details, and stale generated outputs;
- **retrieval/query usage signals**, such as selected slices, graph expansion paths, estimated context cost, repeated co-retrieval, answer sufficiency, missing-context reports, and frequently traversed links.

Feedback outputs must include both a human-readable report and machine-readable findings. Each finding should include:

- affected document or project-local asset path,
- recommendation type,
- reason or evidence,
- expected context-efficiency or answerability benefit,
- action classification: safe automatic repair, proposed semantic change, or explicit aggressive-mode candidate,
- stable machine-readable recommendation ID for apply, dismiss, or review workflows.

Feedback may recommend improvements such as splitting oversized sections, adding missing relations, improving aliases or tags, promoting repeated content into common detail documents, updating project-local templates, updating project-local document type retrieval hints, or adjusting context recipes.

Feedback workflows must target project-local assets when changing templates, type definitions, retrieval recipes, or feedback rules. Installed skill assets are read-only defaults and must never be modified by feedback.

By default, the feedback loop may apply only safe structural repairs, such as stale caption refreshes or resolvable link repairs. Meaning-changing edits, document splits, new semantic links, or asset-behavior changes must be proposed for acceptance. An explicit aggressive mode may allow higher-confidence restructuring actions, but it must be opt-in and clearly reported.

## 11. Visualization

The compiler must be able to generate Mermaid diagrams from declared relations.

First-class diagram families in v1:

1. **C4-style architecture views**
   - system context,
   - container/component views where enough declared relations exist.

2. **ERD**
   - database entities,
   - fields,
   - relationships.

3. **DFD**
   - actors,
   - processes,
   - stores,
   - data flows.

4. **Related dependency/flow views**
   - API-to-component,
   - UI-to-API,
   - requirement-to-implementation traceability where the declared relation model supports it.

The system should generate diagrams from the same compiled relation data used by search and validation, not from separate hand-maintained diagram truth.

## 12. Editing Model

### 12.1 Detailed-document editing

When a user updates a detailed document:

1. the skill helps edit the canonical source file,
2. the compiler validates the result,
3. affected indexes, backlinks, search records, and diagrams are regenerated.

### 12.2 Index-driven editing

When a user asks for a change from an index view:

1. the skill interprets the requested intent,
2. identifies affected canonical documents,
3. updates or creates detailed documents as needed,
4. recompiles the corpus,
5. regenerates the index view.

The user experiences bidirectional editing, but the persisted truth remains in detailed documents.

### 12.3 Conflict handling

The system must not silently preserve inconsistent compiled state. If an AI-assisted change cannot be made without violating deterministic rules, the compiler must report the issue and require reconciliation before accepting the corpus as valid.

### 12.4 Refactor-safe document moves

When a document is moved or renamed while keeping the same durable ID:

1. the compiler or skill must preserve the stable identity,
2. safe inbound Markdown links must be updated to the new path,
3. stale captions must be refreshed from the current title,
4. compiled artifacts must continue to represent the same entity rather than treating the move as deletion plus recreation.

## 13. Acceptance Scenarios

### 13.1 Linked API document

Given a new API document with `[Auth component](../components/auth.md)` inside `Dependencies` and `[User sessions](../database/user-sessions.md)` inside `Reads`, when the corpus is compiled, then the system must:

- validate both Markdown link paths,
- map linked files to their durable frontmatter IDs,
- create reverse links,
- emit searchable section records,
- emit typed relation records connecting the API, component, and database objects.

### 13.2 Index-driven component update

Given a user request from a component index view to add or reorganize a component, when the skill processes the request, then it must:

- update or create the relevant canonical detail documents,
- regenerate the component index,
- keep generated artifacts consistent with the updated detail docs.

### 13.3 Declared-fact contradiction

Given two documents that declare incompatible facts for the same API route or database field, whether those facts come from frontmatter or structured body sections, when compilation runs, then it must report a deterministic contradiction with enough information for a user or AI agent to resolve it.

### 13.4 Minimal-context retrieval

Given an AI agent asking for the minimum relevant context for a feature, when search runs, then it should return ranked slices with IDs, titles, facts, and nearby links instead of loading unrelated full documents.

### 13.5 Generated diagrams

Given a sufficiently linked system corpus, when visualization runs, then the system must emit Mermaid diagrams for C4-style architecture, ERD, and DFD views from declared relations.

### 13.6 Invalid corpus detection

Given broken Markdown links, missing IDs, duplicate IDs, or stale generated outputs, when validation runs, then the system must reject or warn on the invalid corpus before it is treated as accepted state.

### 13.7 Markdown-native navigation

Given a document containing ordinary Markdown links, when a human or agentic AI reads the file directly, then it must be able to follow those links without understanding any product-specific link syntax.

### 13.8 Durable identity through renames

Given a document that is moved or renamed while keeping the same frontmatter ID, when the corpus is recompiled, then generated artifacts must preserve the same durable document identity even though the Markdown link paths are updated.

### 13.9 Multi-set system corpus

Given a modular monolith or microservice system with multiple configured document sets plus a shared common-information folder, when the corpus is compiled, then the system must:

- validate each configured document set,
- resolve links across set boundaries and into the common-information folder,
- emit one system-level compiled knowledge base that preserves document-set membership,
- support cross-set search, graph queries, and generated diagrams.

### 13.10 Common ERD detail reuse

Given a common ERD detail document that defines shared audit columns and multiple data-model documents that link to it, when compilation runs, then the system must:

- validate the links to the common detail document,
- emit relation records showing reuse of the shared detail,
- make the shared columns discoverable through search and graph queries,
- allow contradiction checks to account for the referenced common facts without requiring those fields to be duplicated in every linked document.

### 13.11 Bare ID repair

Given prose containing a resolvable document ID where the document-type rules require a link, when safe repair runs, then the compiler must replace that reference with an ordinary Markdown link whose caption matches the target document title.

### 13.12 Stale caption repair

Given a valid Markdown link whose target document title has changed, when safe repair runs, then the compiler must update the link caption to the current target title without changing the relation target.

### 13.13 Local and global indexes

Given a corpus with multiple configured services or modules, when compilation runs, then the system must emit both per-set JSONL artifacts and a merged global artifact set that preserves set membership while supporting cross-set queries.

### 13.14 AI drift audit

Given an API, component, or data-flow specification and relevant implementation sources, when a user explicitly requests an AI audit, then the system must report likely mismatches with evidence while keeping those findings separate from deterministic validation results and without silently mutating canonical documents.

### 13.15 Project-local asset initialization

Given a new project initialization, when the skill configures the spec corpus, then it must create project-local configuration and asset folders while preserving installed skill assets as read-only defaults and without eagerly copying every built-in template or type definition.

### 13.16 Project-local asset override

Given a project-local template or type definition with the same identity as a skill-provided asset, when the compiler resolves assets, then it must use the project-local override and fall back to the skill asset only when no local override exists.

### 13.17 Feedback targets local assets

Given corpus feedback that recommends changing an API template, document type retrieval hint, or context recipe, when the change is applied or proposed, then the target must be the project-local asset folder rather than the installed skill asset bundle.

### 13.18 Feedback never mutates skill assets

Given any feedback workflow, including aggressive mode, when recommendations are generated or applied, then installed skill-provided assets must remain unchanged.

### 13.19 Conservative feedback action

Given stale link captions and resolvable bare IDs, when feedback or repair runs under the default policy, then safe structural fixes may be applied automatically; document splits, semantic link additions, and template behavior changes must be proposed for acceptance.

### 13.20 Explicit aggressive feedback mode

Given a user explicitly enables aggressive feedback mode, when the system identifies high-confidence restructuring opportunities, then it may prepare or apply those changes according to policy, while clearly reporting the evidence, affected files, and non-default action mode.

### 13.21 Customized type participation

Given a customized project-local document type asset, when compilation runs, then matching documents must still participate in validation, search, graph queries, feedback diagnostics, and context orchestration according to the merged resolved type behavior.

## 14. Success Metrics

The product is successful when:

- AI agents can answer common system-understanding questions using a small subset of the corpus,
- AI agents can load context incrementally under a budget and cite the slices that support an answer,
- structural documentation errors are found before human review rather than after drift accumulates,
- teams can navigate from one concern to connected concerns without manually maintaining wiki backlinks,
- regenerated indexes remain aligned with canonical documents,
- diagrams are derived from source facts rather than maintained separately,
- native-language summaries improve human readability without increasing canonical duplication,
- feedback reports reduce oversized sections, missing links, repeated co-retrieval without explicit relations, and high-token answer paths over time.

## 15. Assumptions and Defaults

1. The primary audience is AI-assisted software teams, not autonomous agents alone.
2. Large-scale software development is a first-class use case from the beginning.
3. AI assistance is part of the user experience, but deterministic local artifacts remain essential.
4. Markdown with frontmatter is the source format.
5. Detailed documents are canonical persisted truth.
6. Index documents are generated views that may serve as natural edit-entry surfaces.
7. Authors navigate with ordinary Markdown links, while opaque stable IDs remain mandatory durable internal identity.
8. Document-level metadata and global facts live in frontmatter; domain-specific structured facts may live in readable body sections.
9. Relation semantics are inferred from surrounding Markdown structure rather than custom link syntax in v1.
10. Native-language content is a concise human-facing summary, not a full translation layer.
11. v1 optimizes for context efficiency, structural consistency, and navigability over full semantic understanding of arbitrary prose.
12. Deterministic text search plus graph query are required in v1; semantic search is optional future work.
13. Supported target systems include monoliths, modular monoliths, and microservice systems.
14. A corpus may contain multiple document sets plus shared/common-information folders.
15. Shared reusable facts may live in linked common detail documents instead of being duplicated across all specific documents.
16. Safe structural repairs may be automatic; semantic changes require explicit acceptance.
17. Explicit AI audits are on-demand workflows, not part of routine deterministic validation.
18. Multi-set corpora emit both local and global compiled artifacts.
19. Skill-provided assets are immutable defaults; project-local assets are the mutable customization layer.
20. Project initialization creates local configuration and override locations without copying all built-in assets by default.
21. Skill upgrades must not overwrite project-local customizations.
22. The default feedback action model is conservative: recommend improvements and apply only safe structural repairs unless an explicit aggressive mode is requested.
