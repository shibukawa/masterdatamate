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
- Resolve provider credentials through [AI secret storage service](ai-secret-storage-service.md) without exposing secrets to the frontend or assistant context.
- Load, persist, switch, and compact [AI assistant session model](../data-model/ai-assistant-session-model.md) conversation state for each workspace.
- Normalize OpenAI, OpenAI-compatible, Ollama, LM Studio, Codex CLI, and Foundation Models `fm` responses into one internal assistant event model.
- Provide a streaming endpoint for the [In-app AI assistant panel](../ui-screen/in-app-ai-assistant-panel.md).
- Provide a Copilot Runtime-compatible endpoint for CopilotKit UI integration when the React frontend uses CopilotKit.
- Support text instructions from users with current table, generation, selection, and diagnostics context.
- Expose [Agent tool contract](agent-tool-contract.md) definitions to providers that support tool calls.
- Emulate tool orchestration for providers without native tool-call support when the host can safely parse structured assistant output.
- Keep all tool execution inside application service boundaries.
- Run validation before surfacing record or schema change proposals as approvable actions.
- Require user confirmation before side-effecting tools write YAML files, create generation folders, or create export artifacts.
- Return clear diagnostics for provider errors, tool errors, validation errors, and blocked confirmations.

## Interfaces

- AI chat run endpoint for assistant turns.
- AI session list, create, load, rename, archive, delete, and compact endpoints.
- Copilot Runtime-compatible endpoint for CopilotKit popup chat integration.
- Optional AG-UI-compatible event stream for frontend integration.
- Provider health check endpoint.
- Provider profile list endpoint for UI selection.
- AI settings load/save endpoint support for provider metadata and credential presence.
- Agent tool registry backed by application services.
- Internal provider adapter interface for OpenAI-compatible HTTP, Ollama, LM Studio, Codex CLI, and Foundation Models `fm`.

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
- OpenAI-compatible adapters must emit tool schemas in the strict baseline defined by [Agent tool contract](agent-tool-contract.md) for all providers, not only for providers known to reject loose schemas.
- The first OpenAI-compatible implementation should target Apple Foundation Models `fm serve` because it is local and can be used without a paid hosted API key when available.
- Ollama and LM Studio adapters may use provider-specific endpoints or OpenAI-compatible endpoints, but their output must be normalized before tool handling.
- Codex CLI adapters should use non-interactive command execution, prefer JSONL event output when available, and map Codex progress, command execution, file changes, and final messages into assistant events.
- Foundation Models `fm` adapters should execute the local `fm` command on macOS, detect installed command capabilities, and map text, structured output, tool-call, and tool-result events into assistant events when the installed command exposes them.
- Foundation Models `fm serve` bridge mode should be represented as an OpenAI-compatible local profile after the host starts or verifies the local server process.
- Foundation Models `fm serve` bridge mode should prefer a Go-managed process over a Unix domain socket when the installed command supports UDS. Loopback TCP is acceptable for user-managed servers, debugging, or compatibility fallback.
- A managed `fm serve` adapter must hide the generated socket path from the browser and expose only profile readiness, health diagnostics, and capability metadata.
- Command-backed adapters must support process cancellation, timeout, stderr diagnostics, and non-zero exit handling.
- A provider that does not support tool calls may still be used for read-only explanations and draft text, but side-effecting workflows require a host-parsed structured proposal or are disabled.
- The service must not send raw workspace file paths, secrets, or unrelated table data unless required by the requested tool context.
- The service should prefer scoped context over full-project context. For example, a selected table request includes the selected table schema, selected records, relevant diagnostics, and generation metadata before unrelated table records.

## Session Persistence And Context Management

The service owns server-side assistant session persistence through [AI assistant session model](../data-model/ai-assistant-session-model.md). A session stores the prompt-context message sequence, safe run summaries, retained tool-card data, and compaction records for one workspace.

- Assistant runs must be associated with an explicit `sessionId`. If the frontend starts a run without one, the service creates a new session and returns its ID in the first run event.
- Session storage is scoped to the resolved workspace under `os.UserConfigDir()/masterdatamate/<workspaceBaseName>-<workspacePathHash>/ai-sessions/`.
- The service sends only the selected session's budgeted prompt context to the provider. Other sessions in the same workspace are never included implicitly.
- Before each provider call, the service estimates the prompt budget from system instructions, retained session messages, current UI context, tool definitions, and reserved output/tool-result headroom.
- When the estimated request exceeds the active provider's budget, the service compacts older session messages into a summary before running or returns a diagnostic if compaction is unavailable.
- Compaction rewrites the session prompt context but records a compaction entry with the replaced message IDs and resulting summary message ID.
- Recent turns, pending user decisions, unconfirmed proposals, validation findings, and accepted constraints are preserved verbatim where practical. Large stale tool outputs and repeated diagnostics are summarized first.
- Confirmation tokens, raw credentials, provider request headers with secrets, and uncommitted side-effect authorization state are never persisted as reusable session data.
- The service should expose manual compaction so a user can clean up a long session before asking follow-up questions.

## Copilot Runtime Integration

The service should expose the MasterDataMate assistant through a Copilot Runtime-compatible endpoint so the React frontend can use CopilotKit prebuilt components.

