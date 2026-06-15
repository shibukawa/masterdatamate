import React, { useEffect, useMemo, useRef, useState } from "react";
import { api } from "./api.js";
import styles from "./AIAssistantPanel.module.css";

function formatTime(value) {
  if (!value) return "";
  try {
    return new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit" }).format(new Date(value));
  } catch {
    return "";
  }
}

function normalizeSession(session = {}) {
  return {
    session_id: session.session_id ?? "",
    title: session.title ?? "New chat",
    status: session.status ?? "active",
    runtime_mode: session.runtime_mode ?? "managed_chat_agent",
    active_profile_id: session.active_profile_id ?? "",
    updated_at: session.updated_at ?? "",
    messages: session.messages ?? [],
    tool_events: session.tool_events ?? []
  };
}

function eventTitle(event) {
  const round = Number.isInteger(event.round) ? ` #${event.round + 1}` : "";
  if (event.kind === "request") return `Request${round}`;
  if (event.kind === "response") return `Response${round}`;
  if (event.kind === "tool_call") return `Tool call: ${event.name ?? ""}`;
  if (event.kind === "tool_result") return `Tool result: ${event.name ?? ""}`;
  if (event.kind === "frontend_staging_requested") return "Frontend staging";
  if (event.kind === "frontend_tool_requested") return `Frontend tool: ${event.name ?? ""}`;
  if (event.kind === "frontend_tool_result") return `Frontend result: ${event.name ?? ""}`;
  if (event.kind === "assistant_message") return "Assistant raw message";
  if (event.kind === "run_started") return "Run started";
  return event.kind ?? "Debug";
}

function eventSummary(event) {
  if (event.kind === "request") return `${event.model ?? ""} ${event.url ?? ""}`.trim();
  if (event.kind === "response") return `HTTP ${event.status ?? ""}`.trim();
  if (event.kind === "tool_call") return JSON.stringify(event.arguments ?? {});
  if (event.kind === "tool_result") return event.result?.summary ?? JSON.stringify(event.result ?? {});
  if (event.kind === "frontend_tool_requested") return JSON.stringify(event.arguments ?? {});
  if (event.kind === "frontend_tool_result") return event.result?.summary ?? JSON.stringify(event.result ?? {});
  if (event.kind === "frontend_staging_requested") {
    if (event.result) return `${event.result.accepted?.length ?? 0} staged, ${event.result.rejected?.length ?? 0} rejected`;
    return JSON.stringify(event.arguments ?? {});
  }
  if (event.kind === "assistant_message") return stringifyInline(event.reasoning_content || event.thinking || event.content || `${event.tool_calls?.length ?? 0} tool call(s)`);
  return "";
}

function stringifyInline(value) {
  if (value === undefined || value === null) return "";
  if (typeof value === "string") return value;
  return JSON.stringify(value);
}

function visibleActivity(event) {
  return ["assistant_message", "tool_call", "tool_result", "frontend_staging_requested", "frontend_tool_requested", "frontend_tool_result"].includes(event?.kind);
}

