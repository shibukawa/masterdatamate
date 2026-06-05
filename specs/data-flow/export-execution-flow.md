---
id: "export-execution-flow"
type: "data-flow"
title: "Export execution flow"
aliases: []
tags: ["export", "validation", "artifact"]
facts:
  lifecycle.status: "blueprint"
---

# Export execution flow

## Summary

The export execution flow lets users choose the generations to export, run strict pre-export validation, and produce downloadable artifacts through export backend adapters. The default selected generations come from the current UI export selection for SPA workflows, but export APIs and batch entry points always receive explicit generation IDs in the request or command arguments.

Export is read-only with respect to canonical schema and generation YAML. It may create temporary or downloadable artifacts, but it must not update source generation folders.

In the SPA, export is launched from a single project-level `Export` button in the left pane. Export is not launched from the selected table's right pane or table toolbar because it produces artifacts for the project-level effective dataset.

Standalone batch export is specified separately by [Go CLI export runner](../batch-component/go-cli-export-runner.md). The CLI follows the same validation and adapter rules without starting the web server.

## Actors

- Developer.
- Planner or other non-engineering editor.
- Export command, web-service endpoint, or packaged app runtime.
- Batch automation or CI runner.

## Components

- [Generation merge and export flow](generation-merge-and-export-flow.md)
- [Schema validation engine](../component/schema-validation-engine.md)
- [Export backend adapters](../component/export-backend-adapters.md)
- [Table schema model](../data-model/table-schema-model.md)
- [Generation model](../data-model/generation-model.md)
- [Export settings model](../data-model/export-settings-model.md)
- [Template export definition model](../data-model/template-export-definition-model.md)
- [Pongo2 template export adapter](../component/pongo2-template-export-adapter.md)
- [Go CLI export runner](../batch-component/go-cli-export-runner.md)

## APIs

- `POST /api/exports/check`: validate a requested export without creating an artifact.
- `POST /api/exports`: validate, merge, and create one export artifact.
- `GET /api/export-settings`: return persisted export settings plus effective built-in defaults.
- `PUT /api/export-settings`: persist project-level export settings for one or more logical export formats.

Both export APIs require explicit `generationIds`. The frontend may initialize the request from the current selected output generations, but the server must not infer export scope from browser-local state. The CLI likewise passes explicit generation IDs into the shared export service after parsing command-line options.

Export option resolution is shared across hosts. Explicit request `options` override [Export settings model](../data-model/export-settings-model.md) values for the selected logical format, and persisted settings override built-in adapter defaults.

Check request:

```json
{
  "generationIds": ["0000_initial", "0010_balance_patch"],
  "format": "csv_zip",
  "options": {
    "includeSchema": false
  }
}
```

Check response:

```json
{
  "exportable": false,
  "generationIds": ["0000_initial", "0010_balance_patch"],
  "orderedGenerationIds": ["0000_initial", "0010_balance_patch"],
  "format": "csv_zip",
  "summary": {
    "tableCount": 3,
    "recordCount": 182,
    "diagnosticCount": 2
  },
  "diagnostics": [
    {
      "severity": "error",
      "table": "product",
      "recordKey": "sword_001",
      "field": "organization_id",
      "message": "Referenced org record org_001 is not present in the selected export generation set."
    }
  ]
}
```

Export request:

```json
{
  "generationIds": ["0000_initial", "0010_balance_patch"],
  "format": "xlsx",
  "options": {
    "includeDiagnosticsSheet": true
  }
}
```

Export response:

```json
{
  "exportId": "2026-05-28T10-15-00Z_xlsx",
  "format": "xlsx",
  "filename": "masterdata-export.xlsx",
  "contentType": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  "downloadUrl": "/api/exports/2026-05-28T10-15-00Z_xlsx/download",
  "generationIds": ["0000_initial", "0010_balance_patch"],
  "orderedGenerationIds": ["0000_initial", "0010_balance_patch"],
  "summary": {
    "tableCount": 3,
    "recordCount": 182
  },
  "diagnostics": []
}
```

## Export Formats

