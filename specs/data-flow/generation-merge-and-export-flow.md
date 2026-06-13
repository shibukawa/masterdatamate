---
id: "generation-merge-and-export-flow"
type: "data-flow"
title: "Generation merge and export flow"
aliases: []
tags: ["generation", "export"]
facts:
  lifecycle.status: "blueprint"
---

# Generation merge and export flow

## Summary

Export starts from a set of selected generations. Generations are ordered by the configured generation index, and selected generation data is merged table by table so that records in newer generations override records with the same primary key from older generations. Export adapters then convert the merged result into target formats.

This flow is read-only with respect to canonical generation data. Persistently creating a new generation from selected generations is handled by [Generation persistent merge flow](generation-persistent-merge-flow.md).
Artifact creation and downloadable export behavior are specified by [Export execution flow](export-execution-flow.md).

## Actors

- Developer.
- Planner or other non-engineering editor.
- Export command, service, or packaged app runtime.

## Components

- [Generic master data model](../data-model/generic-master-data-model.md)
- [Generation model](../data-model/generation-model.md)
- [Schema validation engine](../component/schema-validation-engine.md)
- [Export backend adapters](../component/export-backend-adapters.md)

## APIs

- Future web-service API: `POST /api/generations/merge`.
- Export check and artifact creation use `POST /api/exports/check` and `POST /api/exports`, specified by [Export execution flow](export-execution-flow.md).
- Persistent generation creation uses `POST /api/generations/persistent-merge` and is specified separately.
- The first merge API is dataset-wide only. Table-specific merge preview APIs may be added later if the UI needs a smaller response.
- Table editing may use a generation-aware table view API before export merge is fully implemented.
- The merge request body supplies selected `generationIds`; the server does not read the browser-local active selection.
- The API returns normalized table records grouped by table, plus diagnostics for the effective merged dataset.
- The API returns response-only provenance under each winning record's `comment` object.

### Generation-Aware Table View API

Request:

```http
GET /api/tables/product/generation-view?activeGenerationId=0010_balance_patch&mode=include_previous
```

The server derives the visible generation set from generation metadata. In `active_only` mode, `orderedGenerationIds` contains only `activeGenerationId`. In `include_previous` mode, `orderedGenerationIds` contains output-enabled generations older than or equal to `activeGenerationId`, plus `activeGenerationId` even if its `output` flag is false.

Reference lookup uses a stricter export-facing generation set than the editing table view. Records from generations whose `output` flag is false are not returned as external reference candidates, even when the active generation is visible for editing. This prevents users from creating references to data that is excluded from export.

Each returned row is an editing row, not a raw canonical record. The server adds generation metadata fields to every row so the frontend can enforce readonly behavior and explain provenance without inferring it from the primary key. These metadata fields are not injected as default visible `Generation` or `Status` columns in the ordinary table grid.

Response:

```json
{
  "table": "product",
  "activeGenerationId": "0010_balance_patch",
  "mode": "include_previous",
  "orderedGenerationIds": ["0000_initial", "0010_balance_patch"],
  "rows": [
    {
      "key": "sword_001",
      "name": "Iron Sword",
      "data": {
        "price": 100
      },
      "sourceGenerationId": "0000_initial",
      "sourceGenerationLabel": "0000_initial",
      "isActiveGeneration": false,
      "isReadOnly": true,
      "isOverridden": true,
      "overriddenByGenerationId": "0010_balance_patch"
    },
    {
      "key": "sword_001",
      "name": "Iron Sword",
      "data": {
        "price": 120
      },
      "sourceGenerationId": "0010_balance_patch",
      "sourceGenerationLabel": "0010_balance_patch",
      "isActiveGeneration": true,
      "isReadOnly": false,
      "isOverridden": false
    }
  ],
  "diagnostics": []
}
```

### Merge API

Request:

```json
{
  "generationIds": ["0000_initial", "0010_balance_patch"],
  "includeDiagnostics": true
}
```

Response:

```json
{
  "generationIds": ["0000_initial", "0010_balance_patch"],
  "orderedGenerationIds": ["0000_initial", "0010_balance_patch"],
  "tables": {
    "product": [
      {
        "key": "sword_001",
        "name": "Iron Sword",
        "data": {
          "price": 120
        },
        "comment": {
          "sourceGenerationId": "0010_balance_patch",
          "overrodeGenerationIds": ["0000_initial"]
        }
      }
    ]
  },
  "diagnostics": []
}
```

## Data Stores

