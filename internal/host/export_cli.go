package host

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func RunExportCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var workspace string
	var generations string
	var format string
	var output string
	var diagnosticsFormat string
	var diagnosticsOutput string
	var timeFormat string
	var timezone string
	var mkdirs bool
	var checkOnly bool
	var jsonOut bool
	var forceOverwrite bool
	fs.StringVar(&workspace, "workspace", "", "workspace root containing masterdata")
	fs.StringVar(&generations, "generations", "", "comma-separated generation IDs to export")
	fs.StringVar(&format, "format", "", "export format: csv, excel-csv, json, yaml, ndjson, sql, xlsx, sqlite")
	fs.StringVar(&output, "output", "", "output file or directory")
	fs.BoolVar(&mkdirs, "mkdirs", false, "create missing parent directories for output")
	fs.BoolVar(&checkOnly, "check-only", false, "run validation without writing an artifact")
	fs.StringVar(&diagnosticsFormat, "diagnostics-format", "", "diagnostics format: text or json")
	fs.StringVar(&diagnosticsOutput, "diagnostics-output", "", "optional diagnostics output path")
	fs.StringVar(&timeFormat, "time-format", "", "temporal formatting: iso, epoch-sec, epoch-ms, iso-local")
	fs.StringVar(&timezone, "timezone", "", "IANA timezone for timezone-dependent temporal formats")
	fs.BoolVar(&jsonOut, "json", false, "print machine-readable command result")
	fs.BoolVar(&forceOverwrite, "force-overwrite", false, "replace an existing output after validation passes")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if format == "" {
		fmt.Fprintln(stderr, "missing required --format")
		return 2
	}
	if output == "" && !checkOnly {
		fmt.Fprintln(stderr, "missing required --output")
		return 2
	}
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
	explicit := map[string]any{}
	seen := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	if seen["time-format"] {
		explicit["time_format"] = timeFormat
	}
	if seen["timezone"] {
		explicit["timezone"] = timezone
	}
	ids, err := s.resolveCLIGenerationIDs(generations)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitCodeForError(err)
	}
	dataset, err := s.buildExportDataset(ids, format, explicit, false)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitCodeForError(err)
	}
	result := dataset.payload()
	if output != "" {
		result["output"] = output
		result["outputKind"] = outputKind(dataset.LogicalFormat)
	}
	if err := writeDiagnostics(diagnosticsOutput, diagnosticsFormat, jsonOut, dataset.Diagnostics, stderr); err != nil {
		fmt.Fprintln(stderr, err)
		return 4
	}
	if !dataset.Exportable {
		writeCLIResult(result, jsonOut, stdout, stderr)
		return 1
	}
	if !checkOnly {
		if err := writeCLIExport(dataset, output, mkdirs, forceOverwrite); err != nil {
			fmt.Fprintln(stderr, err)
			return 4
		}
	}
	writeCLIResult(result, jsonOut, stdout, stderr)
	return 0
}

