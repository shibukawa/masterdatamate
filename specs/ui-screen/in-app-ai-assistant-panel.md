---
id: "in-app-ai-assistant-panel"
type: "ui-screen"
title: "In-app AI assistant panel"
aliases: ["AI panel", "LLM assistant panel", "agent panel"]
tags: ["ai", "llm", "assistant", "frontend", "ag-ui"]
facts:
  lifecycle.status: "blueprint"
---

# In-app AI assistant panel

## Summary

The in-app AI assistant panel lets users issue natural-language instructions while editing master data. It provides conversational guidance, tool-backed analysis, proposed record or schema changes, and confirmed execution for advanced operations such as generation analysis, persistent generation merge, and export.

The panel is an assistant surface, not a replacement for the ordinary table, schema, generation, or export screens. It must keep users in control of any canonical data changes.

## User Goals

- Ask questions about the selected table, schema, generation, records, and validation diagnostics.
- Request suggested fixes for validation errors.
- Ask for record completion, sample data, or consistency checks.
- Ask for generation differences, merge impact, and export readiness.
- Request advanced operations such as generation merge or file export through guided confirmation.
- Review proposed changes before applying them.
- Continue using ordinary editing screens as the authoritative repair and inspection surface.

## Layout

- The AI panel may be a right-side dock, bottom drawer, or focused SPA panel depending on available viewport width.
- The panel includes a message timeline, text input, run/cancel controls, provider/profile indicator, and current context indicator.
- The panel can show structured cards for tool results, validation diagnostics, proposed changes, confirmation requests, and export results.
- On narrow viewports, the panel may occupy a full-screen overlay route but must preserve dirty-state guards from the current editing surface.
- The panel must not obscure required save, revert, validation, or navigation controls for the active editing surface.

## States

- Disabled: AI features are disabled or no provider profile is configured.
- Provider error: the active provider is unreachable, missing credentials, or missing required capability.
- Ready: the panel can accept text instructions.
- Running: an assistant run is streaming text, tool calls, or state updates.
- Awaiting confirmation: a side-effecting tool needs user approval.
- Showing proposal: the panel displays proposed changes and validation diagnostics.
- Applying: a confirmed tool is writing canonical files or export artifacts through host services.
- Cancelled: the user stopped the current run before completion.

## Context Sent To Assistant

The frontend sends scoped context rather than full project state:

- active page and route.
- selected table and selected records when available.
- active generation and display mode.
- visible validation diagnostics.
- current schema summary.
- pending local edits only when the user explicitly includes them or the current operation requires them.
- user instruction text and relevant assistant conversation history.

The assistant service may enrich this context with tool reads, but broad project reads should be deliberate and visible through tool events.

## Invoked APIs

- AI provider profile list and health check APIs from [AI assistant service](../component/ai-assistant-service.md).
- AI assistant run or AG-UI-compatible event stream API.
- Agent tools from [Agent tool contract](../component/agent-tool-contract.md).
- Existing table commit, schema save, generation, validation, and export APIs when confirmed tool execution requires them.

## Components

- [Shared web editing frontend](../component/shared-web-editing-frontend.md)
- [AI assistant service](../component/ai-assistant-service.md)
- [Agent tool contract](../component/agent-tool-contract.md)
- [AI provider configuration model](../data-model/ai-provider-configuration-model.md)

## AG-UI Integration

AG-UI is the preferred protocol for the production AI panel when the assistant needs streaming, tool-call progress, state updates, and human-in-the-loop confirmation. The implementation may start with a smaller internal event stream if it preserves the same event concepts and can be mapped to AG-UI later.

AG-UI is used for user-agent interaction only. MCP remains the future external tool/data protocol for agents outside the application, and A2A remains out of scope unless a later multi-agent workflow requires it.

The AI panel should treat AG-UI events as UI state transitions. It should not let arbitrary generated UI replace host-owned confirmation, validation, save, or export controls.

## Confirmation And Proposal UI

- Proposed record changes are shown as affected table, operation type, primary key, changed fields, and validation diagnostics.
- Proposed schema changes are shown with renamed fields, type changes, primary key changes, and affected data cleanup warnings.
- Generation merge previews show source generations, destination metadata, record counts, diagnostics, and overwrite or collision risks.
- Export previews show selected generations, backend, destination path, overwrite behavior, and export-blocking diagnostics.
- The user must explicitly approve side-effecting actions from the panel before the host executes them.
- Approval creates a confirmation token scoped to the exact action preview. Editing the action invalidates the token.

## Related Requirements

- [AI assistant service](../component/ai-assistant-service.md)
- [Agent tool contract](../component/agent-tool-contract.md)
- [AI provider configuration model](../data-model/ai-provider-configuration-model.md)
- [Table editing workspace](table-editing-workspace.md)
- [Generation editing screen](generation-editing-screen.md)
- [Schema editing screen](schema-editing-screen.md)

## Native-Language Summary

アプリ内AIパネルは、自然言語で「検証エラーを説明して」「この世代差分を見て」「マージして」「エクスポートして」と依頼できる対話面。AG-UIはストリーミング、ツール実行状況、確認待ちを扱うためのUIプロトコルとして使う。LLMの提案や高度な操作は、必ずホスト定義ツール、検証、ユーザー承認を通して実行する。
