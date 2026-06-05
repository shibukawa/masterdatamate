import React from "react";
import { GenerationPage } from "./GenerationPage.jsx";
import styles from "./TableEditingPage.module.css";
import generationStyles from "./GenerationPage.module.css";

export function GenerationEditingPage({
  generationEditorRef,
  generationGridVersion,
  generationSettings,
  generationDrafts,
  selectedGenerationIds,
  analysisResult,
  generationDirty,
  generationInvalid,
  generationSaving,
  onCreateGeneration,
  onCommitGenerations,
  onRevertGenerations,
  onGenerationSelectionChange,
  onMergeGenerations,
  onDeleteGenerations,
  onDuplicateGeneration,
  onAnalyzeGenerations,
  onCloseAnalysis,
  onGenerationValidityChange,
  onGenerationDirtyChange
}) {
  const selectedCount = selectedGenerationIds.length;
  const hasSelection = selectedCount > 0;
  const canUseSelection = !generationDirty && !generationSaving;

  return (
    <section className={styles.workspace}>
      <header className={styles.toolbar}>
        <div>
          <h1>Generations</h1>
          <p>Create, reorder, rename, and configure generation metadata.</p>
        </div>
        <div className={styles.actions}>
          <button type="button" onClick={onRevertGenerations} disabled={!generationDirty || generationSaving}>Revert</button>
          <button type="button" onClick={onAnalyzeGenerations} disabled={!hasSelection || !canUseSelection}>Analyze</button>
          <button type="button" onClick={onDuplicateGeneration} disabled={!hasSelection || !canUseSelection}>Duplicate</button>
          <button type="button" onClick={onMergeGenerations} disabled={selectedCount < 2 || !canUseSelection}>Merge</button>
          <button type="button" onClick={onDeleteGenerations} disabled={!hasSelection || !canUseSelection}>Delete</button>
          <button type="button" className={styles.primary} onClick={onCreateGeneration} disabled={generationSaving}>New generation</button>
          <button type="button" className={styles.primary} onClick={onCommitGenerations} disabled={!generationDirty || generationInvalid || generationSaving}>{generationSaving ? "Saving" : "Commit"}</button>
        </div>
      </header>

      <div className={styles.gridWrap}>
        <div className={`${generationStyles.managementLayout} ${analysisResult ? "" : generationStyles.managementLayoutCompact}`}>
          <GenerationPage
            key={`${generationGridVersion}:${selectedGenerationIds.join("|")}`}
            ref={generationEditorRef}
            generationSettings={generationSettings}
            generationDrafts={generationDrafts}
            selectedGenerationIds={selectedGenerationIds}
            selectionDisabled={generationDirty}
            onSelectionChange={onGenerationSelectionChange}
            onValidityChange={onGenerationValidityChange}
            onDirtyChange={onGenerationDirtyChange}
          />
          {analysisResult ? (
            <aside className={generationStyles.analysisPanel}>
              <div className={generationStyles.selectionHead}>
                <strong>Analyze</strong>
                <button type="button" onClick={onCloseAnalysis}>Close</button>
              </div>
              <dl className={generationStyles.analysisStats}>
                <div><dt>Generations</dt><dd>{analysisResult.summary?.generationCount ?? 0}</dd></div>
                <div><dt>Tables</dt><dd>{analysisResult.summary?.tableCount ?? 0}</dd></div>
                <div><dt>Records</dt><dd>{analysisResult.summary?.recordCount ?? 0}</dd></div>
                <div><dt>Overridden</dt><dd>{analysisResult.summary?.overriddenRecordCount ?? 0}</dd></div>
              </dl>
              <div className={generationStyles.analysisList}>
                {Object.entries(analysisResult.tables ?? {}).map(([table, summary]) => (
                  <div key={table}>
                    <strong>{table}</strong>
                    <span>{summary.recordCount} records</span>
                    <small>{summary.overriddenRecordCount} overridden</small>
                  </div>
                ))}
              </div>
              {analysisResult.diagnostics?.length ? (
                <ul className={generationStyles.diagnostics}>
                  {analysisResult.diagnostics.map((diagnostic, index) => (
                    <li key={index}>{diagnostic.generationId ? `${diagnostic.generationId}: ` : ""}{diagnostic.table ? `${diagnostic.table}: ` : ""}{diagnostic.message}</li>
                  ))}
                </ul>
              ) : (
                <p className={generationStyles.selectionHint}>No diagnostics.</p>
              )}
            </aside>
          ) : null}
        </div>
      </div>

      <footer className={styles.statusBar}>
        <span className={generationInvalid ? styles.dirty : (generationDirty ? styles.dirty : "")}>{generationInvalid ? "Generation metadata has errors" : (generationDirty ? "Unsaved generation edits" : "Generation metadata ready")}</span>
        <span>{selectedCount} selected / {generationDrafts.length} generations</span>
      </footer>
    </section>
  );
}
