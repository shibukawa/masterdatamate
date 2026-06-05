# Initial Workflow, Backlog, and First Implementation Experience — Product Requirements

## 1. Purpose

The initial project experience must help users reach the first useful implementation without forcing a long up-front documentation phase. Many projects begin from uncertain material: old legacy code, incomplete proof-of-concept work, a new idea, an existing document set, a change request, or inspiration from another project. The workflow must identify what is useful, avoid committing to unnecessary structure, and create only enough specification to start safely.

The product should produce thin blueprint documents and an explicit project backlog before first implementation. Documentation then grows with the source code as decisions are proven, changed, or deferred.

This document defines the expected experience from initial wizard through the first standalone runnable implementation unit. It complements the core product requirements in [AI-Friendly Spec Compiler — Product Requirements](./REQUIREMENTS.md) and the workflow surface in [Spec Compiler Skills — Product Requirements](./SKILL_WORKFLOW_REQUIREMENTS.md).

## 2. Goals

1. Reduce initial process overhead before implementation begins.
2. Support multiple real project entry points without forcing every project through the same checklist.
3. Create only necessary blueprint documents before coding.
4. Make incomplete documentation visible through backlog-managed work instead of pretending it is complete.
5. Help humans and AI agents jointly manage progress across documentation, implementation, decisions, validation findings, and deferred gaps.
6. Select a first implementation target that can run and be tested locally with minimal dependencies.

## 3. Product Principles

1. **Minimal gates before implementation**  
   The wizard should ask only for decisions that materially affect the first implementation handoff.

2. **Blueprint first, complete docs later**  
   Initial documents may be intentionally thin. They should define direction, boundaries, and the first important design, not the full future system.

3. **Documentation and code evolve together**  
   Implementation work should update related documents when it proves, invalidates, or clarifies a design decision.

4. **Unfinished work is explicit**  
   Missing sections, unresolved decisions, follow-up audits, and known implementation gaps should be tracked in the backlog rather than hidden inside vague prose.

5. **First implementation works standalone**  
   The first implementation unit should be runnable and testable locally. It may use mocks, stubs, fixtures, or local-only adapters for dependencies that are not yet implemented.

6. **Modular monolith first by default**  
   The workflow should prefer modular monolith boundaries unless independent deployment, ownership, scaling, runtime isolation, or integration constraints justify microservices.

## 4. Core Concepts

### 4.1 Initial wizard

The initial wizard is a guided workflow that inspects project material, classifies the starting point, asks minimal high-impact questions, and prepares the first implementation handoff.

The wizard should combine deterministic project inspection with AI-assisted interpretation where judgment is required. It must not require a complete specification corpus before the first implementation task can begin.

### 4.2 Blueprint document

A blueprint document is an intentionally incomplete canonical specification document that is sufficient to guide early work. It records current intent, boundaries, known facts, and open gaps.

Blueprint documents are not second-class documents. They should compile where their known structure is valid, but the compiler and workflows may report explicit warnings or backlog-linked gaps for missing details.

### 4.3 Markdown backlog

A Markdown backlog is a project-local, human-readable work list that tracks documentation and implementation progress. It is part of the project source, can be edited by humans or agents, and can be linked from specification documents.

The backlog must support simple Kanban states:

- `backlog`
- `ready`
- `in progress`
- `blocked`
- `review`
- `done`

### 4.4 First runnable slice

The first runnable slice is the smallest useful implementation unit that proves the core idea or highest-risk assumption. It should cross enough boundaries to validate the development direction, but remain small enough to implement before the full system specification is complete.

The slice may be a vertical feature, a standalone module with a local interface, or a service-like unit when microservice constraints are justified.

## 5. Initial Wizard Flow

### 5.1 Project inspection

The wizard should begin by inspecting available project material before asking the user questions. Useful signals include:

- existing source directories, package manifests, build scripts, and test files;
- existing Markdown documents, design notes, issue lists, diagrams, and generated artifacts;
- repository shape, module boundaries, and likely entry points;
- evidence of legacy, prototype, generated, or abandoned code;
- user-provided intent or project description.

Inspection must be non-destructive. The wizard should summarize relevant evidence and mark low-confidence inferences instead of presenting them as facts.

### 5.2 Entry point classification

The wizard must classify the project into one primary entry point and may record secondary traits.

Supported entry points are:

- **Legacy source code**: an existing codebase has useful behavior but may contain stale, risky, or unnecessary implementation details.
- **Incomplete POC**: a prototype demonstrates intent or feasibility but should not define the final architecture by default.
- **New project from scratch**: the user has an idea or requirements but little or no source material.
- **Inspired by another project**: an external project, product, or codebase informs the direction, but the workflow must capture differentiators instead of copying unnecessary structure.
- **Docs-only or spec-only project**: existing documents contain useful intent before implementation exists.
- **Feature or change request in an active codebase**: the project already exists and the workflow should focus on the affected area.
- **Migration or rewrite**: existing behavior or contracts should be preserved while implementation changes.

