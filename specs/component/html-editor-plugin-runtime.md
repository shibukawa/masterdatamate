---
id: "html-editor-plugin-runtime"
type: "ui-flow"
title: "HTML editor plugin runtime"
aliases: ["custom editor runtime", "plugin editor runtime"]
tags: ["plugin", "editor", "frontend", "html", "javascript"]
facts:
  lifecycle.status: "blueprint"
  owner: "application"
---

# HTML editor plugin runtime

## Summary

The HTML editor plugin runtime lets a project provide an HTML file with embedded or bundled JavaScript that acts as a domain-specific editor for one or more master data tables. The runtime is used when ordinary grid entry is a poor fit, such as editing a map master row together with many map item rows.

The application host owns loading, validation, dirty-state handling, commit confirmation, and YAML persistence. The plugin owns only its visual editing UI and transforms user interactions into table-scoped change proposals.

## Responsibilities

- Discover available editor plugins from [Editor plugin model](../data-model/editor-plugin-model.md) declarations.
- Show applicable plugins for the currently selected table, selected record, or selected grouping row.
- Load the plugin entry HTML in an isolated editor surface.
- Build a scoped data bundle from the active edit generation, selected entry scope, declared target tables, and declared filters.
- Call the plugin entry function with host APIs and initial data.
- Receive plugin change notifications and convert them into pending commit operations.
- Keep plugin edits in the same dirty-state lifecycle as `extable` commit mode.
- Validate pending plugin changes before save.
- Save plugin changes through the same host commit APIs used by table editing.
- Reload canonical records after save so plugin state reflects normalized server output.

## Plugin Entry Contract

Each plugin entry HTML must register one entry function on `window.MasterDataMatePlugin`.

```ts
type PluginEntry = (context: PluginContext) => PluginInstance | Promise<PluginInstance>;

type PluginContext = {
  pluginId: string;
  generationId: string;
  mode: "active_only" | "include_previous";
  entry: PluginEntryScope;
  tables: Record<string, PluginTableData>;
  primarySelection?: PluginRecordRef;
  groupSelection?: PluginGroupRef;
  host: PluginHostApi;
  settings: unknown;
};

type PluginEntryScope =
  | { kind: "record"; record: PluginRecordRef }
  | { kind: "group"; table: string; field: string; value: unknown; label?: string }
  | { kind: "table" };

type PluginTableData = {
  table: string;
  schema: unknown;
  records: PluginRecord[];
  writable: boolean;
};

type PluginRecord = {
  table: string;
  key: unknown;
  name?: string;
  data: Record<string, unknown>;
  sourceGenerationId: string;
  isReadOnly: boolean;
};

type PluginHostApi = {
  setDirty(dirty: boolean): void;
  proposeChanges(changes: PluginChangeSet): Promise<PluginValidationResult>;
  requestSave(options?: { force?: boolean }): Promise<PluginSaveResult>;
  reload(): Promise<void>;
  notify(message: PluginNotice): void;
};

type PluginInstance = {
  dispose?: () => void;
  beforeSave?: () => PluginChangeSet | Promise<PluginChangeSet>;
  onHostDataReloaded?: (context: PluginContext) => void;
};
```

The exact TypeScript names are descriptive. Implementations may ship plain JavaScript, but the runtime behavior must preserve this contract.

## Change Set Contract

- A change set is grouped by table.
- A writable table change can contain inserts, updates, deletes, and moves.
- Inserts must include complete canonical field values required by that table schema.
- Updates must include the previous record key and the new canonical record value.
- Deletes must include the current record key.
- Moves are allowed only when the target table preserves YAML record order.
- Changes must use canonical primary-key values and schema field names, not plugin-local display labels.
- A plugin may call `proposeChanges` many times while users interact; this validates and records pending changes but does not write YAML.
- The runtime calls `beforeSave` immediately before save if the plugin exposes it, then validates the returned change set.
- `requestSave` follows the same validation-error confirmation model as table editing: first save without `force`, then retry with `force: true` only after explicit user confirmation.

## Opening Flow

1. The frontend finds plugins applicable to the selected table.
2. For `open_mode: record`, the app shows the ordinary table `extable`; each row represents one canonical record, and the user selects one row before opening the plugin.
3. For `open_mode: group`, the app shows a derived `extable` grouping grid; each row represents one group key such as an external reference value, and the user selects one grouping row before opening the plugin.
4. For `open_mode: table`, the app skips the selection grid and opens the plugin immediately from the table-level plugin action.
5. The host checks dirty-state for the current editing surface before leaving or replacing it.
6. The host loads schemas and records for every declared target table.
7. The host resolves the entry scope:
   - `record`: selected record plus declared related records.
   - `group`: every record matching the selected group key plus declared related records.
   - `table`: all records in the declared table scope.
8. The plugin entry function initializes its UI from the provided bundle.
9. User interactions call `host.proposeChanges` or local plugin state methods.
10. The shared frontend marks the plugin surface dirty when pending changes exist.
11. Save validates, optionally confirms validation errors, commits through host APIs, reloads data, and updates the plugin context.

## Entry Point Modes

