---
id: "in-app-ai-assistant-panel"
type: "ui-screen"
title: "In-app AI assistant panel"
aliases: ["AI panel", "LLM assistant panel", "agent panel"]
tags: ["ai", "llm", "assistant", "frontend", "ag-ui", "copilotkit"]
facts:
  lifecycle.status: "blueprint"
---

# In-app AI assistant panel

## Summary

The in-app AI assistant panel lets users issue natural-language instructions while editing master data. It provides conversational guidance, table-aware explanation, validation help, and AI-staged record changes in the ordinary editor working copy.

The panel is an assistant surface, not a replacement for the ordinary table, schema, generation, or export screens. The initial managed-chat tool surface focuses on table understanding, validation explanation, and staging table record edits for human review in the existing editor.

The production UI uses CopilotKit with a floating `CopilotPopup`-style entry so the assistant is always reachable from a floating action button (FAB). The FAB opens and closes the assistant without navigating away from the current editing surface.

## User Goals

- Ask questions about the selected table, schema, generation, records, and validation diagnostics.
- Request suggested fixes for validation errors.
- Ask for record completion, sample data, or consistency checks.
- Ask for bounded explanations about the active table, visible generation context, selected records, and validation diagnostics.
- Review proposed changes before applying them.
- Continue using ordinary editing screens as the authoritative repair and inspection surface.

## Layout

- The primary entry is a floating assistant FAB implemented with CopilotKit's popup-style chat surface.
- The FAB remains visible on table editing, schema editing, generation editing, export, and plugin-backed screens unless a modal confirmation or full-screen blocking operation is active, but tool-backed writes are initially limited to table record changes.
- The FAB must have an accessible name such as `Open AI assistant` and a tooltip such as `Ask AI`.
- The opened assistant uses a popup or drawer-sized chat surface depending on viewport width.
- The panel includes a message timeline, text input, run/cancel controls, runtime selector, provider/profile indicator, and current context indicator.
- The panel includes a session selector with create, switch, rename, archive, delete, and compact actions for [AI assistant session model](../data-model/ai-assistant-session-model.md) sessions.
- The panel can show structured cards for tool results, validation diagnostics, and AI-staged table changes.
- On narrow viewports, the panel may occupy a full-screen overlay route but must preserve dirty-state guards from the current editing surface.
- The panel must not obscure required save, revert, validation, generation selection, schema settings, or navigation controls for the active editing surface.
- The FAB position must avoid the main grid scrollbars, row commit controls, and bottom-right toast notifications.

## States

- Disabled: AI features are disabled or no provider profile is configured.
- Provider error: the active provider is unreachable, missing credentials, or missing required capability.
- Runtime unavailable: the selected managed provider or delegated external agent, such as Codex CLI or Foundation Models `fm`, is not installed, not authenticated, unsupported on the host OS, or failed its health check.
- Ready: the panel can accept text instructions.
- Running: an assistant run is streaming text, tool calls, or state updates.
- Staging draft: the assistant is applying validated table operations to the editor working copy.
- Showing staged changes: the ordinary editor marks AI-staged changes as dirty/red, and the panel displays a summary plus diagnostics.
- Cancelled: the user stopped the current run before completion.

## Browser Context Tools

The panel does not send a full UI snapshot with every assistant request. Browser-owned editing state is exposed through frontend tools that the assistant calls when needed. The initial run request contains the user instruction, selected session/runtime/profile, and minimal run metadata.

`get_current_context` returns scoped context rather than full project state:

- active page and route.
- selected table and selected records when available.
- active generation and display mode.
- visible validation diagnostics.
- current schema summary.
- visible table page, selected rows, and pagination metadata when table data is included.
- pending local edits only when the user explicitly includes them or the current operation requires them.
- user instruction text and relevant assistant conversation history.

Conversation history comes from the selected server-side assistant session. Switching sessions changes the loaded timeline and future prompt context for managed chat sessions, but does not change the current table, generation, schema, or unsaved editor state. Delegated external agent sessions show UI-visible task records while the external agent keeps its own internal context.

