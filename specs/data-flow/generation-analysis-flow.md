---
id: "generation-analysis-flow"
type: "data-flow"
title: "Generation analysis flow"
aliases: []
tags: ["generation", "analysis", "editing"]
facts:
  lifecycle.status: "blueprint"
---

# Generation analysis flow

## Summary

The generation analysis flow lets users inspect selected generations before merge, duplicate, delete, or export-related decisions. The operation is read-only and is opened from an `Analyze` button in the generation editing screen. It reports table counts, record counts, readable data paths, diagnostics, and merge-impact information such as duplicate primary keys that would be overridden by newer selected generations.

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
- [Generation persistent merge flow](generation-persistent-merge-flow.md)
- [Generation deletion flow](generation-deletion-flow.md)

## APIs

- `POST /api/generations/analyze`: return a read-only summary and diagnostics for selected generations.
- The request body supplies selected `generationIds`; the server does not read browser-local selection state.
- The API returns generation summaries, per-table counts, total counts, read/parse diagnostics, and merge-impact diagnostics.
- The API does not write canonical YAML files.

Request:

```json
{
  "generationIds": ["0000_initial", "0010_balance_patch"],
  "includeMergeImpact": true
}
```

Response:

```json
{
  "generationIds": ["0000_initial", "0010_balance_patch"],
  "orderedGenerationIds": ["0000_initial", "0010_balance_patch"],
  "summary": {
    "generationCount": 2,
    "tableCount": 3,
    "recordCount": 156,
    "overriddenRecordCount": 12
  },
  "generations": [
    {
      "generationId": "0000_initial",
      "folderPath": "masterdata/generations/0000_initial",
      "output": true,
      "tableCount": 3,
      "recordCount": 144
    }
  ],
  "tables": {
    "product": {
      "recordCount": 128,
      "generationRecordCounts": {
        "0000_initial": 120,
        "0010_balance_patch": 8
      },
      "overriddenRecordCount": 12
    }
  },
  "diagnostics": []
}
```

## Analysis Checks

- Validate that every requested generation exists.
- Validate that every selected generation has a readable and valid `_config.yaml`.
- Validate that configured generation ordering can sort the selected generations.
- Load registered schemas under `masterdata/schema`.
- Read table YAML files for selected generations when present.
- Treat missing table YAML files as zero records, not as errors.
- Normalize table records enough to count records and compare primary keys.
- Report unreadable or malformed YAML files as diagnostics.
- Report duplicate primary keys within the same generation and table as diagnostics.
- When `includeMergeImpact` is true, report how many selected records would be overridden by newer selected generations under normal merge precedence.
- When `includeMergeImpact` is true, report which generation would win for duplicate primary keys if the UI needs row-level drilldown later.
- Do not run full export validation unless a later option explicitly requests it; `Analyze` is an operational impact summary, not an export approval.

## UI Rules

- `Analyze` is enabled when at least one persisted generation row is selected and there are no pending generation metadata edits.
- `Analyze` is disabled while generation metadata edits are dirty because selected row identity and derived folder paths may be stale.
- Activating `Analyze` opens a read-only dialog or side panel.
- `Analyze` does not require a confirmation dialog because it does not write files.
- The analysis view should show total selected generations, table count, record count, per-table record counts, output-enabled status, and diagnostics.
- The analysis view should make merge impact visible when multiple generations are selected.
- The analysis view may offer shortcuts to `Merge`, `Duplicate`, or `Delete`; `Merge` and `Delete` must still open their own confirmation dialogs, while `Duplicate` runs immediately by design.

## Error Cases

- `400`: `generationIds` is missing, malformed, empty, or contains duplicates.
- `404`: a requested generation does not exist.
- `422`: generation configuration, schema configuration, YAML parsing, or record normalization failed for project data.
- `500`: filesystem read failures that are not attributable to invalid project data.

## Related Requirements

- [Generation editing screen](../ui-screen/generation-editing-screen.md)
- [Generation model](../data-model/generation-model.md)
- [Canonical YAML file layout](../data-model/canonical-yaml-file-layout.md)
- [Web service host](../server-component/web-service-host.md)
- [Generation persistent merge flow](generation-persistent-merge-flow.md)
- [Generation deletion flow](generation-deletion-flow.md)

## Native-Language Summary

世代編集画面の `Analyze` ボタンから、選択世代のテーブル数、レコード数、出力対象状態、読み込み診断、マージ時に上書きされる主キー数を読み取り専用で確認する。これは書き込み操作ではないため確認ダイアログは不要。結果から `Merge` や `Delete` に進む場合は確認ダイアログを挟み、`Duplicate` は仕様上の例外として即時実行する。
