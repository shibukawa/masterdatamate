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
  return field.type;
}

function isBlank(value) {
  return value === undefined || value === null || value === "";
}

export function storedValue(value) {
  if (value && typeof value === "object" && "value" in value) return value.value;
  return value;
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

function toExtableSchema(schema, referenceCandidates, generationAware) {
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
      width: field.type === "string" || field.type === "external_reference" ? 180 : 132,
      edit: field.type === "external_reference" ? { lookup: lookupConfigFor(field, referenceCandidates) } : undefined,
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

export const ExtableEditor = forwardRef(function ExtableEditor({ schema, rows, mode, activeGenerationId, referenceCandidates, onDirtyChange, onSelectionChange }, ref) {
  const extableRef = useRef(null);
  const generationAware = useMemo(() => rows.some((row) => row.sourceGenerationId), [rows]);
  const extableSchema = useMemo(() => toExtableSchema(schema, referenceCandidates, generationAware), [schema, referenceCandidates, generationAware]);
  const data = useMemo(() => displayRows(rows, schema, referenceCandidates), [rows, schema, referenceCandidates]);

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
    <div className={styles.extableSurface}>
      <Extable
        key={`${schema.table_id}-${mode}`}
        ref={extableRef}
        schema={extableSchema}
        defaultData={data}
        defaultView={{}}
        options={{ renderMode: mode, editMode: "commit", lockMode: "none", layoutDiagnostics: true }}
        onTableState={(state) => onDirtyChange(state.canCommit)}
        onCellEvent={(selection) => onSelectionChange(selection)}
      />
    </div>
  );
});
