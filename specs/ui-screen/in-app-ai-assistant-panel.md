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

The in-app AI assistant panel lets users issue natural-language instructions while editing master data. It provides conversational guidance, tool-backed analysis, proposed record or schema changes, and confirmed execution for advanced operations such as generation analysis, persistent generation merge, import, and export.

The panel is an assistant surface, not a replacement for the ordinary table, schema, generation, or export screens. It must keep users in control of any canonical data changes.

The production UI uses CopilotKit with a floating `CopilotPopup`-style entry so the assistant is always reachable from a floating action button (FAB). The FAB opens and closes the assistant without navigating away from the current editing surface.

## User Goals

- Ask questions about the selected table, schema, generation, records, and validation diagnostics.
- Request suggested fixes for validation errors.
- Ask for record completion, sample data, or consistency checks.
- Ask for generation differences, merge impact, and export readiness.
- Request advanced operations such as generation merge or file export through guided confirmation.
- Upload or select files and ask for import mapping, record creation, or binary asset attachment proposals.
- Review proposed changes before applying them.
- Continue using ordinary editing screens as the authoritative repair and inspection surface.

## Layout

- The primary entry is a floating assistant FAB implemented with CopilotKit's popup-style chat surface.
- The FAB remains visible on table editing, schema editing, generation editing, export, and plugin-backed screens unless a modal confirmation, file picker, or full-screen blocking operation is active.
- The FAB must have an accessible name such as `Open AI assistant` and a tooltip such as `Ask AI`.
- The opened assistant uses a popup or drawer-sized chat surface depending on viewport width.
- The panel includes a message timeline, text input, run/cancel controls, provider/profile indicator, and current context indicator.
- The panel includes a session selector with create, switch, rename, archive, delete, and compact actions for [AI assistant session model](../data-model/ai-assistant-session-model.md) sessions.
- The panel can show structured cards for tool results, validation diagnostics, proposed changes, confirmation requests, and export results.
- The panel can show uploaded file cards, import mapping cards, and binary asset attachment cards.
- On narrow viewports, the panel may occupy a full-screen overlay route but must preserve dirty-state guards from the current editing surface.
- The panel must not obscure required save, revert, validation, generation selection, schema settings, or navigation controls for the active editing surface.
- The FAB position must avoid the main grid scrollbars, row commit controls, and bottom-right toast notifications.

## States

- Disabled: AI features are disabled or no provider profile is configured.
- Provider error: the active provider is unreachable, missing credentials, or missing required capability.
- Local provider unavailable: the selected command-backed provider, such as Codex CLI or Foundation Models `fm`, is not installed, not authenticated, unsupported on the host OS, or failed its health check.
- Ready: the panel can accept text instructions.
- Running: an assistant run is streaming text, tool calls, or state updates.
- Awaiting confirmation: a side-effecting tool needs user approval.
- Showing proposal: the panel displays proposed changes and validation diagnostics.
- Applying: a confirmed tool is writing canonical files or export artifacts through host services.
- Uploading: the user selected or dropped files and the host is staging them for AI/import tools.
- Cancelled: the user stopped the current run before completion.

## Context Sent To Assistant

The frontend sends scoped context rather than full project state:

- active page and route.
- selected table and selected records when available.
- active generation and display mode.
- visible validation diagnostics.
- current schema summary.
- staged upload IDs and upload summaries when the user has attached files to the assistant session.
- pending local edits only when the user explicitly includes them or the current operation requires them.
- user instruction text and relevant assistant conversation history.

Conversation history comes from the selected server-side assistant session. Switching sessions changes the loaded timeline and future prompt context, but does not change the current table, generation, schema, or unsaved editor state.

## Session Management

- Opening the panel selects the most recently updated active session for the workspace, or creates an empty session when none exists.
- Users can create a new session without navigating away from the current editing surface.
- Users can switch sessions from the panel. If an assistant run is currently streaming, the panel asks the user to cancel or wait before switching.
- Users can rename the current session title.
- Users can archive sessions to remove them from the default active list while keeping them available from an archived filter.
- Users can delete a session after confirmation. Deleting a session removes its uncommitted staged uploads and closes the panel timeline if the deleted session was selected.
- Users can manually compact the current session. The timeline should show a compacted-history marker rather than pretending all original details are still verbatim.
- Automatic compaction markers from the service are shown inline or near the affected older messages.
- Session operations must not bypass ordinary dirty-state guards for table, schema, generation, export, or plugin editing surfaces.

The assistant service may enrich this context with tool reads, but broad project reads should be deliberate and visible through tool events.

