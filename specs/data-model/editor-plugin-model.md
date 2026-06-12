---
id: "editor-plugin-model"
type: "data-model"
title: "Editor plugin model"
aliases: ["custom-editor-plugin", "html-editor-plugin"]
tags: ["plugin", "editor", "extension"]
facts:
  lifecycle.status: "blueprint"
  data.name: "editor-plugin"
---

# Editor plugin model

## Summary

An editor plugin declares a custom editing surface for master data that is awkward to enter through a spreadsheet-like grid. Examples include map editors, graph editors, timeline editors, dialogue tree editors, and other domain-specific visual tools.

The plugin never owns canonical storage. It reads and writes the same schema-defined records used by the ordinary [Table editing workspace](../ui-screen/table-editing-workspace.md), then commits normalized table operations through the host. The ordinary table editor remains the fallback and review surface for every record the plugin edits.

## Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| plugin_id | string | yes | Stable ASCII identifier for the plugin. |
| display_name | string | yes | Human-facing plugin name shown in editor selection UI. |
| description | string | no | Short explanation for schema authors and maintainers. |
| entry_html | path | yes | Built HTML asset loaded as the plugin editor entry point. This path is relative to the `masterdata` directory and is the runtime artifact path, not necessarily the plugin source file. |
| source_dir | path | no | `masterdata`-relative source directory for a buildable plugin project, such as a React, Vue, or vanilla Vite app. The host does not serve files from this directory unless they are also part of the declared built output tree. |
| build | object | no | Optional build metadata for source-managed plugins. |
| build.command | string | no | Command maintainers run from `build.working_dir` to generate runtime assets, such as `npm run build`. |
| build.working_dir | path | no | `masterdata`-relative directory where the build command runs. Defaults to `source_dir` when omitted. |
| build.output_dir | path | no | `masterdata`-relative directory containing the built static assets. It must match or contain the resolved runtime asset directory for `entry_html`. |
| version | string | yes | Plugin contract version, not application release version. |
| entry_points | array | no | UI launch points where this plugin appears. If omitted, the host derives one backward-compatible launch point from `open_mode` and the primary target table. |
| entry_points.entry_id | string | yes | Stable identifier unique within the plugin declaration. |
| entry_points.label | string | no | Optional label for this launch point. Defaults to `display_name`. |
| entry_points.placement | enum | yes | `sidebar`, `table_toolbar`, `record_action`, or `group_action`. |
| entry_points.table | string | no | Table whose navigation or grid surface owns this launch point. Required for table, record, and group placements. |
| entry_points.open_mode | enum | no | Optional override of the plugin's default `open_mode` for this launch point. |
| entry_points.group_by | object | no | Optional grouping override for a `group_action` launch point. Same shape as top-level `group_by`. |
| entry_points.default | boolean | no | Whether this is the preferred launch point when multiple placements are valid in the same UI context. |
| target_tables | array | yes | Tables the plugin reads or writes. At least one target table is required. |
| target_tables.role | enum | yes | `primary`, `child`, `lookup`, or `readonly_context`. |
| target_tables.table | string | yes | Referenced table `system_name`. |
| target_tables.required | boolean | yes | Whether the plugin can open when this table is missing or invalid. |
| target_tables.write | boolean | yes | Whether plugin code may propose changes for this table. |
| target_tables.filter | object | no | Binding from one table's field values to another table's records. |
| target_tables.filter.source_table | string | no | Table that supplies the filter value. |
| target_tables.filter.source_field | string | no | Field on the source record. |
| target_tables.filter.target_field | string | no | Field on the target table records. |
| target_tables.filter.mode | enum | no | `equals` for the first supported slice. |
| open_mode | enum | yes | `record`, `group`, or `table`. Determines whether the plugin opens from one selected record, one selected grouping row, or the whole table without an intermediate grid. |
| group_by | object | no | Required when `open_mode` is `group`. Declares how records are grouped into extable selection rows. |
| group_by.table | string | no | Table whose records are grouped. |
| group_by.field | string | no | Field used as the grouping key, often an external reference field. |
| group_by.label_field | string | no | Optional field or referenced display label used for the grouping row label. |
| capabilities | array | no | Optional declared UI capabilities such as `visual_layout`, `batch_child_edit`, or `coordinate_grid`. |
| permissions | object | yes | Host-granted operations requested by the plugin. |
| permissions.write_tables | array | no | Table names the plugin may write. Must be a subset of writable target tables. |
| permissions.read_tables | array | no | Table names the plugin may read. Must be a subset of target tables. |
| permissions.write_binaries | array | no | Table names whose scoped record binary assets the plugin may upload, replace, or delete. |
| permissions.read_binaries | array | no | Table names whose scoped record binary assets the plugin may preview or download. |
| settings_schema | object | no | Plugin-specific persisted settings schema. Settings must not replace master data. |

## UI Entry Points

`open_mode` describes the data scope the plugin receives. `entry_points` describes where users can launch that scope from the UI. Keeping these separate lets one plugin appear as a left-sidebar editor, a table toolbar action, or a per-record action without changing the runtime data contract.