- `record` mode is for one selected canonical record plus related records. The selection grid row maps one-to-one to a stored record.
- `group` mode is for many records that share one field value, usually an external key. The selection grid row maps to a grouping key and does not correspond to a stored record.
- `table` mode is for whole-table or multi-table editors. It opens immediately and should be used only when showing an intermediate selection grid would add no useful context.
- The host must include the selected mode in `PluginContext.entry`.
- A plugin must not infer its entry mode from UI state; it should use the explicit `entry.kind`.
- The shared frontend must keep entry-mode navigation reversible so the user can return to the table grid or grouping grid that opened the plugin.

## Map Editor Example

- The selected `map` table row is the primary record.
- `map_item` is a child target table filtered by `map_item.map_id == map.map_id`.
- The runtime passes one `map` record and matching `map_item` records to the plugin.
- Dragging an item updates `map_item.x` and `map_item.y`.
- Adding a placed item inserts a `map_item` record with the selected map key materialized into `map_id`.
- Deleting a placed item deletes only matching `map_item` records in the active edit generation.
- Editing map dimensions updates fields on the selected `map` record.
- The save operation commits both table groups as one user action when the host can provide atomic multi-table commit; until then, the runtime must fail clearly if one table save succeeds and another would fail.

## Group Editor Example

- The user opens a plugin from a table such as `map_item`.
- The app shows a grouping grid where each row is one distinct `map_id`.
- Selecting one grouping row opens the plugin with every `map_item` record whose `map_id` matches the selected key.
- The plugin can batch edit, reorder, add, or delete records inside that group when the declaration grants write access.
- The plugin must not update records from other groups unless the declaration includes additional writable scopes and the host included those records in the context.

## Table Editor Example

- The user activates a whole-table plugin action from the table toolbar or plugin menu.
- The app does not render an intermediate `extable` selection surface.
- The plugin receives every scoped record for its declared target tables.
- This mode is appropriate for visual graph editors or global layout editors where the first meaningful action requires seeing the whole table.

## Runtime Isolation

- Plugin HTML is loaded in an iframe or equivalent isolated surface.
- The host exposes only the `PluginHostApi`; it must not expose raw filesystem APIs, arbitrary HTTP clients, shell access, or unscoped application internals.
- The plugin must not receive records from undeclared tables.
- The plugin must not modify DOM outside its isolated surface.
- The plugin should not depend on global application CSS.
- The host should apply a content security policy appropriate for local plugin loading.
- Network access for plugin assets is disabled by default unless a later trusted-plugin policy explicitly allows it.
- Plugin code is treated as project-local code. It is suitable for trusted workspaces but not as a marketplace sandbox boundary.

## Generation Behavior

- Plugin editing uses the same active edit generation as table editing.
- In `active_only` mode, writable records come only from the active generation.
- In `include_previous` mode, previous-generation records may be shown as readonly context when included in the scoped bundle.
- A plugin must not update or delete records whose `isReadOnly` is true.
- Inserts are created in the active edit generation.
- Commit operations must include only active-generation writes.
- Reference lookup and validation follow the same generation-aware rules as ordinary table editing.
- In `group` mode, grouping rows are derived separately for the active display mode; previous-generation rows may appear only as readonly context unless they belong to the active generation.
- In `table` mode, the scoped bundle may be large; the host should still apply the same generation bounds and readonly provenance metadata to every record.

## Validation And Diagnostics

- The plugin may show validation feedback in its own UI, but host diagnostics remain authoritative.
- The host validates canonical proposed records through the schema validation engine.
- Diagnostics returned from `proposeChanges` and `requestSave` include table, record key, field, severity, and message when available.
- The runtime should provide a generic diagnostics area outside the plugin iframe so users can recover when plugin UI fails to render a validation message.
- Invalid plugin declarations block plugin opening and should surface as configuration diagnostics, not runtime JavaScript errors.

## Failure Handling

- If plugin HTML fails to load, the user remains in the ordinary table editing workspace.
- If the entry function is missing or throws during initialization, the host shows a plugin-load diagnostic and keeps canonical data unchanged.
- If `dispose` throws during navigation away, the host logs the error and continues dirty-state handling from host-owned pending changes.
- If a plugin proposes out-of-scope writes, the host rejects the change set and marks it as a plugin contract violation.
- If save partially fails because atomic multi-table commit is unavailable, the host must reload affected tables and show a recovery diagnostic with the tables whose writes completed.

## Dependencies

- [Editor plugin model](../data-model/editor-plugin-model.md)
- [Shared web editing frontend](shared-web-editing-frontend.md)
- [Table editing workspace](../ui-screen/table-editing-workspace.md)
- [Schema validation engine](schema-validation-engine.md)
- [Web service host](../server-component/web-service-host.md)

## Reads

- Plugin declarations.
- Plugin HTML and static assets.
- Scoped schemas and records for declared target tables.
- Active generation and generation display mode from the shared frontend state.

## Writes

- Pending frontend change sets.
- Canonical table records through host commit APIs.
- Optional plugin settings through a future settings API.

## Related Requirements

- [Generic master data model](../data-model/generic-master-data-model.md)
- [Table schema model](../data-model/table-schema-model.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)

## Native-Language Summary

HTML と JavaScript で作った専用 UI を、通常の表編集と同じ保存・検証・世代ルールに接続するランタイム。マップマスター 1 行を選んで、同じ map_id を持つマップアイテム複数行を同時に読み込み、ドラッグや追加削除を canonical なテーブル変更として保存する。プラグインは YAML を直接書かず、ホストがスコープ確認、検証、保存、再読み込みを担当する。
