import React, { forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { Extable } from "@extable/react";
import styles from "./ExtableEditor.module.css";

function blankRow(schema, activeGenerationId) {
  return {
    id: crypto.randomUUID(),
    sourceGenerationId: activeGenerationId,
    sourceGenerationLabel: activeGenerationId,
    isActiveGeneration: true,
    isReadOnly: false,
    isOverridden: false,
    overriddenByGenerationId: "",
    status: "Editable",
    ...Object.fromEntries(schema.fields.map((field) => [field.system_name, field.type === "boolean" ? false : ""]))
  };
}

function toExtableColumnType(field) {
  if (field.type === "integer") return "int";
  if (field.type === "decimal") return "number";
  if (field.type === "constant") return "enum";
  if (field.type === "external_reference") return "labeled";
  if (field.type === "binary_file") return "labeled";
  return field.type;
}

function isBlank(value) {
  return value === undefined || value === null || value === "";
}

export function storedValue(value) {
  if (value && typeof value === "object" && "value" in value) return value.value;
  return value;
}

function binaryLabel(value) {
  const metadata = storedValue(value);
  if (!metadata || typeof metadata !== "object") return "";
  const originalName = metadata.original_name || metadata.originalName;
  const extension = metadata.extension ? `.${metadata.extension}` : "";
  return originalName || `Uploaded${extension}`;
}

function validateMasterDataCell(field, row, referenceCandidates) {
  const value = storedValue(row[field.system_name]);
  if (field.required && isBlank(value)) return `${field.business_name} is required.`;
  if (isBlank(value)) return null;
  if (field.type === "integer" && !Number.isInteger(Number(value))) return `${field.business_name} must be an integer.`;
  if (field.type === "decimal" && Number.isNaN(Number(value))) return `${field.business_name} must be a number.`;
  if (field.type === "constant" && field.constants && !field.constants.includes(value)) {
    return `${field.business_name} must be one of: ${field.constants.join(", ")}.`;
  }
  if (field.type === "external_reference" && field.reference?.table) {
    const candidates = referenceCandidates[field.reference.table]?.ids;
    if (candidates && !candidates.has(String(value))) {
      return `${field.business_name} references missing ${field.reference.table} key: ${value}.`;
    }
  }
  if (field.type === "binary_file") {
    if (!value || typeof value !== "object") return `${field.business_name} must be uploaded.`;
    const extension = String(value.extension ?? "").toLowerCase();
    const allowedExtensions = field.binary?.allowed_extensions ?? field.allowed_extensions ?? [];
    if (allowedExtensions.length && !allowedExtensions.map((item) => String(item).toLowerCase()).includes(extension)) {
      return `${field.business_name} must use one of: ${allowedExtensions.join(", ")}.`;
    }
    const maxSize = field.binary?.max_size_bytes ?? field.max_size_bytes;
    if (maxSize && Number(value.size_bytes ?? 0) > Number(maxSize)) {
      return `${field.business_name} exceeds ${maxSize} bytes.`;
    }
  }
  return null;
}

function lookupConfigFor(field, referenceCandidates) {
  const reference = field.reference?.table ? referenceCandidates[field.reference.table] : null;
  if (!reference) return undefined;
  return {
    candidates: async ({ query }) => {
      const normalizedQuery = query.trim().toLowerCase();
      return reference.candidates.filter((candidate) => {
        if (!normalizedQuery) return true;
        return candidate.label.toLowerCase().includes(normalizedQuery) || candidate.value.toLowerCase().includes(normalizedQuery);
      });
    },
    toStoredValue: (candidate) => ({
      label: candidate.meta?.displayName ?? candidate.label,
      value: candidate.value,
      meta: candidate.meta
    }),
    allowFreeInput: false,
    recentLookup: true,
    debounceMs: 0
  };
}

function binaryEditConfigFor(field, onBinaryUpload, schema) {
  if (!onBinaryUpload) return undefined;
  return {
    externalEditor: async ({ rowId }) => {
      const result = await onBinaryUpload({ rowId, field, schema });
      if (!result) return { kind: "cancel" };
      return {
        kind: "commit",
        value: {
          label: binaryLabel(result),
          value: result,
          meta: result
        }
      };
    }
  };
}

function toExtableSchema(schema, referenceCandidates, generationAware, onBinaryUpload) {
  const generationColumns = generationAware ? [
    {
      key: "sourceGenerationLabel",
      header: "Generation",
      type: "string",
      readonly: true,
      width: 210
    },
    {
      key: "status",
      header: "Status",
      type: "string",
      readonly: true,
      width: 180
    }
  ] : [];

  return {
    columns: [
      ...generationColumns,
      ...schema.fields.map((field) => ({
      key: field.system_name,
      header: field.business_name,
      type: toExtableColumnType(field),
      readonly: (row) => Boolean(field.formula) || Boolean(row.isReadOnly),
      unique: schema.primary_key.includes(field.system_name),
      enum: field.constants,
      enumAllowCustom: false,
      nullable: !field.required,
      width: field.type === "string" || field.type === "external_reference" || field.type === "binary_file" ? 180 : 132,
      edit: field.type === "external_reference"
        ? { lookup: lookupConfigFor(field, referenceCandidates) }
        : (field.type === "binary_file" ? binaryEditConfigFor(field, onBinaryUpload, schema) : undefined),
      conditionalStyle: (row) => {
        const message = validateMasterDataCell(field, row, referenceCandidates);
        return message ? new Error(message) : null;
      }
    }))
    ]
  };
}

export function normalizeRows(rows, tableId) {
  return rows.map(({ __clientId, id, ...row }, index) => ({
    id: id ?? __clientId ?? `${tableId}-${index}-${crypto.randomUUID()}`,
    ...row
  }));
}

export function displayRows(rows, schema, referenceCandidates) {
  return normalizeRows(rows, schema.table_id).map((row) => {
    const next = {
      ...row,
      _readonly: Boolean(row.isReadOnly),
      sourceGenerationLabel: row.sourceGenerationLabel ?? row.sourceGenerationId ?? "",
      status: row.isOverridden ? `Overridden by ${row.overriddenByGenerationId}` : (row.isReadOnly ? "Readonly" : "Editable")
    };
    for (const field of schema.fields) {
      if (field.type !== "external_reference" || !field.reference?.table) continue;
      const value = storedValue(next[field.system_name]);
      if (isBlank(value)) continue;
      const reference = referenceCandidates[field.reference.table];
      const displayName = reference?.labelByValue?.get(String(value)) ?? String(value);
      next[field.system_name] = {
        label: displayName,
        value: String(value),
        meta: { displayName }
      };
    }
    for (const field of schema.fields) {
      if (field.type !== "binary_file") continue;
      const value = storedValue(next[field.system_name]);
      if (isBlank(value)) continue;
      next[field.system_name] = {
        label: binaryLabel(value),
        value,
        meta: value
      };
    }
    return next;
  });
}

function cleanRows(rows) {
  return rows.map(({ id, __clientId, ...row }) => {
    const next = {};
    for (const [key, value] of Object.entries(row)) next[key] = storedValue(value);
    return next;
  });
}

export const ExtableEditor = forwardRef(function ExtableEditor({ schema, rows, mode, activeGenerationId, referenceCandidates, onDirtyChange, onSelectionChange, onBinaryUpload }, ref) {
  const extableRef = useRef(null);
  const fileInputRef = useRef(null);
  const binaryTargetRef = useRef(null);
  const openingTargetRef = useRef("");
  const generationAware = useMemo(() => rows.some((row) => row.sourceGenerationId), [rows]);
  const extableSchema = useMemo(() => toExtableSchema(schema, referenceCandidates, generationAware, openBinaryPicker), [schema, referenceCandidates, generationAware]);
  const data = useMemo(() => displayRows(rows, schema, referenceCandidates), [rows, schema, referenceCandidates]);

  function binaryFields() {
    return schema.fields.filter((field) => field.type === "binary_file");
  }

  function cleanRow(row) {
    const next = {};
    for (const [key, value] of Object.entries(row ?? {})) next[key] = storedValue(value);
    return next;
  }

  function rowKey(row) {
    const cleaned = cleanRow(row);
    if (schema.primary_key.length === 1) return cleaned[schema.primary_key[0]] ?? "";
    return Object.fromEntries(schema.primary_key.map((field) => [field, cleaned[field] ?? ""]));
  }

  function setBinaryCell(rowIdOrIndex, fieldName, metadata) {
    extableRef.current?.setCellValue(rowIdOrIndex, fieldName, {
      label: binaryLabel(metadata),
      value: metadata,
      meta: metadata
    });
    onDirtyChange(true);
  }

  async function uploadFileToTarget(file, target) {
    if (!file || !target || !onBinaryUpload) return null;
    const row = extableRef.current?.getRow(target.rowId ?? target.rowIndex);
    if (!row || row.isReadOnly) return null;
    const metadata = await onBinaryUpload({
      schema,
      field: target.field,
      row: cleanRow(row),
      recordKey: rowKey(row),
      file
    });
    if (metadata) setBinaryCell(target.rowId ?? target.rowIndex, target.field.system_name, metadata);
    return metadata;
  }

  function openBinaryPicker({ rowId, field }) {
    if (!onBinaryUpload) return Promise.resolve(null);
    return new Promise((resolve) => {
      binaryTargetRef.current = { rowId, field, resolve };
      fileInputRef.current?.click();
    });
  }

  async function handleFileInput(event) {
    const file = event.target.files?.[0];
    const target = binaryTargetRef.current;
    event.target.value = "";
    binaryTargetRef.current = null;
    try {
      const metadata = await uploadFileToTarget(file, target);
      target?.resolve?.(metadata);
    } catch (error) {
      target?.resolve?.(null);
    }
  }

  function targetFromSelection(selection) {
    const field = binaryFields().find((item) => item.system_name === selection?.activeColumnKey);
    if (!field) return null;
    return { rowId: selection.activeRowKey, rowIndex: selection.activeRowIndex, field };
  }

  function handleDrop(event) {
    const file = event.dataTransfer?.files?.[0];
    if (!file) return;
    const selection = extableRef.current?.getSelectionSnapshot();
    let target = targetFromSelection(selection);
    if (!target && selection?.activeRowKey) {
      const fields = binaryFields();
      if (fields.length === 1) target = { rowId: selection.activeRowKey, rowIndex: selection.activeRowIndex, field: fields[0] };
    }
    if (!target) return;
    event.preventDefault();
    uploadFileToTarget(file, target).catch(() => {});
  }

  useImperativeHandle(ref, () => ({
    insertRow() {
      return extableRef.current?.insertRow(blankRow(schema, activeGenerationId)) ?? null;
    },
    deleteRow(rowIdOrIndex) {
      if (rowIdOrIndex === null || rowIdOrIndex === undefined) return false;
      return extableRef.current?.deleteRow(rowIdOrIndex) ?? false;
    },
    getRows() {
      return cleanRows(extableRef.current?.getTableData() ?? data);
    },
    getRow(rowIdOrIndex) {
      return extableRef.current?.getRow(rowIdOrIndex) ?? null;
    },
    clearPending() {
      return extableRef.current?.commit() ?? Promise.resolve([]);
    },
    hasPendingChanges() {
      return extableRef.current?.hasPendingChanges() ?? false;
    }
  }), [data, schema]);

  return (
    <div
      className={styles.extableSurface}
      onDragOver={(event) => {
        if (event.dataTransfer?.types?.includes("Files")) event.preventDefault();
      }}
      onDrop={handleDrop}
    >
      <input ref={fileInputRef} className={styles.hiddenFileInput} type="file" onChange={handleFileInput} />
      <Extable
        key={`${schema.table_id}-${mode}`}
        ref={extableRef}
        schema={extableSchema}
        defaultData={data}
        defaultView={{}}
        options={{ renderMode: mode, editMode: "commit", lockMode: "none", layoutDiagnostics: true }}
        onTableState={(state) => onDirtyChange(state.canCommit)}
        onCellEvent={(selection, previous, reason) => {
          onSelectionChange(selection);
          const target = targetFromSelection(selection);
          const targetKey = target ? `${target.rowId ?? target.rowIndex}:${target.field.system_name}` : "";
          if (target && reason === "selection" && targetKey !== openingTargetRef.current) {
            openingTargetRef.current = targetKey;
            setTimeout(() => {
              binaryTargetRef.current = { ...target };
              fileInputRef.current?.click();
            }, 0);
          }
          if (!target) openingTargetRef.current = "";
        }}
      />
    </div>
  );
});