| Placement | Where it appears | Allowed mode | Behavior |
| --- | --- | --- | --- |
| `sidebar` | Left navigation alongside table entries. | `table` | Opens the plugin directly as a top-level workspace item. Use for custom editors that are the primary way to work with a domain area, such as a maze editor. |
| `table_toolbar` | Toolbar for a selected table. | `table` or `group` | Opens a whole-table plugin immediately, or opens/focuses the derived grouping chooser for grouped plugins. |
| `record_action` | Row action, context action, or selected-row action in an ordinary table grid. | `record` | Enabled when exactly one record row is selected or invoked. |
| `group_action` | Table toolbar action that opens a grouping chooser, or an action on a grouping row. | `group` | Lets the user choose one grouping key, then opens all records in that group. |

If `entry_points` is omitted, the host derives one entry point for compatibility:

- `open_mode: record` becomes a `record_action` on the primary target table.
- `open_mode: group` becomes a `group_action` on `group_by.table`.
- `open_mode: table` becomes a `table_toolbar` action on the primary target table.

A `sidebar` entry point should be used when the plugin is intended to feel like a first-class navigation destination. It must not require a preselected row. If it edits records from tables that are hidden from ordinary table navigation, those tables still remain canonical tables and still participate in validation, export, references, and plugin scope checks.

## Entry Point Modes

| Mode | Selection UI | Scoped records | Typical use |
| --- | --- | --- | --- |
| `record` | Show an `extable` table where each row is one record; user selects one row before opening the plugin. | The selected record plus declared related records. | Map master row plus map item child rows. |
| `group` | Show an `extable` table where each row is one grouping key; user selects one grouping row before opening the plugin. | All records in the selected group plus declared related records. | Editing all records that share one external key, region, category, chapter, or parent ID. |
| `table` | Open the plugin immediately after the user activates the table-level plugin action; no intermediate `extable` selection grid is required. | All records in the declared table scope. | Whole-map overview, global graph editor, table-wide layout editor, or bulk visual editor. |

`record` and `group` entry points still use `extable` as the selection surface. In `record`, one selection row maps to one canonical record. In `group`, one selection row maps to a group key and may represent many canonical records.

## Plugin Asset Packaging

`entry_html` identifies the built plugin entry file that the runtime loads. It may be a hand-authored single HTML file, but it may also be generated by a source project such as Vite, webpack, or another frontend build tool. In all cases, `entry_html` points at the distribution artifact served by the host, and any JavaScript or CSS referenced from that HTML must be available through relative paths inside the same built asset tree.

When a plugin is source-managed, `source_dir` and `build` describe how maintainers regenerate the served artifacts. These fields are documentation and tooling inputs; the editor host must not execute build commands automatically while opening a plugin. Plugin source projects should live under `masterdata/plugins` because they are part of the master data workspace rather than the MasterDataMate application source. A project can therefore keep dependencies such as React, Vue, and Vite under the plugin source tree without adding them to the MasterDataMate application package.

The recommended layout separates source and runtime assets:

```text
workspace/
  masterdata/
    editor_plugins.yaml
    plugins/
      map-editor/            # source_dir / build.working_dir
        package.json
        src/
        vite.config.js
        dist/                # build.output_dir
          index.html         # entry_html
          assets/
```

Buildable plugin projects should configure their bundler for relative asset URLs, such as Vite `base: "./"`, so the same output works in iframe loading, local static serving, and packaged hosts.

## Record Entry Example

```yaml
plugin_id: map-editor
display_name: Map editor
description: Visual editor for one map and its placed map items.
entry_html: plugins/map-editor/dist/index.html
source_dir: plugins/map-editor
build:
  working_dir: plugins/map-editor
  command: npm run build
  output_dir: plugins/map-editor/dist
version: "1"
open_mode: record
entry_points:
  - entry_id: map-record
    placement: record_action
    table: map
target_tables:
  - role: primary
    table: map
    required: true
    write: true
  - role: child
    table: map_item
    required: true
    write: true
    filter:
      source_table: map
      source_field: map_id
      target_field: map_id
      mode: equals
permissions:
  read_tables: [map, map_item]
  write_tables: [map, map_item]
capabilities: [visual_layout, batch_child_edit, coordinate_grid]
```

When a user selects one `map` record, the host loads the built plugin entry HTML and supplies the selected map record plus every `map_item` record whose `map_id` matches the selected map's `map_id`. The plugin can update the map record and insert, update, delete, or reorder matching map item records.

## Group Entry Example

```yaml
plugin_id: encounter-placement-editor
display_name: Encounter placement editor
description: Visual editor for every encounter placement that belongs to one map.
entry_html: plugins/encounter-placement-editor/index.html
version: "1"
open_mode: group
group_by:
  table: encounter_placement
  field: map_id
  label_field: map_id
target_tables:
  - role: primary
    table: encounter_placement
    required: true
    write: true
  - role: lookup
    table: encounter
    required: true
    write: false
permissions:
  read_tables: [encounter_placement, encounter]
  write_tables: [encounter_placement]
capabilities: [visual_layout, batch_edit]
```

