import React, { forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { Extable } from "@extable/react";
import styles from "./TableEditingPage.module.css";
import schemaStyles from "./SchemaEditingPage.module.css";

const FIELD_KINDS = ["primary_key", "reference", "data"];
const FIELD_TYPES = ["string", "integer", "decimal", "boolean", "date", "time", "datetime", "constant", "external_reference"];

function normalizeListRow(row) {
  return {
    selected: Boolean(row.selected),
    table_id: row.table_id || row.system_name,
    system_name: row.system_name ?? "",
    business_name: row.business_name ?? row.system_name ?? "",
    export: row.export !== false,
    primary_key: row.primary_key ?? "",
    references: row.references ?? "",
    comment: row.comment ?? ""
  };
}

function normalizeFieldRow(row) {
  const kind = row.kind === "formula" ? "data" : (row.kind || "data");
  return {
    id: row.id || row.original_system_name || row.system_name || crypto.randomUUID(),
    original_system_name: row.original_system_name || row.system_name || "",
    kind,
    system_name: row.system_name ?? "",
    business_name: row.business_name ?? row.system_name ?? "",
    type: kind === "reference" ? "external_reference" : (row.type || "string"),
    formula: row.formula ?? "",
    reference_table: row.reference_table ?? "",
    constants: row.constants ?? "",
    default_value: row.default_value ?? "",
    export: row.export !== false,
    required: Boolean(row.required),
    comment: row.comment ?? ""
  };
}

function schemaNameError(row) {
  if (!row.system_name) return "system_name is required.";
  if (!/^[A-Za-z0-9_][A-Za-z0-9_-]*$/.test(row.system_name)) return "Use letters, numbers, underscore, and hyphen.";
  return null;
}

function fieldNameError(row) {
  if (!row.system_name) return "system_name is required.";
  if (!/^[A-Za-z0-9_][A-Za-z0-9_-]*$/.test(row.system_name)) return "Use letters, numbers, underscore, and hyphen.";
  if (["key", "data", "children"].includes(row.system_name)) return "This name is reserved.";
  return null;
}

function isExternalReferenceRow(row) {
  return row.type === "external_reference" || row.kind === "reference";
}

function referenceTableError(row) {
  const hasReference = Boolean(row.reference_table);
  if (isExternalReferenceRow(row)) {
    return hasReference ? null : "reference_table is required for external_reference fields.";
  }
  return hasReference ? "reference_table is only valid for external_reference fields." : null;
}

function listComparable(row) {
  const normalized = normalizeListRow(row);
  return JSON.stringify({
    table_id: normalized.table_id,
    system_name: normalized.system_name,
    business_name: normalized.business_name,
    export: normalized.export,
    comment: normalized.comment
  });
}

function fieldComparable(row) {
  const normalized = normalizeFieldRow(row);
  return JSON.stringify({
    original_system_name: normalized.original_system_name,
    kind: normalized.kind,
    system_name: normalized.system_name,
    business_name: normalized.business_name,
    type: normalized.type,
    reference_table: normalized.reference_table,
    constants: normalized.constants,
    default_value: normalized.default_value,
    export: normalized.export,
    required: normalized.required,
    comment: normalized.comment
  });
}

export const SchemaListGrid = forwardRef(function SchemaListGrid({ rows, onDirtyChange, onSelectionChange, onUndoRedoChange }, ref) {
  const extableRef = useRef(null);
  const data = useMemo(() => rows.map(normalizeListRow), [rows]);
  const baseMetadata = useMemo(() => data.map(listComparable), [data]);
  const version = useMemo(() => baseMetadata.join("|"), [baseMetadata]);
  const schema = useMemo(() => ({
    columns: [
      { key: "selected", header: "", type: "boolean", width: 48 },
      {
        key: "system_name",
        header: "System name",
        type: "string",
        unique: true,
        width: 180,
        conditionalStyle: (row) => {
          const message = schemaNameError(row);
          return message ? new Error(message) : null;
        }
      },
      { key: "business_name", header: "Label", type: "string", width: 180 },
      { key: "export", header: "Output", type: "boolean", width: 92 },
      { key: "primary_key", header: "Primary key", type: "string", readonly: true, width: 180 },
      { key: "references", header: "References", type: "string", readonly: true, width: 180 },
      { key: "comment", header: "Comment", type: "string", width: 360 }
    ]
  }), []);

  function emitState(state) {
    const history = extableRef.current?.getUndoRedoHistory() ?? { undo: [], redo: [] };
    const tableData = extableRef.current?.getTableData() ?? data;
    const nextMetadata = tableData.map(listComparable);
    const metadataDirty = nextMetadata.length !== baseMetadata.length || nextMetadata.some((value, index) => value !== baseMetadata[index]);
    onUndoRedoChange?.({ canUndo: history.undo.length > 0, canRedo: history.redo.length > 0 });
    onDirtyChange(Boolean(state.canCommit && metadataDirty));
  }

  useImperativeHandle(ref, () => ({
    insertBlank() {
      return extableRef.current?.insertRow(normalizeListRow({ system_name: "", export: true })) ?? null;
    },
    deleteRow(target) {
      return extableRef.current?.deleteRow(target) ?? false;
    },
    getRow(target) {
      return extableRef.current?.getRow(target);
    },
    getRows() {
      return (extableRef.current?.getTableData() ?? data).map(normalizeListRow);
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
        key={version}
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

export const SchemaDetailGrid = forwardRef(function SchemaDetailGrid({ rows, tableOptions, onDirtyChange, onSelectionChange, onUndoRedoChange }, ref) {
  const extableRef = useRef(null);
  const data = useMemo(() => rows.map(normalizeFieldRow), [rows]);
  const referenceOptions = useMemo(() => ["", ...tableOptions], [tableOptions]);
  const baseMetadata = useMemo(() => data.map(fieldComparable), [data]);
  const version = useMemo(() => `${baseMetadata.join("|")}:${tableOptions.join("|")}`, [baseMetadata, tableOptions]);
  const schema = useMemo(() => ({
    columns: [
      { key: "selected", header: "", type: "boolean", width: 48 },
      { key: "kind", header: "Kind", type: "enum", enum: FIELD_KINDS, enumAllowCustom: false, width: 132 },
      {
        key: "system_name",
        header: "System name",
        type: "string",
        unique: true,
        width: 180,
        conditionalStyle: (row) => {
          const message = fieldNameError(row);
          return message ? new Error(message) : null;
        }
      },
      { key: "business_name", header: "Label", type: "string", width: 180 },
      { key: "type", header: "Type", type: "enum", enum: FIELD_TYPES, enumAllowCustom: false, width: 150 },
      tableOptions.length
        ? {
          key: "reference_table",
          header: "Reference",
          type: "enum",
          enum: referenceOptions,
          enumAllowCustom: false,
          nullable: true,
          readonly: (row) => !isExternalReferenceRow(row),
          width: 160,
          conditionalStyle: (row) => {
            const message = referenceTableError(row);
            return message ? new Error(message) : null;
          }
        }
        : {
          key: "reference_table",
          header: "Reference",
          type: "string",
          readonly: (row) => !isExternalReferenceRow(row),
          width: 160,
          conditionalStyle: (row) => {
            const message = referenceTableError(row);
            return message ? new Error(message) : null;
          }
        },
      { key: "constants", header: "Constants", type: "string", width: 220 },
      { key: "default_value", header: "Default", type: "string", width: 150 },
      { key: "export", header: "Output", type: "boolean", width: 92 },
      { key: "required", header: "Required", type: "boolean", width: 100 },
      { key: "comment", header: "Comment", type: "string", width: 320 },
      { key: "formula", header: "Formula", type: "string", readonly: true, width: 220 }
    ]
  }), [tableOptions, referenceOptions]);

  function emitState(state) {
    const history = extableRef.current?.getUndoRedoHistory() ?? { undo: [], redo: [] };
    const tableData = extableRef.current?.getTableData() ?? data;
    const nextMetadata = tableData.map(fieldComparable);
    const metadataDirty = nextMetadata.length !== baseMetadata.length || nextMetadata.some((value, index) => value !== baseMetadata[index]);
    onUndoRedoChange?.({ canUndo: history.undo.length > 0, canRedo: history.redo.length > 0 });
    onDirtyChange(Boolean(state.canCommit && metadataDirty));
  }

  useImperativeHandle(ref, () => ({
    insertBlank() {
      return extableRef.current?.insertRow(normalizeFieldRow({ kind: "data", type: "string", export: true, required: false })) ?? null;
    },
    deleteRow(target) {
      return extableRef.current?.deleteRow(target) ?? false;
    },
    getRow(target) {
      return extableRef.current?.getRow(target);
    },
    getRows() {
      return (extableRef.current?.getTableData() ?? data).map(normalizeFieldRow);
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
        key={version}
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

export function SchemaEditingPage({
  view,
  listRef,
  detailRef,
  schemaRows,
  detailSchema,
  fieldRows,
  tableOptions,
  dirty,
  saving,
  status,
  selection,
  undoRedo,
  onCreateSchema,
  onOpenDetail,
  onBackToList,
  onAddField,
  onDeleteSelected,
  onCommit,
  onRevert,
  onUndo,
  onRedo,
  onDirtyChange,
  onSelectionChange,
  onUndoRedoChange
}) {
  const isDetail = view === "detail";
  return (
    <section className={styles.workspace}>
      <header className={styles.toolbar}>
        <div>
          <h1>{isDetail ? `Schema: ${detailSchema?.system_name ?? ""}` : "Schemas"}</h1>
          <p>{isDetail ? (detailSchema?.comment || "Edit fields, keys, references, and defaults.") : "Edit schema metadata and open table definitions."}</p>
        </div>
        <div className={styles.actions}>
          {isDetail ? <button type="button" onClick={onBackToList} disabled={saving}>Schema list</button> : null}
          <button type="button" onClick={onUndo} disabled={!undoRedo.canUndo || saving}>Undo</button>
          <button type="button" onClick={onRedo} disabled={!undoRedo.canRedo || saving}>Redo</button>
          <button type="button" onClick={onRevert} disabled={!dirty || saving}>Revert</button>
          {isDetail ? (
            <button type="button" onClick={onAddField} disabled={saving}>Add field</button>
          ) : (
            <>
              <button type="button" onClick={onOpenDetail} disabled={saving || !selection}>Open</button>
              <button type="button" onClick={onCreateSchema} disabled={saving}>New schema</button>
            </>
          )}
          <button type="button" onClick={onDeleteSelected} disabled={saving || !selection}>Delete</button>
          <button type="button" className={styles.primary} onClick={onCommit} disabled={!dirty || saving}>{saving ? "Saving" : "Commit"}</button>
        </div>
      </header>
      <div className={styles.gridWrap}>
        {isDetail ? (
          <SchemaDetailGrid
            key={detailSchema?.table_id}
            ref={detailRef}
            rows={fieldRows}
            tableOptions={tableOptions}
            onDirtyChange={onDirtyChange}
            onSelectionChange={onSelectionChange}
            onUndoRedoChange={onUndoRedoChange}
          />
        ) : (
          <SchemaListGrid
            ref={listRef}
            rows={schemaRows}
            onDirtyChange={onDirtyChange}
            onSelectionChange={onSelectionChange}
            onUndoRedoChange={onUndoRedoChange}
          />
        )}
      </div>
      <footer className={styles.statusBar}>
        <span className={dirty ? styles.dirty : ""}>{dirty ? "Unsaved schema edits" : status}</span>
        <span>{isDetail ? `${fieldRows.length} fields` : `${schemaRows.length} schemas`}</span>
      </footer>
    </section>
  );
}
