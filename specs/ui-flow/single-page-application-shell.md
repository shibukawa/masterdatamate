---
id: "single-page-application-shell"
type: "ui-flow"
title: "Single page application shell"
aliases: ["SPA shell", "Application shell"]
tags: ["ui", "spa", "layout", "navigation"]
facts:
  lifecycle.status: "blueprint"
---

# Single page application shell

## Summary

The web service frontend is a single page application. The main editing route uses a two-pane layout with table navigation on the left and the `extable` editing grid as the center of the experience. Generation selection, generation editing, and schema editing are separate SPA pages.

The table editing shell presents active generation selection in the left pane only. Generation metadata editing is reached from an edit icon next to that sidebar generation selector, not from a peer navigation item mixed with table links.

Schema editing is also reached from the left pane. It appears as a compact schema/settings action in the table navigation area, not as a normal table row and not as a primary top-bar destination. Template generation definition editing is reached from the same project action area because it configures project-level generated artifacts rather than table schemas or runtime export data.

Schema editing does not have separate application-level permissions or feature gating. Any user who can use the editing app can navigate to schema editing.

Export is a project-level operation, not a table-level action. The main editing shell exposes a single `Export` command in the left pane, outside the table list and outside table-specific toolbars. Activating this command opens an export dialog where the user chooses the output destination and export options before artifact creation starts.

The table editing left sidebar is divided into three vertical flex regions: a fixed header region, a flexible table navigation region, and a fixed project action footer. Only the table navigation region scrolls when many tables exist.

## Scope

- In scope:
  - SPA-level route structure.
  - Persistent application chrome.
  - Page-level navigation between editing, generation selection, generation editing, and schema editing.
  - Sidebar placement for active generation controls.
  - Sidebar placement for project-level export access.
  - Sidebar scroll behavior for large table lists.
  - Shared dirty-state and validation-state navigation guards.
- Out of scope:
  - GitHub approval workflow.
  - Authentication and repository permission UI.
  - Backend export implementation details.

## Routes

| Route | Screen | Purpose |
| --- | --- | --- |
| `/tables/:tableId?` | [Table editing workspace](../ui-screen/table-editing-workspace.md) | Main master data editing screen. |
| `/generations` | [Generation selection screen](../ui-screen/generation-selection-screen.md) | Later slice: choose active generations for preview/export and inspect current selection. |
| `/generations/edit` | [Generation editing screen](../ui-screen/generation-editing-screen.md) | Create, rename, order, and edit generation metadata in a dedicated grid page. |
| `/schemas/:tableId?` | [Schema editing screen](../ui-screen/schema-editing-screen.md) | Edit table schema definitions. |
| `/generate/definitions` | [Template export definition screen](../ui-screen/template-export-definition-screen.md) | Edit project-level Pongo2 template generation definitions. |

## Layout Rules

