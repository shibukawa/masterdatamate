---
id: "generation-editing-screen"
type: "ui-screen"
title: "Generation editing screen"
aliases: []
tags: ["ui", "generation", "editing"]
facts:
  lifecycle.status: "blueprint"
---

# Generation editing screen

## Summary

The generation editing screen is a dedicated focused workspace for generation metadata and ordering. It manages generation folder creation, derived folder renaming, output defaults, and descriptions with an `extable` grid. It is separate from the table data editing workspace so generation administration does not reduce or distract from the primary data grid.

The screen behaves like a modal step in the editing flow even though it has its own route: users enter it from the pencil/edit icon next to the sidebar generation selector, complete generation administration, and return to table data editing.

Generation metadata editing uses the same `extable` interaction model as table data editing so users can batch edit, copy, paste, and commit multiple generation metadata rows consistently.

Generation output inclusion is edited only as a column in the generation metadata grid. The app must not show a persistent standalone generation export/output checkbox elsewhere in the application chrome.

## User Goals

- Create a new generation.
- Edit generation index values that define ordering.
- Edit whether each generation is included in output.
- Edit the path name used as the folder-name suffix.
- Rename or describe a generation through the description field.
- Select multiple generation rows and merge them into a newly created generation.
- Select one or more generation rows and delete them after explicit confirmation.
- Select one or more generation rows and duplicate them into new generations immediately.
- Analyze selected generations to inspect table counts, record counts, diagnostics, and merge impact before changing files.
- Review pending generation metadata changes before committing them.
- Configure or inspect the global ordering mode and numeric digit width.
- See which YAML path stores the generation's table data.
- See a human-friendly generation display label without exposing raw folder-name punctuation as the primary UI label.
- Save generation metadata back to canonical files.
- Revert pending generation metadata edits before they are committed.
- Navigate back to table data editing without losing active table and active edit generation context.
- Return to the same table editing context that opened generation editing.

## States

- First slice: screen not available.
- Later slice: single generation metadata shown read-only or minimally editable.
- Editing generation metadata in an `extable` grid.
- Creating a generation folder and its generation `_config.yaml`.
- Numeric ordering mode with zero-padded folder prefixes.
- Release-date ordering mode.
- Reordering generations by changing `generation_index`.
- Renaming the physical generation folder when `generation_index` or `path_name` changes.
- Dirty metadata changes.
- Dirty metadata changes with navigation blocked until the user confirms or saves.
- Save ready.
- Commit confirmation dialog open with a summary of pending generation metadata changes.
- Revert confirmation shown because pending metadata edits exist.
- Pending metadata edits reverted to the last loaded or last committed generation metadata snapshot.
- Save blocked by invalid generation order or duplicate generation ID.
- Save blocked by duplicate generated folder name.
- Save blocked by invalid path name.
- Generation data path missing or unreadable.
- Multiple generation rows selected.
- Merge command enabled because two or more generation rows are selected.
- Merge command disabled because fewer than two generation rows are selected.
- Merge command disabled because pending generation metadata edits exist.
- Merge dialog open with selected source generations and destination generation metadata fields.
- Merge validation failed because the destination generation metadata collides with an existing generation.
- Persistent merge succeeded and the newly created generation is highlighted in the grid.
- Delete command enabled because one or more persisted generation rows are selected.
- Delete command disabled because no persisted generation rows are selected.
- Delete command disabled because pending generation metadata edits exist.
- Delete confirmation dialog open with selected generation folders and warnings.
- Delete validation failed because the selected set would remove every generation.
- Delete succeeded and generation metadata has reloaded.
- Duplicate command enabled because one or more persisted generation rows are selected.
- Duplicate request running immediately without an input or confirmation dialog.
- Duplicate succeeded and all generation row selection is cleared.
- Analyze command enabled because one or more persisted generation rows are selected.
- Analyze results open with table counts, record counts, diagnostics, and merge impact.

## Invoked APIs

- Load generation metadata.
- Load generation global settings.
- Create generation metadata.
- Update generation metadata.
- Create generation folder and generation `_config.yaml`.
- Rename generation folder when derived `folder_name` changes.
- Reorder generations by sorted generation index.
- Validate generation metadata.
- Save generation configuration.
- Revert generation metadata edits locally without writing YAML files.
- Persistently merge selected generations into a new generation.
- Delete selected generations.
- Duplicate one or more selected generations into new generations.
- Analyze selected generations.

## Table Component

