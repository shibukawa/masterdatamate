package host

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/flosch/pongo2/v6"
)

const generateDefinitionsVersion = 1

type templateExportDefinitions struct {
	Version     int                        `json:"version" yaml:"version"`
	OutputRoot  string                     `json:"output_root,omitempty" yaml:"output_root,omitempty"`
	Defaults    templateGenerateDefaults   `json:"defaults,omitempty" yaml:"defaults,omitempty"`
	Definitions []templateExportDefinition `json:"definitions" yaml:"definitions"`
}

type templateGenerateDefaults struct {
	DefinitionIDs []string `json:"definition_ids,omitempty" yaml:"definition_ids,omitempty"`
}

type templateExportDefinition struct {
	ID                       string              `json:"id" yaml:"id"`
	Name                     string              `json:"name" yaml:"name"`
	Enabled                  bool                `json:"enabled" yaml:"enabled"`
	Description              string              `json:"description,omitempty" yaml:"description,omitempty"`
	Scope                    string              `json:"scope" yaml:"scope"`
	Table                    string              `json:"table,omitempty" yaml:"table,omitempty"`
	GroupBy                  templateExportGroup `json:"group_by,omitempty" yaml:"group_by,omitempty"`
	Template                 string              `json:"template,omitempty" yaml:"template,omitempty"`
	TemplateFile             string              `json:"template_file,omitempty" yaml:"template_file,omitempty"`
	OutputPath               string              `json:"output_path" yaml:"output_path"`
	Overwrite                string              `json:"overwrite,omitempty" yaml:"overwrite,omitempty"`
	Formatter                string              `json:"formatter,omitempty" yaml:"formatter,omitempty"`
	LineEnding               string              `json:"line_ending,omitempty" yaml:"line_ending,omitempty"`
	IncludeNonExportedFields bool                `json:"include_non_exported_fields,omitempty" yaml:"include_non_exported_fields,omitempty"`
	Required                 *bool               `json:"required,omitempty" yaml:"required,omitempty"`
	Comment                  string              `json:"comment,omitempty" yaml:"comment,omitempty"`
}

type templateExportGroup struct {
	Table         string   `json:"table,omitempty" yaml:"table,omitempty"`
	Field         string   `json:"field,omitempty" yaml:"field,omitempty"`
	RelatedTables []string `json:"related_tables,omitempty" yaml:"related_tables,omitempty"`
}

type templateDefinitionRow struct {
	Selected     bool   `json:"selected,omitempty"`
	ID           string `json:"id"`
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	Scope        string `json:"scope"`
	Table        string `json:"table"`
	GroupField   string `json:"group_field"`
	TemplateFile string `json:"template_file"`
	OutputPath   string `json:"output_path"`
	Formatter    string `json:"formatter"`
	Comment      string `json:"comment"`
}

func init() {
	_ = pongo2.RegisterFilter("pascal", pongoStringFilter(func(s string) string { return caseWords(s, true, true) }))
	_ = pongo2.RegisterFilter("camel", pongoStringFilter(func(s string) string { return caseWords(s, false, true) }))
	_ = pongo2.RegisterFilter("snake", pongoStringFilter(func(s string) string { return joinWords(s, "_") }))
	_ = pongo2.RegisterFilter("kebab", pongoStringFilter(func(s string) string { return joinWords(s, "-") }))
	_ = pongo2.RegisterFilter("go_ident", pongoStringFilter(goIdent))
	_ = pongo2.RegisterFilter("go_string", func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		data, err := json.Marshal(fmt.Sprint(in.Interface()))
		if err != nil {
			return nil, &pongo2.Error{Sender: "filter:go_string", OrigError: err}
		}
		return pongo2.AsSafeValue(string(data)), nil
	})
	_ = pongo2.RegisterFilter("quote", func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		data, err := json.Marshal(fmt.Sprint(in.Interface()))
		if err != nil {
			return nil, &pongo2.Error{Sender: "filter:quote", OrigError: err}
		}
		return pongo2.AsSafeValue(string(data)), nil
	})
	_ = pongo2.RegisterFilter("indent", func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		width := param.Integer()
		if width < 0 {
			width = 0
		}
		prefix := strings.Repeat(" ", width)
		lines := strings.Split(fmt.Sprint(in.Interface()), "\n")
		for i, line := range lines {
			if line != "" {
				lines[i] = prefix + line
			}
		}
		return pongo2.AsValue(strings.Join(lines, "\n")), nil
	})
	_ = pongo2.RegisterFilter("comment", pongoStringFilter(func(s string) string {
		lines := strings.Split(s, "\n")
		for i, line := range lines {
			if strings.TrimSpace(line) == "" {
				lines[i] = "//"
			} else {
				lines[i] = "// " + line
			}
		}
		return strings.Join(lines, "\n")
	}))
}