### 5.3 Minimal questions

After inspection, the wizard should ask only unresolved questions that affect one of:

- project partitioning;
- first runnable slice selection;
- source areas to inherit or ignore;
- immediate user-visible behavior;
- local run and test expectations;
- blockers that would prevent first implementation.

The wizard should not ask for complete architecture, complete data models, exhaustive API lists, or final document taxonomy before first implementation unless the project context makes those choices unavoidable.

### 5.4 Architecture shaping

The wizard must recommend an initial architecture shape before selecting the first implementation target.

The default recommendation is a modular monolith with explicit module boundaries. Microservices should be recommended only when one or more of these conditions is clear:

- modules require independent deployment cadence;
- teams or owners need strong operational separation;
- runtime scaling or availability requirements differ substantially;
- external integration boundaries already force service separation;
- security or compliance constraints require isolation.

When microservices are not justified, the wizard should still preserve module boundaries so future extraction remains possible.

### 5.5 First implementation target selection

The wizard should select or recommend a first runnable slice that:

- proves the core product idea or highest-risk assumption;
- can run locally with minimal dependencies;
- can be tested with deterministic checks, fixtures, mocks, or stubs;
- has clear component or module boundaries;
- has enough blueprint documentation for an AI coding agent to begin;
- creates follow-up backlog items for deferred documentation or implementation work.

## 6. Entry Point Behavior

### 6.1 Legacy source code

For legacy code, the wizard should identify source areas to inherit, source areas to ignore, and source areas requiring audit before reuse. It should avoid treating the existing structure as authoritative when the code appears stale, inconsistent, or unrelated to the new goal.

The first runnable slice should reuse legacy code only where the value is clear enough for early implementation.

### 6.2 Incomplete POC

For an incomplete POC, the wizard should preserve useful behavior, domain discoveries, and working examples while separating them from throwaway architecture, shortcuts, or incomplete abstractions.

Cleanup, redesign, and verification work should become backlog tasks instead of blocking the first slice unless they are required for local execution.

### 6.3 New project from scratch

For a scratch project, the wizard should create a project overview blueprint, architecture partitioning note, and first slice design. It should avoid generating a complete corpus before implementation begins.

### 6.4 Inspired by another project

For an inspiration-based project, the wizard should capture:

- which behaviors or qualities are being inherited as intent;
- how the new project differs;
- which external structures should not be copied;
- any licensing, ownership, or dependency concerns that should become backlog items.

### 6.5 Docs-only or spec-only project

For docs-only projects, the wizard should convert useful existing material into thin canonical blueprints and backlog tasks. It should mark unclear, stale, or conflicting parts as unresolved rather than forcing premature normalization.

### 6.6 Feature or change request in an active codebase

For an active codebase change, the wizard should start from the requested behavior and affected implementation area. It should create focused documentation tasks only for the documents needed to understand, implement, validate, and review the change.

### 6.7 Migration or rewrite

For a migration or rewrite, the wizard should identify preserved behavior, compatibility constraints, replacement boundaries, and the first safe replacement slice. Legacy behavior contracts should be documented before implementation details are rewritten.

## 7. Backlog Management

### 7.1 Backlog creation

Initialization should create or identify a project-local Markdown backlog. The exact file path is configurable, but the wizard output must report the path it will use.

The backlog should be compact enough for humans to read and structured enough for scripts or agents to parse predictably.

### 7.2 Backlog item fields

Each backlog item should support:

- stable item ID;
- title;
- state;
- type, such as documentation, implementation, decision, validation, audit, cleanup, or release;
- owner or responsible agent when known;
- linked specification documents;
- linked source paths when known;
- related first slice, component, module, or document set;
- acceptance notes;
- blockers;
- last updated date or event when available.

The final serialized syntax is deferred, but the Markdown representation must remain easy to review and edit directly.

### 7.3 Backlog states

Backlog states have the following meanings:

- `backlog`: useful work that is not yet ready or scheduled;
- `ready`: sufficiently understood to start;
- `in progress`: actively being worked by a human or agent;
- `blocked`: cannot continue until a named blocker is resolved;
- `review`: implementation or document changes are ready for human or agent review;
- `done`: accepted and no longer active.

### 7.4 Documentation work tracking

Document creation and update work should be represented as backlog items when it is not completed immediately. WIP documents may link to backlog items for:

- missing sections;
- unresolved facts;
- design decisions;
- validation warnings;
- known drift against implementation;
- follow-up examples, diagrams, or native-language summaries.

The product must not require every document to be complete before coding begins.

### 7.5 Human and agent updates

