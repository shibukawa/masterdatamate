---
id: "agent-tool-contract"
type: "server-component"
title: "Agent tool contract"
aliases: ["AI tool contract", "LLM tool contract"]
tags: ["ai", "agent", "tools", "llm"]
facts:
  lifecycle.status: "blueprint"
  owner: "application"
---

# Agent tool contract

## Summary

The agent tool contract defines the application-owned tools that an AI assistant may use to inspect, analyze, propose, and execute MasterDataMate operations. Tools expose business operations, not raw filesystem access.

The same tool implementation should be reusable by the in-app AI assistant, command-backed local providers such as Codex CLI and Foundation Models `fm`, and by a future MCP server. The entry transport may differ, but validation, authorization, confirmation, and application service calls remain shared.

## Responsibilities

- Define tool schemas with explicit request and response shapes.
- Mark each tool as read-only, proposal-only, or side-effecting.
- Scope tool access to declared workspace, table, generation, and export contexts.
- Normalize tool results into assistant-readable summaries and machine-readable payloads.
- Validate proposed records, schema changes, generation operations, and export operations before execution.
- Require confirmation tokens for side-effecting tools.
- Keep all writes inside existing application services.
- Make tool results auditable enough for users to understand what the assistant did.

## Tool Categories

| Category | Examples | Confirmation | Notes |
| --- | --- | --- | --- |
| Discovery | `list_tables`, `list_generations`, `get_export_backends` | no | Returns available project structure and capabilities. |
| Read | `get_table_schema`, `read_generation_records`, `read_generation_config`, `read_export_settings` | no | Must be scoped and may enforce row or table limits. |
| Analysis | `validate_table`, `analyze_generations`, `compare_generations`, `explain_validation_errors` | no | Does not write files. |
| Proposal | `propose_record_changes`, `propose_schema_changes`, `propose_export_settings` | no | Returns pending changes and diagnostics, not writes. |
| Execution | `commit_record_changes`, `save_schema_changes`, `persistent_merge_generations`, `delete_generations`, `duplicate_generations`, `export_generation` | yes | Writes canonical files or export artifacts. |
| Frontend control | `focus_table`, `open_generation_editor`, `open_file_picker` | no, unless paired with a write | Runs in the browser through the AI panel UI to navigate or request user input. |

## Initial Tool Set

| Tool | Kind | Purpose |
| --- | --- | --- |
| `list_tables` | discovery | Return table identifiers and summaries from loaded schemas. |
| `get_table_schema` | read | Return one canonical table schema and UI-relevant field summary. |
| `read_generation_records` | read | Return scoped records for one table and generation view mode. |
| `list_generations` | discovery | Return generation IDs, labels, output flags, and ordering metadata. |
| `validate_table` | analysis | Validate one table's current or proposed records. |
| `explain_validation_errors` | analysis | Convert validation diagnostics into user-facing explanation and possible repair direction. |
| `propose_record_changes` | proposal | Return insert, update, delete, or move operations without writing YAML. |
| `commit_record_changes` | execution | Commit validated or explicitly forced record operations through the table commit service. |
| `analyze_generations` | analysis | Return counts, diagnostics, and optional merge-impact data for selected generations. |
| `compare_generations` | analysis | Return record-level differences across selected generations. |
| `persistent_merge_generations` | execution | Create a new generation by merging selected sources. |
| `export_generation` | execution | Run a selected export backend for selected generations or merged data. |

## Recommended Tool Catalog

The tool catalog is implemented in phases. Each tool must use structured parameters, return JSON-serializable results, and map to the same application services used by ordinary UI actions.

### Phase 1: Read And Explain

| Tool | Kind | Purpose |
| --- | --- | --- |
| `get_current_context` | read | Return current route, selected table, selected records, active generation, display mode, pending edit status, and visible diagnostics. |
| `list_tables` | discovery | Return table identifiers, comments, export flags, primary keys, and visibility metadata. |
| `get_table_schema` | read | Return one canonical table schema with fields, types, primary keys, references, defaults, export flags, and comments. |
| `list_generations` | discovery | Return generation IDs, labels, output flags, descriptions, sort order, and active generation metadata. |
| `read_generation_records` | read | Return scoped records for one table and generation view mode with row limits and optional field projection. |
| `validate_records` | analysis | Validate current records or a proposed record set without writing YAML. |
| `explain_validation_errors` | analysis | Explain diagnostics and suggest repair directions without producing a write operation. |
| `summarize_table` | analysis | Summarize table purpose, record count, key fields, obvious data quality issues, and export relevance. |