func pongoStringFilter(fn func(string) string) pongo2.FilterFunction {
	return func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		return pongo2.AsValue(fn(fmt.Sprint(in.Interface()))), nil
	}
}

func (s *server) loadTemplateExportDefinitions() (templateExportDefinitions, error) {
	defs := templateExportDefinitions{Version: generateDefinitionsVersion, Definitions: []templateExportDefinition{}}
	file := s.generateDefinitionsFile()
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return defs, nil
	}
	if err := s.readYAML(file, &defs); err != nil {
		return defs, appError{422, "Generate definitions are invalid: " + err.Error()}
	}
	if defs.Version == 0 {
		defs.Version = generateDefinitionsVersion
	}
	if defs.Definitions == nil {
		defs.Definitions = []templateExportDefinition{}
	}
	return defs, nil
}

func (s *server) templateExportDefinitionsPayload() (map[string]any, error) {
	defs, err := s.loadTemplateExportDefinitions()
	if err != nil {
		return nil, err
	}
	schemas, err := s.loadSchemas()
	if err != nil {
		return nil, err
	}
	fields := map[string][]string{}
	tables := make([]string, 0, len(schemas))
	for _, item := range schemas {
		tables = append(tables, item.TableID)
		names := make([]string, 0, len(item.Fields))
		for _, f := range item.Fields {
			names = append(names, f.SystemName)
		}
		fields[item.TableID] = names
	}
	sort.Strings(tables)
	return map[string]any{"definitions": defs, "rows": templateDefinitionRows(defs.Definitions), "tables": tables, "fields": fields}, nil
}

func (s *server) saveTemplateExportDefinitions(defs templateExportDefinitions) (map[string]any, error) {
	if defs.Version == 0 {
		defs.Version = generateDefinitionsVersion
	}
	if defs.Definitions == nil {
		defs.Definitions = []templateExportDefinition{}
	}
	if diags := validateTemplateDefinitions(defs.Definitions, nil); hasErrorDiagnostics(diags) {
		return nil, appError{422, fmt.Sprint(diags[0]["message"])}
	}
	if err := s.writeYAMLAtomic(s.generateDefinitionsFile(), defs); err != nil {
		return nil, err
	}
	return s.templateExportDefinitionsPayload()
}

func templateDefinitionRows(defs []templateExportDefinition) []templateDefinitionRow {
	rows := make([]templateDefinitionRow, 0, len(defs))
	for _, def := range defs {
		rows = append(rows, templateDefinitionRow{
			ID: def.ID, Name: def.Name, Enabled: def.Enabled, Scope: def.Scope, Table: def.Table,
			GroupField: def.GroupBy.Field, TemplateFile: def.TemplateFile, OutputPath: def.OutputPath,
			Formatter: def.Formatter, Comment: firstNonEmpty(def.Comment, def.Description),
		})
	}
	return rows
}

func definitionsFromRows(rows []templateDefinitionRow) []templateExportDefinition {
	defs := make([]templateExportDefinition, 0, len(rows))
	for _, row := range rows {
		def := templateExportDefinition{
			ID: strings.TrimSpace(row.ID), Name: strings.TrimSpace(row.Name), Enabled: row.Enabled,
			Scope: strings.TrimSpace(row.Scope), Table: strings.TrimSpace(row.Table),
			TemplateFile: strings.TrimSpace(row.TemplateFile), OutputPath: strings.TrimSpace(row.OutputPath),
			Formatter: strings.TrimSpace(row.Formatter), Comment: strings.TrimSpace(row.Comment),
		}
		if row.GroupField != "" {
			def.GroupBy.Field = strings.TrimSpace(row.GroupField)
		}
		defs = append(defs, def)
	}
	return defs
}

