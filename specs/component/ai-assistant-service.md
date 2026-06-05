---
id: "ai-assistant-service"
type: "server-component"
title: "AI assistant service"
aliases: ["LLM service", "agent runtime service"]
tags: ["ai", "llm", "agent", "provider"]
facts:
  lifecycle.status: "blueprint"
  owner: "application"
---

# AI assistant service

## Summary

The AI assistant service is the host-owned runtime that receives in-app AI assistant requests, calls the configured LLM provider, streams assistant events to the frontend, and mediates access to MasterDataMate agent tools.

The service keeps LLM interaction separate from canonical YAML persistence. The LLM may propose changes, call read-only tools, or request side-effecting tools, but writes and export actions are executed only through explicit host services and user confirmation rules.

## Responsibilities

- Load and validate [AI provider configuration model](../data-model/ai-provider-configuration-model.md) profiles.
- Normalize OpenAI, OpenAI-compatible, Ollama, and LM Studio responses into one internal assistant event model.
- Provide a streaming endpoint for the [In-app AI assistant panel](../ui-screen/in-app-ai-assistant-panel.md).
- Support text instructions from users with current table, generation, selection, and diagnostics context.
- Expose [Agent tool contract](agent-tool-contract.md) definitions to providers that support tool calls.
- Emulate tool orchestration for providers without native tool-call support when the host can safely parse structured assistant output.
- Keep all tool execution inside application service boundaries.
- Run validation before surfacing record or schema change proposals as approvable actions.
- Require user confirmation before side-effecting tools write YAML files, create generation folders, or create export artifacts.
- Return clear diagnostics for provider errors, tool errors, validation errors, and blocked confirmations.

## Interfaces

- AI chat run endpoint for assistant turns.
- Optional AG-UI-compatible event stream for frontend integration.
- Provider health check endpoint.
- Provider profile list endpoint for UI selection.
- Agent tool registry backed by application services.
- Internal provider adapter interface for OpenAI-compatible HTTP, Ollama, and LM Studio.

## Assistant Event Model

The internal assistant event model should be compatible with AG-UI concepts even when the first implementation uses a smaller local protocol.

Events include:

- run started and run finished lifecycle events.
- assistant text delta events.
- tool call requested, tool call started, tool call result, and tool call failed events.
- state update events for selected table, active generation, current diagnostics, and pending proposed changes.
- confirmation requested events for side-effecting actions.
- cancellation and error events.

When AG-UI is implemented, the service maps the internal event model to AG-UI standard events rather than rewriting the application tool layer.

## Provider Adapter Rules

- Provider adapters must receive normalized request objects containing messages, tool definitions when allowed, response format hints, and run context.
- Provider adapters must return normalized assistant messages, tool calls, usage metadata, and provider diagnostics.
- OpenAI-compatible adapters should support configurable base URL, model, API key source, headers, timeout, streaming, and tool-call capability.
- Ollama and LM Studio adapters may use provider-specific endpoints or OpenAI-compatible endpoints, but their output must be normalized before tool handling.
- A provider that does not support tool calls may still be used for read-only explanations and draft text, but side-effecting workflows require a host-parsed structured proposal or are disabled.
- The service must not send raw workspace file paths, secrets, or unrelated table data unless required by the requested tool context.
- The service should prefer scoped context over full-project context. For example, a selected table request includes the selected table schema, selected records, relevant diagnostics, and generation metadata before unrelated table records.

## Confirmation Rules

- Read-only tools may run without confirmation when they are scoped to the current workspace.
- Proposal tools that only return pending changes may run without confirmation.
- Tools that save records, save schemas, write generation metadata, create persistent merged generations, delete generations, duplicate generations, or export files require explicit user confirmation.
- The confirmation UI must show the action, affected tables or generations, destination paths when applicable, and validation diagnostics before execution.
- A denied confirmation cancels the tool call and returns a tool result explaining that the user declined the action.
- The service must not convert a proposal into a write without a frontend confirmation token issued for that proposed action.

## Dependencies

- [AI provider configuration model](../data-model/ai-provider-configuration-model.md)
- [Agent tool contract](agent-tool-contract.md)
- [In-app AI assistant panel](../ui-screen/in-app-ai-assistant-panel.md)
- [Web service host](../server-component/web-service-host.md)
- [Go embedded web server host](../server-component/go-embedded-web-server-host.md)
- [Wails desktop host](../server-component/wails-desktop-host.md)
- [Schema validation engine](schema-validation-engine.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)
- [Export execution flow](../data-flow/export-execution-flow.md)

## Reads

- AI provider configuration.
- User prompt and assistant conversation history.
- Current frontend context supplied by the AI panel.
- Schemas, records, generation metadata, validation diagnostics, and export settings through application services.

## Writes

- Assistant conversation state when retained by the host.
- Pending proposed changes shown to the user.
- Canonical YAML files only through confirmed agent tools and existing application services.
- Export artifacts only through confirmed export tools.

## Related Requirements

- [Shared web editing frontend](shared-web-editing-frontend.md)
- [HTML editor plugin runtime](html-editor-plugin-runtime.md)
- [Product overview](../generic/product-overview.md)

## Native-Language Summary

アプリ内AIパネルからの指示を受け、OpenAI互換API、Ollama、LM Studioなどへ接続し、応答やツール実行イベントをフロントエンドへ流すホスト側サービス。LLMは正本YAMLを直接書かず、読み取り・提案・検証・ユーザー承認済み実行という段階を通して既存サービスを呼び出す。
