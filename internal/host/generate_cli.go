package host

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type pendingGeneratedFile struct {
	tmp    string
	target string
}

func RunGenerateCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var workspace string
	var generations string
	var definitions string
	var outputRoot string
	var diagnosticsFormat string
	var diagnosticsOutput string
	var mkdirs bool
	var checkOnly bool
	var jsonOut bool
	var forceOverwrite bool
	fs.StringVar(&workspace, "workspace", "", "workspace root containing masterdata")
	fs.StringVar(&generations, "generations", "", "comma-separated generation IDs to use")
	fs.StringVar(&definitions, "definitions", "", "comma-separated template generation definition IDs")
	fs.StringVar(&outputRoot, "output-root", "", "override generation output root")
	fs.BoolVar(&checkOnly, "check-only", false, "run validation without writing generated files")
	fs.BoolVar(&mkdirs, "mkdirs", false, "accepted for compatibility; generate creates configured output directories")
	fs.BoolVar(&forceOverwrite, "force-overwrite", false, "replace existing generated files after validation passes")
	fs.StringVar(&diagnosticsFormat, "diagnostics-format", "", "diagnostics format: text or json")
	fs.StringVar(&diagnosticsOutput, "diagnostics-output", "", "optional diagnostics output path")
	fs.BoolVar(&jsonOut, "json", false, "print machine-readable command result")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	_ = mkdirs
	if workspace == "" {
		resolved, err := ResolveWorkspace(".")
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 3
		}
		workspace = resolved
	}
	workspace, err := NewWorkspacePath(workspace)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	s, err := NewData(workspace)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	ids, err := s.resolveCLIGenerationIDs(generations)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitCodeForError(err)
	}
	result, files, outputAbs, err := s.buildGenerateResult(ids, stringSliceValue(definitions), outputRoot)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitCodeForError(err)
	}
	if err := writeDiagnostics(diagnosticsOutput, diagnosticsFormat, jsonOut, diagnosticsFromResult(result), stderr); err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}
	if result["generatable"] == false {
		writeGenerateCLIResult(result, jsonOut, stdout, stderr, checkOnly)
		return 1
	}
	if !checkOnly {
		if err := writeGeneratedFiles(outputAbs, files, forceOverwrite); err != nil {
			fmt.Fprintln(stderr, err)
			return 4
		}
	}
	writeGenerateCLIResult(result, jsonOut, stdout, stderr, checkOnly)
	return 0
}

func writeGenerateCLIResult(result map[string]any, jsonOut bool, stdout io.Writer, stderr io.Writer, checkOnly bool) {
	if jsonOut {
		writeCLIResult(result, true, stdout, stderr)
		return
	}
	summary, _ := result["summary"].(map[string]any)
	if result["generatable"] == false {
		fmt.Fprintf(stderr, "Generate blocked by %v diagnostic(s).\n", summary["diagnosticCount"])
		return
	}
	files := stringListFromAny(result["files"])
	outputRoot, _ := result["outputRoot"].(string)
	if outputRoot == "" {
		outputRoot = "."
	}
	verb := "created"
	if checkOnly {
		verb = "check passed"
	}
	fmt.Fprintf(stdout, "Generate %s: %d file(s) under %s\n", verb, len(files), outputRoot)
	for _, file := range files {
		fmt.Fprintf(stdout, "- %s\n", file)
	}
}