func (s *server) renderTemplateExportFiles(dataset exportDataset) ([]namedFile, []map[string]any) {
	defs, err := s.loadTemplateExportDefinitions()
	if err != nil {
		return nil, []map[string]any{exportDiagnostic("error", "", "", "", err.Error())}
	}
	ids := dataset.Options.DefinitionIDs
	if len(ids) == 0 && len(defs.Defaults.DefinitionIDs) > 0 {
		ids = defs.Defaults.DefinitionIDs
	}
	selected, diags := selectTemplateDefinitions(defs.Definitions, ids)
	if hasErrorDiagnostics(diags) {
		return nil, diags
	}
	schemas, err := s.loadSchemas()
	if err != nil {
		return nil, []map[string]any{exportDiagnostic("error", "", "", "", err.Error())}
	}
	schemaByTable := map[string]schema{}
	for _, item := range schemas {
		schemaByTable[item.TableID] = item
	}
	diags = append(diags, validateTemplateDefinitions(selected, schemaByTable)...)

	base := templateBaseContext(dataset)
	files := []namedFile{}
	outputs := map[string]int{}
	for _, def := range selected {
		source, err := s.templateSource(def)
		if err != nil {
			diags = append(diags, exportDiagnostic("error", def.ID, def.Table, "", err.Error()))
			continue
		}
		tpl, err := pongo2.FromString(source)
		if err != nil {
			diags = append(diags, exportDiagnostic("error", def.ID, def.Table, "", "Template parse error: "+err.Error()))
			continue
		}
		pathTpl, err := pongo2.FromString(def.OutputPath)
		if err != nil {
			diags = append(diags, exportDiagnostic("error", def.ID, def.Table, "", "Output path template parse error: "+err.Error()))
			continue
		}
		jobs := renderJobs(def, dataset, base)
		for _, ctx := range jobs {
			outputPath, err := pathTpl.Execute(ctx)
			if err != nil {
				diags = append(diags, exportDiagnostic("error", def.ID, def.Table, recordKeyForContext(ctx), "Output path render error: "+err.Error()))
				continue
			}
			safePath, err := cleanExportOutputPath(outputPath)
			if err != nil {
				diags = append(diags, exportDiagnostic("error", def.ID, def.Table, recordKeyForContext(ctx), err.Error()))
				continue
			}
			if previous, exists := outputs[safePath]; exists {
				switch def.Overwrite {
				case "replace":
					files = append(files[:previous], files[previous+1:]...)
					outputs[safePath] = len(files)
				case "skip":
					diags = append(diags, exportDiagnostic("warning", def.ID, def.Table, recordKeyForContext(ctx), "Skipped duplicate output path: "+safePath))
					continue
				default:
					diags = append(diags, exportDiagnostic("error", def.ID, def.Table, recordKeyForContext(ctx), "Duplicate output path: "+safePath))
					continue
				}
			} else {
				outputs[safePath] = len(files)
			}
			rendered, err := tpl.Execute(ctx)
			if err != nil {
				diags = append(diags, exportDiagnostic("error", def.ID, def.Table, recordKeyForContext(ctx), "Template render error: "+err.Error()))
				continue
			}
			data := []byte(normalizeTemplateLineEndings(rendered, def.LineEnding))
			if def.Formatter == "gofmt" {
				formatted, err := gofmtBytes(data)
				if err != nil {
					diags = append(diags, exportDiagnostic("error", def.ID, def.Table, recordKeyForContext(ctx), "gofmt failed: "+err.Error()))
					continue
				}
				data = formatted
			}
			files = append(files, namedFile{Name: safePath, Data: data})
		}
	}
	return files, diags
}

