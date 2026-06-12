---
id: "binary-asset-model"
type: "data-model"
title: "Binary asset model"
aliases: ["file-upload-model", "record-binary-asset"]
tags: ["binary", "file", "upload", "asset"]
facts:
  lifecycle.status: "blueprint"
  data.name: "binary-asset"
---

# Binary asset model

## Summary

Binary assets are files attached to schema-defined records, such as images, icons, maps, audio clips, or other non-YAML data. The file bytes live under `masterdata/binaries`, while canonical table records may store only metadata needed for display, validation, and export.

The first supported storage pattern is one binary asset per record and table:

```text
masterdata/binaries/<table>/<primary-key>.<extension>
```

For example, the icon for item record `potion_small` in table `item` may be stored as:

```text
masterdata/binaries/item/potion_small.png
```

## Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| binary_root | path | yes | `masterdata/binaries`. Contains uploaded file bytes. |
| table_directory | path segment | yes | Table `system_name`, such as `item` or `map`. |
| asset_filename | path segment | yes | Deterministic primary-key filename plus extension. |
| asset_extension | string | yes | Lowercase extension derived from the uploaded file or accepted upload metadata. |
| record_key | scalar or object | yes | Record primary key used to derive the filename. |
| field_metadata | object | no | Optional YAML field value for schema fields of type `binary_file`. |
| field_metadata.extension | string | yes | File extension without leading dot. |
| field_metadata.mime_type | string | no | Browser- or server-detected MIME type. |
| field_metadata.size_bytes | integer | no | Stored file size. |
| field_metadata.sha256 | string | no | Hex digest for change detection and export verification. |
| field_metadata.original_name | string | no | Original client filename for display only. |
| field_metadata.updated_at | datetime | no | Host-generated update timestamp when the implementation records it. |

## Path Derivation

- The physical path is derived from table name, record primary key, and extension.
- Scalar primary keys use the scalar value as the filename stem after safe filename encoding.
- Composite primary keys use deterministic key serialization before safe filename encoding.
- Safe filename encoding must prevent path traversal and directory separators.
- The encoded primary key must be stable across hosts so Git diffs and file references remain predictable.
- The extension is normalized to lowercase and must not contain path separators.
- The default path does not include field name, so the first implementation supports at most one `binary_file` field per table.
- If multiple file fields per record are needed later, the path pattern must be extended explicitly, such as `masterdata/binaries/<table>/<field>/<primary-key>.<extension>`.

## Table Schema Integration

- A schema field may declare `type: binary_file`.
- A `binary_file` field stores file metadata, not file bytes.
- The field value should not store a relative path when the default path can be derived from table, primary key, and extension.
- A required `binary_file` field requires both valid metadata and a matching file under `masterdata/binaries`.
- Optional `binary_file` fields may be empty and have no stored file.
- Schema constraints may declare allowed extensions, allowed MIME types, and maximum file size.
- `binary_file` fields are editable in table editing and plugin editing, but bytes are saved through binary upload APIs rather than ordinary YAML row commit operations.

## Upload Behavior

- Uploading a file for a record writes file bytes under `masterdata/binaries/<table>/<primary-key>.<extension>`.
- If the previous asset for the same record has a different extension, the host should delete or mark the old file for cleanup as part of the same user action.
- The upload API returns metadata that can be stored in the record's `binary_file` field.
- The host should update file bytes and metadata as one user-visible action when practical.
- If metadata save fails after file bytes are written, the host must return recovery diagnostics and reload the record and asset state.
- File deletion removes the stored binary file and clears or updates the associated `binary_file` metadata.

## Rules / Constraints

- Binary assets are scoped to `masterdata/binaries`; upload APIs must not write elsewhere.
- Upload APIs must reject path traversal, unknown tables, unknown records, invalid primary keys, unsupported extensions, and files that exceed configured size limits.
- The browser must not write files directly; uploads always go through host APIs.
- The table editor and plugin runtime must use the same binary asset APIs.
- The file dialog and drag-and-drop flows are UI conveniences over the same host upload operation.
- Binary files are not stored inside generation folders in the first model; they are project-level assets keyed by table and primary key.
- Because binaries are project-level in the first model, generation-specific binary variants require a later extension to the path model.
- Export adapters may include binary metadata or copy binary files only when the export format explicitly supports assets.
- Removing or renaming a primary key that has a binary asset must either move the asset to the new derived path or report a cleanup diagnostic.
- Table schema rename must move matching binary table directories or report a cleanup diagnostic.

## Uses Common Details

- None yet.

## Reads

- [Canonical YAML file layout](canonical-yaml-file-layout.md)
- [Table schema model](table-schema-model.md)
- [Generic master data model](generic-master-data-model.md)

## Writes

- File bytes under `masterdata/binaries`.
- Optional `binary_file` metadata values inside table records.

## Related Requirements

- [Table editing workspace](../ui-screen/table-editing-workspace.md)
- [HTML editor plugin runtime](../component/html-editor-plugin-runtime.md)
- [Web service host](../server-component/web-service-host.md)
- [Schema validation engine](../component/schema-validation-engine.md)

## Native-Language Summary

画像などのファイル実体は `masterdata/binaries/<table>/<primary-key>.<extension>` に保存する。YAML レコード側には `binary_file` フィールドとして拡張子、MIME、サイズ、ハッシュ、元ファイル名などのメタデータだけを持てる。表形式 UI でも非表形式プラグインでも同じアップロード API を使い、ブラウザやプラグインが直接ファイルを書かない。
