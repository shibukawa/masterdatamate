---
id: "ai-secret-storage-service"
type: "server-component"
title: "AI secret storage service"
aliases: ["AI credential storage", "LLM credential storage"]
tags: ["ai", "security", "credentials", "keychain", "wincred"]
facts:
  lifecycle.status: "blueprint"
  owner: "application"
---

# AI secret storage service

## Summary

The AI secret storage service stores provider credentials supplied through the AI settings screen. It keeps API keys out of browser state after the save request, out of project-local YAML configuration, and out of canonical master data.

On desktop-capable hosts, credentials are stored in the operating system credential store, such as macOS Keychain, Windows Credential Manager, or Linux Secret Service implementations such as GNOME Keyring. The Go implementation should use the `github.com/99designs/keyring` library as the default cross-platform keyring adapter. Other hosts must use an equivalent server-side secret store or clearly report that persistent credential storage is unavailable.

## Responsibilities

- Accept API keys only through explicit credential update requests from the AI settings screen.
- Store hosted-provider API keys in OS-backed credential storage when available.
- Return only credential presence and metadata needed by the UI, never the raw secret value.
- Delete stored provider credentials when the user clears a credential.
- Provide provider adapters with credential values only inside backend process memory for the duration needed to call the provider.
- Redact credentials from logs, diagnostics, validation errors, and assistant context.
- Keep secret references stable across provider profile edits when the profile ID remains stable.
- Use an existing Go keyring library rather than custom OS-specific credential-store implementations.

## Interfaces

- Save or replace a provider credential.
- Delete a provider credential.
- Check whether a provider credential exists.
- Resolve a credential for backend provider calls.

## Storage Rules

- The default Go implementation uses `github.com/99designs/keyring`.
- The keyring service name should be stable and product-scoped, such as `MasterDataMate`.
- Credential item keys should be stable, non-secret identifiers derived from provider profile IDs, such as `ai-provider:<profileId>:api-key`.
- macOS uses the library's macOS Keychain backend when available.
- Windows uses the library's Windows Credential Manager backend when available.
- Linux desktop hosts use the library's Secret Service backend when available, which can be backed by GNOME Keyring or KWallet depending on the user's desktop environment.
- macOS packaged desktop and local web hosts should use Keychain when available.
- Windows packaged desktop and local web hosts should use Windows Credential Manager when available.
- Linux or unsupported hosts should use Secret Service or another platform-appropriate secure store when available; otherwise the AI settings screen must show that persistent credential storage is unsupported.
- File-backed or plaintext fallback stores are not acceptable for ordinary AI API key persistence.
- Any encrypted-file fallback must be explicitly configured for development or unsupported environments and must show a warning in AI settings.
- Project-local AI configuration may contain only a secret reference or credential-presence metadata, not the API key.
- Browser clients receive only `hasCredential: true` or equivalent masked metadata.
- The settings editor displays stored credentials as a masked placeholder such as `********`.
- The masked placeholder is not a value that can be saved as a credential. Saving without entering a new key preserves the existing credential.
- Entering a new key replaces the stored credential only after the user saves the AI settings form.
- Clearing the credential field and confirming removal deletes the stored credential.

## Security Rules

- API keys must not be sent back to the browser after initial submission.
- API keys must not be included in CopilotKit shared state, AG-UI events, provider health-check responses, agent tools, or exported artifacts.
- API keys must not be stored in environment variables by the application as a persistence mechanism.
- Environment variables may still be supported as an advanced fallback for development, but the ordinary UI flow must not require users to configure them.
- Health-check failures must not echo provider request headers or credential fragments.
- Secret lookup failures must return a credential-missing diagnostic rather than a raw storage error unless developer diagnostics are explicitly enabled.
- Keyring backend selection, service name, item key, and credential presence may be logged at debug level; credential values must never be logged.
- The application must not shell out to platform credential commands for ordinary storage when the Go keyring library can provide the required backend.

## Go Library Requirements

- Preferred module: `github.com/99designs/keyring`.
- The service should wrap the library behind a small internal interface so tests can use an in-memory fake.
- The wrapper should expose `Set(profileId, secret)`, `Get(profileId)`, `Delete(profileId)`, and `Has(profileId)` operations.
- `Has(profileId)` should avoid returning secret bytes to callers that only need credential presence.
- The implementation should classify library errors into user-facing categories: unavailable backend, locked keyring, credential missing, permission denied, and unexpected storage error.
- The build should avoid optional backends that require unavailable native dependencies on target platforms unless explicitly enabled by build tags.
- Automated tests should not write to the user's real Keychain, Credential Manager, or Secret Service. Use a fake backend or isolated integration tests gated by explicit environment variables.

## Dependencies

- [AI provider configuration model](../data-model/ai-provider-configuration-model.md)
- [AI settings screen](../ui-screen/ai-settings-screen.md)
- [AI assistant service](ai-assistant-service.md)
- [Web service host](../server-component/web-service-host.md)
- [Wails desktop host](../server-component/wails-desktop-host.md)

## Reads

- Provider profile IDs and credential references.
- OS credential store entries.
- Keyring backend metadata when needed for diagnostics.

## Writes

- OS credential store entries for provider API keys through `github.com/99designs/keyring` or the configured secure-store adapter.
- Credential presence metadata in API responses.
- No canonical master data records.

## Related Requirements

- [In-app AI assistant panel](../ui-screen/in-app-ai-assistant-panel.md)
- [Go embedded web server host](../server-component/go-embedded-web-server-host.md)

## Native-Language Summary

AI設定画面から入力されたAPIキーを、ブラウザやプロジェクトYAMLに残さず、Goの `github.com/99designs/keyring` を通して macOS Keychain、Windows Credential Manager、Linux Secret Service/GNOME Keyring/KWallet などのOS認証情報ストアに保存するサービス。画面やAPIレスポンスには「保存済み」という情報だけを返し、編集欄には `********` のようなマスク表示を出す。保存済みキーはバックエンドがプロバイダ呼び出し時だけ取り出す。
