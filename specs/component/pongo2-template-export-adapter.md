---
id: "pongo2-template-export-adapter"
type: "batch-component"
title: "Pongo2 template export adapter"
aliases: ["template export adapter", "code generation export adapter"]
tags: ["export", "template", "pongo2", "adapter", "code-generation"]
facts:
  lifecycle.status: "blueprint"
---

# Pongo2 template export adapter

## Summary

The Pongo2 template export adapter renders project-defined template export definitions against the same normalized, validated, generation-merged dataset used by ordinary export backends. It is intended for code generation and text artifact generation, such as generating Go constants, error types, localization files, or backend-specific registries from master data.

The adapter is part of the shared export service boundary. HTTP, Wails, and CLI hosts must invoke the same adapter implementation so template parsing, context shape, diagnostics, and path safety are consistent.

## Responsibilities

- Load selected template export definitions from `masterdata/export_definitions.yaml`.
- Load template files from `masterdata/export_templates/`.
- Parse all Pongo2 templates and rendered output path templates before writing artifacts.
- Build deterministic render contexts for `project`, `table`, `record`, and `group` scopes.
- Render text files into a temporary artifact directory.
- Run optional post-render formatters such as `gofmt`.
- Report definition, template, render, path, duplicate-output, and formatter diagnostics.
- Return generated files to the host as a multi-file export artifact.

## Inputs

- Validated merged export dataset from [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md).
- Export-effective schemas and records after table and field export flags are applied.
- Selected definition IDs or the default enabled definition set.
- Resolved template export options from [Export settings model](../data-model/export-settings-model.md) and request or CLI flags.
- Project workspace root.

## Outputs

- One or more generated text files.
- Diagnostics with definition ID, scope, table, record key, output path, severity, and message when available.
- Artifact summary including generated file count, rendered definition count, and skipped definition count.

## Render Pipeline

1. Resolve selected definition IDs.
2. Load `masterdata/export_definitions.yaml`.
3. Validate definition identity, scope, table references, group references, template source, and output path templates.
4. Load external template files.
5. Parse all Pongo2 template sources and output path templates.
6. Build the all-table base context from the normalized export dataset.
7. Expand each definition into render jobs according to its scope.
8. Render each output path.
9. Clean and validate the rendered relative output path.
10. Render file content.
11. Normalize line endings.
12. Run optional formatter.
13. Detect duplicate outputs and apply the definition overwrite policy.
14. Write files to a temporary artifact directory.
15. Return artifact metadata to the host for ZIP packaging, Wails delivery, or CLI atomic directory rename.

## Rules / Constraints

- The adapter must use Pongo2 as the template language.
- Templates are trusted project files but still run without host side effects.
- Template rendering must not expose arbitrary filesystem, process, network, environment, or clock access.
- The only data exposed to templates is the normalized context documented by [Template export definition model](../data-model/template-export-definition-model.md).
- The adapter must not expose Go objects with mutating methods to templates.
- Template parsing for every selected definition must finish before any output file is written.
- Output path rendering must finish for every render job before any output file is written when practical, so duplicate paths are found early.
- Rendered output paths must be relative, clean paths under the output root.
- Generated files must be deterministic for the same workspace content, generation IDs, selected definitions, options, and tool version.
- Definition order in `export_definitions.yaml` is the default render order.
- When multiple definitions render the same output path and no overwrite policy allows it, the adapter reports a blocking diagnostic and writes no artifacts.
- `overwrite: replace` allows a later render job to replace an earlier generated file in the same export run.
- `overwrite: skip` keeps the first generated file and records a warning for skipped later jobs.
- `formatter: gofmt` may be used only for generated Go files. Formatter failure blocks the affected definition when `required: true`.
- The adapter does not create directories outside the export artifact root.
- The adapter does not update `masterdata/export_definitions.yaml` or template files during export.
- Template export respects table `export` flags by default. Definitions targeting `export: false` tables are invalid unless a later explicit internal-export option is specified.
- Template export respects field `export` flags by default. Definitions may opt into non-exported fields only with `include_non_exported_fields: true`.
- External references exposed in contexts use stored primary-key values. Display labels are optional metadata, not canonical values.

## Diagnostics

Diagnostic codes should distinguish:

- Missing or invalid `export_definitions.yaml`.
- Unknown definition ID.
- Duplicate definition ID.
- Missing target table.
- Missing group field.
- Invalid template file path.
- Template parse error.
- Output path template parse error.
- Render error.
- Unsafe rendered output path.
- Duplicate rendered output path.
- Formatter failure.
- Filesystem write failure.

## Dependencies

- [Template export definition model](../data-model/template-export-definition-model.md)
- [Export backend adapters](export-backend-adapters.md)
- [Export execution flow](../data-flow/export-execution-flow.md)
- [Go CLI export runner](../batch-component/go-cli-export-runner.md)

## Native-Language Summary

Pongo2 テンプレート export の実行アダプタ。通常の export と同じ検証済み・世代統合済みデータを入力にし、`masterdata/export_definitions.yaml` と `masterdata/export_templates/` のテンプレートから複数テキストファイルを生成する。出力パスは必ず export 先ディレクトリ配下に制限し、重複パスやテンプレートエラー、`gofmt` エラーは診断として返す。Web/Wails/CLI は同じアダプタを使う。