When a user opens this plugin from `encounter_placement`, the app first shows a grouping grid whose rows are distinct `map_id` values. Selecting one row opens the plugin with every `encounter_placement` record that has that `map_id`.

## Table Entry Example

```yaml
plugin_id: world-graph-editor
display_name: World graph editor
description: Graph editor for every world node and connection.
entry_html: plugins/world-graph-editor/index.html
version: "1"
open_mode: table
entry_points:
  - entry_id: world-graph
    label: World graph
    placement: sidebar
target_tables:
  - role: primary
    table: world_node
    required: true
    write: true
  - role: child
    table: world_edge
    required: true
    write: true
permissions:
  read_tables: [world_node, world_edge]
  write_tables: [world_node, world_edge]
capabilities: [visual_graph, batch_edit]
```

When a user activates this table-level plugin, the host skips the selection grid and immediately loads all scoped `world_node` and `world_edge` records.

## Rules / Constraints

- Plugin declarations are project configuration, not application code generated from schema fields.
- `plugin_id` values must be unique within one workspace.
- `entry_points.entry_id` values must be unique within one plugin declaration.
- `entry_points.placement` must be compatible with the effective `open_mode`.
- A `sidebar` entry point must use `open_mode: table` and must not depend on a selected table row.
- A `record_action` entry point must use `open_mode: record`.
- A `group_action` entry point must use `open_mode: group` and must provide either top-level `group_by` or entry-point `group_by`.
- A `table_toolbar` entry point must identify the table whose toolbar shows the action.
- Entry-point table names must be declared in `target_tables` unless the placement is `sidebar`.
- `entry_html` must resolve to a file inside the declared runtime plugin asset area for the workspace.
- `source_dir`, `build.working_dir`, and `build.output_dir` are resolved relative to `masterdata/` and must stay under `masterdata/plugins/`.
- `build.output_dir` must resolve to the runtime asset tree that contains `entry_html`; source files and dependency folders such as `node_modules` must not be served as plugin runtime assets unless they are intentionally copied into the built output tree.
- Every target table must exist in the loaded table schema set before the plugin can open.
- A plugin can read only declared target tables.
- A plugin can write only target tables listed in both `target_tables[write=true]` and `permissions.write_tables`.
- A plugin can preview binary assets only for declared target tables listed in `permissions.read_binaries`.
- A plugin can upload, replace, or delete binary assets only for declared writable target tables listed in `permissions.write_binaries`.
- `open_mode: record` requires exactly one `primary` target table.
- `open_mode: group` requires `group_by.table` and `group_by.field`.
- `open_mode: group` grouping rows are derived from canonical records and must not create synthetic writable records.
- `open_mode: table` must define a bounded table scope through declared target tables; it does not grant project-wide data access.
- The initial plugin slice supports filter mode `equals` only.
- Filter fields must exist in their table schemas and must use scalar or primary-key-compatible values.
- Group fields must exist in their table schema and must use scalar or primary-key-compatible values.
- The host resolves target table filters before sending data to the plugin; plugin code must not scan arbitrary workspace files.
- The host must reject proposed changes that target records outside the plugin's declared writable scope.
- The host must reject binary asset operations for records outside the plugin's current scoped context.
- The host validates all proposed changes through the normal [Schema validation engine](../component/schema-validation-engine.md).
- The plugin must not persist canonical data directly to YAML files.
- Plugin-specific settings may be persisted separately, but exported game/application data must come from canonical table records.
- A custom plugin editor must not become the only way to inspect or repair data. Every edited record remains editable in the ordinary table workspace.

## Uses Common Details

- None yet.

## Reads

- Table schemas from [Table schema model](table-schema-model.md).
- Canonical records from [Generic master data model](generic-master-data-model.md).

## Writes

- Normalized commit operations for declared writable tables.
- Optional plugin source/build metadata and plugin settings separate from canonical master data.

## Related Requirements

- [HTML editor plugin runtime](../component/html-editor-plugin-runtime.md)
- [Shared web editing frontend](../component/shared-web-editing-frontend.md)
- [Table editing workspace](../ui-screen/table-editing-workspace.md)
- [Web service host](../server-component/web-service-host.md)

## Native-Language Summary

表形式では入力しづらいマップ、配置物、グラフ、会話ツリーなどを、HTML/JS の専用 UI で編集するためのプラグイン定義。`open_mode` はデータスコープ、`entry_points` は左ナビ・表ツールバー・レコードアクションなどの起動場所を表す。実行時にロードする `entry_html` は `masterdata` 相対のビルド済み成果物を指し、React/Vue/Vite などのソースプロジェクトは `masterdata/plugins` 配下の `source_dir` と `build` で任意に記述できる。プラグインは正本 YAML を直接書かず、対象テーブル・読み書き権限・親子レコードの絞り込み条件を宣言し、通常の保存 API と検証エンジンを通して変更を反映する。
