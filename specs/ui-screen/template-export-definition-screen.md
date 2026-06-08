---
id: "template-export-definition-screen"
type: "ui-screen"
title: "Template generation definition screen"
aliases: ["generation definition list screen", "template generation settings screen", "template export settings screen"]
tags: ["ui", "generate", "template", "extable", "pongo2"]
facts:
  lifecycle.status: "blueprint"
---

# Template generation definition screen

## Summary

The template generation definition screen lets users manage Pongo2-based generation definitions separately from table schema definitions. It uses `extable` commit mode for the definition list so users can batch-edit output definitions, review changes, and save them to `masterdata/generate_definitions.yaml`.

The screen is intended for users who maintain generated code or text artifacts, such as Go constants and error types generated from an error-message master table.

## User Goals

- See all configured template generation definitions.
- Add, edit, duplicate, reorder, enable, disable, and delete generation definitions.
- Choose whether a definition renders per project, table, record, or group.
- Select target tables and group fields from existing schemas.
- Choose inline template text or an external template file.
- Edit output path templates.
- Validate Pongo2 syntax and output path safety before saving.
- Run a check for selected definitions without writing artifacts.
- Navigate back to ordinary editing and use saved definitions from GUI or `masterdatamate generate`.

## Route

The route is `/generate/definitions`.

The main table editing shell links to this screen from the left-pane project action area. The action label should be `Generate definitions`.

## Layout

- The screen uses focused application chrome similar to schema editing and generation editing.
- The primary surface is an `extable` grid of definition rows.
- A secondary detail panel or expandable row may edit long template text, description, and advanced options when those values are too large for comfortable grid editing.
- The top bar includes return-to-table, add definition, duplicate selected, delete selected, save, revert, undo, redo, and check selected commands.
- Save and check commands are disabled while the grid has invalid cells.
- Unsaved definition edits trigger the ordinary navigation guard.

## Extable Columns

The first slice grid columns are:

| Column | Editor | Notes |
| --- | --- | --- |
| selection | checkbox | No text title. Disabled while dirty when used for destructive actions. |
| id | text | Unique stable ASCII identifier. |
| name | text | Human-facing label. |
| enabled | boolean | Default inclusion for template generation runs. |
| scope | enum | `project`, `table`, `record`, or `group`. |
| table | enum | Existing schema table `system_name`; empty only for `project`. |
| group_field | enum | Required when `scope` is `group`; sourced from the selected table's fields. |
| template_file | text | Relative path under `masterdata/generate_templates/`. |
| output_path | text | Pongo2 path template relative to generation output root. |
| formatter | enum | Empty or `gofmt` in the first slice. |
| comment | text | Maintainer note. |

Advanced fields such as inline template body, related tables, overwrite policy, line ending, and `include_non_exported_fields` may live in a detail panel until the grid supports comfortable editing for nested values.

## Validation

Frontend validation should catch immediate editing errors:

- Duplicate or empty definition IDs.
- Invalid identifier characters.
- Missing required table for table, record, or group scope.
- Missing group field for group scope.
- Missing both `template` and `template_file`.
- Setting both `template` and `template_file`.
- Output path template empty.
- Absolute output paths.
- Obvious `..` path traversal in static output path text.

Server validation remains authoritative and must re-parse Pongo2 templates, resolve schema references, validate template file paths, and check rendered path safety during generate check.

## Save Behavior

- Pending edits stay local until the user saves.
- Save opens a confirmation dialog summarizing added, removed, renamed, and changed definitions.
- Confirming save writes only `masterdata/generate_definitions.yaml`.
- The save operation must not write schema YAML, generation YAML, or template files unless a separate template-file editor is specified later.
- Revert restores the last loaded or committed definition list without writing files.
- Undo and redo operate only on frontend-local pending edits.
- Deleting definitions requires confirmation.
- Duplicating a definition creates a new draft with a new ID placeholder and keeps it invalid until the user makes the ID unique.

## Check Behavior

`Check selected` calls the generate check path with the selected definition IDs. It validates selected definitions, templates, output paths, formatters, and data but does not write generated artifacts.

If the current screen has unsaved edits, check uses the pending in-memory definitions only when the API explicitly accepts draft definitions. Otherwise check is disabled until the user saves.

## Rules / Constraints

- Generation definitions are not schema fields and must not appear inside the schema editing screen.
- The screen manages `masterdata/generate_definitions.yaml`.
- Template file body editing is out of scope for the first slice unless a simple text editor is added explicitly.
- The ordinary table editor remains the way to edit source records used by generated artifacts.
- The generation definition screen must not infer schema changes from templates.
- Users who can edit project data can edit generation definitions; no separate permission model is specified.
- Definition ordering is meaningful because it controls default render order.
- The UI should preserve unknown YAML keys when practical to avoid destructive rewrites of hand-authored advanced settings.

## Dependencies

- [Template export definition model](../data-model/template-export-definition-model.md)
- [Pongo2 template export adapter](../component/pongo2-template-export-adapter.md)
- [Go CLI generate runner](../batch-component/go-cli-generate-runner.md)
- [Single page application shell](../ui-flow/single-page-application-shell.md)
- [Table schema model](../data-model/table-schema-model.md)

## Native-Language Summary

Pongo2 ベースの generate 定義を編集する画面。スキーマ編集とは別の `/generate/definitions` に置き、`extable` commit mode で一覧を管理する。定義ごとに scope、対象 table、group field、template file、output path、formatter などを編集し、保存すると `masterdata/generate_definitions.yaml` にだけ書く。選択定義の check は generate check 経路を使い、生成前にテンプレート構文、参照、出力パス安全性、formatter、データ検証を確認する。