- The main editing route uses a fixed two-pane layout.
- The left pane lists table names vertically.
- The left pane also owns active edit generation selection for table editing.
- On the table editing route, the left pane uses a vertical flexbox layout with three regions.
- The fixed top region contains the product title and active edit generation controls, including the generation metadata edit icon.
- The flexible middle region contains only the table navigation list and consumes remaining sidebar height.
- The fixed bottom region contains project-level actions, including `Export` and `Edit schemas`.
- If the table list is taller than the available middle-region height, only the middle table navigation region shows a scrollbar.
- The product title, active generation selector, `Export`, and `Edit schemas` controls must remain visible while the user scrolls a long table list.
- The left pane itself must not become the primary scrolling container during normal table editing because that would scroll fixed header or footer controls out of view.
- The active edit generation selector appears only once in the table editing shell.
- The table editing top bar must not duplicate the active edit generation selector from the left pane.
- A compact edit icon, preferably a pencil icon button, appears adjacent to the left-pane generation selector and navigates to `/generations/edit`.
- The generation edit icon must have an accessible name and tooltip such as `Edit generations` so users understand that it opens generation metadata editing, not the selected table.
- A compact schema/settings icon button appears in the left pane near the table navigation header or footer and navigates to `/schemas`.
- The schema/settings icon must have an accessible name and tooltip such as `Edit schemas` so users understand that it opens table schema editing, not a table record grid.
- The schema/settings icon must not appear as a normal table row because that would visually compete with table choices.
- A single project-level `Export` button appears in the left pane, preferably in the sidebar footer or project action area, visually separated from table rows.
- A project-level `Generate definitions` action appears in the project action area and navigates to `/generate/definitions`.
- The `Export` button must not appear at the bottom of the right pane, inside a selected table panel, or as a table-specific action.
- The left-pane `Export` button opens an export dialog instead of immediately writing files.
- The export dialog collects the output destination, export format, and any export options required by [Export execution flow](../data-flow/export-execution-flow.md).
- Confirming the export dialog runs pre-export validation before artifact creation.
- The table navigation list must not include separate `Data editing`, `Generation editing`, or `Schema editing` items that visually compete with table choices.
- The center pane is dominated by the `extable` component.
- Secondary controls should live in top bars, side panels, drawers, or inspectors rather than reducing the grid's primary area.
- Generation selection and schema editing are separate SPA views.
- Schema editing uses a focused route entered from the left pane and keeps a clear return path to the previous table editing context.
- Template generation definition editing uses a focused route entered from the left pane and keeps a clear return path to the previous table editing context.
- Generation metadata editing uses a focused route that feels modal-like relative to table editing: it temporarily replaces normal table navigation chrome, keeps a clear return path, and does not appear as a normal sidebar navigation destination.
- Generation metadata editing must not be embedded inside the table data editing workspace. It uses a dedicated generation editing route with its own table-like grid.
- On the generation editing route, the left pane must not show active generation selection or table navigation. It should show only a return/back control and minimal product context.
- On the generation editing route, the chrome must not display the current active generation label in the left pane or top bar; generation metadata editing is list-level administration.
- Generation routes are not part of the first runnable slice.
- Navigation must preserve unsaved-change warnings when the current page has dirty YAML or schema edits.
- Navigating from table editing to generation editing must run the same dirty-state guard as other route changes.
- Navigating from table editing to schema editing must run the same dirty-state guard as other route changes.
- Returning from generation editing must restore the previous table editing context: active table, active edit generation, generation display mode, and, when practical, grid scroll position and selected row.
- If `/generations/edit` is opened directly with no previous table context, its return control navigates to the default table editing route.
- Browser back and in-app back controls should follow the same dirty-state and context-restoration behavior.

## Shared State

- Active table.
- Active generation selection.
- Active edit generation.
- Data editing generation display mode: active generation only, or active generation plus previous generations.
- Validation summary.
- Dirty state.
- Save behavior policy for validation errors.
- Export dialog state.
- Current workspace/repository root when applicable.

## Related Documents

- [Shared web editing frontend](../component/shared-web-editing-frontend.md)
- [Table editing workspace](../ui-screen/table-editing-workspace.md)
- [Generation selection screen](../ui-screen/generation-selection-screen.md)
- [Generation editing screen](../ui-screen/generation-editing-screen.md)
- [Schema editing screen](../ui-screen/schema-editing-screen.md)
- [Template export definition screen](../ui-screen/template-export-definition-screen.md)
- [Web service host](../server-component/web-service-host.md)

## Native-Language Summary

WebService のUIはSPA。メイン編集画面は左に世代選択とテーブル名一覧、中央に `extable` を大きく置く2ペイン構成。世代選択は左ペインに一元化し、世代編集は世代セレクタ横の鉛筆アイコンから focused/modal-like な別ページへ遷移する。左サイドバーは flexbox で固定ヘッダー、可変テーブル一覧、固定フッターの3パートに分け、テーブルが多い場合は中央のテーブル一覧だけをスクロールさせる。スキーマ編集も左ペインの管理アイコンから別ページへ遷移し、通常のテーブル行としては表示しない。Export はテーブル単位ではなくプロジェクト全体の操作なので左ペインのプロジェクト操作として1つだけ表示し、押下後に出力先を選ぶダイアログを開く。Pongo2 テンプレート generate の定義編集は `Generate definitions` から `/generate/definitions` に遷移する。