Frontend tool execution rules:

- The panel executes browser-owned tools against the current mounted editor state, not against server-side cached state.
- If the active screen does not have the requested table editor mounted, the frontend tool returns `blocked` instead of fabricating state.
- If the user navigates or the editor context changes while a run is waiting for a frontend tool result, the frontend tool returns the current state with its route/table/generation identifiers so the model and service can detect mismatch.
- Pending editor values are returned only for the active editor working copy and only in bounded form.
- Frontend tool results may be shown in the assistant activity timeline so users understand what state the assistant inspected.

## Session Management

- Opening the panel selects the most recently updated active session for the workspace, or creates an empty session when none exists.
- Users can create a new session without navigating away from the current editing surface.
- Users can switch sessions from the panel. If an assistant run is currently streaming, the panel asks the user to cancel or wait before switching.
- Users can rename the current session title.
- Users can archive sessions to remove them from the default active list while keeping them available from an archived filter.
- Users can delete a session after confirmation. Deleting a session closes the panel timeline if the deleted session was selected.
- Users can manually compact the current session. The timeline should show a compacted-history marker rather than pretending all original details are still verbatim.
- Manual and automatic compaction are available only for managed chat sessions. Delegated external agent sessions hide compaction controls unless the external agent exposes an equivalent operation.
- Automatic compaction markers from the service are shown inline or near the affected older messages.
- Session operations must not bypass ordinary dirty-state guards for table, schema, generation, export, or plugin editing surfaces.

The assistant service may enrich this browser context with backend tool reads, but broad project reads should be deliberate and visible through tool events.

## Invoked APIs

- AI provider profile list and health check APIs from [AI assistant service](../component/ai-assistant-service.md).
- AI assistant session list, create, load, update, delete, and compact APIs from [AI assistant service](../component/ai-assistant-service.md).
- AI assistant run or AG-UI-compatible event stream API.
- Agent tools from [Agent tool contract](../component/agent-tool-contract.md).
- Existing frontend editor state APIs for `get_current_context` and draft staging. Canonical save and validation display still use the ordinary table editor flow.

## Components

- [Shared web editing frontend](../component/shared-web-editing-frontend.md)
- [AI assistant service](../component/ai-assistant-service.md)
- [AI assistant session model](../data-model/ai-assistant-session-model.md)
- [Agent tool contract](../component/agent-tool-contract.md)
- [AI provider configuration model](../data-model/ai-provider-configuration-model.md)

## CopilotKit Integration

- The shared frontend should wrap the application shell in CopilotKit's provider and point it at the host's Copilot Runtime-compatible endpoint.
- The default UI surface is CopilotKit's popup chat component, because it provides a floating chat bubble that can toggle open and closed alongside the product UI.
- The host registers the MasterDataMate assistant as the default agent so the popup can open without requiring per-screen agent wiring.
- The implementation should import CopilotKit's default styles once at the application root, then override only the slots needed to match MasterDataMate's compact tool-oriented UI.
- CopilotKit frontend tools should expose scoped UI context on demand: active route, active generation, selected table, selected record keys, visible diagnostics, visible rows, and editor dirty state.
- CopilotKit frontend tools may be used for browser-only actions such as focusing a table, switching views, or highlighting proposed rows.
- CopilotKit server tools map to [Agent tool contract](../component/agent-tool-contract.md) backend tools and must preserve validation and no-canonical-write rules for the initial managed-chat surface.
- Direct frontend-to-agent connections are not used for production. The Copilot Runtime-compatible backend remains responsible for authentication, provider selection, tool registration, and AG-UI event streaming.

## Provider Selection