export function AIAssistantPanel({ getCurrentContext, onStatus, onStageTableChanges }) {
  const [open, setOpen] = useState(false);
  const [settings, setSettings] = useState(null);
  const [sessions, setSessions] = useState([]);
  const [currentSession, setCurrentSession] = useState(null);
  const [selectedSessionId, setSelectedSessionId] = useState("");
  const [input, setInput] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [activityEvents, setActivityEvents] = useState([]);
  const [optimisticMessage, setOptimisticMessage] = useState(null);
  const timelineRef = useRef(null);

  const profiles = settings?.profiles ?? [];
  const managedProfiles = useMemo(
    () => profiles.filter((profile) => profile.provider_type !== "codex_cli"),
    [profiles]
  );
  const activeProfileId = currentSession?.active_profile_id || settings?.active_profile || managedProfiles[0]?.id || "";
  const activeProfile = managedProfiles.find((profile) => profile.id === activeProfileId) ?? managedProfiles[0] ?? null;
  const ready = Boolean(settings?.enabled && activeProfile);
  const messages = currentSession?.messages ?? [];
  const displayMessages = optimisticMessage ? [...messages, optimisticMessage] : messages;

  useEffect(() => {
    if (!open) return;
    reload().catch((loadError) => {
      setError(loadError.message);
      onStatus?.(loadError.message);
    });
  }, [open]);

  useEffect(() => {
    timelineRef.current?.scrollTo({ top: timelineRef.current.scrollHeight });
  }, [displayMessages.length, activityEvents.length, busy, open]);

  async function reload() {
    setBusy(true);
    setError("");
    try {
      const [settingsResponse, sessionsResponse] = await Promise.all([
        api("/api/ai/settings"),
        api("/api/ai/sessions")
      ]);
      const nextSettings = settingsResponse.payload;
      const nextSessions = sessionsResponse.payload.sessions ?? [];
      setSettings(nextSettings);
      setSessions(nextSessions);
      const targetId = selectedSessionId || nextSessions[0]?.session_id || "";
      if (targetId) {
        await loadSession(targetId);
      } else {
        setCurrentSession(null);
      }
      setOptimisticMessage(null);
    } finally {
      setBusy(false);
    }
  }

  async function loadSession(sessionId) {
    if (!sessionId) return;
    const { payload } = await api(`/api/ai/sessions/${encodeURIComponent(sessionId)}`);
    const session = normalizeSession(payload.session);
    setCurrentSession(session);
    setSelectedSessionId(session.session_id);
    setOptimisticMessage(null);
  }

  async function createSession() {
    setBusy(true);
    setError("");
    try {
      const { payload } = await api("/api/ai/sessions", {
        method: "POST",
        body: JSON.stringify({
          runtime_mode: "managed_chat_agent",
          profile_id: settings?.active_profile || managedProfiles[0]?.id || ""
        })
      });
      const session = normalizeSession(payload.session);
      if (payload.debug_events?.length) {
        session.tool_events = payload.debug_events;
      }
      setCurrentSession(session);
      setSelectedSessionId(session.session_id);
      setOptimisticMessage(null);
      const { payload: listPayload } = await api("/api/ai/sessions");
      setSessions(listPayload.sessions ?? []);
      onStatus?.("AI session created.");
    } catch (createError) {
      setError(createError.message);
      onStatus?.(createError.message);
    } finally {
      setBusy(false);
    }
  }

  async function compactSession() {
    if (!currentSession?.session_id) return;
    setBusy(true);
    setError("");
    try {
      const { payload } = await api(`/api/ai/sessions/${encodeURIComponent(currentSession.session_id)}/compact`, { method: "POST", body: JSON.stringify({}) });
      setCurrentSession(normalizeSession(payload.session));
      setOptimisticMessage(null);
      onStatus?.("AI session compacted.");
    } catch (compactError) {
      setError(compactError.message);
      onStatus?.(compactError.message);
    } finally {
      setBusy(false);
    }
  }

  async function deleteSession() {
    if (!currentSession?.session_id) return;
    if (!window.confirm(`Delete AI session "${currentSession.title}"?`)) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/ai/sessions/${encodeURIComponent(currentSession.session_id)}`, { method: "DELETE" });
      setCurrentSession(null);
      setSelectedSessionId("");
      setOptimisticMessage(null);
      const { payload } = await api("/api/ai/sessions");
      setSessions(payload.sessions ?? []);
      if (payload.sessions?.[0]?.session_id) await loadSession(payload.sessions[0].session_id);
      onStatus?.("AI session deleted.");
    } catch (deleteError) {
      setError(deleteError.message);
      onStatus?.(deleteError.message);
    } finally {
      setBusy(false);
    }
  }

  async function send() {
    const message = input.trim();
    if (!message || busy) return;
    setInput("");
    setOptimisticMessage({
      id: `optimistic-${Date.now()}`,
      role: "user",
      content: message,
      created_at: new Date().toISOString(),
      source: "optimistic_user_input"
    });
    setBusy(true);
    setError("");
    setActivityEvents([]);
    try {
      const response = await fetch("/api/ai/runs/stream", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          sessionId: currentSession?.session_id || "",
          profileId: activeProfileId,
          message,
          context: getCurrentContext?.() ?? {}
        })
      });
      if (!response.ok || !response.body) {
        const payload = await response.json().catch(() => ({ error: "AI request failed" }));
        throw new Error(payload.error ?? "AI request failed");
      }
      let finalPayload = null;
      let stagedDuringStream = false;
      for await (const event of readNDJSON(response.body)) {
        if (event.kind === "frontend_tool_request") {
          const toolResult = await executeFrontendTool(event.request, { getCurrentContext, onStageTableChanges });
          await postFrontendToolResult(event.request?.request_id, toolResult);
          if (event.request?.name === "stage_table_changes" && toolResult.result?.success) stagedDuringStream = true;
          setActivityEvents((current) => [...current.slice(-7), {
            kind: "frontend_tool_result",
            timestamp: new Date().toISOString(),
            name: event.request?.name,
            result: toolResult.result ?? { success: false, error: toolResult.error }
          }]);
          continue;
        }
        if (event.kind === "debug_event" && visibleActivity(event.event)) {
          setActivityEvents((current) => [...current.slice(-7), event.event]);
          continue;
        }
        if (event.kind === "error") {
          finalPayload = event.payload ?? null;
          if (finalPayload?.session) {
            const session = normalizeSession(finalPayload.session);
            setCurrentSession(session);
            setSelectedSessionId(session.session_id);
            setOptimisticMessage(null);
            const { payload: listPayload } = await api("/api/ai/sessions").catch(() => ({ payload: { sessions: [] } }));
            setSessions(listPayload.sessions ?? []);
          }
          throw new Error(event.error ?? "AI request failed");
        }
        if (event.kind === "final") {
          finalPayload = event.payload;
        }
      }
      if (!finalPayload) throw new Error("AI response did not finish.");
      const session = normalizeSession(finalPayload.session);
      setCurrentSession(session);
      setSelectedSessionId(session.session_id);
      setOptimisticMessage(null);
      if (finalPayload.stage_table_changes && onStageTableChanges && !stagedDuringStream) {
        const stageResult = onStageTableChanges(finalPayload.stage_table_changes);
        setActivityEvents((current) => [...current.slice(-7), {
          kind: "frontend_staging_requested",
          timestamp: new Date().toISOString(),
          arguments: finalPayload.stage_table_changes,
          result: stageResult
        }]);
        onStatus?.(`AI staged ${stageResult.accepted?.length ?? 0} operation(s); ${stageResult.rejected?.length ?? 0} rejected.`);
      }
      const { payload: listPayload } = await api("/api/ai/sessions");
      setSessions(listPayload.sessions ?? []);
      setActivityEvents([]);
      onStatus?.("AI response received.");
    } catch (sendError) {
      setActivityEvents([]);
      setError(sendError.message);
      onStatus?.(sendError.message);
      setInput(message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <button
        type="button"
        className={styles.fab}
        aria-label={open ? "Close AI assistant" : "Open AI assistant"}
        title="Ask AI"
        onClick={() => setOpen((value) => !value)}
      >
        <svg aria-hidden="true" viewBox="0 0 24 24" focusable="false">
          <path d="M12 3c-4.4 0-8 3-8 6.8 0 2.1 1.1 4 2.9 5.3L6 20.5l5.4-3.1h.6c4.4 0 8-3 8-6.8S16.4 3 12 3Zm-3 8.4a1.2 1.2 0 1 1 0-2.4 1.2 1.2 0 0 1 0 2.4Zm3 0a1.2 1.2 0 1 1 0-2.4 1.2 1.2 0 0 1 0 2.4Zm3 0a1.2 1.2 0 1 1 0-2.4 1.2 1.2 0 0 1 0 2.4Z" />
        </svg>
      </button>
      {open ? (
        <section className={styles.panel} aria-label="AI assistant">
          <header className={styles.header}>
            <div>
              <h2>AI Assistant</h2>
              <span>{activeProfile ? `${activeProfile.display_name} · ${activeProfile.model || "system"}` : "No managed profile"}</span>
            </div>
            <button type="button" className={styles.iconButton} aria-label="Close" title="Close" onClick={() => setOpen(false)}>
              <svg aria-hidden="true" viewBox="0 0 24 24" focusable="false">
                <path d="m6.4 5 12.6 12.6-1.4 1.4L5 6.4 6.4 5Zm12.6 1.4L6.4 19 5 17.6 17.6 5 19 6.4Z" />
              </svg>
            </button>
          </header>

          <div className={styles.sessionBar}>
            <select
              value={selectedSessionId}
              onChange={(event) => loadSession(event.target.value).catch((loadError) => setError(loadError.message))}
              disabled={busy}
              aria-label="AI session"
            >
              <option value="">New chat</option>
              {sessions.map((session) => (
                <option key={session.session_id} value={session.session_id}>
                  {session.title || "New chat"}
                </option>
              ))}
            </select>
            <button type="button" className={styles.iconButton} aria-label="New session" title="New session" onClick={createSession} disabled={busy}>
              <svg aria-hidden="true" viewBox="0 0 24 24" focusable="false"><path d="M11 5h2v6h6v2h-6v6h-2v-6H5v-2h6V5Z" /></svg>
            </button>
            <button type="button" className={styles.iconButton} aria-label="Compact session" title="Compact session" onClick={compactSession} disabled={busy || !currentSession}>
              <svg aria-hidden="true" viewBox="0 0 24 24" focusable="false"><path d="M4 5h16v2H4V5Zm3 6h10v2H7v-2Zm3 6h4v2h-4v-2Z" /></svg>
            </button>
            <button type="button" className={styles.iconButton} aria-label="Delete session" title="Delete session" onClick={deleteSession} disabled={busy || !currentSession}>
              <svg aria-hidden="true" viewBox="0 0 24 24" focusable="false"><path d="M9 3h6l1 2h4v2H4V5h4l1-2Zm-2 6h10l-.7 11H7.7L7 9Z" /></svg>
            </button>
          </div>

          {!ready ? (
            <div className={styles.notice}>Enable AI and select an available managed provider in AI settings.</div>
          ) : null}
          {error ? <div className={styles.error}>{error}</div> : null}

          <div className={styles.timeline} ref={timelineRef}>
            {displayMessages.length === 0 ? (
              <div className={styles.empty}>Ask about the current table, records, schema, or validation diagnostics.</div>
            ) : displayMessages.map((message) => (
              <article key={message.id} className={`${styles.message} ${styles[message.role] ?? ""}`}>
                <div className={styles.messageMeta}>
                  <span>{message.role === "summary" ? "summary" : message.role}</span>
                  <time>{formatTime(message.created_at)}</time>
                </div>
                <p>{message.content}</p>
              </article>
            ))}
            {busy ? <div className={styles.running}>Running...</div> : null}
            {busy && activityEvents.length ? (
            <div className={styles.activityInline} aria-live="polite">
              {activityEvents.map((event, index) => (
                <div key={`${event.timestamp ?? ""}-${event.kind ?? ""}-${index}`} className={styles.activityItem}>
                  <strong>{eventTitle(event)}</strong>
                  <span>{eventSummary(event)}</span>
                </div>
              ))}
            </div>
            ) : null}
          </div>

          <form
            className={styles.composer}
            onSubmit={(event) => {
              event.preventDefault();
              send();
            }}
          >
            <textarea
              value={input}
              onChange={(event) => setInput(event.target.value)}
              placeholder="Ask about the current data..."
              rows={3}
              disabled={busy || !ready}
            />
            <button type="submit" disabled={busy || !ready || !input.trim()} aria-label="Send message" title="Send">
              <svg aria-hidden="true" viewBox="0 0 24 24" focusable="false">
                <path d="M3 20 21 12 3 4v6l10 2-10 2v6Z" />
              </svg>
            </button>
          </form>
        </section>
      ) : null}
    </>
  );
}

async function executeFrontendTool(request, handlers) {
  try {
    if (request?.name === "get_current_context") {
      const context = handlers.getCurrentContext?.() ?? {};
      return { result: { success: true, status: "ok", context } };
    }
    if (request?.name === "stage_table_changes") {
      if (!handlers.onStageTableChanges) {
        return { result: { success: false, status: "blocked", error: "No active table editor." } };
      }
      const stageResult = handlers.onStageTableChanges(request.arguments ?? {});
      const acceptedCount = stageResult?.accepted?.length ?? 0;
      const rejectedCount = stageResult?.rejected?.length ?? 0;
      const staged = Boolean(stageResult?.staged || acceptedCount > 0);
      return {
        result: {
          success: staged,
          status: staged ? "staged" : "rejected",
          summary: `Staged ${acceptedCount} operation(s); ${rejectedCount} rejected.`,
          ...stageResult
        }
      };
    }
    return { result: { success: false, status: "blocked", error: `Unknown frontend tool: ${request?.name ?? ""}` } };
  } catch (error) {
    return { error: error.message ?? "Frontend tool failed" };
  }
}

async function postFrontendToolResult(requestId, toolResult) {
  if (!requestId) return;
  const response = await fetch("/api/ai/frontend-tool-results", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      request_id: requestId,
      result: toolResult.result,
      error: toolResult.error
    })
  });
  if (!response.ok) {
    const payload = await response.json().catch(() => ({ error: "Failed to send frontend tool result" }));
    throw new Error(payload.error ?? "Failed to send frontend tool result");
  }
  await response.json().catch(() => ({}));
}

async function* readNDJSON(stream) {
  const reader = stream.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    let lineEnd = buffer.indexOf("\n");
    while (lineEnd >= 0) {
      const line = buffer.slice(0, lineEnd).trim();
      buffer = buffer.slice(lineEnd + 1);
      if (line) yield JSON.parse(line);
      lineEnd = buffer.indexOf("\n");
    }
  }
  buffer += decoder.decode();
  const line = buffer.trim();
  if (line) yield JSON.parse(line);
}