- Use npm package `extable` as the primary editor for generation metadata.
- The generation metadata grid must be an `extable` grid rather than a bespoke HTML table.
- The grid should use `extable` commit mode so multiple generation rows can be edited before saving.
- Pending generation metadata edits stay local until the user presses the commit/save button.
- Pressing the commit/save button must open a confirmation dialog before any generation metadata save API is called.
- The commit confirmation dialog summarizes added, updated, renamed, reordered, and removed metadata rows.
- The commit confirmation dialog must identify physical folder renames and newly created folders before the user confirms.
- Cancelling the commit confirmation leaves pending edits unchanged and performs no writes.
- Confirming the commit dialog is the only path that calls generation metadata save APIs.
- A `Revert` command must be available when pending generation metadata edits exist.
- Activating `Revert` shows a confirmation dialog before discarding edits.
- Confirmed `Revert` restores the grid to the last loaded or last committed metadata snapshot, clears generation dirty state, and does not call save APIs or write YAML files.
- Cancelling the `Revert` confirmation leaves pending edits unchanged.
- Columns should include a leftmost row selection checkbox column, `generation_index`, `path_name`, `output`, and `description`.
- The row selection checkbox column has no text title.
- The row selection checkbox column is interactive while the grid has no pending metadata edits.
- The row selection checkbox column is disabled or ignored while generation metadata edits are dirty.
- The `output` checkbox column appears to the right of `path_name`.
- The `output` checkbox column is the only always-editable UI for generation default export inclusion.
- A standalone `Export` or `Output` checkbox must not be shown persistently in the table editing sidebar, table toolbar, right pane, or generation editing chrome.
- Users who need to change generation default export inclusion do so by editing the `output` column in this generation metadata grid and committing the metadata change.
- The generation metadata grid should not show separate `Label` or `Folder` columns by default because they duplicate information derivable from `generation_index` and `path_name`.
- The grid must not include an `Edit` radio-button column. Active edit generation selection belongs to the table editing sidebar, not to generation metadata editing.
- The grid includes row selection controls inside the table component for generation administration actions.
- Row selection controls must not live in a separate side panel.
- Row selection is independent from the active edit generation used by table data editing.
- Selecting rows in the generation metadata grid must not change the active edit generation in the table editing workspace.
- `generation_index` uses a number editor in numeric mode and a date editor in release-date mode.
- `generation_index` must set the `extable` column `unique` flag so duplicate indexes are reported while editing.
- Commit/save is disabled while the grid has duplicate `generation_index` values or invalid generation metadata cells.
- `output` uses a boolean checkbox-style editor.
- `path_name` edits the generation label/suffix used to derive the folder name.
- The canonical physical folder name uses the configured sortable prefix and path name, for example `0010_hidden_refs`.
- The human-facing generation display label should be formatted as `(0010) hidden_refs` in numeric mode: the zero-padded generation index in parentheses followed by the path label.
- The UI may keep underscores in the displayed path label when that is the canonical path segment, but it must visually separate the numeric prefix from the label rather than showing the raw folder name as the primary label.
- The raw derived folder name can appear in confirmation dialogs or secondary detail when useful for filesystem clarity, but it is not a default grid column.
- Rows should be sorted by `generation_index` ascending so old generations appear before new generations.
- The generation editing page should be the only screen that edits generation metadata; the table data editing workspace may link to it but must not embed this grid.

## Recommended Administration Actions

- Bulk delete selected generations.
- Duplicate one or more selected generations into new generations with automatic `generation_index` and `path_name` derivation.
- Analyze selected generations with a read-only summary showing tables, record counts, diagnostics, and merge impact.
- These actions should use the same row selection model as merge and delete.
- Actions that write files, rename folders, delete folders, or change metadata must require a confirmation dialog unless a flow explicitly defines a no-confirmation exception.
- Duplicate is the no-confirmation exception and runs immediately with server-side automatic destination metadata.
- Non-writing preview actions may run without confirmation.

## Confirmation Policy

- Generation editing commit/save must always show a confirmation dialog before writing metadata or renaming/creating folders.
- Persistent merge must always show a confirmation dialog before creating the destination generation folder.
- Duplicate creates destination generation folders immediately without an input dialog and without a confirmation dialog.
- Delete must always show a confirmation dialog before deleting generation folders.
- Analyze does not require a confirmation dialog because it is read-only.
- Confirmation dialogs must show the operation name, affected generation labels, affected folder paths, and whether the operation creates, renames, overwrites, or deletes filesystem paths.
- Confirmation dialogs must expose blocking diagnostics before the destructive or persistent action is submitted.
- Confirmation dialogs must have an explicit cancel action that leaves local state unchanged and performs no API call.
- Confirmation dialogs must prevent duplicate submission while the API request is running.
- Confirmation dialogs for delete and merge should be visually stronger than ordinary navigation guards because they change canonical project files.

