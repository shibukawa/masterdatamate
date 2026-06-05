---
id: "generation-deletion-flow"
type: "data-flow"
title: "Generation deletion flow"
aliases: []
tags: ["generation", "delete", "editing"]
facts:
  lifecycle.status: "blueprint"
---

# Generation deletion flow

## Summary

The generation deletion flow lets users delete one or more selected generations from the generation editing screen. Deletion is destructive because it removes generation metadata and generation data folders under `masterdata/generations`, so the UI must always require an explicit confirmation dialog before the API is called.

## Actors

- Developer.
- Planner or other non-engineering editor.
- Web service host.

## Components

- [Generation editing screen](../ui-screen/generation-editing-screen.md)
- [Generation model](../data-model/generation-model.md)
- [Canonical YAML file layout](../data-model/canonical-yaml-file-layout.md)
- [Web service host](../server-component/web-service-host.md)

## User Flow

1. The user opens the generation editing screen.
2. The user selects one or more generation rows.
3. The `Delete` command becomes enabled only when at least one persisted generation row is selected and there are no pending generation metadata edits.
4. Activating `Delete` opens a confirmation dialog.
5. The dialog lists every selected generation by display label, raw folder name, generation index, output flag, and data path.
6. The dialog warns when the selected set contains the current active edit generation, output-enabled generations, or all available generations.
7. The user confirms deletion explicitly.
8. The server validates that the selected generations can be deleted.
9. The server deletes the selected generation folders and metadata.
10. The generation editing screen reloads generation metadata, clears deleted row selection, and resolves the active edit generation if it was deleted.

## APIs

- `POST /api/generations/delete`: delete one or more selected generations.
- The request body supplies `generationIds`; the server does not read browser-local selection state.
- The API may use `POST` rather than `DELETE` because the operation accepts a request body, performs validation, and deletes multiple resources.
- The server must not accept wildcard deletion or implicit "delete all" requests.
- The response returns deleted generation IDs, deleted paths, the remaining generation list summary, the resolved active generation when applicable, and diagnostics.

Request:

```json
{
  "generationIds": ["0010_balance_patch", "0020_experiment"],
  "activeGenerationId": "0010_balance_patch"
}
```

Response:

```json
{
  "deletedGenerationIds": ["0010_balance_patch", "0020_experiment"],
  "deletedPaths": [
    "masterdata/generations/0010_balance_patch",
    "masterdata/generations/0020_experiment"
  ],
  "remainingGenerationIds": ["0000_initial"],
  "resolvedActiveGenerationId": "0000_initial",
  "diagnostics": []
}
```

## Rules / Constraints

- Deletion requires at least one selected generation.
- Deletion must always be initiated from a confirmation dialog.
- Delete is disabled while generation metadata edits are dirty because row identity, derived folder names, and active-generation fallback may be stale.
- Unsaved newly inserted generation rows are removed by local revert or grid editing behavior, not by the delete API.
- The server must reject deletion of unknown generation IDs.
- The server must reject duplicate generation IDs in the delete request.
- The server must reject deleting all generations unless a later explicit project reset feature is designed.
- The first implementation should preserve at least one valid generation after deletion.
- If the active edit generation is deleted, the server or frontend must resolve a replacement deterministically, preferring the nearest older generation by ordering and otherwise the earliest remaining generation.
- Deleting generations must not modify table schema files.
- Deleting generations must not modify non-selected generation folders.
- Deleting output-enabled generations is allowed only after the confirmation dialog explicitly marks them as output-enabled.
- The server should remove only folders that correspond to selected valid generations under `masterdata/generations`.
- The server must not follow symlinks or delete paths outside `masterdata/generations`.
- A failed deletion must report which generations, if any, were deleted before the failure.

## Error Cases

- `400`: `generationIds` is missing, malformed, empty, contains duplicates, or attempts an implicit all-generation deletion.
- `404`: a requested generation does not exist.
- `409`: deletion would leave no valid generation, or the server detects a stale active generation resolution conflict.
- `422`: a generation `_config.yaml` is invalid enough that safe folder resolution cannot be proven.
- `500`: filesystem delete failures that are not attributable to invalid project data.

## Related Requirements

- [Generation editing screen](../ui-screen/generation-editing-screen.md)
- [Generation model](../data-model/generation-model.md)
- [Canonical YAML file layout](../data-model/canonical-yaml-file-layout.md)
- [Web service host](../server-component/web-service-host.md)

## Native-Language Summary

世代編集画面で1つ以上の世代行を選択し、必ず確認ダイアログを挟んでから世代フォルダを削除する。削除APIは `POST /api/generations/delete` とし、選択IDを明示的に受け取る。全世代削除は初期仕様では禁止し、アクティブ世代が削除された場合は残っている世代から決定的に代替世代を選ぶ。
