---
id: "generation-duplication-flow"
type: "data-flow"
title: "Generation duplication flow"
aliases: []
tags: ["generation", "duplicate", "editing"]
facts:
  lifecycle.status: "blueprint"
---

# Generation duplication flow

## Summary

The generation duplication flow lets users copy one or more selected generations into new generations without opening an input or confirmation dialog. It is useful when users want to branch from existing generations while preserving the selected sources' ordering relationship. Duplicate writes new generation folders immediately using automatic index and path-name derivation.

## Actors

- Developer.
- Planner or other non-engineering editor.
- Web service host.

## Components

- [Generation editing screen](../ui-screen/generation-editing-screen.md)
- [Generation model](../data-model/generation-model.md)
- [Canonical YAML file layout](../data-model/canonical-yaml-file-layout.md)
- [Web service host](../server-component/web-service-host.md)

## APIs

- `POST /api/generations/duplicate`: create one or more new generations by copying selected source generation folders.
- The request body supplies `sourceGenerationIds`.
- For backward compatibility, the request may supply one `sourceGenerationId`; if `destination` is also supplied, the server preserves the older explicit single-duplicate behavior.
- The response returns created generation IDs, copied file paths, updated generation metadata, and diagnostics.

Request:

```json
{
  "sourceGenerationIds": ["0010_balance_patch", "0030_event_patch"]
}
```

## Rules / Constraints

- Duplication requires at least one selected persisted generation.
- Duplicate is disabled while generation metadata edits are dirty because source row identity and destination collision checks may be stale.
- Activating `Duplicate` immediately calls the duplicate API without an input dialog and without a confirmation dialog.
- Selected source generations are sorted by configured generation ordering before duplicate indexes are assigned.
- In numeric ordering mode, destination `generation_index` values start at the current maximum generation index plus 10 and continue by +10 for each selected source.
- In release-date ordering mode, destination `generation_index` values start after the latest existing date and advance by one day for each selected source.
- Destination `path_name` is `<source path_name>_copy`.
- If that `path_name` collides, the server appends a numeric suffix such as `_copy2`, `_copy3`, and so on.
- Destination `output` is copied from the source generation.
- Destination `description` is copied from the source generation.
- Destination `generation_index`, `path_name`, and derived `folder_name` must not collide with existing or newly planned generations.
- The server must fail rather than overwrite an existing destination generation folder.
- Source generation folders and files must not be modified.
- On success, the frontend reloads generation metadata and clears all generation row selection so the user cannot accidentally repeat the same duplicate action.

## Error Cases

- `400`: source generation selection is missing, malformed, or duplicated.
- `404`: a source generation does not exist.
- `409`: destination generation index, path name, derived folder name, or folder path collides with an existing or planned generation.
- `422`: source generation configuration or derived destination metadata is invalid.
- `500`: filesystem or YAML read/write failures that are not attributable to invalid project data.

## Related Requirements

- [Generation editing screen](../ui-screen/generation-editing-screen.md)
- [Generation model](../data-model/generation-model.md)
- [Canonical YAML file layout](../data-model/canonical-yaml-file-layout.md)
- [Web service host](../server-component/web-service-host.md)

## Native-Language Summary

世代編集画面で1つ以上の世代を選択し、`Duplicate` を押すと入力・確認ダイアログなしで即座にコピーする。コピー先は現在の最新 index から `+10` ずつ採番し、`path_name` には `_copy` を付ける。複数選択時は選択元世代の順序関係を維持してコピー先 index を割り当てる。元世代は変更しない。
