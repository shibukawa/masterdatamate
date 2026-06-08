---
id: "template-export-definition-model"
type: "data-model"
title: "Template generation definition model"
aliases: ["pongo2 generation definitions", "code generation definitions", "template export definitions"]
tags: ["generate", "template", "pongo2", "code-generation"]
facts:
  lifecycle.status: "blueprint"
  data.name: "template-export-definition"
---

# Template generation definition model

## Summary

Template generation definitions describe project-local Pongo2-based generation jobs that render validated merged master data into arbitrary text files. They are separate from table schema definitions so schema authors can change output artifacts, code generation templates, file names, and grouping rules without changing canonical table structure.

The definitions are stored in `masterdata/generate_definitions.yaml`. Template source may be inline in that YAML for small templates or referenced from files under `masterdata/generate_templates/` for reviewable larger templates.

## Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| version | integer | yes | Definition schema version. The first version is `1`. |
| output_root | path | yes | Default generation output root relative to the workspace root. `masterdatamate generate` writes rendered files under this root when `--output-root` is omitted. |
| defaults | object | no | Project-level default generation selection and behavior. |
| defaults.definition_ids | array | no | Default definition IDs for `generate`. When omitted, enabled definitions are used. |
| definitions | array | yes | Ordered generation definition list edited by the generation definition screen. |
| definitions.id | string | yes | Stable ASCII identifier used by GUI, API, and CLI selection. |
| definitions.name | string | yes | Human-facing label. |
| definitions.enabled | boolean | yes | Whether this definition is selected by default when users run all template generation. |
| definitions.description | string | no | Human-facing note for maintainers. |
| definitions.scope | enum | yes | `project`, `table`, `record`, or `group`. |
| definitions.table | string | conditional | Required for `table`, `record`, and `group` scopes. References a table schema `system_name`. |
| definitions.group_by | object | conditional | Required for `group` scope. Declares grouping input. |
| definitions.group_by.table | string | no | Table whose records are grouped. Defaults to `definitions.table`. |
| definitions.group_by.field | string | yes | Field used as the grouping key. May be an external reference field. |
| definitions.group_by.related_tables | array | no | Extra tables to include in each group context after reference-aware filtering. |
| definitions.template | string | conditional | Inline Pongo2 template. Mutually exclusive with `template_file`. |
| definitions.template_file | path | conditional | Relative path under `masterdata/generate_templates/` or a project-local path under `masterdata/`. Mutually exclusive with `template`. |
| definitions.output_path | string | yes | Pongo2 template for output path relative to the resolved generation output root. |
| definitions.overwrite | enum | no | `error`, `replace`, or `skip`; defaults to `error`. |
| definitions.formatter | enum | no | Optional post-render formatter. First supported value is `gofmt`. |
| definitions.line_ending | enum | no | `lf` or `native`; defaults to `lf` for deterministic output. |
| definitions.include_non_exported_fields | boolean | no | Defaults to `false`. When false, record field maps expose only primary keys and `export: true` fields. |
| definitions.required | boolean | no | When true, failure blocks the template generation set. Defaults to `true`. |

Example:

```yaml
version: 1
output_root: generated
definitions:
  - id: go_error_constants
    name: Go error constants
    enabled: true
    scope: table
    table: error_message
    template_file: go/error_constants.go.pongo2
    output_path: "{{ table.system_name }}_errors.go"
    formatter: gofmt
  - id: go_error_type_per_record
    name: Go error type per record
    enabled: true
    scope: record
    table: error_message
    template_file: go/error_type.go.pongo2
    output_path: "errors/{{ record.code|pascal }}.go"
    formatter: gofmt
  - id: errors_by_domain
    name: Domain grouped errors
    enabled: true
    scope: group
    table: error_message
    group_by:
      field: domain_id
      related_tables: [error_domain]
    template_file: go/domain_errors.go.pongo2
    output_path: "errors/{{ group.key|go_ident }}_errors.go"
    formatter: gofmt
```

## Render Scopes

`project` renders one file from the full export-effective dataset. It is useful for manifests, registries, or files that coordinate multiple tables.

`table` renders one file for one table. The template receives the selected table schema, normalized records for that table, and relation metadata for referenced or referencing tables.

`record` renders once per record in the selected table. The template receives the selected record, the table schema, and scoped relation helpers. Its `output_path` must vary by record key unless the definition intentionally uses `overwrite: replace` or `skip`.

`group` renders once per derived group. The first slice groups records by a scalar field value or external-reference primary-key value. When `group_by.field` is an external reference, the group key is the stored referenced primary key and the group label may use the referenced record's `name` when available.

## Pongo2 Context

Every render receives a deterministic context object:

