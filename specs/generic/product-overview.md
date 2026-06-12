---
id: "product-overview"
type: "system"
title: "Product overview"
aliases: []
tags: ["product", "master-data"]
facts:
  lifecycle.status: "blueprint"
---

# Product overview

## Summary

MasterDataMate is a schema-driven master data editing application for tabular data that teams previously maintained in Excel. It stores canonical data in Git-friendly YAML, validates records against table-local schemas, and exports selected generations into runtime-friendly formats.

## Scope

- In scope:
  - General-purpose table-like master data editing.
  - YAML as the canonical Git-friendly storage format.
  - Schema-driven validation and input assistance.
  - Generation-based dataset layering.
  - Export to backend-specific formats.
  - Web service as the first host shape.
  - Single-binary web distribution that embeds the frontend build into a Go server.
  - Desktop application distribution with Wails using the shared frontend and Go application services.
  - Shared web frontend core that can later be reused by editor extensions and standalone apps.
  - Optional project-local editor plugins for visual or domain-specific master data editing.
  - Optional in-app AI assistant panel for natural-language guidance, analysis, proposed changes, and confirmed tool execution.
  - OpenAI-compatible hosted or local LLM provider configuration, including OpenAI, Ollama, and LM Studio profiles.
- Out of scope:
  - Approval workflow.
  - Repository permission management.
  - GitHub review orchestration.
  - A single fixed production database backend.

## Goals

- Support both developers and non-engineering planners.
- Replace Excel-based editing with structured, validated data entry.
- Keep Git diffs reviewable and merge-friendly.
- Let GitHub or a similar platform handle review, approval, and access control.
- Ship the first runnable application as a web service.
- Ship packaged web delivery as one Go executable with embedded frontend assets.
- Keep the Wails desktop app and web server host aligned through shared Go services.
- Keep the frontend core reusable for later editor-extension and standalone-app hosts.
- Allow teams to provide custom editing surfaces for data such as maps while keeping YAML, schemas, validation, and export as the canonical system of record.

## Primary Concepts

- A **table** is a schema-defined collection of records.
- A **record** is a row-like value object with schema fields, primary key values, optional display name, optional memo, and metadata.
- A **schema** belongs to a table and defines field types, primary key columns, constants, validation, and external reference behavior.
- A **formula field** is a schema-defined read-only field computed from non-formula fields in the same record.
- An **export flag** controls whether a schema field is included in generated outputs.
- A **generation** is an ordered dataset layer. Exporting selected generations merges records by primary key, with newer generations taking precedence.
- An **export backend** converts merged canonical data into CSV, YAML, SQLite, SQL DML, or other target formats.
- An **editor plugin** is a project-local custom editing surface, delivered as built HTML/JavaScript/CSS assets and optionally maintained from a source project such as Vite, that reads and writes declared table records through the same validation and save APIs as the ordinary table editor.
- An **AI assistant** is an optional host-owned natural-language assistant that can read scoped project context, propose changes, and request tool execution through explicit application tools.
- An **agent tool** is an application-owned operation exposed to the AI assistant for discovery, reading, analysis, proposal, or confirmed execution. Tools never expose raw filesystem or shell access.

## Initial Runnable Slice

- Host shape: web service.
- Packaged host shape: Go web server with embedded React + Vite assets.
- Desktop host shape: Wails app using the same frontend and shared Go services.
- Schema editing: out of scope for the first slice; schemas are configured through files.
- Formula evaluation: out of scope for the first slice unless a minimal evaluator is implemented opportunistically.
- Generations: exactly one fixed generation in the first slice, stored in `masterdata/generations/0000_initial`.
- Generation UI: out of scope for the first slice.
- Editing: provide schema-driven table editing through the [Table editing workspace](../ui-screen/table-editing-workspace.md).
- Persistence: load and save canonical YAML files.
- Validation: show validation diagnostics during editing.
- Save behavior: use `warn_and_save`; validation errors trigger a confirmation dialog, and confirmed saves are sent with `force: true`.
- Export: not required for the first slice except that export-blocking validation semantics must be specified for later work.

## Related Documents

- [Generic master data model](../data-model/generic-master-data-model.md)
- [Table schema model](../data-model/table-schema-model.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)
- [Schema validation engine](../component/schema-validation-engine.md)
- [Shared web editing frontend](../component/shared-web-editing-frontend.md)
- [Editor plugin model](../data-model/editor-plugin-model.md)
- [HTML editor plugin runtime](../component/html-editor-plugin-runtime.md)
- [Export backend adapters](../component/export-backend-adapters.md)
- [AI provider configuration model](../data-model/ai-provider-configuration-model.md)
- [AI assistant service](../component/ai-assistant-service.md)
- [Agent tool contract](../component/agent-tool-contract.md)
- [In-app AI assistant panel](../ui-screen/in-app-ai-assistant-panel.md)
- [Web service host](../server-component/web-service-host.md)
- [Go embedded web server host](../server-component/go-embedded-web-server-host.md)
- [Wails desktop host](../server-component/wails-desktop-host.md)
- [Single page application shell](../ui-flow/single-page-application-shell.md)

## Native-Language Summary

汎用的な表形式マスタデータを、Gitで扱いやすいYAMLを正本として編集・検証・出力するアプリケーション。承認や権限はGitHub側に任せ、アプリ本体はスキーマ駆動の編集体験とエクスポートに集中する。配布時はReact + ViteフロントエンドをGoへ同梱したWebサーバー版と、同じ共有サービスを使うWailsデスクトップ版を提供する。