## Persistent Merge Action

- The top bar includes a `Merge` command for persistently merging selected generation rows.
- `Merge` is enabled only when at least two generation rows are selected and the generation metadata grid has no pending edits.
- `Merge` is disabled when fewer than two rows are selected.
- `Merge` is disabled while generation metadata edits are dirty because source row identity, index ordering, and folder names may be stale.
- If users need to merge after editing generation metadata, they must save or revert those edits first.
- Activating `Merge` opens a dialog rather than immediately writing files.
- The merge dialog shows selected source generations sorted by `generation_index` ascending.
- The dialog must make precedence visible: larger numeric indexes or later release-date indexes win when duplicate primary keys exist.
- The dialog collects destination generation metadata: `generation_index`, `path_name`, `output`, and optional `description`.
- The dialog previews the derived destination `folder_name`.
- The dialog validates destination metadata with the same rules as manual generation creation.
- The dialog should default `output` to true.
- The dialog should prefill `description` with a concise explanation of the selected source generations when the field is empty.
- The dialog must include a final confirmation action that clearly states it will create a new generation folder.
- The confirm action calls `POST /api/generations/persistent-merge`.
- While the API request is running, the dialog prevents duplicate submission and keeps the source selection visible.
- On success, the screen reloads generation metadata and clears all generation row selection so write action buttons return to their unselected disabled state.
- On validation failure, the dialog stays open and displays blocking diagnostics next to the relevant destination fields when possible.
- Persistent merge must not modify or delete the selected source generation rows.

## Delete Action

- The top bar includes a `Delete` command for deleting selected generation rows.
- `Delete` is enabled only when at least one persisted generation row is selected and the generation metadata grid has no pending edits.
- `Delete` is disabled when no persisted generation row is selected.
- `Delete` is disabled while generation metadata edits are dirty because source row identity, derived folder names, and active-generation fallback may be stale.
- Activating `Delete` opens a confirmation dialog rather than immediately deleting files.
- The delete confirmation dialog lists every selected generation display label and derived folder path.
- The delete confirmation dialog must warn when selected generations are output-enabled.
- The delete confirmation dialog must warn when the active edit generation is included and show the replacement active generation if it is known.
- The delete confirmation dialog must block confirmation when the selected set would delete every generation.
- The confirm action calls `POST /api/generations/delete`.
- While the API request is running, the dialog prevents duplicate submission and keeps the source selection visible.
- On success, the screen reloads generation metadata, clears deleted row selection, and updates return context if the active edit generation was deleted.
- On validation failure, the dialog stays open and displays blocking diagnostics.

## Duplicate Action

- The top bar includes a `Duplicate` command for copying selected generations into new generations.
- `Duplicate` is enabled when one or more persisted generation rows are selected and the generation metadata grid has no pending edits.
- `Duplicate` is disabled when zero rows or unsaved inserted rows are selected.
- `Duplicate` is disabled while generation metadata edits are dirty because source row identity and destination collision checks may be stale.
- Activating `Duplicate` immediately calls `POST /api/generations/duplicate` without an input dialog and without a confirmation dialog.
- Duplicate assigns destination generation indexes from the current maximum index by +10 per copied generation.
- Multiple selected sources are sorted by generation ordering before destination indexes are assigned so the copied generations preserve the source ordering relationship.
- Duplicate derives each destination path name from the source `path_name` plus `_copy`, adding suffixes such as `_copy2` when needed for uniqueness.
- Duplicate copies each source generation's `output` and `description`.
- The action calls `POST /api/generations/duplicate`.
- On success, the screen reloads generation metadata and clears all generation row selection so the action buttons return to their unselected disabled state.
- On validation failure, the screen keeps the source selection and displays blocking diagnostics.

## Analyze Action