- The panel should first distinguish runtime mode: `Managed chat` for MasterDataMate-owned sessions and API-level providers, or `External agent` for delegated runtimes such as Codex CLI.
- Managed chat provider choices include configured profiles such as OpenAI-compatible API, Ollama, LM Studio, and Apple Foundation Models `fm serve`.
- External agent choices include Codex CLI when available and authenticated.
- Provider availability is determined by the host health check, not by configuration presence alone.
- API-backed profiles show endpoint, model, and credential diagnostics.
- External agent profiles show command availability, version or capability diagnostics when available, sandbox mode, and whether workspace writes are allowed.
- Codex CLI profiles should be labeled as heavier delegated agent execution, suitable for repository-aware analysis and changes. The UI should not imply that Codex shares the managed chat session memory or compaction pipeline.
- Foundation Models `fm` profiles should be labeled as local macOS model execution, with capability status for structured output and tool calling.

## AG-UI Integration

AG-UI is the preferred protocol for the production AI panel when the assistant needs streaming, tool-call progress, state updates, and editor draft-staging events. The implementation may start with a smaller internal event stream if it preserves the same event concepts and can be mapped to AG-UI later.

AG-UI is used for user-agent interaction only. MCP remains the future external tool/data protocol for agents outside the application, and A2A remains out of scope unless a later multi-agent workflow requires it.

The AI panel should treat AG-UI events as UI state transitions. It should not let arbitrary generated UI replace host-owned confirmation, validation, save, schema, generation, or export controls.

CopilotKit is the preferred React UI integration for AG-UI in the shared frontend. CopilotKit components consume the host event stream, while MasterDataMate retains ownership of domain tools, editor dirty-state, ordinary commit controls, and persistence.

## Staged Change UI

- AI-staged record changes are applied to the same editor working copy as manual edits.
- AI-staged record changes are delivered to the browser through a frontend draft-staging tool. The server does not update table rows directly and does not use a cached copy of frontend state to apply them.
- The existing changed-cell or changed-row styling is the primary preview surface.
- The panel summarizes affected table, operation type, primary key, changed fields, and validation diagnostics.
- The editor does not need to persist or visually distinguish AI-authored changes from human-authored unsaved changes after staging.
- Delete staging is represented by primary key only. Move staging is not available in the initial AI tool surface.
- Partial staging may occur when some operations are accepted by the editor and others are rejected; rejected operations are shown with ordinary editor diagnostics or panel summary errors.
- The user saves AI-staged changes with the ordinary editor commit button, or discards them with the ordinary undo/revert controls.
- AI tools must not call the canonical table commit API in the initial managed-chat surface.

## Related Requirements

- [AI assistant service](../component/ai-assistant-service.md)
- [AI assistant session model](../data-model/ai-assistant-session-model.md)
- [Agent tool contract](../component/agent-tool-contract.md)
- [AI provider configuration model](../data-model/ai-provider-configuration-model.md)
- [Table editing workspace](table-editing-workspace.md)
- [Generation editing screen](generation-editing-screen.md)
- [Schema editing screen](schema-editing-screen.md)

## Native-Language Summary

アプリ内AIパネルは、自然言語で「検証エラーを説明して」「このテーブルの問題点を見て」「このレコード案を作って」と依頼できる対話面。React UIはCopilotKitの `CopilotPopup` 相当のFABとして常時呼び出せるようにし、複数チャットセッションの作成・切り替え・リネーム・アーカイブ・削除・コンパクト化を提供する。ユーザーはMasterDataMateが管理する自前チャットエージェントと、Codex CLIなどの外部エージェントを明示的に切り替える。自前エージェントではOpenAI互換API、Ollama、LM Studio、`fm serve` などを使い、セッションとコンパクト化をMasterDataMateが管理する。現在の表・世代・選択行・dirty状態などは毎回リクエストへ常時混ぜず、AIが `get_current_context` frontend tool を呼んだ時にブラウザが返す。初期ツールはテーブル取得、検証、AIが作った変更案のエディタ作業コピー反映に絞る。変更反映もfrontend toolとしてブラウザの通常エディタ作業コピーに適用され、赤い差分表示がpreviewになり、保存は人間が既存commitボタンで行う。
