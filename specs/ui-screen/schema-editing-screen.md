---
id: "schema-editing-screen"
type: "ui-screen"
title: "Schema editing screen"
aliases: []
tags: ["ui", "schema", "editing"]
facts:
  lifecycle.status: "blueprint"
---

# Schema editing screen

## Summary

The schema editing screen manages table schema definitions through dedicated list and detail routes. It is not required for the first runnable slice, where schemas can be configured by files, but the SPA reserves it as a separate page for future schema authoring.

Schema editing uses the same `extable` commit-mode interaction model as generation metadata editing: users batch edit cells locally, review a confirmation dialog before writing schema YAML, revert uncommitted edits, and use row selection for bulk administration.

## User Goals

- Select a table schema to inspect or edit.
- Edit table-level schema metadata in a table list.
- Edit table system identifier, export flag, and comment.
- Inspect primary key and external reference summaries in the table list without editing them inline.
- Edit field definitions, primary keys, constants, formula fields, export flags, and external references.
- Edit primary key field system names, types, defaults, and comments.
- Edit external reference fields by choosing a referenced table and adding comments.
- Edit ordinary data fields with system names, types, optional formula expressions, default values, and comments.
- Add new schema fields with default values prefilled.
- Delete schema fields after explicit confirmation and remove that field from every existing record in every affected generation.
- Validate schema changes before saving.
- See reference-cycle errors before the schema can be used.
- Save schema YAML back to `masterdata/schema`.
- Return to table editing and reload affected table schemas.

## States

- First slice: schema page unavailable, read-only, or marked as future work.
- Schema selected.
- Creating a blank schema draft.
- Editing schema table list in an `extable` grid.
- One or more schema table rows selected.
- Dirty table-list schema metadata changes.
- Undo available for uncommitted schema list edits.
- Redo available after undoing schema list edits.
- Commit confirmation dialog open with a summary of pending table-level schema changes.
- Revert confirmation shown because pending table-list schema edits exist.
- Detail editor open for one table schema.
- Editing field definitions.
- Editing mixed field rows in one `extable` detail grid.
- Undo available for uncommitted schema detail edits.
- Redo available after undoing schema detail edits.
- Creating schema fields with schema-defined default values prefilled.
- Formula fields visible but authoring disabled while formula implementation is deferred.
- Editing external references.
- Field delete confirmation dialog open with impacted table, column, and generation data summary.
- Field delete confirmed and pending schema plus pending data-cleanup operations prepared.
- Validation errors in schema definition.
- Dirty schema changes.
- Save ready.
- Save blocked by invalid schema.

## Invoked APIs

- Load schema list.
- Load table schema.
- Validate schema definition.
- Validate formula expressions.
- Validate external reference graph.
- Save schema YAML.
- Save schema field deletion with data cleanup across existing generation records.
- Save schema field rename with data key rename across existing generation records.
- Reload table editing schema after save.

## Schema List Screen

- The schema list route is `/schemas`.
- Use npm package `extable` as the primary editor for the schema table list.
- The schema table list grid must be an `extable` grid rather than a bespoke HTML table.
- The grid should use `extable` commit mode so multiple table metadata rows can be edited before saving.
- Pending schema list edits stay local until the user presses the commit/save button.
- Pressing the commit/save button must open a confirmation dialog before any schema save API is called.
- Cancelling the commit confirmation leaves pending edits unchanged and performs no writes.
- Confirming the commit dialog is the only path that calls schema list save APIs.
- A `Revert` command must be available when pending schema list edits exist.
- Activating `Revert` shows a confirmation dialog before discarding edits.
- Confirmed `Revert` restores the grid to the last loaded or last committed schema list snapshot, clears dirty state, and does not call save APIs or write YAML files.
- `Undo` and `Redo` commands must be available for schema list edits.
- `Undo` steps backward through uncommitted schema list cell edits, row insertions, row deletions, row selections that affect pending operations, and row moves when supported by the grid.
- `Redo` reapplies schema list operations that were undone.
- `Undo` and `Redo` operate only on frontend-local pending edits and must not call save APIs or write YAML files.
- After the user makes a new edit following `Undo`, the redo stack is cleared.
- `Revert` is distinct from `Undo`: it discards all pending schema list edits and returns to the last loaded or last committed snapshot after confirmation.
- Columns should include a leftmost row selection checkbox column, `system_name`, `export`, readonly `primary_key`, readonly `references`, and `comment`.
- The row selection checkbox column has no text title.
- The row selection checkbox column is interactive while the grid has no pending schema list edits.
- The row selection checkbox column is disabled or ignored while schema list edits are dirty.
- `system_name` edits the stable table identifier used by the system and schema file identity.
- `export` uses a boolean checkbox-style editor.
- `primary_key` is readonly and shows the primary key field system names in declared order.
- `references` is readonly and shows referenced target table identifiers used by fields in the table schema.
- `comment` is editable text.
- Rows should be sorted by table system identifier unless the project defines an explicit schema ordering rule.
- Double-clicking or activating a table row opens the schema detail editor for that table.
- The top bar includes create schema, commit/save schema list changes, revert schema list changes, delete selected schemas, and return-to-table controls.
- The top bar includes undo and redo controls for schema list edits when those operations are available.
- Creating a schema inserts a blank draft schema row rather than cloning an existing schema.
- A blank draft schema starts with empty `system_name`, `export: true`, empty `comment`, no primary key fields, and no data fields.
- A blank draft schema is invalid until the user supplies a unique `system_name` and at least one primary key field in detail editing.
- `Delete` is enabled only when at least one persisted schema row is selected and the schema list grid has no pending edits.
- `Delete` must open a confirmation dialog before removing schema YAML files.
- Schema delete removes only the selected schema file under `masterdata/schema`; it does not delete existing generation table data files.
- The schema delete confirmation explains that the project is expected to be under Git management and that data files may remain for review, recovery, or manual cleanup.
- The schema list does not provide a schema duplicate action.
- Changing table `system_name` is treated as a table schema rename operation, not as an ordinary metadata edit.
- A table schema rename must rename the schema file identity, rename table data files in generation folders that use the table system name as their file name, and update schema references that point to the renamed table.
- The commit confirmation dialog must explicitly list table schema renames, old and new `system_name` values, and affected reference definitions before saving.
- Administration actions must follow the same selection, dirty-state disabling, confirmation, and reload behavior as generation metadata editing unless this document explicitly defines a different behavior.

