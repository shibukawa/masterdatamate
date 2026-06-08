---
id: "go-cli-generate-runner"
type: "batch-component"
title: "Go CLI generate runner"
aliases: ["standalone generate CLI", "template generation runner"]
tags: ["go", "cli", "generate", "template", "batch"]
facts:
  lifecycle.status: "blueprint"
  owner: "application"
---

# Go CLI generate runner

## Summary

The Go CLI generate runner renders project-defined Pongo2 templates into source files or other generated text artifacts. It is intentionally separate from `masterdatamate export`: export writes runtime data artifacts, while generate writes build-time or source-controlled artifacts such as Go files, SQL DDL, registries, or error definitions.

The command reads a MasterDataMate workspace, resolves the requested generation set, validates and merges the data through the same export-effective dataset rules, loads `masterdata/generate_definitions.yaml`, renders selected enabled definitions, and writes files under the configured generation output root.

## Command Shape

```bash
masterdatamate generate \
  --workspace /path/to/project \
  --generations 0000_initial,0010_balance_patch
```

By default, `generate` uses the selected generation definitions' configured output root from `masterdata/generate_definitions.yaml`. Users do not need to pass an output path for normal project generation.

## Flags

| Flag | Required | Description |
| --- | --- | --- |
| `--workspace` | no | Project root containing `masterdata`. When omitted, use the same upward workspace discovery as [Go embedded web server host](../server-component/go-embedded-web-server-host.md). |
| `--generations` | no | Comma-separated generation IDs. When omitted, use every generation whose `_config.yaml` has `output: true`. |
| `--definitions` | no | Comma-separated template definition IDs. When omitted, use `defaults.definition_ids`, then enabled definitions from `masterdata/generate_definitions.yaml`. |
| `--output-root` | no | Override the definition file's configured output root for this run. Intended for CI preview or temporary generation. |
| `--check-only` | no | Validate selected definitions, templates, data, and rendered output paths without writing files. |
| `--mkdirs` | no | Create the output root when it does not exist. |
| `--force-overwrite` | no | Allow replacing existing generated files after validation succeeds. |
| `--json` | no | Print a machine-readable command result JSON object. |

## Result JSON

Successful check-only execution writes:

```json
{
  "generatable": true,
  "generationIds": ["0000_initial"],
  "orderedGenerationIds": ["0000_initial"],
  "outputRoot": "generated",
  "summary": {
    "definitionCount": 4,
    "fileCount": 6,
    "diagnosticCount": 0
  },
  "diagnostics": []
}
```

Successful file generation writes the same shape with `outputRoot` and generated file metadata. Validation failure writes `generatable: false`, non-empty diagnostics, and no file writes.

## Flow

1. Parse command-line flags without prompting.
2. Resolve the workspace root.
3. Resolve generation selection from `--generations`; if omitted, select output-enabled generations.
4. Load and validate `masterdata/generate_definitions.yaml`.
5. Resolve selected definitions from `--definitions`, configured defaults, or enabled definitions.
6. Resolve output root from `--output-root`, then `output_root` in the definition file.
7. Validate generation selection, merge records, and run normal export-effective validation.
8. Parse all selected Pongo2 templates and output path templates.
9. Render all output paths and detect unsafe or duplicate paths.
10. If `--check-only` is set, report diagnostics without writing.
11. Render file contents, normalize line endings, and run optional formatters.
12. Write to a temporary sibling directory under the output root parent.
13. Atomically replace or create generated files only after validation and rendering succeed.
14. Print a concise success message or result JSON.

## Rules / Constraints

- `generate` is non-interactive.
- `generate` must not require `--format`.
- Normal generation must not require an output argument; the project definition owns the relative output root.
- The configured output root is relative to the workspace root.
- `--output-root` is an explicit run-level override and may be absolute only for local preview output; configured project output roots must be relative.
- Generated output paths from definitions are relative to the resolved output root.
- Generated output paths must not be absolute and must not contain path traversal after rendering and cleaning.
- Generation writes source artifacts, not canonical schema YAML, generation YAML, or export settings.
- Failed validation or rendering must not partially update generated files.
- Existing generated files are rejected unless `--force-overwrite` is supplied.
- `generate` uses the same normalized merged dataset and reference validation as export, but it is not an export format.
- `export --format template` is not a new public interface for template generation. New documentation, samples, and UI flows must use `masterdatamate generate`; a legacy compatibility alias may exist only for migration and must share the same generate implementation.
- CLI diagnostics must include definition ID, table, record key, output path, severity, and message when applicable.

## Test Requirements

- `masterdatamate generate --workspace examples/templates --check-only --json` validates the sample project without writing.
- `masterdatamate generate --workspace examples/templates --force-overwrite --json` writes the configured generated files without requiring `--output`.
- `--definitions` limits generation to the requested definitions.
- `--output-root` writes to an alternate destination without changing `masterdata/generate_definitions.yaml`.
- Failed template parsing, unsafe paths, duplicate paths, and formatter failures write no generated files.

## Related Requirements

- [Template export definition model](../data-model/template-export-definition-model.md)
- [Pongo2 template export adapter](../component/pongo2-template-export-adapter.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)
- [Schema validation engine](../component/schema-validation-engine.md)
- [Go embedded web server host](../server-component/go-embedded-web-server-host.md)

## Native-Language Summary

`masterdatamate generate` は Pongo2 テンプレートから Go や SQL などの生成ファイルを作るためのCLI。通常のデータ出力である `export` とは分ける。出力先は `masterdata/generate_definitions.yaml` の `output_root` を使うため、普段は `--output` を指定しない。CIや一時確認では `--output-root` で上書きできる。生成前にデータ検証、テンプレート構文、出力パス安全性、重複パス、formatter を確認し、失敗時はファイルを書かない。
