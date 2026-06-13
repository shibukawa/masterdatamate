import React, { useEffect, useMemo, useState } from "react";
import { api } from "./api.js";
import styles from "./AISettingsPage.module.css";

const MASKED_CREDENTIAL = "********";
const APPLE_FM_PROFILE_ID = "apple-fm-serve";
const CODEX_CLI_PROFILE_ID = "local-codex";
const APPLE_FM_FALLBACK_MODELS = ["system", "pcc"];

function normalizeProfile(profile = {}) {
  return {
    id: profile.id ?? "",
    display_name: profile.display_name ?? profile.id ?? "",
    provider_type: profile.provider_type ?? "openai_compatible",
    base_url: profile.base_url ?? "",
    model: profile.model ?? "",
    command: profile.command ?? "",
    args: profile.args ?? [],
    stdin_mode: profile.stdin_mode ?? "",
    stdout_format: profile.stdout_format ?? "",
    sandbox: profile.sandbox ?? "",
    supports_streaming: Boolean(profile.supports_streaming),
    supports_tool_calls: Boolean(profile.supports_tool_calls),
    local_network_allowed: Boolean(profile.local_network_allowed),
    requires_api_key: Boolean(profile.requires_api_key),
    has_api_key: Boolean(profile.has_api_key),
    api_key_ref: profile.api_key_ref ?? "",
    notes: profile.notes ?? "",
    health_status: profile.health_status ?? "",
    health_message: profile.health_message ?? ""
  };
}

function newProfile() {
  return normalizeProfile({
    id: `custom-${Date.now()}`,
    display_name: "Custom OpenAI compatible",
    provider_type: "openai_compatible",
    base_url: "http://127.0.0.1:8000/v1",
    supports_streaming: true,
    supports_tool_calls: true
  });
}