- Git-friendly YAML table files.
- Table schema files.
- Generation metadata configuration.
- Canonical storage under `masterdata/schema` and `masterdata/generations`.
- Generated export artifacts such as CSV, YAML, SQLite files, or SQL DML files.

## Rules / Constraints

- Every record belongs to exactly one generation.
- Generations have a sortable `generation_index`.
- Numeric generation indexes sort ascending from old to new, with larger numbers treated as newer.
- Release-date generation indexes sort ascending from old to new, with later dates treated as newer.
- Users can toggle which generations participate in export.
- Each generation has an `output` boolean that provides the default export inclusion state.
- The first runnable slice supports exactly one generation and does not require generation toggles.
- Data editing has two display modes: active generation only, or active generation plus previous output-enabled generations.
- In active-only mode, export preview and commit scope contain only the active edit generation.
- In include-previous mode, export preview uses output-enabled generations older than or equal to the active edit generation, plus the active edit generation even if its own `output` flag is false.
- Generation-aware table view APIs never include generations newer than the active edit generation.
- Generation-aware table view APIs add row-level generation metadata to the data returned for editing.
- `sourceGenerationId` is the authoritative generation provenance for a returned editing row.
- `isReadOnly` is true for every row whose `sourceGenerationId` is not the active edit generation.
- `isReadOnly` is false only for rows from the active edit generation.
- Previous-generation records shown in the editor are readonly context and cannot be committed through the active generation editor.
- External reference candidates in include-previous mode are resolved from every output-enabled generation participating in the export-effective set, not just the active edit generation.
- External reference candidates exclude every generation whose `output` flag is false.
- If the active edit generation has `output: false`, it can still be visible and editable in the table view, but its records are not selectable as external reference candidates.
- Reference candidate responses should expose source generation metadata for explanation, including `sourceGenerationId`, optional `sourceGenerationLabel`, and optional `overrodeGenerationIds`.
- Merge API callers pass the selected generation IDs explicitly in the request body.
- The server sorts requested generations by the configured ordering mode before merge precedence is evaluated.
- Merge is evaluated per table.
- The first merge API returns the full dataset, not a single table.
- Generation merge operates on normalized logical table records, regardless of whether source YAML used table-per-file storage or embedded dependent table storage.
- A missing table YAML file in a selected generation means that generation contributes no records for that table.
- Merge runs across all registered schemas under `masterdata/schema`.
- If two selected generations contain the same table primary key, the record from the newer generation wins.
- Generation-aware table views may show both the older losing row and newer winning row so users can understand overrides before export.
- Older losing rows are marked `isOverridden: true` and identify the winning generation when known.
- A winning merged record includes `comment.sourceGenerationId` identifying the generation that supplied it.
- A winning merged record includes `comment.overrodeGenerationIds` when older selected generations had the same primary key.
- Merge provenance comments are response metadata and are not written back to canonical YAML.
- Primary key uniqueness is checked in the effective merged result and should also be reported within a single generation when duplicated.
- External references are validated before export as stored target primary key values.
- Export behavior can vary by backend, but all backends consume the same validated merged data model.
- Merge API calls do not modify canonical YAML files.
- Merge API calls must not create, update, delete, or rename generation folders.
- Users who need a new canonical generation produced from selected sources use the persistent merge flow instead of the export merge API.

## Error Cases

- `400`: `generationIds` is empty, missing, or malformed.
- `404`: a requested generation does not exist.
- `422`: a generation `_config.yaml` is missing or invalid, generation ordering is invalid, or merged primary keys cannot be normalized.
- `500`: filesystem or YAML read failures that are not attributable to invalid project data.

## Related Requirements

- [Product overview](../generic/product-overview.md)
- [Generic master data model](../data-model/generic-master-data-model.md)
- [Generation model](../data-model/generation-model.md)
- [Canonical YAML file layout](../data-model/canonical-yaml-file-layout.md)
- [Schema validation engine](../component/schema-validation-engine.md)
- [Export backend adapters](../component/export-backend-adapters.md)
- [Generation persistent merge flow](generation-persistent-merge-flow.md)
- [Export execution flow](export-execution-flow.md)

## Native-Language Summary

選択された世代を `generation_index` の昇順で統合し、同じ主キーでは新しい世代のレコードを優先する。これは preview/export 用の非破壊マージであり、世代フォルダやYAMLは書き換えない。新しい世代フォルダを作る永続マージは別仕様で扱う。データ編集ではアクティブ世代のみ、または過去の output 有効世代を含めた表示を選べる。過去世代の行は readonly とし、同一主キーで上書きされた行は上書き状態と勝者世代を表示する。
