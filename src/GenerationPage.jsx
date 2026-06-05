import React, { forwardRef, useCallback, useImperativeHandle, useMemo, useRef } from "react";
import { Extable } from "@extable/react";
import styles from "./GenerationPage.module.css";

function toGenerationRow(generation, selectedGenerationIds) {
  return {
    selected: selectedGenerationIds.includes(generation.id),
    id: generation.id,
    generationId: generation.id,
    output: Boolean(generation.output),
    generation_index: generation.generation_index,
    path_name: generation.path_name,
    description: generation.description ?? ""
  };
}

function normalizeGenerationRow(row, settings) {
  const generation = {
    id: row.generationId ?? row.id,
    generation_index: settings.ordering_mode === "numeric" ? Number(row.generation_index) : String(row.generation_index),
    output: Boolean(row.output),
    path_name: row.path_name,
    description: row.description ?? ""
  };
  return {
    ...generation,
    selected: Boolean(row.selected)
  };
}

function metadataComparable(row) {
  return JSON.stringify({
    id: row.id,
    generation_index: row.generation_index,
    output: Boolean(row.output),
    path_name: row.path_name,
    description: row.description ?? ""
  });
}

function validateCell(column, row, settings) {
  if (column === "generation_index") {
    if (settings.ordering_mode === "numeric" && !Number.isFinite(Number(row.generation_index))) return "Index must be a number.";
    if (settings.ordering_mode === "release_date" && !row.generation_index) return "Index date is required.";
  }
  if (column === "path_name") {
    if (!row.path_name) return "Path is required.";
    if (!/^[A-Za-z0-9_-]+$/.test(String(row.path_name))) return "Path may contain only letters, numbers, underscore, and hyphen.";
  }
  return null;
}

export const GenerationPage = forwardRef(function GenerationPage({
  generationSettings,
  generationDrafts,
  selectedGenerationIds,
  selectionDisabled,
  onSelectionChange,
  onValidityChange,
  onDirtyChange
}, ref) {
  const extableRef = useRef(null);
  const data = useMemo(
    () => generationDrafts.map((generation) => toGenerationRow(generation, selectedGenerationIds)),
    [generationDrafts, selectedGenerationIds]
  );
  const baseMetadata = useMemo(
    () => data.map((row) => metadataComparable(normalizeGenerationRow(row, generationSettings))),
    [data, generationSettings]
  );
  const schema = useMemo(() => ({
    columns: [
      { key: "selected", header: "", type: "boolean", width: 48 },
      {
        key: "generation_index",
        header: "Index",
        type: generationSettings.ordering_mode === "numeric" ? "int" : "date",
        unique: true,
        width: 120,
        conditionalStyle: (row) => {
          const message = validateCell("generation_index", row, generationSettings);
          return message ? new Error(message) : null;
        }
      },
      {
        key: "path_name",
        header: "Path",
        type: "string",
        width: 180,
        conditionalStyle: (row) => {
          const message = validateCell("path_name", row, generationSettings);
          return message ? new Error(message) : null;
        }
      },
      { key: "output", header: "Output", type: "boolean", width: 96 },
      { key: "description", header: "Description", type: "string", width: 360 }
    ]
  }), [generationSettings]);

  const handleTableState = useCallback((state) => {
    const tableData = extableRef.current?.getTableData() ?? data;
    const selected = tableData.filter((row) => row.selected).map((row) => row.generationId ?? row.id);
    onSelectionChange(selected);
    const normalizedRows = tableData.map((row) => normalizeGenerationRow(row, generationSettings));
    const nextMetadata = normalizedRows.map((row) => metadataComparable(row));
    const metadataDirty = nextMetadata.length !== baseMetadata.length || nextMetadata.some((value, index) => value !== baseMetadata[index]);
    const indexCounts = new Map();
    for (const row of normalizedRows) {
      const key = String(row.generation_index);
      indexCounts.set(key, (indexCounts.get(key) ?? 0) + 1);
    }
    const invalid = normalizedRows.some((row) => (
      validateCell("generation_index", row, generationSettings)
      || validateCell("path_name", row, generationSettings)
      || indexCounts.get(String(row.generation_index)) > 1
    ));
    onValidityChange(invalid);
    onDirtyChange(Boolean(state.canCommit && metadataDirty));
  }, [baseMetadata, data, generationSettings, onDirtyChange, onSelectionChange, onValidityChange]);

  const handleCellEvent = useCallback((selection, _previous, reason) => {
    if (selectionDisabled || reason !== "selection") return;
    const column = selection?.activeColumnKey ?? selection?.columnKey ?? selection?.key;
    if (column !== "selected") return;
    const target = selection.activeRowKey ?? selection.activeRowIndex;
    const row = extableRef.current?.getRow(target);
    const generationId = row?.generationId ?? row?.id;
    if (!generationId) return;
    const next = selectedGenerationIds.includes(generationId)
      ? selectedGenerationIds.filter((id) => id !== generationId)
      : [...selectedGenerationIds, generationId];
    onSelectionChange(next);
  }, [onSelectionChange, selectedGenerationIds, selectionDisabled]);

  useImperativeHandle(ref, () => ({
    getRows() {
      return (extableRef.current?.getTableData() ?? data).map((row) => {
        const { selected, ...generation } = normalizeGenerationRow(row, generationSettings);
        return generation;
      });
    },
    clearPending() {
      return extableRef.current?.commit() ?? Promise.resolve([]);
    },
    hasPendingChanges() {
      return extableRef.current?.hasPendingChanges() ?? false;
    }
  }), [data, generationSettings]);

  return (
    <section className={styles.generationPanel}>
      <div className={styles.generationPanelHead}>
        <div>
          <h2>Generations</h2>
          <p>{generationSettings.ordering_mode} ordering</p>
        </div>
      </div>
      <div className={styles.generationGrid}>
        <Extable
          ref={extableRef}
          schema={schema}
          defaultData={data}
          defaultView={{}}
          options={{ renderMode: "html", editMode: "commit", lockMode: "none", layoutDiagnostics: true }}
          onTableState={handleTableState}
          onCellEvent={handleCellEvent}
        />
      </div>
    </section>
  );
});
