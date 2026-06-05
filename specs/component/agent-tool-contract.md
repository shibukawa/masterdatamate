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

The same tool implementation should be reusable by the in-app AI assistant and by a future MCP server. The entry transport may differ, but validation, authorization, confirmation, and application service calls remain shared.

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

## Safety Rules

- Tools must not expose arbitrary shell execution, raw filesystem read/write, unrestricted HTTP requests, or environment variables.
- Tools must not accept path traversal input or wildcard deletion targets.
- Write tools must reuse the same validation, save, merge, delete, duplicate, and export services used by ordinary UI actions.
- Write tools that affect canonical files must be previewable before confirmation.
- Export tools must report destination paths and overwrite behavior before confirmation.
- If a provider cannot reliably emit structured tool calls, execution tools are disabled unless the host can parse and validate an exact structured proposal.
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

AIが使える操作を、読み取り・分析・提案・実行に分けて定義する。世代マージやファイルエクスポートのような高度で副作用のある操作は、LLMが直接ファイルを触るのではなく、ホストが定義したツールとして実行する。将来のMCPサーバーにも同じツール実装を再利用できるようにする。
