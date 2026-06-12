# Project Backlog

## Items

- id: BLG-20260608-001
  state: backlog
  type: implementation
  title: Implement record binary file upload support
  docs: [binary-asset-model, canonical-yaml-file-layout, table-schema-model, table-editing-workspace, shared-web-editing-frontend, html-editor-plugin-runtime, web-service-host]
  sources: [user:2026-06-08]
  acceptance: Schema fields can declare `binary_file` metadata with allowed extensions, MIME types, and size constraints; uploaded files are stored under `masterdata/binaries/<table>/<primary-key>.<extension>` using safe deterministic filename encoding; table editing renders binary fields as upload controls that support click-to-select and drag-and-drop; upload, replace, preview/download, and delete flows call host APIs and never embed bytes in ordinary row commit payloads; upload responses return normalized metadata that can be saved in the row field; custom HTML editor plugins receive scoped host API functions for uploading, deleting, and previewing binary assets for records in their context; host APIs reject path traversal, unknown tables or records, unsupported extensions, oversized files, readonly records, and out-of-scope plugin requests; required binary fields validate that metadata and the stored file both exist.
  blockers: []
  updated: 2026-06-08

- id: BLG-20260605-001
  state: backlog
  type: implementation
  title: Implement HTML editor plugin runtime for visual master data editors
  docs: [editor-plugin-model, html-editor-plugin-runtime, shared-web-editing-frontend, table-editing-workspace, web-service-host]
  sources: [user:2026-06-05]
  acceptance: Project-local plugin declarations can register a custom editor delivered as built HTML/JavaScript/CSS assets under `masterdata/plugins`, with optional `masterdata/plugins`-scoped source_dir and build metadata for Vite/React/Vue or similar source projects, target tables, writable scopes, record filters, explicit entry_points for sidebar navigation, table-toolbar actions, record actions, and group actions, and one of three data scope modes: one selected record plus related records, one selected grouping key representing multiple records such as the same external key, or immediate whole-table opening without an intermediate grid; record and group entry modes use extable selection surfaces before the plugin opens; table entry mode opens the custom editor directly; schema UI metadata can hide plugin-only or implementation-detail tables from the ordinary table list while keeping them available to plugin scopes, validation, export, diagnostics, and repair tooling; the host loads scoped schemas and records for declared tables only; runtime asset routes serve the built output tree and do not serve source directories or dependency folders; plugin code receives an explicit entry context, entry point context, and host API, proposes canonical table change sets, participates in dirty-state, validation, save confirmation, revert, and reload flows, and cannot write YAML directly; map editor style parent/child editing supports one map row plus matching map item rows by key; group editors support all records sharing one declared field value; commits validate through the schema validation engine and either update all affected tables consistently or fail with clear recovery diagnostics.
  blockers: []
  updated: 2026-06-05

- id: BLG-20260603-002
  state: backlog
  type: implementation
  title: Implement standalone Go CLI export runner
  docs: [go-cli-export-runner, export-execution-flow, export-backend-adapters, export-settings-model, go-embedded-web-server-host, schema-validation-engine]
  sources: [user:2026-06-03]
  acceptance: Packaged Go executable supports `masterdatamate export` without starting the web server; export accepts `--format` and `--output`, uses generation metadata `output: true` as the generation selection when `--generations` is omitted, and accepts explicit `--generations` when callers need a pinned generation set; CLI formats are `csv`, `excel-csv`, `json`, `yaml`, `ndjson`, `sql`, `xlsx`, and `sqlite`, with CSV/Excel CSV/JSON/YAML/NDJSON written as unpacked output directories instead of ZIP archives; standard CSV is UTF-8 without BOM, LF, header rows, RFC 4180-style double-quote escaping, and `true/false` booleans; `excel-csv` writes UTF-8 BOM, `TRUE/FALSE` booleans, and prepends a single apostrophe to formula-like string values before CSV quoting; optional `--time-format`, `--timezone`, `--check-only`, `--json`, `--diagnostics-output`, `--mkdirs`, and `--force-overwrite` follow the CLI spec; when format options such as `--time-format` or `--timezone` are omitted, CLI resolves them from `masterdata/export_settings.yaml` for the selected logical format before built-in defaults; CLI uses the same Go export service and adapters as HTTP/Wails hosts; validation errors block artifact writes and return exit code 1; output writes are atomic and never partially overwrite existing files or directories; CLI and HTTP checks return equivalent diagnostics for the same workspace, generation set, format, and options.
  blockers: [BLG-20260528-001, BLG-20260530-001]
  updated: 2026-06-03

