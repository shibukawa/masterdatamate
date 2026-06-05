---
id: "go-cli-export-runner"
type: "batch-component"
title: "Go CLI export runner"
aliases: ["standalone export CLI", "batch export runner"]
tags: ["go", "cli", "export", "batch"]
facts:
  lifecycle.status: "blueprint"
  owner: "application"
---

# Go CLI export runner

## Summary

The Go CLI export runner lets users run the same checked export workflow without starting the web UI. It is intended for local scripts, CI, release builds, and scheduled batch jobs. The command reads a MasterDataMate workspace, validates the requested generation set, merges records, runs strict pre-export validation, and writes an export artifact to a filesystem destination.

The CLI must use the same Go export service as the packaged web server and desktop hosts. It must not reimplement export semantics separately from [Export execution flow](../data-flow/export-execution-flow.md) or [Export backend adapters](../component/export-backend-adapters.md).

## Actors

- Developer.
- Build or release automation.
- CI job.
- Scheduled batch process.
- Non-engineering editor running a packaged binary from a terminal.

## Components

- [Export execution flow](../data-flow/export-execution-flow.md)
- [Export backend adapters](../component/export-backend-adapters.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)
- [Schema validation engine](../component/schema-validation-engine.md)
- [Export settings model](../data-model/export-settings-model.md)
- [Template export definition model](../data-model/template-export-definition-model.md)
- [Pongo2 template export adapter](../component/pongo2-template-export-adapter.md)
- [Go embedded web server host](../server-component/go-embedded-web-server-host.md)

## Command Shape

The packaged executable supports a standalone export subcommand:

```bash
masterdatamate export \
  --workspace /path/to/project \
  --generations 0000_initial,0010_balance_patch \
  --format csv \
  --output /path/to/out/exported-masterdata
```

The same executable may keep the existing no-subcommand server behavior for backward compatibility. New command dispatch should treat `serve` as the explicit web-server subcommand and no subcommand as an alias for `serve` until a breaking CLI version is intentionally introduced.

## Flags

| Flag | Required | Description |
| --- | --- | --- |
| `--workspace` | no | Project root containing `masterdata`. When omitted, use the same upward workspace discovery as [Go embedded web server host](../server-component/go-embedded-web-server-host.md). |
| `--generations` | no | Comma-separated generation IDs to export. Request order does not define precedence. When omitted, use every generation whose `_config.yaml` has `output: true`. |
| `--format` | yes | CLI export format. Uses unpacked identifiers for multi-file formats: `csv`, `excel-csv`, `json`, `yaml`, `ndjson`, and `template`, plus single-artifact formats `sql`, `xlsx`, and `sqlite`. ZIP packaging is intentionally not a CLI format; callers can run a separate zip command when needed. |
| `--output` | yes unless `--check-only` is set | Artifact destination path. For `csv`, `excel-csv`, `json`, `yaml`, `ndjson`, and `template`, this is an output directory. For `sql`, `xlsx`, and `sqlite`, this is an output file. Parent directories must already exist unless `--mkdirs` is supplied. |
| `--mkdirs` | no | Create missing parent directories for `--output`. |
| `--check-only` | no | Run validation and print diagnostics without writing an artifact. |
| `--diagnostics-format` | no | `text` or `json`; defaults to `text` for terminals and `json` when `--json` is set. |
| `--diagnostics-output` | no | Optional path for diagnostics. If omitted, diagnostics go to stderr for text output or stdout for `--json`. |
| `--time-format` | no | Temporal formatting for CSV-like exports. Supported values are `iso`, `epoch-sec`, `epoch-ms`, and `iso-local`. When omitted, use the selected format's persisted export setting, then built-in default `iso`. |
| `--timezone` | no | IANA timezone name such as `Asia/Tokyo` used by timezone-dependent temporal formatting, especially `iso-local`. When omitted, use the selected format's persisted export setting, then the runtime local timezone. |
| `--template-definitions` | no | Comma-separated template export definition IDs used when `--format template` is selected. When omitted, use `formats.template.definition_ids` from export settings, then enabled definitions from `masterdata/export_definitions.yaml`. |
| `--json` | no | Print a machine-readable command result JSON object. |
| `--force-overwrite` | no | Allow replacing an existing output file after validation passes. |

## Result JSON

When `--json` is set, successful check-only execution writes:

