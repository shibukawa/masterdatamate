---
id: "table-editing-workspace"
type: "ui-screen"
title: "Table editing workspace"
aliases: []
tags: ["ui", "editing"]
facts:
  lifecycle.status: "blueprint"
---

# Table editing workspace

## Summary

The table editing workspace is the primary screen for browsing tables, editing records through an Excel-like grid, resolving references, reviewing validation diagnostics, and saving YAML-backed changes. It uses a two-pane layout: table names are listed vertically on the left, and the `extable` grid dominates the center pane.

Generation metadata editing is not part of this workspace. Users navigate to the dedicated [Generation editing screen](generation-editing-screen.md) from the edit icon next to the sidebar generation selector when they need to create, rename, reorder, or edit generation settings.

Schema editing is also not embedded in the table record grid. Users navigate to the dedicated [Schema editing screen](schema-editing-screen.md) from a compact schema/settings action in the left pane when they need to add tables, change field definitions, or maintain schema metadata. The app does not require separate schema-editing permissions.

## User Goals

- Find and edit records in a table without editing YAML directly.
- Add records with valid primary key values and schema-defined fields.
- Select constant values from allowed options.
- Select external references by human-readable names while storing only referenced primary key values.
- Select external references from lookup candidates labeled as `<referenced name> (<primary key>)`, show the selected referenced name in the grid cell, and persist only the referenced primary key value.
- See validation errors close to the affected table cell or record.
- See formula field values as read-only computed cells.
- Save edited table data back to canonical YAML.
- Choose the active edit generation for data changes.
- Open generation metadata editing from the active generation selector when generation settings need to change.
- Open schema editing from the left pane when table schemas need to change.
- Return from generation metadata editing without losing the current table editing context.
- Return from schema editing without losing the current table editing context.
- Choose whether the grid shows only the active edit generation or includes previous generations that participate in output.
- See which generation each visible record comes from.
- Edit records only in the active edit generation.
- See previous-generation records as read-only context when previous generations are included.
- See when an older record with the same primary key is overridden by a newer selected generation.
- Open project export from the left pane without treating export as an operation on the selected table.
- Scroll a long table list without losing access to the product title, generation selector, Export, or schema editing actions.
- Open a configured visual or domain-specific editor plugin for the selected record when grid editing is a poor fit.
- Open a configured plugin for one selected record, one selected grouping key, or the whole table depending on the plugin entry mode.
- Return from a plugin editor without losing table, generation, display mode, or dirty-state protections.

## States

- No repository or workspace selected.
- Table cannot open because schema is invalid.
- Table cannot open because `masterdata/generations/0000_initial/_config.yaml` is missing or invalid.
- Table selected with no generation filter.
- First slice: table selected in the only configured generation.
- Later slice: table selected with one or more generations enabled.
- Active edit generation selected.
- Display mode: active generation only.
- Display mode: active generation plus previous output generations.
- Previous-generation rows shown read-only.
- Older row overridden by a newer row with the same primary key.
- Older row visible but not exported because it is overridden.
- Record editing with valid changes.
- Record editing with validation errors.
- Pending frontend-only changes in `extable` commit mode.
- Commit confirmation shown because validation errors exist.
- External reference lookup with no match, one match, or ambiguous matches.
- External reference selected with display label resolved but canonical stored value remaining as the referenced primary key.
- Save ready.
- Save allowed with validation errors after user confirmation and `force: true`.
- Save blocked by validation errors when configured.
- Later slice: export ready.
- Later slice: export blocked by validation errors.
- Plugin editor available for selected table or record.
- Plugin record-entry selection row selected.
- Plugin group-entry grouping row selected.
- Plugin table-entry action opened without intermediate selection.
- Plugin editor loading, loaded, dirty, validation-error, save-ready, and load-failed states.

## Invoked APIs

