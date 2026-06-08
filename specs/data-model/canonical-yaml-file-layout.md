---
id: "canonical-yaml-file-layout"
type: "data-model"
title: "Canonical YAML file layout"
aliases: []
tags: ["yaml", "storage", "schema", "generation"]
facts:
  lifecycle.status: "blueprint"
  data.name: "canonical-yaml-layout"
---

# Canonical YAML file layout

## Summary

Canonical master data lives under `masterdata/`. Schemas live under `masterdata/schema`, generation data lives under `masterdata/generations`, project-level export defaults may live in `masterdata/export_settings.yaml`, and project-defined template generation settings may live in `masterdata/generate_definitions.yaml` plus `masterdata/generate_templates/`. The default data file pattern is one YAML file per table, but a parent table file may also embed records for dependent child tables.

## Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| schema_root | path | yes | `masterdata/schema`. Contains table schema definitions. |
| generations_root | path | yes | `masterdata/generations`. Contains generation datasets. |
| export_settings_file | path | no | `masterdata/export_settings.yaml`. Contains project-level defaults for export formats. |
| generate_definitions_file | path | no | `masterdata/generate_definitions.yaml`. Contains project-level Pongo2 template generation definitions. |
| generate_templates_root | path | no | `masterdata/generate_templates`. Contains external Pongo2 template files referenced by generation definitions. |
| generation_id | string | yes | Generation folder or generation key under `masterdata/generations`. |
| generation_metadata | object | yes | Generation index, output flag, path name, description, and derived folder naming settings. |
| table_file | path | yes | YAML file named after the primary table, such as `org.yaml`. |
| table_entry | sequence | yes | YAML top-level key matching the table name, such as `org:`. |
| record_order | sequence order | yes | Record order in the YAML sequence is preserved by editing APIs. |
| record.key | scalar or object | yes | Primary key value. Composite keys use an object. |
| record.name | string | no | Human-readable record label. |
| record.data | object | yes | Non-key schema fields and ordinary data values. |
| record.children | sequence | no | Dependent child table records nested under the parent record. |
| child.table | string | yes | Dependent child table name. |
| child.key | scalar or object | yes | Child table primary key value. |
| child.name | string | no | Human-readable child record label. |
| child.data | object | yes | Child table non-key schema fields. |
| embedded_table | object | no | Dependent child table records nested through `children`. |
| parent_table | string | conditional | Required for an embedded table. |
| parent_foreign_key | field mapping | conditional | Required for an embedded table and maps child foreign key fields to parent primary key fields. |

## Default Layout

```text
masterdata/
  export_settings.yaml
  generate_definitions.yaml
  generate_templates/
    go/
      error_constants.go.pongo2
  schema/
    org.yaml
    user.yaml
  generations/
    0000_initial/
      _config.yaml
      org.yaml
      user.yaml
```

The first runnable slice uses the fixed `masterdata/generations/0000_initial` folder. The layout keeps the generation directory boundary so later multi-generation support does not require data migration.

## Data Root Discovery

The canonical data root is always the `masterdata/` directory under a resolved workspace root. Runtime discovery must not derive the data root from the native binary path, bundled frontend asset path, npm wrapper path, or `dist-native/` output directory.

When the user does not pass an explicit workspace, the host starts at the process current working directory and walks toward the filesystem root. Each directory is checked for project root markers such as `go.mod`, `.git`, `package.json`, or configured equivalents. The first marked project root that contains `masterdata/` becomes the workspace root. The loader then reads schemas from `<workspace>/masterdata/schema` and generation data from `<workspace>/masterdata/generations`.

If the current working directory is a child of the project, the same upward search applies. If it is already under `masterdata/`, the containing project root is still selected, not the nested `masterdata` subdirectory. Directly launching a packaged binary from `dist-native/masterdatamate` therefore works when the shell current working directory is the project root or one of its descendants.

An explicit workspace path bypasses upward discovery but must resolve to a directory that contains `masterdata/`. Invalid or missing `masterdata/schema` and `masterdata/generations` paths are startup-blocking errors for hosts that read project data.

## Table-Per-File Pattern

In the default pattern, each table has one YAML file per generation:

```text
masterdata/generations/0000_initial/_config.yaml
masterdata/generations/0000_initial/org.yaml
masterdata/generations/0000_initial/user.yaml
```

Each file stores records for one table. The file name should match the table name unless a schema explicitly configures an alternate file mapping.