```json
{
  "exportable": true,
  "format": "csv",
  "generationIds": ["0000_initial", "0010_balance_patch"],
  "orderedGenerationIds": ["0000_initial", "0010_balance_patch"],
  "summary": {
    "tableCount": 3,
    "recordCount": 182,
    "diagnosticCount": 0
  },
  "diagnostics": []
}
```

Successful artifact creation writes:

```json
{
  "exportable": true,
  "format": "csv",
  "output": "/path/to/out/exported-masterdata",
  "outputKind": "directory",
  "generationIds": ["0000_initial", "0010_balance_patch"],
  "orderedGenerationIds": ["0000_initial", "0010_balance_patch"],
  "summary": {
    "tableCount": 3,
    "recordCount": 182,
    "diagnosticCount": 0
  },
  "diagnostics": []
}
```

Validation failure writes the same shape with `exportable: false`, non-empty diagnostics, and no artifact write.

## Flow

1. Parse command-line flags without prompting.
2. Resolve the workspace root.
3. Resolve generation selection from `--generations`; if omitted, select output-enabled generations from generation metadata.
4. Load persisted export settings from `masterdata/export_settings.yaml` if present.
5. Resolve format options from explicit CLI flags, selected format settings, and built-in defaults.
6. Validate the requested format and output destination.
7. Run the same pre-export check as [Export execution flow](../data-flow/export-execution-flow.md).
8. If diagnostics contain blocking errors, print diagnostics and exit without writing the artifact.
9. If `--check-only` is set, exit after reporting the check result.
10. Generate the artifact using the shared export adapter boundary.
11. For file output formats, write to a temporary file in the destination directory.
12. For directory output formats, write to a temporary sibling directory and populate one file per exported table plus any manifest file.
13. Atomically rename the temporary file or temporary directory to `--output`.
14. Print a concise success message or result JSON.

## Rules / Constraints

- CLI export is non-interactive.
- CLI export must be deterministic for the same workspace content, generation IDs, format, options, and output path.
- CLI multi-file formats are unpacked directory outputs; the CLI must not create ZIP archives for `csv`, `excel-csv`, `json`, `yaml`, or `ndjson`.
- CLI template export is an unpacked directory output; the CLI must not create ZIP archives for `template`.
- CLI ZIP packaging is out of scope because callers can run a separate `zip` command over the output directory.
- CLI export uses output-enabled generations from generation metadata when `--generations` is omitted.
- CLI export reads persisted project export settings when format options are omitted.
- CLI template export reads `masterdata/export_definitions.yaml` and template files under `masterdata/export_templates/`.
- `--template-definitions` is valid only with `--format template`.
- CLI option resolution order is explicit flags, persisted settings for the selected logical format, then built-in adapter defaults.
- The export subcommand must not mutate `masterdata/export_settings.yaml` during normal artifact creation.
- Batch jobs may still pass `--generations` explicitly when they require a pinned generation set independent of metadata changes.
- Batch jobs should pass explicit format flags when they require output independent of project-level export settings changes.
- `--generations` values are sorted by configured generation ordering before merge; request order is not merge precedence.
- Default output-enabled generation IDs are also sorted by configured generation ordering before merge.
- If no generations are output-enabled and `--generations` is omitted, the command fails with a usage/configuration error and writes no artifact.
- CLI export always runs pre-export validation before artifact generation.
- Validation errors block artifact writing.
- Failed validation must not create, truncate, or partially overwrite the output file.
- Existing output files are rejected unless `--force-overwrite` is supplied.
- Existing output directories are rejected unless `--force-overwrite` is supplied.
- Artifact writes must be atomic from the user's perspective by writing a temporary file or temporary directory and renaming it after success.
- Directory output formats must use deterministic file names based on table system names, such as `product.csv`, `product.json`, `product.yaml`, or `product.ndjson`.
- Template output file names are determined by each selected template export definition's `output_path` template and must pass the same relative path safety checks as HTTP export.
- `csv` writes UTF-8 without BOM, LF line endings, mandatory header rows, comma delimiters, RFC 4180-style double-quote escaping, and boolean values as `true` or `false`.
- `excel-csv` writes `.csv` table files with UTF-8 BOM, boolean values as `TRUE` or `FALSE`, and prepends a single apostrophe to string values beginning with `=`, `+`, `-`, or `@` before CSV quoting.
- `--time-format=iso` emits date fields as `YYYY-MM-DD`, time fields as `HH:mm:ss` with optional fractional seconds when present, and datetime fields as ISO 8601/RFC 3339 strings.
- `--time-format=epoch-sec` and `--time-format=epoch-ms` emit datetime fields as Unix seconds or milliseconds. Date-only and time-only fields remain ISO strings.
- `--time-format=iso-local` converts datetime fields to `--timezone` when supplied, otherwise to the runtime local timezone, and emits an ISO 8601 string with offset.
- Batch jobs should pass `--timezone` explicitly when using `iso-local` so output does not depend on the machine locale.
- CLI diagnostics must include table, generation, record key, field, severity, and message when applicable.
- CLI export must not start the web server, open a browser, or require frontend assets.
- CLI export must not write canonical schema YAML, generation YAML, or generation config.
- CLI export may be called from cron or CI without a TTY.
- The Go web API, Wails host, and CLI must share export service code so behavior drift is testable and minimized.

