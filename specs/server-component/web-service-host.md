---
id: "web-service-host"
type: "server-component"
title: "Web service host"
aliases: []
tags: ["web-service", "host", "initial-slice"]
facts:
  lifecycle.status: "blueprint"
  owner: "application"
---

# Web service host

## Summary

The web service host is the first delivery shape for MasterDataMate. It is implemented with Hono, serves the React + Vite shared editing frontend, loads configured schemas and the single initial generation, validates table data, and saves canonical YAML files.

The current Hono implementation is a development and prototype host. Packaged web delivery is specified separately as [Go embedded web server host](go-embedded-web-server-host.md), which preserves this HTTP API surface while embedding the frontend build into one Go executable.

## Responsibilities

- Serve the table editing workspace.
- Serve the React + Vite single page application.
- Load schema configuration from files.
- Load records from the canonical YAML file layout.
- Load both table-per-file YAML and parent files that embed dependent table records.
- Reject table open/load when the table schema is invalid.
- Reject table open/load when the active generation `_config.yaml` is missing or invalid.
- Treat a missing table YAML file in the active generation as an empty table.
- Create a missing table YAML file on first successful commit.
- Provide APIs for table data, commit-mode row mutations, validation diagnostics, external reference lookup, and file save.
- Provide APIs for binary asset upload, preview/download, metadata lookup, and deletion.
- Provide APIs for editor plugin discovery, plugin asset loading, scoped plugin data, and plugin change-set commit.
- Provide APIs for SPA page data: generation list, generation metadata, schema list, schema detail, and schema validation.
- Provide APIs for optional AI assistant runs, CopilotKit runtime integration, AI settings load/save, provider profile discovery, provider health checks, secure credential storage, and agent tool execution when AI features are enabled.
- Enforce configured save behavior when validation errors exist.
- Do not implement separate application-level authorization for schema editing; users who can access the editing app may call schema editing APIs.
- Keep Git, GitHub review, repository permissions, and approval workflow outside the application.

## Interfaces

- Table list API.
- Table schema API.
- Table generation records API.
- Generation-aware table view API.
- Validation API.
- External reference lookup API.
- Save YAML API.
- Later slice: generation list API.
- Later slice: generation metadata API.
- Later slice: generation global settings API.
- Later slice: generation deletion API.
- Later slice: generation duplication API.
- Later slice: generation analysis API.
- Schema list API.
- Schema detail API.
- Schema validation API.
- Future slice: full dataset generation merge API.
- Future slice: persistent generation merge API.
- Later slice: export API.
- Future slice: AI assistant run API.
- Future slice: CopilotKit runtime-compatible AI endpoint.
- Future slice: AI provider health check API.
- Future slice: AI settings and credential storage APIs.
- Future slice: AI agent tool execution API.
- Later slice: editor plugin discovery API.
- Later slice: editor plugin scoped data API.
- Later slice: editor plugin multi-table commit API.
- Later slice: binary asset upload API.

