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

The AI provider configuration model defines how MasterDataMate connects its in-app AI assistant and agent tools to hosted or local LLM backends. It supports OpenAI-compatible chat APIs as the primary transport shape, local providers such as Ollama and LM Studio through provider-specific defaults, and local command-backed agent providers such as Codex CLI and Apple's `fm` Foundation Models command.

Configuration selects one active provider profile for interactive use, while keeping provider credentials and local endpoint details outside canonical master data exports.

## Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| enabled | boolean | yes | Enables or disables AI features for the workspace or host runtime. |
| active_profile | string | no | Profile ID used by the in-app AI assistant when the user has not selected a profile explicitly. |
| profiles | array | yes | Ordered provider profiles available to the host. Empty is allowed when `enabled` is false. |
| profiles.id | string | yes | Stable ASCII identifier unique within the configuration. |
| profiles.display_name | string | yes | Human-facing provider name shown in AI settings. |
| profiles.provider_type | enum | yes | `openai_compatible`, `openai`, `ollama`, `lmstudio`, `codex_cli`, or `foundation_models_cli`. |
| profiles.base_url | URL | no | Endpoint base URL. Required for local or custom OpenAI-compatible providers. |
| profiles.api_key_env | string | no | Environment variable name that contains the API key. Preferred for hosted providers. |
| profiles.api_key_ref | string | no | Host-specific secret reference used by backend credential storage. |
| profiles.has_api_key | boolean | no | Response-only UI metadata indicating whether a credential is stored. Never persisted as the credential itself. |
| profiles.requires_api_key | boolean | no | Whether the selected provider needs a credential before health checks or assistant runs can succeed. |
| profiles.model | string | no | Model identifier sent to the provider when the provider exposes model selection. Required for API-backed profiles unless the provider has a host default. |
| profiles.command | string | no | Local command name or absolute command path for command-backed providers. |
| profiles.args | array | no | Static arguments passed to a command-backed provider before the host-supplied prompt or request payload. |
| profiles.transport | enum | no | `http` or `unix_socket` for API-backed profiles. Defaults to `http` when `base_url` is used. |
| profiles.unix_socket_path | string | no | Host-local Unix domain socket path for API-backed local providers such as managed `fm serve`. Never exposed to the browser as a selectable filesystem path. |
| profiles.managed_process | boolean | no | Whether the host may start, stop, and health-check a local provider process for this profile. Defaults to false unless the profile is a built-in local provider. |
| profiles.managed_process_command | string | no | Host-controlled command used when `managed_process` is true. For `fm serve`, this is normally `fm`. |
| profiles.managed_process_args | array | no | Host-controlled process arguments for the managed provider. These are not arbitrary user-entered shell strings. |
| profiles.working_directory_policy | enum | no | `workspace`, `temporary`, or `none` for command-backed providers. Defaults to `workspace` when the provider must inspect project files. |
| profiles.stdin_mode | enum | no | `prompt`, `json`, or `none` for command-backed providers. |
| profiles.stdout_format | enum | no | `text`, `json`, or `jsonl` for command-backed providers. |
| profiles.sandbox | enum | no | Provider-specific sandbox hint such as `read_only`, `workspace_write`, or `none`. Applies only to command-backed providers that support sandboxing. |
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

`codex_cli` targets the local Codex CLI. Its default command is `codex` and its first supported mode should be non-interactive `codex exec --json`, optionally with `--output-schema` when the host needs structured results. Codex CLI profiles use the user's local Codex authentication, model settings, and license. They are command-backed agent providers, not low-latency chat completion providers.

`foundation_models_cli` targets Apple's local Foundation Models command on macOS, currently represented by the `fm` command profile. Its default command is `fm`. This provider type is intended for local on-device model use and should support tool-capable or structured interactions when the installed command exposes them. Local help for the installed command identifies `respond`, `chat`, `token-count`, `schema`, `serve`, `available`, and `quota-usage` commands, with `system` as the default on-device model and `pcc` as an Apple Foundation Model on Private Cloud Compute. Because the `fm` command contract is not yet treated as stable in this corpus, concrete flags, request formats, and tool-call event shapes must be verified on the target macOS version before implementation.

`foundation_models_cli` may be used in direct command mode through commands such as `fm respond`, or in API bridge mode by starting `fm serve` and treating its local Chat Completions API server as an `openai_compatible` profile. Direct command mode avoids managing a long-running server process. API bridge mode lets the host reuse the OpenAI-compatible adapter, but still requires local process lifecycle, port or Unix socket, and health-check management.

When `fm serve` supports Unix domain sockets on the target macOS version, the preferred managed bridge shape is a Go-hosted managed process bound to a host-owned UDS under the runtime temp directory. UDS avoids consuming a TCP port, avoids accidental exposure beyond the local host process boundary, and lets the host isolate concurrent workspaces by socket path. Loopback TCP remains available for manual setups, debugging with tools such as mitmproxy, or providers that do not support UDS.

## Command-Backed Provider Rules

Command-backed providers run a local process rather than calling an HTTP model API.

