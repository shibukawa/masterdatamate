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

The AI assistant service is the host-owned runtime that receives in-app AI assistant requests, either runs the MasterDataMate-managed chat agent against an API-level model provider or delegates heavier work to an external agent runtime, streams assistant events to the frontend, and mediates access to MasterDataMate agent tools.

The service keeps LLM interaction separate from canonical YAML persistence. In the initial managed-chat runtime, the LLM may read context, draft table operations, validate them, and stage accepted operations into the ordinary editor working copy, but it does not write canonical YAML.

## Responsibilities

- Load and validate [AI provider configuration model](../data-model/ai-provider-configuration-model.md) profiles.
- Resolve provider credentials through [AI secret storage service](ai-secret-storage-service.md) without exposing secrets to the frontend or assistant context.
- Load, persist, switch, and compact [AI assistant session model](../data-model/ai-assistant-session-model.md) conversation state for each workspace when running the managed chat agent.
- Normalize OpenAI, OpenAI-compatible, Ollama, LM Studio, and Foundation Models `fm serve` responses into one internal assistant event model for the managed chat agent.
- Delegate Codex CLI and similar external agent runs through a separate external-agent adapter path instead of mixing them into the managed chat provider pipeline.
- Provide a streaming endpoint for the [In-app AI assistant panel](../ui-screen/in-app-ai-assistant-panel.md).
- Provide a Copilot Runtime-compatible endpoint for CopilotKit UI integration when the React frontend uses CopilotKit.
- Support text instructions from users and let the model request current browser state through frontend tools when that state is needed.
- Expose [Agent tool contract](agent-tool-contract.md) definitions to providers that support tool calls.
- Broker frontend tool calls between the provider-facing tool loop and the active [In-app AI assistant panel](../ui-screen/in-app-ai-assistant-panel.md) browser session.
- Select a compact or specialized runtime tool profile from [Agent tool contract](agent-tool-contract.md) before each managed chat run.
- Require reliable tool-calling support for draft-staging workflows. Providers without tool calls may be used only for read-only explanation and text suggestions in the initial managed-chat implementation.
- Keep all tool execution inside application service boundaries.
- Run structural checks before applying AI-drafted table record operations to the editor working copy, then rely on the existing editor validation flow for full validation display.
- Keep canonical table YAML writes behind the ordinary editor commit flow.
- Return clear diagnostics for provider errors, tool errors, validation errors, and blocked draft staging.

## Interfaces

- AI chat run endpoint for assistant turns.
- AI session list, create, load, rename, archive, delete, and compact endpoints.
- Copilot Runtime-compatible endpoint for CopilotKit popup chat integration.
- Optional AG-UI-compatible event stream for frontend integration.
- Provider health check endpoint.
- Provider profile list endpoint for UI selection.
- AI settings load/save endpoint support for provider metadata and credential presence.
- Agent tool registry backed by application services.
- Internal provider adapter interface for managed chat providers such as OpenAI-compatible HTTP, Ollama, LM Studio, and Foundation Models `fm serve`.
- External-agent adapter interface for delegated agent runtimes such as Codex CLI.

## Assistant Event Model

The internal assistant event model should be compatible with AG-UI concepts even when the first implementation uses a smaller local protocol.

Events include:

- run started and run finished lifecycle events.
- assistant text delta events.
- tool call requested, tool call started, tool call result, and tool call failed events.
- frontend tool requested, frontend tool result, frontend tool timeout, and frontend tool blocked events.
- state update events for selected table, active generation, current diagnostics, table pagination, and editor dirty state.
- editor draft staged events for AI-applied working-copy changes.
- cancellation and error events.

When AG-UI is implemented, the service maps the internal event model to AG-UI standard events rather than rewriting the application tool layer.

## Provider Adapter Rules

- Provider adapters must receive normalized request objects containing messages, tool definitions when allowed, response format hints, and run metadata. Current browser state should be supplied through frontend tool results instead of being eagerly embedded into every request.
- Provider adapters must return normalized assistant messages, tool calls, usage metadata, and provider diagnostics.
- OpenAI-compatible adapters should support configurable base URL, model, API key source, headers, timeout, streaming, and tool-call capability.
- OpenAI-compatible adapters must emit tool schemas in the strict baseline defined by [Agent tool contract](agent-tool-contract.md) for all providers, not only for providers known to reject loose schemas.
- The first OpenAI-compatible implementation should target Apple Foundation Models `fm serve` because it is local and can be used without a paid hosted API key when available.
- Foundation Models `fm serve` with the `system` model should default to the `minimal_managed_chat` tool profile because its context budget is small. The service must not send the full canonical tool catalog to this profile.
- Ollama and LM Studio adapters may use provider-specific endpoints or OpenAI-compatible endpoints, but their output must be normalized before tool handling.
- Codex CLI adapters belong to the external-agent adapter path. They should use non-interactive command execution, prefer JSONL event output when available, and map Codex progress, command execution, file changes, and final messages into assistant events without importing Codex's internal prompt history into the managed chat session model.
- Foundation Models `fm` adapters should execute the local `fm` command on macOS, detect installed command capabilities, and map text, structured output, tool-call, and tool-result events into assistant events when the installed command exposes them.
- Foundation Models `fm serve` bridge mode should be represented as an OpenAI-compatible local profile after the host verifies that the user-managed local server is reachable.
- Foundation Models `fm serve` bridge mode should use loopback TCP by default for the initial implementation. Unix domain socket and host-managed process lifecycle are deferred until the installed `fm` command behavior is proven reliable.
- The `fm serve` adapter must expose only profile readiness, health diagnostics, and capability metadata to the browser.
- Command-backed adapters must support process cancellation, timeout, stderr diagnostics, and non-zero exit handling.
- A provider that does not support tool calls may still be used for read-only explanations and draft text, but editor draft staging is disabled.
- The service must not send raw workspace file paths, secrets, or unrelated table data unless required by the requested tool context.
- The service should prefer tool-gathered scoped context over full-project context. For example, a selected table task first asks the frontend for active table and selection, then uses backend reads only for the table, records, diagnostics, and generation metadata needed to answer the current instruction.