### Phase 2: Record Proposals And Commits

| Tool | Kind | Purpose |
| --- | --- | --- |
| `propose_record_changes` | proposal | Produce canonical insert, update, delete, or move operations for one table without writing YAML. |
| `preview_record_changes` | analysis | Return a diff-style preview, affected keys, diagnostics, and confirmation summary for a proposed record change set. |
| `commit_record_changes` | execution | Commit a confirmed record change set through the table commit service. |
| `find_suspicious_records` | analysis | Find missing values, unresolved references, duplicate-looking display names, suspicious numeric outliers, or inconsistent enum usage. |

### Phase 3: Schema Tools

| Tool | Kind | Purpose |
| --- | --- | --- |
| `list_schema_summaries` | read | Return schema list data used by schema editing. |
| `get_schema_detail` | read | Return a UI-friendly schema detail representation for one table. |
| `propose_schema_changes` | proposal | Propose field additions, comments, defaults, export flags, or type adjustments without writing YAML. |
| `preview_schema_changes` | analysis | Return affected schema files, affected table data files, validation impact, and confirmation summary. |
| `save_schema_changes` | execution | Save confirmed schema changes through schema save services. |
| `rename_table_schema` | execution | Rename a table schema and update references and matching data filenames after confirmation. |
| `rename_schema_field` | execution | Rename a schema field and matching YAML keys after confirmation. |
| `delete_schema_field` | execution | Delete a schema field and clean existing table data after confirmation. |

### Phase 4: Generation Tools

| Tool | Kind | Purpose |
| --- | --- | --- |
| `read_generation_config` | read | Return one generation `_config.yaml` as structured metadata. |
| `propose_generation_changes` | proposal | Propose output flag, description, or ordering metadata changes without writing YAML. |
| `save_generation_config` | execution | Save confirmed generation metadata changes. |
| `compare_generations` | analysis | Return record-level differences and override explanations across selected generations. |
| `analyze_generations` | analysis | Return counts, diagnostics, output states, and optional merge-impact counts. |
| `persistent_merge_generations` | execution | Create a new generation folder by merging selected source generations after confirmation. |
| `duplicate_generations` | execution | Duplicate selected generations through the host duplication service. |
| `delete_generations` | execution | Delete selected generation folders after strong confirmation. |

### Phase 5: Export Tools

| Tool | Kind | Purpose |
| --- | --- | --- |
| `list_export_backends` | discovery | Return available export backend IDs, names, supported formats, and required settings. |
| `read_export_settings` | read | Return current export settings. |
| `propose_export_settings` | proposal | Propose export settings from schemas, selected generations, and target runtime needs. |
| `validate_export` | analysis | Return export-blocking diagnostics and destination risks without writing artifacts. |
| `export_generation` | execution | Generate export artifacts after confirmation, reporting destination paths and overwrite behavior. |

### Phase 6: Upload And Import Tools

| Tool | Kind | Purpose |
| --- | --- | --- |
| `list_uploaded_files` | read | Return files staged in the current AI/import session. |
| `inspect_uploaded_file` | analysis | Parse metadata, file type, row count, columns, sheet names, sample rows, and parse diagnostics. |
| `propose_import_mapping` | proposal | Map uploaded CSV, TSV, XLSX, YAML, JSON, or binary files to target tables and fields. |
| `propose_records_from_file` | proposal | Produce canonical records or binary asset references from uploaded files without saving. |
| `commit_imported_records` | execution | Commit confirmed imported records through normal table commit services. |
| `attach_binary_asset` | execution | Add a confirmed binary file to the asset store and return canonical metadata for a `binary_file` field. |

### Phase 7: Spec And Help Tools

| Tool | Kind | Purpose |
| --- | --- | --- |
| `search_specs` | read | Search compiled specification sections relevant to the user's request. |
| `get_spec_context` | read | Return selected specification sections, relations, and backlog items for grounded assistant answers. |
| `explain_feature_behavior` | analysis | Explain current product behavior using specs and current workspace context. |

## Tool Schema Rules