- Load table schemas.
- Load YAML records by table and generation through `/api/tables/:table/generations/:generationId/records`.
- Later slice: load a generation-aware table view that includes active-generation records plus optional previous-generation records.
- Validate table data in the frontend while changes are pending.
- Resolve external reference candidates through `/api/tables/:table/references`.
- Commit pending row operations through `POST /api/tables/:table/generations/:generationId/records/commit`.
- On commit with validation errors, show a confirmation dialog before retrying with `force: true`.
- Notify users when saving with validation errors is allowed by explicit confirmation.
- Block saving when validation errors exist and the table or project configuration requires strict save validation.
- Later slice: merge selected generations for export and override status display.
- Later slice: run export adapter.
- Later slice: discover editor plugins applicable to the selected table or record.
- Later slice: load scoped plugin data and commit plugin change sets through host APIs.

## Table Component

- Use npm package `extable` as the primary grid component.
- Prefer `@extable/react` for the React web service implementation.
- Convert canonical records to `extable` rows in the frontend.
- Convert `extable` pending operations back to canonical commit operations in the frontend.
- Every editable primary key column must set the `extable` column `unique` flag so duplicate primary key values are reported while editing.
- Place `extable` as the main center-pane surface, not as a secondary widget inside another large panel.
- Keep the left pane dedicated to active generation context and vertical table navigation.
- Structure the left pane as a vertical flexbox with fixed top controls, flexible table navigation, and fixed bottom project actions.
- The fixed top controls include the product title, active edit generation selector, and generation metadata edit icon.
- The flexible middle region contains the table list and is the only sidebar region that scrolls when tables exceed available height.
- The fixed bottom project action region contains the project-level `Export` button and the schema/settings action.
- The sidebar container itself should constrain height to the viewport and avoid scrolling during normal table editing.
- The top and bottom sidebar regions must not shrink or scroll away when the middle table list overflows.
- Place the active edit generation selector in the left pane, above the table list.
- Place a compact pencil/edit icon button adjacent to the active edit generation selector. Activating it navigates to `/generations/edit`.
- The pencil/edit icon must expose an accessible label and tooltip such as `Edit generations`.
- If there are unsaved table edits, activating the pencil/edit icon must use the same dirty-state confirmation flow as any other navigation away from table editing.
- Place a compact schema/settings icon button in the left pane near the table navigation header or footer. Activating it navigates to `/schemas`.
- The schema/settings icon must expose an accessible label and tooltip such as `Edit schemas`.
- If there are unsaved table edits, activating the schema/settings icon must use the same dirty-state confirmation flow as any other navigation away from table editing.
- Place one project-level `Export` button in the left pane, outside the table list and outside the table toolbar.
- The project-level `Export` button must remain available regardless of which table is selected.
- The project-level `Export` button must not be duplicated in the right pane, table footer, table top bar, or selected-table inspector.
- Activating the project-level `Export` button opens an export dialog that asks the user to choose an output destination before export execution.
- The export dialog may initialize generation selection from current output-enabled generations, but it must not make export look like a selected-table operation.
- The export dialog must initialize format option controls from persisted project export settings for the selected logical format.
- When the user changes export options in the dialog and runs check or export, those options are saved as the next defaults for that logical format.
- Persisted export options include temporal formatting and timezone values when supported by the selected format.
- The export dialog must keep explicit generation selection separate from persisted format options.
- The active edit generation selector must not be duplicated in the table top bar.
- The table top bar may contain the generation display mode control, save controls, grid mode controls, and table-specific actions.
- The table top bar may contain a compact plugin action when one or more editor plugins apply to the selected table or selected record.
- A `record` entry plugin action is enabled only when exactly one editable or readonly context record row is selected as the entry record.
- A `group` entry plugin action opens or focuses a grouping `extable` selection surface where each row represents one distinct grouping key.
- A `table` entry plugin action opens the plugin immediately without showing an intermediate `extable` selection grid.
- The grouping selection surface is not canonical data editing; it is a derived chooser for opening a plugin scope.
- Grouping rows should show the grouping value, optional display label, record count, and validation status summary when available.
- Selecting a grouping row must preserve the selected table, active generation, and display mode.
- Plugin actions must be visually distinct from table navigation rows and project-level actions.
- Opening a plugin editor for the selected record must preserve the active generation, display mode, selected table, and selected record context.
- Opening a plugin editor from a grouping row must preserve the active generation, display mode, selected table, and selected grouping key.
- Opening a table-entry plugin must preserve the active generation, display mode, and selected table.
- If the table grid has unsaved edits, opening a plugin editor must use the same dirty-state confirmation flow as other navigation away from table editing.
- If the plugin editor has unsaved edits, returning to the grid must use the same dirty-state confirmation flow before discarding plugin changes.
- The plugin editor should reuse shared frontend chrome for Save, Revert, diagnostics, and return-to-grid controls where practical.
- The plugin editor must not hide the existence of the ordinary table editor; users must be able to return to the grid for the same records.
- Do not present `Data editing`, `Generation editing`, or `Schema editing` as peer navigation items in the table list; those page labels are too visually similar to table choices and make the left pane ambiguous.
- Keep the grid area large enough for spreadsheet-like editing across many columns and rows.
- Do not embed generation metadata editing controls or a generation metadata grid in the table editing workspace.
- Do not embed schema metadata editing controls or a schema field grid in the table editing workspace.
- Provide compact controls for generation display mode without duplicating active edit generation selection.
- The generation display mode control offers:
  - `active_only`: show and export only records from the active edit generation.
  - `include_previous`: show records from output-enabled generations older than or equal to the active edit generation, plus the active edit generation even if its own `output` flag is false.