- The host must treat command-backed providers as local agent runtimes with their own latency, authentication, logging, and cancellation behavior.
- The host must resolve the command from a configured allow list or explicit user-approved path; arbitrary command strings are not allowed.
- The host must pass prompts, scoped context, and tool definitions through a deterministic stdin or argument contract.
- The host must parse stdout according to `stdout_format` and translate stderr or non-zero exit codes into provider diagnostics.
- The host must support cancellation by terminating the provider process and returning a cancelled assistant event.
- The host must not pass raw secrets or unscoped workspace data to a command-backed provider.
- The host should default command-backed providers to read-only behavior unless the provider is explicitly invoked for a confirmed write workflow.
- Codex CLI write access must use explicit sandbox settings and should start with `read_only` for analysis tasks.
- Foundation Models `fm` command access must be limited to macOS hosts where the command is installed and available.
- Foundation Models `fm` command feature detection must verify model availability through `fm available`, text generation through `fm respond`, optional token counting through `fm token-count`, structured output through `fm respond --schema` or `fm schema`, streaming through `fm respond --stream`, and tool-call behavior before enabling those capabilities in the AI panel.
- Foundation Models `fm serve` bridge mode must bind to loopback or a Unix domain socket by default. Binding to `0.0.0.0` or another non-loopback interface requires explicit user configuration.
- Foundation Models `fm serve` bridge mode must health-check the local `/health` endpoint and model list endpoint before the profile is marked ready.
- Managed `fm serve` profiles must be started by the Go host only from an allow-listed command and arguments, never from arbitrary user-supplied shell text.
- Managed `fm serve` profiles should prefer `transport: unix_socket` and a host-generated `unix_socket_path` when supported by the installed `fm` command.
- Managed `fm serve` lifecycle must include startup timeout, readiness probing, cancellation, stderr diagnostics, and cleanup of stale socket files owned by the current process.

## Rules / Constraints

- AI provider configuration is application configuration, not exported master data.
- Canonical table records, schemas, generation configs, and export settings remain the source of truth for project data.
- Raw API keys must not be written into project-local YAML configuration.
- Hosted provider credentials are configured through the [AI settings screen](../ui-screen/ai-settings-screen.md) and stored through [AI secret storage service](../component/ai-secret-storage-service.md).
- Environment variables may exist as a developer fallback, but the ordinary user setup flow must not require them.
- Local provider URLs default to loopback addresses. Non-loopback local network URLs require `local_network_allowed: true`.
- The host must display a clear diagnostic when an active profile is missing, disabled, unreachable, or lacks a required capability.
- The host must display credential presence as masked metadata only. It must never return a stored API key to the browser.
- Tool-calling features must check `supports_tool_calls` before sending tool definitions to a provider.
- Streaming UI must check `supports_streaming`; non-streaming providers may still be used through buffered assistant responses.
- Provider capability declarations are advisory until verified by a provider health check.
- The host should expose a provider health check that validates endpoint reachability or command availability, configured model availability when practical, streaming support, structured output support, and tool-call support.
- OpenAI-compatible responses must be normalized into the same internal assistant message, tool-call, and token usage shape before the frontend sees them.
- Command-backed provider outputs must be normalized into the same internal assistant message, tool-call, tool-result, and usage shape when that metadata is available.
- Provider-specific errors must be translated into user-facing diagnostics without exposing raw secrets, full request headers, or credential storage internals.
- Local provider use should be allowed without internet access after the local provider and model are installed.
- On macOS, when `fm` is available and no explicit user provider selection exists, the host should default to the local `apple-fm-serve` profile.

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
  - id: local-codex
    display_name: Codex CLI
    provider_type: codex_cli
    command: codex
    args: [exec, --json]
    stdin_mode: prompt
    stdout_format: jsonl
    sandbox: read_only
    supports_streaming: true
    supports_tool_calls: true
  - id: apple-fm
    display_name: Apple Foundation Models
    provider_type: foundation_models_cli
    command: fm
    args: [respond]
    model: system
    stdin_mode: prompt
    stdout_format: text
    supports_streaming: true
    supports_tool_calls: true
    notes: "Concrete fm flags and event schema must be verified on the target macOS version."
  - id: apple-fm-serve
    display_name: Apple Foundation Models server
    provider_type: openai_compatible
    transport: unix_socket
    unix_socket_path: host-generated
    managed_process: true
    managed_process_command: fm
    managed_process_args: [serve]
    model: system
    supports_streaming: true
    supports_tool_calls: true
    notes: "Backed by managed fm serve over a Unix domain socket when supported; loopback TCP is a manual/debug fallback."
```

## Uses Common Details

- None yet.

## Reads

- Host environment variables for provider credentials only when explicitly enabled as a development fallback.
- Host-specific secret references when configured.
- Credential presence metadata from [AI secret storage service](../component/ai-secret-storage-service.md).
- Local command availability and version output for command-backed providers.
- Optional project-local AI configuration file when the host supports workspace profiles.

## Writes

- AI configuration updates from [AI settings screen](../ui-screen/ai-settings-screen.md).
- No canonical master data records.
- No raw API keys in project-local configuration files.

## Related Requirements

- [AI assistant service](../component/ai-assistant-service.md)
- [In-app AI assistant panel](../ui-screen/in-app-ai-assistant-panel.md)
- [AI settings screen](../ui-screen/ai-settings-screen.md)
- [AI secret storage service](../component/ai-secret-storage-service.md)
- [Agent tool contract](../component/agent-tool-contract.md)

## Native-Language Summary

OpenAI互換API、OpenAI公式API、Ollama、LM Studioに加えて、Codex CLIやmacOSの `fm` Foundation Modelsコマンドを同じAI設定モデルで扱う。初期実装では無料で利用できるローカル `fm serve` をOpenAI互換プロファイルとして優先対応する。API providerとcommand providerは区別し、APIキーは環境変数やホスト側の秘密情報として扱う。ローカルLLMやローカルagentは、ツール呼び出し、構造化出力、ストリーミングの能力をプロファイルで明示し、実行前のhealth checkで検証する。
