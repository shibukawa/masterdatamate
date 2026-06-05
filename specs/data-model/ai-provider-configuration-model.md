---
id: "ai-provider-configuration-model"
type: "data-model"
title: "AI provider configuration model"
aliases: ["LLM provider configuration", "AI settings model"]
tags: ["ai", "llm", "configuration", "provider"]
facts:
  lifecycle.status: "blueprint"
  data.name: "ai-provider-configuration"
---

# AI provider configuration model

## Summary

The AI provider configuration model defines how MasterDataMate connects its in-app AI assistant and agent tools to hosted or local LLM backends. It supports OpenAI-compatible chat APIs as the primary transport shape and allows local providers such as Ollama and LM Studio through provider-specific defaults.

Configuration selects one active provider profile for interactive use, while keeping provider credentials and local endpoint details outside canonical master data exports.

## Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| enabled | boolean | yes | Enables or disables AI features for the workspace or host runtime. |
| active_profile | string | no | Profile ID used by the in-app AI assistant when the user has not selected a profile explicitly. |
| profiles | array | yes | Ordered provider profiles available to the host. Empty is allowed when `enabled` is false. |
| profiles.id | string | yes | Stable ASCII identifier unique within the configuration. |
| profiles.display_name | string | yes | Human-facing provider name shown in AI settings. |
| profiles.provider_type | enum | yes | `openai_compatible`, `openai`, `ollama`, or `lmstudio`. |
| profiles.base_url | URL | no | Endpoint base URL. Required for local or custom OpenAI-compatible providers. |
| profiles.api_key_env | string | no | Environment variable name that contains the API key. Preferred for hosted providers. |
| profiles.api_key_ref | string | no | Host-specific secret reference used by packaged or desktop builds when environment variables are not practical. |
| profiles.model | string | yes | Model identifier sent to the provider. |
| profiles.temperature | number | no | Default sampling temperature for assistant turns. |
| profiles.max_output_tokens | integer | no | Host-level response budget for assistant turns. |
| profiles.request_timeout_ms | integer | no | Request timeout for provider calls. |
| profiles.supports_streaming | boolean | no | Whether the profile may stream assistant output. Defaults from provider type when omitted. |
| profiles.supports_tool_calls | boolean | no | Whether the profile may receive structured tool definitions and emit tool calls. Defaults from provider type when omitted. |
| profiles.local_network_allowed | boolean | no | Whether the host may connect to non-loopback local network endpoints for this profile. |
| profiles.headers | object | no | Optional static headers for custom OpenAI-compatible providers. Must not contain raw secrets. |
| profiles.notes | string | no | Maintainer-facing explanation of the profile. |

## Provider Types

`openai` is a convenience provider type for the official OpenAI API and uses OpenAI-compatible chat behavior with OpenAI defaults.

`openai_compatible` targets providers that expose an OpenAI-compatible chat or responses-style API. The host must not assume full feature parity beyond the capabilities declared by the profile.

`ollama` targets a local Ollama server. Its default `base_url` is `http://127.0.0.1:11434` when omitted. Ollama profiles may map to OpenAI-compatible endpoints when available, but the host should tolerate model lists and tool-call support differing by installed model.

`lmstudio` targets a local LM Studio server. Its default `base_url` is `http://127.0.0.1:1234` when omitted. LM Studio profiles are treated as OpenAI-compatible only for the features reported or configured on the profile.

## Rules / Constraints

- AI provider configuration is application configuration, not exported master data.
- Canonical table records, schemas, generation configs, and export settings remain the source of truth for project data.
- Raw API keys must not be written into project-local YAML configuration.
- Hosted provider credentials should be read from environment variables or host-specific secret storage.
- Local provider URLs default to loopback addresses. Non-loopback local network URLs require `local_network_allowed: true`.
- The host must display a clear diagnostic when an active profile is missing, disabled, unreachable, or lacks a required capability.
- Tool-calling features must check `supports_tool_calls` before sending tool definitions to a provider.
- Streaming UI must check `supports_streaming`; non-streaming providers may still be used through buffered assistant responses.
- Provider capability declarations are advisory until verified by a provider health check.
- The host should expose a provider health check that validates endpoint reachability, configured model availability when practical, streaming support, and tool-call support.
- OpenAI-compatible responses must be normalized into the same internal assistant message, tool-call, and token usage shape before the frontend sees them.
- Provider-specific errors must be translated into user-facing diagnostics without exposing raw secrets or full request headers.
- Local provider use should be allowed without internet access after the local provider and model are installed.

## Example

```yaml
enabled: true
active_profile: local-ollama
profiles:
  - id: openai-default
    display_name: OpenAI GPT
    provider_type: openai
    api_key_env: OPENAI_API_KEY
    model: gpt-4.1-mini
    supports_streaming: true
    supports_tool_calls: true
  - id: local-ollama
    display_name: Ollama local
    provider_type: ollama
    base_url: http://127.0.0.1:11434
    model: qwen2.5-coder:7b
    supports_streaming: true
    supports_tool_calls: false
  - id: lmstudio-local
    display_name: LM Studio local
    provider_type: lmstudio
    base_url: http://127.0.0.1:1234
    model: local-model
    supports_streaming: true
    supports_tool_calls: true
```

## Uses Common Details

- None yet.

## Reads

- Host environment variables for provider credentials.
- Host-specific secret references when configured.
- Optional project-local AI configuration file when the host supports workspace profiles.

## Writes

- AI configuration updates when the AI settings UI is implemented.
- No canonical master data records.
- No raw API keys in project-local configuration files.

## Related Requirements

- [AI assistant service](../component/ai-assistant-service.md)
- [In-app AI assistant panel](../ui-screen/in-app-ai-assistant-panel.md)
- [Agent tool contract](../component/agent-tool-contract.md)

## Native-Language Summary

OpenAI互換API、OpenAI公式API、Ollama、LM Studioを同じAI設定モデルで扱う。APIキーは環境変数やホスト側の秘密情報として扱い、プロジェクトの正本データやエクスポート対象には含めない。ローカルLLMはループバック接続を基本とし、ツール呼び出しやストリーミングはプロファイルの能力として明示する。
