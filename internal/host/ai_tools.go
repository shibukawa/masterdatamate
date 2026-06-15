package host

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const defaultAIToolPageLimit = 30
const maxAIToolPageLimit = 100

func (s *server) executeAITool(toolName string, args map[string]any) (map[string]any, error) {
	switch toolName {
	case "get_current_context":
		return map[string]any{
			"success": true,
			"summary": "Current UI context returned.",
			"context": args["context"],
			"capabilities": []string{
				"get_current_context",
				"get_project_overview",
				"get_table",
				"validate_table",
				"stage_table_changes",
			},
		}, nil
	case "get_project_overview":
		return s.aiProjectOverview()
	case "get_table":
		return s.aiGetTable(args)
	case "validate_table":
		return s.aiValidateTable(args)
	case "stage_table_changes":
		return map[string]any{
			"success":                   true,
			"summary":                   "Table changes require frontend editor staging.",
			"requires_frontend_staging": true,
			"change_set":                args,
		}, nil
	default:
		return nil, appError{404, "Unknown AI tool: " + toolName}
	}
}

func (s *server) aiProjectOverview() (map[string]any, error) {
	schemas, err := s.loadSchemas()
	if err != nil {
		return nil, err
	}
	generationsPayload, err := s.loadGenerations()
	if err != nil {
		return nil, err
	}
	tables := make([]map[string]any, 0, len(schemas))
	for _, item := range schemas {
		if visibility := tableListVisibility(item); visibility == "plugin_only" || visibility == "hidden" {
			continue
		}
		tables = append(tables, map[string]any{
			"table_id":      item.TableID,
			"business_name": item.BusinessName,
			"comment":       item.Comment,
			"primary_key":   item.PrimaryKey,
			"field_count":   len(item.Fields),
			"export":        item.Export,
		})
	}
	sort.Slice(tables, func(i, j int) bool {
		return fmt.Sprint(tables[i]["table_id"]) < fmt.Sprint(tables[j]["table_id"])
	})
	return map[string]any{
		"success":             true,
		"summary":             fmt.Sprintf("Project has %d visible table(s).", len(tables)),
		"tables":              tables,
		"generations":         generationsPayload["generations"],
		"generation_settings": generationsPayload["settings"],
		"capabilities": map[string]any{
			"table_record_draft_staging": true,
			"schema_editing":             false,
			"file_import":                false,
			"binary_attachment":          false,
			"export_execution":           false,
		},
	}, nil
}

func (s *server) aiGetTable(args map[string]any) (map[string]any, error) {
	tableID := stringValue(args["tableId"], "")
	if tableID == "" {
		tableID = stringValue(args["table_id"], "")
	}
	if tableID == "" {
		return nil, appError{400, "tableId is required."}
	}
	generationID := stringValue(args["generationId"], "")
	if generationID == "" {
		generationID = stringValue(args["generation_id"], defaultGeneration)
	}
	mode := stringValue(args["mode"], "active_only")
	offset := aiIntValue(args["offset"], 0)
	limit := aiIntValue(args["limit"], defaultAIToolPageLimit)
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = defaultAIToolPageLimit
	}
	if limit > maxAIToolPageLimit {
		limit = maxAIToolPageLimit
	}
	payload, err := s.loadGenerationView(tableID, generationID, mode)
	if err != nil {
		return nil, err
	}
	rows := sliceMap(payload["rows"])
	total := len(rows)
	end := offset + limit
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
		end = total
	}
	pageRows := rows[offset:end]
	fields := aiStringSlice(args["fields"])
	if len(fields) > 0 {
		pageRows = projectRows(pageRows, fields)
	}
	return map[string]any{
		"success":     true,
		"summary":     fmt.Sprintf("Returned %d of %d row(s) for %s.", len(pageRows), total, tableID),
		"schema":      aiSchemaSummary(payload["schema"]),
		"records":     pageRows,
		"diagnostics": payload["diagnostics"],
		"pagination": map[string]any{
			"offset": offset,
			"limit":  limit,
			"total":  total,
			"next_offset": func() any {
				if end >= total {
					return nil
				}
				return end
			}(),
		},
	}, nil
}

func (s *server) aiValidateTable(args map[string]any) (map[string]any, error) {
	tableID := stringValue(args["tableId"], "")
	if tableID == "" {
		tableID = stringValue(args["table_id"], "")
	}
	if tableID == "" {
		return nil, appError{400, "tableId is required."}
	}
	generationID := stringValue(args["generationId"], "")
	if generationID == "" {
		generationID = stringValue(args["generation_id"], defaultGeneration)
	}
	mode := stringValue(args["mode"], "active_only")
	rows := sliceMap(args["rows"])
	if len(rows) == 0 {
		rows = rowsFromJSONString(args["rows_json"])
	}
	if len(rows) == 0 {
		payload, err := s.loadGenerationView(tableID, generationID, mode)
		if err != nil {
			return nil, err
		}
		rows = sliceMap(payload["rows"])
	}
	diagnostics, err := s.validateRows(tableID, generationID, rows, mode)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"success":     true,
		"summary":     fmt.Sprintf("Validation returned %d diagnostic(s).", len(diagnostics)),
		"diagnostics": diagnostics,
	}, nil
}

func aiSchemaSummary(value any) map[string]any {
	schemaMap, _ := value.(map[string]any)
	if schemaMap == nil {
		if item, ok := value.(schema); ok {
			fields := make([]map[string]any, 0, len(item.Fields))
			for _, field := range item.Fields {
				fields = append(fields, map[string]any{
					"system_name":   field.SystemName,
					"business_name": field.BusinessName,
					"type":          field.Type,
					"required":      field.Required,
					"reference":     field.Reference,
					"comment":       field.Comment,
				})
			}
			return map[string]any{
				"table_id":      item.TableID,
				"business_name": item.BusinessName,
				"primary_key":   item.PrimaryKey,
				"fields":        fields,
				"comment":       item.Comment,
			}
		}
		return map[string]any{}
	}
	return schemaMap
}

func projectRows(rows []map[string]any, fields []string) []map[string]any {
	include := map[string]bool{
		"sourceGenerationId":       true,
		"sourceGenerationLabel":    true,
		"isActiveGeneration":       true,
		"isReadOnly":               true,
		"isOverridden":             true,
		"overriddenByGenerationId": true,
	}
	for _, field := range fields {
		include[field] = true
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		next := map[string]any{}
		for key, value := range row {
			if include[key] {
				next[key] = value
			}
		}
		out = append(out, next)
	}
	return out
}

func sliceMap(value any) []map[string]any {
	switch rows := value.(type) {
	case []map[string]any:
		return rows
	case []any:
		out := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			if item, ok := row.(map[string]any); ok {
				out = append(out, item)
			}
		}
		return out
	default:
		return []map[string]any{}
	}
}

func rowsFromJSONString(value any) []map[string]any {
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" || strings.TrimSpace(text) == "[]" {
		return []map[string]any{}
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(text), &rows); err != nil {
		return []map[string]any{}
	}
	return rows
}

func aiStringSlice(value any) []string {
	raw := sliceAny(value)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if text := fmt.Sprint(item); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func aiIntValue(value any, fallback int) int {
	switch item := value.(type) {
	case int:
		return item
	case int64:
		return int(item)
	case float64:
		return int(item)
	case jsonNumber:
		if parsed, err := item.Int64(); err == nil {
			return int(parsed)
		}
	}
	return fallback
}

type jsonNumber interface {
	Int64() (int64, error)
}