func (s *server) addTemplateOnlyTables(dataset *exportDataset, defs templateExportDefinitions, definitionIDs []string) error {
	selectedIDs := selectedDefinitionIDs(defs, definitionIDs)
	selected, diags := selectTemplateDefinitions(defs.Definitions, selectedIDs)
	if hasErrorDiagnostics(diags) {
		return nil
	}
	needs := map[string]bool{}
	for _, def := range selected {
		if def.Table == "" {
			continue
		}
		if _, ok := dataset.Tables[def.Table]; !ok {
			needs[def.Table] = true
		}
	}
	if len(needs) == 0 {
		return nil
	}
	schemas, err := s.loadSchemas()
	if err != nil {
		return err
	}
	schemaByTable := map[string]schema{}
	for _, item := range schemas {
		schemaByTable[item.TableID] = item
	}
	ordered, err := s.generationsByIDs(dataset.OrderedGenerationIDs)
	if err != nil {
		return err
	}
	for tableID := range needs {
		item, ok := schemaByTable[tableID]
		if !ok {
			continue
		}
		rows, err := s.mergedTemplateRows(item, ordered)
		if err != nil {
			return err
		}
		dataset.Tables[tableID] = exportTable{Schema: item, Rows: rows}
		dataset.Summary["tableCount"] = len(dataset.Tables)
		dataset.Summary["recordCount"] = intValue(dataset.Summary["recordCount"]) + len(rows)
	}
	return nil
}

func (s *server) generationsByIDs(ids []string) ([]generation, error) {
	payload, err := s.loadGenerations()
	if err != nil {
		return nil, err
	}
	all := payload["generations"].([]generation)
	byID := map[string]generation{}
	for _, gen := range all {
		byID[gen.ID] = gen
	}
	ordered := make([]generation, 0, len(ids))
	for _, id := range ids {
		gen, ok := byID[id]
		if !ok {
			return nil, appError{404, "Generation not found: " + id}
		}
		ordered = append(ordered, gen)
	}
	sortGenerations(ordered)
	return ordered, nil
}

func (s *server) mergedTemplateRows(item schema, ordered []generation) ([]map[string]any, error) {
	byKey := map[string]map[string]any{}
	order := []string{}
	for _, gen := range ordered {
		records, err := s.loadRecords(item.TableID, gen.ID)
		if err != nil {
			return nil, err
		}
		for _, rec := range records {
			row := recordToRow(rec, item)
			key := normalizeComparable(keyFromRow(row, item))
			if _, ok := byKey[key]; !ok {
				order = append(order, key)
			}
			byKey[key] = row
		}
	}
	rows := make([]map[string]any, 0, len(order))
	for _, key := range order {
		rows = append(rows, byKey[key])
	}
	return rows, nil
}

func selectTemplateDefinitions(defs []templateExportDefinition, ids []string) ([]templateExportDefinition, []map[string]any) {
	byID := map[string]templateExportDefinition{}
	for _, def := range defs {
		byID[def.ID] = def
	}
	if len(ids) > 0 {
		selected := make([]templateExportDefinition, 0, len(ids))
		diags := []map[string]any{}
		seen := map[string]bool{}
		for _, id := range ids {
			if seen[id] {
				diags = append(diags, exportDiagnostic("error", id, "", "", "Template definition ID is duplicated in selection: "+id))
				continue
			}
			seen[id] = true
			def, ok := byID[id]
			if !ok {
				diags = append(diags, exportDiagnostic("error", id, "", "", "Template definition not found: "+id))
				continue
			}
			selected = append(selected, def)
		}
		return selected, diags
	}
	selected := []templateExportDefinition{}
	for _, def := range defs {
		if def.Enabled {
			selected = append(selected, def)
		}
	}
	return selected, nil
}