- id: BLG-20260603-001
  state: backlog
  type: implementation
  title: Resolve default workspace by upward project-root discovery
  docs: [go-embedded-web-server-host, canonical-yaml-file-layout]
  sources: [user:2026-06-03]
  acceptance: When `--workspace` is omitted, the Go web server starts discovery from the process current working directory, selects the current directory immediately when it contains `masterdata/`, otherwise walks upward to the nearest ancestor containing `masterdata/`, and reads `masterdata/schema` plus `masterdata/generations` from that workspace; nested sample workspaces such as `examples/maze` win over an ancestor repository when launched from the sample directory; project root markers such as `go.mod`, `.git`, or `package.json` may help diagnostics but must not override the nearest `masterdata/` workspace; direct execution of `dist-native/masterdatamate` from the repository root or a descendant directory loads data correctly; implicit resolution never uses the native binary path, npm wrapper path, embedded asset path, or `dist-native/` directory as the workspace.
  blockers: []
  updated: 2026-06-03

- id: BLG-20260530-001
  state: backlog
  type: implementation
  title: Package Go embedded web server and Wails desktop hosts
  docs: [go-embedded-web-server-host, wails-desktop-host, web-service-host, shared-web-editing-frontend, product-overview]
  sources: [user:2026-05-30]
  acceptance: React + Vite production assets are embedded into a Go web server binary with `go:embed`; packaged web server serves `/api/*`, static assets, and SPA fallback from one executable without Node.js at runtime; shared Go services own YAML persistence, validation, generation, schema, and export behavior used by both hosts; Wails desktop packaging renders the same frontend without a runtime Vite or Node server; frontend components call a host adapter boundary so HTTP web mode and Wails binding mode expose the same logical operations; build scripts fail clearly when embedded frontend assets are stale or missing.
  blockers: []
  updated: 2026-05-30

- id: BLG-20260528-001
  state: backlog
  type: implementation
  title: Implement checked multi-format export execution
  docs: [export-execution-flow, export-backend-adapters, export-settings-model, generation-merge-and-export-flow, schema-validation-engine, web-service-host, shared-web-editing-frontend]
  sources: [user:2026-05-28]
  acceptance: Export UI defaults generation selection from the current output generation selection but sends explicit generationIds; export dialog loads persisted project-level defaults for each logical format from `masterdata/export_settings.yaml`, saves changed format options such as temporal formatting and timezone, and uses them as the next initial values; `POST /api/exports/check` runs strict pre-export validation before artifact creation; validation reports missing external reference targets when the referenced record is absent from the selected effective export dataset or belongs only to non-selected/output-disabled/export-false data; `POST /api/exports` repeats or verifies the check and produces deterministic downloadable artifacts for implemented formats; initial supported formats are chosen from CSV ZIP, Excel CSV (BOM) ZIP, JSON ZIP, YAML ZIP, SQL, Excel workbook, SQLite, and NDJSON ZIP; unsupported recognized formats return a deterministic not-implemented error.
  blockers: [BLG-20260520-002]
  updated: 2026-05-28

- id: BLG-20260528-002
  state: backlog
  type: implementation
  title: Implement translation editing and export support
  docs: [table-schema-model, table-editing-workspace, schema-validation-engine, export-backend-adapters, web-service-host]
  sources: [user:2026-05-28]
  acceptance: Master data can define translatable fields and supported locales; editing workflows provide locale-specific values without duplicating primary records; validation reports missing required translations and invalid locale keys; generation merge preserves translation values with the same override rules as base records; export adapters can include translations in deterministic format-specific output.
  blockers: []
  updated: 2026-05-28

- id: BLG-20260527-001
  state: backlog
  type: implementation
  title: Implement schema editing extable list and detail workflows
  docs: [schema-editing-screen, table-schema-model, shared-web-editing-frontend, schema-validation-engine, web-service-host]
  sources: [user:2026-05-27]
  acceptance: Schema editing is available to any app user without a separate permission split; it provides an extable commit-mode schema list with editable table system_name, export, and comment plus readonly primary key and reference summaries; schema list and detail editing both support local undo/redo for uncommitted edits; table system_name changes are handled as confirmed rename operations that rename schema files and matching generation table data files; new schema creation starts as a blank draft; schema deletion removes schema files only; schema duplication is not provided; detail editing uses one extable for primary key, reference, data, and formula rows with defaults while formula authoring is disabled until formula implementation; field system_name changes, including primary key fields, rename matching YAML keys; defaults are materialized into loaded editable row values and saved normally when the row is saved; type changes can save with warning confirmation when existing data mismatches; primary key changes use the same commit confirmation as other schema changes; reference target changes keep stored primary-key values unchanged and report mismatches as validation errors; save and revert match generation editing behavior; after schema changes, the frontend reloads schema caches and record data is loaded lazily on table selection; field deletion shows confirmation and removes the deleted column from existing table data atomically.
  blockers: []
  updated: 2026-05-27