## Generation Config File

Each generation folder contains `_config.yaml` for generation metadata.

```yaml
generation_index: 10
output: true
path_name: base
description: Initial data set
```

The first runnable slice uses `0000_initial` regardless of later folder derivation settings. In later numeric mode with four digits, a folder may be named with a prefix such as `0010_base`.

Example:

```yaml
org:
  - key: org-001
    name: Product Team
    data:
      description: Core product planning group
  - key:
      region: jp
      org_id: org-002
    name: Japan LiveOps
    data:
      description: Japan live operations group
```

`key` is a scalar for a single-column primary key and an object for a composite primary key. `data` contains ordinary fields other than the primary key fields and reserved metadata fields.

## Persistent Merge Output

A persistent generation merge creates a new folder under `masterdata/generations` using the same derived folder naming rules as manual generation creation.

Example:

```text
masterdata/generations/0020_merged_balance/
  _config.yaml
  org.yaml
  user.yaml
```

The destination `_config.yaml` stores only the destination generation metadata:

```yaml
generation_index: 20
output: true
path_name: merged_balance
description: Merged baseline and balance patch
```

Persistent merge provenance is not written into table records by default. The destination table YAML files contain ordinary canonical records so the newly created generation can be edited like any other generation.

## Generation Duplication Output

A generation duplication creates a new folder under `masterdata/generations` by copying one selected source generation and writing destination `_config.yaml` metadata.

Example:

```text
masterdata/generations/0020_balance_experiment/
  _config.yaml
  org.yaml
  user.yaml
```

The source generation folder is not modified. The destination generation can be edited like any other generation after creation.

## Generation Deletion

Deleting a generation removes its folder under `masterdata/generations`.

Example:

```text
masterdata/generations/0010_balance_patch/
```

Deletion is folder-scoped. It must not delete `masterdata/schema`, non-selected generation folders, or paths outside `masterdata/generations`.

## Embedded Dependent Table Pattern

A parent table file may include records for dependent child tables when the child records are naturally maintained with the parent record.

Example: `user` is dependent on `org`, so `masterdata/generations/base/org.yaml` can contain `org` records and nested `user` records for each organization.

```yaml
org:
  - key: org-001
    name: Product Team
    data:
      description: Core product planning group
    children:
      - table: user
        key: user-001
        name: Alice
        data:
          role: planner
      - table: user
        key: user-002
        name: Bob
        data:
          role: developer
```

Each item in `children` declares the child `table`, child `key`, optional child `name`, and child `data`. The `user` schema must declare that it is dependent on `org` and must declare the foreign key fields that reference the parent `org` primary key.

## Rules / Constraints

