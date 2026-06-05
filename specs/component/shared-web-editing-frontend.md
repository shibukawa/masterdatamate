---
id: "shared-web-editing-frontend"
type: "ui-flow"
title: "Shared web editing frontend"
aliases: []
tags: ["frontend", "web"]
facts:
  lifecycle.status: "blueprint"
---

# Shared web editing frontend

## Summary

The shared web editing frontend is the reusable React + Vite UI core for editing schema-driven master data. The first host is a web service, and the core should remain usable by later editor-extension and standalone-app shells.

## Scope

- In scope:
  - Table navigation and record editing.
  - Schema-aware editors for scalar fields, constants, and external references.
  - Inline validation diagnostics from [Schema validation engine](schema-validation-engine.md).
  - Generation selection and export target selection UI.
  - YAML-backed load and save integration through host-specific adapters.
  - Integration with npm package `extable` as the primary Excel-like table editing component.
  - Optional HTML editor plugin surfaces for schema-defined data that is difficult to edit as a grid.
  - Optional in-app AI assistant panel for natural-language guidance, tool-backed analysis, proposed changes, and confirmed execution.
  - SPA navigation across table editing, generation selection, generation editing, and schema editing pages.
- Out of scope:
  - GitHub approval workflow UI.
  - Repository permission management.
  - Backend-specific export implementation details.
  - Schema editor UI for the first runnable slice.

## Rules / Constraints