## Schema Detail Screen

- The schema detail route is `/schemas/:table/edit`.
- The detail editor uses one `extable` grid for all field kinds in the selected table schema.
- The detail grid should use `extable` commit mode so users can edit several field rows before saving.
- Pending field edits stay local until the user presses the commit/save button.
- Pressing the commit/save button must open a confirmation dialog before saving schema YAML or applying data cleanup.
- A `Revert` command must be available when pending detail edits exist and must restore the last loaded or last committed table schema without writing files.
- `Undo` and `Redo` commands must be available for schema detail edits.
- `Undo` steps backward through uncommitted detail cell edits, field row insertions, row deletion preparations, row kind changes, row moves, and primary-key ordering changes when supported by the grid.
- `Redo` reapplies schema detail operations that were undone.
- `Undo` and `Redo` operate only on frontend-local pending edits and must not call save APIs or write YAML files.
- After the user makes a new edit following `Undo`, the redo stack is cleared.
- `Revert` is distinct from `Undo`: it discards all pending detail edits and returns to the last loaded or last committed schema after confirmation.
- The detail top bar includes undo and redo controls when those operations are available.
- The detail grid includes a leftmost row selection checkbox column for field administration.
- Field row selection is disabled or ignored while detail edits are dirty.
- The detail grid row model has a `kind` value: `primary_key`, `reference`, `data`, or `formula`.
- The detail grid columns should include `kind`, `system_name`, `type`, `formula`, `reference_table`, `default_value`, `export`, `required`, and `comment`.
- `kind` uses an enum editor and controls which cells are editable.
- `primary_key` rows define primary key fields. Their `system_name`, `type`, `default_value`, and `comment` are editable; `formula` and `reference_table` are readonly or empty.
- `reference` rows define external reference fields. Their `system_name`, `reference_table`, `default_value`, `export`, `required`, and `comment` are editable; `reference_table` uses a table lookup from the current schema list.
- `data` rows define ordinary non-reference, non-formula fields. Their `system_name`, `type`, `default_value`, `export`, `required`, and `comment` are editable.
- `formula` rows define computed fields. While formula implementation is deferred, formula rows are shown as disabled/read-only or hidden behind a disabled row-kind option; users cannot create or edit formula rows in the first schema editing implementation.
- The grid may later allow switching a row between `data` and `formula` when formula implementation is enabled and the resulting row validates; switching to or from `primary_key` or `reference` must re-run schema validation before commit.
- Changing `reference_table` keeps existing stored reference values unchanged. If those primary key values do not exist in the new referenced table, validation reports ordinary unresolved-reference errors.
- Changing a field `type` does not rewrite existing stored values. If existing values do not validate against the new type, save may proceed only through a warning confirmation that shows affected field names and validation diagnostics.
- Changing any field `system_name`, including primary key fields, is allowed as a field rename.
- A field rename updates the schema field name and renames the corresponding YAML keys in existing records.
- For non-primary-key fields, field rename updates keys under record `data`.
- For composite primary keys, field rename updates the matching property name inside object-shaped reserved YAML `key` values.
- For a single scalar primary key, field rename updates the schema `primary_key` entry while scalar reserved YAML `key` values remain unchanged.
- Primary key changes, field renames, type changes, table renames, reference target changes, and field deletions are summarized together in the ordinary commit confirmation dialog; there is no separate primary-key-only confirmation dialog.
- `system_name` uses the `extable` `unique` flag across all field rows in one table schema.
- Commit/save is disabled while the grid has duplicate field `system_name` values or invalid field cells.
- The detail grid should preserve field order as displayed, and row move operations should update schema field order.
- Creating a new row applies schema field defaults immediately before the row becomes editable.
- Default inserted field values are: `kind: data`, `system_name` empty, `type: string`, `default_value` empty string, `export: true`, `required: false`, and `comment` empty.
- For boolean fields, an absent `default_value` should load as `false` when data omits the field.
- For numeric fields, an absent `default_value` should load as `0` when data omits the field unless the field is optional and the project config chooses empty optional values.
- For date, time, and datetime fields, an absent `default_value` remains empty unless the field explicitly defines a default expression supported by the validation engine.
- External reference fields have no implicit default unless a default referenced primary key is explicitly configured.
- Formula fields have no stored default value.
- When a schema field has a configured `default_value`, table data loading materializes it into the editable row value when the YAML data omits that field.
- Once a default has been loaded into the editable row model, later saving that table writes the row value normally.
- Export uses the same loaded or normalized row value; it does not need a separate default lookup path when the table was loaded through the editor.