func (s *server) buildGenerateResult(generationIDs []string, definitionIDs []string, outputRootOverride string) (map[string]any, []namedFile, string, error) {
	defs, err := s.loadTemplateExportDefinitions()
	if err != nil {
		return nil, nil, "", err
	}
	outputAbs, outputDisplay, outputRootDiags := s.resolveGenerateOutputRoot(defs.OutputRoot, outputRootOverride)
	diags := append([]map[string]any{}, outputRootDiags...)
	dataset, err := s.buildExportDataset(generationIDs, "json", nil, false)
	if err != nil {
		return nil, nil, "", err
	}
	options := dataset.Options
	options.DefinitionIDs = definitionIDs
	dataset.Options = options
	if err := s.addTemplateOnlyTables(&dataset, defs, definitionIDs); err != nil {
		return nil, nil, "", err
	}
	files, renderDiags := s.renderTemplateExportFiles(dataset)
	diags = append(diags, dataset.Diagnostics...)
	diags = append(diags, renderDiags...)
	fileNames := make([]string, 0, len(files))
	for _, file := range files {
		fileNames = append(fileNames, file.Name)
	}
	result := map[string]any{
		"generatable":          !hasErrorDiagnostics(diags),
		"generationIds":        dataset.GenerationIDs,
		"orderedGenerationIds": dataset.OrderedGenerationIDs,
		"outputRoot":           outputDisplay,
		"summary": map[string]any{
			"definitionCount": len(selectedDefinitionIDs(defs, definitionIDs)),
			"fileCount":       len(files),
			"diagnosticCount": len(diags),
		},
		"files":       fileNames,
		"diagnostics": diags,
	}
	return result, files, outputAbs, nil
}

func (s *server) resolveGenerateOutputRoot(configured string, override string) (string, string, []map[string]any) {
	value := strings.TrimSpace(override)
	if value == "" {
		value = strings.TrimSpace(configured)
	}
	if value == "" {
		return "", "", []map[string]any{exportDiagnostic("error", "", "", "", "generate_definitions.yaml output_root is required.")}
	}
	clean := filepath.Clean(filepath.FromSlash(value))
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", value, []map[string]any{exportDiagnostic("error", "", "", "", "output_root must stay under the workspace root unless --output-root is absolute.")}
	}
	if filepath.IsAbs(clean) {
		if override == "" {
			return "", value, []map[string]any{exportDiagnostic("error", "", "", "", "Configured output_root must be relative to the workspace root.")}
		}
		return clean, value, nil
	}
	return filepath.Join(s.root, clean), filepath.ToSlash(clean), nil
}

func selectedDefinitionIDs(defs templateExportDefinitions, ids []string) []string {
	if len(ids) == 0 && len(defs.Defaults.DefinitionIDs) > 0 {
		ids = defs.Defaults.DefinitionIDs
	}
	if len(ids) > 0 {
		return ids
	}
	var selected []string
	for _, def := range defs.Definitions {
		if def.Enabled {
			selected = append(selected, def.ID)
		}
	}
	return selected
}

func writeGeneratedFiles(outputRoot string, files []namedFile, force bool) error {
	if outputRoot == "" {
		return appError{400, "output_root is required"}
	}
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return err
	}
	for _, file := range files {
		target := filepath.Join(outputRoot, filepath.FromSlash(file.Name))
		if _, err := os.Stat(target); err == nil && !force {
			return appError{409, "Generated file already exists: " + target}
		}
	}
	var pending []pendingGeneratedFile
	for _, file := range files {
		target := filepath.Join(outputRoot, filepath.FromSlash(file.Name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			cleanupPendingFiles(pending)
			return err
		}
		tmp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".*.tmp")
		if err != nil {
			cleanupPendingFiles(pending)
			return err
		}
		tmpName := tmp.Name()
		if _, err := tmp.Write(file.Data); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
			cleanupPendingFiles(pending)
			return err
		}
		if err := tmp.Close(); err != nil {
			_ = os.Remove(tmpName)
			cleanupPendingFiles(pending)
			return err
		}
		pending = append(pending, pendingGeneratedFile{tmp: tmpName, target: target})
	}
	for _, file := range pending {
		if force {
			_ = os.Remove(file.target)
		}
		if err := os.Rename(file.tmp, file.target); err != nil {
			cleanupPendingFiles(pending)
			return err
		}
	}
	return nil
}

func cleanupPendingFiles(files []pendingGeneratedFile) {
	for _, file := range files {
		_ = os.Remove(file.tmp)
	}
}

func diagnosticsFromResult(result map[string]any) []map[string]any {
	items, _ := result["diagnostics"].([]map[string]any)
	return items
}
