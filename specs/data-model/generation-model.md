---
id: "generation-model"
type: "data-model"
title: "Generation model"
aliases: []
tags: ["generation", "ordering", "export", "folder"]
facts:
  lifecycle.status: "blueprint"
  data.name: "generation"
---

# Generation model

## Summary

A generation is an ordered dataset layer. Generations are edited as table-like metadata, preferably through `extable`, and are sorted by a stable generation index. The index can be numeric with zero-padded folder prefixes or date-based when the project chooses release-date ordering.

## Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| generation_index | number or date | yes | Sortable key that defines old-to-new ordering. Stored in generation `_config.yaml`. |
| output | boolean | yes | Whether this generation participates in export by default. Stored in generation `_config.yaml`. |
| path_name | string | yes | Suffix used in the generation folder name after any generated prefix. Stored in generation `_config.yaml`. |
| description | string | no | Human-facing explanation of the generation. Stored in generation `_config.yaml`. |
| folder_name | string | derived | Physical folder name under `masterdata/generations`. |
| source_generation_ids | string[] | operation | Source generation IDs selected for a persistent merge operation. Not stored in destination `_config.yaml` unless a later provenance field is explicitly added. |
| deleted_generation_ids | string[] | operation | Generation IDs selected for a deletion operation. Not stored. |
| duplicated_source_generation_id | string | operation | Source generation ID selected for a duplication operation. Not stored in destination `_config.yaml` unless a later provenance field is explicitly added. |
| ordering_mode | enum | project | Global setting: `numeric` or `release_date`. |
| numeric_digits | integer | project | Global zero-padding width for numeric folder prefixes, such as `4` for `0010`. |

## Ordering Modes

- `numeric`: `generation_index` is a number. Generations are ordered by numeric ascending order, with larger numbers treated as newer.
- `release_date`: `generation_index` is a date. Generations are ordered by date ascending order, with later dates treated as newer.
- The project chooses one ordering mode globally.
- Ordering mode changes require validating all existing generations.

## Folder Naming

- The first runnable slice uses the fixed generation folder `0000_initial`.
- `0000_initial` represents a single initial generation and does not require generation selection or generation editing UI.
- In numeric mode, `folder_name` is derived from a zero-padded numeric prefix plus `path_name`.
- With `numeric_digits: 4` and `generation_index: 10`, the prefix is `0010`.
- The derived folder name should sort correctly in ordinary filesystem lexicographic order.
- The separator between prefix and `path_name` is `_`, producing names such as `0010_base`.
- In release-date mode, `folder_name` begins with an ISO-like sortable date prefix and `_`, such as `2026-05-19_release`.

## Rules / Constraints

- `generation_index` must be unique within a project.
- Each generation folder must contain `_config.yaml`.
- `path_name` must be unique after folder-name derivation.
- `folder_name` must be deterministic from global settings and generation fields.
- Generations sort ascending from old to new.
- Merge precedence treats later sorted generations as newer.
- Merge API callers pass selected generation IDs explicitly; request order does not define precedence.
- The server sorts requested generation IDs by the configured ordering mode before evaluating merge precedence.
- Persistent merge callers pass selected source generation IDs and destination metadata explicitly.
- Persistent merge creates a new generation from the selected sources and does not mutate the source generations.
- Persistent merge destination metadata uses the same `generation_index`, `output`, `path_name`, `description`, and derived `folder_name` rules as manual generation creation.
- Persistent merge destination `generation_index`, `path_name`, and derived `folder_name` must not collide with existing generations.
- Persistent merge record precedence is based only on source generation ordering; the destination generation's new index does not affect which source record wins.
- Generation deletion callers pass selected generation IDs explicitly.
- Generation deletion must leave at least one valid generation in the project.
- If the active edit generation is deleted, the application resolves a replacement deterministically from the remaining sorted generations.
- Generation deletion removes selected generation folders and does not alter generation ordering settings.
- Generation duplication copies exactly one source generation to a new destination generation.
- Generation duplication destination metadata uses the same rules as manual generation creation.
- Generation analysis is read-only and does not change generation metadata or table data.
- `output: false` excludes the generation from default export selection.
- A generation can still exist and be editable even when `output: false`.
- Generation metadata should be editable through an `extable` grid because it is tabular master data.
- Generation selection and generation editing screens are future features after the first runnable slice.

## Uses Common Details

- None yet.

## Reads

- [Canonical YAML file layout](canonical-yaml-file-layout.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)

## Writes

- Generation metadata configuration.
- Generation `_config.yaml` files.
- Derived generation folder names.

## Related Requirements

- [Generation editing screen](../ui-screen/generation-editing-screen.md)
- [Generation selection screen](../ui-screen/generation-selection-screen.md)
- [Web service host](../server-component/web-service-host.md)
- [Generation persistent merge flow](../data-flow/generation-persistent-merge-flow.md)
- [Generation deletion flow](../data-flow/generation-deletion-flow.md)
- [Generation duplication flow](../data-flow/generation-duplication-flow.md)
- [Generation analysis flow](../data-flow/generation-analysis-flow.md)

## Native-Language Summary

初期スライスでは固定フォルダ `0000_initial` の1世代だけを使い、世代関連UIは提供しない。将来的には世代を `generation_index`、`output`、`path_name`、`description` を持つ表形式メタデータとして扱い、数字キーや日付キーで並べる。フォルダ名は prefix と `path_name` を `_` で連結する。永続マージでは複数の選択元世代を通常の順序で統合し、同じメタデータ規則を使って新しい世代を作成する。世代複製では1つ以上の選択元世代を、現在の最大 index からの自動採番と `_copy` 系の一意な `path_name` で新しい世代にコピーする。世代削除では明示選択された世代だけを削除し、少なくとも1つの有効な世代を残す。