- The top bar includes an `Analyze` command for inspecting selected generation rows.
- `Analyze` is enabled when at least one persisted generation row is selected and the generation metadata grid has no pending edits.
- `Analyze` is disabled while generation metadata edits are dirty because selected row identity and derived folder paths may be stale.
- Activating `Analyze` opens a read-only dialog or side panel.
- The analyze view shows selected generation count, table count, total record count, per-table record counts, output-enabled state, and diagnostics.
- When multiple generations are selected, the analyze view shows merge impact using normal generation precedence, including records that would be overridden by newer selected generations.
- Analyze diagnostics check generation config readability, folder existence, schema loading, table YAML parseability, record normalization, duplicate primary keys within a selected generation, and selected-generation merge impact.
- Analyze is not an export approval step and does not replace strict export validation.
- Analyze may offer shortcuts to `Merge`, `Duplicate`, or `Delete`; `Merge` and `Delete` must still open their own confirmation dialogs, while `Duplicate` runs immediately by design.

## Navigation And Layout

- The route is `/generations/edit`.
- The page uses a focused variant of the application chrome. It replaces normal table navigation with a minimal left pane.
- The left pane shows only a back/return control and minimal product context.
- The left pane must not show the active generation selector while generation metadata is being edited.
- The left pane must not show the current active generation ID or generation label under the product title while generation metadata is being edited.
- The left pane must not show the table list or a `Data editing` / `Generation editing` navigation group.
- The top bar includes create generation, commit/save metadata changes, and revert metadata changes.
- The generation editing top bar must not show the current active generation ID or generation label because generation editing operates on the full generation list, not a single selected generation.
- Return-to-table is available as the primary left-pane control and may also be repeated as a secondary top-bar action if the layout needs it.
- The route preserves the previous active table, active edit generation, and display mode so returning to data editing restores context.
- When practical, returning should also restore table grid scroll position and selected row.
- If generation metadata changes rename the currently active generation, the returning table editing context should follow the renamed generation rather than falling back to a stale generation ID.
- If the previous active generation was deleted or cannot be resolved after editing, returning must choose a valid generation deterministically and notify the user.
- If `/generations/edit` is opened directly without prior table context, the return control navigates to the default table editing route.
- Unsaved metadata changes trigger a navigation guard before leaving the page.
- Pressing the left-pane return/back control while generation metadata edits are pending must show a confirmation dialog before discarding edits and returning.
- If the user cancels the return confirmation, the app stays on the generation editing screen and preserves pending edits.
- If the user confirms the return confirmation, pending metadata edits are discarded locally, no YAML writes occur, and the app returns to the previous table editing context.
- Browser back, the left-pane return control, and any secondary return action use the same unsaved-change guard.

## Components

- [Single page application shell](../ui-flow/single-page-application-shell.md)
- [Generation selection screen](generation-selection-screen.md)
- [Generation model](../data-model/generation-model.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)
- [Generation persistent merge flow](../data-flow/generation-persistent-merge-flow.md)
- [Generation deletion flow](../data-flow/generation-deletion-flow.md)
- [Generation duplication flow](../data-flow/generation-duplication-flow.md)
- [Generation analysis flow](../data-flow/generation-analysis-flow.md)
- [Web service host](../server-component/web-service-host.md)

## Related Requirements

- [Canonical YAML file layout](../data-model/canonical-yaml-file-layout.md)
- [Product overview](../generic/product-overview.md)
- [Generation persistent merge flow](../data-flow/generation-persistent-merge-flow.md)
- [Generation deletion flow](../data-flow/generation-deletion-flow.md)
- [Generation duplication flow](../data-flow/generation-duplication-flow.md)
- [Generation analysis flow](../data-flow/generation-analysis-flow.md)

## Native-Language Summary

世代メタデータと順序をnpm package版 `extable` で編集する画面。テーブル編集画面の世代セレクタ横にある鉛筆アイコンから遷移し、左ペインには戻る操作だけを置く focused/modal-like な画面として扱う。編集対象世代を選ぶ `Edit` ラジオ列は持たず、世代一覧をまとめて編集できる。コミット、マージ、削除など canonical ファイルを書き換える操作は必ず確認ダイアログを挟むが、`Duplicate` は確認なしで即時実行する例外として扱う。複数の世代行を選択すると、未保存のメタデータ編集がない場合だけ `Merge`、`Duplicate`、`Delete`、`Analyze` が有効になる。`Analyze` はテーブル数、レコード数、診断、マージ影響を読み取り専用で確認する。フォルダ名は `0010_hidden_refs` のような正本形式を保つが、Web UI の主表示は `(0010) hidden_refs` のように数値 prefix とラベルを分けて表示する。