- id: BLG-20260519-005
  state: ready
  type: decision
  title: Specify formula supported operators, functions, coercion, errors, and formatting
  docs: [table-schema-model, schema-validation-engine, table-editing-workspace]
  sources: [user:2026-05-19, https://www.skypack.dev/view/lib-jessie, https://npm.io/package/lib-jessie]
  acceptance: Formula language remains a Jessie-compatible expression-only subset, and the spec defines the exact supported operators/functions, type coercion rules, error behavior, and display formatting.
  blockers: []
  updated: 2026-05-26

- id: BLG-20260519-005A
  state: backlog
  type: implementation
  title: Implement formula fields after initial slice
  docs: [table-schema-model, schema-validation-engine, table-editing-workspace]
  sources: [user:2026-05-19, user:2026-05-28]
  acceptance: Formula fields are authored from schema editing, evaluated from Jessie-compatible expressions, recalculated when dependent values change, rendered read-only in data editing, validated with field-level diagnostics, and optionally exported.
  blockers: [BLG-20260519-005]
  updated: 2026-05-28

- id: BLG-20260519-006
  state: backlog
  type: implementation
  title: Implement schema dependency graph validation for external references
  docs: [table-schema-model, schema-validation-engine]
  sources: [user:2026-05-19]
  acceptance: Validator rejects external reference and parent dependency cycles with table/field-level diagnostics.
  blockers: []
  updated: 2026-05-19

- id: BLG-20260519-008
  state: backlog
  type: documentation
  title: Design diverse sample master data set
  docs: [table-schema-model, canonical-yaml-file-layout, table-editing-workspace]
  sources: [user:2026-05-19]
  acceptance: Sample data covers multiple scalar types, constants, formula fields, export:false fields, external references, and embedded dependent tables.
  blockers: []
  updated: 2026-05-19

- id: BLG-20260520-002
  state: backlog
  type: implementation
  title: Implement full dataset generation merge API
  docs: [generation-merge-and-export-flow, generation-model, web-service-host, export-backend-adapters]
  sources: [user:2026-05-20]
  acceptance: API accepts selected generation IDs in the request body, validates generation existence and configuration, sorts generations by the configured ordering mode, merges the full dataset table by table, lets newer generations override older records with the same primary key, returns response-only provenance comment metadata for winning records, returns diagnostics for the effective merged dataset, and produces a normalized dataset shape that export adapters can consume later.
  blockers: []
  updated: 2026-05-20

- id: BLG-20260520-003
  state: backlog
  type: implementation
  title: Add confirmation before generation metadata writes
  docs: [generation-editing-screen, web-service-host]
  sources: [user:2026-05-20, user:2026-05-25]
  acceptance: Creating a generation folder and committing generation metadata edits both ask for confirmation before writing files; failed validation keeps the user in context and reports blocking diagnostics.
  blockers: []
  updated: 2026-05-26

- id: BLG-20260525-001
  state: backlog
  type: implementation
  title: Implement generation bulk administration actions
  docs: [generation-editing-screen, generation-deletion-flow, generation-persistent-merge-flow, generation-duplication-flow, generation-analysis-flow, web-service-host]
  sources: [user:2026-05-25, user:2026-05-26]
  acceptance: Generation editing supports extable leftmost checkbox column based administration actions including delete, merge, duplicate generation, and Analyze; selection is not shown in a separate side panel; generation metadata columns are selection, index, path, output, and description, with output to the right of path and no default Label/Folder columns; Duplicate supports one or more selected generations, runs immediately without input or confirmation dialogs, copies output and description from each source, assigns destination indexes from current max by +10 in source order, and derives unique path names from `<source path_name>_copy`; Analyze returns selected-generation table counts, record counts, diagnostics, and merge impact as a read-only result; output bulk toggle is intentionally excluded; write actions other than Duplicate require a confirmation dialog before their API call.
  blockers: [BLG-20260520-003]
  updated: 2026-05-26