## Frontend Tool Brokering

The managed chat agent uses one provider-facing tool-call loop, but individual tools may execute in the browser or on the host service.

1. The user sends a message for the selected session. The run request includes session ID, profile/runtime selection, and the text instruction, but should not include a full UI-state snapshot by default.
2. If the model calls a frontend tool such as `get_current_context`, the service emits a frontend tool request event over the active assistant event stream.
3. The in-app panel executes the requested frontend tool against the current browser/editor state and returns a structured tool result.
4. The service appends that tool result to the provider conversation and continues the provider tool-call loop.
5. If the model calls a frontend draft-staging tool such as `stage_table_changes`, the same event path sends the proposed operations to the browser editor. The browser applies accepted operations to the existing working copy and returns accepted/rejected details.
6. Backend tools such as `get_project_overview`, `get_table`, and `validate_table` execute directly on the service and return results through the same provider-facing tool-result shape.

Frontend tool calls are tied to an active browser connection for the run. If the browser disconnects, changes route, or can no longer supply the requested state safely, the frontend tool result is `blocked` or `error`; the service must not synthesize stale UI state on the server.

The service may include a small run envelope in the initial provider request, such as workspace identity, available tool profile, and session summary. Browser-owned details such as selected row, visible row page, dirty state, pending editor values, and scroll/page context are obtained through `get_current_context`.

## Session Persistence And Context Management

The service owns server-side assistant session persistence through [AI assistant session model](../data-model/ai-assistant-session-model.md). A managed chat session stores the prompt-context message sequence, safe run summaries, retained tool-card data, and compaction records for one workspace.

- Assistant runs must be associated with an explicit `sessionId`. If the frontend starts a run without one, the service creates a new session and returns its ID in the first run event.
- Session storage is scoped to the resolved workspace under `os.UserConfigDir()/masterdatamate/<workspaceBaseName>-<workspacePathHash>/ai-sessions/`.
- The service sends only the selected session's budgeted prompt context to the provider. Other sessions in the same workspace are never included implicitly.
- Before each provider call, the service estimates the prompt budget from system instructions, retained session messages, tool definitions, retained safe tool results, and reserved output/tool-result headroom.
- Tool definitions are part of the prompt budget. The service may reduce the emitted tool profile before compacting conversation history when tool schemas are the dominant prompt cost.
- When the estimated request exceeds the active provider's budget, the service compacts older session messages into a summary before running or returns a diagnostic if compaction is unavailable.
- Compaction rewrites the session prompt context but records a compaction entry with the replaced message IDs and resulting summary message ID.
- Recent turns, pending user decisions, staged draft summaries, validation findings, and accepted constraints are preserved verbatim where practical. Large stale tool outputs and repeated diagnostics are summarized first.
- Raw credentials, provider request headers with secrets, and uncommitted editor draft state are never persisted as reusable session authorization data. Frontend tool results that contain pending editor values may be retained only as short-lived run data or safe summaries, not as durable authority for future changes.
- The service should expose manual compaction so a user can clean up a long session before asking follow-up questions.

For delegated external agent sessions, the service persists UI-facing task records and safe summaries only. It does not compact, replay, or rewrite the external agent's internal context. Codex CLI session continuation, context window handling, and compaction are delegated to Codex itself when Codex exposes that behavior.

## Runtime Selection

The service exposes two top-level assistant runtime families:

- Managed chat agent: MasterDataMate owns sessions, prompt context, automatic compaction, tool definitions, provider calls, draft staging, and AG-UI event streaming. This path supports API-level providers such as OpenAI-compatible HTTP, Ollama, LM Studio, and `fm serve`.
- Delegated external agent: MasterDataMate launches or connects to an existing agent runtime such as Codex CLI, supplies a scoped task instruction and workspace constraints, and renders its events/results. The external agent owns planning, context management, compaction, and low-level model choice.

Runtime selection is explicit per run or per session. The service must not silently continue a managed chat session inside Codex CLI, and must not convert a Codex CLI task transcript into managed prompt history. A delegated external agent can still call back into MasterDataMate tools only through explicit adapter contracts and confirmation rules.