func validateTemplateDefinitions(defs []templateExportDefinition, schemas map[string]schema) []map[string]any {
	diags := []map[string]any{}
	seen := map[string]bool{}
	for _, def := range defs {
		if def.ID == "" {
			diags = append(diags, exportDiagnostic("error", "", def.Table, "", "Template definition id is required."))
		}
		if seen[def.ID] {
			diags = append(diags, exportDiagnostic("error", def.ID, def.Table, "", "Template definition id must be unique: "+def.ID))
		}
		seen[def.ID] = true
		if def.Scope == "" {
			diags = append(diags, exportDiagnostic("error", def.ID, def.Table, "", "Template definition scope is required."))
		}
		if !contains([]string{"project", "table", "record", "group"}, def.Scope) {
			diags = append(diags, exportDiagnostic("error", def.ID, def.Table, "", "Template definition scope must be project, table, record, or group."))
		}
		if def.Scope != "project" && def.Table == "" {
			diags = append(diags, exportDiagnostic("error", def.ID, def.Table, "", "Template definition table is required for this scope."))
		}
		if schemas != nil && def.Table != "" {
			item, ok := schemas[def.Table]
			if !ok {
				diags = append(diags, exportDiagnostic("error", def.ID, def.Table, "", "Template definition target table does not exist: "+def.Table))
			} else if def.Scope == "group" {
				groupField := def.GroupBy.Field
				if groupField == "" {
					diags = append(diags, exportDiagnostic("error", def.ID, def.Table, "", "Template definition group field is required."))
				} else if !schemaHasField(item, groupField) {
					diags = append(diags, exportDiagnostic("error", def.ID, def.Table, "", "Template definition group field does not exist: "+groupField))
				}
			}
		}
		if (def.Template == "") == (def.TemplateFile == "") {
			diags = append(diags, exportDiagnostic("error", def.ID, def.Table, "", "Set exactly one of template or template_file."))
		}
		if def.OutputPath == "" {
			diags = append(diags, exportDiagnostic("error", def.ID, def.Table, "", "Template definition output_path is required."))
		}
		if def.Formatter != "" && def.Formatter != "gofmt" {
			diags = append(diags, exportDiagnostic("error", def.ID, def.Table, "", "Template definition formatter must be gofmt or empty."))
		}
	}
	return diags
}

func (s *server) templateSource(def templateExportDefinition) (string, error) {
	if def.Template != "" {
		return def.Template, nil
	}
	rel := filepath.Clean(filepath.FromSlash(def.TemplateFile))
	if rel == "." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", fmt.Errorf("Invalid template file path: %s", def.TemplateFile)
	}
	path := filepath.Join(s.generateTemplatesRoot(), rel)
	root, err := filepath.Abs(s.generateTemplatesRoot())
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if abs != root && !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		return "", fmt.Errorf("Template file path escapes generate_templates: %s", def.TemplateFile)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func templateBaseContext(dataset exportDataset) pongo2.Context {
	tables := pongo2.Context{}
	schemas := pongo2.Context{}
	for _, table := range sortedExportTables(dataset.Tables) {
		rows := make([]map[string]any, 0, len(table.Rows))
		for _, row := range table.Rows {
			rows = append(rows, templateRecord(row, table.Schema))
		}
		tableCtx := pongo2.Context{"schema": table.Schema, "system_name": table.Schema.TableID, "fields": exportFields(table.Schema), "records": rows}
		tables[table.Schema.TableID] = tableCtx
		schemas[table.Schema.TableID] = table.Schema
	}
	return pongo2.Context{
		"project":        pongo2.Context{"name": filepath.Base(filepath.Clean("."))},
		"generation_ids": dataset.GenerationIDs, "ordered_generation_ids": dataset.OrderedGenerationIDs,
		"tables": tables, "schemas": schemas,
	}
}

func renderJobs(def templateExportDefinition, dataset exportDataset, base pongo2.Context) []pongo2.Context {
	ctx := cloneContext(base)
	ctx["definition"] = def
	if def.Scope == "project" {
		return []pongo2.Context{ctx}
	}
	table, ok := dataset.Tables[def.Table]
	if !ok {
		return []pongo2.Context{ctx}
	}
	tableCtx := base["tables"].(pongo2.Context)[def.Table].(pongo2.Context)
	ctx["table"] = tableCtx
	switch def.Scope {
	case "table":
		ctx["records"] = tableCtx["records"]
		return []pongo2.Context{ctx}
	case "record":
		jobs := make([]pongo2.Context, 0, len(table.Rows))
		for _, row := range table.Rows {
			job := cloneContext(ctx)
			job["record"] = templateRecord(row, table.Schema)
			jobs = append(jobs, job)
		}
		return jobs
	case "group":
		groups := map[string][]map[string]any{}
		keys := []string{}
		for _, row := range table.Rows {
			key := normalizeComparable(row[def.GroupBy.Field])
			if _, ok := groups[key]; !ok {
				keys = append(keys, key)
			}
			groups[key] = append(groups[key], templateRecord(row, table.Schema))
		}
		sort.Strings(keys)
		jobs := make([]pongo2.Context, 0, len(keys))
		for _, key := range keys {
			rows := groups[key]
			displayKey := rows[0][def.GroupBy.Field]
			job := cloneContext(ctx)
			job["records"] = rows
			job["group"] = pongo2.Context{"key": displayKey, "label": fmt.Sprint(displayKey), "field": def.GroupBy.Field, "table": def.Table, "records": rows}
			jobs = append(jobs, job)
		}
		return jobs
	default:
		return []pongo2.Context{ctx}
	}
}