## HTTP Routes

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/api/tables` | List table schemas available under `masterdata/schema`. |
| GET | `/api/tables/:table/schema` | Load one table schema. |
| GET | `/api/tables/:table/generations/:generationId/records` | Load records for one table in one generation. |
| GET | `/api/tables/:table/generation-view` | Later slice: load active-generation records plus optional previous-generation readonly context and override metadata. |
| POST | `/api/tables/:table/generations/:generationId/records/commit` | Apply pending row operations from the frontend commit mode. Request body supplies operations and optional `force: true`. |
| POST | `/api/tables/:table/generations/:generationId/validate` | Validate one table's records in one generation. |
| GET | `/api/tables/:table/references` | Return reference candidates for a target table as primary key and display name pairs. Supports generation-aware lookup parameters in later slices. |
| GET | `/api/binaries/:table/:key` | Later slice: download or preview a binary asset by table and encoded primary key. |
| POST | `/api/binaries/:table/:key` | Later slice: upload or replace a binary asset for a table record. Uses multipart file upload. |
| DELETE | `/api/binaries/:table/:key` | Later slice: delete a binary asset for a table record. |
| GET | `/api/schemas` | Later slice: list editable schemas for schema editing page. |
| GET | `/api/schemas/:table` | Later slice: load one editable schema detail. |
| PUT | `/api/schemas` | Later slice: save schema list metadata edits. |
| PUT | `/api/schemas/:table` | Later slice: save a table schema. |
| POST | `/api/schemas/:table/rename` | Later slice: rename a table schema system name and update schema references. |
| POST | `/api/schemas/:table/fields/rename` | Later slice: rename schema field system names and matching YAML keys. |
| POST | `/api/schemas/:table/fields/delete` | Later slice: delete selected schema fields and remove matching data columns. |
| POST | `/api/schemas/delete` | Later slice: delete selected schema files without deleting generation data files. |
| GET | `/api/generations` | Later slice: list generations. |
| PUT | `/api/generations/:generationId/config` | Later slice: save generation `_config.yaml`. |
| POST | `/api/generations/merge` | Future slice: merge a selected generation set into one effective dataset. |
| POST | `/api/generations/persistent-merge` | Future slice: create a new generation folder by merging selected source generations. |
| POST | `/api/generations/delete` | Future slice: delete one or more selected generation folders after UI confirmation. |
| POST | `/api/generations/duplicate` | Future slice: copy one or more selected generation folders into new generations with automatic destination metadata. |
| POST | `/api/generations/analyze` | Future slice: return read-only counts, diagnostics, and merge impact for selected generations. |
| GET | `/api/editor-plugins` | Future slice: list configured editor plugins, entry points, declared target tables, and table visibility diagnostics. |
| GET | `/api/editor-plugins/:pluginId/assets/*` | Future slice: serve a plugin's declared built HTML and static assets from the resolved runtime plugin asset tree. |
| POST | `/api/editor-plugins/:pluginId/context` | Future slice: build scoped plugin data for the active generation, selected entry scope, and declared target table filters. |
| POST | `/api/editor-plugins/:pluginId/changes/validate` | Future slice: validate a plugin change set without writing YAML. |
| POST | `/api/editor-plugins/:pluginId/changes/commit` | Future slice: commit a validated plugin change set for one or more writable target tables. |
| GET | `/api/ai/profiles` | Future slice: list configured AI provider profiles visible to the current host. |
| GET | `/api/ai/settings` | Future slice: load AI settings, provider profiles, credential presence metadata, and local provider detection summary. |
| PUT | `/api/ai/settings` | Future slice: save AI settings and provider profile metadata. Raw credentials are accepted only for replacement and are never returned. |
| DELETE | `/api/ai/profiles/:profileId/credential` | Future slice: delete the stored credential for one provider profile after confirmation. |
| POST | `/api/ai/profiles/:profileId/health` | Future slice: check endpoint, model, streaming, and tool-call capability for one provider profile. |
| GET | `/api/ai/local-providers` | Future slice: detect local provider availability such as `fm`, Ollama, LM Studio, and Codex CLI. |
| GET | `/api/ai/sessions` | Future slice: list AI assistant sessions for the current workspace. |
| POST | `/api/ai/sessions` | Future slice: create a new AI assistant session for the current workspace. |
| GET | `/api/ai/sessions/:sessionId` | Future slice: load one AI assistant session timeline and metadata. |
| PATCH | `/api/ai/sessions/:sessionId` | Future slice: rename, archive, or restore one AI assistant session. |
| DELETE | `/api/ai/sessions/:sessionId` | Future slice: delete one AI assistant session. |
| POST | `/api/ai/sessions/:sessionId/compact` | Future slice: compact old assistant context into a summary for one session. |
| POST | `/api/ai/runs` | Future slice: start an AI assistant run and stream assistant events, preferably through AG-UI-compatible event semantics. |
| POST | `/api/ai/tools/:toolName/confirm` | Future slice: issue a confirmation token for a side-effecting AI tool preview. |
| POST | `/api/copilotkit` | Future slice: CopilotKit runtime-compatible endpoint for the in-app AI assistant popup. |

The first runnable slice uses `generationId = 0000_initial` for record load, commit, and validation routes. The generation segment remains in the path so the API shape does not need to change when multi-generation support is added.

AI routes are disabled unless [AI provider configuration model](../data-model/ai-provider-configuration-model.md) enables AI features and at least one usable provider profile is configured. AI assistant runs may use OpenAI-compatible hosted APIs, local HTTP providers such as Ollama and LM Studio, local `fm serve`, or command-backed providers such as Codex CLI and Apple Foundation Models `fm` through [AI assistant service](../component/ai-assistant-service.md). AI tool execution follows [Agent tool contract](../component/agent-tool-contract.md) and must not expose raw filesystem, shell, unrestricted network, or unscoped application internals.

The `/api/copilotkit` route is the preferred frontend integration route for the [In-app AI assistant panel](../ui-screen/in-app-ai-assistant-panel.md). It exposes a Copilot Runtime-compatible surface for CopilotKit prebuilt components and streams AG-UI-compatible events from the MasterDataMate assistant. The route registers MasterDataMate server tools from [Agent tool contract](../component/agent-tool-contract.md), accepts scoped frontend context, and delegates provider calls to [AI assistant service](../component/ai-assistant-service.md).

AI settings routes support [AI settings screen](../ui-screen/ai-settings-screen.md). The settings load response returns provider metadata and credential presence only. It must never include raw API keys. Settings save may receive a new API key once from the browser and must immediately hand it to [AI secret storage service](../component/ai-secret-storage-service.md). Subsequent settings loads display saved credentials as masked placeholders through frontend behavior using `hasCredential` metadata.

On macOS, if no explicit active AI profile is configured and `fm` is available, the AI settings response should mark the local `apple-fm-serve` profile as the default selection. This default does not require an API key.

AI session routes are backed by [AI assistant session model](../data-model/ai-assistant-session-model.md). They store host-local conversation state below `os.UserConfigDir()/masterdatamate/<workspaceBaseName>-<workspacePathHash>/ai-sessions/`, where the hash is derived from the cleaned absolute workspace root. Session route responses must expose IDs, titles, timestamps, provider metadata, compaction markers, and safe timeline content, but not the host storage path. `POST /api/ai/runs` must accept or create a `sessionId` and persist the resulting user, assistant, tool, run, and compaction data through [AI assistant service](../component/ai-assistant-service.md).

Editor plugin APIs are disabled unless plugin declarations are present. The plugin discovery route returns metadata from [Editor plugin model](../data-model/editor-plugin-model.md) declarations, normally loaded from `masterdata/editor_plugins.yaml`, including normalized entry points for sidebar, table toolbar, record, and group placements. It must not expose filesystem paths outside the declared built plugin asset root under `masterdata/plugins`. Source directories and build metadata may be returned only as developer diagnostics; asset routes must not serve source trees, lockfiles, package manifests, or dependency folders unless they are intentionally part of the built output.

Discovery should also report schema UI visibility metadata relevant to navigation. Tables with `ui.table_list_visibility: plugin_only` or `hidden` are omitted from ordinary table navigation, but the server must still include them in plugin scope resolution and diagnostics when plugin declarations reference them.

The plugin context route accepts `pluginId`, `activeGenerationId`, `mode`, and an entry scope. For `record` scope, it validates that the selected primary record belongs to the declared primary table. For `group` scope, it validates the declared grouping table and field, then resolves the selected group value to matching records. For `table` scope, it requires no selected row and returns the full declared table scope. In every mode, the route resolves declared `equals` filters and returns only records from declared target tables. Returned rows include the same generation provenance and readonly metadata used by generation-aware table views.

Plugin validation and commit routes accept canonical change sets grouped by table. They reject unknown tables, undeclared writes, writes to readonly previous-generation records, path traversal attempts, and record values that cannot be normalized under the table schema. Validation uses the schema validation engine and returns diagnostics without writing files. Commit follows the same validation-error behavior as ordinary table save: without `force`, validation errors block writes; with explicit `force: true`, the host may save and return diagnostics when project save behavior allows it.

Multi-table plugin commits should be atomic from the user's perspective. If the implementation cannot atomically update every affected table file, the route must either reject multi-table commits up front or return a recovery diagnostic that identifies which table writes completed and force the frontend to reload those tables.

Binary asset routes operate under `masterdata/binaries` as specified by [Binary asset model](../data-model/binary-asset-model.md). Upload requests validate table existence, record key existence, editable generation context when relevant, schema `binary_file` constraints, extension, MIME type when detectable, file size, and path safety. The upload response returns normalized metadata and a preview/download URL; clients store returned metadata in the matching `binary_file` field through ordinary table or plugin change flows. Delete requests remove the stored file and return metadata suitable for clearing the field. These routes never accept caller-supplied filesystem paths.

Editor plugin host APIs for binary upload call the same binary asset service as table editing. Plugin calls are additionally scoped to records and tables included in the plugin context.

The later `GET /api/tables/:table/generation-view` route accepts required query parameters `activeGenerationId` and `mode`. `mode=active_only` returns only records from the active generation. `mode=include_previous` returns records from output-enabled generations older than or equal to the active generation, plus the active generation even when its own `output` flag is false. The route never returns records from generations newer than the active generation. The server derives `orderedGenerationIds` from generation metadata; clients do not pass arbitrary previous generation IDs for this view. Response rows are editing rows with generation metadata added to the data: `sourceGenerationId`, `sourceGenerationLabel`, `isActiveGeneration`, `isReadOnly`, `isOverridden`, and optional `overriddenByGenerationId`. The route is read-only and does not modify YAML files.

`sourceGenerationId` is the authoritative provenance field for a generation-aware table row. Rows whose `sourceGenerationId` is not `activeGenerationId` must have `isReadOnly: true`. Rows whose `sourceGenerationId` is `activeGenerationId` may have `isReadOnly: false`. The frontend should use these fields directly rather than recomputing readonly state from row order.

Row indexes are zero-based positions in the YAML record sequence. Commit operations can include insert, update, delete, and move operations. Update operations include the previous primary key to reduce accidental updates when the client row index is stale. Commit responses should return current validation diagnostics and enough ordering metadata for the UI to keep `extable` synchronized without reloading the full table when practical.

Commit requests may only write rows in the route generation ID. If the frontend displays previous-generation rows through the generation-aware table view, those rows must not be submitted as active-generation writes. The server should reject commit operations that attempt to modify readonly previous-generation rows when row provenance is supplied.

Reference candidate responses are table-scoped, not field-scoped. A reference candidate contains the referenced table primary key and human-readable name. Record data stores only the referenced primary key; the name is display-only for editing and lookup UI. Frontends may combine these fields into lookup labels such as `Name (primary-key)` for selection, but commit requests and YAML writes must use only the primary key value.

Schema list responses include editable table-level metadata and readonly summaries needed by the schema list grid: table `system_name`, `export`, `comment`, primary key field names, and referenced target table identifiers. Schema list save requests may update table-level metadata but must not silently change primary key or reference definitions; those are edited through the schema detail route. If `system_name` changes, the server treats it as a table schema rename operation rather than a generic metadata update.

The later `GET /api/schemas/:table` route returns the canonical table schema plus a UI-friendly ordered field row representation with `kind` values derived from primary key membership, `reference`, and `formula`. The UI representation is an adapter shape; schema YAML remains the source of truth.

Schema save routes validate the full schema before writing YAML. They must reject duplicate field names, invalid defaults, invalid formula declarations, broken primary keys, missing reference targets, and reference cycles. Field type changes that make existing stored values invalid may be saved only after the frontend presents a warning confirmation and sends the user's explicit confirmation. If schema changes affect table editing, the frontend reloads the schema list and full schema cache after save; table record data is loaded lazily when the user selects a table for editing.

The later `POST /api/schemas/:table/rename` route renames a table schema `system_name`, renames the schema file identity, renames matching table data file names under generation folders, and updates other schema definitions that reference the renamed table. It must report old and new names, changed schema files, renamed data file paths, and any reference-update diagnostics.

The later `POST /api/schemas/:table/fields/rename` route accepts explicit old and new field system names and renames matching YAML keys for existing records. For non-primary-key fields it renames keys under record `data`. For composite primary keys it renames the matching property inside object-shaped reserved YAML `key` values. For a single scalar primary key it updates schema `primary_key` metadata and leaves scalar reserved YAML `key` values structurally unchanged. This route is used for primary key and non-primary-key field renames alike.

The later `POST /api/schemas/delete` route accepts explicit schema table identifiers and removes only the matching schema YAML files under `masterdata/schema`. It does not remove generation table YAML files. The request must reject unknown schemas and path traversal, must constrain deletion to the schema root, and should return deleted file paths for Git-backed review.

The later `POST /api/schemas/:table/fields/delete` route accepts explicit field system names and removes those fields from the schema. It also removes the corresponding values from canonical YAML record `data` for every generation that stores the table, including embedded dependent table records for the affected table. It must reject deletion that would leave no valid primary key, deletion of fields still used by formulas or relationships, unknown fields, and requests that would require unsafe partial cleanup. The operation should be atomic from the user's perspective: if any affected YAML file cannot be parsed or written, no schema or data deletion should be committed.

Schema default values are materialized when table data is loaded into the editable row model. If the user later saves that loaded row, the materialized value is written like any other row value. Export consumes the same normalized row values.

Changing an external reference target table keeps stored reference primary key values unchanged. Validation resolves those unchanged values against the new target table and reports unresolved references as normal validation errors.

When the table editor is in `include_previous` mode, reference lookup uses output-enabled generations older than or equal to the active edit generation. When the editor is in `active_only` mode, reference lookup uses the active edit generation only if that generation is output-enabled. Reference lookup must exclude records from generations whose `output` flag is false, even when the active generation is visible and editable in the table view. If duplicate referenced primary keys exist across lookup generations, the candidate response should expose the effective winning candidate and include `sourceGenerationId` and `overrodeGenerationIds` metadata for explanation.

Generation-aware reference lookup uses `activeGenerationId` and `mode` query parameters. If these parameters are omitted, the route preserves first-slice behavior by using `generationId` or the default generation. The response shape remains compatible with simple lookup candidates, but later slices can add optional metadata:

```http
GET /api/tables/org/references?activeGenerationId=0010_balance_patch&mode=include_previous
```

```json
{
  "candidates": [
    {
      "key": "org-product",
      "label": "Product Team",
      "sourceGenerationId": "0010_balance_patch",
      "sourceGenerationLabel": "0010_balance_patch",
      "overrodeGenerationIds": ["0000_initial"]
    }
  ]
}
```

The first runnable slice assumes single-user editing and does not require locking, ETags, or revision conflict checks. Supplying the previous primary key in row update operations is still required to reduce accidental updates when the client row index is stale. If the row at the target index no longer has the supplied previous primary key, the server should reject the commit with a stale-row diagnostic.

If a commit has validation errors and `force` is not true, the server returns diagnostics and does not save. If `force: true` is supplied, the server saves the requested operations despite validation errors and returns diagnostics for display.

The future `POST /api/generations/merge` route accepts a request body with `generationIds` and optional `includeDiagnostics`. It validates every requested generation, sorts the set by the configured generation ordering mode, reads all registered table schemas, and returns a normalized full dataset grouped by table. Newer generations override older records with the same table primary key. Winning records include response-only `comment.sourceGenerationId` and optional `comment.overrodeGenerationIds` metadata so the UI and export workflow can explain which generation supplied the effective value.

Generation merge errors use `400` for malformed or empty `generationIds`, `404` for missing generation folders, `422` for invalid generation configuration or unnormalizable merged primary keys, and `500` for unrelated filesystem or YAML read failures. Merge requests do not write canonical YAML files.

The future `POST /api/generations/persistent-merge` route accepts a request body with `sourceGenerationIds`, destination generation metadata, and optional `includeDiagnostics`. It validates at least two selected source generations, sorts sources by the configured generation ordering mode, merges normalized table records across all registered schemas, and writes a new destination generation folder under `masterdata/generations`. This route follows the same duplicate primary-key precedence as the non-persistent merge route: the source generation with the larger numeric index or later release-date index wins. The route writes `_config.yaml` and destination table YAML files only after validation succeeds, must not modify source generation folders, and must fail rather than overwrite an existing destination generation folder.

Persistent merge errors use `400` for malformed source selection or destination metadata, `404` for missing source generations, `409` for destination index/path/folder collisions, `422` for invalid generation configuration, invalid destination metadata, invalid schema data, or unnormalizable merged primary keys, and `500` for unrelated filesystem or YAML read/write failures.

Generation metadata save APIs are called only after the generation editing screen shows a commit confirmation dialog. The server still validates every metadata write independently because UI confirmation is not an authorization or consistency mechanism.

The future `POST /api/generations/delete` route accepts a request body with explicit `generationIds` and optional `activeGenerationId`. It deletes one or more selected generation folders under `masterdata/generations`, rejects requests that would delete all generations, rejects duplicate or unknown generation IDs, and returns the remaining generation IDs plus the resolved active generation when the current active generation was deleted. The server must constrain deletion to validated generation folders under `masterdata/generations`, must not follow symlinks outside that root, and must never accept wildcard or implicit all-generation deletion.

Generation delete errors use `400` for malformed or empty `generationIds`, duplicate IDs, or implicit all-generation deletion, `404` for unknown generations, `409` when deletion would leave no valid generation or active-generation fallback cannot be resolved consistently, `422` when invalid generation config prevents safe folder resolution, and `500` for unrelated filesystem delete failures.

The future `POST /api/generations/duplicate` route accepts `sourceGenerationIds` for one or more selected source generations. For backward compatibility it may also accept one `sourceGenerationId`; if legacy destination metadata is supplied, the server may preserve the older explicit single-duplicate behavior. In the automatic flow, the server sorts sources by generation ordering, assigns destination indexes from the current maximum by +10 per source, derives unique path names from `<source path_name>_copy`, copies each source generation folder, writes destination `_config.yaml` metadata with copied `output` and `description`, and does not modify source generations. The generation editing screen calls this API immediately without an input or confirmation dialog.

The future `POST /api/generations/analyze` route accepts explicit selected `generationIds` and optional `includeMergeImpact`. It reads selected generation configs, schemas, and table YAML files, then returns table counts, record counts, output states, diagnostics, and merge-impact counts. It does not write files and does not require a confirmation dialog.

## Implementation Stack

- Frontend: React + Vite.
- Server: Hono.
- Table component: `extable` from npm packages. The `vendor/extable` checkout is a reference for specification and API behavior, not the application dependency source.
- The first implementation should keep the frontend and Hono server deployable as one web service.
- Packaged web distribution should migrate this host surface to [Go embedded web server host](go-embedded-web-server-host.md).
- The first implementation assumes single-user editing.
- Optional AI assistant support should use the same host service boundary as ordinary APIs. The Hono development host may proxy or implement AI routes, while packaged Go delivery should reuse shared Go services for provider configuration, tools, validation, generation operations, and export execution.
- The first AI provider target should be `fm serve` through the OpenAI-compatible adapter when available, because it can run locally without a hosted API key.

## Dependencies

- [Product overview](../generic/product-overview.md)
- [Shared web editing frontend](../component/shared-web-editing-frontend.md)
- [HTML editor plugin runtime](../component/html-editor-plugin-runtime.md)
- [Schema validation engine](../component/schema-validation-engine.md)
- [Generic master data model](../data-model/generic-master-data-model.md)
- [Binary asset model](../data-model/binary-asset-model.md)
- [AI provider configuration model](../data-model/ai-provider-configuration-model.md)
- [AI secret storage service](../component/ai-secret-storage-service.md)
- [AI assistant service](../component/ai-assistant-service.md)
- [Agent tool contract](../component/agent-tool-contract.md)

## Reads

- Schema configuration files.
- YAML table records under `masterdata/generations`.
- Binary files under `masterdata/binaries`.
- Embedded dependent table records inside parent table YAML files.
- Generation configuration from `masterdata/generations/0000_initial/_config.yaml`.
- Later generation-aware table view: selected generation configuration and previous output-enabled table YAML files up to the active generation.
- Future merge API: selected generation configuration and table YAML files across all requested generations.
- Future persistent merge API: selected source generation configuration and table YAML files, destination generation metadata, and table schemas.
- Future delete API: selected generation configuration, folder paths, and active generation fallback context.
- Future duplicate API: selected source generation configurations and folders, automatically derived destination metadata, and copied file paths.
- Future analyze API: selected generation configuration, table YAML files, table schemas, and normalized record counts.
- Future AI provider APIs: provider configuration, host environment variables or secret references, local provider command availability, and provider health check responses.
- Future AI settings APIs: provider configuration, credential presence metadata, and local provider detection results.
- Future AI assistant runs: scoped frontend context, schemas, records, diagnostics, generation metadata, export settings, and assistant conversation state through application services.
- Later schema field deletion API: selected schema fields, table schemas, generation table YAML files, and embedded dependent records for cleanup impact.
- Later schema rename API: source schema, target schema name, and schema files that reference the renamed table.
- Later schema field rename API: selected field rename mappings, table schema, generation table YAML files, and embedded dependent records for key rename impact.
- Later schema delete API: selected schema files under `masterdata/schema`.

## Writes

- YAML table records under `masterdata/generations`.
- Parent table YAML files containing embedded dependent table records.
- Newly created table YAML files when a table has no file in the active generation.
- Validation and save operation responses.
- Future merge API responses with normalized effective datasets and response-only provenance comments.
- Future persistent merge API writes a new generation `_config.yaml` and merged destination table YAML files under `masterdata/generations`.
- Future delete API removes selected generation folders under `masterdata/generations` and returns remaining generation metadata.
- Future duplicate API writes new generation `_config.yaml` files and copied destination table YAML files under `masterdata/generations`.
- Future analyze API writes no files.
- Future AI assistant proposal tools write pending proposals and confirmation previews.
- Future AI assistant execution tools write canonical YAML files or export artifacts only after explicit user confirmation and through existing application services.
- Future AI settings APIs write provider profile metadata and OS credential-store entries through the AI secret storage service. They do not write raw API keys into project files.
- Later schema save APIs write schema YAML files under `masterdata/schema`.
- Later schema rename API writes renamed schema YAML, renamed table data file paths, and any schema YAML files whose references were updated.
- Later schema field rename API writes the affected schema YAML and table YAML files whose record keys were renamed.
- Later schema delete API removes selected schema YAML files only.
- Later schema field deletion API writes the affected schema YAML and cleaned generation table YAML files.

## Related Requirements

- [Table editing workspace](../ui-screen/table-editing-workspace.md)
- [Single page application shell](../ui-flow/single-page-application-shell.md)
- [Go embedded web server host](go-embedded-web-server-host.md)
- [Wails desktop host](wails-desktop-host.md)
- [Generation selection screen](../ui-screen/generation-selection-screen.md)
- [Generation editing screen](../ui-screen/generation-editing-screen.md)
- [Schema editing screen](../ui-screen/schema-editing-screen.md)
- [Canonical YAML file layout](../data-model/canonical-yaml-file-layout.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)
- [Generation persistent merge flow](../data-flow/generation-persistent-merge-flow.md)
- [Generation deletion flow](../data-flow/generation-deletion-flow.md)
- [Generation duplication flow](../data-flow/generation-duplication-flow.md)
- [Generation analysis flow](../data-flow/generation-analysis-flow.md)
- [In-app AI assistant panel](../ui-screen/in-app-ai-assistant-panel.md)
- [AI settings screen](../ui-screen/ai-settings-screen.md)
- [AI assistant service](../component/ai-assistant-service.md)
- [Agent tool contract](../component/agent-tool-contract.md)

## Native-Language Summary

最初の提供形態はWebService。React + Vite のSPAをHonoで配信し、設定済みスキーマと固定 `0000_initial` 世代のYAMLデータを読み書きする。保存は `warn_and_save` で、確認後に `force: true` で保存できる。配布版では、このHTTP API面をGoへ移し、Viteビルド成果物を `go:embed` で同梱した単一バイナリとして提供する。将来の永続世代マージAPIは、複数の選択元世代を通常の優先順位で統合し、新しい世代フォルダとYAMLを書き込む。世代削除APIは明示選択された世代フォルダだけを削除し、全世代削除は拒否する。世代複製APIは1つの選択元世代を新しい世代フォルダへコピーする。Analyze APIは読み取り専用で件数・診断・マージ影響を返す。`extable` はnpm packageとして利用する。
