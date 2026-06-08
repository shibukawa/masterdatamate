import React, { forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { Extable } from "@extable/react";
import styles from "./TableEditingPage.module.css";
import schemaStyles from "./SchemaEditingPage.module.css";

const SCOPES = ["project", "table", "record", "group"];
const FORMATTERS = ["", "gofmt"];

function normalizeDefinitionRow(row) {
  return {
    selected: Boolean(row.selected),
    id: row.id ?? "",
    name: row.name ?? "",
    enabled: row.enabled !== false,
    scope: row.scope || "table",
    table: row.table ?? "",
    group_field: row.group_field ?? "",
    template_file: row.template_file ?? "",
    output_path: row.output_path ?? "",
    formatter: row.formatter ?? "",
    comment: row.comment ?? ""
  };
}

function definitionError(row) {
  if (!row.id) return "id is required.";
  if (!/^[A-Za-z0-9_][A-Za-z0-9_-]*$/.test(row.id)) return "Use letters, numbers, underscore, and hyphen.";
  if (!row.output_path) return "output_path is required.";
  if (row.output_path.startsWith("/") || row.output_path.includes("..")) return "output_path must stay under the generation output root.";
  if (row.scope !== "project" && !row.table) return "table is required for this scope.";
  if (row.scope === "group" && !row.group_field) return "group_field is required for group scope.";
  if (!row.template_file) return "template_file is required.";
  return null;
}

function comparable(row) {
  const normalized = normalizeDefinitionRow(row);
  return JSON.stringify(normalized);
}

export const TemplateExportDefinitionGrid = forwardRef(function TemplateExportDefinitionGrid({
  rows,
  tableOptions,
  fieldOptions,
  onDirtyChange,
  onSelectionChange,
  onUndoRedoChange
}, ref) {
  const extableRef = useRef(null);
  const data = useMemo(() => rows.map(normalizeDefinitionRow), [rows]);
  const baseMetadata = useMemo(() => data.map(comparable), [data]);
  const allFields = useMemo(() => {
    const set = new Set();
    Object.values(fieldOptions ?? {}).forEach((fields) => fields.forEach((field) => set.add(field)));
    return [...set].sort();
  }, [fieldOptions]);
  const schema = useMemo(() => ({
    columns: [
      { key: "selected", header: "", type: "boolean", width: 48 },
      {
        key: "id",
        header: "ID",
        type: "string",
        unique: true,
        width: 190,
        conditionalStyle: (row) => {
          const message = definitionError(row);
          return message ? new Error(message) : null;
        }
      },
      { key: "name", header: "Name", type: "string", width: 180 },
      { key: "enabled", header: "Enabled", type: "boolean", width: 96 },
      { key: "scope", header: "Scope", type: "enum", enum: SCOPES, enumAllowCustom: false, width: 120 },
      { key: "table", header: "Table", type: "enum", enum: tableOptions, enumAllowCustom: false, width: 150 },
      { key: "group_field", header: "Group field", type: "enum", enum: allFields, enumAllowCustom: true, width: 150 },
      { key: "template_file", header: "Template file", type: "string", width: 260 },
      { key: "output_path", header: "Output path", type: "string", width: 260 },
      { key: "formatter", header: "Formatter", type: "enum", enum: FORMATTERS, enumAllowCustom: false, width: 120 },
      { key: "comment", header: "Comment", type: "string", width: 300 }
    ]
  }), [tableOptions, allFields]);

  function emitState(state) {
    const history = extableRef.current?.getUndoRedoHistory() ?? { undo: [], redo: [] };
    const tableData = extableRef.current?.getTableData() ?? data;
    const nextMetadata = tableData.map(comparable);
    const metadataDirty = nextMetadata.length !== baseMetadata.length || nextMetadata.some((value, index) => value !== baseMetadata[index]);
    onUndoRedoChange?.({ canUndo: history.undo.length > 0, canRedo: history.redo.length > 0 });
    onDirtyChange(Boolean(state.canCommit && metadataDirty));
  }

  useImperativeHandle(ref, () => ({
    insertBlank() {
      return extableRef.current?.insertRow(normalizeDefinitionRow({ id: "", enabled: true, scope: "table" })) ?? null;
    },
    deleteRow(target) {
      return extableRef.current?.deleteRow(target) ?? false;
    },
    getRow(target) {
      return extableRef.current?.getRow(target);
    },
    getRows() {
      return (extableRef.current?.getTableData() ?? data).map(normalizeDefinitionRow);
    },
    clearPending() {
      return extableRef.current?.commit() ?? Promise.resolve([]);
    },
    undo() {
      extableRef.current?.undo();
    },
    redo() {
      extableRef.current?.redo();
    }
  }), [data]);

  return (
    <div className={schemaStyles.grid}>
      <Extable
        ref={extableRef}
        schema={schema}
        defaultData={data}
        defaultView={{}}
        options={{ renderMode: "html", editMode: "commit", lockMode: "none", layoutDiagnostics: true }}
        onTableState={emitState}
        onCellEvent={(selection) => onSelectionChange?.(selection)}
      />
    </div>
  );
});

export function TemplateExportDefinitionPage({
  gridRef,
  rows,
  outputRoot,
  tableOptions,
  fieldOptions,
  dirty,
  saving,
  status,
  selection,
  undoRedo,
  onCreate,
  onDeleteSelected,
  onCheckSelected,
  onCommit,
  onRevert,
  onUndo,
  onRedo,
  onDirtyChange,
  onOutputRootChange,
  onSelectionChange,
  onUndoRedoChange
}) {
  return (
    <section className={styles.workspace}>
      <header className={styles.toolbar}>
        <div>
          <h1>Generate definitions</h1>
          <p>Edit Pongo2 template generation for generated text artifacts.</p>
        </div>
        <div className={styles.actions}>
          <button type="button" onClick={onUndo} disabled={!undoRedo.canUndo || saving}>Undo</button>
          <button type="button" onClick={onRedo} disabled={!undoRedo.canRedo || saving}>Redo</button>
          <button type="button" onClick={onRevert} disabled={!dirty || saving}>Revert</button>
          <button type="button" onClick={onCreate} disabled={saving}>New definition</button>
          <button type="button" onClick={onDeleteSelected} disabled={saving || !selection}>Delete</button>
          <button type="button" onClick={onCheckSelected} disabled={saving || dirty}>Check selected</button>
          <button type="button" className={styles.primary} onClick={onCommit} disabled={!dirty || saving}>{saving ? "Saving" : "Commit"}</button>
        </div>
      </header>
      <div className={styles.toolbar}>
        <label className={styles.outputRootField}>
          <span>Output root</span>
          <input
            value={outputRoot}
            onChange={(event) => onOutputRootChange?.(event.target.value)}
            placeholder="generated"
            disabled={saving}
          />
        </label>
      </div>
      <div className={styles.gridWrap}>
        <TemplateExportDefinitionGrid
          ref={gridRef}
          rows={rows}
          tableOptions={tableOptions}
          fieldOptions={fieldOptions}
          onDirtyChange={onDirtyChange}
          onSelectionChange={onSelectionChange}
          onUndoRedoChange={onUndoRedoChange}
        />
      </div>
      <footer className={styles.statusBar}>
        <span className={dirty ? styles.dirty : ""}>{dirty ? "Unsaved generate definition edits" : status}</span>
        <span>{rows.length} definitions</span>
      </footer>
    </section>
  );
}
