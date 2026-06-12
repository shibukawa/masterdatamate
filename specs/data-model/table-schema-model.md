---
id: "table-schema-model"
type: "data-model"
title: "Table schema model"
aliases: []
tags: ["schema", "formula", "export", "foreign-key"]
facts:
  lifecycle.status: "blueprint"
  data.name: "table-schema"
---

# Table schema model

## Summary

Each table schema defines fields, primary keys, export behavior, formula fields, constants, and references. Formula fields behave like Excel-like derived columns: they are computed from non-formula values in the same record, are read-only in the editor, and may be exported or kept internal according to field export flags.

## Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| table_id | string | yes | Stable table identifier. |
| system_name | string | yes | ASCII-oriented system name for the table. |
| business_name | string | yes | Human-facing business name, such as Japanese display name. |
| primary_key | array | yes | Field names that form the table primary key. |
| export | boolean | yes | Whether this table is included in export output by default. |
| ui | object | no | Editor-facing presentation metadata. It does not affect canonical storage or export by itself. |
| ui.table_list_visibility | enum | no | `visible`, `plugin_only`, or `hidden`. Defaults to `visible`. |
| ui.preferred_plugin | string | no | Optional plugin ID to suggest when `ui.table_list_visibility` is `plugin_only`. |
| dependent_tables | array | no | Child table definitions that may be embedded under parent records. |
| comment | string | no | Human-facing schema note. |
| fields | array | yes | Schema field definitions. |
| field.system_name | string | yes | ASCII-oriented system field name. |
| field.business_name | string | yes | Human-facing business field name, such as Japanese display name. |
| field.type | enum | yes | Scalar type, constant, external reference, binary file metadata, or formula result type. |
| field.required | boolean | yes | Whether the field is required in YAML data. |
| field.export | boolean | yes | Whether this field is included in export output. |
| field.default_value | typed value | no | Value used when a new field value is materialized or when existing data omits the field. |
| field.formula | Jessie expression | no | Expression-only Jessie-compatible formula for computed fields. |
| field.readonly | boolean | derived | Formula fields are always read-only. |
| field.reference | object | no | External reference definition pointing to a target table's primary key. |
| field.constants | array | no | Allowed values for constant fields. |
| field.binary | object | no | Binary upload constraints when `field.type` is `binary_file`. |
| field.binary.allowed_extensions | array | no | Allowed lowercase extensions without leading dots. |
| field.binary.allowed_mime_types | array | no | Allowed MIME types when the host can detect them. |
| field.binary.max_size_bytes | integer | no | Maximum accepted upload size. |
| field.comment | string | no | Human-facing field note. |

## Minimal Schema YAML

Schema files live under `masterdata/schema`.

```yaml
system_name: org
business_name: 組織
primary_key:
  - org_id
export: true
ui:
  table_list_visibility: visible
dependent_tables:
  - table: user
    foreign_key:
      org_id: org_id
comment: Organization master.
fields:
  - system_name: org_id
    business_name: 組織ID
    type: string
    required: true
    export: true
    default_value: ""
    comment: Stable organization identifier.
  - system_name: display_name
    business_name: 表示名
    type: string
    required: true
    export: true
    default_value: ""
  - system_name: internal_note
    business_name: 内部メモ
    type: string
    required: false
    export: false
    default_value: ""
```

## Default Values

- Non-formula schema fields may define `default_value`.
- `default_value` must validate against the field's declared type, constants, or external reference target.
- A field with constants may use only one of its allowed constant values as `default_value`.
- An external reference field may use a referenced primary key value as `default_value`; the default must resolve under the same generation-aware reference rules as ordinary field values when validation runs.
- Formula fields must not define stored `default_value` because their values are computed.
- If `default_value` is omitted, table data loading may materialize type-based defaults into the editable row model: string is empty string, boolean is false, numeric fields are zero, and date/time/datetime remain empty unless an explicit supported default expression is configured.
- Optional fields may remain empty if the project configuration chooses empty optional values over type-based materialization.
- When table data omits a non-formula field, the normalized editable record exposes the configured or type-based default as the row value.
- Saving an editable row writes the row value normally, including defaults that were loaded into omitted fields.
- Validation diagnostics should distinguish an explicitly stored invalid value from an omitted value that only failed because no valid default could be applied.

