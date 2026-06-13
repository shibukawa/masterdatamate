---
id: "wails-desktop-host"
type: "server-component"
title: "Wails desktop host"
aliases: ["desktop app host", "Wails app"]
tags: ["wails", "desktop", "packaging", "go"]
facts:
  lifecycle.status: "blueprint"
  owner: "application"
---

# Wails desktop host

## Summary

The Wails desktop host packages MasterDataMate as a desktop application while reusing the same React + Vite frontend and Go application services used by the [Go embedded web server host](go-embedded-web-server-host.md). The desktop app is a local editor for a selected `masterdata` workspace and should not require a separate browser or Node.js process after packaging.

The Wails host is a packaging and host integration layer. Business rules, YAML parsing, validation, generation merge behavior, schema editing behavior, and export execution should live in shared Go packages so the web server host and desktop host do not diverge.

## Responsibilities

- Package the React + Vite frontend into the Wails app asset bundle.
- Start a native desktop window that renders the shared SPA.
- Bind Go application services to the frontend through Wails bindings or a local HTTP bridge.
- Let the user open or configure a workspace root containing `masterdata`.
- Reuse the same validation, persistence, generation, schema, and export services as the web server host.
- Reuse the same AI assistant, provider configuration, and agent tool services as the web server host when AI features are enabled.
- Reuse the same AI settings and OS-backed credential storage services as the web server host when AI features are enabled.
- Use the shared Go keyring adapter for AI credential storage so desktop and local web behavior match.
- Preserve SPA navigation and dirty-state behavior from [Single page application shell](../ui-flow/single-page-application-shell.md).
- Provide desktop build artifacts for supported platforms.

## Interfaces

- Wails frontend asset bundle.
- Wails Go bindings for workspace and application services, or an internal localhost HTTP adapter matching [Web service host](web-service-host.md).
- Wails bindings or an internal HTTP adapter for optional AI assistant runs and agent tool execution.
- Wails bindings or an internal HTTP adapter for AI settings and credential storage.
- Native file or directory selection for workspace root selection.
- Shared Go service packages used by both desktop and web server hosts.

## Host Integration Rules

- The desktop app must not fork a separate Node.js API server.
- The desktop app must not depend on a runtime Vite dev server.
- File writes go through shared Go services, not direct browser filesystem access.
- The frontend API adapter may target Wails bindings in desktop mode and HTTP `fetch` in web-server mode, but both adapters must expose the same logical operations to React components.
- The desktop app should remember the last opened workspace when platform storage is available.
- If the selected workspace is missing required `masterdata` roots, the app reports a blocking workspace diagnostic before editing screens open.
- Desktop packaging must keep Git approval, review, and repository permission management outside the application scope, matching [Product overview](../generic/product-overview.md).

## Build Pipeline

1. Install frontend dependencies from `package-lock.json`.
2. Run the Vite production build used by Wails.
3. Run Go tests for shared application packages.
4. Run the Wails build for the target platform.
5. Produce platform-native desktop artifacts.

The Wails build pipeline should share as much frontend build configuration as practical with the Go web server build. Differences between web and desktop runtime adapters should be explicit and covered by tests at the adapter boundary.

## Dependencies

- [Product overview](../generic/product-overview.md)
- [Shared web editing frontend](../component/shared-web-editing-frontend.md)
- [Go embedded web server host](go-embedded-web-server-host.md)
- [Web service host](web-service-host.md)
- [Canonical YAML file layout](../data-model/canonical-yaml-file-layout.md)
- [Schema validation engine](../component/schema-validation-engine.md)
- [AI assistant service](../component/ai-assistant-service.md)
- [AI secret storage service](../component/ai-secret-storage-service.md)
- [Agent tool contract](../component/agent-tool-contract.md)

## Reads

- Bundled frontend assets.
- Workspace root preference when available.
- Schema configuration files under the selected workspace root.
- YAML table records and generation files under the selected workspace root.
- AI provider configuration and host-specific secret references when AI features are enabled.
- OS credential store entries for AI provider credentials through the shared Go keyring adapter when AI settings are enabled.

## Writes

- Workspace root preference when available.
- YAML table records under the selected workspace root.
- Schema YAML files and generation metadata files when the corresponding services are implemented.
- Export artifacts when export services are implemented.
- AI assistant conversation state, pending proposals, and confirmed AI tool outputs when AI features are enabled.
- AI provider profile metadata. Raw API keys are written only to the OS credential store through the secret storage service.
- No frontend source or build output files at runtime.

## Related Requirements

- [Single page application shell](../ui-flow/single-page-application-shell.md)
- [Table editing workspace](../ui-screen/table-editing-workspace.md)
- [Generation editing screen](../ui-screen/generation-editing-screen.md)
- [Schema editing screen](../ui-screen/schema-editing-screen.md)

## Native-Language Summary

Wails版は、同じReact + ViteフロントエンドとGoの共有サービスを使ってデスクトップアプリとして配布する。配布後はNode.jsやVite dev serverを起動せず、ワークスペース選択・YAML読み書き・検証・エクスポートは共有Goサービス経由で行う。