- When `include_previous` is enabled, the grid includes a read-only generation/source column.
- Rows whose `sourceGenerationId` is not the active edit generation are read-only.
- Rows whose `sourceGenerationId` is the active edit generation remain editable, subject to schema validation.
- A row from an older generation that is overridden by a newer visible row with the same primary key should remain visible but marked as overridden.
- Overridden rows are not included in the effective export result.
- Override indicators should be visible at row level and should name the winning generation when practical.
- Users should not be able to accidentally edit or delete previous-generation rows from the active generation's table editor.
- Inserted rows are always created in the active edit generation.
- Commit operations must include only changes to rows in the active edit generation.
- Use `extable` column schemas for scalar, enum, boolean, date, time, datetime, and lookup-like reference fields.
- Map external reference fields to an `extable` lookup-capable display type, such as `labeled`, when the grid stores `{ label, value }` objects for display while canonical commits store only `value`.
- Lookup candidate labels should include both the referenced display name and primary key, such as `Product Team (org-product)`, to disambiguate similarly named records.
- After a lookup candidate is selected, the grid cell should display the referenced display name only, while the commit payload and YAML output retain the referenced primary key.
- In `include_previous` mode, external reference lookup candidates are built from every generation that participates in the visible/export-effective generation set, not only from the active edit generation.
- External reference lookup candidates must exclude records from generations whose `output` flag is false, even if those records are visible in the table editor because they belong to the active edit generation.
- If multiple export-enabled generations contain the same referenced primary key, the lookup list should expose the effective winning candidate and include generation/source metadata for explanation.
- Lookup candidate labels should remain human-readable and disambiguated; when generation context is necessary, include source generation information in the candidate metadata or a tooltip rather than making the main label noisy.
- Render formula fields as read-only columns.
- Omit direct editors for formula fields because their values come from schema formula evaluation.
- Show formula evaluation errors as diagnostics on the formula field cell.
- Formula field behavior may be disabled or hidden in the first runnable slice if formula evaluation is deferred.
- Non-exported fields may still be visible and editable when the schema uses them as planning inputs or formula inputs.
- Use row or cell diagnostics to surface validation errors without requiring users to open raw YAML.
- Project validation errors into `extable` cell diagnostics or validation markers so users see Excel-like error indicators on the affected cell, not only in a separate diagnostics panel.
- Preserve spreadsheet-like workflows such as keyboard navigation, copy/paste, and direct cell editing where supported by `extable`.
- Use `extable` commit mode so edits remain frontend-local until the user presses the commit button.
- Run validation while changes are pending in the frontend.
- Keep pending insert, update, delete, and move operations locally until commit.
- Preserve YAML record order when users insert, delete, or move rows.
- Allow users to edit primary key cells; the change is submitted as a row update.
- Do not open the table editor when the table schema is invalid.
- Do not open the table editor when the active generation `_config.yaml` is missing or invalid.
- If the table YAML file is missing for the active generation, open an empty table and create the YAML file on first commit.
- The editor layout must constrain page height so the browser page itself does not scroll during normal table editing. The left navigation, toolbar, grid viewport, and footer should be arranged with grid or flexbox, and table overflow should remain inside the `extable` viewport.
- Large table catalogs must not push `Export` or `Edit schemas` below the viewport; only the table navigation list receives overflow scrolling.