- The editing core should avoid assumptions about a single hosting mode.
- Host integrations provide file access, repository operations, authentication, and export execution.
- Non-engineering users should be able to edit records without understanding YAML syntax.
- Developers should still be able to review and modify generated YAML directly.
- The first implementation uses React + Vite.
- The packaged web server uses the Vite production build embedded in a Go binary.
- The Wails desktop app uses the same React + Vite frontend through a desktop runtime adapter.
- React components should call a host adapter layer for application operations so web-server HTTP mode and Wails binding mode can share UI behavior.
- The first implementation should use the React wrapper from `@extable/react`.
- Custom editor plugins are alternative editing surfaces, not alternate storage systems. They must read and write through the same host adapter boundary as the ordinary table editor.
- The shared frontend should discover applicable editor plugins for the selected table or record from [Editor plugin model](../data-model/editor-plugin-model.md) declarations.
- HTML editor plugins are loaded through the [HTML editor plugin runtime](html-editor-plugin-runtime.md).
- The shared frontend may render the [In-app AI assistant panel](../ui-screen/in-app-ai-assistant-panel.md) as a dock, drawer, or focused route.
- The AI panel must send scoped context through the host adapter rather than reading canonical YAML files directly.
- The AI panel must render proposed changes and side-effecting tool confirmations through host-owned UI controls.
- The AI panel must not bypass the same validation, save confirmation, generation, schema, and export rules used by ordinary screens.
- AG-UI events may drive assistant timeline, tool progress, state updates, and confirmation states, but generated UI must not replace host-owned save or export controls.
- Opening a plugin surface must use the same dirty-state confirmation flow as navigation away from an `extable` commit-mode surface.
- Plugin edits must participate in the same pending-edit, validation, confirmation, save, revert, and reload lifecycle as table grid edits.
- Plugin save controls may be rendered by the shared frontend chrome even when the plugin provides its own domain-specific controls inside the isolated editor surface.
- The frontend must keep a user-reachable path back to ordinary table editing for every plugin-edited table.
- `vendor/extable` is reference material only; application dependencies should come from npm packages.
- The main editing screen should use a two-pane layout with active generation context and vertical table navigation on the left and `extable` as the central working surface.
- Active edit generation selection belongs in the left pane and must not be duplicated in the table editing toolbar.
- Generation metadata editing is opened from a compact pencil/edit icon next to the left-pane generation selector.
- The generation edit icon must provide an accessible name and tooltip such as `Edit generations`.
- Schema editing is opened from a compact schema/settings icon in the left pane near the table navigation header or footer.
- The schema/settings icon must provide an accessible name and tooltip such as `Edit schemas`.
- The schema/settings icon must not be rendered as a normal table navigation row.
- Schema editing is available to any app user; the frontend does not hide it behind a separate schema-authoring permission or feature flag.
- Generation metadata editing uses a focused, modal-like SPA route rather than a normal peer item in the left navigation.
- Generation metadata editing must use `extable` for the metadata grid so batch editing, copy/paste, and commit-mode behavior are consistent with table data editing.
- Generation metadata editing must not include an active-generation `Edit` radio-button column.
- Generation metadata editing includes a leftmost checkbox column in the `extable` grid for administrative row selection; the checkbox column has no text title.
- Generation metadata row selection must remain separate from the active edit generation selector.
- Generation metadata row selection must not be implemented as a separate side panel.
- The generation metadata grid default columns are selection, generation index, path name, output, and description.
- The output checkbox appears to the right of path name.
- Separate Label and Folder columns are omitted by default because their values are derived from generation index and path name.
- The generation index column uses the `extable` `unique` flag.
- Generation metadata commit/save controls are disabled while duplicate generation indexes or invalid metadata cells are present.
- The persistent generation `Merge` action is enabled only when two or more generation metadata rows are selected and the metadata grid has no pending edits.
- The persistent generation merge dialog collects destination generation metadata and calls the host merge API; it must not attempt to merge or write generation folders in browser-local state.
- The generation metadata commit/save action must always show a confirmation dialog before host save APIs are called.
- The generation metadata `Delete` action is enabled only when one or more persisted generation rows are selected and the metadata grid has no pending edits.
- The generation metadata delete dialog calls the host delete API; it must not delete folders directly from browser-local state.
- The generation metadata `Duplicate` action is enabled when one or more persisted generation rows are selected and the metadata grid has no pending edits.
- The generation metadata duplicate action immediately calls the host duplicate API without an input or confirmation dialog; it must not copy generation folders directly from browser-local state.
- The generation metadata `Analyze` action is enabled when one or more persisted generation rows are selected and the metadata grid has no pending edits.
- The generation metadata analyze view calls the host analyze API and remains read-only.
- Generation metadata commit/save, persistent merge, and delete actions must require confirmation dialogs because they change canonical project files.
- Generation metadata duplicate is the explicit exception: it writes canonical generation folders immediately using server-side automatic index and path-name derivation.
- Generation metadata display labels should separate the sortable prefix from the label, such as `(0010) hidden_refs`, while preserving the canonical folder name such as `0010_hidden_refs` as filesystem metadata.
- The focused generation editing chrome must not repeat the current active generation label in the left pane or top bar.
- The left navigation must not mix page-level `Data editing` / `Generation editing` buttons with table choices because those controls are visually too similar to table selection.
- Navigation between table editing and generation editing must preserve dirty-state guards and restore the previous table, active generation, display mode, and practical grid context when returning.
- Generation selection and schema editing are separate SPA pages rather than modal overlays on the main editing grid.
- Schema editing uses dedicated list and detail SPA routes and must use `extable` commit mode for both table-level metadata and field-level detail editing.
- The schema list grid default columns are selection, table `system_name`, `export`, readonly `primary_key`, readonly `references`, and `comment`.
- The schema detail grid uses one `extable` surface for `primary_key`, `reference`, `data`, and `formula` rows instead of separate panels per field kind.
- Schema detail field rows should include editors for `system_name`, `type`, `formula`, `reference_table`, `default_value`, `export`, `required`, and `comment`, with editability controlled by row kind.
- Schema field `system_name` columns use the `extable` `unique` flag within a table schema.
- Schema editing commit/save controls are disabled while duplicate field names or invalid schema cells are present.
- Schema editing commit/save and schema or field delete actions must require confirmation dialogs before host save APIs are called.
- Schema editing must provide local `Revert` commands that discard uncommitted `extable` edits without calling save APIs.
- Schema editing must provide local `Undo` and `Redo` commands for uncommitted `extable` edits.
- Schema editing `Undo` and `Redo` affect only frontend-local pending operations and must not call host save APIs.
- Schema editing `Undo` and `Redo` should cover cell edits, row insertions, row deletion preparations, row moves, row kind changes, table rename edits, and primary-key ordering changes when those operations are supported by the current grid.
- A new edit after `Undo` clears the redo stack.
- `Revert` remains a separate destructive local action that discards all pending schema edits after confirmation.
- Schema field deletion must call a host API that updates schema YAML and removes the deleted column from existing table data; the browser must not directly mutate YAML files.
- Schema table deletion deletes schema files only and must not delete generation data files from browser-local state.
- Schema table `system_name` changes are rename operations and must be confirmed with old and new names before host save APIs are called.
- Schema table rename includes schema file rename and matching generation table data file rename.
- Schema duplication is not provided.
- New schema creation starts as a blank draft rather than an auto-populated sample schema.
- Field `system_name` changes, including primary key field names, are rename operations that update corresponding YAML keys through host APIs.
- New schema field rows may prefill `default_value` values from schema when defined; table data loading materializes defaults into editable row values, and later saves write those values normally.
- Field type changes that make existing values invalid require a warning confirmation before save.
- Primary key definition changes use the same schema commit confirmation as other schema changes.
- Changing an external reference target keeps stored reference primary-key values unchanged and surfaces unresolved values as validation errors.
- Formula authoring is disabled until formula implementation is delivered.
- MasterDataMate schema fields should be mapped into `extable` column schema types where possible: string, number, int, boolean, date, time, datetime, enum, and lookup-style editors.
- The frontend owns the conversion boundary between canonical API/YAML record shape and the `extable` row model.
- Server APIs return canonical records and diagnostics; the frontend adapts them to `extable` schemas, rows, pending operations, and diagnostics.
- External references should use `extable` lookup editor hooks when possible so users select referenced rows by human-readable labels while storing only primary key values.
- External reference fields that are represented as `extable` display objects must use a compatible `extable` column type, such as `labeled`, rather than `string`, because native `extable` validation treats object values in string columns as invalid.
- The frontend should adapt canonical external reference values into display objects such as `{ label, value }`, where `label` is the referenced record display name and `value` is the referenced primary key. Before validation commit and YAML save, the frontend must normalize that object back to `value`.
- Lookup dropdown candidate labels should include the display name and primary key, for example `Product Team (org-product)`. The selected cell display should use only the display name when the reference is resolved.
- Lookup candidates from generation-aware APIs must exclude records from generations whose `output` flag is false.
- Lookup candidate metadata should include source generation information when the candidate comes from a generation-aware lookup. The UI may surface that metadata in a tooltip or secondary text so users can understand which generation supplied the effective candidate.
- Validation diagnostics from MasterDataMate must be projected into table row or cell state; rows with unresolved or ambiguous external references should be visually marked as error rows.
- Frontend validation should use the same canonical value extraction as save conversion. For lookup display objects, validation must compare the stored primary-key `value`, not the display `label` or the whole object.
- Primary key columns in ordinary data editing use the `extable` `unique` flag so duplicate key input is surfaced immediately in the grid.
- The first implementation uses `extable` commit mode.
- Pending edits are held in frontend state until the user commits.
- Any commit-mode editing surface that can hold pending edits should provide a local `Revert` command when users need to discard uncommitted changes.
- Revert must ask for confirmation before discarding pending edits.
- Confirmed Revert restores the last loaded or last committed state without calling save APIs.
- Navigation away from a dirty commit-mode surface, including generation editing back navigation, must ask for confirmation before discarding pending edits.
- Frontend validation runs against pending edits before and during commit.
- If pending edits have validation errors, the UI asks for confirmation before saving with `force: true`.
- The first slice only needs one active generation, but the UI model should not make multi-generation support impossible.
- Formula authoring and formula evaluation may be deferred after the first runnable slice; while deferred, schema editing should disable formula row creation and formula cell editing.
- The table editor should keep all normal table scrolling inside the `extable` viewport. The application shell should use explicit viewport-height grid or flexbox constraints so the page footer does not create a second browser-level scrollbar.