## Formula Fields

- Formula field implementation may be deferred after the first runnable slice.
- A formula field is declared in schema, not stored as user-entered data.
- Formula fields do not store default values.
- Formula syntax should use a Jessie-compatible expression subset.
- MasterDataMate formulas use expressions only, not general Jessie programs or statements.
- A formula uses only non-formula fields from the same logical record as input.
- A formula must not use another formula field as input.
- Because formulas cannot depend on formulas, formula dependency cycles are structurally disallowed.
- A formula field result becomes the effective field value for validation, display, and export when `field.export` is true.
- Formula result values must validate against the formula field's declared result type.
- Formula fields are read-only in table editing UI.
- User-entered YAML `data` must not provide values for formula fields unless a migration or repair mode explicitly allows cleanup.

## Export Flags

- Every table schema has an `export` flag.
- Every schema field has an `export` flag.
- A table with `export: false` is omitted from export outputs by default.
- `export: true` fields are included in export outputs.
- `export: false` fields are available for editing, validation, formula inputs, lookup labels, and other internal workflows, but are omitted from export outputs.
- A non-exported field may be used as formula input.
- A formula field may be exported or non-exported.
- Primary key fields are exported unless a specific backend has a documented alternate key mapping.

## Editor Visibility

`ui.table_list_visibility` controls whether the ordinary table editing workspace lists the table as a direct navigation item. It does not change validation, export, file layout, reference resolution, or plugin access.

| Value | Behavior |
| --- | --- |
| `visible` | Default. Show the table in the ordinary table list and allow grid editing subject to schema validity and permissions. |
| `plugin_only` | Hide the table from the ordinary table list because normal users are expected to edit it through one or more editor plugins. The table remains loadable by plugin scopes, validation, export, references, schema editing, diagnostics, and developer/admin repair paths. |
| `hidden` | Hide the table from ordinary data navigation. Use for implementation-detail tables that should not be presented as normal edit targets. The table still remains canonical data and must be reachable by validation and explicit tooling. |

When `plugin_only` is used, `ui.preferred_plugin` may name the plugin that should be presented as the primary editing destination for this table. The named plugin must exist in `masterdata/editor_plugins.yaml` before the host can use it as a navigation hint. If the plugin is missing or invalid, the host should surface a configuration diagnostic rather than silently making the table uneditable.

## External References

- External reference fields define a directed dependency from the referencing table to the referenced table.
- External references can target only the referenced table's primary key.
- External reference field values store only the referenced primary key value or composite primary key object.
- Changing an external reference target table keeps stored reference values unchanged; validation then resolves the unchanged primary key values against the new target table.
- Referenced record names are display-only labels for lookup and editing UI.
- Reference candidate APIs return primary key and name pairs for a target table.
- The schema graph formed by external references must be acyclic.
- A schema definition that would introduce a foreign key reference cycle is invalid.

## Binary File Fields

- A `binary_file` field declares that the record may have one uploaded file stored through [Binary asset model](binary-asset-model.md).
- The field stores metadata such as extension, MIME type, file size, hash, and original filename; it does not store file bytes.
- The default physical path is `masterdata/binaries/<table>/<primary-key>.<extension>`.
- The first implementation supports at most one `binary_file` field per table because the default path does not include field name.
- Required `binary_file` fields require a matching stored binary file and valid metadata.
- Optional `binary_file` fields may be empty.
- `field.binary.allowed_extensions` constrains accepted upload extensions.
- `field.binary.allowed_mime_types` constrains detected MIME types when detection is available.
- `field.binary.max_size_bytes` constrains upload size.
- Uploading, replacing, or deleting file bytes uses host binary asset APIs rather than ordinary row commit operations.
- The ordinary row commit may store metadata returned by the upload API.

## Schema Editing Representation

