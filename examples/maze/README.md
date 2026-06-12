# Maze Master Data Example

This example is a maze game master data workspace with two non-table editor plugin samples.

It defines:

- `enemy`: enemy character parameters and one preview image file.
- `maze_map`: map-level metadata for a 10x10 maze.
- `maze_cell`: one record per grid cell, with terrain and optional enemy placement.
- `masterdata/editor_plugins.yaml`: declarations for two non-table editors.

The plugin HTML programs are built from the React and Vue sources under `masterdata/plugins/`.

## Plugin Samples

The plugin source and built output both live under `masterdata/plugins`:

- `masterdata/plugins/enemy-status-editor-react`: React source for `enemy-status-editor`.
- `masterdata/plugins/enemy-status-editor`: built runtime assets for `enemy-status-editor`.
- `masterdata/plugins/maze-grid-editor-vue`: Vue source for `maze-grid-editor`.
- `masterdata/plugins/maze-grid-editor`: built runtime assets for `maze-grid-editor`.

Build the sample editors from source:

```bash
cd examples/maze/masterdata/plugins
npm install
npm run build
```

The build writes the plugin HTML files referenced by `masterdata/editor_plugins.yaml`:

- `masterdata/plugins/enemy-status-editor/index.html`
- `masterdata/plugins/maze-grid-editor/index.html`

Run MasterDataMate from this directory so `examples/maze` is resolved as the workspace root:

```bash
cd examples/maze
../../dist-native/masterdatamate
```