- Runtime data discovery starts from the process current working directory unless the user supplies an explicit workspace path.
- Runtime data discovery must not use the executable path, npm wrapper path, embedded asset path, or build output directory as the implicit workspace.
- Upward discovery stops at the first project root marker that contains `masterdata/`.
- A discovered workspace root must expose `masterdata/schema` and `masterdata/generations`.
- `masterdata/schema` is the canonical schema root.
- `masterdata/generations` is the canonical generation data root.
- `masterdata/export_settings.yaml` is the optional canonical project-level export settings file.
- `masterdata/generate_definitions.yaml` is the optional canonical project-level template generation definition file.
- `masterdata/generate_templates` is the optional canonical template source directory for external template files.
- The first runnable slice reads and writes only `masterdata/generations/0000_initial`.
- Each generation folder contains `_config.yaml`.
- Missing or invalid `_config.yaml` is a blocking error; the table editor must not open.
- A generation folder may contain only tables changed in that generation.
- A missing table YAML file in a generation is normal and means the table has no records in that generation yet.
- When editing a missing table YAML file, the UI opens an empty table and the server creates the file on first commit.
- Generation folders are derived from generation metadata, including sortable index and path name.
- Numeric generation folders should include a zero-padded prefix when the project uses numeric ordering.
- Date-based generation folders should begin with a sortable date prefix when the project uses release-date ordering.
- Table-per-file YAML remains the default and simplest storage pattern.
- A canonical table data file uses a top-level `<table>:` key whose value is a sequence of records.
- The order of records in that sequence is canonical and must be preserved.
- Commit-mode row operations modify the sequence by index rather than replacing the whole file.
- Every record uses the reserved keys `key`, `name`, `data`, and optionally `children`.
- `key` is scalar for a single primary key and object-shaped for composite primary keys.
- `data` contains ordinary schema fields and should not duplicate reserved record metadata unless a schema explicitly maps those fields.
- `children` contains dependent table records, each with `table`, `key`, optional `name`, `data`, and optional nested `children` if multi-level dependencies are supported.
- Embedded dependent table records are allowed only for tables declared as dependent tables in schema.
- A dependent table must have a foreign key reference to its parent table.
- When reading embedded child records, the loader must derive or validate the child foreign key from the containing parent record.
- If a child record explicitly contains the parent foreign key, it must match the containing parent record.
- If a child record omits the parent foreign key, the loader may materialize it from the parent context for validation, editing, and export.
- Embedded child record order is also preserved within each `children` sequence.
- A table may not be loaded twice from both a standalone table file and an embedded section in the same generation unless the schema defines deterministic merge behavior.
- Export consumes normalized table records, not the physical YAML nesting shape.
- Persistent generation merge consumes normalized table records and writes ordinary canonical destination YAML files.
- Persistent generation merge writes a complete effective table file for each registered schema that has merged records, not only source deltas.
- Persistent generation merge must not write response-only provenance comments into canonical table YAML.
- Persistent generation merge must not modify source generation folders or files.
- Generation duplication writes a new generation folder copied from exactly one selected source generation.
- Generation duplication must not modify the source generation folder or files.
- Generation analysis reads generation folders, table YAML, and schemas but writes no files.
- Export settings reads and writes are scoped to `masterdata/export_settings.yaml`.
- Template generation definition reads and writes are scoped to `masterdata/generate_definitions.yaml`.
- Template generation template files are read from `masterdata/generate_templates` during generate.
- Generation deletion removes only selected validated generation folders under `masterdata/generations`.
- Generation deletion must not follow symlinks outside `masterdata/generations`.
- Generation deletion must leave at least one valid generation unless a separate project reset feature is later specified.
- The first runnable slice may support one nesting level first, while preserving the record shape for deeper dependent trees later.
- YAML key ordering and quote style may follow the selected YAML library as long as data values round-trip correctly.
- `null` values are not used in canonical YAML.
- Missing optional values should be omitted.
- Empty strings are represented as empty strings, not `null`.
- Record sequence order is more important than mapping key order for Git review.

## Uses Common Details

- None yet.

## Reads

- [Generic master data model](generic-master-data-model.md)
- [Generation model](generation-model.md)
- [Export settings model](export-settings-model.md)
- [Template export definition model](template-export-definition-model.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)
- [Generation persistent merge flow](../data-flow/generation-persistent-merge-flow.md)
- [Generation deletion flow](../data-flow/generation-deletion-flow.md)
- [Generation duplication flow](../data-flow/generation-duplication-flow.md)
- [Generation analysis flow](../data-flow/generation-analysis-flow.md)

## Writes

- Normalized in-memory table records.
- YAML files under `masterdata/generations`.
- Export settings YAML at `masterdata/export_settings.yaml`.
- Template generation definitions YAML at `masterdata/generate_definitions.yaml`.

## Related Requirements

- [Schema validation engine](../component/schema-validation-engine.md)
- [Template export definition model](template-export-definition-model.md)
- [Web service host](../server-component/web-service-host.md)
- [Generation persistent merge flow](../data-flow/generation-persistent-merge-flow.md)
- [Generation deletion flow](../data-flow/generation-deletion-flow.md)
- [Generation duplication flow](../data-flow/generation-duplication-flow.md)
- [Generation analysis flow](../data-flow/generation-analysis-flow.md)

## Native-Language Summary

正本は `masterdata/schema` と `masterdata/generations` に置き、エクスポートの形式別既定値は任意の `masterdata/export_settings.yaml` に置く。Pongo2 テンプレート generate 定義は任意の `masterdata/generate_definitions.yaml` に置き、外部テンプレートは `masterdata/generate_templates/` 配下に置く。初期スライスでは `masterdata/generations/0000_initial` だけを使う。各世代フォルダには `_config.yaml` を置く。データYAMLは `<table>:` 配下に `key/name/data/children` を持つレコード配列として表現する。永続世代マージや世代複製で作成されるフォルダも同じ正本レイアウトを使い、元世代には書き込まない。Analyze は読み取り専用で件数や診断を返す。世代削除は選択された世代フォルダだけを対象にし、少なくとも1つの有効な世代を残す。
