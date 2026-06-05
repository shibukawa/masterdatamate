import React, { useEffect, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import "@extable/core/style.css";
import "./global.css";
import styles from "./App.module.css";
import { api } from "./api.js";
import { displayRows, normalizeRows } from "./ExtableEditor.jsx";
import { GenerationEditingPage } from "./GenerationEditingPage.jsx";
import { SchemaEditingPage } from "./SchemaEditingPage.jsx";
import { TableEditingPage } from "./TableEditingPage.jsx";
import { defaultOutputGenerationIds, displayGenerationName, nextEditGeneration, sortGenerations } from "./generationUtils.js";
import { EDIT_GENERATION_KEY, writeJsonStorage } from "./storage.js";

const TABLE_ROUTE = "/";
const GENERATION_ROUTE = "/generations/edit";
const SCHEMA_ROUTE = "/schemas";
const EXPORT_FORMATS = [
  ["csv_zip", "CSV ZIP"],
  ["excel_csv_zip", "Excel CSV (BOM) ZIP"],
  ["json_zip", "JSON ZIP"],
  ["yaml_zip", "YAML ZIP"],
  ["sql", "SQL"],
  ["xlsx", "Excel"],
  ["ndjson_zip", "NDJSON ZIP"],
  ["sqlite", "SQLite"]
];
const EXPORT_LOGICAL_FORMATS = {
  csv_zip: "csv",
  excel_csv_zip: "excel-csv",
  json_zip: "json",
  yaml_zip: "yaml",
  ndjson_zip: "ndjson",
  sql: "sql",
  xlsx: "xlsx",
  sqlite: "sqlite"
};
const DEFAULT_EXPORT_SETTINGS = { version: 1, formats: {} };
const DEFAULT_EXPORT_OPTIONS = { time_format: "iso", timezone: "" };

function pageFromPath(pathname) {
  if (pathname === GENERATION_ROUTE) return "generations";
  if (pathname === SCHEMA_ROUTE || pathname.startsWith(`${SCHEMA_ROUTE}/`)) return "schemas";
  return "tables";
}

function schemaTableFromPath(pathname) {
  const match = pathname.match(/^\/schemas\/([^/]+)\/edit$/);
  return match ? decodeURIComponent(match[1]) : "";
}

function ExportDialog({
  generations,
  generationSettings,
  selectedGenerationIds,
  format,
  options,
  destination,
  result,
  busy,
  onToggleGeneration,
  onSetFormat,
  onSetOptions,
  onSetDestination,
  onCheck,
  onExport,
  onClose
}) {
  return (
    <div className={styles.dialogBackdrop} role="presentation">
      <section className={styles.dialog} role="dialog" aria-modal="true" aria-labelledby="export-dialog-title">
        <header className={styles.dialogHeader}>
          <div>
            <h2 id="export-dialog-title">Export</h2>
            <p>Project export</p>
          </div>
          <button type="button" className={styles.iconButton} aria-label="Close export dialog" onClick={onClose} disabled={busy}>x</button>
        </header>

        <div className={styles.dialogBody}>
          <label className={styles.field}>
            <span>Output destination</span>
            <select value={destination} onChange={(event) => onSetDestination(event.target.value)} disabled={busy}>
              <option value="browser_download">Browser download</option>
            </select>
          </label>

          <label className={styles.field}>
            <span>Format</span>
            <select value={format} onChange={(event) => onSetFormat(event.target.value)} disabled={busy}>
              {EXPORT_FORMATS.map(([value, label]) => (
                <option key={value} value={value}>{label}</option>
              ))}
            </select>
          </label>

          <label className={styles.field}>
            <span>Time format</span>
            <select value={options.time_format} onChange={(event) => onSetOptions({ ...options, time_format: event.target.value })} disabled={busy}>
              <option value="iso">ISO</option>
              <option value="epoch-sec">Epoch seconds</option>
              <option value="epoch-ms">Epoch milliseconds</option>
              <option value="iso-local">ISO (local timezone)</option>
            </select>
          </label>

          <label className={styles.field}>
            <span>Timezone</span>
            <input
              value={options.timezone}
              onChange={(event) => onSetOptions({ ...options, timezone: event.target.value })}
              placeholder="Asia/Tokyo"
              disabled={busy}
            />
          </label>

          <div className={styles.field}>
            <span>Generations</span>
            <div className={styles.dialogCheckboxList}>
              {generations.map((generation) => (
                <label key={generation.id} className={styles.checkboxRow}>
                  <input
                    type="checkbox"
                    checked={selectedGenerationIds.includes(generation.id)}
                    onChange={(event) => onToggleGeneration(generation.id, event.target.checked)}
                    disabled={busy}
                  />
                  <span>{displayGenerationName(generation, generationSettings)}</span>
                </label>
              ))}
            </div>
          </div>

          {result ? (
            <div className={result.exportable === false ? styles.exportResultError : styles.exportResultOk}>
              {result.exportable === false
                ? `${result.diagnostics?.length ?? 0} export issue(s)`
                : `${result.summary?.recordCount ?? 0} record(s) ready`}
            </div>
          ) : null}

          {result?.diagnostics?.length ? (
            <div className={styles.exportDiagnostics}>
              {result.diagnostics.slice(0, 6).map((diagnostic, index) => (
                <span key={`${diagnostic.table ?? "export"}-${diagnostic.field ?? ""}-${index}`}>
                  {diagnostic.table ? `${diagnostic.table}: ` : ""}{diagnostic.field ? `${diagnostic.field}: ` : ""}{diagnostic.message}
                </span>
              ))}
              {result.diagnostics.length > 6 ? <span>{result.diagnostics.length - 6} more issue(s)</span> : null}
            </div>
          ) : null}
        </div>

        <footer className={styles.dialogActions}>
          <button type="button" onClick={onClose} disabled={busy}>Cancel</button>
          <button type="button" onClick={onCheck} disabled={busy}>{busy ? "Checking" : "Check"}</button>
          <button type="button" className={styles.primary} onClick={onExport} disabled={busy}>{busy ? "Exporting" : "Export"}</button>
        </footer>
      </section>
    </div>
  );
}

function App() {
  const editorRef = useRef(null);
  const generationEditorRef = useRef(null);
  const schemaListRef = useRef(null);
  const schemaDetailRef = useRef(null);
  const [tables, setTables] = useState([]);
  const [generations, setGenerations] = useState([]);
  const [generationSettings, setGenerationSettings] = useState({ ordering_mode: "numeric", numeric_digits: 4 });
  const [generationDrafts, setGenerationDrafts] = useState([]);
  const [outputGenerationIds, setOutputGenerationIds] = useState([]);
  const [editGenerationId, setEditGenerationId] = useState("");
  const [tableViewMode, setTableViewMode] = useState("active_only");
  const [page, setPage] = useState(pageFromPath(window.location.pathname));
  const [schemaView, setSchemaView] = useState(schemaTableFromPath(window.location.pathname) ? "detail" : "list");
  const [schemaRows, setSchemaRows] = useState([]);
  const [schemaDetail, setSchemaDetail] = useState(null);
  const [schemaFieldRows, setSchemaFieldRows] = useState([]);
  const [schemaTableOptions, setSchemaTableOptions] = useState([]);
  const [schemaDirty, setSchemaDirty] = useState(false);
  const [schemaSaving, setSchemaSaving] = useState(false);
  const [schemaSelection, setSchemaSelection] = useState(null);
  const [schemaUndoRedo, setSchemaUndoRedo] = useState({ canUndo: false, canRedo: false });
  const [selectedTable, setSelectedTable] = useState("");
  const [schema, setSchema] = useState(null);
  const [rows, setRows] = useState([]);
  const [diagnostics, setDiagnostics] = useState([]);
  const [referenceCandidates, setReferenceCandidates] = useState({});
  const [mode, setMode] = useState("html");
  const [status, setStatus] = useState("Loading tables...");
  const [saving, setSaving] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [generationDirty, setGenerationDirty] = useState(false);
  const [generationInvalid, setGenerationInvalid] = useState(false);
  const [generationGridVersion, setGenerationGridVersion] = useState(0);
  const [generationSaving, setGenerationSaving] = useState(false);
  const [selectedGenerationIds, setSelectedGenerationIds] = useState([]);
  const [analysisResult, setAnalysisResult] = useState(null);
  const [selection, setSelection] = useState(null);
  const [exportFormat, setExportFormat] = useState("csv_zip");
  const [exportSettings, setExportSettings] = useState(DEFAULT_EXPORT_SETTINGS);
  const [exportOptions, setExportOptions] = useState(DEFAULT_EXPORT_OPTIONS);
  const [exportResult, setExportResult] = useState(null);
  const [exportBusy, setExportBusy] = useState(false);
  const [exportDialogOpen, setExportDialogOpen] = useState(false);
  const [exportDialogGenerationIds, setExportDialogGenerationIds] = useState([]);
  const [exportDestination, setExportDestination] = useState("browser_download");

  useEffect(() => {
    api("/api/tables").then(({ payload }) => {
      setTables(payload.tables);
      setSelectedTable(payload.tables[0]?.table_id ?? "");
    }).catch((error) => setStatus(error.message));
  }, []);

  useEffect(() => {
    if (page !== "schemas") return;
    loadSchemaList().catch((error) => setStatus(error.message));
  }, [page]);

  useEffect(() => {
    if (page !== "schemas") return;
    const table = schemaTableFromPath(window.location.pathname);
    if (!table) {
      setSchemaView("list");
      return;
    }
    openSchemaDetail(table, false).catch((error) => setStatus(error.message));
  }, [page]);

  useEffect(() => {
    loadGenerations().catch((error) => setStatus(error.message));
  }, []);

  useEffect(() => {
    if (editGenerationId) writeJsonStorage(EDIT_GENERATION_KEY, editGenerationId);
  }, [editGenerationId]);

  useEffect(() => {
    if (!selectedTable || !editGenerationId) return;
    setStatus("Loading table...");
    setSchema(null);
    setDirty(false);
    setSelection(null);
    api(`/api/tables/${selectedTable}/generation-view?activeGenerationId=${encodeURIComponent(editGenerationId)}&mode=${encodeURIComponent(tableViewMode)}`).then(async ({ payload }) => {
      const references = {};
      for (const field of payload.schema.fields) {
        const target = field.reference?.table;
        if (!target || references[target]) continue;
        const { payload: referencePayload } = await api(`/api/tables/${target}/references?activeGenerationId=${encodeURIComponent(editGenerationId)}&mode=${encodeURIComponent(tableViewMode)}`);
        const candidates = referencePayload.candidates.map((candidate) => {
          const value = candidateKeyValue(candidate.key);
          const displayName = candidate.label || value;
          return {
            label: `${displayName} (${value})`,
            value,
            meta: {
              displayName,
              sourceGenerationId: candidate.sourceGenerationId,
              sourceGenerationLabel: candidate.sourceGenerationLabel,
              overrodeGenerationIds: candidate.overrodeGenerationIds ?? []
            }
          };
        });
        references[target] = {
          candidates,
          ids: new Set(candidates.map((candidate) => candidate.value)),
          labelByValue: new Map(candidates.map((candidate) => [candidate.value, candidate.meta.displayName]))
        };
      }
      setReferenceCandidates(references);
      setSchema(payload.schema);
      setRows(displayRows(payload.rows, payload.schema, references));
      setDiagnostics(payload.diagnostics ?? []);
      setStatus(`Loaded ${payload.schema.business_name} in ${editGenerationId} (${tableViewMode}).`);
    }).catch((error) => setStatus(error.message));
  }, [selectedTable, editGenerationId, tableViewMode]);

  function candidateKeyValue(key) {
    return typeof key === "object" && key !== null ? JSON.stringify(key) : String(key);
  }

  async function loadGenerations(preserveEditId = editGenerationId) {
    const { payload } = await api("/api/generations");
    const sorted = sortGenerations(payload.generations);
    setGenerationSettings(payload.settings);
    setGenerations(sorted);
    setGenerationDrafts(sorted.map((generation) => ({ ...generation })));
    setOutputGenerationIds((current) => {
      const validIds = new Set(sorted.map((generation) => generation.id));
      const kept = current.filter((id) => validIds.has(id));
      return kept.length ? kept : defaultOutputGenerationIds(sorted);
    });
    setEditGenerationId(nextEditGeneration(sorted, preserveEditId));
    setGenerationDirty(false);
    setGenerationInvalid(false);
    setSelectedGenerationIds((current) => current.filter((id) => sorted.some((generation) => generation.id === id)));
    setGenerationGridVersion((version) => version + 1);
  }

  function confirmTableSwitch() {
    if (!dirty) return true;
    return window.confirm("Pending table edits exist. Switch without saving?");
  }

  function confirmGenerationSwitch() {
    if (!generationDirty) return true;
    const confirmed = window.confirm("Unsaved generation edits exist. Discard them and leave?");
    if (confirmed) resetGenerationDrafts();
    return confirmed;
  }

  function confirmSchemaSwitch() {
    if (!schemaDirty) return true;
    return window.confirm("Unsaved schema edits exist. Discard them and leave?");
  }

  function confirmNavigation(nextPage = page) {
    if (page === "tables" && nextPage !== "tables") return confirmTableSwitch();
    if (page === "generations" && nextPage !== "generations") return confirmGenerationSwitch();
    if (page === "schemas" && nextPage !== "schemas") return confirmSchemaSwitch();
    return true;
  }

  function selectTable(tableId) {
    if (tableId === selectedTable) return;
    if (!confirmTableSwitch()) return;
    setSelectedTable(tableId);
  }

  function selectEditGeneration(generationId) {
    if (generationId === editGenerationId) return;
    if (!confirmTableSwitch()) return;
    setEditGenerationId(generationId);
  }

  function navigate(nextPage) {
    if (nextPage === page) return;
    if (!confirmNavigation(nextPage)) return;
    const path = nextPage === "generations" ? GENERATION_ROUTE : (nextPage === "schemas" ? SCHEMA_ROUTE : TABLE_ROUTE);
    window.history.pushState({ page: nextPage }, "", path);
    setPage(nextPage);
    if (nextPage === "schemas") setSchemaView("list");
  }

  useEffect(() => {
    window.history.replaceState({ page }, "", window.location.pathname);
    const handlePopState = () => {
      const nextPage = pageFromPath(window.location.pathname);
      if (nextPage === page) return;
      if (!confirmNavigation(nextPage)) {
        const currentPath = page === "generations" ? GENERATION_ROUTE : (page === "schemas" ? (schemaView === "detail" && schemaDetail ? `${SCHEMA_ROUTE}/${encodeURIComponent(schemaDetail.table_id)}/edit` : SCHEMA_ROUTE) : TABLE_ROUTE);
        window.history.pushState({ page }, "", currentPath);
        return;
      }
      setPage(nextPage);
      if (nextPage === "schemas") setSchemaView(schemaTableFromPath(window.location.pathname) ? "detail" : "list");
    };
    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, [page, dirty, generationDirty, schemaDirty, schemaView, schemaDetail]);

  async function loadTables() {
    const { payload } = await api("/api/tables");
    setTables(payload.tables);
    setSelectedTable((current) => payload.tables.some((table) => table.table_id === current) ? current : (payload.tables[0]?.table_id ?? ""));
  }

  async function loadSchemaList() {
    const { payload } = await api("/api/schemas");
    setSchemaRows(payload.rows ?? []);
    setSchemaTableOptions((payload.schemas ?? []).map((item) => item.table_id));
    setSchemaDirty(false);
    setSchemaSelection(null);
    setSchemaUndoRedo({ canUndo: false, canRedo: false });
    setStatus("Schema list loaded.");
  }

  async function openSchemaDetail(tableId = "", push = true) {
    if (schemaDirty && !confirmSchemaSwitch()) return;
    const target = tableId || selectedSchemaRow()?.table_id || selectedSchemaRow()?.system_name;
    if (!target) return;
    const { payload } = await api(`/api/schemas/${encodeURIComponent(target)}`);
    setSchemaDetail(payload.schema);
    setSchemaFieldRows(payload.fieldRows ?? []);
    setSchemaTableOptions(payload.tables ?? []);
    setSchemaView("detail");
    setSchemaDirty(false);
    setSchemaSelection(null);
    setSchemaUndoRedo({ canUndo: false, canRedo: false });
    if (push) window.history.pushState({ page: "schemas" }, "", `${SCHEMA_ROUTE}/${encodeURIComponent(target)}/edit`);
    setStatus(`Loaded schema ${target}.`);
  }

  function selectedSchemaRow() {
    const rows = schemaListRef.current?.getRows() ?? schemaRows;
    const selected = rows.find((row) => row.selected);
    if (selected) return selected;
    const target = schemaSelection?.activeRowKey ?? schemaSelection?.activeRowIndex;
    return target === undefined || target === null ? null : schemaListRef.current?.getRow(target);
  }

  function resetSchemaDraft() {
    if (schemaView === "detail" && schemaDetail) {
      openSchemaDetail(schemaDetail.table_id, false).catch((error) => setStatus(error.message));
    } else {
      loadSchemaList().catch((error) => setStatus(error.message));
    }
  }

  function revertSchema() {
    if (!schemaDirty) return;
    if (!window.confirm("Discard pending schema edits?")) return;
    resetSchemaDraft();
  }

  function createSchema() {
    schemaListRef.current?.insertBlank();
    setSchemaDirty(true);
  }

  function addSchemaField() {
    schemaDetailRef.current?.insertBlank();
    setSchemaDirty(true);
  }

  function deleteSelectedSchemaItem() {
    if (schemaView === "list") return deleteSelectedSchemas();
    return deleteSelectedSchemaFields();
  }

  async function deleteSelectedSchemas() {
    const rows = schemaListRef.current?.getRows() ?? [];
    const selected = rows.filter((row) => row.selected && row.table_id);
    if (!selected.length) {
      const row = selectedSchemaRow();
      if (row?.table_id) selected.push(row);
    }
    if (!selected.length) {
      setStatus("Select schema rows to delete.");
      return;
    }
    if (!window.confirm(`Delete schema files only?\n\n${selected.map((row) => row.table_id).join("\n")}`)) return;
    setSchemaSaving(true);
    try {
      const { payload } = await api("/api/schemas/delete", {
        method: "POST",
        body: JSON.stringify({ tableIds: selected.map((row) => row.table_id) })
      });
      setSchemaRows(payload.rows ?? []);
      await loadTables();
      setSchemaDirty(false);
      setStatus(`Deleted ${payload.deletedTableIds.length} schema file(s).`);
    } catch (error) {
      setStatus(error.message);
    } finally {
      setSchemaSaving(false);
    }
  }

  function deleteSelectedSchemaFields() {
    const rows = schemaDetailRef.current?.getRows() ?? [];
    const selected = rows.filter((row) => row.selected);
    if (!selected.length) {
      const target = schemaSelection?.activeRowKey ?? schemaSelection?.activeRowIndex;
      if (schemaDetailRef.current?.deleteRow(target)) {
        setSchemaDirty(true);
        return;
      }
      setStatus("Select field rows to delete.");
      return;
    }
    if (!window.confirm(`Delete selected fields and remove their data columns on commit?\n\n${selected.map((row) => row.system_name).join("\n")}`)) return;
    for (const row of selected) {
      const index = (schemaDetailRef.current?.getRows() ?? []).findIndex((item) => item.id === row.id || item.system_name === row.system_name);
      if (index >= 0) schemaDetailRef.current?.deleteRow(index);
    }
    setSchemaDirty(true);
  }

  function schemaCommitSummary() {
    if (schemaView === "list") {
      const rows = schemaListRef.current?.getRows() ?? [];
      const byId = new Map(schemaRows.map((row) => [row.table_id, row]));
      return rows.flatMap((row) => {
        const current = byId.get(row.table_id);
        if (!current) return [`Create blank schema ${row.system_name || "(missing system_name)"}`];
        const changes = [];
        if (current.system_name !== row.system_name) changes.push(`rename ${current.system_name} -> ${row.system_name}`);
        if (Boolean(current.export) !== Boolean(row.export)) changes.push(`output ${current.export} -> ${Boolean(row.export)}`);
        if ((current.comment ?? "") !== (row.comment ?? "")) changes.push("comment");
        return changes.length ? [`Update ${current.table_id}: ${changes.join(", ")}`] : [];
      });
    }
    const rows = schemaDetailRef.current?.getRows() ?? [];
    const currentByName = new Map(schemaFieldRows.map((row) => [row.system_name, row]));
    return rows.flatMap((row) => {
      const current = currentByName.get(row.original_system_name || row.system_name);
      if (!current) return [`Create field ${row.system_name || "(missing system_name)"}`];
      const changes = [];
      if (current.system_name !== row.system_name) changes.push(`rename ${current.system_name} -> ${row.system_name}`);
      if (current.kind !== row.kind) changes.push(`kind ${current.kind} -> ${row.kind}`);
      if (current.type !== row.type) changes.push(`type ${current.type} -> ${row.type}`);
      if (current.reference_table !== row.reference_table) changes.push(`reference ${current.reference_table || "(none)"} -> ${row.reference_table || "(none)"}`);
      if (String(current.default_value ?? "") !== String(row.default_value ?? "")) changes.push("default");
      return changes.length ? [`Update field ${current.system_name}: ${changes.join(", ")}`] : [];
    });
  }

  async function commitSchema(force = false) {
    const summary = schemaCommitSummary();
    if (!force && !window.confirm(`Commit schema changes?\n\n${summary.length ? summary.join("\n") : "No structural summary available."}`)) return;
    setSchemaSaving(true);
    try {
      if (schemaView === "list") {
        const { status: responseStatus, payload } = await api("/api/schemas", {
          method: "PUT",
          body: JSON.stringify({ rows: schemaListRef.current?.getRows() ?? [], force })
        });
        if (responseStatus === 409 && payload.requiresForce) {
          setSchemaSaving(false);
          if (window.confirm(`Schema validation warnings exist. Save anyway?\n\n${payload.diagnostics.map((item) => item.message).join("\n")}`)) return commitSchema(true);
          return;
        }
        setSchemaRows(payload.rows ?? []);
      } else {
        const { status: responseStatus, payload } = await api(`/api/schemas/${encodeURIComponent(schemaDetail.table_id)}`, {
          method: "PUT",
          body: JSON.stringify({ schema: schemaDetail, fields: schemaDetailRef.current?.getRows() ?? [], force })
        });
        if (responseStatus === 409 && payload.requiresForce) {
          setSchemaSaving(false);
          if (window.confirm(`Schema validation warnings exist. Save anyway?\n\n${payload.diagnostics.map((item) => item.message).join("\n")}`)) return commitSchema(true);
          return;
        }
        setSchemaDetail(payload.schema);
        setSchemaFieldRows(payload.fieldRows ?? []);
        window.history.replaceState({ page: "schemas" }, "", `${SCHEMA_ROUTE}/${encodeURIComponent(payload.schema.table_id)}/edit`);
      }
      await loadTables();
      await loadSchemaList();
      setSchemaDirty(false);
      setSchemaUndoRedo({ canUndo: false, canRedo: false });
      setStatus("Schema changes committed.");
    } catch (error) {
      setStatus(error.message);
    } finally {
      setSchemaSaving(false);
    }
  }

  async function createGeneration() {
    setGenerationSaving(true);
    try {
      const { payload } = await api("/api/generations", { method: "POST", body: JSON.stringify({}) });
      const sorted = sortGenerations(payload.generations);
      const nextId = sorted[sorted.length - 1]?.id ?? "";
      setGenerationSettings(payload.settings);
      setGenerations(sorted);
      setGenerationDrafts(sorted.map((generation) => ({ ...generation })));
      setOutputGenerationIds(defaultOutputGenerationIds(sorted));
      setEditGenerationId(nextId);
      setGenerationDirty(false);
      setGenerationInvalid(false);
      setSelectedGenerationIds([nextId]);
      setAnalysisResult(null);
      setGenerationGridVersion((version) => version + 1);
      setStatus(`Created generation ${nextId}.`);
    } catch (error) {
      setStatus(error.message);
    } finally {
      setGenerationSaving(false);
    }
  }

  function resetGenerationDrafts() {
    setGenerationDrafts(sortGenerations(generations).map((generation) => ({ ...generation })));
    setGenerationDirty(false);
    setGenerationInvalid(false);
    setAnalysisResult(null);
    setGenerationGridVersion((version) => version + 1);
  }

  function revertGenerations() {
    if (!generationDirty) return;
    if (!window.confirm("Discard pending generation edits?")) return;
    resetGenerationDrafts();
    setStatus("Generation edits reverted.");
  }

  async function commitGenerations() {
    if (generationInvalid) {
      setStatus("Fix generation metadata errors before committing.");
      return;
    }
    const rows = generationEditorRef.current?.getRows() ?? generationDrafts;
    const currentEditId = editGenerationId;
    const preserveId = rows.find((row) => row.id === currentEditId)?.folder_name ?? currentEditId;
    const summary = generationCommitSummary(rows);
    if (!summary.length) {
      setStatus("No generation edits to commit.");
      return;
    }
    if (!window.confirm(`Commit generation metadata changes?\n\n${summary.join("\n")}`)) {
      setStatus("Generation commit cancelled.");
      return;
    }
    setGenerationSaving(true);
    try {
      let latest = null;
      for (const row of rows) {
        const { payload } = await api(`/api/generations/${row.id}/config`, {
          method: "PUT",
          body: JSON.stringify({
            config: {
              generation_index: generationSettings.ordering_mode === "numeric" ? Number(row.generation_index) : String(row.generation_index),
              output: Boolean(row.output),
              path_name: row.path_name,
              description: row.description ?? ""
            }
          })
        });
        latest = payload;
      }
      const sorted = sortGenerations(latest.generations);
      setGenerationSettings(latest.settings);
      setGenerations(sorted);
      setGenerationDrafts(sorted.map((generation) => ({ ...generation })));
      setOutputGenerationIds(defaultOutputGenerationIds(sorted));
      setEditGenerationId(nextEditGeneration(sorted, preserveId));
      setSelectedGenerationIds((current) => current.filter((id) => sorted.some((generation) => generation.id === id)));
      setAnalysisResult(null);
      await generationEditorRef.current?.clearPending();
      setGenerationDirty(false);
      setGenerationInvalid(false);
      setGenerationGridVersion((version) => version + 1);
      setStatus("Generation metadata committed.");
    } catch (error) {
      setStatus(error.message);
    } finally {
      setGenerationSaving(false);
    }
  }

  function generationCommitSummary(rows) {
    const currentById = new Map(generations.map((generation) => [generation.id, generation]));
    const lines = [];
    for (const row of rows) {
      const current = currentById.get(row.id);
      if (!current) {
        lines.push(`Create ${row.folder_name}`);
        continue;
      }
      const changes = [];
      if (String(current.generation_index) !== String(row.generation_index)) changes.push(`index ${current.generation_index} -> ${row.generation_index}`);
      if (Boolean(current.output) !== Boolean(row.output)) changes.push(`output ${current.output} -> ${Boolean(row.output)}`);
      if (current.path_name !== row.path_name) changes.push(`path ${current.path_name} -> ${row.path_name}`);
      if ((current.description ?? "") !== (row.description ?? "")) changes.push("description");
      if (current.id !== row.folder_name) changes.push(`folder ${current.id} -> ${row.folder_name}`);
      if (changes.length) lines.push(`Update ${current.id}: ${changes.join(", ")}`);
    }
    return lines;
  }

  function selectedGenerations() {
    return selectedGenerationIds
      .map((id) => generations.find((generation) => generation.id === id))
      .filter(Boolean);
  }

  function updateGenerationSelection(ids) {
    setSelectedGenerationIds((current) => {
      if (current.length === ids.length && current.every((id, index) => id === ids[index])) return current;
      return ids;
    });
  }

  function updateOutputGenerationSelection(generationId, selected) {
    setExportDialogGenerationIds((current) => {
      const next = selected ? [...new Set([...current, generationId])] : current.filter((id) => id !== generationId);
      return next.length ? next : current;
    });
    setOutputGenerationIds((current) => {
      const next = selected ? [...new Set([...current, generationId])] : current.filter((id) => id !== generationId);
      return next.length ? next : current;
    });
    setExportResult(null);
  }

  function nextGenerationIndex() {
    if (generationSettings.ordering_mode === "release_date") return new Date().toISOString().slice(0, 10);
    const maxIndex = generations.reduce((max, generation) => {
      const value = Number(generation.generation_index);
      return Number.isFinite(value) ? Math.max(max, value) : max;
    }, -10);
    return String(maxIndex + 10);
  }

  function promptDestination(action, defaultPathName, defaultOutput = true, defaultDescription = "") {
    const generationIndex = window.prompt(`${action}: destination generation_index`, nextGenerationIndex());
    if (generationIndex === null) return null;
    const pathName = window.prompt(`${action}: destination path_name`, defaultPathName);
    if (pathName === null) return null;
    const outputText = window.prompt(`${action}: output? Enter true or false`, String(defaultOutput));
    if (outputText === null) return null;
    const description = window.prompt(`${action}: description`, defaultDescription);
    if (description === null) return null;
    return {
      generation_index: generationSettings.ordering_mode === "numeric" ? Number(generationIndex) : String(generationIndex),
      path_name: pathName,
      output: outputText.trim().toLowerCase() !== "false",
      description
    };
  }

  function requireCleanGenerationSelection(min, max = Infinity) {
    if (generationDirty) {
      setStatus("Save or revert generation edits before using generation actions.");
      return false;
    }
    if (selectedGenerationIds.length < min || selectedGenerationIds.length > max) {
      setStatus(`Select ${max === min ? min : `${min}-${max}`} generation(s).`);
      return false;
    }
    return true;
  }

  async function applyGenerationPayload(payload, preserveId = editGenerationId, selectedIds = []) {
    const sorted = sortGenerations(payload.generations);
    setGenerationSettings(payload.settings);
    setGenerations(sorted);
    setGenerationDrafts(sorted.map((generation) => ({ ...generation })));
    setOutputGenerationIds(defaultOutputGenerationIds(sorted));
    setEditGenerationId(nextEditGeneration(sorted, preserveId));
    setSelectedGenerationIds(selectedIds.filter((id) => sorted.some((generation) => generation.id === id)));
    setGenerationDirty(false);
    setGenerationInvalid(false);
    setGenerationGridVersion((version) => version + 1);
  }

  async function mergeGenerations() {
    if (!requireCleanGenerationSelection(2)) return;
    const selected = selectedGenerations();
    const destination = promptDestination("Merge generations", "merged_generation", true, `Merged ${selected.map((generation) => generation.id).join(", ")}`);
    if (!destination) return;
    if (!window.confirm(`Create merged generation ${destination.path_name} from:\n\n${selected.map((generation) => generation.id).join("\n")}`)) {
      setStatus("Generation merge cancelled.");
      return;
    }
    setGenerationSaving(true);
    try {
      const { payload } = await api("/api/generations/persistent-merge", {
        method: "POST",
        body: JSON.stringify({ sourceGenerationIds: selectedGenerationIds, destination })
      });
      await applyGenerationPayload(payload, payload.generationId, []);
      setAnalysisResult(null);
      setStatus(`Merged generations into ${payload.generationId}.`);
    } catch (error) {
      setStatus(error.message);
    } finally {
      setGenerationSaving(false);
    }
  }

  async function deleteGenerations() {
    if (!requireCleanGenerationSelection(1)) return;
    const selected = selectedGenerations();
    if (!window.confirm(`Delete selected generation folders?\n\n${selected.map((generation) => `${displayGenerationName(generation, generationSettings)} (${generation.id})`).join("\n")}`)) {
      setStatus("Generation delete cancelled.");
      return;
    }
    setGenerationSaving(true);
    try {
      const { payload } = await api("/api/generations/delete", {
        method: "POST",
        body: JSON.stringify({ generationIds: selectedGenerationIds, activeGenerationId: editGenerationId })
      });
      await applyGenerationPayload(payload, payload.resolvedActiveGenerationId, []);
      setAnalysisResult(null);
      setStatus(`Deleted ${payload.deletedGenerationIds.length} generation(s).`);
    } catch (error) {
      setStatus(error.message);
    } finally {
      setGenerationSaving(false);
    }
  }

  async function duplicateGeneration() {
    if (!requireCleanGenerationSelection(1)) return;
    setGenerationSaving(true);
    try {
      const { payload } = await api("/api/generations/duplicate", {
        method: "POST",
        body: JSON.stringify({ sourceGenerationIds: selectedGenerationIds })
      });
      const createdIds = payload.createdGenerationIds ?? payload.generationIds ?? [payload.generationId].filter(Boolean);
      await applyGenerationPayload(payload, createdIds[0] ?? editGenerationId, []);
      setAnalysisResult(null);
      setStatus(`Duplicated ${createdIds.length} generation(s).`);
    } catch (error) {
      setStatus(error.message);
    } finally {
      setGenerationSaving(false);
    }
  }

  async function analyzeGenerations() {
    if (!requireCleanGenerationSelection(1)) return;
    setGenerationSaving(true);
    try {
      const { payload } = await api("/api/generations/analyze", {
        method: "POST",
        body: JSON.stringify({ generationIds: selectedGenerationIds, includeMergeImpact: true })
      });
      setAnalysisResult(payload);
      setStatus(`Analyzed ${payload.summary.generationCount} generation(s), ${payload.summary.recordCount} records.`);
    } catch (error) {
      setStatus(error.message);
    } finally {
      setGenerationSaving(false);
    }
  }

  function exportGenerationIds(sourceIds = outputGenerationIds) {
    const validIds = new Set(generations.map((generation) => generation.id));
    const ids = sourceIds.filter((id) => validIds.has(id));
    return ids.length ? ids : defaultOutputGenerationIds(generations);
  }

  function openExportDialog() {
    const generationIds = defaultOutputGenerationIds(generations);
    setExportDialogGenerationIds(generationIds);
    setOutputGenerationIds(generationIds);
    setExportResult(null);
    setExportDialogOpen(true);
    loadExportSettings(exportFormat).catch((error) => setStatus(error.message));
  }

  function logicalExportFormat(format) {
    return EXPORT_LOGICAL_FORMATS[format] ?? format;
  }

  function optionsForFormat(format, settings = exportSettings) {
    const saved = settings.formats?.[logicalExportFormat(format)] ?? {};
    return {
      ...DEFAULT_EXPORT_OPTIONS,
      time_format: saved.time_format || DEFAULT_EXPORT_OPTIONS.time_format,
      timezone: saved.timezone || ""
    };
  }

  async function loadExportSettings(format = exportFormat) {
    const { payload } = await api("/api/export-settings");
    const settings = payload.settings ?? DEFAULT_EXPORT_SETTINGS;
    setExportSettings(settings);
    setExportOptions(optionsForFormat(format, settings));
  }

  async function persistExportSettings(format = exportFormat, options = exportOptions) {
    const logicalFormat = logicalExportFormat(format);
    const nextSettings = {
      version: exportSettings.version || 1,
      formats: {
        ...(exportSettings.formats ?? {}),
        [logicalFormat]: {
          time_format: options.time_format || DEFAULT_EXPORT_OPTIONS.time_format,
          ...(options.timezone ? { timezone: options.timezone } : {})
        }
      }
    };
    const { payload } = await api("/api/export-settings", {
      method: "PUT",
      body: JSON.stringify(nextSettings)
    });
    const saved = payload.settings ?? nextSettings;
    setExportSettings(saved);
    return nextSettings.formats[logicalFormat];
  }

  async function checkExport(sourceIds = exportDialogGenerationIds, format = exportFormat) {
    const generationIds = exportGenerationIds(sourceIds);
    if (!generationIds.length) {
      setStatus("Select at least one generation for export.");
      return null;
    }
    setExportBusy(true);
    try {
      const savedOptions = await persistExportSettings(format, exportOptions);
      const { payload } = await api("/api/exports/check", {
        method: "POST",
        body: JSON.stringify({ generationIds, format, options: { ...savedOptions, destination: exportDestination } })
      });
      setExportResult(payload);
      setOutputGenerationIds(generationIds);
      setStatus(payload.exportable ? `Export check passed for ${payload.summary.recordCount} record(s).` : `Export blocked by ${payload.diagnostics.length} diagnostic(s).`);
      return payload;
    } catch (error) {
      setStatus(error.message);
      return null;
    } finally {
      setExportBusy(false);
    }
  }

  async function runExport() {
    const check = await checkExport(exportDialogGenerationIds, exportFormat);
    if (!check?.exportable) return;
    setExportBusy(true);
    try {
      const { payload } = await api("/api/exports", {
        method: "POST",
        body: JSON.stringify({ generationIds: check.generationIds, format: exportFormat, options: { ...exportOptions, destination: exportDestination } })
      });
      setExportResult({ ...payload, exportable: true });
      window.location.assign(payload.downloadUrl);
      setStatus(`Export created: ${payload.filename}.`);
    } catch (error) {
      setStatus(error.message);
    } finally {
      setExportBusy(false);
    }
  }

  function addRow() {
    editorRef.current?.insertRow();
    setDirty(true);
  }

  function switchMode(nextMode) {
    if (schema && editorRef.current?.hasPendingChanges()) {
      setRows(normalizeRows(editorRef.current.getRows(), selectedTable));
    }
    setMode(nextMode);
  }

  function deleteSelectedRow() {
    const target = selection?.activeRowKey ?? selection?.activeRowIndex;
    const row = editorRef.current?.getRow(target);
    if (row?.isReadOnly) {
      setStatus("Readonly generation rows cannot be deleted.");
      return;
    }
    if (editorRef.current?.deleteRow(target)) setDirty(true);
  }

  async function commit(force = false) {
    setSaving(true);
    const nextRows = editorRef.current?.getRows() ?? [];
    const { status: responseStatus, payload } = await api(`/api/tables/${selectedTable}/generations/${editGenerationId}/records/commit`, {
      method: "POST",
      body: JSON.stringify({ rows: nextRows, force, mode: tableViewMode })
    });
    setDiagnostics(payload.diagnostics ?? []);
    if (responseStatus === 409 && payload.requiresForce) {
      const confirmed = window.confirm("Validation errors exist. Save anyway?");
      setSaving(false);
      if (confirmed) return commit(true);
      setStatus("Save cancelled because validation errors remain.");
      return;
    }
    const committedRows = normalizeRows(payload.rows, selectedTable);
    setRows(displayRows(committedRows, schema, referenceCandidates));
    await editorRef.current?.clearPending();
    setDirty(false);
    setStatus(payload.diagnostics?.length ? "Saved with validation diagnostics." : "Saved.");
    setSaving(false);
  }

  return (
    <main className={styles.appShell}>
      <aside className={styles.sidebar}>
        <div className={styles.brand}>
          <strong>MasterDataMate</strong>
        </div>
        {page === "tables" ? (
          <>
            <div className={styles.sidebarTop}>
              <div className={styles.sideSection}>
                <label htmlFor="edit-generation">Edit generation</label>
                <div className={styles.generationControl}>
                  <select id="edit-generation" value={editGenerationId} onChange={(event) => selectEditGeneration(event.target.value)}>
                    {generations.map((generation) => (
                      <option key={generation.id} value={generation.id}>{displayGenerationName(generation, generationSettings)}</option>
                    ))}
                  </select>
                  <button
                    type="button"
                    className={styles.iconButton}
                    aria-label="Edit generations"
                    title="Edit generations"
                    onClick={() => navigate("generations")}
                  >
                    <svg aria-hidden="true" viewBox="0 0 24 24" focusable="false">
                      <path d="M4 20h4.7L19.4 9.3a2 2 0 0 0 0-2.8l-1.9-1.9a2 2 0 0 0-2.8 0L4 15.3V20Zm2-3.9L16.1 6l1.9 1.9L7.9 18H6v-1.9Z" />
                    </svg>
                  </button>
                </div>
              </div>
            </div>
            <nav className={styles.nav} aria-label="Tables">
              {tables.map((table) => (
                <button
                  type="button"
                  key={table.table_id}
                  className={table.table_id === selectedTable ? styles.active : ""}
                  onClick={() => selectTable(table.table_id)}
                >
                  <span>{table.business_name}</span>
                  <small>{table.table_id}</small>
                </button>
              ))}
            </nav>
            <div className={styles.sidebarBottom}>
              <div className={styles.sideSection}>
                <button
                  type="button"
                  className={styles.sidebarAction}
                  aria-label="Export project"
                  title="Export project"
                  onClick={openExportDialog}
                >
                  <svg aria-hidden="true" viewBox="0 0 24 24" focusable="false">
                    <path d="M12 3a1 1 0 0 1 1 1v8.6l2.3-2.3 1.4 1.4-4.7 4.7-4.7-4.7 1.4-1.4 2.3 2.3V4a1 1 0 0 1 1-1ZM5 17h2v2h10v-2h2v3a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1v-3Z" />
                  </svg>
                  <span>Export</span>
                </button>
              </div>
              <div className={styles.sideSection}>
                <button
                  type="button"
                  className={styles.sidebarAction}
                  aria-label="Edit schemas"
                  title="Edit schemas"
                  onClick={() => navigate("schemas")}
                >
                  <svg aria-hidden="true" viewBox="0 0 24 24" focusable="false">
                    <path d="M4 5.5A2.5 2.5 0 0 1 6.5 3h11A2.5 2.5 0 0 1 20 5.5v13A2.5 2.5 0 0 1 17.5 21h-11A2.5 2.5 0 0 1 4 18.5v-13Zm2 0v13c0 .28.22.5.5.5h11a.5.5 0 0 0 .5-.5v-13a.5.5 0 0 0-.5-.5h-11a.5.5 0 0 0-.5.5Zm2 2h8v2H8v-2Zm0 4h8v2H8v-2Zm0 4h5v2H8v-2Z" />
                  </svg>
                  <span>Edit schemas</span>
                </button>
              </div>
            </div>
          </>
        ) : (
          <button type="button" className={styles.backButton} onClick={() => navigate("tables")}>Back to data</button>
        )}
      </aside>

      {exportDialogOpen ? (
        <ExportDialog
          generations={generations}
          generationSettings={generationSettings}
          selectedGenerationIds={exportGenerationIds(exportDialogGenerationIds)}
          format={exportFormat}
          options={exportOptions}
          destination={exportDestination}
          result={exportResult}
          busy={exportBusy}
          onToggleGeneration={updateOutputGenerationSelection}
          onSetFormat={(format) => {
            setExportFormat(format);
            setExportOptions(optionsForFormat(format));
            setExportResult(null);
          }}
          onSetOptions={(options) => {
            setExportOptions(options);
            setExportResult(null);
          }}
          onSetDestination={(destination) => {
            setExportDestination(destination);
            setExportResult(null);
          }}
          onCheck={() => checkExport(exportDialogGenerationIds, exportFormat)}
          onExport={runExport}
          onClose={() => setExportDialogOpen(false)}
        />
      ) : null}

      {page === "generations" ? (
        <GenerationEditingPage
          generationEditorRef={generationEditorRef}
          generationGridVersion={generationGridVersion}
          generationSettings={generationSettings}
          generationDrafts={generationDrafts}
          selectedGenerationIds={selectedGenerationIds}
          analysisResult={analysisResult}
          generationDirty={generationDirty}
          generationInvalid={generationInvalid}
          generationSaving={generationSaving}
          onCreateGeneration={createGeneration}
          onCommitGenerations={commitGenerations}
          onRevertGenerations={revertGenerations}
          onGenerationSelectionChange={updateGenerationSelection}
          onMergeGenerations={mergeGenerations}
          onDeleteGenerations={deleteGenerations}
          onDuplicateGeneration={duplicateGeneration}
          onAnalyzeGenerations={analyzeGenerations}
          onCloseAnalysis={() => setAnalysisResult(null)}
          onGenerationValidityChange={setGenerationInvalid}
          onGenerationDirtyChange={setGenerationDirty}
        />
      ) : page === "schemas" ? (
        <SchemaEditingPage
          view={schemaView}
          listRef={schemaListRef}
          detailRef={schemaDetailRef}
          schemaRows={schemaRows}
          detailSchema={schemaDetail}
          fieldRows={schemaFieldRows}
          tableOptions={schemaTableOptions}
          dirty={schemaDirty}
          saving={schemaSaving}
          status={status}
          selection={schemaSelection}
          undoRedo={schemaUndoRedo}
          onCreateSchema={createSchema}
          onOpenDetail={() => openSchemaDetail()}
          onBackToList={() => {
            if (!confirmSchemaSwitch()) return;
            window.history.pushState({ page: "schemas" }, "", SCHEMA_ROUTE);
            setSchemaView("list");
          }}
          onAddField={addSchemaField}
          onDeleteSelected={deleteSelectedSchemaItem}
          onCommit={() => commitSchema(false)}
          onRevert={revertSchema}
          onUndo={() => (schemaView === "detail" ? schemaDetailRef.current?.undo() : schemaListRef.current?.undo())}
          onRedo={() => (schemaView === "detail" ? schemaDetailRef.current?.redo() : schemaListRef.current?.redo())}
          onDirtyChange={setSchemaDirty}
          onSelectionChange={setSchemaSelection}
          onUndoRedoChange={setSchemaUndoRedo}
        />
      ) : (
        <TableEditingPage
        editorRef={editorRef}
        schema={schema}
        rows={rows}
        mode={mode}
        selectedTable={selectedTable}
        editGenerationId={editGenerationId}
        tableViewMode={tableViewMode}
        referenceCandidates={referenceCandidates}
        status={status}
        diagnostics={diagnostics}
        dirty={dirty}
        saving={saving}
        selection={selection}
        onSetTableViewMode={setTableViewMode}
        onAddRow={addRow}
        onDeleteSelectedRow={deleteSelectedRow}
        onCommit={commit}
        onSwitchMode={switchMode}
        onDirtyChange={setDirty}
        onSelectionChange={setSelection}
      />
      )}
    </main>
  );
}

createRoot(document.getElementById("root")).render(<App />);