## Copilot Runtime Integration

The service should expose the MasterDataMate assistant through a Copilot Runtime-compatible endpoint so the React frontend can use CopilotKit prebuilt components.

- The default CopilotKit agent maps to the active MasterDataMate provider profile, initially favoring an OpenAI-compatible `fm serve` profile when available.
- Server tools registered with the Copilot Runtime-compatible layer are adapters over [Agent tool contract](agent-tool-contract.md) tools.
- Frontend tools registered by CopilotKit may request browser UI actions, but they must call host APIs for any canonical data operation.
- Agent state should be obtainable through frontend tools for selected table, selected generation, visible diagnostics, visible rows, and editor dirty-state metadata.
- AG-UI event streams remain the normalized transport for messages, tool calls, state updates, and editor draft staging.
- The runtime endpoint must enforce the same authentication and workspace scoping as ordinary application APIs.
- Provider profile responses used by CopilotKit or ordinary AI settings must include only masked credential presence, never raw API keys.

## Confirmation Rules

- Read-only and validation tools may run without confirmation when they are scoped to the current workspace.
- Initial managed-chat draft-staging tools may update the ordinary editor working copy, but they must not save canonical YAML.
- The user saves or discards AI-staged changes through the ordinary editor commit, undo, revert, and dirty-state UI.
- Future tools that save schemas, write generation metadata, create persistent merged generations, delete generations, duplicate generations, or export files require explicit user confirmation if they are later added.
- The service must not convert AI-staged editor changes into a canonical write.

## Local Agent Provider Rules

Codex CLI and Foundation Models `fm` are local command-backed integrations, but they are not the same kind of runtime. Codex CLI is a delegated external agent. Foundation Models `fm serve` is an API-level local provider for the managed chat agent, while direct `fm` command use may be a managed command adapter only after its request and tool-call behavior are verified.

- Codex CLI is intended for heavier repository-aware tasks such as spec edits, code changes, review, and multi-step diagnosis.
- Foundation Models `fm` is intended for local macOS model use where low external-data exposure and local tool-capable inference are desirable.
- Foundation Models `fm` may run either as a direct command adapter or through `fm serve` as a local Chat Completions API bridge.
- The AI panel may expose both providers side by side in the provider selector when the host health check confirms availability.
- Local command providers must not require an OpenAI-compatible API endpoint.
- Codex CLI must not be treated as a drop-in replacement for OpenAI-compatible chat completions. Its internal session and compaction behavior are delegated to Codex.
- Local command providers may still consume the same [Agent tool contract](agent-tool-contract.md) definitions after the adapter maps the host tool schema into the provider's supported format.
- If a command provider has native tool calling, host tools should be exposed through that mechanism.
- If a command provider lacks a stable native tool-call schema, only read-only explanation and structured proposal workflows are enabled until an adapter-specific parser is implemented and verified.
- The service must clearly indicate when a provider is unavailable because the command is missing, the macOS version lacks Foundation Models support, Codex is not authenticated, or the provider cannot support the requested tool workflow.
- The service must not assume `fm` tool-call support from command availability alone; it must be proven by adapter capability tests or disabled for execution tools.

## Default Provider Rules

- If AI configuration has no explicit active profile and the host is macOS with `fm` available, the service should synthesize or select the `apple-fm-serve` profile by default.
- The automatic `apple-fm-serve` default must be treated as a local provider that requires no API key.
- The automatic `apple-fm-serve` default should use a verified loopback `fm serve` endpoint. Host-managed UDS bridge mode is deferred.
- A user-selected active profile always overrides automatic `fm` defaulting.
- If `fm serve` is not reachable, provider health-check APIs report that the user-managed server must be started or reconfigured.

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

## Reads

- AI provider configuration.
- AI assistant session metadata and prompt context.
- User prompt and assistant conversation history.
- Current frontend context returned by frontend tools through the AI panel event stream.
- Schemas, records, compact generation summaries, and validation diagnostics through application services.

## Writes

- Assistant session metadata, prompt context, run summaries, safe tool-card data, and compaction records when retained by the host.
- AI-staged editor working-copy changes shown to the user.
- No canonical YAML writes from the initial managed-chat tool set.

## Related Requirements

- [Shared web editing frontend](shared-web-editing-frontend.md)
- [HTML editor plugin runtime](html-editor-plugin-runtime.md)
- [Product overview](../generic/product-overview.md)

## Native-Language Summary

アプリ内AIパネルからの指示を受け、自前エージェントではOpenAI互換API、Ollama、LM Studio、`fm serve` などのAPIレベルproviderへ接続し、MasterDataMate側でセッション、コンテキスト、コンパクト化、ツール実行を管理する。Codex CLIのような既存エージェントは別ランタイムとして明示的に起動し、内部セッション管理やコンパクト化はCodexへ委譲する。初期の自前エージェントツールは正本YAMLを書かず、AIが作ったテーブル変更案を通常エディタの作業コピーへ反映するだけにする。赤い差分表示がpreviewになり、保存は人間が既存commitボタンで行う。
