---
id: "generation-persistent-merge-flow"
type: "data-flow"
title: "Generation persistent merge flow"
aliases: []
tags: ["generation", "merge", "editing"]
facts:
  lifecycle.status: "blueprint"
---

# Generation persistent merge flow

## Summary

The generation persistent merge flow lets users select multiple generation rows in the generation editing screen and merge them into a new generation. Unlike the export merge API, this flow writes canonical generation metadata and table YAML files under `masterdata/generations`. Source generations remain unchanged. The merge precedence follows normal generation ordering: records from the selected source generation with the larger generation index win over records with the same table primary key from smaller indexes.

## Actors

- Developer.
- Planner or other non-engineering editor.
- Web service host.

## Components

- [Generation editing screen](../ui-screen/generation-editing-screen.md)
- [Generation model](../data-model/generation-model.md)
- [Canonical YAML file layout](../data-model/canonical-yaml-file-layout.md)
- [Schema validation engine](../component/schema-validation-engine.md)
- [Web service host](../server-component/web-service-host.md)
- [Generation merge and export flow](generation-merge-and-export-flow.md)

## User Flow

1. The user opens the generation editing screen.
2. The user selects two or more generation rows with the row selection controls in the generation metadata grid.
3. The `Merge` command becomes enabled only while at least two valid generation rows are selected and there are no pending generation metadata edits.
4. Activating `Merge` opens a merge dialog.
5. The dialog shows the selected source generations sorted by `generation_index` ascending so the user can confirm old-to-new precedence.
6. The user enters the destination generation metadata: `generation_index`, `path_name`, `output`, and optional `description`.
7. The dialog previews the derived destination `folder_name` and warns when the destination index is greater than, equal to, or smaller than the selected source indexes.
8. The user confirms the dialog with an explicit action that creates the destination generation.
9. The server validates source generations, destination metadata, schema availability, and merge results.
10. The server creates the destination generation folder, writes `_config.yaml`, and writes merged table YAML files.
11. The generation editing screen reloads generation metadata, clears all generation row selection, and shows diagnostics if any non-blocking warnings were returned.

## APIs

- `POST /api/generations/persistent-merge`: create a new generation by persistently merging selected source generations.
- This API is distinct from `POST /api/generations/merge`, which returns an effective merged dataset for preview/export and does not write YAML files.
- The request body supplies selected `sourceGenerationIds`; the server does not read browser-local selection state.
- The request body supplies destination metadata instead of deriving it from the selected source generations.
- The server sorts source generations by the configured ordering mode before merge precedence is evaluated.
- The server writes canonical YAML only after the full request validates successfully.
- The response returns created generation metadata, written table file paths, row counts by table, and diagnostics.
- The frontend must show the merge confirmation dialog before calling this API.
- The API still validates the request independently; UI confirmation is not a substitute for server validation.

Request:

```json
{
  "sourceGenerationIds": ["0000_initial", "0010_balance_patch"],
  "destination": {
    "generation_index": 20,
    "path_name": "merged_balance",
    "output": true,
    "description": "Merged baseline and balance patch"
  },
  "includeDiagnostics": true
}
```

Response:

```json
{
  "generationId": "0020_merged_balance",
  "folderName": "0020_merged_balance",
  "generation": {
    "generation_index": 20,
    "path_name": "merged_balance",
    "output": true,
    "description": "Merged baseline and balance patch"
  },
  "sourceGenerationIds": ["0000_initial", "0010_balance_patch"],
  "orderedSourceGenerationIds": ["0000_initial", "0010_balance_patch"],
  "tables": {
    "product": {
      "file": "masterdata/generations/0020_merged_balance/product.yaml",
      "recordCount": 128,
      "overriddenRecordCount": 12
    }
  },
  "writtenFiles": [
    "masterdata/generations/0020_merged_balance/_config.yaml",
    "masterdata/generations/0020_merged_balance/product.yaml"
  ],
  "diagnostics": []
}
```

## Data Stores

