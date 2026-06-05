---
id: "generic-master-data-model"
type: "data-model"
title: "Generic master data model"
aliases: []
tags: ["master-data", "schema", "yaml"]
facts:
  lifecycle.status: "blueprint"
  data.name: "generic-master-data"
---

# Generic master data model

## Summary

The canonical data model represents generic schema-driven master data as table records stored in YAML. It supports scalar fields, constants, composite primary keys, user-facing names and memos, and external references that resolve display selections into primary key values during export.

## Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| table_id | string | yes | Stable identifier for a master data table. |
| schema | object | yes | Table-local schema defining fields, primary keys, constants, formulas, export flags, validation, and references. |
| generation_id | string | yes | Ordered dataset layer that owns the record. |
| row_index | integer | yes | Zero-based position of the record in the YAML sequence for a table and generation. |
| record | object | yes | Row-like value object conforming to the table schema. |
| key | scalar or object | yes | Primary key value in YAML. Scalar for a single primary key, object for composite primary keys. |
| primary_key | object | yes | Normalized one-or-more-field key that uniquely identifies a record within a table after generation merge. |
| name | string | no | Human-readable record label for search and external reference selection. |
| memo | string | no | Free-form note for planners and developers. |
| data | object | yes | YAML field container for ordinary non-key data. |
| fields | object | yes | Normalized typed field values. Supported scalar kinds include integer, decimal, string, boolean, date, time, datetime, and constant values. |
| formula_fields | object | no | Computed field values derived from non-formula fields in the same record. |
| export_fields | array | yes | Fields selected for export after applying schema `export` flags. |
| children | array | no | YAML child record entries for dependent tables. |
| external_refs | object | no | Fields that store referenced table primary key values and resolve display names for editing UI. |
| parent_table | string | no | Parent table name when this table is a dependent table. |
| parent_foreign_key | object | no | Mapping from dependent table foreign key fields to parent table primary key fields. |

## Rules / Constraints

- Each table owns its schema.
- Primary keys may be composite.
- YAML records use `key`, `name`, `data`, and optional `children` as reserved record keys.
- A scalar YAML `key` maps to the table's single primary key field.
- An object YAML `key` maps to composite primary key fields.
- YAML `data` stores ordinary schema fields and is normalized into table fields for validation and export.
- YAML `children` stores dependent table records and is normalized into the child table.
- Record order is significant and must be preserved when reading and writing YAML.
- Row-level editing APIs address records by `row_index` for insert, update, delete, and move operations.
- Frontend commit mode may batch insert, update, delete, and move operations before saving.
- Row update operations include the previous primary key so the server can detect common stale-index mistakes.
- Primary key values are editable.
- Changing a primary key is treated as an update to the same row, not as delete plus insert.
- Primary key uniqueness is still validated independently from row position.
- Duplicate records are detected by table plus primary key after normalizing key values.
- Scalar field validation is type-specific.
- Constant fields must be selected from schema-defined values.
- Formula fields are computed from non-formula fields in the same record and are read-only for data entry.
- Formula fields must not depend on other formula fields.
- Schema fields may set `export: false`; non-exported fields can still be edited and used as formula input, but are omitted from export output.
- External reference fields store only referenced primary key values.
- External reference fields may display referenced record names during editing, but names are not stored as the reference value.
- External reference candidate lists contain target table primary keys and names.
- External reference definitions form a directed schema graph and must not contain cycles.
- A dependent table is a table whose records belong to a parent table through a foreign key reference.
- Dependent table records may be physically embedded in the parent table YAML file when the schema declares the parent relationship and foreign key mapping.
- The logical data model is always normalized by table, even when the physical YAML storage nests dependent records under parent records.
- YAML is the canonical persistence format and should remain diff-friendly for Git review.
- Canonical YAML does not use `null`; optional absent values are omitted and empty strings remain strings.
- Records can carry non-export or optional metadata such as name and memo when the target backend does not need those fields.

## Uses Common Details

- None yet.

## Reads

- Table schemas.
- [Table schema model](table-schema-model.md).
- [Canonical YAML file layout](canonical-yaml-file-layout.md).
- Generation ordering.
- Referenced table records for external reference validation.

## Writes

- YAML table data files.
- Validation diagnostics.

## Related Requirements

- [Product overview](../generic/product-overview.md)
- [Canonical YAML file layout](canonical-yaml-file-layout.md)
- [Table schema model](table-schema-model.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)
- [Schema validation engine](../component/schema-validation-engine.md)

## Native-Language Summary

表ごとにスキーマを持ち、整数・小数・文字列・boolean・日付・時刻・日時・定数列などを扱う。スキーマは数式フィールド、export対象フラグ、外部参照を持てる。従属テーブルは親への外部キーを持ち、親テーブルのYAMLにネストして保存できる。