func templateRecord(row map[string]any, schema schema) map[string]any {
	out := map[string]any{}
	for key, value := range row {
		out[key] = value
	}
	out["_key"] = keyFromRow(row, schema)
	out["_name"] = stringValue(row["name"], "")
	out["_table"] = schema.TableID
	return out
}

func cleanExportOutputPath(value string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(value)))
	if clean == "." || clean == "" {
		return "", fmt.Errorf("Rendered output path is empty.")
	}
	if strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("Rendered output path is unsafe: %s", value)
	}
	return clean, nil
}

func normalizeTemplateLineEndings(text string, mode string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text
}

func gofmtBytes(data []byte) ([]byte, error) {
	cmd := exec.Command("gofmt")
	cmd.Stdin = bytes.NewReader(data)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return nil, errors.New(strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}
	return out, nil
}

func exportDiagnostic(severity, definitionID, table, recordKey, message string) map[string]any {
	item := map[string]any{"severity": severity, "message": message}
	if definitionID != "" {
		item["definitionId"] = definitionID
	}
	if table != "" {
		item["table"] = table
	}
	if recordKey != "" {
		item["recordKey"] = recordKey
	}
	return item
}

func recordKeyForContext(ctx pongo2.Context) string {
	if record, ok := ctx["record"].(map[string]any); ok {
		return normalizeComparable(record["_key"])
	}
	return ""
}

func cloneContext(ctx pongo2.Context) pongo2.Context {
	next := pongo2.Context{}
	for key, value := range ctx {
		next[key] = value
	}
	return next
}

func schemaHasField(item schema, name string) bool {
	for _, f := range item.Fields {
		if f.SystemName == name {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func stringSliceValue(value any) []string {
	switch items := value.(type) {
	case []string:
		return append([]string{}, items...)
	case []any:
		out := make([]string, 0, len(items))
		for _, item := range items {
			if text := strings.TrimSpace(stringValue(item, "")); text != "" {
				out = append(out, text)
			}
		}
		return out
	case string:
		parts := strings.Split(items, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			if text := strings.TrimSpace(part); text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func stringListFromAny(value any) []string {
	switch items := value.(type) {
	case []string:
		return append([]string{}, items...)
	case []any:
		out := make([]string, 0, len(items))
		for _, item := range items {
			out = append(out, fmt.Sprint(item))
		}
		return out
	default:
		return nil
	}
}

func caseWords(value string, upperFirst bool, upperRest bool) string {
	words := splitIdentifierWords(value)
	var b strings.Builder
	for i, word := range words {
		if word == "" {
			continue
		}
		lower := strings.ToLower(word)
		if i == 0 && !upperFirst {
			b.WriteString(lower)
			continue
		}
		runes := []rune(lower)
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}
	return b.String()
}

func joinWords(value string, sep string) string {
	return strings.Join(splitIdentifierWords(value), sep)
}

func splitIdentifierWords(value string) []string {
	var words []string
	var current []rune
	var previousLower bool
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if previousLower && unicode.IsUpper(r) && len(current) > 0 {
				words = append(words, strings.ToLower(string(current)))
				current = []rune{r}
			} else {
				current = append(current, r)
			}
			previousLower = unicode.IsLower(r) || unicode.IsDigit(r)
			continue
		}
		if len(current) > 0 {
			words = append(words, strings.ToLower(string(current)))
			current = nil
		}
		previousLower = false
	}
	if len(current) > 0 {
		words = append(words, strings.ToLower(string(current)))
	}
	return words
}

func goIdent(value string) string {
	name := caseWords(value, true, true)
	if name == "" {
		return "Value"
	}
	var b strings.Builder
	for i, r := range name {
		if i == 0 {
			if unicode.IsLetter(r) || r == '_' {
				b.WriteRune(r)
			} else {
				b.WriteRune('_')
				if unicode.IsDigit(r) {
					b.WriteRune(r)
				}
			}
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "Value"
	}
	return out
}