- Generation metadata configuration and `_config.yaml` files.
- Table schema files under `masterdata/schema`.
- Source generation table YAML files under `masterdata/generations/<source_generation>`.
- Destination generation table YAML files under `masterdata/generations/<destination_generation>`.

## Merge Semantics

- Persistent merge operates across all registered schemas under `masterdata/schema`.
- The source generation set must contain at least two generations.
- The source generation set must not contain duplicates.
- Source generation request order does not define precedence.
- The server sorts source generations by `generation_index` according to the configured ordering mode.
- In numeric ordering mode, larger numeric `generation_index` values win over smaller values.
- In release-date ordering mode, later date `generation_index` values win over earlier values.
- Merge is evaluated independently per normalized logical table.
- A missing table YAML file in a source generation means that generation contributes no records for that table.
- If multiple selected source generations contain the same table primary key, the record from the highest-precedence source generation is written to the destination generation.
- Records that win the merge are written as ordinary canonical records and must not include response-only merge provenance comments.
- The destination generation should contain the full effective merged dataset for each registered table, not only rows changed relative to source generations.
- Record order in each destination table file should be deterministic: old-to-new source generation order, preserving each source file's record order, while replacing older duplicate keys at the older key's effective position when a newer record wins.
- Embedded dependent table records are normalized for merge evaluation and written back using the project's canonical storage policy for the destination generation.
- The persistent merge must not modify, delete, or rename source generation folders or files.
- The persistent merge must not change global generation ordering settings.

## Destination Generation Rules

- Destination `generation_index` is required and must be valid for the configured ordering mode.
- Destination `path_name` is required and must pass the same validation as manual generation creation.
- Destination `folder_name` is derived from destination metadata and global generation settings.
- Destination `generation_index`, `path_name`, and derived `folder_name` must not collide with any existing generation.
- Destination `generation_index` may be larger than all selected source indexes, equal to no selected source index, or between existing source indexes as long as it is globally unique.
- The destination generation is not included in merge precedence for the request that creates it.
- Destination `output` defaults to true in the merge dialog unless the user changes it.
- Destination `description` should default to a concise explanation listing the selected source generation display labels when the user has not typed a value.
- The confirmation dialog must show the destination folder path and selected source generation labels before submit.

## Atomicity And Failure Handling

- The server should validate the full operation before writing any destination files.
- If a destination generation folder already exists, the request fails before writing.
- If filesystem writes fail after partial destination creation, the server should remove only files and folders created by the failed request when it can do so safely.
- If safe cleanup is not possible, the response must identify partial paths and return a blocking diagnostic.
- The API is not required to provide cross-process locking in the first implementation, but it must avoid overwriting an existing destination folder.

## Error Cases

- `400`: `sourceGenerationIds` is missing, malformed, contains fewer than two IDs, contains duplicates, or destination metadata is missing.
- `404`: a requested source generation does not exist.
- `409`: the destination generation index, path name, derived folder name, or folder path collides with an existing generation.
- `422`: a source generation `_config.yaml` is missing or invalid, destination metadata is invalid, generation ordering is invalid, schema loading fails due to project data, or merged primary keys cannot be normalized.
- `500`: filesystem or YAML read/write failures that are not attributable to invalid project data.

## Related Requirements

- [Generation editing screen](../ui-screen/generation-editing-screen.md)
- [Generation model](../data-model/generation-model.md)
- [Canonical YAML file layout](../data-model/canonical-yaml-file-layout.md)
- [Web service host](../server-component/web-service-host.md)
- [Generation merge and export flow](generation-merge-and-export-flow.md)

## Native-Language Summary

世代編集画面で複数の世代行を選択し、`Merge` ダイアログで新しい世代の `generation_index`、`path_name`、`output`、説明を入力して、選択元世代を新しい世代フォルダへ永続マージする。通常の世代マージと同じく `generation_index` が大きい世代のレコードを優先し、元世代は変更しない。`POST /api/generations/persistent-merge` は `_config.yaml` とテーブルYAMLを作成する書き込みAPIであり、非破壊の preview/export 用 `POST /api/generations/merge` とは分ける。
