package host

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

const (
	exportSettingsVersion = 1
	defaultTimeFormat     = "iso"
)

var httpExportFormats = map[string]string{
	"csv_zip":       "csv",
	"excel_csv_zip": "excel-csv",
	"json_zip":      "json",
	"yaml_zip":      "yaml",
	"ndjson_zip":    "ndjson",
	"sql":           "sql",
	"xlsx":          "xlsx",
	"sqlite":        "sqlite",
}

var logicalExportFormats = map[string]bool{
	"csv": true, "excel-csv": true, "json": true, "yaml": true, "ndjson": true,
	"sql": true, "xlsx": true, "sqlite": true,
}

type exportSettings struct {
	Version int                            `json:"version" yaml:"version"`
	Formats map[string]exportFormatOptions `json:"formats" yaml:"formats"`
}

type exportFormatOptions struct {
	TimeFormat              string   `json:"time_format,omitempty" yaml:"time_format,omitempty"`
	Timezone                string   `json:"timezone,omitempty" yaml:"timezone,omitempty"`
	IncludeSchema           *bool    `json:"include_schema,omitempty" yaml:"include_schema,omitempty"`
	IncludeDiagnosticsSheet *bool    `json:"include_diagnostics_sheet,omitempty" yaml:"include_diagnostics_sheet,omitempty"`
	SQLDialect              string   `json:"sql_dialect,omitempty" yaml:"sql_dialect,omitempty"`
	DefinitionIDs           []string `json:"definition_ids,omitempty" yaml:"definition_ids,omitempty"`
	UpdatedAt               string   `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
}

type exportOptions struct {
	TimeFormat              string
	Timezone                string
	IncludeSchema           bool
	IncludeDiagnosticsSheet bool
	SQLDialect              string
	DefinitionIDs           []string
}

type exportDataset struct {
	Exportable           bool
	GenerationIDs        []string
	OrderedGenerationIDs []string
	Format               string
	LogicalFormat        string
	Options              exportOptions
	Tables               map[string]exportTable
	TemplateFiles        []namedFile
	Summary              map[string]any
	Diagnostics          []map[string]any
}

type exportTable struct {
	Schema schema
	Rows   []map[string]any
}

func (s *server) exportSettingsFile() string {
	return filepath.Join(s.root, "masterdata", "export_settings.yaml")
}

func (s *server) generateDefinitionsFile() string {
	return filepath.Join(s.root, "masterdata", "generate_definitions.yaml")
}

func (s *server) generateTemplatesRoot() string {
	return filepath.Join(s.root, "masterdata", "generate_templates")
}

func (s *server) loadExportSettings() (exportSettings, error) {
	settings := exportSettings{Version: exportSettingsVersion, Formats: map[string]exportFormatOptions{}}
	file := s.exportSettingsFile()
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return settings, nil
	}
	if err := s.readYAML(file, &settings); err != nil {
		return settings, appError{422, "Export settings are invalid: " + err.Error()}
	}
	if settings.Version == 0 {
		settings.Version = exportSettingsVersion
	}
	if settings.Formats == nil {
		settings.Formats = map[string]exportFormatOptions{}
	}
	for format, options := range settings.Formats {
		logical, ok := normalizeLogicalExportFormat(format)
		if !ok {
			continue
		}
		if err := validateExportFormatOptions(options); err != nil {
			return settings, appError{422, fmt.Sprintf("Export settings for %s are invalid: %s", format, err.Error())}
		}
		if logical != format {
			delete(settings.Formats, format)
			settings.Formats[logical] = options
		}
	}
	return settings, nil
}

func (s *server) exportSettingsPayload() (map[string]any, error) {
	settings, err := s.loadExportSettings()
	if err != nil {
		return nil, err
	}
	effective := map[string]exportOptions{}
	for format := range logicalExportFormats {
		options, err := s.resolveExportOptions(format, nil)
		if err != nil {
			return nil, err
		}
		effective[format] = options
	}
	return map[string]any{"settings": settings, "effective": effective}, nil
}

func (s *server) saveExportSettings(settings exportSettings) (map[string]any, error) {
	if settings.Version == 0 {
		settings.Version = exportSettingsVersion
	}
	if settings.Formats == nil {
		settings.Formats = map[string]exportFormatOptions{}
	}
	normalized := exportSettings{Version: settings.Version, Formats: map[string]exportFormatOptions{}}
	for format, options := range settings.Formats {
		logical, ok := normalizeLogicalExportFormat(format)
		if !ok {
			return nil, appError{400, "Unknown export format: " + format}
		}
		if err := validateExportFormatOptions(options); err != nil {
			return nil, appError{422, fmt.Sprintf("Export settings for %s are invalid: %s", logical, err.Error())}
		}
		options.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		normalized.Formats[logical] = options
	}
	if err := s.writeYAMLAtomic(s.exportSettingsFile(), normalized); err != nil {
		return nil, err
	}
	return s.exportSettingsPayload()
}

func (s *server) writeYAMLAtomic(file string, value any) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(file), "."+filepath.Base(file)+".*.tmp")
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
	return os.Rename(tmpName, file)
}

func validateExportFormatOptions(options exportFormatOptions) error {
	if options.TimeFormat != "" && !contains([]string{"iso", "epoch-sec", "epoch-ms", "iso-local"}, options.TimeFormat) {
		return fmt.Errorf("time_format must be iso, epoch-sec, epoch-ms, or iso-local")
	}
	if options.Timezone != "" {
		if _, err := time.LoadLocation(options.Timezone); err != nil {
			return fmt.Errorf("timezone must be an IANA timezone name")
		}
	}
	return nil
}

func normalizeHTTPExportFormat(format string) (string, string, error) {
	value := strings.TrimSpace(format)
	if value == "" {
		return "", "", appError{400, "format is required."}
	}
	logical, ok := httpExportFormats[value]
	if !ok {
		return "", "", appError{400, "Unknown export format: " + value}
	}
	return value, logical, nil
}

func normalizeCLIExportFormat(format string) (string, error) {
	value := strings.TrimSpace(format)
	if value == "" {
		return "", appError{400, "format is required."}
	}
	logical, ok := normalizeLogicalExportFormat(value)
	if !ok {
		return "", appError{400, "Unknown export format: " + value}
	}
	return logical, nil
}

func normalizeLogicalExportFormat(format string) (string, bool) {
	if logical, ok := httpExportFormats[format]; ok {
		return logical, true
	}
	if logicalExportFormats[format] {
		return format, true
	}
	return "", false
}

func (s *server) resolveExportOptions(logicalFormat string, explicit map[string]any) (exportOptions, error) {
	settings, err := s.loadExportSettings()
	if err != nil {
		return exportOptions{}, err
	}
	options := exportOptions{TimeFormat: defaultTimeFormat, IncludeSchema: true, SQLDialect: "generic"}
	if saved, ok := settings.Formats[logicalFormat]; ok {
		applySavedExportOptions(&options, saved)
	}
	applyExplicitExportOptions(&options, explicit)
	if !contains([]string{"iso", "epoch-sec", "epoch-ms", "iso-local"}, options.TimeFormat) {
		return options, appError{422, "time_format must be iso, epoch-sec, epoch-ms, or iso-local."}
	}
	if options.Timezone != "" {
		if _, err := time.LoadLocation(options.Timezone); err != nil {
			return options, appError{422, "timezone must be an IANA timezone name."}
		}
	}
	return options, nil
}

func applySavedExportOptions(options *exportOptions, saved exportFormatOptions) {
	if saved.TimeFormat != "" {
		options.TimeFormat = saved.TimeFormat
	}
	if saved.Timezone != "" {
		options.Timezone = saved.Timezone
	}
	if saved.IncludeSchema != nil {
		options.IncludeSchema = *saved.IncludeSchema
	}
	if saved.IncludeDiagnosticsSheet != nil {
		options.IncludeDiagnosticsSheet = *saved.IncludeDiagnosticsSheet
	}
	if saved.SQLDialect != "" {
		options.SQLDialect = saved.SQLDialect
	}
	if len(saved.DefinitionIDs) > 0 {
		options.DefinitionIDs = append([]string{}, saved.DefinitionIDs...)
	}
}

func applyExplicitExportOptions(options *exportOptions, explicit map[string]any) {
	if explicit == nil {
		return
	}
	if v := stringValue(explicit["time_format"], ""); v == "" {
		if camel := stringValue(explicit["timeFormat"], ""); camel != "" {
			options.TimeFormat = camel
		}
	} else {
		options.TimeFormat = v
	}
	if v := stringValue(explicit["timezone"], ""); v != "" {
		options.Timezone = v
	}
	if v, ok := explicit["include_schema"].(bool); ok {
		options.IncludeSchema = v
	}
	if v, ok := explicit["includeSchema"].(bool); ok {
		options.IncludeSchema = v
	}
	if v, ok := explicit["include_diagnostics_sheet"].(bool); ok {
		options.IncludeDiagnosticsSheet = v
	}
	if v, ok := explicit["includeDiagnosticsSheet"].(bool); ok {
		options.IncludeDiagnosticsSheet = v
	}
	if v := stringValue(explicit["sql_dialect"], ""); v != "" {
		options.SQLDialect = v
	}
	if v := stringValue(explicit["sqlDialect"], ""); v != "" {
		options.SQLDialect = v
	}
	if values := stringSliceValue(explicit["definition_ids"]); len(values) > 0 {
		options.DefinitionIDs = values
	}
	if values := stringSliceValue(explicit["definitionIds"]); len(values) > 0 {
		options.DefinitionIDs = values
	}
}

func (s *server) buildExportDataset(ids []string, format string, explicitOptions map[string]any, httpFormat bool) (exportDataset, error) {
	var outputFormat, logical string
	var err error
	if httpFormat {
		outputFormat, logical, err = normalizeHTTPExportFormat(format)
	} else {
		logical, err = normalizeCLIExportFormat(format)
		outputFormat = logical
	}
	if err != nil {
		return exportDataset{}, err
	}
	if len(ids) == 0 {
		return exportDataset{}, appError{400, "generationIds must contain at least 1 generation id(s)."}
	}
	unique := map[string]bool{}
	for _, id := range ids {
		if unique[id] {
			return exportDataset{}, appError{400, "generationIds must be unique."}
		}
		unique[id] = true
	}
	options, err := s.resolveExportOptions(logical, explicitOptions)
	if err != nil {
		return exportDataset{}, err
	}
	generationPayload, err := s.loadGenerations()
	if err != nil {
		return exportDataset{}, err
	}
	allGenerations := generationPayload["generations"].([]generation)
	byID := map[string]generation{}
	for _, gen := range allGenerations {
		byID[gen.ID] = gen
	}
	var ordered []generation
	for _, id := range ids {
		gen, ok := byID[id]
		if !ok {
			return exportDataset{}, appError{404, "Generation not found: " + id}
		}
		ordered = append(ordered, gen)
	}
	sortGenerations(ordered)

	schemas, err := s.loadSchemas()
	if err != nil {
		return exportDataset{}, err
	}
	exportSchemas := make([]schema, 0, len(schemas))
	schemaByTable := map[string]schema{}
	for _, item := range schemas {
		schemaByTable[item.TableID] = item
		if item.Export {
			exportSchemas = append(exportSchemas, item)
		}
	}

	tables := map[string]exportTable{}
	exportableKeys := map[string]map[string]bool{}
	diagnostics := []map[string]any{}
	recordCount := 0
	for _, item := range exportSchemas {
		rows, err := s.mergedExportRows(item, ordered)
		if err != nil {
			return exportDataset{}, err
		}
		keys := map[string]bool{}
		for rowIndex, row := range rows {
			key := keyFromRow(row, item)
			keyLabel := normalizeComparable(key)
			if keys[keyLabel] {
				diagnostics = append(diagnostics, diagnostic("error", item.TableID, "", rowIndex, first(item.PrimaryKey), "Primary key is duplicated in the effective export dataset."))
			}
			keys[keyLabel] = true
			for _, f := range exportFields(item) {
				value := normalizeReferenceValue(row[f.SystemName])
				if msg := validateExportValue(value, f); msg != "" {
					diagnostics = append(diagnostics, diagnostic("error", item.TableID, "", rowIndex, f.SystemName, msg))
				}
			}
		}
		exportableKeys[item.TableID] = keys
		tables[item.TableID] = exportTable{Schema: item, Rows: rows}
		recordCount += len(rows)
	}

	for _, item := range exportSchemas {
		table := tables[item.TableID]
		for rowIndex, row := range table.Rows {
			recordKey := normalizeComparable(keyFromRow(row, item))
			for _, f := range item.Fields {
				if f.Type != "external_reference" {
					continue
				}
				target := stringMapValue(f.Reference, "table")
				if target == "" {
					continue
				}
				value := normalizeReferenceValue(row[f.SystemName])
				if isBlank(value) {
					continue
				}
				targetSchema, ok := schemaByTable[target]
				if !ok || !targetSchema.Export || !exportableKeys[target][normalizeComparable(value)] {
					diagnostics = append(diagnostics, map[string]any{
						"severity": "error", "table": item.TableID, "rowIndex": rowIndex, "recordKey": recordKey, "field": f.SystemName,
						"message": fmt.Sprintf("Referenced %s record %v is not present in the selected export generation set.", target, value),
					})
				}
			}
		}
	}

	orderedIDs := make([]string, len(ordered))
	for i, gen := range ordered {
		orderedIDs[i] = gen.ID
	}
	return exportDataset{
		Exportable:           !hasErrorDiagnostics(diagnostics),
		GenerationIDs:        ids,
		OrderedGenerationIDs: orderedIDs,
		Format:               outputFormat,
		LogicalFormat:        logical,
		Options:              options,
		Tables:               tables,
		Summary:              map[string]any{"tableCount": len(tables), "recordCount": recordCount, "diagnosticCount": len(diagnostics)},
		Diagnostics:          diagnostics,
	}, nil
}

func (s *server) mergedExportRows(item schema, ordered []generation) ([]map[string]any, error) {
	byKey := map[string]map[string]any{}
	order := []string{}
	for _, gen := range ordered {
		records, err := s.loadRecords(item.TableID, gen.ID)
		if err != nil {
			return nil, err
		}
		for _, rec := range records {
			full := recordToRow(rec, item)
			row := exportedRow(full, item)
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

func exportFields(item schema) []field {
	primary := map[string]bool{}
	fields := []field{}
	for _, key := range item.PrimaryKey {
		primary[key] = true
		found := false
		for _, f := range item.Fields {
			if f.SystemName == key {
				fields = append(fields, f)
				found = true
				break
			}
		}
		if !found {
			fields = append(fields, field{SystemName: key, BusinessName: key, Type: "string"})
		}
	}
	for _, f := range item.Fields {
		if primary[f.SystemName] {
			continue
		}
		if f.Export != nil && !*f.Export {
			continue
		}
		fields = append(fields, f)
	}
	return fields
}

func exportedRow(row map[string]any, item schema) map[string]any {
	out := map[string]any{}
	for _, f := range exportFields(item) {
		out[f.SystemName] = normalizeReferenceValue(row[f.SystemName])
	}
	return out
}

func validateExportValue(value any, f field) string {
	if f.Required && isBlank(value) {
		return fmt.Sprintf("%s is required.", f.BusinessName)
	}
	if isBlank(value) {
		return ""
	}
	switch f.Type {
	case "integer":
		if _, ok := toInt(value); !ok {
			return fmt.Sprintf("%s must be an integer.", f.BusinessName)
		}
	case "decimal":
		if _, ok := toFloat(value); !ok {
			return fmt.Sprintf("%s must be a number.", f.BusinessName)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Sprintf("%s must be true or false.", f.BusinessName)
		}
	case "constant":
		if len(f.Constants) > 0 && !contains(f.Constants, fmt.Sprint(value)) {
			return fmt.Sprintf("%s must be one of: %s.", f.BusinessName, strings.Join(f.Constants, ", "))
		}
	}
	return ""
}

func (d exportDataset) payload() map[string]any {
	return map[string]any{
		"exportable": d.Exportable, "generationIds": d.GenerationIDs, "orderedGenerationIds": d.OrderedGenerationIDs,
		"format": d.Format, "logicalFormat": d.LogicalFormat, "options": d.Options, "summary": d.Summary, "diagnostics": d.Diagnostics,
	}
}

func (s *server) checkExport(ids []string, format string, options map[string]any) (map[string]any, error) {
	dataset, err := s.buildExportDataset(ids, format, options, true)
	if err != nil {
		return nil, err
	}
	return dataset.payload(), nil
}

func (s *server) createExport(ids []string, format string, options map[string]any) (map[string]any, int, error) {
	dataset, err := s.buildExportDataset(ids, format, options, true)
	if err != nil {
		return nil, 0, err
	}
	if !dataset.Exportable {
		return dataset.payload(), 422, nil
	}
	data, filename, contentType, err := buildHTTPExportArtifact(dataset)
	if err != nil {
		return nil, 0, err
	}
	exportID := strings.ReplaceAll(time.Now().UTC().Format(time.RFC3339Nano), ":", "-") + "_" + dataset.Format
	s.exports[exportID] = exportArtifact{data: data, filename: filename, contentType: contentType, createdAt: time.Now()}
	payload := dataset.payload()
	payload["exportId"] = exportID
	payload["filename"] = filename
	payload["contentType"] = contentType
	payload["downloadUrl"] = "/api/exports/" + exportID + "/download"
	return payload, 201, nil
}

func buildHTTPExportArtifact(dataset exportDataset) ([]byte, string, string, error) {
	switch dataset.Format {
	case "csv_zip", "excel_csv_zip", "json_zip", "yaml_zip", "ndjson_zip":
		files, err := buildMultiFiles(dataset)
		if err != nil {
			return nil, "", "", err
		}
		data, err := createZip(files)
		return data, exportFilename(dataset.Format), "application/zip", err
	case "sql":
		return buildSQL(dataset), "masterdata-export.sql", "application/sql; charset=utf-8", nil
	case "xlsx":
		data, err := buildXLSX(dataset)
		return data, "masterdata-export.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", err
	case "sqlite":
		data, err := buildSQLite(dataset)
		return data, "masterdata-export.sqlite", "application/vnd.sqlite3", err
	}
	return nil, "", "", appError{501, "Export format is not implemented yet: " + dataset.Format}
}

func exportFilename(format string) string {
	switch format {
	case "csv_zip":
		return "masterdata-export.csv.zip"
	case "excel_csv_zip":
		return "masterdata-export.excel-csv.zip"
	case "json_zip":
		return "masterdata-export.json.zip"
	case "yaml_zip":
		return "masterdata-export.yaml.zip"
	case "ndjson_zip":
		return "masterdata-export.ndjson.zip"
	case "sqlite":
		return "masterdata-export.sqlite"
	default:
		return "masterdata-export.dat"
	}
}

type namedFile struct {
	Name string
	Data []byte
}

func buildMultiFiles(dataset exportDataset) ([]namedFile, error) {
	var files []namedFile
	tables := sortedExportTables(dataset.Tables)
	for _, table := range tables {
		fields := exportFields(table.Schema)
		switch dataset.LogicalFormat {
		case "csv", "excel-csv":
			files = append(files, namedFile{Name: table.Schema.TableID + ".csv", Data: buildCSV(table.Rows, fields, dataset.Options, dataset.LogicalFormat == "excel-csv")})
		case "json":
			data, err := json.MarshalIndent(table.Rows, "", "  ")
			if err != nil {
				return nil, err
			}
			files = append(files, namedFile{Name: table.Schema.TableID + ".json", Data: data})
		case "yaml":
			data, err := yaml.Marshal(map[string]any{table.Schema.TableID: table.Rows})
			if err != nil {
				return nil, err
			}
			files = append(files, namedFile{Name: table.Schema.TableID + ".yaml", Data: data})
		case "ndjson":
			var b bytes.Buffer
			for _, row := range table.Rows {
				data, err := json.Marshal(row)
				if err != nil {
					return nil, err
				}
				b.Write(data)
				b.WriteByte('\n')
			}
			files = append(files, namedFile{Name: table.Schema.TableID + ".ndjson", Data: b.Bytes()})
		}
	}
	if dataset.Options.IncludeSchema {
		manifest, err := json.MarshalIndent(exportManifest(dataset), "", "  ")
		if err != nil {
			return nil, err
		}
		files = append(files, namedFile{Name: "manifest.json", Data: manifest})
	}
	return files, nil
}

func sortedExportTables(tables map[string]exportTable) []exportTable {
	keys := make([]string, 0, len(tables))
	for key := range tables {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]exportTable, 0, len(keys))
	for _, key := range keys {
		out = append(out, tables[key])
	}
	return out
}

func buildCSV(rows []map[string]any, fields []field, options exportOptions, excel bool) []byte {
	var b bytes.Buffer
	if excel {
		b.Write([]byte{0xef, 0xbb, 0xbf})
	}
	writer := csv.NewWriter(&b)
	writer.UseCRLF = false
	header := make([]string, len(fields))
	for i, f := range fields {
		header[i] = f.SystemName
	}
	_ = writer.Write(header)
	for _, row := range rows {
		record := make([]string, len(fields))
		for i, f := range fields {
			record[i] = formatScalar(row[f.SystemName], f, options, excel)
		}
		_ = writer.Write(record)
	}
	writer.Flush()
	return b.Bytes()
}

func formatScalar(value any, f field, options exportOptions, excel bool) string {
	if value == nil {
		return ""
	}
	if f.Type == "boolean" {
		if b, ok := value.(bool); ok {
			if excel {
				if b {
					return "TRUE"
				}
				return "FALSE"
			}
			if b {
				return "true"
			}
			return "false"
		}
	}
	if f.Type == "datetime" || f.Type == "date" || f.Type == "time" {
		return formatTemporal(value, f.Type, options)
	}
	var text string
	if s, ok := value.(string); ok {
		text = s
	} else if data, err := json.Marshal(value); err == nil && (strings.HasPrefix(string(data), "{") || strings.HasPrefix(string(data), "[")) {
		text = string(data)
	} else {
		text = fmt.Sprint(value)
	}
	if excel && text != "" && strings.ContainsRune("=+-@", rune(text[0])) {
		return "'" + text
	}
	return text
}

func formatTemporal(value any, kind string, options exportOptions) string {
	if t, ok := value.(time.Time); ok {
		switch kind {
		case "date":
			return t.Format("2006-01-02")
		case "time":
			return t.Format("15:04:05")
		default:
			return formatDateTime(t, options)
		}
	}
	text := fmt.Sprint(value)
	if kind == "date" || kind == "time" {
		return text
	}
	t, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return text
	}
	return formatDateTime(t, options)
}

func formatDateTime(t time.Time, options exportOptions) string {
	switch options.TimeFormat {
	case "epoch-sec":
		return strconv.FormatInt(t.Unix(), 10)
	case "epoch-ms":
		return strconv.FormatInt(t.UnixMilli(), 10)
	case "iso-local":
		loc := time.Local
		if options.Timezone != "" {
			if loaded, err := time.LoadLocation(options.Timezone); err == nil {
				loc = loaded
			}
		}
		return t.In(loc).Format(time.RFC3339Nano)
	default:
		return t.Format(time.RFC3339Nano)
	}
}

func exportManifest(dataset exportDataset) map[string]any {
	tables := map[string]any{}
	for _, table := range sortedExportTables(dataset.Tables) {
		fields := exportFields(table.Schema)
		names := make([]string, len(fields))
		for i, f := range fields {
			names[i] = f.SystemName
		}
		tables[table.Schema.TableID] = map[string]any{"recordCount": len(table.Rows), "fields": names}
	}
	return map[string]any{
		"generationIds": dataset.GenerationIDs, "orderedGenerationIds": dataset.OrderedGenerationIDs,
		"format": dataset.Format, "logicalFormat": dataset.LogicalFormat, "summary": dataset.Summary, "tables": tables,
	}
}

func createZip(files []namedFile) ([]byte, error) {
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	for _, file := range files {
		header := &zip.FileHeader{Name: file.Name, Method: zip.Store}
		header.SetModTime(time.Unix(0, 0).UTC())
		w, err := zw.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(file.Data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func buildSQL(dataset exportDataset) []byte {
	var b strings.Builder
	b.WriteString("-- MasterDataMate export\n")
	b.WriteString("-- Generations: ")
	b.WriteString(strings.Join(dataset.OrderedGenerationIDs, ", "))
	b.WriteString("\n\n")
	for _, table := range sortedExportTables(dataset.Tables) {
		fields := exportFields(table.Schema)
		b.WriteString("CREATE TABLE IF NOT EXISTS ")
		b.WriteString(sqlIdent(table.Schema.TableID))
		b.WriteString(" (\n")
		for i, f := range fields {
			if i > 0 {
				b.WriteString(",\n")
			}
			b.WriteString("  ")
			b.WriteString(sqlIdent(f.SystemName))
			b.WriteByte(' ')
			b.WriteString(sqlType(f))
		}
		if len(table.Schema.PrimaryKey) > 0 {
			b.WriteString(",\n  PRIMARY KEY (")
			for i, key := range table.Schema.PrimaryKey {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(sqlIdent(key))
			}
			b.WriteString(")")
		}
		b.WriteString("\n);\n")
		b.WriteString("TRUNCATE TABLE ")
		b.WriteString(sqlIdent(table.Schema.TableID))
		b.WriteString(";\n")
		for _, row := range table.Rows {
			b.WriteString("INSERT INTO ")
			b.WriteString(sqlIdent(table.Schema.TableID))
			b.WriteString(" (")
			for i, f := range fields {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(sqlIdent(f.SystemName))
			}
			b.WriteString(") VALUES (")
			for i, f := range fields {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(sqlLiteral(row[f.SystemName], f, dataset.Options))
			}
			b.WriteString(");\n")
		}
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func sqlType(f field) string {
	switch f.Type {
	case "integer":
		return "INTEGER"
	case "decimal":
		return "REAL"
	case "boolean":
		return "BOOLEAN"
	default:
		return "TEXT"
	}
}

func sqlIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func sqlLiteral(value any, f field, options exportOptions) string {
	if value == nil || value == "" {
		return "NULL"
	}
	switch v := value.(type) {
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	}
	text := formatScalar(value, f, options, false)
	return "'" + strings.ReplaceAll(text, "'", "''") + "'"
}

func buildXLSX(dataset exportDataset) ([]byte, error) {
	var files []namedFile
	tables := sortedExportTables(dataset.Tables)
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>`
	workbookSheets := ""
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`
	used := map[string]bool{}
	for i, table := range tables {
		id := i + 1
		name := sheetName(table.Schema.TableID, used)
		contentTypes += fmt.Sprintf(`<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, id)
		workbookSheets += fmt.Sprintf(`<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, xmlEscape(name), id, id)
		rels += fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, id, id)
		files = append(files, namedFile{Name: fmt.Sprintf("xl/worksheets/sheet%d.xml", id), Data: []byte(buildSheetXML(table, dataset.Options))})
	}
	contentTypes += `</Types>`
	rels += `</Relationships>`
	files = append(files,
		namedFile{Name: "[Content_Types].xml", Data: []byte(contentTypes)},
		namedFile{Name: "_rels/.rels", Data: []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`)},
		namedFile{Name: "xl/workbook.xml", Data: []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>` + workbookSheets + `</sheets></workbook>`)},
		namedFile{Name: "xl/_rels/workbook.xml.rels", Data: []byte(rels)},
	)
	return createZip(files)
}

func buildSQLite(dataset exportDataset) ([]byte, error) {
	tmp, err := os.CreateTemp("", "masterdatamate-export-*.sqlite")
	if err != nil {
		return nil, err
	}
	path := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	defer os.Remove(path)

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	for _, table := range sortedExportTables(dataset.Tables) {
		fields := exportFields(table.Schema)
		if _, err := db.Exec(sqliteCreateTable(table.Schema, fields)); err != nil {
			_ = db.Close()
			return nil, err
		}
		if _, err := db.Exec("DELETE FROM " + sqlIdent(table.Schema.TableID)); err != nil {
			_ = db.Close()
			return nil, err
		}
		if len(fields) == 0 {
			continue
		}
		insertSQL := sqliteInsert(table.Schema.TableID, fields)
		for _, row := range table.Rows {
			values := make([]any, len(fields))
			for i, f := range fields {
				values[i] = sqliteValue(row[f.SystemName], f, dataset.Options)
			}
			if _, err := db.Exec(insertSQL, values...); err != nil {
				_ = db.Close()
				return nil, err
			}
		}
	}
	if err := db.Close(); err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

func sqliteCreateTable(item schema, fields []field) string {
	var b strings.Builder
	b.WriteString("CREATE TABLE IF NOT EXISTS ")
	b.WriteString(sqlIdent(item.TableID))
	b.WriteString(" (")
	for i, f := range fields {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(sqlIdent(f.SystemName))
		b.WriteByte(' ')
		b.WriteString(sqliteType(f))
	}
	if len(item.PrimaryKey) > 0 {
		b.WriteString(", PRIMARY KEY (")
		for i, key := range item.PrimaryKey {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(sqlIdent(key))
		}
		b.WriteString(")")
	}
	b.WriteString(")")
	return b.String()
}

func sqliteInsert(table string, fields []field) string {
	columns := make([]string, len(fields))
	holders := make([]string, len(fields))
	for i, f := range fields {
		columns[i] = sqlIdent(f.SystemName)
		holders[i] = "?"
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", sqlIdent(table), strings.Join(columns, ", "), strings.Join(holders, ", "))
}

func sqliteType(f field) string {
	switch f.Type {
	case "integer":
		return "INTEGER"
	case "decimal":
		return "REAL"
	case "boolean":
		return "INTEGER"
	default:
		return "TEXT"
	}
}

func sqliteValue(value any, f field, options exportOptions) any {
	if value == nil || value == "" {
		return nil
	}
	if f.Type == "boolean" {
		if b, ok := value.(bool); ok {
			if b {
				return 1
			}
			return 0
		}
	}
	if f.Type == "integer" {
		if i, ok := toInt(value); ok {
			return i
		}
	}
	if f.Type == "decimal" {
		if f64, ok := toFloat(value); ok {
			return f64
		}
	}
	return formatScalar(value, f, options, false)
}

func buildSheetXML(table exportTable, options exportOptions) string {
	fields := exportFields(table.Schema)
	rows := [][]string{make([]string, len(fields))}
	for i, f := range fields {
		rows[0][i] = f.SystemName
	}
	for _, row := range table.Rows {
		record := make([]string, len(fields))
		for i, f := range fields {
			record[i] = formatScalar(row[f.SystemName], f, options, false)
		}
		rows = append(rows, record)
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for rowIndex, row := range rows {
		b.WriteString(fmt.Sprintf(`<row r="%d">`, rowIndex+1))
		for colIndex, value := range row {
			ref := cellRef(colIndex, rowIndex+1)
			b.WriteString(fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`, ref, xmlEscape(value)))
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData></worksheet>`)
	return b.String()
}

func sheetName(name string, used map[string]bool) string {
	replacer := strings.NewReplacer("\\", "_", "/", "_", "?", "_", "*", "_", ":", "_", "[", "_", "]", "_")
	base := replacer.Replace(name)
	if len(base) > 31 {
		base = base[:31]
	}
	if base == "" {
		base = "Sheet"
	}
	candidate := base
	for index := 2; used[candidate]; index++ {
		suffix := fmt.Sprintf("_%d", index)
		candidate = base
		if len(candidate)+len(suffix) > 31 {
			candidate = candidate[:31-len(suffix)]
		}
		candidate += suffix
	}
	used[candidate] = true
	return candidate
}

func cellRef(col int, row int) string {
	name := ""
	for col >= 0 {
		name = string(rune('A'+(col%26))) + name
		col = col/26 - 1
	}
	return fmt.Sprintf("%s%d", name, row)
}

func xmlEscape(value string) string {
	return html.EscapeString(value)
}
