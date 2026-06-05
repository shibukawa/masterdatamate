---
id: "generation-selection-screen"
type: "ui-screen"
title: "Generation selection screen"
aliases: []
tags: ["ui", "generation", "selection"]
facts:
  lifecycle.status: "blueprint"
---

# Generation selection screen

## Summary

The generation selection screen lets users choose which ordered generations are active for preview and later export. This screen is out of scope for the first runnable slice, which uses fixed generation folder `0000_initial`.

Generation selection is not the same as generation metadata editing. Metadata editing is handled by the dedicated [Generation editing screen](generation-editing-screen.md).

## User Goals

- See the available generations in order.
- Toggle generations that participate in effective dataset preview and export, initialized from each generation's `output` flag.
- Choose whether table data editing previews only the active edit generation or includes previous output-enabled generations.
- Understand which generation wins when records share the same primary key.
- See generation folder names and descriptions.
- Navigate to generation metadata editing.
- Return to table editing with the selected generation set applied.

## States

- First slice: screen not available.
- Later slice: one generation available and selected.
- Multiple generations available.
- No generation selected.
- Some generations have `output: false`.
- Selection has validation warnings.
- Selection is dirty and not applied.
- Generation list failed to load.

## Invoked APIs

- Load generation list.
- Load active generation selection.
- Load default output flags from generation metadata.
- Save active generation selection.
- Validate selected generation set.
- Navigate to generation editing.
- Later slice: compute effective record preview summary.

## Components

- [Single page application shell](../ui-flow/single-page-application-shell.md)
- [Generation model](../data-model/generation-model.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)
- [Web service host](../server-component/web-service-host.md)

## Related Requirements

- [Product overview](../generic/product-overview.md)
- [Canonical YAML file layout](../data-model/canonical-yaml-file-layout.md)

## Native-Language Summary

世代選択画面。初期スライスでは提供せず、固定 `0000_initial` を使う。将来は世代メタデータの `output` フラグを初期選択として使い、複数世代をトグル選択する。
