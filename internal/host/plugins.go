package host

import (
	"encoding/json"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type editorPluginsFile struct {
	Plugins []editorPlugin `json:"plugins" yaml:"plugins"`
}

type editorPlugin struct {
	PluginID       string              `json:"plugin_id" yaml:"plugin_id"`
	DisplayName    string              `json:"display_name" yaml:"display_name"`
	Description    string              `json:"description,omitempty" yaml:"description,omitempty"`
	EntryHTML      string              `json:"entry_html" yaml:"entry_html"`
	SourceDir      string              `json:"source_dir,omitempty" yaml:"source_dir,omitempty"`
	Build          map[string]any      `json:"build,omitempty" yaml:"build,omitempty"`
	Version        string              `json:"version,omitempty" yaml:"version,omitempty"`
	OpenMode       string              `json:"open_mode,omitempty" yaml:"open_mode,omitempty"`
	EntryPoints    []pluginEntryPoint  `json:"entry_points,omitempty" yaml:"entry_points,omitempty"`
	TargetTables   []pluginTargetTable `json:"target_tables,omitempty" yaml:"target_tables,omitempty"`
	GroupBy        map[string]any      `json:"group_by,omitempty" yaml:"group_by,omitempty"`
	Permissions    pluginPermissions   `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	Capabilities   []string            `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	SettingsSchema map[string]any      `json:"settings_schema,omitempty" yaml:"settings_schema,omitempty"`
}

type pluginEntryPoint struct {
	EntryID     string `json:"entry_id,omitempty" yaml:"entry_id,omitempty"`
	ID          string `json:"id,omitempty" yaml:"id,omitempty"`
	Placement   string `json:"placement" yaml:"placement"`
	Label       string `json:"label,omitempty" yaml:"label,omitempty"`
	Table       string `json:"table,omitempty" yaml:"table,omitempty"`
	OpenMode    string `json:"open_mode,omitempty" yaml:"open_mode,omitempty"`
	Default     bool   `json:"default,omitempty" yaml:"default,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type pluginTargetTable struct {
	Role     string         `json:"role" yaml:"role"`
	Table    string         `json:"table" yaml:"table"`
	Required bool           `json:"required,omitempty" yaml:"required,omitempty"`
	Write    bool           `json:"write,omitempty" yaml:"write,omitempty"`
	Filter   map[string]any `json:"filter,omitempty" yaml:"filter,omitempty"`
}

type pluginPermissions struct {
	ReadTables    []string `json:"read_tables,omitempty" yaml:"read_tables,omitempty"`
	WriteTables   []string `json:"write_tables,omitempty" yaml:"write_tables,omitempty"`
	ReadBinaries  []string `json:"read_binaries,omitempty" yaml:"read_binaries,omitempty"`
	WriteBinaries []string `json:"write_binaries,omitempty" yaml:"write_binaries,omitempty"`
}

type pluginContextRequest struct {
	ActiveGenerationID string         `json:"activeGenerationId"`
	Mode               string         `json:"mode"`
	EntryPointID       string         `json:"entryPointId"`
	Entry              map[string]any `json:"entry"`
}

type pluginChangesRequest struct {
	pluginContextRequest
	Changes map[string]any `json:"changes"`
	Force   bool           `json:"force"`
}

func (s *server) dispatchEditorPluginAPI(r *http.Request, parts []string) (int, any, string, []byte, error) {
	if len(parts) == 2 && r.Method == http.MethodGet {
		return s.editorPluginsIndex()
	}
	if len(parts) < 3 {
		return 404, nil, "", nil, appError{404, "API route not found"}
	}
	plugin, err := s.loadEditorPlugin(parts[2])
	if err != nil {
		return 0, nil, "", nil, err
	}
	if len(parts) >= 4 && parts[3] == "assets" && r.Method == http.MethodGet {
		assetPath := strings.Join(parts[4:], "/")
		return s.serveEditorPluginAsset(plugin, assetPath)
	}
	if len(parts) == 4 && parts[3] == "context" && r.Method == http.MethodPost {
		var body pluginContextRequest
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		payload, err := s.editorPluginContext(plugin, body)
		return 200, payload, "", nil, err
	}
	if len(parts) == 5 && parts[3] == "changes" && (parts[4] == "validate" || parts[4] == "commit") && r.Method == http.MethodPost {
		var body pluginChangesRequest
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		if parts[4] == "validate" {
			payload, err := s.validatePluginChanges(plugin, body)
			return 200, payload, "", nil, err
		}
		payload, status, err := s.commitPluginChanges(plugin, body)
		return status, payload, "", nil, err
	}
	return 404, nil, "", nil, appError{404, "API route not found"}
}

func (s *server) editorPluginsIndex() (int, any, string, []byte, error) {
	plugins, err := s.loadEditorPlugins()
	if err != nil {
		return 0, nil, "", nil, err
	}
	schemas, err := s.loadSchemas()
	if err != nil {
		return 0, nil, "", nil, err
	}
	visibility := map[string]string{}
	for _, item := range schemas {
		visibility[item.TableID] = tableListVisibility(item)
	}
	return 200, map[string]any{"plugins": plugins, "tableVisibility": visibility}, "", nil, nil
}

func (s *server) loadEditorPlugins() ([]editorPlugin, error) {
	file := filepath.Join(s.root, "masterdata", "editor_plugins.yaml")
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return []editorPlugin{}, nil
	}
	var raw editorPluginsFile
	if err := s.readYAML(file, &raw); err != nil {
		return nil, err
	}
	out := make([]editorPlugin, 0, len(raw.Plugins))
	seen := map[string]bool{}
	for _, plugin := range raw.Plugins {
		normalized, err := s.normalizeEditorPlugin(plugin)
		if err != nil {
			return nil, err
		}
		if seen[normalized.PluginID] {
			return nil, appError{422, "Duplicate editor plugin id: " + normalized.PluginID}
		}
		seen[normalized.PluginID] = true
		out = append(out, normalized)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PluginID < out[j].PluginID })
	return out, nil
}

func (s *server) loadEditorPlugin(pluginID string) (editorPlugin, error) {
	plugins, err := s.loadEditorPlugins()
	if err != nil {
		return editorPlugin{}, err
	}
	for _, plugin := range plugins {
		if plugin.PluginID == pluginID {
			return plugin, nil
		}
	}
	return editorPlugin{}, appError{404, "Editor plugin not found: " + pluginID}
}

func (s *server) normalizeEditorPlugin(plugin editorPlugin) (editorPlugin, error) {
	if !safePathSegment(plugin.PluginID) {
		return plugin, appError{422, "Editor plugin plugin_id must be a safe path segment."}
	}
	if strings.TrimSpace(plugin.EntryHTML) == "" {
		return plugin, appError{422, "Editor plugin entry_html is required: " + plugin.PluginID}
	}
	if plugin.DisplayName == "" {
		plugin.DisplayName = plugin.PluginID
	}
	if plugin.OpenMode == "" {
		plugin.OpenMode = "record"
	}
	if len(plugin.EntryPoints) == 0 {
		plugin.EntryPoints = derivedPluginEntryPoints(plugin)
	}
	for i := range plugin.EntryPoints {
		if plugin.EntryPoints[i].EntryID == "" {
			plugin.EntryPoints[i].EntryID = plugin.EntryPoints[i].ID
		}
		if plugin.EntryPoints[i].EntryID == "" {
			plugin.EntryPoints[i].EntryID = plugin.EntryPoints[i].Placement
		}
		if plugin.EntryPoints[i].ID == "" {
			plugin.EntryPoints[i].ID = plugin.EntryPoints[i].EntryID
		}
		if plugin.EntryPoints[i].Label == "" {
			plugin.EntryPoints[i].Label = plugin.DisplayName
		}
		if plugin.EntryPoints[i].OpenMode == "" {
			plugin.EntryPoints[i].OpenMode = plugin.OpenMode
		}
	}
	entry, err := s.editorPluginEntryFile(plugin)
	if err != nil {
		return plugin, err
	}
	if _, err := os.Stat(entry); err != nil {
		return plugin, appError{404, "Editor plugin entry_html is missing: " + plugin.EntryHTML}
	}
	return plugin, nil
}

func derivedPluginEntryPoints(plugin editorPlugin) []pluginEntryPoint {
	table := primaryPluginTable(plugin)
	switch plugin.OpenMode {
	case "table":
		return []pluginEntryPoint{{EntryID: "table", ID: "table", Placement: "table_toolbar", Table: table, OpenMode: "table", Default: true}}
	case "group":
		return []pluginEntryPoint{{EntryID: "group", ID: "group", Placement: "group_action", Table: table, OpenMode: "group", Default: true}}
	default:
		return []pluginEntryPoint{{EntryID: "record", ID: "record", Placement: "record_action", Table: table, OpenMode: "record", Default: true}}
	}
}

func primaryPluginTable(plugin editorPlugin) string {
	for _, target := range plugin.TargetTables {
		if target.Role == "primary" {
			return target.Table
		}
	}
	if len(plugin.TargetTables) > 0 {
		return plugin.TargetTables[0].Table
	}
	return ""
}

func (s *server) editorPluginEntryFile(plugin editorPlugin) (string, error) {
	entry := filepath.Join(s.root, "masterdata", filepath.FromSlash(plugin.EntryHTML))
	if err := ensurePathInside(filepath.Join(s.root, "masterdata"), entry); err != nil {
		return "", err
	}
	return entry, nil
}

func (s *server) serveEditorPluginAsset(plugin editorPlugin, assetPath string) (int, any, string, []byte, error) {
	entry, err := s.editorPluginEntryFile(plugin)
	if err != nil {
		return 0, nil, "", nil, err
	}
	root := filepath.Dir(entry)
	if assetPath == "" {
		assetPath = filepath.Base(entry)
	}
	file := filepath.Join(root, filepath.FromSlash(pathClean(assetPath)))
	if err := ensurePathInside(root, file); err != nil {
		return 0, nil, "", nil, err
	}
	data, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return 0, nil, "", nil, appError{404, "Editor plugin asset not found."}
	}
	if err != nil {
		return 0, nil, "", nil, err
	}
	contentType := mime.TypeByExtension(filepath.Ext(file))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	return 200, nil, contentType, data, nil
}

func pathClean(value string) string {
	cleaned := filepath.ToSlash(filepath.Clean("/" + value))
	return strings.TrimPrefix(cleaned, "/")
}

func (s *server) editorPluginContext(plugin editorPlugin, req pluginContextRequest) (map[string]any, error) {
	if req.ActiveGenerationID == "" {
		return nil, appError{400, "activeGenerationId is required."}
	}
	if err := s.requireGeneration(req.ActiveGenerationID); err != nil {
		return nil, err
	}
	mode := req.Mode
	if mode == "" {
		mode = "active_only"
	}
	entryPoint := selectPluginEntryPoint(plugin, req.EntryPointID)
	sourceTable := stringValue(req.Entry["table"], primaryPluginTable(plugin))
	sourceKey := req.Entry["key"]
	sourceRow, err := s.pluginSourceRow(sourceTable, sourceKey, req.ActiveGenerationID, mode)
	if err != nil {
		return nil, err
	}
	tables := map[string]any{}
	for _, target := range plugin.TargetTables {
		bundle, err := s.pluginTableBundle(target, sourceTable, sourceRow, req.ActiveGenerationID, mode)
		if err != nil {
			return nil, err
		}
		tables[target.Table] = bundle
	}
	return map[string]any{
		"pluginId":           plugin.PluginID,
		"generationId":       req.ActiveGenerationID,
		"mode":               mode,
		"entryPoint":         entryPoint,
		"entry":              req.Entry,
		"primarySelection":   map[string]any{"table": sourceTable, "key": sourceKey, "row": sourceRow},
		"tables":             tables,
		"capabilities":       plugin.Capabilities,
		"settingsSchema":     plugin.SettingsSchema,
		"hostApiVersion":     "1",
		"binaryAssetBaseUrl": "/api/binaries",
	}, nil
}

func selectPluginEntryPoint(plugin editorPlugin, id string) pluginEntryPoint {
	for _, entry := range plugin.EntryPoints {
		if pluginEntryPointID(entry) == id || entry.ID == id {
			return entry
		}
	}
	for _, entry := range plugin.EntryPoints {
		if entry.Default {
			return entry
		}
	}
	if len(plugin.EntryPoints) > 0 {
		return plugin.EntryPoints[0]
	}
	return pluginEntryPoint{}
}

func pluginEntryPointID(entry pluginEntryPoint) string {
	if entry.EntryID != "" {
		return entry.EntryID
	}
	return entry.ID
}

func (s *server) pluginSourceRow(table string, key any, generationID, mode string) (map[string]any, error) {
	if table == "" || key == nil {
		return nil, nil
	}
	payload, err := s.loadGenerationView(table, generationID, mode)
	if err != nil {
		return nil, err
	}
	schema := payload["schema"].(schema)
	for _, row := range payload["rows"].([]map[string]any) {
		if normalizeComparable(keyFromRow(row, schema)) == normalizeComparable(key) {
			return row, nil
		}
	}
	return nil, nil
}

func (s *server) pluginTableBundle(target pluginTargetTable, sourceTable string, sourceRow map[string]any, generationID, mode string) (map[string]any, error) {
	payload, err := s.loadGenerationView(target.Table, generationID, mode)
	if err != nil {
		return nil, err
	}
	schema := payload["schema"].(schema)
	rows := payload["rows"].([]map[string]any)
	filtered := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if pluginRowMatchesTarget(target, row, sourceTable, sourceRow, schema) {
			filtered = append(filtered, row)
		}
	}
	records := make([]record, 0, len(filtered))
	for _, row := range filtered {
		records = append(records, rowToRecord(row, schema))
	}
	return map[string]any{"role": target.Role, "schema": schema, "records": records, "rows": filtered, "write": target.Write}, nil
}

func pluginRowMatchesTarget(target pluginTargetTable, row map[string]any, sourceTable string, sourceRow map[string]any, schema schema) bool {
	if target.Role == "lookup" || target.Role == "readonly_context" {
		return true
	}
	if target.Role == "primary" {
		if sourceTable == "" || target.Table != sourceTable || sourceRow == nil {
			return true
		}
		return normalizeComparable(keyFromRow(row, schema)) == normalizeComparable(keyFromRow(sourceRow, schema))
	}
	filter := target.Filter
	if stringValue(filter["mode"], "equals") != "equals" {
		return true
	}
	if sourceRow == nil || sourceTable != stringValue(filter["source_table"], sourceTable) {
		return false
	}
	sourceField := stringValue(filter["source_field"], "")
	targetField := stringValue(filter["target_field"], "")
	if sourceField == "" || targetField == "" {
		return true
	}
	return normalizeComparable(row[targetField]) == normalizeComparable(sourceRow[sourceField])
}

func (s *server) validatePluginChanges(plugin editorPlugin, req pluginChangesRequest) (map[string]any, error) {
	return s.applyPluginChanges(plugin, req, false)
}

func (s *server) commitPluginChanges(plugin editorPlugin, req pluginChangesRequest) (map[string]any, int, error) {
	payload, err := s.applyPluginChanges(plugin, req, true)
	if err != nil {
		return nil, 0, err
	}
	if boolValue(payload["requiresForce"]) {
		return payload, 409, nil
	}
	return payload, 200, nil
}

func (s *server) applyPluginChanges(plugin editorPlugin, req pluginChangesRequest, commit bool) (map[string]any, error) {
	if req.ActiveGenerationID == "" {
		return nil, appError{400, "activeGenerationId is required."}
	}
	tablesRaw, _ := req.Changes["tables"].(map[string]any)
	if tablesRaw == nil {
		return map[string]any{"saved": false, "diagnostics": []any{}}, nil
	}
	writeable := pluginWriteTables(plugin)
	allDiagnostics := []map[string]any{}
	results := map[string]any{}
	for table, raw := range tablesRaw {
		if !writeable[table] {
			return nil, appError{403, "Plugin cannot write table: " + table}
		}
		nextRows, err := s.rowsAfterPluginTableChanges(table, req.ActiveGenerationID, raw)
		if err != nil {
			return nil, err
		}
		body := map[string]any{"rows": nextRows, "force": req.Force, "mode": req.Mode}
		if !commit {
			diagnostics, err := s.validateRows(table, req.ActiveGenerationID, nextRows, stringValue(req.Mode, "active_only"))
			if err != nil {
				return nil, err
			}
			allDiagnostics = append(allDiagnostics, diagnostics...)
			results[table] = map[string]any{"diagnostics": diagnostics, "rows": nextRows}
			continue
		}
		payload, status, err := s.commitRows(table, req.ActiveGenerationID, body)
		if err != nil {
			return nil, err
		}
		if diagnostics, ok := payload["diagnostics"].([]map[string]any); ok {
			allDiagnostics = append(allDiagnostics, diagnostics...)
		}
		results[table] = payload
		if status == http.StatusConflict {
			return map[string]any{"saved": false, "requiresForce": true, "diagnostics": allDiagnostics, "tables": results}, nil
		}
	}
	return map[string]any{"saved": commit, "diagnostics": allDiagnostics, "tables": results}, nil
}

func pluginWriteTables(plugin editorPlugin) map[string]bool {
	out := map[string]bool{}
	for _, table := range plugin.Permissions.WriteTables {
		out[table] = true
	}
	for _, target := range plugin.TargetTables {
		if target.Write {
			out[target.Table] = true
		}
	}
	return out
}

func (s *server) rowsAfterPluginTableChanges(table, generationID string, raw any) ([]map[string]any, error) {
	schema, err := s.loadSchema(table)
	if err != nil {
		return nil, err
	}
	records, err := s.loadRecords(table, generationID)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(records))
	byKey := map[string]int{}
	for _, item := range records {
		row := recordToRow(item, schema)
		byKey[normalizeComparable(keyFromRow(row, schema))] = len(rows)
		rows = append(rows, row)
	}
	tableChange, _ := raw.(map[string]any)
	for _, rawDelete := range sliceAny(tableChange["deletes"]) {
		key := rawDelete
		if m, ok := rawDelete.(map[string]any); ok && m["key"] != nil {
			key = m["key"]
		}
		comparable := normalizeComparable(key)
		if index, ok := byKey[comparable]; ok {
			rows = append(rows[:index], rows[index+1:]...)
			byKey = reindexRows(rows, schema)
		}
	}
	for _, rawUpdate := range sliceAny(tableChange["updates"]) {
		change, _ := rawUpdate.(map[string]any)
		row, err := pluginChangeRecordRow(change["record"], schema)
		if err != nil {
			return nil, err
		}
		key := change["previousKey"]
		if key == nil {
			key = keyFromRow(row, schema)
		}
		comparable := normalizeComparable(key)
		if index, ok := byKey[comparable]; ok {
			rows[index] = row
		} else {
			rows = append(rows, row)
		}
		byKey = reindexRows(rows, schema)
	}
	for _, rawInsert := range sliceAny(tableChange["inserts"]) {
		row, err := pluginChangeRecordRow(rawInsert, schema)
		if err != nil {
			return nil, err
		}
		comparable := normalizeComparable(keyFromRow(row, schema))
		if index, ok := byKey[comparable]; ok {
			rows[index] = row
		} else {
			byKey[comparable] = len(rows)
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func pluginChangeRecordRow(value any, schema schema) (map[string]any, error) {
	if row, ok := value.(map[string]any); ok {
		if data, ok := row["data"].(map[string]any); ok || row["key"] != nil {
			item := record{Key: row["key"], Name: stringValue(row["name"], ""), Data: data}
			return recordToRow(item, schema), nil
		}
		return row, nil
	}
	var item record
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, err
	}
	return recordToRow(item, schema), nil
}

func reindexRows(rows []map[string]any, schema schema) map[string]int {
	out := map[string]int{}
	for i, row := range rows {
		out[normalizeComparable(keyFromRow(row, schema))] = i
	}
	return out
}

func tableListVisibility(item schema) string {
	if item.UI == nil {
		return "visible"
	}
	visibility := stringValue(item.UI["table_list_visibility"], "visible")
	if visibility == "plugin_only" || visibility == "hidden" {
		return visibility
	}
	return "visible"
}