## Field Deletion

- The detail top bar includes a `Delete field` command for deleting selected field rows.
- `Delete field` is enabled only when one or more persisted field rows are selected and the detail grid has no pending edits.
- Activating `Delete field` opens a confirmation dialog rather than immediately deleting the field.
- The confirmation dialog lists the selected field system names, whether each field is a primary key, external reference, ordinary data, or formula field, and the affected table data locations.
- Deleting a primary key field is blocked unless the resulting table schema still has at least one valid primary key field.
- Deleting a field referenced by a formula, external reference mapping, dependent-table relationship, or another schema relationship is blocked until the relationship is changed or removed.
- Confirming field deletion updates the pending schema definition and removes the deleted non-formula column from every existing YAML record for that table in every generation that stores the table.
- For embedded dependent table records, cleanup must remove the deleted column from every matching embedded child record as well as table-per-file records.
- Formula fields are deleted from schema only because user-entered YAML data must not store formula values.
- The delete operation must not remove reserved YAML record keys `key`, `name`, `data`, or `children`; it removes only schema field values from canonical data locations.
- The server must perform the schema update and data cleanup atomically from the user's perspective: if any affected YAML file cannot be parsed, validated for safe cleanup, or written, no partial deletion should be committed.
- After successful schema changes, the frontend reloads the full schema list and schema cache.
- Table record data is loaded lazily when a table is selected for editing; schema editing does not need to eagerly reload every table's record data unless the confirmed schema operation itself must scan or rewrite data files, such as field deletion, field rename, or table rename.

## Navigation And Layout

- Schema editing is a separate SPA page rather than a modal overlay on the main table editing grid.
- The table editing workspace always links to the schema list through a compact schema/settings action in the left pane.
- The schema/settings action should be near the table navigation header or footer, visually separated from table rows.
- The schema/settings action must not be rendered as a normal table navigation row.
- The schema/settings action navigates to `/schemas`.
- If the table editing workspace has unsaved edits, activating the schema/settings action uses the same dirty-state confirmation flow as other navigation away from table editing.
- The schema list and detail screens use focused application chrome similar to generation metadata editing.
- The application does not separate schema editing permissions from ordinary editing permissions; any user who can use the app can open and change schemas.
- Unsaved schema edits trigger a navigation guard before leaving the page.
- Browser back, return-to-table, and row-to-detail navigation use the same unsaved-change guard.
- Returning to table editing preserves the previous active table, active edit generation, and display mode when practical.

## Components

- [Single page application shell](../ui-flow/single-page-application-shell.md)
- [Table schema model](../data-model/table-schema-model.md)
- [Schema validation engine](../component/schema-validation-engine.md)
- [Web service host](../server-component/web-service-host.md)

## Related Requirements

- [Generic master data model](../data-model/generic-master-data-model.md)
- [Canonical YAML file layout](../data-model/canonical-yaml-file-layout.md)
- [Table editing workspace](table-editing-workspace.md)

## Native-Language Summary

スキーマ編集画面。初期スライスでは設定ファイルのみでもよいが、SPA上では別ページとして確保する。フィールド、主キー、定数、数式、exportフラグ、外部参照を編集し、保存前に検証する。