## Exit Codes

| Code | Meaning |
| --- | --- |
| `0` | Export check or artifact creation succeeded. |
| `1` | Pre-export validation found blocking diagnostics. |
| `2` | Command-line usage error, such as missing `--format` or generation selection. |
| `3` | Workspace or canonical data loading error. |
| `4` | Output path, permission, or artifact write error. |
| `5` | Requested format is recognized but not implemented. |

## Error Cases

- Missing `--format`.
- Omitted `--generations` when no generation has `output: true`.
- Unknown generation ID.
- Duplicate generation ID.
- Invalid generation `_config.yaml`.
- Requested format is unknown or recognized but not implemented.
- Output parent directory does not exist and `--mkdirs` is not set.
- Output file or directory exists and `--force-overwrite` is not set.
- Export validation fails, including missing external reference targets in the selected effective export dataset.
- Filesystem write or atomic rename fails.

## Test Requirements

- CLI and HTTP export check return equivalent diagnostics for the same workspace, generation set, format, and options.
- CLI unpacked directory outputs and HTTP ZIP outputs produce equivalent logical table files for the same input.
- `--check-only` never writes an artifact.
- Failed validation does not modify an existing output file or directory.
- `--force-overwrite` replaces an existing artifact path only after validation succeeds.
- Omitted `--generations` selects exactly the output-enabled generations from generation metadata.
- Exit codes distinguish validation failure from usage and write failures.

## Related Requirements

- [Product overview](../generic/product-overview.md)
- [Export execution flow](../data-flow/export-execution-flow.md)
- [Export backend adapters](../component/export-backend-adapters.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)
- [Schema validation engine](../component/schema-validation-engine.md)
- [Export settings model](../data-model/export-settings-model.md)
- [Template export definition model](../data-model/template-export-definition-model.md)
- [Pongo2 template export adapter](../component/pongo2-template-export-adapter.md)
- [Go embedded web server host](../server-component/go-embedded-web-server-host.md)
- [Wails desktop host](../server-component/wails-desktop-host.md)

## Native-Language Summary

Go CLI の `masterdatamate export` は、Web UI を起動せずに同じ export チェックと成果物生成を実行する。CLI では CSV/Excel CSV/JSON/YAML/NDJSON/template を ZIP に固めず、出力ディレクトリへ table ごとのファイルまたはテンプレート生成ファイルとして直接書く。ZIP が必要な場合は利用者が別途 `zip` コマンドを実行する。`--format template` では `masterdata/export_definitions.yaml` と `masterdata/export_templates/` を読み、`--template-definitions` で選択定義を指定できる。標準CSVはUTF-8 BOMなし/LF/ヘッダーあり/RFC 4180相当で boolean は `true/false`、`excel-csv` はBOMつきで boolean は `TRUE/FALSE`、`=`、`+`、`-`、`@` 始まりの文字列に先頭アポストロフィを付ける。日時は `--time-format` で `iso`、`epoch-sec`、`epoch-ms`、`iso-local` を選べ、`--timezone` は任意。これらの形式別オプションが省略された場合は `masterdata/export_settings.yaml` の保存済み設定を読み、明示フラグがあればそれを優先する。`--generations` が省略された場合は generation 設定の `output: true` 世代を対象にし、固定したいバッチ処理では `--generations` で対象世代を明示できる。事前検証に失敗した場合は成果物を書かない。Web API、Wails、CLI は同じ Go export service を共有して、フォーマット出力と診断の差分を防ぐ。
