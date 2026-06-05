import React from "react";
import { ExtableEditor } from "./ExtableEditor.jsx";
import styles from "./TableEditingPage.module.css";

export function TableEditingPage({
  editorRef,
  schema,
  rows,
  mode,
  selectedTable,
  editGenerationId,
  tableViewMode,
  referenceCandidates,
  status,
  diagnostics,
  dirty,
  saving,
  selection,
  onSetTableViewMode,
  onAddRow,
  onDeleteSelectedRow,
  onCommit,
  onSwitchMode,
  onDirtyChange,
  onSelectionChange
}) {
  return (
    <section className={styles.workspace}>
      <header className={styles.toolbar}>
        <div>
          <h1>{schema?.business_name ?? "Table"}</h1>
          <p>{schema?.comment ?? status}</p>
        </div>
        <div className={styles.actions}>
          <div className={styles.segmented} aria-label="Generation view mode">
            <button type="button" className={tableViewMode === "active_only" ? styles.selected : ""} onClick={() => onSetTableViewMode("active_only")}>Active only</button>
            <button type="button" className={tableViewMode === "include_previous" ? styles.selected : ""} onClick={() => onSetTableViewMode("include_previous")}>Include previous</button>
          </div>
          <div className={styles.segmented} aria-label="Grid mode">
            <button type="button" className={mode === "html" ? styles.selected : ""} onClick={() => onSwitchMode("html")}>HTML</button>
            <button type="button" className={mode === "canvas" ? styles.selected : ""} onClick={() => onSwitchMode("canvas")}>Canvas</button>
          </div>
          <button type="button" onClick={onAddRow} disabled={!schema}>Add</button>
          <button type="button" onClick={onDeleteSelectedRow} disabled={!schema || selection?.activeRowIndex == null}>Delete</button>
          <button type="button" className={styles.primary} onClick={() => onCommit(false)} disabled={!dirty || saving}>{saving ? "Saving" : "Commit"}</button>
        </div>
      </header>

      <div className={styles.gridWrap}>
        {schema ? (
          <ExtableEditor
            key={`${selectedTable}-${editGenerationId}-${tableViewMode}`}
            ref={editorRef}
            schema={schema}
            rows={rows}
            mode={mode}
            activeGenerationId={editGenerationId}
            referenceCandidates={referenceCandidates}
            onDirtyChange={onDirtyChange}
            onSelectionChange={onSelectionChange}
          />
        ) : (
          <div className={styles.emptyState}>{status}</div>
        )}
      </div>

      <footer className={styles.statusBar}>
        <span className={dirty ? styles.dirty : ""}>{dirty ? "Pending edits" : status}</span>
        <span>{diagnostics.length} diagnostics</span>
      </footer>
    </section>
  );
}
