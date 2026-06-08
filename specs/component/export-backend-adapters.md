---
id: "export-backend-adapters"
type: "batch-component"
title: "Export backend adapters"
aliases: []
tags: ["export", "adapter"]
facts:
  lifecycle.status: "blueprint"
---

# Export backend adapters

## Summary

Export backend adapters convert validated merged master data into backend-specific runtime data artifacts. The logical formats are CSV, Excel CSV, JSON, YAML, SQL, Excel workbook, SQLite, and NDJSON. Web delivery may package multi-file formats as ZIP archives, while CLI delivery writes multi-file formats directly to an output directory. Pongo2 source generation is handled by [Go CLI generate runner](../batch-component/go-cli-generate-runner.md), not by export formats.

## Scope

- In scope:
  - Consume validated merged data from [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md).
- Consume export requests and validation results from [Export execution flow](../data-flow/export-execution-flow.md).
- Serve both HTTP export APIs and [Go CLI export runner](../batch-component/go-cli-export-runner.md) through the same adapter boundary.
- Consume external reference values as stored primary key values according to schema rules.
- Evaluate or consume formula field results from the normalized validated data.
- Filter fields according to schema export flags.
  - Emit backend-specific files or statements.
  - Report export-time diagnostics when target-specific constraints fail.
- Out of scope:
  - Editing canonical YAML.
  - Owning schema validation rules that should live in [Schema validation engine](schema-validation-engine.md).

## Rules / Constraints

- All adapters share the same input model.
- CSV export emits one file per exported table and must define scalar formatting.
- Standard CSV uses UTF-8 without BOM, LF line endings, a header row, comma delimiter, and RFC 4180-style double-quote escaping. Fields containing comma, double quote, or line break are quoted; embedded double quotes are doubled.
- CSV header rows are always emitted.
- CSV column order is primary key columns first, followed by exported schema fields in schema order.
- CSV fields containing line breaks are allowed and are emitted as quoted fields using LF inside the field.
- Standard CSV boolean values are emitted as `true` and `false`.
- Excel CSV boolean values are emitted as `TRUE` and `FALSE`.
- Standard CSV and Excel CSV default temporal formatting is `iso`.
- Temporal `iso` formatting emits date fields as `YYYY-MM-DD`, time fields as `HH:mm:ss` with optional fractional seconds when present, and datetime fields as ISO 8601/RFC 3339 strings.
- Temporal `epoch-sec` and `epoch-ms` formatting may be selected for datetime fields. Date-only and time-only fields remain ISO strings under epoch temporal modes unless a backend explicitly documents a different mapping.
- Temporal `iso-local` formatting converts datetime values to the selected export timezone and emits an ISO 8601 string with offset. Date-only and time-only fields remain ISO strings.
- The export timezone is optional. When omitted, timezone-dependent temporal formats use the runtime local timezone; batch callers should pass an explicit timezone for reproducibility.
- Excel CSV is a separate format intended for direct Excel opening. It uses UTF-8 with BOM and applies Excel-safe handling for string values that Excel would otherwise treat as formulas.
- Excel CSV formula-like string handling prepends a single apostrophe (`'`) to exported string values that begin with `=`, `+`, `-`, or `@` before CSV quoting is applied.
- JSON export emits one file per exported table and preserves JSON scalar types.
- YAML export emits merged export data, not canonical generation folders.
- SQLite export creates a file containing all selected exported data.
- SQL export emits `CREATE TABLE IF NOT EXISTS`, `TRUNCATE`, and `INSERT` statements for a target database dialect; dialect support is backend-specific.
- Excel export creates one worksheet per exported table and may include manifest or diagnostics worksheets.
- NDJSON export is optional and intended for large-data or ingestion workflows.
- Web API downloads may wrap CSV, Excel CSV, JSON, YAML, and NDJSON multi-file outputs in ZIP archives.
- CLI output must not wrap CSV, Excel CSV, JSON, YAML, or NDJSON in ZIP archives; it writes table files to a directory so callers can package them separately.
- Adapters receive already-resolved format options. They must not read or write persisted export settings directly.
- Backend-specific behavior must not change canonical data semantics.
- Adapter output must be host-independent: web API, Wails, and CLI callers receive the same logical artifact for the same normalized input, format, and options.
- Export must be blocked when schema validation reports errors.
- Export should notify users about blocking validation errors before producing target artifacts.
- Export output includes only fields whose schema has `export: true`, plus backend-required primary key fields.
- Fields with `export: false` are omitted even when they are used as formula inputs or lookup labels.
- Formula fields are exported only when their schema has `export: true`.

## Dependencies

- [Generic master data model](../data-model/generic-master-data-model.md)
- [Table schema model](../data-model/table-schema-model.md)
- [Schema validation engine](schema-validation-engine.md)
- [Export settings model](../data-model/export-settings-model.md)
- [Export execution flow](../data-flow/export-execution-flow.md)
- [Go CLI export runner](../batch-component/go-cli-export-runner.md)
- [Go CLI generate runner](../batch-component/go-cli-generate-runner.md)

## Related Documents

- [Product overview](../generic/product-overview.md)

## Native-Language Summary

検証済み・世代統合済みのデータをCSV、Excel CSV、JSON、YAML、SQL、Excel workbook、SQLite、NDJSONなどの実行時データ成果物に変換する境界。標準CSVはUTF-8 BOMなし/LF/ヘッダーあり/RFC 4180相当のダブルクオートエスケープで、boolean は `true/false`。Excel CSVはUTF-8 BOMつきで、boolean は `TRUE/FALSE`、Excelが数式扱いする `=`、`+`、`-`、`@` 始まりの文字列に先頭アポストロフィを付けて安全化する。日時は既定 `iso`、必要に応じて `epoch-sec`、`epoch-ms`、`iso-local` を選べる。Web API のダウンロードでは複数ファイル形式を ZIP に包めるが、Go CLI は ZIP にせず出力ディレクトリへ直接書く。Pongo2 テンプレートによる Go や SQL などのソース生成は `masterdatamate generate` が扱う。Web API、Wails、Go CLI は同じアダプタ境界を使う。外部参照は保存済みの主キー値として扱い、表示名は出力しない。各バックエンド固有の制約はアダプタ側に閉じ込める。