## Dependencies

- [Generic master data model](../data-model/generic-master-data-model.md)
- [Schema validation engine](schema-validation-engine.md)
- [HTML editor plugin runtime](html-editor-plugin-runtime.md)
- [AI assistant service](ai-assistant-service.md)
- [Agent tool contract](agent-tool-contract.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)
- [Generation persistent merge flow](../data-flow/generation-persistent-merge-flow.md)
- [Generation deletion flow](../data-flow/generation-deletion-flow.md)
- [Generation duplication flow](../data-flow/generation-duplication-flow.md)
- [Generation analysis flow](../data-flow/generation-analysis-flow.md)
- [Web service host](../server-component/web-service-host.md)
- [Go embedded web server host](../server-component/go-embedded-web-server-host.md)
- [Wails desktop host](../server-component/wails-desktop-host.md)
- [Single page application shell](../ui-flow/single-page-application-shell.md)

## Related Documents

- [Table editing workspace](../ui-screen/table-editing-workspace.md)
- [Generation selection screen](../ui-screen/generation-selection-screen.md)
- [Generation editing screen](../ui-screen/generation-editing-screen.md)
- [Schema editing screen](../ui-screen/schema-editing-screen.md)
- [In-app AI assistant panel](../ui-screen/in-app-ai-assistant-panel.md)

## Native-Language Summary

最初はReact + ViteのWebServiceとして提供する。配布時はViteビルドをGoバイナリへ埋め込むWebサーバー版と、同じUIをWailsに載せるデスクトップ版を用意する。UIの中心にはnpm package版 `extable` のExcelライクなテーブルコンポーネントを使う。メイン編集は左テーブル一覧と中央グリッドの2ペイン、世代選択・世代編集・スキーマ編集はSPAの別ページとして扱う。