## Generation-Aware Data View

- The active edit generation is a single generation chosen by the user.
- `active_only` mode loads and edits only that generation's table YAML.
- `include_previous` mode loads output-enabled generations older than or equal to the active edit generation, plus the active edit generation even if its own `output` flag is false.
- The editing API must not include generations newer than the active edit generation.
- Previous generations are ordered old-to-new by configured generation ordering.
- The grid row model includes `sourceGenerationId`, `sourceGenerationLabel`, `isActiveGeneration`, `isReadOnly`, `isOverridden`, and optional `overriddenByGenerationId`.
- The server adds the generation/source metadata to each row returned for editing.
- The generation/source column is system-managed and read-only.
- The UI uses `isReadOnly` from the API response to lock rows.
- Any row whose `sourceGenerationId` is not the active edit generation must be readonly.
- Only rows whose `sourceGenerationId` is the active edit generation can be edited, deleted, or committed.
- The active generation's records can override older records with the same normalized primary key.
- If several previous generations contain the same primary key, only the newest visible winner is part of the effective export result; older duplicate rows are marked overridden.
- The UI can still show overridden rows for traceability, but visual treatment should make clear that they are historical context rather than exported effective records.
- External reference lookup in `include_previous` mode resolves candidates from output-enabled generations older than or equal to the active edit generation.
- Reference lookup in `active_only` mode resolves candidates only from the active edit generation when that generation is output-enabled.
- If the active edit generation has `output: false`, its rows may remain visible and editable in the active table view, but they must not be offered as external reference candidates.
- Reference candidate metadata should include `sourceGenerationId` and may include `sourceGenerationLabel` and `overrodeGenerationIds` so lookup UIs can explain which generation supplied the effective candidate.

## Components

- [Single page application shell](../ui-flow/single-page-application-shell.md)
- [Shared web editing frontend](../component/shared-web-editing-frontend.md)
- [HTML editor plugin runtime](../component/html-editor-plugin-runtime.md)
- [Schema validation engine](../component/schema-validation-engine.md)
- [Export backend adapters](../component/export-backend-adapters.md)
- [Web service host](../server-component/web-service-host.md)

## Related Requirements

- [Generic master data model](../data-model/generic-master-data-model.md)
- [Table schema model](../data-model/table-schema-model.md)
- [Editor plugin model](../data-model/editor-plugin-model.md)
- [Export settings model](../data-model/export-settings-model.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)

## Native-Language Summary

表とレコードを中心に編集する主画面。世代選択は左ペインに一元化し、世代メタデータ編集は世代セレクタ横の鉛筆アイコンから別画面に遷移する。左ペインは固定のタイトル/世代選択、可変のテーブル一覧、固定の Export/Edit schemas の3パート構成にし、テーブルが大量にある場合は中央のテーブル一覧だけをスクロールする。Export は選択中テーブルではなくプロジェクト全体の操作なので左ペインに1つだけ置き、右ペインや表ツールバーには置かない。Export ダイアログは形式別の保存済みオプションを初期値にし、ユーザーが操作した日時形式や timezone などを次回の初期値として保存する。データ編集ではアクティブ編集世代を選び、必要に応じて過去世代を readonly の文脈として表示する。過去世代の同一主キーが新しい世代で上書きされている場合は、上書き状態を行で分かるようにする。