| Format | Artifact | Required behavior |
| --- | --- | --- |
| `csv_zip` | ZIP archive containing one UTF-8 CSV file per exported table. | Include primary key columns and `export: true` fields only. Use deterministic file names based on table system names. |
| `excel_csv_zip` | ZIP archive containing one Excel-oriented CSV file per exported table. | Same table and column selection as `csv_zip`, but emit UTF-8 BOM and prepend a single apostrophe to string values beginning with `=`, `+`, `-`, or `@` before CSV quoting. UI label: `Excel CSV (BOM)`. |
| `json_zip` | ZIP archive containing one JSON file per exported table and optional manifest. | Emit arrays of flat records using exported field names. Preserve JSON scalar types. |
| `yaml_zip` | ZIP archive containing one YAML file per exported table and optional manifest. | Emit merged export data, not canonical generation folders. Omit response-only provenance unless an option explicitly asks for metadata. |
| `sql` | Single SQL script. | Emit `CREATE TABLE IF NOT EXISTS`, `TRUNCATE`, and `INSERT` statements in dependency order where possible. SQL dialect is an option and defaults to a conservative generic dialect. |
| `xlsx` | Excel workbook. | Emit one worksheet per exported table. Include header rows, exported fields, and optional diagnostics or manifest worksheet. |
| `sqlite` | SQLite database file. | Emit all exported tables into one local database file. This is useful for local validation, ad hoc querying, and downstream tools that prefer a single binary artifact. |
| `ndjson_zip` | ZIP archive containing one NDJSON file per exported table. | Optional streaming-friendly format for large datasets and ingestion pipelines. |
| `template_zip` | ZIP archive containing files generated by selected Pongo2 template export definitions. | Render project-defined templates from `masterdata/export_definitions.yaml` against the validated merged dataset. |

The first implementation may support a subset of formats, but unsupported requested formats must return a deterministic error before running export work.

HTTP download format identifiers keep the `_zip` suffix for multi-file formats because the browser receives one downloadable artifact. The [Go CLI export runner](../batch-component/go-cli-export-runner.md) uses unpacked format identifiers `csv`, `excel-csv`, `json`, `yaml`, `ndjson`, and `template` and writes those formats to an output directory instead of creating ZIP archives.

## Pre-Export Validation

Pre-export validation runs after generation selection and effective merge calculation, and before any artifact is produced. It uses the same strict validation rules as [Schema validation engine](../component/schema-validation-engine.md), with export-specific reference scope.

Validation must check:

- Every requested generation exists and has valid metadata.
- Requested generation IDs are unique and sorted by configured generation ordering before merge.
- Exported tables are limited to schemas with `export: true`.
- Exported fields are limited to primary key fields and fields with `export: true`.
- Primary key values are present, type-valid, and unique in the effective merged table.
- Required exported fields are present after default and formula evaluation.
- Formula fields selected for export have valid evaluated results.
- External reference fields store primary key values, not display labels.
- External reference targets exist in the effective exported dataset for the selected generation set.
- External reference targets from non-selected generations, output-disabled generations, or `export: false` tables are treated as missing for export.
- External reference fields whose own field schema has `export: false` may still be validated when another exported field or backend constraint depends on them, but they are not emitted.
- Backend-specific constraints, such as SQL identifier validity, duplicate generated column names, Excel sheet-name length, or CSV scalar formatting, are reported as export diagnostics.
- Standard CSV uses UTF-8 without BOM, LF line endings, a mandatory header row, comma delimiter, and RFC 4180-style double-quote escaping.
- Excel CSV uses UTF-8 with BOM and prepends a single apostrophe to string values beginning with `=`, `+`, `-`, or `@` before CSV quoting.
- Standard CSV emits boolean values as `true` and `false`.
- Excel CSV emits boolean values as `TRUE` and `FALSE`.
- CSV temporal formatting defaults to `iso`; optional temporal modes are `epoch-sec`, `epoch-ms`, and `iso-local`.
- `iso-local` uses an optional export timezone. If no timezone is supplied, the runtime local timezone is used.
- Template export definitions are loaded and parsed during pre-export validation when the requested format is `template_zip`.
- Template export validation checks selected definition IDs, target table existence, group field existence, Pongo2 syntax, template file path safety, rendered output path safety, and duplicate rendered output paths before artifact creation.

Validation errors block export. Warnings may allow export when they do not make the artifact ambiguous or invalid for the selected backend.

## Flow

1. The user activates the project-level `Export` button in the left pane.
2. The frontend opens an export dialog instead of immediately creating an artifact.
3. The frontend loads export settings through `GET /api/export-settings`.
4. The frontend initializes `generationIds` from the current selected output generations.
5. The frontend initializes format option controls from persisted settings for the selected logical format plus built-in defaults.
6. The user chooses an output destination, export format, and any supported export options in the dialog.
7. The user may edit the generation selection for the export request.
8. The frontend persists the selected logical format options through `PUT /api/export-settings` before or together with the export check.
9. The frontend calls `POST /api/exports/check`.
10. The server resolves export options from explicit request options, persisted format settings, and built-in defaults.
11. The server validates generation selection, merges records according to [Generation merge and export flow](generation-merge-and-export-flow.md), filters exportable tables and fields, and runs pre-export validation.
12. If blocking diagnostics exist, the frontend shows diagnostics and does not call `POST /api/exports`.
13. If the check is exportable, the frontend calls `POST /api/exports` with the same generation IDs, format, destination, and options.
14. The server repeats validation or verifies a short-lived check token, then invokes the selected adapter.
15. The response returns artifact metadata and a download URL or confirms writing to the selected destination.