Humans and agents may both update backlog state. Agent updates should preserve user-authored context and should record enough evidence to explain why a task moved state, became blocked, or was marked done.

When an implementation task is completed, the agent should update or propose updates to related documents and move associated documentation tasks to `review` or `done` only when the acceptance notes are satisfied.

## 8. Blueprint Outputs

The initial wizard should create or propose the following outputs:

- project overview blueprint;
- architecture partitioning note;
- first component or vertical-slice design;
- initial document set configuration;
- first implementation handoff context;
- Markdown backlog with deferred documentation and implementation tasks;
- source inheritance notes for existing code, POC material, or inspiration sources;
- open decisions and blockers.

Blueprint outputs should be sufficient for the first implementation task, not exhaustive for the whole project.

## 9. Development Experience

### 9.1 First implementation handoff

The handoff to an AI coding agent should include:

- selected first runnable slice;
- relevant blueprint slices;
- architecture recommendation;
- local run and test expectations;
- active backlog items;
- source areas to reuse, inspect, or ignore;
- unresolved decisions that must not be guessed.

### 9.2 During implementation

Implementation may proceed while some documents remain WIP. The agent should keep related backlog items current and should not silently treat missing documentation as completed design.

If implementation discovers a new decision, dependency, source constraint, or documentation gap, the agent should update the relevant blueprint or create a backlog item.

### 9.3 After implementation

After the first runnable slice works, the workflow should:

- update affected blueprint documents with proven behavior and changed decisions;
- move implementation tasks to `review` or `done` according to acceptance notes;
- create or update documentation follow-up tasks for remaining gaps;
- run validation or drift checks where practical;
- treat validation or drift findings as actionable backlog items when immediate repair is not required.

## 10. Workflow Integration

### 10.1 Initial wizard to first implementation

The skill surface should expose a workflow equivalent to initial wizard to first implementation. The workflow output should include:

- detected project entry point;
- architecture recommendation;
- selected first runnable slice;
- initial blueprint document list;
- inherited, ignored, or uncertain source areas;
- Markdown backlog path;
- open decisions, blockers, and deferred tasks;
- first implementation handoff context.

### 10.2 Manage project backlog

The skill surface should expose a workflow equivalent to managing the project backlog. The workflow should support:

- creating or locating the Markdown backlog;
- listing active tasks by state, type, owner, document, component, or source area;
- creating backlog items from user intent, validation findings, drift findings, or implementation discoveries;
- moving items between Kanban states;
- linking tasks to documents and source paths;
- summarizing progress for human review.

### 10.3 Related workflows

The following workflows should interact with the backlog:

- initialization should create or locate the backlog;
- document creation should create follow-up tasks for intentionally deferred sections;
- context orchestration should include active backlog items relevant to the current task;
- drift audit should create or propose tasks for findings that are not immediately fixed;
- corpus feedback should create or propose backlog tasks for recommendations that require later human or agent work.

## 11. Acceptance Scenarios

### 11.1 Scratch project

Given a user starts from a new idea, when the initial wizard runs, then it creates minimal blueprints, a Markdown backlog, and a first runnable slice target without requiring a complete system specification.

### 11.2 Legacy project

Given an old source tree, when the initial wizard runs, then it separates reusable modules from stale or irrelevant code and records follow-up audit or cleanup tasks in the backlog.

### 11.3 Incomplete POC

Given an incomplete proof of concept, when the initial wizard runs, then it preserves useful behavior while backlog-tracking cleanup, redesign, and verification work.

### 11.4 Architecture default

Given weak justification for independent services, when the wizard recommends an architecture, then it defaults to modular monolith boundaries rather than microservices.

### 11.5 First implementation handoff

Given initial blueprints and a selected slice, when a coding agent requests context, then the workflow returns enough relevant context and active backlog items to implement locally without a complete spec corpus.

### 11.6 WIP documents

Given a blueprint document with known missing sections, when validation or compilation runs, then the system reports the WIP gaps clearly and does not treat the missing sections as completed design.

### 11.7 Agent progress update

Given an agent completes a code task, when it updates progress, then it updates linked documents where needed and moves related backlog items to `review` or `done` according to the acceptance notes.

### 11.8 Findings become tasks

Given validation, feedback, or drift audit findings that are not fixed immediately, when the workflow records them, then it creates backlog tasks instead of forcing immediate document rewrites.

## 12. Assumptions and Defaults

1. The initial workflow is a product and workflow requirement, not a final command syntax specification.
2. The backlog is Markdown-first in v1.
3. Backlog scope includes both documentation and implementation work.
4. Progress uses simple Kanban states.
5. Documentation starts as blueprints and becomes more complete through implementation feedback.
6. Built-in document type assets and templates remain immutable defaults; backlog-driven customization targets project-local assets.
