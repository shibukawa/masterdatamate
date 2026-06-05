---
id: "schema-validation-engine"
type: "server-component"
title: "Schema validation engine"
aliases: []
tags: ["validation", "schema"]
facts:
  lifecycle.status: "blueprint"
---

# Schema validation engine

## Summary

The schema validation engine validates table schemas, YAML records, pending frontend edits, generation merge results, and external references. It provides diagnostics usable by the shared editing frontend and by export workflows.

## Scope

- In scope:
  - Validate scalar values for integer, decimal, string, boolean, date, time, datetime, and constants.
  - Validate required fields and missing values.
  - Validate the reserved YAML record shape: `key`, optional `name`, `data`, and optional `children`.
  - Validate composite primary keys.
  - Detect duplicate primary keys.
  - Validate schema formula fields and evaluate formula results.
  - Validate schema field default values.
  - Validate schema export flags.
  - Validate external references by resolving stored primary key values to target table records.
  - Validate that external reference definitions do not create cycles.
  - Validate pending frontend edits before commit.
  - Produce diagnostics that can be shown inline in a table editor and used to block export.
- Out of scope:
  - Repository permissions.
  - Approval workflow.
  - Business review status.

## Rules / Constraints

- Validation must be deterministic and independent of UI state.
- Validation must work from canonical YAML and schema files so it can run in CI.
- The same validation rules must be usable in the frontend for pending edits and on the server for commit requests.
- Invalid table schema is a blocking diagnostic for that table; the table editor must not open.
- Missing or invalid generation `_config.yaml` is a blocking diagnostic for the active generation; table editing must not open.
- Missing table YAML is not a validation error; it is interpreted as an empty table for that generation.
- Diagnostics should identify table, generation, record key, field, severity, and message.
- Duplicate primary key checks must understand composite keys.
- Single-field primary key tables may use scalar YAML `key`; composite primary key tables must use object-shaped YAML `key`.
- YAML `data` must be an object and must validate against the table schema fields.
- YAML `children` must be a sequence of child record entries with `table`, `key`, optional `name`, `data`, and optional nested `children`.
- User-entered YAML `data` must not contain values for formula fields.
- Formula fields must be read-only and computed from non-formula fields in the same record.
- Formula validation and evaluation may be deferred after the first runnable slice.
- Formula syntax must validate as a Jessie-compatible expression-only subset.
- Formula validation must reject statements, imports, host API access, global object access, mutation, and I/O.
- Formula expressions must not reference other formula fields.
- Formula result values must validate against the formula field declared type.
- Schema field `default_value` values must validate against declared field type, constants, requiredness, and external reference target.
- Formula fields must not define stored `default_value`.
- If a non-formula field omits `default_value`, type-based defaults may be materialized when table data is loaded into the editable row model, but validation should preserve the distinction between omitted source YAML and explicitly stored data when diagnostics need to explain the source.
- Default-value application during table load may be written later if the user saves the loaded editable row.
- Explicitly stored values that fail after a field type change are validation diagnostics.
- Schema saves that change a field type may proceed only after a warning confirmation when existing stored values do not validate against the new type.
- Fields with `export: false` remain valid input fields and may be used by formulas, but must be marked as non-exported in the normalized record metadata.
- External reference resolution uses target table primary keys, not arbitrary lookup fields.
- In active-only editing mode, external reference resolution checks target records in the active edit generation only when that generation is output-enabled.
- In include-previous editing or export preview mode, external reference resolution checks the effective target records from output-enabled generations older than or equal to the active edit generation.
- External reference resolution must exclude records from generations whose `output` flag is false, even if the active edit generation is visible and editable in the table editor.
- If the same referenced primary key exists in multiple participating generations, validation resolves it to the newest effective record according to generation ordering.
- Reference display names are not authoritative and are used only for editing UI.
- If the frontend uses display objects for lookup cells, such as `{ label, value }`, validation must normalize the object to the referenced primary-key `value` before checking existence. Display `label` must never be used as the authoritative reference value.
- Missing referenced primary keys are validation errors.
- Changing an external reference target table does not rewrite existing stored reference values; unresolved values against the new target table are reported as missing referenced primary key errors.
- External reference definitions must form an acyclic directed graph.
- Schema validation must reject a new external reference or parent dependency that would create a cycle.
- Schema validation must reject deleting a field that is still referenced by primary key definitions, formula expressions, external reference mappings, or dependent-table relationships unless the same pending schema change removes or repairs that relationship.
- Primary key definition changes and field renames use the ordinary schema commit confirmation, but validation must still report records whose reserved YAML `key` cannot be normalized under the new definition.
- Dependent table schemas must declare their parent table and foreign key mapping.
- Embedded dependent records must resolve to exactly one containing parent record.
- If an embedded dependent record contains parent foreign key fields, those values must match the containing parent record.
- If an embedded dependent record omits parent foreign key fields, validation should evaluate the record with parent-derived key values.
- Export should only use data that has passed schema and reference validation.
- Unresolved or ambiguous external references are validation errors.
- Rows containing unresolved or ambiguous external references should be markable as error rows in the editing UI.
- Save behavior for validation errors is configurable: either warn and allow saving, or block saving.
- The first runnable slice uses `warn_and_save` through an explicit confirmation flow.
- A normal commit with validation errors should be rejected with diagnostics.
- After user confirmation, the frontend retries the commit with `force: true`; the server saves despite validation errors and returns diagnostics.
- Export behavior is strict: validation errors block export.
- Save and export operations must notify users when validation errors exist.

## Dependencies

- [Generic master data model](../data-model/generic-master-data-model.md)
- [Canonical YAML file layout](../data-model/canonical-yaml-file-layout.md)
- [Table schema model](../data-model/table-schema-model.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)

## Related Documents

- [Product overview](../generic/product-overview.md)
- [Table editing workspace](../ui-screen/table-editing-workspace.md)

## Native-Language Summary

表ごとのスキーマに従い、型、必須、定数、複合主キー、重複、数式、export対象、外部参照を検証する。外部参照グラフは循環を禁止し、UIだけでなくCIやエクスポート前チェックでも同じ検証を使う。