For `template_zip`, request `options.definitionIds` may select specific template export definitions. When omitted, the service uses `formats.template.definition_ids` from [Export settings model](../data-model/export-settings-model.md), then enabled definitions from `masterdata/export_definitions.yaml`.

## Rules / Constraints

- Export scope is always explicit in API requests.
- CLI execution uses generation metadata `output: true` as the default generation selection when no generation IDs are supplied.
- The default export generation selection is the current UI selection.
- `generation.output` defines default selection, not mandatory inclusion.
- Users may export an output-disabled generation only when they explicitly include it in the request.
- The export entry point is project-level UI in the left pane, not table-level UI in the right pane.
- The export dialog must collect or confirm the output destination before export execution.
- The export dialog must initialize option controls from persisted settings for the selected logical format.
- The export dialog must save changed format options so they become the next initial values for that format.
- Persisted export settings are project-level defaults stored in `masterdata/export_settings.yaml`.
- Export settings are keyed by logical format IDs; HTTP ZIP format IDs map to logical IDs before settings lookup.
- Explicit request options override persisted export settings.
- Export never writes canonical generation YAML, schema YAML, or generation config.
- Response-only merge provenance is not emitted by default.
- Export diagnostics identify table, generation, record key, field, severity, and message whenever applicable.
- Artifact contents must be deterministic for the same source data, generation IDs, format, and options.
- Template export artifact contents must also be deterministic for the same selected definition IDs and template files.
- Backend adapters must consume a normalized, validated merged dataset instead of reading generation YAML directly.
- Backend adapters may add target-specific diagnostics but must not weaken core validation.
- Web API, Wails, and CLI hosts must share the same export service implementation boundary so check results and artifact semantics do not drift.
- Host-specific packaging may differ: HTTP may package multi-file outputs as ZIP downloads, while CLI writes multi-file outputs as directories.
- Generated downloads may be temporary and may expire.

## Error Cases

- `400`: `generationIds`, `format`, or `options` are malformed.
- `404`: a requested generation does not exist.
- `409`: export check passed earlier but source data changed before artifact creation.
- `422`: pre-export validation failed or the selected backend rejects the normalized dataset.
- `501`: requested export format is recognized but not implemented.
- `500`: artifact creation or filesystem failure not attributable to invalid project data.

## Related Requirements

- [Product overview](../generic/product-overview.md)
- [Generic master data model](../data-model/generic-master-data-model.md)
- [Table schema model](../data-model/table-schema-model.md)
- [Generation model](../data-model/generation-model.md)
- [Export settings model](../data-model/export-settings-model.md)
- [Template export definition model](../data-model/template-export-definition-model.md)
- [Generation merge and export flow](generation-merge-and-export-flow.md)
- [Schema validation engine](../component/schema-validation-engine.md)
- [Export backend adapters](../component/export-backend-adapters.md)
- [Pongo2 template export adapter](../component/pongo2-template-export-adapter.md)
- [Go CLI export runner](../batch-component/go-cli-export-runner.md)

## Native-Language Summary

エクスポートでは対象世代を明示指定でき、SPA のデフォルトは画面上の現在の出力対象選択を使う。GUI は形式ごとのエクスポート設定を読み込み、操作した日時形式や timezone、テンプレート definition 選択などを `masterdata/export_settings.yaml` に保存して次回の初期値にする。Go CLI では `--generations` が省略された場合に generation 設定の `output: true` 世代を使い、`--time-format` や `--timezone` などが省略された場合は保存済みの形式別設定を使う。成果物作成前に、選択世代で統合した有効データだけを対象に厳密な事前チェックを行う。外部参照は保存済み主キーで検証し、参照先が選択世代の有効なエクスポート対象データに存在しない場合はエラーとして export を止める。Pongo2 テンプレート export では `masterdata/export_definitions.yaml` の選択定義を読み、テンプレート構文、参照、出力パス安全性、重複出力を事前チェックする。標準CSVはUTF-8 BOMなし/LF/ヘッダーあり/RFC 4180相当で boolean は `true/false`、Excel CSVはBOMつきで boolean は `TRUE/FALSE` かつExcel向けの数式安全化を行う。日時は既定 `iso`、オプションで `epoch-sec`、`epoch-ms`、`iso-local` を選べる。出力形式は CSV ZIP、Excel CSV (BOM) ZIP、JSON ZIP、YAML ZIP、SQL、Excel workbook、SQLite、NDJSON ZIP、Template ZIP を候補とする。
