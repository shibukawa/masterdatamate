---
id: "export-settings-model"
type: "data-model"
title: "Export settings model"
aliases: ["export defaults", "export format settings"]
tags: ["export", "settings", "format"]
facts:
  lifecycle.status: "blueprint"
  data.name: "export-settings"
---

# Export settings model

## Summary

Export settings store project-level defaults for export formats and their options. They allow the GUI to remember the last chosen options for each export format and allow the Go CLI to use the same defaults when corresponding flags are omitted.

The settings file is stored under the workspace's canonical data root as `masterdata/export_settings.yaml`. It is project-local so browser, Wails, and batch execution can resolve the same defaults from the same workspace.

## Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| version | integer | yes | Settings schema version. The first version is `1`. |
| formats | map | yes | Map keyed by logical export format ID. |
| formats.<format>.time_format | enum | no | `iso`, `epoch-sec`, `epoch-ms`, or `iso-local`. Defaults to `iso`. |
| formats.<format>.timezone | string | no | Optional IANA timezone such as `Asia/Tokyo`. Used by timezone-dependent temporal formats. |
| formats.<format>.include_schema | boolean | no | Optional manifest/schema emission preference for formats that support it. |
| formats.<format>.include_diagnostics_sheet | boolean | no | Optional Excel workbook diagnostics worksheet preference. |
| formats.<format>.sql_dialect | string | no | Optional SQL dialect for `sql`; defaults to the generic SQL dialect specified by the adapter. |
| formats.<format>.updated_at | string | no | Optional ISO 8601 timestamp for display or conflict diagnostics. |

Logical format IDs are host-neutral: `csv`, `excel-csv`, `json`, `yaml`, `ndjson`, `sql`, `xlsx`, and `sqlite`. HTTP ZIP download IDs map to these logical IDs before settings lookup:

| HTTP format | Logical settings format |
| --- | --- |
| `csv_zip` | `csv` |
| `excel_csv_zip` | `excel-csv` |
| `json_zip` | `json` |
| `yaml_zip` | `yaml` |
| `ndjson_zip` | `ndjson` |

Example:

```yaml
version: 1
formats:
  csv:
    time_format: iso
  excel-csv:
    time_format: iso-local
    timezone: Asia/Tokyo
  sql:
    sql_dialect: generic
    time_format: iso
```

## Resolution Order

Export option resolution is shared by HTTP, Wails, and CLI hosts:

1. Explicit request options or CLI flags.
2. Persisted settings for the selected logical export format.
3. Built-in format defaults from [Export backend adapters](../component/export-backend-adapters.md).

Only recognized options for the selected format are applied. Unknown settings keys are ignored for export execution but should be preserved when rewriting the settings file when practical.

## GUI Persistence

The export dialog loads effective settings for each supported format when it opens. When the user changes the selected format, the dialog initializes option controls from that format's persisted settings plus built-in defaults.

When the user runs export check or export from the GUI, the selected format options are saved to `masterdata/export_settings.yaml` before or together with the request. The next GUI export uses those saved values as initial control values for the same logical format.

GUI persistence stores export options, not generated artifacts. It must not write schema YAML, generation YAML, or table data. Absolute output destinations should not be persisted in the project-level settings unless a separate host-local preference model is specified later.

## CLI Behavior

The Go CLI reads `masterdata/export_settings.yaml` during option resolution. If `--time-format`, `--timezone`, or another format option is omitted, the CLI uses the selected format's persisted setting before falling back to built-in defaults.

The export subcommand does not mutate export settings as part of normal artifact creation. Batch jobs remain reproducible when they pass explicit flags, and they can intentionally change project defaults only through a separately specified settings command or by editing the YAML file.

If the settings file is missing, the CLI and GUI use built-in defaults. If the file exists but is invalid YAML or has invalid option values, export settings loading reports a configuration diagnostic and the export request fails before artifact generation.

## Rules / Constraints

- `masterdata/export_settings.yaml` is optional for existing projects.
- Missing per-format settings are equivalent to an empty settings object for that format.
- Format aliases are normalized before settings lookup.
- Settings are keyed by logical format, not by HTTP packaging format.
- GUI writes settings for the logical format selected by the user.
- CLI flags and HTTP request options always override persisted settings.
- Persisted `timezone` is optional; timezone-dependent formats use the runtime local timezone only when neither an explicit option nor a persisted timezone exists.
- Settings reads and writes must be atomic from the user's perspective.
- Settings updates must not reorder or rewrite canonical table records.
- Settings changes are project data changes and may appear in version control diffs.
- Template generation definition content, default definition selection, and generation output root are stored in `masterdata/generate_definitions.yaml`, not inside export settings.

## Dependencies

- [Canonical YAML file layout](canonical-yaml-file-layout.md)
- [Export backend adapters](../component/export-backend-adapters.md)
- [Template export definition model](template-export-definition-model.md)
- [Export execution flow](../data-flow/export-execution-flow.md)
- [Go CLI export runner](../batch-component/go-cli-export-runner.md)
- [Go CLI generate runner](../batch-component/go-cli-generate-runner.md)

## Native-Language Summary

エクスポート設定は `masterdata/export_settings.yaml` に保存するプロジェクト単位の既定値。GUI は形式ごとに日時形式や timezone などのオプションを読み込み、ユーザーが操作した値を保存して次回の初期値にする。CLI は `--time-format` や `--timezone` などが省略された場合に同じ設定を読み、明示フラグがあればそれを優先する。HTTP の `csv_zip` などは保存時・参照時に `csv` のような論理形式へ正規化する。テンプレート生成の本文、scope、既定の definition 選択、出力rootは `masterdata/generate_definitions.yaml` に置く。