## Invoked APIs

- AI provider profile list and health check APIs from [AI assistant service](../component/ai-assistant-service.md).
- AI assistant session list, create, load, update, delete, and compact APIs from [AI assistant service](../component/ai-assistant-service.md).
- AI assistant run or AG-UI-compatible event stream API.
- Agent tools from [Agent tool contract](../component/agent-tool-contract.md).
- Upload staging, upload inspection, and binary asset APIs when file import or asset attachment is requested.
- Existing table commit, schema save, generation, validation, and export APIs when confirmed tool execution requires them.

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
- CopilotKit shared state or agent context should expose scoped UI context: active route, active generation, selected table, selected record keys, visible diagnostics, and staged upload summaries.
- CopilotKit frontend tools may be used for browser-only actions such as focusing a table, opening a file picker, switching views, or highlighting proposed rows.
- CopilotKit server tools map to [Agent tool contract](../component/agent-tool-contract.md) backend tools and must preserve confirmation and validation rules.
- Direct frontend-to-agent connections are not used for production. The Copilot Runtime-compatible backend remains responsible for authentication, provider selection, tool registration, and AG-UI event streaming.

## Provider Selection

- The panel should let the user select among configured profiles such as OpenAI-compatible API, Ollama, LM Studio, Codex CLI, and Apple Foundation Models `fm`.
- Provider availability is determined by the host health check, not by configuration presence alone.
- API-backed profiles show endpoint, model, and credential diagnostics.
- Command-backed profiles show command availability, version or capability diagnostics when available, sandbox mode, and whether tool calling is enabled.
- Codex CLI profiles should be labeled as heavier local agent execution, suitable for repository-aware analysis and changes.
- Foundation Models `fm` profiles should be labeled as local macOS model execution, with capability status for structured output and tool calling.

## AG-UI Integration

AG-UI is the preferred protocol for the production AI panel when the assistant needs streaming, tool-call progress, state updates, and human-in-the-loop confirmation. The implementation may start with a smaller internal event stream if it preserves the same event concepts and can be mapped to AG-UI later.

AG-UI is used for user-agent interaction only. MCP remains the future external tool/data protocol for agents outside the application, and A2A remains out of scope unless a later multi-agent workflow requires it.

The AI panel should treat AG-UI events as UI state transitions. It should not let arbitrary generated UI replace host-owned confirmation, validation, save, or export controls.

CopilotKit is the preferred React UI integration for AG-UI in the shared frontend. CopilotKit components consume the host event stream, while MasterDataMate retains ownership of domain tools, previews, confirmations, and persistence.

## Confirmation And Proposal UI

- Proposed record changes are shown as affected table, operation type, primary key, changed fields, and validation diagnostics.
- Proposed schema changes are shown with renamed fields, type changes, primary key changes, and affected data cleanup warnings.
- Generation merge previews show source generations, destination metadata, record counts, diagnostics, and overwrite or collision risks.
- Export previews show selected generations, backend, destination path, overwrite behavior, and export-blocking diagnostics.
- Import previews show upload IDs, detected file type, target table, proposed field mapping, affected primary keys, parse diagnostics, and binary asset destinations when applicable.
- The user must explicitly approve side-effecting actions from the panel before the host executes them.
- Approval creates a confirmation token scoped to the exact action preview. Editing the action invalidates the token.

## Related Requirements

- [AI assistant service](../component/ai-assistant-service.md)
- [AI assistant session model](../data-model/ai-assistant-session-model.md)
- [Agent tool contract](../component/agent-tool-contract.md)
- [AI provider configuration model](../data-model/ai-provider-configuration-model.md)
- [Table editing workspace](table-editing-workspace.md)
- [Generation editing screen](generation-editing-screen.md)
- [Schema editing screen](schema-editing-screen.md)

## Native-Language Summary

アプリ内AIパネルは、自然言語で「検証エラーを説明して」「この世代差分を見て」「マージして」「エクスポートして」「このCSVを取り込んで」と依頼できる対話面。React UIはCopilotKitの `CopilotPopup` 相当のFABとして常時呼び出せるようにし、複数チャットセッションの作成・切り替え・リネーム・アーカイブ・削除・コンパクト化を提供する。AG-UIはストリーミング、ツール実行状況、確認待ちを扱うためのUIプロトコルとして使う。OpenAI互換API、Ollama、LM Studio、Codex CLI、macOSの `fm` Foundation Modelsコマンドをプロファイルとして選べる。LLMの提案や高度な操作は、必ずホスト定義ツール、検証、ユーザー承認を通して実行する。