func (s *server) resolveCLIGenerationIDs(generations string) ([]string, error) {
	if strings.TrimSpace(generations) != "" {
		parts := strings.Split(generations, ",")
		ids := make([]string, 0, len(parts))
		for _, part := range parts {
			id := strings.TrimSpace(part)
			if id != "" {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			return nil, appError{400, "--generations must contain at least 1 generation id(s)."}
		}
		return ids, nil
	}
	payload, err := s.loadGenerations()
	if err != nil {
		return nil, err
	}
	gens := payload["generations"].([]generation)
	var ids []string
	for _, gen := range gens {
		if gen.Output {
			ids = append(ids, gen.ID)
		}
	}
	if len(ids) == 0 {
		return nil, appError{400, "No output-enabled generations are configured."}
	}
	return ids, nil
}

func outputKind(format string) string {
	switch format {
	case "csv", "excel-csv", "json", "yaml", "ndjson":
		return "directory"
	default:
		return "file"
	}
}

func writeCLIExport(dataset exportDataset, output string, mkdirs bool, force bool) error {
	if outputKind(dataset.LogicalFormat) == "directory" {
		files, err := buildMultiFiles(dataset)
		if err != nil {
			return err
		}
		return writeDirectoryArtifact(output, files, mkdirs, force)
	}
	var data []byte
	switch dataset.LogicalFormat {
	case "sql":
		data = buildSQL(dataset)
	case "xlsx":
		xlsx, err := buildXLSX(dataset)
		if err != nil {
			return err
		}
		data = xlsx
	case "sqlite":
		sqlite, err := buildSQLite(dataset)
		if err != nil {
			return err
		}
		data = sqlite
	default:
		return appError{501, "Export format is not implemented yet: " + dataset.LogicalFormat}
	}
	return writeFileArtifact(output, data, mkdirs, force)
}

func writeDirectoryArtifact(output string, files []namedFile, mkdirs bool, force bool) error {
	parent := filepath.Dir(output)
	if mkdirs {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return err
		}
	}
	if _, err := os.Stat(parent); err != nil {
		return err
	}
	if _, err := os.Stat(output); err == nil && !force {
		return appError{409, "Output path already exists: " + output}
	}
	tmp, err := os.MkdirTemp(parent, "."+filepath.Base(output)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	for _, file := range files {
		path := filepath.Join(tmp, file.Name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, file.Data, 0o644); err != nil {
			return err
		}
	}
	if force {
		if err := os.RemoveAll(output); err != nil {
			return err
		}
	}
	return os.Rename(tmp, output)
}

func writeFileArtifact(output string, data []byte, mkdirs bool, force bool) error {
	parent := filepath.Dir(output)
	if mkdirs {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return err
		}
	}
	if _, err := os.Stat(parent); err != nil {
		return err
	}
	if _, err := os.Stat(output); err == nil && !force {
		return appError{409, "Output path already exists: " + output}
	}
	tmp, err := os.CreateTemp(parent, "."+filepath.Base(output)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if force {
		_ = os.Remove(output)
	}
	return os.Rename(tmpName, output)
}

func writeDiagnostics(path string, format string, jsonOut bool, diagnostics []map[string]any, stderr io.Writer) error {
	if format == "" {
		if jsonOut {
			format = "json"
		} else {
			format = "text"
		}
	}
	var data []byte
	switch format {
	case "json":
		encoded, err := json.MarshalIndent(diagnostics, "", "  ")
		if err != nil {
			return err
		}
		data = append(encoded, '\n')
	case "text":
		var b strings.Builder
		for _, item := range diagnostics {
			b.WriteString(fmt.Sprintf("%s", item["severity"]))
			if table := stringValue(item["table"], ""); table != "" {
				b.WriteString(" ")
				b.WriteString(table)
			}
			if field := stringValue(item["field"], ""); field != "" {
				b.WriteString(".")
				b.WriteString(field)
			}
			b.WriteString(": ")
			b.WriteString(stringValue(item["message"], ""))
			b.WriteByte('\n')
		}
		data = []byte(b.String())
	default:
		return appError{400, "--diagnostics-format must be text or json"}
	}
	if path != "" {
		return os.WriteFile(path, data, 0o644)
	}
	if len(diagnostics) > 0 && format == "text" && !jsonOut {
		_, err := stderr.Write(data)
		return err
	}
	return nil
}

func writeCLIResult(result map[string]any, jsonOut bool, stdout io.Writer, stderr io.Writer) {
	if jsonOut {
		_ = json.NewEncoder(stdout).Encode(result)
		return
	}
	if result["exportable"] == false {
		fmt.Fprintf(stderr, "Export blocked by %v diagnostic(s).\n", result["summary"].(map[string]any)["diagnosticCount"])
		return
	}
	if output, ok := result["output"].(string); ok && output != "" {
		fmt.Fprintf(stdout, "Export created: %s\n", output)
		return
	}
	fmt.Fprintln(stdout, "Export check passed.")
}

func exitCodeForError(err error) int {
	if ae, ok := err.(appError); ok {
		switch ae.status {
		case 400:
			return 2
		case 404, 422:
			return 3
		case 501:
			return 5
		}
	}
	return 3
}