export function AISettingsPage({ dirty, onDirtyChange, onStatus }) {
  const [settings, setSettings] = useState({ version: 1, enabled: false, active_profile: "", profiles: [] });
  const [selectedProfileId, setSelectedProfileId] = useState("");
  const [credentialEdits, setCredentialEdits] = useState({});
  const [localProviders, setLocalProviders] = useState([]);
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [health, setHealth] = useState(null);
  const [profileModels, setProfileModels] = useState({});

  useEffect(() => {
    reload().catch((loadError) => {
      setError(loadError.message);
      onStatus?.(loadError.message);
    });
  }, []);

  const selectedProfile = useMemo(
    () => settings.profiles.find((profile) => profile.id === selectedProfileId) ?? settings.profiles[0] ?? null,
    [settings.profiles, selectedProfileId]
  );
  const selectedIsAppleFM = selectedProfile?.id === APPLE_FM_PROFILE_ID;
  const selectedIsCodex = selectedProfile?.id === CODEX_CLI_PROFILE_ID;
  const selectedIsFixedLocalAgent = selectedIsAppleFM || selectedIsCodex;
  const selectedModelOptions = useMemo(() => {
    if (!selectedProfile) return [];
    const fetched = profileModels[selectedProfile.id] ?? [];
    const base = selectedIsAppleFM ? (fetched.length ? fetched : APPLE_FM_FALLBACK_MODELS) : fetched;
    return [...new Set([...base, selectedProfile.model].filter(Boolean))];
  }, [profileModels, selectedIsAppleFM, selectedProfile]);

  useEffect(() => {
    if (!selectedProfile || selectedProfile.id !== APPLE_FM_PROFILE_ID) return;
    loadProfileModels(selectedProfile.id).catch(() => {
      setProfileModels((current) => ({ ...current, [selectedProfile.id]: APPLE_FM_FALLBACK_MODELS }));
    });
  }, [selectedProfile?.id, selectedProfile?.base_url]);

  async function reload() {
    setLoading(true);
    const [{ payload }, localResponse] = await Promise.all([
      api("/api/ai/settings"),
      api("/api/ai/local-providers").catch(() => ({ payload: { providers: [] } }))
    ]);
    const profiles = (payload.profiles ?? []).map(normalizeProfile);
    setSettings({ ...payload, profiles });
    setSelectedProfileId(payload.active_profile || profiles[0]?.id || "");
    setCredentialEdits({});
    setLocalProviders(localResponse.payload.providers ?? []);
    setProfileModels({});
    setHealth(null);
    onDirtyChange?.(false);
    setLoading(false);
    onStatus?.("AI settings loaded.");
  }

  function updateSettings(patch) {
    setSettings((current) => ({ ...current, ...patch }));
    onDirtyChange?.(true);
  }

  function updateProfile(profileId, patch) {
    if (profileId === APPLE_FM_PROFILE_ID || profileId === CODEX_CLI_PROFILE_ID) {
      patch = {
        ...patch,
        provider_type: profileId === APPLE_FM_PROFILE_ID ? "openai_compatible" : "codex_cli",
        requires_api_key: false,
        supports_streaming: true,
        supports_tool_calls: true
      };
      if (profileId === CODEX_CLI_PROFILE_ID) {
        patch = {
          ...patch,
          base_url: "",
          model: "",
          command: "codex",
          args: ["exec", "--json"],
          stdin_mode: "prompt",
          stdout_format: "jsonl",
          sandbox: "read_only"
        };
      }
    }
    setSettings((current) => ({
      ...current,
      profiles: current.profiles.map((profile) => profile.id === profileId ? normalizeProfile({ ...profile, ...patch }) : profile)
    }));
    onDirtyChange?.(true);
  }

  function addProfile() {
    const profile = newProfile();
    setSettings((current) => ({
      ...current,
      active_profile: current.active_profile || profile.id,
      profiles: [...current.profiles, profile]
    }));
    setSelectedProfileId(profile.id);
    onDirtyChange?.(true);
  }

  function removeProfile() {
    if (!selectedProfile || settings.profiles.length <= 1) return;
    if (!window.confirm(`Delete AI profile ${selectedProfile.display_name}?`)) return;
    const nextProfiles = settings.profiles.filter((profile) => profile.id !== selectedProfile.id);
    const nextActive = settings.active_profile === selectedProfile.id ? nextProfiles[0]?.id ?? "" : settings.active_profile;
    setSettings((current) => ({ ...current, profiles: nextProfiles, active_profile: nextActive }));
    setSelectedProfileId(nextActive || nextProfiles[0]?.id || "");
    onDirtyChange?.(true);
  }

  function credentialValue(profile) {
    if (!profile) return "";
    if (Object.prototype.hasOwnProperty.call(credentialEdits, profile.id)) return credentialEdits[profile.id];
    return profile.has_api_key ? MASKED_CREDENTIAL : "";
  }

  function setCredential(profileId, value) {
    if (profileId === APPLE_FM_PROFILE_ID || profileId === CODEX_CLI_PROFILE_ID) return;
    setCredentialEdits((current) => ({ ...current, [profileId]: value }));
    onDirtyChange?.(true);
  }

  async function loadProfileModels(profileId) {
    const { payload } = await api(`/api/ai/profiles/${encodeURIComponent(profileId)}/models`);
    const models = (payload.models ?? []).map((model) => model.id).filter(Boolean);
    setProfileModels((current) => ({ ...current, [profileId]: models }));
    return models;
  }

  async function clearCredential() {
    if (!selectedProfile) return;
    setSaving(true);
    try {
      const { payload } = await api(`/api/ai/profiles/${encodeURIComponent(selectedProfile.id)}/credential`, { method: "DELETE" });
      setSettings({ ...payload, profiles: (payload.profiles ?? []).map(normalizeProfile) });
      setCredentialEdits((current) => ({ ...current, [selectedProfile.id]: "" }));
      onDirtyChange?.(false);
      onStatus?.("AI credential cleared.");
    } catch (clearError) {
      setError(clearError.message);
      onStatus?.(clearError.message);
    } finally {
      setSaving(false);
    }
  }

  async function save() {
    setSaving(true);
    setError("");
    try {
      const payload = {
        ...settings,
        profiles: settings.profiles.map((profile) => {
          const apiKey = credentialEdits[profile.id];
          if (profile.id === APPLE_FM_PROFILE_ID || profile.id === CODEX_CLI_PROFILE_ID) {
            return {
              ...profile,
              provider_type: profile.id === APPLE_FM_PROFILE_ID ? "openai_compatible" : "codex_cli",
              base_url: profile.id === CODEX_CLI_PROFILE_ID ? "" : profile.base_url,
              model: profile.id === CODEX_CLI_PROFILE_ID ? "" : profile.model,
              command: profile.id === CODEX_CLI_PROFILE_ID ? "codex" : "",
              args: profile.id === CODEX_CLI_PROFILE_ID ? ["exec", "--json"] : [],
              stdin_mode: profile.id === CODEX_CLI_PROFILE_ID ? "prompt" : "",
              stdout_format: profile.id === CODEX_CLI_PROFILE_ID ? "jsonl" : "",
              sandbox: profile.id === CODEX_CLI_PROFILE_ID ? "read_only" : "",
              requires_api_key: false,
              supports_streaming: true,
              supports_tool_calls: true,
              api_key: "",
              clear_api_key: false,
              api_key_ref: ""
            };
          }
          return {
            ...profile,
            api_key: apiKey && apiKey !== MASKED_CREDENTIAL ? apiKey : "",
            clear_api_key: apiKey === "" && profile.has_api_key ? false : undefined
          };
        })
      };
      const { payload: saved } = await api("/api/ai/settings", {
        method: "PUT",
        body: JSON.stringify(payload)
      });
      setSettings({ ...saved, profiles: (saved.profiles ?? []).map(normalizeProfile) });
      setCredentialEdits({});
      onDirtyChange?.(false);
      onStatus?.("AI settings saved.");
    } catch (saveError) {
      setError(saveError.message);
      onStatus?.(saveError.message);
    } finally {
      setSaving(false);
    }
  }

  async function testProfile() {
    if (!selectedProfile) return;
    setSaving(true);
    setHealth(null);
    try {
      const { payload } = await api(`/api/ai/profiles/${encodeURIComponent(selectedProfile.id)}/health`, { method: "POST" });
      setHealth(payload);
      onStatus?.(payload.ok ? "AI profile check passed." : "AI profile check failed.");
    } catch (testError) {
      setError(testError.message);
      onStatus?.(testError.message);
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <section className={styles.page}>
        <header className={styles.toolbar}>
          <div className={styles.title}>
            <h1>AI Settings</h1>
            <span>Loading</span>
          </div>
        </header>
      </section>
    );
  }

  return (
    <section className={styles.page}>
      <header className={styles.toolbar}>
        <div className={styles.title}>
          <h1>AI Settings</h1>
          <span>{dirty ? "Unsaved changes" : "Saved"}</span>
        </div>
        <div className={styles.actions}>
          <button type="button" onClick={reload} disabled={saving}>Reload</button>
          <button type="button" className={styles.primary} onClick={save} disabled={saving || !dirty}>
            {saving ? "Saving" : "Save"}
          </button>
        </div>
      </header>

      <div className={styles.body}>
        <aside className={styles.panel}>
          <div className={styles.panelHeader}>
            <label className={styles.toggleRow}>
              <input
                type="checkbox"
                checked={Boolean(settings.enabled)}
                onChange={(event) => updateSettings({ enabled: event.target.checked })}
              />
              <span>Enable AI panel</span>
            </label>
            <label className={styles.field}>
              <span>Active backend</span>
              <select value={settings.active_profile} onChange={(event) => updateSettings({ active_profile: event.target.value })}>
                {settings.profiles.map((profile) => (
                  <option key={profile.id} value={profile.id}>{profile.display_name}</option>
                ))}
              </select>
            </label>
            <div className={styles.localProviders}>
              {localProviders.map((provider) => (
                <span
                  key={provider.id}
                  className={`${styles.providerBadge} ${provider.available ? styles.providerAvailable : ""}`}
                  title={provider.message || provider.command}
                >
                  {provider.id}
                </span>
              ))}
            </div>
          </div>
          <div className={styles.profileList}>
            {settings.profiles.map((profile) => (
              <button
                key={profile.id}
                type="button"
                className={`${styles.profileButton} ${profile.id === selectedProfile?.id ? styles.active : ""}`}
                onClick={() => setSelectedProfileId(profile.id)}
              >
                <strong>{profile.display_name}</strong>
                <span className={styles.muted}>{profile.provider_type}</span>
              </button>
            ))}
            <button type="button" onClick={addProfile}>Add profile</button>
          </div>
        </aside>

        <section className={styles.editor}>
          <header className={styles.editorHeader}>
            <div>
              <h2>{selectedProfile?.display_name ?? "Profile"}</h2>
              <span className={styles.statusText}>{selectedProfile?.health_message || selectedProfile?.health_status || selectedProfile?.id}</span>
            </div>
            <div className={styles.actions}>
              <button type="button" onClick={testProfile} disabled={!selectedProfile || saving}>Test</button>
              <button type="button" onClick={removeProfile} disabled={!selectedProfile || selectedIsFixedLocalAgent || settings.profiles.length <= 1 || saving}>Delete</button>
            </div>
          </header>

          {selectedProfile ? (
            <div className={styles.form}>
              <label className={styles.field}>
                <span>ID</span>
                <input value={selectedProfile.id} readOnly />
              </label>
              <label className={styles.field}>
                <span>Display name</span>
                <input value={selectedProfile.display_name} onChange={(event) => updateProfile(selectedProfile.id, { display_name: event.target.value })} />
              </label>
              <label className={styles.field}>
                <span>Provider type</span>
                <select
                  value={selectedIsAppleFM ? "openai_compatible" : selectedProfile.provider_type}
                  onChange={(event) => updateProfile(selectedProfile.id, { provider_type: event.target.value })}
                  disabled={selectedIsFixedLocalAgent}
                >
                  <option value="openai_compatible">OpenAI compatible</option>
                  <option value="openai">OpenAI</option>
                  <option value="ollama">Ollama</option>
                  <option value="lmstudio">LM Studio</option>
                  <option value="codex_cli">Codex CLI</option>
                  <option value="foundation_models_cli">Foundation Models CLI</option>
                </select>
              </label>
              {!selectedIsCodex ? (
                <label className={styles.field}>
                  <span>Model</span>
                  {selectedIsAppleFM ? (
                    <select value={selectedProfile.model || selectedModelOptions[0] || "system"} onChange={(event) => updateProfile(selectedProfile.id, { model: event.target.value })}>
                      {selectedModelOptions.map((model) => (
                        <option key={model} value={model}>{model}</option>
                      ))}
                    </select>
                  ) : (
                    <input value={selectedProfile.model} onChange={(event) => updateProfile(selectedProfile.id, { model: event.target.value })} />
                  )}
                </label>
              ) : null}
              {!selectedIsCodex ? (
                <label className={styles.wideField}>
                  <span>Base URL</span>
                  <input value={selectedProfile.base_url} onChange={(event) => updateProfile(selectedProfile.id, { base_url: event.target.value })} />
                </label>
              ) : null}
              <label className={styles.field}>
                <span>Command</span>
                <input value={selectedProfile.command} onChange={(event) => updateProfile(selectedProfile.id, { command: event.target.value })} readOnly={selectedIsCodex} />
              </label>
              <label className={styles.field}>
                <span>Arguments</span>
                <input
                  value={(selectedProfile.args ?? []).join(" ")}
                  onChange={(event) => updateProfile(selectedProfile.id, { args: event.target.value.trim() ? event.target.value.trim().split(/\s+/) : [] })}
                  readOnly={selectedIsCodex}
                />
              </label>
              {!selectedIsCodex ? (
                <label className={styles.wideField}>
                  <span>API key</span>
                  <input
                    type="password"
                    value={selectedIsAppleFM ? "" : credentialValue(selectedProfile)}
                    onFocus={() => {
                      if (!selectedIsAppleFM && credentialValue(selectedProfile) === MASKED_CREDENTIAL) setCredential(selectedProfile.id, "");
                    }}
                    onChange={(event) => setCredential(selectedProfile.id, event.target.value)}
                    autoComplete="off"
                    disabled={selectedIsAppleFM}
                  />
                </label>
              ) : null}
              <div className={styles.wideField}>
                <span>Capabilities</span>
                <div className={styles.checkboxGrid}>
                  <label>
                    <input
                      type="checkbox"
                      checked={selectedIsFixedLocalAgent ? false : selectedProfile.requires_api_key}
                      onChange={(event) => updateProfile(selectedProfile.id, { requires_api_key: event.target.checked })}
                      disabled={selectedIsFixedLocalAgent}
                    />
                    API key required
                  </label>
                  <label>
                    <input
                      type="checkbox"
                      checked={selectedIsFixedLocalAgent ? true : selectedProfile.supports_streaming}
                      onChange={(event) => updateProfile(selectedProfile.id, { supports_streaming: event.target.checked })}
                      disabled={selectedIsFixedLocalAgent}
                    />
                    Streaming
                  </label>
                  <label>
                    <input
                      type="checkbox"
                      checked={selectedIsFixedLocalAgent ? true : selectedProfile.supports_tool_calls}
                      onChange={(event) => updateProfile(selectedProfile.id, { supports_tool_calls: event.target.checked })}
                      disabled={selectedIsFixedLocalAgent}
                    />
                    Tool calls
                  </label>
                </div>
              </div>
              <label className={styles.wideField}>
                <span>Notes</span>
                <textarea value={selectedProfile.notes} onChange={(event) => updateProfile(selectedProfile.id, { notes: event.target.value })} />
              </label>
              {selectedProfile.has_api_key ? (
                <div className={styles.wideField}>
                  <button type="button" className={styles.button} onClick={clearCredential} disabled={saving}>Clear saved API key</button>
                </div>
              ) : null}
              {health ? (
                <div className={styles.wideField}>
                  <span>{health.ok ? "Connection check passed" : "Connection check failed"}</span>
                  <div className={styles.statusText}>
                    {(health.checks ?? []).map((check) => `${check.name}: ${check.message}`).join(" / ")}
                  </div>
                </div>
              ) : null}
              {error ? <div className={`${styles.wideField} ${styles.error}`}>{error}</div> : null}
            </div>
          ) : null}
        </section>
      </div>
    </section>
  );
}
