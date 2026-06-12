---
id: "go-embedded-web-server-host"
type: "server-component"
title: "Go embedded web server host"
aliases: ["single-binary web host", "Go web host"]
tags: ["go", "embed", "packaging", "web-service"]
facts:
  lifecycle.status: "blueprint"
  owner: "application"
---

# Go embedded web server host

## Summary

The Go embedded web server host is the single-binary web distribution shape for MasterDataMate. It builds the React + Vite frontend, embeds the generated `dist` assets with `go:embed`, exposes the same HTTP API surface as [Web service host](web-service-host.md), and serves both API responses and the SPA from one executable.

This host replaces the development-only split between the Vite dev server and the Node/Hono API server for packaged web delivery. The frontend source remains React + Vite, but the distributable artifact is a Go executable that can run without Node.js or a checked-out `dist` directory.

## Responsibilities

- Build the frontend with Vite before Go packaging.
- Embed the frontend build output into the Go binary with `go:embed`.
- Serve immutable static assets from the embedded filesystem.
- Serve `index.html` as the SPA fallback for non-API routes.
- Implement MasterDataMate API routes in Go or through Go package boundaries that can also be reused by the Wails host.
- Read and write canonical `masterdata` project files from a configured workspace root.
- Read and write project export settings from the configured workspace root.
- Preserve the HTTP route semantics specified by [Web service host](web-service-host.md).
- Provide a CLI entry point for running the packaged web server.
- Provide a CLI entry point for standalone export batch execution through [Go CLI export runner](../batch-component/go-cli-export-runner.md).
- Optionally provide AI assistant HTTP routes, provider configuration, and agent tools through shared Go services when AI features are enabled.
- Package web delivery as one executable plus user/project data, not as separate frontend and backend processes.

## Interfaces

- HTTP API routes from [Web service host](web-service-host.md).
- Optional AI assistant and agent tool routes from [AI assistant service](../component/ai-assistant-service.md).
- Embedded SPA static file server.
- Command-line server launcher.
- Command-line export runner.
- Workspace root configuration.

## HTTP Serving Rules

- All `/api/*` paths are handled by the Go API router.
- Built asset paths such as `/assets/*` are served from the embedded Vite `dist` filesystem.
- The server returns `index.html` for non-API paths that do not match an embedded static file so SPA routes such as `/tables/product` and `/schemas` work after browser refresh.
- Missing API routes return API-style JSON errors rather than falling through to `index.html`.
- Static asset responses should include deterministic cache headers. Hashed Vite assets may be cached long-term; `index.html` should not be cached aggressively.
- The server must not require the current working directory to contain `dist` at runtime.

## CLI Behavior

- The executable starts the web server with a default localhost bind address.
- The executable supports an `export` subcommand for standalone batch export.
- The executable supports an explicit `serve` subcommand for web serving; no subcommand may remain an alias for `serve` for backward compatibility.
- The default bind host is loopback-only unless explicitly configured otherwise.
- The port is configurable. If a default port is chosen, startup failure due to port conflict must be reported clearly.
- The workspace root is configurable with `--workspace`.
- When `--workspace` is omitted, the server resolves the workspace from the process current working directory, not from the executable or wrapper script path.
- Default workspace resolution first checks whether the current working directory itself contains `masterdata/`; if it does, that directory is the workspace root.
- If the current working directory does not contain `masterdata/`, default workspace resolution walks upward and selects the nearest ancestor that contains `masterdata/`.
- Project root markers such as `go.mod`, `.git`, `package.json`, or another configured root marker may help diagnostics, but they must not cause the resolver to skip a nearer nested workspace that contains `masterdata/`.
- If the current working directory itself is inside a `masterdata` tree, resolution must return the containing workspace root so schema and generation paths remain `masterdata/schema` and `masterdata/generations`.
- If no directory containing `masterdata/` is found, startup fails clearly unless the user supplied an explicit `--workspace` path.
- An explicit `--workspace` path is resolved to an absolute path and must contain the canonical `masterdata` directory.
- Packaged binaries launched directly from `dist-native/` must still load project data based on the user's current working directory or explicit `--workspace`, not based on `dist-native/`.
- The process logs the listening URL and resolved workspace root on startup.
- The process exits non-zero when the workspace root cannot be resolved or when required `masterdata` paths are invalid.
- The `export` subcommand resolves the workspace with the same rules but does not start the web server or require embedded frontend assets at runtime.

## Build Pipeline

1. Install frontend dependencies from `package-lock.json`.
2. Run the Vite production build.
3. Run Go tests for the server and shared application packages.
4. Compile the Go executable, embedding the generated frontend assets.
5. Produce a release artifact for the target platform.

The build pipeline must make asset freshness deterministic. A Go release build should not silently embed stale frontend output when frontend sources changed but `dist` was not rebuilt.

## Dependencies

- [Product overview](../generic/product-overview.md)
- [Shared web editing frontend](../component/shared-web-editing-frontend.md)
- [Web service host](web-service-host.md)
- [Canonical YAML file layout](../data-model/canonical-yaml-file-layout.md)
- [Schema validation engine](../component/schema-validation-engine.md)
- [AI assistant service](../component/ai-assistant-service.md)
- [Agent tool contract](../component/agent-tool-contract.md)

## Reads

- Embedded Vite `dist` files.
- Schema configuration files under the configured workspace root.
- YAML table records under the configured workspace root.
- Generation configuration and generation table files under the configured workspace root.
- Export settings under `masterdata/export_settings.yaml` when export settings or export execution routes are used.
- AI provider configuration and host environment variables or secret references when AI routes are enabled.

## Writes

- YAML table records under the configured workspace root.
- Schema YAML files and generation metadata files when the corresponding API routes are implemented.
- Export settings at `masterdata/export_settings.yaml` when the export dialog saves format defaults.
- Export artifacts when export APIs are implemented.
- AI assistant conversation state, pending proposals, and confirmed AI tool outputs when AI routes are enabled.
- No frontend asset files at runtime.

## Related Requirements

- [Wails desktop host](wails-desktop-host.md)
- [Single page application shell](../ui-flow/single-page-application-shell.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)
- [Export execution flow](../data-flow/export-execution-flow.md)
- [Go CLI export runner](../batch-component/go-cli-export-runner.md)

## Native-Language Summary

Webサーバー配布版は、ViteでビルドしたReact SPAを `go:embed` でGoバイナリに同梱し、APIと静的配信を1つの実行ファイルから提供する。開発時はViteとAPIを分けてもよいが、配布時にNode.jsや外部 `dist` ディレクトリへ依存しない。同じ実行ファイルは `export` サブコマンドで Web UI を起動しないバッチエクスポートも提供する。
