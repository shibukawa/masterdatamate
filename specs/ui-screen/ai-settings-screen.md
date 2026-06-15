---
id: "ai-settings-screen"
type: "ui-screen"
title: "AI settings screen"
aliases: ["AI configuration screen", "LLM settings screen"]
tags: ["ui", "ai", "settings", "credentials"]
facts:
  lifecycle.status: "blueprint"
---

# AI settings screen

## Summary

The AI settings screen lets ordinary users configure the AI backend used by the in-app assistant. Users can choose a backend profile, configure provider endpoints and models, enter API keys when required, run health checks, and enable or disable AI features without editing environment variables.

API keys are entered in the browser only during the save interaction. After save, the backend stores them through [AI secret storage service](../component/ai-secret-storage-service.md), and subsequent loads return only whether a key exists. Stored credentials are displayed as a masked placeholder such as `********`.

When the host runs on macOS and a local `fm serve` endpoint is reachable, the default configuration selects local Apple Foundation Models even when the user has not configured any hosted provider.

## User Goals

- Enable or disable AI assistant features.
- Choose the active AI backend from available profiles.
- Use an already-running local `fm serve` endpoint on macOS when available.
- Configure OpenAI-compatible endpoints, Ollama, LM Studio, Codex CLI, and Apple Foundation Models profiles.
- Enter or replace hosted-provider API keys without editing environment variables.
- Confirm that an API key is saved without seeing the raw key again.
- Clear a saved API key after confirmation.
- Run a provider health check and see actionable diagnostics.
- Return to table editing without losing the previous editing context.

## States

- AI disabled.
- AI enabled with default local `fm serve` profile.
- AI enabled with user-selected provider profile.
- No provider available.
- Provider available but not healthy.
- Hosted provider missing credential.
- Hosted provider credential saved.
- Credential edit in progress.
- Credential removal confirmation open.
- Health check running.
- Health check passed with streaming, structured output, and tool-call capability summary.
- Health check failed with endpoint, command, credential, model, or capability diagnostics.
- Dirty settings changes.
- Save ready.
- Save blocked by invalid endpoint, duplicate profile ID, unsupported provider type, or missing required model.

## Layout

- The route is `/settings/ai`.
- The screen uses focused settings chrome similar to schema and generation editing screens.
- The top bar includes a return-to-table control, `Save`, `Revert`, and `Test connection` commands.
- The main content is a compact settings form, not an `extable` grid.
- A provider selector appears near the top and controls the active profile.
- Profile details appear below the selector with fields appropriate to the selected provider type.
- Credential fields use password-style inputs and show `********` when a credential is already stored.
- The screen should show a read-only status area for provider health, detected local commands, credential presence, and capability flags.
- The screen must avoid exposing raw API keys in summaries, diagnostics, browser dev-visible shared state, or assistant context.

## Fields

| Field | Applies To | Notes |
| --- | --- | --- |
| AI enabled | all | Turns AI assistant features on or off. |
| Active profile | all | Selects the provider used by the AI assistant by default. |
| Provider type | all | `openai`, `openai_compatible`, `ollama`, `lmstudio`, `codex_cli`, or `foundation_models_cli`. |
| Display name | all | Human-facing label in settings and assistant provider selector. |
| Base URL | API-backed providers | Endpoint for OpenAI-compatible providers, Ollama, LM Studio, or `fm serve`. |
| Model | providers with model selection | Defaults to `system` for `fm serve` when not specified. |
| API key | hosted providers | New value is write-only; stored value is shown as `********`. |
| Command | command-backed providers | Command name or approved path such as `codex` or `fm`. |
| Timeout | all | Provider request timeout. |
| Streaming | all | Capability flag, verified by health check. |
| Tool calling | all | Capability flag, verified by health check before execution tools are enabled. |

## Default Provider Resolution

- On macOS, if the default local `fm serve` endpoint is reachable and exposes an available `system` model, the first-run default active profile is `apple-fm-serve`.
- The default `apple-fm-serve` profile uses `provider_type: openai_compatible`, `base_url: http://127.0.0.1:1976`, and `model: system`.
- If `fm serve` is not already reachable, the screen shows setup diagnostics and does not ask the host to start it in the initial implementation.
- If `fm` is unavailable, the AI settings screen opens with AI disabled or with no active profile and a clear setup diagnostic.
- User selection overrides automatic default resolution.
- Once the user explicitly selects a profile, later automatic `fm` detection must not silently replace it.

## Credential Behavior

- API key entry is optional for local providers that do not require credentials.
- Hosted providers requiring a key show a credential field with saved/unsaved status.
- When no key is stored, the credential field is empty.
- When a key is stored, the credential field displays `********` and the API response includes only credential presence metadata.
- Leaving the masked placeholder unchanged preserves the stored credential.
- Typing a new value replaces the stored credential only after save.
- Clearing a stored credential requires explicit confirmation before deletion.
- The browser must not persist API keys in local storage, session storage, IndexedDB, URL parameters, CopilotKit state, or AG-UI events.
- The settings form may hold a newly typed API key in component memory only until the save or cancel interaction completes.

## Invoked APIs

- Load AI settings and provider profiles.
- Save AI settings and provider profile metadata.
- Save or replace provider credential.
- Delete provider credential.
- Run provider health check.
- Detect local provider availability for `fm`, Ollama, LM Studio, and Codex CLI when applicable.

## Components

- [AI provider configuration model](../data-model/ai-provider-configuration-model.md)
- [AI secret storage service](../component/ai-secret-storage-service.md)
- [AI assistant service](../component/ai-assistant-service.md)
- [Single page application shell](../ui-flow/single-page-application-shell.md)
- [Shared web editing frontend](../component/shared-web-editing-frontend.md)

## Related Requirements

- [In-app AI assistant panel](in-app-ai-assistant-panel.md)
- [Web service host](../server-component/web-service-host.md)
- [Wails desktop host](../server-component/wails-desktop-host.md)

## Native-Language Summary

AI設定画面では、利用するAIバックエンド、エンドポイント、モデル、APIキーを一般ユーザー向けに設定できる。APIキーは保存時だけブラウザが扱い、バックエンドがKeychainやWinCredなどへ保存する。再表示時はキー本文を返さず、保存済み状態だけを返して入力欄には `********` を表示する。macOSでローカル `fm serve` endpoint が起動済みで利用できる場合は、未設定でもローカルFoundation Modelsをデフォルト選択する。