- The schema editing UI represents one table schema's fields as a single ordered field list.
- Each row has an editing `kind`: `primary_key`, `reference`, `data`, or `formula`.
- Primary key rows are ordinary fields that also participate in the table `primary_key` list.
- Reference rows are fields whose `reference` object points to a target table primary key.
- Data rows are non-reference, non-formula fields.
- Formula rows are fields with `formula` and a declared result `type`.
- The persisted YAML may keep `primary_key` as a table-level ordered array and `fields` as field definitions; the UI adapter derives row `kind` from those canonical values and converts edits back to canonical schema YAML.
- Field `system_name` values are unique across all field kinds within one table schema.
- Field `system_name` changes are field rename operations.
- Field rename updates schema definitions and existing YAML keys for that field.
- For non-primary-key fields, field rename changes the corresponding keys under record `data`.
- For composite primary keys, field rename changes the corresponding property name inside object-shaped reserved YAML `key` values.
- For a single scalar primary key, field rename changes the schema `primary_key` field name; scalar reserved YAML `key` values do not need structural changes.
- Field deletion removes the field definition from schema YAML and removes the corresponding value from existing canonical record `data` locations for that table, except formula fields which should not have stored data values.
- Table schema rename changes the table `system_name`, schema file identity, and table data file names in generation folders together, and references to that table in other schemas must be updated as part of the same schema save.
- Table schema deletion removes the schema file only; generation data files are left in place for Git-backed review, recovery, or manual cleanup.
- Schema duplication is intentionally not part of the schema editing model.
- Dependent table parent references are part of the same directed reference graph and must also not create cycles.
- Reference cycles should be reported as schema validation errors before data editing or export.

## Rules / Constraints

- Schema validation runs before record validation.
- Invalid table schema is a blocking error; affected tables cannot be opened for editing.
- `system_name` values are stable implementation identifiers and should use ASCII-friendly names.
- `business_name` values are human-facing labels and may use Japanese or other natural-language text.
- Field `system_name` values must not collide with reserved YAML record keys `key`, `name`, `data`, or `children` unless the schema explicitly maps them inside `data`.
- Formula expressions are deterministic and must not depend on external state.
- Formula expressions should be parsed and evaluated by a JavaScript library when a suitable Jessie implementation is available.
- The evaluator must expose only the current record's non-formula field values as formula inputs.
- Formula expressions must not perform table lookups.
- Field `default_value` values must not depend on external state or call host APIs.
- Binary file fields must not define non-empty `default_value` values because uploaded file bytes cannot be synthesized from schema.
- Formula expressions must not mutate data.
- Formula expressions must not call host APIs, access global objects, import modules, or perform I/O.
- Formula evaluation errors are validation errors on the formula field.
- Export adapters consume the normalized effective record and then filter by `field.export`.
- Export adapters that support binary assets may copy files from [Binary asset model](binary-asset-model.md); export adapters that do not support binary assets should export only metadata or omit the field according to format rules.
- `ui.table_list_visibility` must not affect export eligibility; use table `export` for export behavior.
- A `plugin_only` table should have at least one valid editor plugin entry point that can edit or inspect it, or the UI must expose a configuration diagnostic.

## Uses Common Details

- None yet.

## Reads

- [Generic master data model](generic-master-data-model.md)
- [Canonical YAML file layout](canonical-yaml-file-layout.md)
- [Binary asset model](binary-asset-model.md)

## Writes

- Normalized schema metadata.
- Formula evaluation diagnostics.
- Export field selection metadata.

## Related Requirements

- [Schema validation engine](../component/schema-validation-engine.md)
- [Export backend adapters](../component/export-backend-adapters.md)
- [Table editing workspace](../ui-screen/table-editing-workspace.md)

## Native-Language Summary

テーブルスキーマはフィールド、主キー、export対象フラグ、数式、外部参照を定義する。`ui.table_list_visibility` により、通常のテーブル一覧に表示するか、プラグイン経由を主導線にするかも指定できる。数式はJessie互換のexpression-only subsetを使う方針。数式フィールドは同一レコード内の非数式フィールドだけを入力にし、編集UIではリードオンリー。外部キー参照グラフは循環を禁止する。