- The default CopilotKit agent maps to the active MasterDataMate provider profile, initially favoring an OpenAI-compatible `fm serve` profile when available.
- Server tools registered with the Copilot Runtime-compatible layer are adapters over [Agent tool contract](agent-tool-contract.md) tools.
- Frontend tools registered by CopilotKit may request browser UI actions, but they must call host APIs for any canonical data operation.
- Agent state should include selected table, selected generation, visible diagnostics, staged uploads, pending proposal IDs, and confirmation state.
- AG-UI event streams remain the normalized transport for messages, tool calls, state updates, and confirmation prompts.
- The runtime endpoint must enforce the same authentication and workspace scoping as ordinary application APIs.
- Provider profile responses used by CopilotKit or ordinary AI settings must include only masked credential presence, never raw API keys.

## Confirmation Rules

- Read-only tools may run without confirmation when they are scoped to the current workspace.
- Proposal tools that only return pending changes may run without confirmation.
- Tools that save records, save schemas, write generation metadata, create persistent merged generations, delete generations, duplicate generations, or export files require explicit user confirmation.
- The confirmation UI must show the action, affected tables or generations, destination paths when applicable, and validation diagnostics before execution.
- A denied confirmation cancels the tool call and returns a tool result explaining that the user declined the action.
- The service must not convert a proposal into a write without a frontend confirmation token issued for that proposed action.

## Local Agent Provider Rules

Codex CLI and Foundation Models `fm` are local command-backed providers. They may be selected independently of OpenAI-compatible HTTP providers.

- Codex CLI is intended for heavier repository-aware tasks such as spec edits, code changes, review, and multi-step diagnosis.
- Foundation Models `fm` is intended for local macOS model use where low external-data exposure and local tool-capable inference are desirable.
- Foundation Models `fm` may run either as a direct command adapter or through `fm serve` as a local Chat Completions API bridge.
- The AI panel may expose both providers side by side in the provider selector when the host health check confirms availability.
- Local command providers must not require an OpenAI-compatible API endpoint.
- Local command providers may still consume the same [Agent tool contract](agent-tool-contract.md) definitions after the adapter maps the host tool schema into the provider's supported format.
- If a command provider has native tool calling, host tools should be exposed through that mechanism.
- If a command provider lacks a stable native tool-call schema, only read-only explanation and structured proposal workflows are enabled until an adapter-specific parser is implemented and verified.
- The service must clearly indicate when a provider is unavailable because the command is missing, the macOS version lacks Foundation Models support, Codex is not authenticated, or the provider cannot support the requested tool workflow.
- The service must not assume `fm` tool-call support from command availability alone; it must be proven by adapter capability tests or disabled for execution tools.

## Default Provider Rules

- If AI configuration has no explicit active profile and the host is macOS with `fm` available, the service should synthesize or select the `apple-fm-serve` profile by default.
- The automatic `apple-fm-serve` default must be treated as a local provider that requires no API key.
- The automatic `apple-fm-serve` default should use managed UDS bridge mode when supported; otherwise it may fall back to a verified loopback `fm serve` endpoint or direct command mode with tool execution disabled until verified.
- A user-selected active profile always overrides automatic `fm` defaulting.
- If `fm serve` must be started by the host, startup state and failure diagnostics are reported through provider health-check APIs.

## Dependencies

- [AI provider configuration model](../data-model/ai-provider-configuration-model.md)
- [AI assistant session model](../data-model/ai-assistant-session-model.md)
- [AI secret storage service](ai-secret-storage-service.md)
- [Agent tool contract](agent-tool-contract.md)
- [In-app AI assistant panel](../ui-screen/in-app-ai-assistant-panel.md)
- [AI settings screen](../ui-screen/ai-settings-screen.md)
- [Web service host](../server-component/web-service-host.md)
- [Go embedded web server host](../server-component/go-embedded-web-server-host.md)
- [Wails desktop host](../server-component/wails-desktop-host.md)
- [Schema validation engine](schema-validation-engine.md)
- [Generation merge and export flow](../data-flow/generation-merge-and-export-flow.md)
- [Export execution flow](../data-flow/export-execution-flow.md)

## Reads

- AI provider configuration.
- AI assistant session metadata and prompt context.
- User prompt and assistant conversation history.
- Current frontend context supplied by the AI panel.
- Schemas, records, generation metadata, validation diagnostics, and export settings through application services.

## Writes

- Assistant session metadata, prompt context, run summaries, safe tool-card data, and compaction records when retained by the host.
- Pending proposed changes shown to the user.
- Canonical YAML files only through confirmed agent tools and existing application services.
- Export artifacts only through confirmed export tools.

## Related Requirements

- [Shared web editing frontend](shared-web-editing-frontend.md)
- [HTML editor plugin runtime](html-editor-plugin-runtime.md)
- [Product overview](../generic/product-overview.md)

## Native-Language Summary

アプリ内AIパネルからの指示を受け、OpenAI互換API、Ollama、LM Studio、Codex CLI、macOSの `fm` Foundation Modelsコマンドなどへ接続し、応答やツール実行イベントをフロントエンドへ流すホスト側サービス。LLMやローカルagentは正本YAMLを直接書かず、読み取り・提案・検証・ユーザー承認済み実行という段階を通して既存サービスを呼び出す。