| Name | Scope | Notes |
| --- | --- | --- |
| project | all | Workspace-level metadata and export timestamp when supplied by the caller. |
| generation_ids | all | Requested generation IDs. |
| ordered_generation_ids | all | Generation IDs sorted by generation ordering. |
| tables | all | Map of table ID to table context for every exportable table in the merged dataset. |
| schemas | all | Map of table ID to normalized schema context. |
| definition | all | Current template generation definition. |
| table | table, record, group | Current table context. |
| records | table, group | Normalized records in current scope. |
| record | record | Current normalized record. |
| group | group | Group metadata: `key`, `label`, `field`, `table`, `records`, and `related`. |

Normalized records expose primary key fields as top-level properties and exported non-key fields as top-level properties. They also expose `_key`, `_name`, and `_table` metadata. Internal provenance fields are not exposed unless a later option explicitly adds them.

When `include_non_exported_fields` is `false`, non-exported fields are omitted from top-level record maps even if they exist in canonical YAML. Validation and grouping may still read those fields internally when needed.

## Built-In Filters

The template runtime provides Pongo2's built-in filters plus project-defined filters for code generation:

| Filter | Purpose |
| --- | --- |
| `pascal` | Convert a string to PascalCase. |
| `camel` | Convert a string to camelCase. |
| `snake` | Convert a string to snake_case. |
| `kebab` | Convert a string to kebab-case. |
| `go_ident` | Convert a value into a valid exported or unexported Go identifier according to the filter option. |
| `go_string` | Quote a value as a Go string literal. |
| `quote` | Emit a quoted text literal for the selected backend. |
| `indent` | Indent every non-empty line by a caller-supplied width. |
| `comment` | Wrap text as line comments for the selected language. The first implementation supports Go `//` comments. |
| `default` | Provide a fallback for missing or empty values. |

Filters must be deterministic and must not perform filesystem, network, time, random, or host API access.

## Output Root

`output_root` is the normal destination for generated files. It is project-local and relative to the workspace root, such as `generated`, `internal/generated`, or `build/masterdata`. The CLI may accept `--output-root` as an explicit run-level override for CI previews or temporary generation, but the saved definition file remains the source of truth for normal generation.

Rendered `output_path` values are always relative to the resolved output root. A definition must not render files outside that root.

## Rules / Constraints

- Template generation definitions are project data, not table schemas.
- `masterdata/generate_definitions.yaml` is optional for existing projects.
- Missing `masterdata/generate_definitions.yaml` means no project-defined template generation jobs exist.
- Template definitions must not change canonical schema YAML or generation YAML.
- `id` values are unique within one `generate_definitions.yaml`.
- Template file paths must stay under `masterdata/generate_templates/` unless a later trusted-workspace option explicitly allows another project-local directory.
- `output_root` is required for generation and must be a relative project path.
- Output paths are always relative paths under the resolved generation output root.
- Output paths must not be absolute and must not contain `..` path traversal after template rendering and cleaning.
- The renderer must reject duplicate output paths within one generate run unless the later definition's `overwrite` policy allows it.
- Pongo2 templates must be parsed before any output file is written.
- Template rendering failures produce generation diagnostics with the definition ID and scope item when available.
- Generated text uses LF line endings by default.
- `formatter: gofmt` runs after rendering and before writing. Formatter failure is a blocking diagnostic for that file.
- Template definitions may depend on table schemas and external reference metadata, but they must not add new schema constraints.
- Template definitions may use external reference grouping for output organization, but reference validation remains owned by the normal export validation flow.
- The UI should preserve unknown YAML keys when rewriting definitions when practical, but unknown keys are ignored by execution.

## Dependencies

- [Canonical YAML file layout](canonical-yaml-file-layout.md)
- [Table schema model](table-schema-model.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)
- [Export execution flow](../data-flow/export-execution-flow.md)
- [Pongo2 template export adapter](../component/pongo2-template-export-adapter.md)
- [Template export definition screen](../ui-screen/template-export-definition-screen.md)
- [Go CLI generate runner](../batch-component/go-cli-generate-runner.md)

## Native-Language Summary

Pongo2 を使った任意テキスト生成用の定義。スキーマとは別に `masterdata/generate_definitions.yaml` で管理し、テンプレート本文はインラインまたは `masterdata/generate_templates/` 配下のファイルで持つ。`masterdatamate generate` は定義ファイルの `output_root` を既定出力先として使うため、通常は毎回出力先を指定しなくてよい。scope は project/table/record/group。table 単位で1ファイル、record 単位で複数ファイル、外部キーなどの field で group 化して1グループ1ファイル、といった生成に対応する。出力パスも Pongo2 で組み立てるが、必ず generation output root 配下に制限する。