- Every tool request includes a `workspaceContextId` or equivalent host-issued context identifier; the assistant must not pass arbitrary filesystem roots.
- Table and generation IDs must be canonical identifiers loaded by the host.
- Record changes use canonical primary-key values and schema field names.
- Proposal responses include a stable `proposalId`, affected tables, operation list, validation diagnostics, and a human summary.
- Side-effecting requests include a `confirmationToken` issued by the frontend for the matching proposal or execution preview.
- Tool responses distinguish `success`, `blocked`, `requires_confirmation`, `validation_failed`, and `error`.
- Tool responses should include compact natural-language summaries for the assistant and structured payloads for UI rendering.
- Large record reads should support limits, filters, and selected field projection.
- The host may reject broad reads when the assistant request would exceed context or privacy limits.
- Tool schemas must be authored in the strictest supported JSON Schema subset used by the host, rather than relying on provider-specific normalization at request time.
- OpenAI-compatible tool definitions emitted by the host must use strict object schemas by default: every object schema sets `additionalProperties: false`, every property key is listed in `required`, optional values are represented as nullable or explicit sentinel values, and unsupported schema decoration such as `$schema` is omitted from provider payloads.
- Nested object schemas and array item object schemas follow the same strict required-property and `additionalProperties: false` rules as top-level tool parameters.
- Tool schemas must avoid ambiguous optional properties because local providers such as Foundation Models `fm serve` may reject otherwise valid JSON Schema shapes that hosted providers tolerate.
- The canonical tool catalog should remain transport-neutral, but its baseline schema shape must already be acceptable to strict OpenAI-compatible providers, Codex CLI structured prompts, Foundation Models `fm` tool declarations, and future MCP tools.
- Browser-only actions such as opening a file picker or focusing the active table are frontend tools. They may update UI state but must not save canonical project data directly.
- Backend tools are the only tools that may read canonical workspace files, validate data, write YAML, manage generations, or create export artifacts.
- Uploaded files are referenced by host-issued upload IDs. Agent tools must not receive arbitrary local filesystem paths from the browser.

## Safety Rules

- Tools must not expose arbitrary shell execution, raw filesystem read/write, unrestricted HTTP requests, or environment variables.
- Tools must not accept path traversal input or wildcard deletion targets.
- Write tools must reuse the same validation, save, merge, delete, duplicate, and export services used by ordinary UI actions.
- Write tools that affect canonical files must be previewable before confirmation.
- Export tools must report destination paths and overwrite behavior before confirmation.
- Upload/import execution tools must report source upload IDs, detected file type, target table, affected records, and binary asset destinations before confirmation.
- If a provider cannot reliably emit structured tool calls, execution tools are disabled unless the host can parse and validate an exact structured proposal.
- Command-backed providers must not receive a capability to run arbitrary commands as an agent tool; their process execution is limited to the configured provider command itself.
- The ordinary table editor remains the fallback repair surface for records changed through AI tools.

## MCP Reuse

The tool implementation should be transport-neutral so a future MCP server can expose the same tools to external agents. MCP exposure may use stricter defaults than the in-app assistant, because the user may not see every intermediate UI event inside MasterDataMate.

An MCP server must preserve the same rules: scoped context, no raw filesystem access, proposal before write, validation before write, and explicit confirmation or host policy before side effects.

## Dependencies

- [AI assistant service](ai-assistant-service.md)
- [AI provider configuration model](../data-model/ai-provider-configuration-model.md)
- [Schema validation engine](schema-validation-engine.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)
- [Generation persistent merge flow](../data-flow/generation-persistent-merge-flow.md)
- [Generation deletion flow](../data-flow/generation-deletion-flow.md)
- [Generation duplication flow](../data-flow/generation-duplication-flow.md)
- [Generation analysis flow](../data-flow/generation-analysis-flow.md)
- [Export execution flow](../data-flow/export-execution-flow.md)

## Reads

- Table schemas, generation records, generation metadata, validation diagnostics, and export settings through application services.
- Host-issued AI session context.

## Writes

- Pending proposals and confirmation previews.
- Canonical YAML files only through confirmed execution tools.
- Export artifacts only through confirmed export tools.

## Related Requirements

- [In-app AI assistant panel](../ui-screen/in-app-ai-assistant-panel.md)
- [Web service host](../server-component/web-service-host.md)
- [Wails desktop host](../server-component/wails-desktop-host.md)

## Native-Language Summary

AIが使える操作を、読み取り・分析・提案・実行に分けて定義する。世代マージやファイルエクスポートのような高度で副作用のある操作は、LLMが直接ファイルを触るのではなく、ホストが定義したツールとして実行する。将来のMCPサーバー、Codex CLI、macOSの `fm` Foundation Modelsコマンドにも同じツール実装を再利用できるようにする。
