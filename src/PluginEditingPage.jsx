import React, { useEffect, useRef, useState } from "react";
import { api, binaryAssetUrl, editorPluginAssetUrl, uploadBinaryAsset } from "./api.js";
import styles from "./TableEditingPage.module.css";

export function PluginEditingPage({
  session,
  editGenerationId,
  tableViewMode,
  onClose,
  onSaved,
  onStatus
}) {
  const iframeRef = useRef(null);
  const instanceRef = useRef(null);
  const pendingChangesRef = useRef(null);
  const [context, setContext] = useState(null);
  const [diagnostics, setDiagnostics] = useState([]);
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [message, setMessage] = useState("Loading plugin...");

  useEffect(() => {
    setLoading(true);
    setDiagnostics([]);
    setDirty(false);
    pendingChangesRef.current = null;
    api(`/api/editor-plugins/${encodeURIComponent(session.plugin.plugin_id)}/context`, {
      method: "POST",
      body: JSON.stringify({
        activeGenerationId: editGenerationId,
        mode: tableViewMode,
        entryPointId: session.entryPoint.id,
        entry: session.entry
      })
    }).then(({ payload }) => {
      setContext(payload);
      setMessage("Plugin context loaded.");
    }).catch((error) => {
      setMessage(error.message);
      onStatus(error.message);
    }).finally(() => setLoading(false));
  }, [session, editGenerationId, tableViewMode, onStatus]);

  async function mountPlugin() {
    const frame = iframeRef.current;
    if (!frame || !context) return;
    const pluginEntry = frame.contentWindow?.MasterDataMatePlugin;
    if (typeof pluginEntry !== "function") {
      setMessage("Plugin entry function was not found.");
      return;
    }
    instanceRef.current?.dispose?.();
    const host = {
      setDirty(value = true) {
        setDirty(Boolean(value));
      },
      async proposeChanges(changes) {
        pendingChangesRef.current = changes;
        setDirty(true);
        const { payload } = await api(`/api/editor-plugins/${encodeURIComponent(session.plugin.plugin_id)}/changes/validate`, {
          method: "POST",
          body: JSON.stringify({
            activeGenerationId: editGenerationId,
            mode: tableViewMode,
            entryPointId: session.entryPoint.id,
            entry: session.entry,
            changes
          })
        });
        setDiagnostics(payload.diagnostics ?? []);
        setMessage(payload.diagnostics?.length ? "Plugin changes have diagnostics." : "Plugin changes are valid.");
        return payload;
      },
      async requestSave(options = {}) {
        return savePluginChanges(Boolean(options.force));
      },
      async uploadBinaryAsset({ table, key, field, file }) {
        return uploadBinaryAsset({ table, key, field, generationId: editGenerationId, file });
      },
      getBinaryAssetUrl({ table, key }) {
        return binaryAssetUrl(table, key);
      },
      notify(payload) {
        const text = typeof payload === "string" ? payload : payload?.message;
        if (text) {
          setMessage(text);
          onStatus(text);
        }
      }
    };
    const nextContext = { ...context, host };
    const instance = await pluginEntry(nextContext);
    instanceRef.current = instance ?? {};
    instanceRef.current?.mount?.();
    setMessage(`${session.plugin.display_name} loaded.`);
  }

  async function savePluginChanges(force = false) {
    setSaving(true);
    try {
      const beforeSave = await instanceRef.current?.beforeSave?.();
      const changes = beforeSave ?? pendingChangesRef.current;
      if (!changes) {
        setMessage("No plugin changes to save.");
        setSaving(false);
        return null;
      }
      pendingChangesRef.current = changes;
      const { status, payload } = await api(`/api/editor-plugins/${encodeURIComponent(session.plugin.plugin_id)}/changes/commit`, {
        method: "POST",
        body: JSON.stringify({
          activeGenerationId: editGenerationId,
          mode: tableViewMode,
          entryPointId: session.entryPoint.id,
          entry: session.entry,
          changes,
          force
        })
      });
      setDiagnostics(payload.diagnostics ?? []);
      if (status === 409 && payload.requiresForce) {
        if (window.confirm("Validation errors exist. Save anyway?")) return savePluginChanges(true);
        setMessage("Plugin save cancelled.");
        return payload;
      }
      setDirty(false);
      setMessage(payload.diagnostics?.length ? "Plugin saved with diagnostics." : "Plugin saved.");
      await onSaved?.();
      return payload;
    } catch (error) {
      setMessage(error.message);
      onStatus(error.message);
      return null;
    } finally {
      setSaving(false);
    }
  }

  useEffect(() => () => instanceRef.current?.dispose?.(), []);

  return (
    <section className={styles.workspace}>
      <header className={styles.toolbar}>
        <div>
          <h1>{session.plugin.display_name}</h1>
          <p>{session.plugin.description || message}</p>
        </div>
        <div className={styles.actions}>
          <button type="button" onClick={onClose} disabled={saving}>Back</button>
          <button type="button" className={styles.primary} onClick={() => savePluginChanges(false)} disabled={saving || loading}>
            {saving ? "Saving" : "Save"}
          </button>
        </div>
      </header>
      <div className={styles.pluginFrameWrap}>
        {context ? (
          <iframe
            key={`${session.plugin.plugin_id}-${session.entryPoint.id}-${editGenerationId}`}
            ref={iframeRef}
            title={session.plugin.display_name}
            className={styles.pluginFrame}
            src={editorPluginAssetUrl(session.plugin.plugin_id)}
            onLoad={mountPlugin}
          />
        ) : (
          <div className={styles.emptyState}>{message}</div>
        )}
      </div>
      <footer className={styles.statusBar}>
        <span className={dirty ? styles.dirty : ""}>{dirty ? "Pending plugin edits" : message}</span>
        <span>{diagnostics.length} diagnostics</span>
      </footer>
    </section>
  );
}
