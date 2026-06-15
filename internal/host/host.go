package host

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultGeneration = "0000_initial"

type appError struct {
	status int
	msg    string
}

func (e appError) Error() string { return e.msg }

type server struct {
	root           string
	schemaRoot     string
	generationRoot string
	binaryRoot     string
	exports        map[string]exportArtifact
	static         fs.FS
	aiFrontendMu   sync.Mutex
	aiFrontendWait map[string]chan aiFrontendToolResult
}

type exportArtifact struct {
	data        []byte
	filename    string
	contentType string
	createdAt   time.Time
}

type schema struct {
	TableID         string           `json:"table_id" yaml:"-"`
	SystemName      string           `json:"system_name" yaml:"system_name"`
	BusinessName    string           `json:"business_name" yaml:"business_name"`
	PrimaryKey      []string         `json:"primary_key" yaml:"primary_key"`
	Export          bool             `json:"export" yaml:"export"`
	DependentTables []map[string]any `json:"dependent_tables,omitempty" yaml:"dependent_tables,omitempty"`
	UI              map[string]any   `json:"ui,omitempty" yaml:"ui,omitempty"`
	Comment         string           `json:"comment" yaml:"comment,omitempty"`
	Fields          []field          `json:"fields" yaml:"fields"`
}

type field struct {
	SystemName   string         `json:"system_name" yaml:"system_name"`
	BusinessName string         `json:"business_name" yaml:"business_name,omitempty"`
	Type         string         `json:"type" yaml:"type,omitempty"`
	Required     bool           `json:"required" yaml:"required,omitempty"`
	Export       *bool          `json:"export,omitempty" yaml:"export,omitempty"`
	Reference    map[string]any `json:"reference,omitempty" yaml:"reference,omitempty"`
	Constants    []string       `json:"constants,omitempty" yaml:"constants,omitempty"`
	Binary       map[string]any `json:"binary,omitempty" yaml:"binary,omitempty"`
	DefaultValue any            `json:"default_value,omitempty" yaml:"default_value,omitempty"`
	Formula      string         `json:"formula,omitempty" yaml:"formula,omitempty"`
	Comment      string         `json:"comment,omitempty" yaml:"comment,omitempty"`
}

type generationSettings struct {
	OrderingMode  string `json:"ordering_mode" yaml:"ordering_mode"`
	NumericDigits int    `json:"numeric_digits" yaml:"numeric_digits"`
}

type generation struct {
	ID                string `json:"id"`
	FolderName        string `json:"folder_name"`
	DerivedFolderName string `json:"derived_folder_name"`
	GenerationIndex   any    `json:"generation_index" yaml:"generation_index"`
	Output            bool   `json:"output" yaml:"output"`
	PathName          string `json:"path_name" yaml:"path_name"`
	Description       string `json:"description" yaml:"description"`
}

type record struct {
	Key      any              `json:"key" yaml:"key"`
	Name     string           `json:"name,omitempty" yaml:"name,omitempty"`
	Data     map[string]any   `json:"data,omitempty" yaml:"data,omitempty"`
	Children []map[string]any `json:"children,omitempty" yaml:"children,omitempty"`
}

func New(root string, embeddedDist fs.FS) (*server, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	static, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		return nil, err
	}
	s := &server{
		root:           absRoot,
		schemaRoot:     filepath.Join(absRoot, "masterdata", "schema"),
		generationRoot: filepath.Join(absRoot, "masterdata", "generations"),
		binaryRoot:     filepath.Join(absRoot, "masterdata", "binaries"),
		exports:        map[string]exportArtifact{},
		static:         static,
		aiFrontendWait: map[string]chan aiFrontendToolResult{},
	}
	if err := s.validateWorkspace(); err != nil {
		return nil, err
	}
	return s, nil
}

func NewData(root string) (*server, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	s := &server{
		root:           absRoot,
		schemaRoot:     filepath.Join(absRoot, "masterdata", "schema"),
		generationRoot: filepath.Join(absRoot, "masterdata", "generations"),
		binaryRoot:     filepath.Join(absRoot, "masterdata", "binaries"),
		exports:        map[string]exportArtifact{},
		aiFrontendWait: map[string]chan aiFrontendToolResult{},
	}
	if err := s.validateWorkspace(); err != nil {
		return nil, err
	}
	return s, nil
}

func Handler(root string, embeddedDist fs.FS) (http.Handler, error) {
	s, err := New(root, embeddedDist)
	if err != nil {
		return nil, err
	}
	return s.routes(), nil
}

func ListenAndServe(host string, port int, root string, embeddedDist fs.FS) error {
	s, err := New(root, embeddedDist)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	return http.ListenAndServe(addr, s.routes())
}

func (s *server) validateWorkspace() error {
	for _, dir := range []string{s.schemaRoot, s.generationRoot} {
		info, err := os.Stat(dir)
		if err != nil {
			return fmt.Errorf("workspace is missing required directory %s: %w", dir, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("workspace path is not a directory: %s", dir)
		}
	}
	return nil
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	return withJSONErrors(mux)
}

func withJSONErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": fmt.Sprint(recovered)})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *server) handle(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		s.handleAPI(w, r)
		return
	}
	s.handleStatic(w, r)
}

func (s *server) handleAPI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/ai/runs/stream" {
		s.handleAIRunStream(w, r)
		return
	}
	if r.URL.Path == "/api/ai/frontend-tool-results" {
		s.handleAIFrontendToolResult(w, r)
		return
	}
	status, payload, contentType, data, err := s.dispatchAPI(r)
	if err != nil {
		var ae appError
		if errors.As(err, &ae) {
			writeJSON(w, ae.status, map[string]any{"error": ae.msg})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if data != nil {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(status)
		_, _ = w.Write(data)
		return
	}
	writeJSON(w, status, payload)
}

func (s *server) dispatchAPI(r *http.Request) (int, any, string, []byte, error) {
	parts := splitPath(r.URL.Path)
	if len(parts) == 0 || parts[0] != "api" {
		return 404, nil, "", nil, appError{404, "API route not found"}
	}
	if len(parts) == 2 && parts[1] == "tables" && r.Method == http.MethodGet {
		schemas, err := s.loadSchemas()
		if err != nil {
			return 0, nil, "", nil, err
		}
		tables := make([]map[string]any, 0, len(schemas))
		for _, schema := range schemas {
			if visibility := tableListVisibility(schema); visibility == "plugin_only" || visibility == "hidden" {
				continue
			}
			tables = append(tables, map[string]any{
				"table_id": schema.TableID, "system_name": schema.SystemName, "business_name": schema.BusinessName, "comment": schema.Comment,
			})
		}
		return 200, map[string]any{"generationId": defaultGeneration, "tables": tables}, "", nil, nil
	}
	if len(parts) >= 2 && parts[1] == "editor-plugins" {
		return s.dispatchEditorPluginAPI(r, parts)
	}
	if len(parts) >= 3 && parts[1] == "tables" {
		return s.dispatchTableAPI(r, parts)
	}
	if len(parts) >= 3 && parts[1] == "binaries" {
		return s.dispatchBinaryAPI(r, parts)
	}
	if len(parts) >= 2 && parts[1] == "schemas" {
		return s.dispatchSchemaAPI(r, parts)
	}
	if len(parts) >= 2 && parts[1] == "generations" {
		return s.dispatchGenerationAPI(r, parts)
	}
	if len(parts) >= 2 && parts[1] == "ai" {
		return s.dispatchAIAPI(r, parts)
	}
	if len(parts) >= 2 && parts[1] == "export-settings" {
		return s.dispatchExportSettingsAPI(r, parts)
	}
	if len(parts) >= 2 && parts[1] == "generate-definitions" {
		return s.dispatchExportDefinitionsAPI(r, parts)
	}
	if len(parts) >= 2 && parts[1] == "generate" {
		return s.dispatchGenerateAPI(r, parts)
	}
	if len(parts) >= 2 && parts[1] == "exports" {
		return s.dispatchExportAPI(r, parts)
	}
	return 404, nil, "", nil, appError{404, "API route not found"}
}

func (s *server) dispatchTableAPI(r *http.Request, parts []string) (int, any, string, []byte, error) {
	table := parts[2]
	if len(parts) == 4 && parts[3] == "schema" && r.Method == http.MethodGet {
		schema, err := s.loadSchema(table)
		return 200, map[string]any{"schema": schema}, "", nil, err
	}
	if len(parts) == 4 && parts[3] == "generation-view" && r.Method == http.MethodGet {
		generationID := r.URL.Query().Get("activeGenerationId")
		if generationID == "" {
			return 0, nil, "", nil, appError{400, "activeGenerationId is required."}
		}
		mode := queryDefault(r, "mode", "active_only")
		payload, err := s.loadGenerationView(table, generationID, mode)
		return 200, payload, "", nil, err
	}
	if len(parts) == 4 && parts[3] == "references" && r.Method == http.MethodGet {
		generationID := r.URL.Query().Get("activeGenerationId")
		mode := r.URL.Query().Get("mode")
		if generationID == "" {
			generationID = queryDefault(r, "generationId", defaultGeneration)
		}
		payload, err := s.referenceCandidates(table, generationID, mode)
		return 200, payload, "", nil, err
	}
	if len(parts) == 6 && parts[3] == "generations" && parts[5] == "records" && r.Method == http.MethodGet {
		payload, err := s.loadRows(table, parts[4])
		return 200, payload, "", nil, err
	}
	if len(parts) == 6 && parts[3] == "generations" && parts[5] == "validate" && r.Method == http.MethodPost {
		var body map[string]any
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		rows, _ := body["rows"].([]any)
		diagnostics, err := s.validateRows(table, parts[4], anyRows(rows), stringValue(body["mode"], "active_only"))
		return 200, map[string]any{"diagnostics": diagnostics}, "", nil, err
	}
	if len(parts) == 7 && parts[3] == "generations" && parts[5] == "records" && parts[6] == "commit" && r.Method == http.MethodPost {
		var body map[string]any
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		payload, status, err := s.commitRows(table, parts[4], body)
		return status, payload, "", nil, err
	}
	return 404, nil, "", nil, appError{404, "API route not found"}
}

func (s *server) dispatchBinaryAPI(r *http.Request, parts []string) (int, any, string, []byte, error) {
	if len(parts) != 4 {
		return 404, nil, "", nil, appError{404, "API route not found"}
	}
	table := parts[2]
	key := parseKeyParam(parts[3])
	if _, err := s.loadSchema(table); err != nil {
		return 0, nil, "", nil, err
	}
	switch r.Method {
	case http.MethodGet:
		file, ext, err := s.findBinaryAsset(table, key)
		if err != nil {
			return 0, nil, "", nil, err
		}
		if file == "" {
			return 0, nil, "", nil, appError{404, "Binary asset not found."}
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return 0, nil, "", nil, err
		}
		contentType := mime.TypeByExtension("." + ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		return 200, nil, contentType, data, nil
	case http.MethodPost:
		return s.uploadBinaryAsset(r, table, key)
	case http.MethodDelete:
		file, _, err := s.findBinaryAsset(table, key)
		if err != nil {
			return 0, nil, "", nil, err
		}
		if file == "" {
			return 200, map[string]any{"deleted": false, "metadata": nil}, "", nil, nil
		}
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			return 0, nil, "", nil, err
		}
		return 200, map[string]any{"deleted": true, "metadata": nil}, "", nil, nil
	default:
		return 405, nil, "", nil, appError{405, "Method not allowed"}
	}
}

func (s *server) uploadBinaryAsset(r *http.Request, table string, key any) (int, any, string, []byte, error) {
	schema, err := s.loadSchema(table)
	if err != nil {
		return 0, nil, "", nil, err
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return 0, nil, "", nil, appError{400, "Multipart upload is required."}
	}
	generationID := r.FormValue("generationId")
	if generationID == "" {
		generationID = defaultGeneration
	}
	if err := s.requireGeneration(generationID); err != nil {
		return 0, nil, "", nil, err
	}
	exists, err := s.recordExists(table, generationID, key)
	if err != nil {
		return 0, nil, "", nil, err
	}
	if !exists {
		return 0, nil, "", nil, appError{404, "Record not found for binary upload."}
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return 0, nil, "", nil, appError{400, "Multipart field 'file' is required."}
	}
	defer file.Close()
	field, err := binaryUploadField(schema, r.FormValue("field"))
	if err != nil {
		return 0, nil, "", nil, err
	}
	extension := strings.ToLower(strings.TrimPrefix(filepath.Ext(header.Filename), "."))
	if !safeExtension(extension) {
		return 0, nil, "", nil, appError{400, "Uploaded file must have a safe extension."}
	}
	if allowed := allowedExtensions(field); len(allowed) > 0 && !contains(allowed, extension) {
		return 0, nil, "", nil, appError{422, "Unsupported file extension: " + extension}
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return 0, nil, "", nil, err
	}
	if max := maxBinarySize(field); max > 0 && int64(len(data)) > max {
		return 0, nil, "", nil, appError{413, fmt.Sprintf("Uploaded file exceeds %d bytes.", max)}
	}
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = mime.TypeByExtension("." + extension)
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if allowed := allowedMimeTypes(field); len(allowed) > 0 && !contains(allowed, strings.ToLower(mimeType)) {
		return 0, nil, "", nil, appError{422, "Unsupported MIME type: " + mimeType}
	}
	if existing, existingExt, err := s.findBinaryAsset(table, key); err != nil {
		return 0, nil, "", nil, err
	} else if existing != "" && existingExt != extension {
		_ = os.Remove(existing)
	}
	dest, err := s.binaryAssetPath(table, key, extension)
	if err != nil {
		return 0, nil, "", nil, err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return 0, nil, "", nil, err
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return 0, nil, "", nil, err
	}
	sum := sha256.Sum256(data)
	metadata := map[string]any{
		"extension":     extension,
		"mime_type":     mimeType,
		"size_bytes":    len(data),
		"sha256":        hex.EncodeToString(sum[:]),
		"original_name": header.Filename,
		"updated_at":    time.Now().UTC().Format(time.RFC3339Nano),
	}
	keyString := fmt.Sprint(key)
	if _, ok := key.(map[string]any); ok {
		keyString = stableStringify(key)
	}
	return 200, map[string]any{
		"saved":    true,
		"metadata": metadata,
		"asset": map[string]any{
			"table": table, "key": key, "field": field.SystemName,
			"url": "/api/binaries/" + pathEscape(table) + "/" + pathEscape(keyString),
		},
	}, "", nil, nil
}

func (s *server) dispatchSchemaAPI(r *http.Request, parts []string) (int, any, string, []byte, error) {
	if len(parts) == 2 && r.Method == http.MethodGet {
		schemas, err := s.loadSchemas()
		if err != nil {
			return 0, nil, "", nil, err
		}
		return 200, map[string]any{"schemas": schemas, "rows": schemaListRows(schemas)}, "", nil, nil
	}
	if len(parts) == 2 && r.Method == http.MethodPut {
		var body struct {
			Rows []map[string]any `json:"rows"`
		}
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		payload, err := s.saveSchemaList(body.Rows)
		return 200, payload, "", nil, err
	}
	if len(parts) == 3 && parts[2] == "delete" && r.Method == http.MethodPost {
		var body struct {
			TableIDs []string `json:"tableIds"`
		}
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		payload, err := s.deleteSchemas(body.TableIDs)
		return 200, payload, "", nil, err
	}
	if len(parts) == 3 && r.Method == http.MethodGet {
		schema, err := s.loadSchema(parts[2])
		if err != nil {
			return 0, nil, "", nil, err
		}
		schemas, err := s.loadSchemas()
		if err != nil {
			return 0, nil, "", nil, err
		}
		tables := make([]string, 0, len(schemas))
		for _, item := range schemas {
			tables = append(tables, item.TableID)
		}
		return 200, map[string]any{"schema": schema, "fieldRows": schemaFieldRows(schema), "tables": tables}, "", nil, nil
	}
	if len(parts) == 3 && r.Method == http.MethodPut {
		var body map[string]any
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		payload, status, err := s.saveSchemaDetail(parts[2], body)
		return status, payload, "", nil, err
	}
	return 404, nil, "", nil, appError{404, "API route not found"}
}

func (s *server) dispatchGenerationAPI(r *http.Request, parts []string) (int, any, string, []byte, error) {
	if len(parts) == 2 && r.Method == http.MethodGet {
		payload, err := s.loadGenerations()
		return 200, payload, "", nil, err
	}
	if len(parts) == 2 && r.Method == http.MethodPost {
		payload, err := s.createGeneration(nil)
		return 201, payload, "", nil, err
	}
	if len(parts) == 4 && parts[3] == "config" && r.Method == http.MethodPut {
		var body map[string]any
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		config, _ := body["config"].(map[string]any)
		if config == nil {
			config = body
		}
		payload, err := s.updateGeneration(parts[2], config)
		return 200, payload, "", nil, err
	}
	if len(parts) == 3 && parts[2] == "delete" && r.Method == http.MethodPost {
		var body struct {
			GenerationIDs      []string `json:"generationIds"`
			ActiveGenerationID string   `json:"activeGenerationId"`
		}
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		payload, err := s.deleteGenerations(body.GenerationIDs, body.ActiveGenerationID)
		return 200, payload, "", nil, err
	}
	if len(parts) == 3 && parts[2] == "duplicate" && r.Method == http.MethodPost {
		var body struct {
			SourceGenerationIDs []string `json:"sourceGenerationIds"`
			SourceGenerationID  string   `json:"sourceGenerationId"`
		}
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		ids := body.SourceGenerationIDs
		if len(ids) == 0 && body.SourceGenerationID != "" {
			ids = []string{body.SourceGenerationID}
		}
		payload, err := s.duplicateGenerations(ids)
		return 201, payload, "", nil, err
	}
	if len(parts) == 3 && parts[2] == "persistent-merge" && r.Method == http.MethodPost {
		var body struct {
			SourceGenerationIDs []string       `json:"sourceGenerationIds"`
			Destination         map[string]any `json:"destination"`
		}
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		payload, err := s.persistentMerge(body.SourceGenerationIDs, body.Destination)
		return 200, payload, "", nil, err
	}
	if len(parts) == 3 && parts[2] == "analyze" && r.Method == http.MethodPost {
		var body struct {
			GenerationIDs []string `json:"generationIds"`
		}
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		payload, err := s.analyzeGenerations(body.GenerationIDs)
		return 200, payload, "", nil, err
	}
	return 404, nil, "", nil, appError{404, "API route not found"}
}

func (s *server) dispatchExportAPI(r *http.Request, parts []string) (int, any, string, []byte, error) {
	if len(parts) == 3 && parts[2] == "check" && r.Method == http.MethodPost {
		var body struct {
			GenerationIDs []string       `json:"generationIds"`
			Format        string         `json:"format"`
			Options       map[string]any `json:"options"`
		}
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		payload, err := s.checkExport(body.GenerationIDs, body.Format, body.Options)
		return 200, payload, "", nil, err
	}
	if len(parts) == 2 && r.Method == http.MethodPost {
		var body struct {
			GenerationIDs []string       `json:"generationIds"`
			Format        string         `json:"format"`
			Options       map[string]any `json:"options"`
		}
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		payload, status, err := s.createExport(body.GenerationIDs, body.Format, body.Options)
		return status, payload, "", nil, err
	}
	if len(parts) == 4 && parts[3] == "download" && r.Method == http.MethodGet {
		artifact, ok := s.exports[parts[2]]
		if !ok {
			return 0, nil, "", nil, appError{404, "Export artifact not found or expired."}
		}
		return 200, nil, artifact.contentType, artifact.data, nil
	}
	return 404, nil, "", nil, appError{404, "API route not found"}
}

func (s *server) dispatchExportSettingsAPI(r *http.Request, parts []string) (int, any, string, []byte, error) {
	if len(parts) != 2 {
		return 404, nil, "", nil, appError{404, "API route not found"}
	}
	if r.Method == http.MethodGet {
		payload, err := s.exportSettingsPayload()
		return 200, payload, "", nil, err
	}
	if r.Method == http.MethodPut {
		var body exportSettings
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		payload, err := s.saveExportSettings(body)
		return 200, payload, "", nil, err
	}
	return 405, nil, "", nil, appError{405, "Method not allowed"}
}

func (s *server) dispatchExportDefinitionsAPI(r *http.Request, parts []string) (int, any, string, []byte, error) {
	if len(parts) != 2 {
		return 404, nil, "", nil, appError{404, "API route not found"}
	}
	if r.Method == http.MethodGet {
		payload, err := s.templateExportDefinitionsPayload()
		return 200, payload, "", nil, err
	}
	if r.Method == http.MethodPut {
		var body struct {
			Version     int                        `json:"version"`
			OutputRoot  string                     `json:"output_root"`
			Defaults    templateGenerateDefaults   `json:"defaults"`
			Definitions []templateExportDefinition `json:"definitions"`
			Rows        []templateDefinitionRow    `json:"rows"`
		}
		if err := readJSON(r, &body); err != nil {
			return 0, nil, "", nil, err
		}
		defs := templateExportDefinitions{Version: body.Version, OutputRoot: body.OutputRoot, Defaults: body.Defaults, Definitions: body.Definitions}
		if len(defs.Defaults.DefinitionIDs) == 0 {
			if current, err := s.loadTemplateExportDefinitions(); err == nil {
				defs.Defaults = current.Defaults
			}
		}
		if len(defs.Definitions) == 0 && len(body.Rows) > 0 {
			defs.Definitions = definitionsFromRows(body.Rows)
		}
		payload, err := s.saveTemplateExportDefinitions(defs)
		return 200, payload, "", nil, err
	}
	return 405, nil, "", nil, appError{405, "Method not allowed"}
}

func (s *server) dispatchGenerateAPI(r *http.Request, parts []string) (int, any, string, []byte, error) {
	if len(parts) != 3 || parts[2] != "check" {
		return 404, nil, "", nil, appError{404, "API route not found"}
	}
	if r.Method != http.MethodPost {
		return 405, nil, "", nil, appError{405, "Method not allowed"}
	}
	var body struct {
		GenerationIDs []string `json:"generationIds"`
		DefinitionIDs []string `json:"definitionIds"`
		OutputRoot    string   `json:"outputRoot"`
	}
	if err := readJSON(r, &body); err != nil {
		return 0, nil, "", nil, err
	}
	result, _, _, err := s.buildGenerateResult(body.GenerationIDs, body.DefinitionIDs, body.OutputRoot)
	return 200, result, "", nil, err
}

func (s *server) handleStatic(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name == "." || name == "" {
		name = "index.html"
	}
	if data, err := fs.ReadFile(s.static, name); err == nil {
		if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		if contentType := mime.TypeByExtension(filepath.Ext(name)); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		_, _ = w.Write(data)
		return
	}
	data, err := fs.ReadFile(s.static, "index.html")
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": "embedded index.html is missing"})
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func splitPath(value string) []string {
	clean := strings.Trim(path.Clean(value), "/")
	if clean == "" {
		return nil
	}
	raw := strings.Split(clean, "/")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		unescaped, err := url.PathUnescape(part)
		if err != nil {
			parts = append(parts, part)
			continue
		}
		parts = append(parts, unescaped)
	}
	return parts
}

func pathEscape(value string) string {
	return url.PathEscape(value)
}

func queryDefault(r *http.Request, name string, fallback string) string {
	if value := r.URL.Query().Get(name); value != "" {
		return value
	}
	return fallback
}

func readJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return appError{400, "Invalid JSON request body."}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *server) readYAML(file string, target any) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, target)
}

func (s *server) writeYAML(file string, value any) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o644)
}

func tableIDFromFile(file string) string {
	return strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
}

func (s *server) schemaFile(table string) string {
	return filepath.Join(s.schemaRoot, table+".yaml")
}

func (s *server) tableFile(table, generationID string) string {
	return filepath.Join(s.generationRoot, generationID, table+".yaml")
}

func stableStringify(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func parseKeyParam(value string) any {
	if strings.HasPrefix(value, "{") || strings.HasPrefix(value, "[") {
		var out any
		if err := json.Unmarshal([]byte(value), &out); err == nil {
			return out
		}
	}
	return value
}

func safePrimaryKeyStem(key any) string {
	if s, ok := key.(string); ok && safePathSegment(s) {
		return s
	}
	serialized := fmt.Sprint(key)
	if _, ok := key.(string); !ok {
		serialized = stableStringify(key)
	}
	return "key_" + base64URL([]byte(serialized))
}

func base64URL(data []byte) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	var out strings.Builder
	for i := 0; i < len(data); i += 3 {
		var n uint32
		remaining := len(data) - i
		n |= uint32(data[i]) << 16
		if remaining > 1 {
			n |= uint32(data[i+1]) << 8
		}
		if remaining > 2 {
			n |= uint32(data[i+2])
		}
		out.WriteByte(alphabet[(n>>18)&63])
		out.WriteByte(alphabet[(n>>12)&63])
		if remaining > 1 {
			out.WriteByte(alphabet[(n>>6)&63])
		}
		if remaining > 2 {
			out.WriteByte(alphabet[n&63])
		}
	}
	return out.String()
}

func safePathSegment(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' {
			continue
		}
		return false
	}
	return true
}

func safeExtension(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' {
			continue
		}
		return false
	}
	return true
}

func (s *server) binaryAssetPath(table string, key any, extension string) (string, error) {
	if !safePathSegment(table) {
		return "", appError{400, "Invalid table name."}
	}
	extension = strings.ToLower(extension)
	if !safeExtension(extension) {
		return "", appError{400, "Invalid extension."}
	}
	file := filepath.Join(s.binaryRoot, table, safePrimaryKeyStem(key)+"."+extension)
	if err := ensurePathInside(s.binaryRoot, filepath.Dir(file)); err != nil {
		return "", err
	}
	return file, nil
}

func (s *server) findBinaryAsset(table string, key any) (string, string, error) {
	dir := filepath.Join(s.binaryRoot, table)
	if err := ensurePathInside(s.binaryRoot, dir); err != nil {
		return "", "", err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	prefix := safePrimaryKeyStem(key) + "."
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		ext := strings.TrimPrefix(filepath.Ext(entry.Name()), ".")
		file := filepath.Join(dir, entry.Name())
		if err := ensurePathInside(s.binaryRoot, file); err != nil {
			return "", "", err
		}
		return file, ext, nil
	}
	return "", "", nil
}

func ensurePathInside(root, target string) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return appError{422, "Resolved path is outside the allowed root."}
	}
	return nil
}

func (s *server) generationPath(generationID string) string {
	return filepath.Join(s.generationRoot, generationID)
}

func (s *server) loadSchemas() ([]schema, error) {
	entries, err := os.ReadDir(s.schemaRoot)
	if err != nil {
		return nil, err
	}
	var schemas []schema
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		table := tableIDFromFile(entry.Name())
		item, err := s.loadSchema(table)
		if err != nil {
			return nil, err
		}
		schemas = append(schemas, item)
	}
	sort.Slice(schemas, func(i, j int) bool { return schemas[i].TableID < schemas[j].TableID })
	return schemas, nil
}

func (s *server) loadSchema(table string) (schema, error) {
	var item schema
	if err := s.readYAML(s.schemaFile(table), &item); err != nil {
		if os.IsNotExist(err) {
			return item, appError{404, "Schema not found: " + table}
		}
		return item, err
	}
	return normalizeSchema(table, item), nil
}

func normalizeSchema(table string, item schema) schema {
	if item.SystemName == "" {
		item.SystemName = table
	}
	if item.TableID == "" {
		item.TableID = item.SystemName
	}
	if item.BusinessName == "" {
		item.BusinessName = item.SystemName
	}
	for i := range item.Fields {
		if item.Fields[i].BusinessName == "" {
			item.Fields[i].BusinessName = item.Fields[i].SystemName
		}
		if item.Fields[i].Type == "" {
			item.Fields[i].Type = "string"
		}
	}
	return item
}

func schemaListRows(schemas []schema) []map[string]any {
	rows := make([]map[string]any, 0, len(schemas))
	for _, item := range schemas {
		refs := map[string]bool{}
		for _, field := range item.Fields {
			if table, ok := field.Reference["table"].(string); ok && table != "" {
				refs[table] = true
			}
		}
		var refList []string
		for ref := range refs {
			refList = append(refList, ref)
		}
		sort.Strings(refList)
		rows = append(rows, map[string]any{
			"selected": false, "table_id": item.TableID, "system_name": item.SystemName,
			"business_name": item.BusinessName, "export": item.Export, "primary_key": strings.Join(item.PrimaryKey, ", "),
			"references": strings.Join(refList, ", "), "comment": item.Comment,
		})
	}
	return rows
}

func schemaFieldRows(item schema) []map[string]any {
	rows := make([]map[string]any, 0, len(item.Fields))
	for _, f := range item.Fields {
		kind := "data"
		if contains(item.PrimaryKey, f.SystemName) {
			kind = "primary_key"
		} else if f.Formula != "" {
			kind = "formula"
		} else if _, ok := f.Reference["table"]; ok || f.Type == "external_reference" {
			kind = "reference"
		}
		export := true
		if f.Export != nil {
			export = *f.Export
		}
		rows = append(rows, map[string]any{
			"id": f.SystemName, "original_system_name": f.SystemName, "kind": kind,
			"system_name": f.SystemName, "business_name": f.BusinessName, "type": fieldTypeForRow(f),
			"formula": f.Formula, "reference_table": stringMapValue(f.Reference, "table"),
			"constants": strings.Join(f.Constants, ", "), "default_value": defaultString(f.DefaultValue),
			"export": export, "required": f.Required, "comment": f.Comment,
		})
	}
	return rows
}

func fieldTypeForRow(f field) string {
	if _, ok := f.Reference["table"]; ok {
		return "external_reference"
	}
	if f.Type == "" {
		return "string"
	}
	return f.Type
}

func (s *server) loadGenerationSettings() (generationSettings, error) {
	settings := generationSettings{OrderingMode: "numeric", NumericDigits: 4}
	file := filepath.Join(s.generationRoot, "_config.yaml")
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return settings, nil
	}
	var raw map[string]any
	if err := s.readYAML(file, &raw); err != nil {
		return settings, err
	}
	if raw["ordering_mode"] == "release_date" {
		settings.OrderingMode = "release_date"
	}
	if digits, ok := toInt(raw["numeric_digits"]); ok && digits > 0 {
		settings.NumericDigits = digits
	}
	return settings, nil
}

func (s *server) loadGenerations() (map[string]any, error) {
	settings, err := s.loadGenerationSettings()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.generationRoot)
	if err != nil {
		return nil, err
	}
	var generations []generation
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		var raw map[string]any
		id := entry.Name()
		if err := s.readYAML(filepath.Join(s.generationRoot, id, "_config.yaml"), &raw); err != nil {
			return nil, appError{422, "Generation config is missing or invalid: " + id}
		}
		config, err := normalizeGenerationConfig(raw, settings)
		if err != nil {
			return nil, err
		}
		config.ID = id
		config.FolderName = id
		config.DerivedFolderName = generationFolderName(config, settings)
		generations = append(generations, config)
	}
	sortGenerations(generations)
	if err := validateGenerationSet(generations); err != nil {
		return nil, err
	}
	return map[string]any{"settings": settings, "generations": generations}, nil
}

func normalizeGenerationConfig(raw map[string]any, settings generationSettings) (generation, error) {
	pathName := strings.TrimSpace(stringValue(raw["path_name"], ""))
	if !validPathName(pathName) {
		return generation{}, appError{422, "Generation path_name must start with an alphanumeric character and contain only letters, numbers, underscores, and hyphens."}
	}
	output := true
	if v, ok := raw["output"].(bool); ok {
		output = v
	}
	result := generation{Output: output, PathName: pathName, Description: stringValue(raw["description"], "")}
	if settings.OrderingMode == "release_date" {
		value := strings.TrimSpace(fmt.Sprint(raw["generation_index"]))
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return result, appError{422, "Generation generation_index must be a YYYY-MM-DD date in release_date mode."}
		}
		result.GenerationIndex = value
		return result, nil
	}
	index, ok := toInt(raw["generation_index"])
	if !ok || index < 0 {
		return result, appError{422, "Generation generation_index must be a non-negative integer in numeric mode."}
	}
	result.GenerationIndex = index
	return result, nil
}

func generationFolderName(gen generation, settings generationSettings) string {
	if settings.OrderingMode == "release_date" {
		return fmt.Sprintf("%s_%s", gen.GenerationIndex, gen.PathName)
	}
	return fmt.Sprintf("%0*d_%s", settings.NumericDigits, intValue(gen.GenerationIndex), gen.PathName)
}

func sortGenerations(generations []generation) {
	sort.Slice(generations, func(i, j int) bool {
		left := generationSortValue(generations[i])
		right := generationSortValue(generations[j])
		if left == right {
			return generations[i].ID < generations[j].ID
		}
		return left < right
	})
}

func generationSortValue(gen generation) int64 {
	if value, ok := toInt(gen.GenerationIndex); ok {
		return int64(value)
	}
	if date, err := time.Parse("2006-01-02", fmt.Sprint(gen.GenerationIndex)); err == nil {
		return date.Unix()
	}
	return 0
}

func validateGenerationSet(generations []generation) error {
	indexes := map[string]bool{}
	folders := map[string]bool{}
	for _, gen := range generations {
		index := fmt.Sprint(gen.GenerationIndex)
		if indexes[index] {
			return appError{422, "Generation index is duplicated: " + index}
		}
		indexes[index] = true
		if folders[gen.DerivedFolderName] {
			return appError{422, "Derived generation folder is duplicated: " + gen.DerivedFolderName}
		}
		folders[gen.DerivedFolderName] = true
	}
	return nil
}

func (s *server) requireGeneration(generationID string) error {
	if _, err := os.Stat(filepath.Join(s.generationRoot, generationID, "_config.yaml")); err != nil {
		return appError{422, "Generation config is missing or invalid: " + generationID}
	}
	return nil
}

func (s *server) loadRecords(table, generationID string) ([]record, error) {
	if err := s.requireGeneration(generationID); err != nil {
		return nil, err
	}
	file := s.tableFile(table, generationID)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return []record{}, nil
	}
	var raw map[string][]record
	if err := s.readYAML(file, &raw); err != nil {
		return nil, err
	}
	return raw[table], nil
}

func (s *server) loadRows(table, generationID string) (map[string]any, error) {
	schema, err := s.loadSchema(table)
	if err != nil {
		return nil, err
	}
	records, err := s.loadRecords(table, generationID)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(records))
	for _, record := range records {
		rows = append(rows, recordToRow(record, schema))
	}
	diagnostics, err := s.validateRows(table, generationID, rows, "active_only")
	return map[string]any{"schema": schema, "records": records, "rows": rows, "diagnostics": diagnostics}, err
}

func recordToRow(item record, schema schema) map[string]any {
	row := map[string]any{}
	if len(schema.PrimaryKey) == 1 {
		row[schema.PrimaryKey[0]] = item.Key
	} else if m, ok := item.Key.(map[string]any); ok {
		for key, value := range m {
			row[key] = value
		}
	}
	if item.Name != "" {
		row["name"] = item.Name
	}
	for key, value := range item.Data {
		row[key] = value
	}
	for _, f := range schema.Fields {
		if _, ok := row[f.SystemName]; !ok {
			row[f.SystemName] = materializedDefault(f)
		}
	}
	return row
}

func rowToRecord(row map[string]any, schema schema) record {
	data := map[string]any{}
	for _, f := range schema.Fields {
		if contains(schema.PrimaryKey, f.SystemName) || f.Formula != "" {
			continue
		}
		value, ok := row[f.SystemName]
		if ok && !isBlank(value) {
			data[f.SystemName] = normalizeReferenceValue(value)
		}
	}
	result := record{Key: keyFromRow(row, schema), Data: data}
	if name := stringValue(row["name"], ""); name != "" {
		result.Name = name
	}
	return result
}

func materializedDefault(f field) any {
	if f.DefaultValue != nil {
		return f.DefaultValue
	}
	switch f.Type {
	case "boolean":
		return false
	case "integer", "decimal":
		return 0
	default:
		return ""
	}
}

func keyFromRow(row map[string]any, schema schema) any {
	if len(schema.PrimaryKey) == 1 {
		return row[schema.PrimaryKey[0]]
	}
	key := map[string]any{}
	for _, field := range schema.PrimaryKey {
		key[field] = row[field]
	}
	return key
}

func normalizeComparable(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func binaryFields(item schema) []field {
	out := []field{}
	for _, f := range item.Fields {
		if f.Type == "binary_file" {
			out = append(out, f)
		}
	}
	return out
}

func binaryUploadField(item schema, fieldName string) (field, error) {
	fields := binaryFields(item)
	if fieldName == "" && len(fields) == 1 {
		return fields[0], nil
	}
	for _, f := range fields {
		if f.SystemName == fieldName {
			return f, nil
		}
	}
	return field{}, appError{422, "Binary upload field is not declared in schema."}
}

func allowedExtensions(f field) []string {
	return stringSlice(f.Binary["allowed_extensions"])
}

func allowedMimeTypes(f field) []string {
	return stringSlice(f.Binary["allowed_mime_types"])
}

func maxBinarySize(f field) int64 {
	value, ok := f.Binary["max_size_bytes"]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	default:
		return 0
	}
}

func stringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		if stringsValue, ok := value.([]string); ok {
			out := make([]string, 0, len(stringsValue))
			for _, item := range stringsValue {
				out = append(out, strings.ToLower(item))
			}
			return out
		}
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, strings.ToLower(fmt.Sprint(item)))
	}
	return out
}

func binaryMetadataMap(value any) map[string]any {
	value = normalizeReferenceValue(value)
	if out, ok := value.(map[string]any); ok {
		return out
	}
	if out, ok := value.(map[any]any); ok {
		next := map[string]any{}
		for k, v := range out {
			next[fmt.Sprint(k)] = v
		}
		return next
	}
	return nil
}

func (s *server) recordExists(table, generationID string, key any) (bool, error) {
	item, err := s.loadSchema(table)
	if err != nil {
		return false, err
	}
	records, err := s.loadRecords(table, generationID)
	if err != nil {
		return false, err
	}
	comparable := normalizeComparable(key)
	for _, record := range records {
		row := recordToRow(record, item)
		if normalizeComparable(keyFromRow(row, item)) == comparable {
			return true, nil
		}
	}
	return false, nil
}

func (s *server) validateRows(table, generationID string, rows []map[string]any, mode string) ([]map[string]any, error) {
	schema, err := s.loadSchema(table)
	if err != nil {
		return nil, err
	}
	diagnostics := []map[string]any{}
	seen := map[string]bool{}
	referenceCache := map[string]map[string]bool{}
	for index, row := range rows {
		key := normalizeComparable(keyFromRow(row, schema))
		if seen[key] {
			diagnostics = append(diagnostics, diagnostic("error", table, generationID, index, first(schema.PrimaryKey), "Primary key is duplicated."))
		}
		seen[key] = true
		for _, f := range schema.Fields {
			value := normalizeReferenceValue(row[f.SystemName])
			if f.Required && isBlank(value) {
				diagnostics = append(diagnostics, diagnostic("error", table, generationID, index, f.SystemName, f.BusinessName+" is required."))
				continue
			}
			if isBlank(value) {
				continue
			}
			switch f.Type {
			case "integer":
				if _, ok := toInt(value); !ok {
					diagnostics = append(diagnostics, diagnostic("error", table, generationID, index, f.SystemName, f.BusinessName+" must be an integer."))
				}
			case "decimal":
				if _, ok := toFloat(value); !ok {
					diagnostics = append(diagnostics, diagnostic("error", table, generationID, index, f.SystemName, f.BusinessName+" must be a number."))
				}
			case "boolean":
				if _, ok := value.(bool); !ok {
					diagnostics = append(diagnostics, diagnostic("error", table, generationID, index, f.SystemName, f.BusinessName+" must be true or false."))
				}
			case "constant":
				if len(f.Constants) > 0 && !contains(f.Constants, fmt.Sprint(value)) {
					diagnostics = append(diagnostics, diagnostic("error", table, generationID, index, f.SystemName, f.BusinessName+" must be one of: "+strings.Join(f.Constants, ", ")+"."))
				}
			case "binary_file":
				metadata := binaryMetadataMap(value)
				if metadata == nil {
					diagnostics = append(diagnostics, diagnostic("error", table, generationID, index, f.SystemName, f.BusinessName+" must be uploaded."))
					continue
				}
				extension := strings.ToLower(stringValue(metadata["extension"], ""))
				if extension == "" {
					diagnostics = append(diagnostics, diagnostic("error", table, generationID, index, f.SystemName, f.BusinessName+" metadata is missing extension."))
				} else if allowed := allowedExtensions(f); len(allowed) > 0 && !contains(allowed, extension) {
					diagnostics = append(diagnostics, diagnostic("error", table, generationID, index, f.SystemName, f.BusinessName+" must use one of: "+strings.Join(allowed, ", ")+"."))
				}
				if allowed := allowedMimeTypes(f); len(allowed) > 0 {
					mimeType := strings.ToLower(stringValue(metadata["mime_type"], ""))
					if mimeType != "" && !contains(allowed, mimeType) {
						diagnostics = append(diagnostics, diagnostic("error", table, generationID, index, f.SystemName, f.BusinessName+" MIME type is not allowed: "+mimeType+"."))
					}
				}
				if max := maxBinarySize(f); max > 0 {
					if size, ok := toInt(metadata["size_bytes"]); ok && int64(size) > max {
						diagnostics = append(diagnostics, diagnostic("error", table, generationID, index, f.SystemName, fmt.Sprintf("%s exceeds %d bytes.", f.BusinessName, max)))
					}
				}
				if extension != "" {
					file, err := s.binaryAssetPath(table, keyFromRow(row, schema), extension)
					if err != nil {
						return nil, err
					}
					if _, err := os.Stat(file); err != nil {
						if os.IsNotExist(err) {
							diagnostics = append(diagnostics, diagnostic("error", table, generationID, index, f.SystemName, f.BusinessName+" file is missing from masterdata/binaries."))
						} else {
							return nil, err
						}
					}
				}
			}
			if f.Type == "external_reference" {
				target := stringMapValue(f.Reference, "table")
				if target != "" {
					if _, ok := referenceCache[target]; !ok {
						keys, err := s.referenceKeys(target, generationID, mode)
						if err != nil {
							return nil, err
						}
						referenceCache[target] = keys
					}
					if !referenceCache[target][normalizeComparable(value)] {
						diagnostics = append(diagnostics, diagnostic("error", table, generationID, index, f.SystemName, fmt.Sprintf("%s references missing %s key: %v.", f.BusinessName, target, value)))
					}
				}
			}
		}
	}
	return diagnostics, nil
}

func diagnostic(severity, table, generationID string, rowIndex int, field, message string) map[string]any {
	return map[string]any{"severity": severity, "table": table, "generationId": generationID, "rowIndex": rowIndex, "field": field, "message": message}
}

func (s *server) commitRows(table, generationID string, body map[string]any) (map[string]any, int, error) {
	schema, err := s.loadSchema(table)
	if err != nil {
		return nil, 0, err
	}
	if err := s.requireGeneration(generationID); err != nil {
		return nil, 0, err
	}
	rows := anyRows(sliceAny(body["rows"]))
	cleanRows := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if sourceID := stringValue(row["sourceGenerationId"], ""); sourceID != "" && sourceID != generationID {
			continue
		}
		for _, key := range []string{"sourceGenerationId", "sourceGenerationLabel", "isActiveGeneration", "isReadOnly", "isOverridden", "overriddenByGenerationId", "status"} {
			delete(row, key)
		}
		cleanRows = append(cleanRows, row)
	}
	mode := stringValue(body["mode"], "active_only")
	diagnostics, err := s.validateRows(table, generationID, cleanRows, mode)
	if err != nil {
		return nil, 0, err
	}
	if hasErrorDiagnostics(diagnostics) && !boolValue(body["force"]) {
		return map[string]any{"saved": false, "diagnostics": diagnostics, "requiresForce": true}, 409, nil
	}
	records := make([]record, 0, len(cleanRows))
	for _, row := range cleanRows {
		records = append(records, rowToRecord(row, schema))
	}
	if err := s.writeYAML(s.tableFile(table, generationID), map[string]any{table: records}); err != nil {
		return nil, 0, err
	}
	return map[string]any{"saved": true, "diagnostics": diagnostics, "records": records, "rows": cleanRows}, 200, nil
}

func (s *server) loadGenerationView(table, activeGenerationID, mode string) (map[string]any, error) {
	schema, err := s.loadSchema(table)
	if err != nil {
		return nil, err
	}
	payload, err := s.loadGenerations()
	if err != nil {
		return nil, err
	}
	gens := payload["generations"].([]generation)
	visible, err := visibleGenerations(gens, activeGenerationID, mode)
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	for _, gen := range visible {
		records, err := s.loadRecords(table, gen.ID)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			row := recordToRow(record, schema)
			row["sourceGenerationId"] = gen.ID
			row["sourceGenerationLabel"] = generationDisplayName(gen)
			row["isActiveGeneration"] = gen.ID == activeGenerationID
			row["isReadOnly"] = gen.ID != activeGenerationID
			row["isOverridden"] = false
			row["overriddenByGenerationId"] = ""
			row["_keyComparable"] = normalizeComparable(keyFromRow(row, schema))
			rows = append(rows, row)
		}
	}
	winner := map[string]string{}
	for _, row := range rows {
		winner[stringValue(row["_keyComparable"], "")] = stringValue(row["sourceGenerationId"], "")
	}
	viewRows := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		key := stringValue(row["_keyComparable"], "")
		delete(row, "_keyComparable")
		if winner[key] != stringValue(row["sourceGenerationId"], "") {
			row["isOverridden"] = true
			row["overriddenByGenerationId"] = winner[key]
		}
		viewRows = append(viewRows, row)
	}
	activeRows := []map[string]any{}
	for _, row := range viewRows {
		if row["sourceGenerationId"] == activeGenerationID {
			activeRows = append(activeRows, row)
		}
	}
	diagnostics, err := s.validateRows(table, activeGenerationID, activeRows, mode)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(visible))
	for i, gen := range visible {
		ids[i] = gen.ID
	}
	return map[string]any{"schema": schema, "activeGenerationId": activeGenerationID, "mode": mode, "orderedGenerationIds": ids, "rows": viewRows, "diagnostics": diagnostics}, nil
}

func visibleGenerations(gens []generation, activeGenerationID, mode string) ([]generation, error) {
	var active *generation
	for i := range gens {
		if gens[i].ID == activeGenerationID {
			active = &gens[i]
			break
		}
	}
	if active == nil {
		return nil, appError{404, "Generation not found: " + activeGenerationID}
	}
	if mode == "active_only" || mode == "" {
		return []generation{*active}, nil
	}
	if mode != "include_previous" {
		return nil, appError{400, "Unknown generation view mode: " + mode}
	}
	activeSort := generationSortValue(*active)
	var visible []generation
	for _, gen := range gens {
		if gen.ID == active.ID || (gen.Output && generationSortValue(gen) <= activeSort) {
			visible = append(visible, gen)
		}
	}
	return visible, nil
}

func referenceGenerations(gens []generation, activeGenerationID, mode string) ([]generation, error) {
	visible, err := visibleGenerations(gens, activeGenerationID, mode)
	if err != nil {
		return nil, err
	}
	if mode == "active_only" || mode == "" {
		if len(visible) == 1 && visible[0].Output {
			return visible, nil
		}
		return []generation{}, nil
	}
	out := visible[:0]
	for _, gen := range visible {
		if gen.Output {
			out = append(out, gen)
		}
	}
	return out, nil
}

func (s *server) referenceCandidates(table, generationID, mode string) (map[string]any, error) {
	schema, err := s.loadSchema(table)
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	if mode != "" {
		payload, err := s.loadGenerations()
		if err != nil {
			return nil, err
		}
		gens, err := referenceGenerations(payload["generations"].([]generation), generationID, mode)
		if err != nil {
			return nil, err
		}
		byKey := map[string]map[string]any{}
		for _, gen := range gens {
			records, err := s.loadRecords(table, gen.ID)
			if err != nil {
				return nil, err
			}
			for _, record := range records {
				row := recordToRow(record, schema)
				row["sourceGenerationId"] = gen.ID
				row["sourceGenerationLabel"] = generationDisplayName(gen)
				byKey[normalizeComparable(keyFromRow(row, schema))] = row
			}
		}
		for _, row := range byKey {
			rows = append(rows, row)
		}
	} else {
		records, err := s.loadRecords(table, generationID)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			rows = append(rows, recordToRow(record, schema))
		}
	}
	candidates := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		candidates = append(candidates, map[string]any{
			"key": keyFromRow(row, schema), "label": rowLabel(row, schema),
			"sourceGenerationId": row["sourceGenerationId"], "sourceGenerationLabel": row["sourceGenerationLabel"],
			"overrodeGenerationIds": []string{},
		})
	}
	return map[string]any{"candidates": candidates}, nil
}

func (s *server) referenceKeys(table, generationID, mode string) (map[string]bool, error) {
	payload, err := s.referenceCandidates(table, generationID, mode)
	if err != nil {
		return nil, err
	}
	keys := map[string]bool{}
	if candidates, ok := payload["candidates"].([]map[string]any); ok {
		for _, item := range candidates {
			keys[normalizeComparable(item["key"])] = true
		}
		return keys, nil
	}
	for _, candidate := range sliceAny(payload["candidates"]) {
		if item, ok := candidate.(map[string]any); ok {
			keys[normalizeComparable(item["key"])] = true
		}
	}
	return keys, nil
}

func generationDisplayName(gen generation) string {
	if gen.Description != "" {
		return gen.ID + ": " + gen.Description
	}
	return gen.ID
}

func rowLabel(row map[string]any, schema schema) string {
	if name := stringValue(row["name"], ""); name != "" {
		return name
	}
	key := keyFromRow(row, schema)
	if text := stringValue(key, ""); text != "" {
		return text
	}
	return normalizeComparable(key)
}

func (s *server) createGeneration(config map[string]any) (map[string]any, error) {
	current, err := s.loadGenerations()
	if err != nil {
		return nil, err
	}
	settings := current["settings"].(generationSettings)
	gens := current["generations"].([]generation)
	if config == nil {
		config = nextGenerationConfig(gens, settings)
	}
	gen, err := normalizeGenerationConfig(config, settings)
	if err != nil {
		return nil, err
	}
	folder := generationFolderName(gen, settings)
	if err := os.Mkdir(filepath.Join(s.generationRoot, folder), 0o755); err != nil {
		if os.IsExist(err) {
			return nil, appError{409, "Generation folder already exists: " + folder}
		}
		return nil, err
	}
	if err := s.writeYAML(filepath.Join(s.generationRoot, folder, "_config.yaml"), gen); err != nil {
		return nil, err
	}
	return s.loadGenerations()
}

func nextGenerationConfig(gens []generation, settings generationSettings) map[string]any {
	used := map[string]bool{}
	for _, gen := range gens {
		used[gen.PathName] = true
	}
	name := "new_generation"
	for i := 2; used[name]; i++ {
		name = fmt.Sprintf("new_generation_%d", i)
	}
	if settings.OrderingMode == "release_date" {
		return map[string]any{"generation_index": time.Now().Format("2006-01-02"), "output": true, "path_name": name, "description": ""}
	}
	max := -10
	for _, gen := range gens {
		if value, ok := toInt(gen.GenerationIndex); ok && value > max {
			max = value
		}
	}
	return map[string]any{"generation_index": max + 10, "output": true, "path_name": name, "description": ""}
}

func (s *server) updateGeneration(generationID string, config map[string]any) (map[string]any, error) {
	current, err := s.loadGenerations()
	if err != nil {
		return nil, err
	}
	settings := current["settings"].(generationSettings)
	gen, err := normalizeGenerationConfig(config, settings)
	if err != nil {
		return nil, err
	}
	nextID := generationFolderName(gen, settings)
	oldPath := filepath.Join(s.generationRoot, generationID)
	nextPath := filepath.Join(s.generationRoot, nextID)
	if generationID != nextID {
		if _, err := os.Stat(nextPath); err == nil {
			return nil, appError{409, "Generation folder already exists: " + nextID}
		}
		if err := os.Rename(oldPath, nextPath); err != nil {
			return nil, err
		}
	}
	if err := s.writeYAML(filepath.Join(nextPath, "_config.yaml"), gen); err != nil {
		return nil, err
	}
	return s.loadGenerations()
}

func (s *server) deleteGenerations(ids []string, active string) (map[string]any, error) {
	if len(ids) == 0 {
		return nil, appError{400, "generationIds must contain at least 1 generation id(s)."}
	}
	current, err := s.loadGenerations()
	if err != nil {
		return nil, err
	}
	gens := current["generations"].([]generation)
	if len(ids) >= len(gens) {
		return nil, appError{409, "At least one generation must remain."}
	}
	exists := map[string]bool{}
	for _, gen := range gens {
		exists[gen.ID] = true
	}
	for _, id := range ids {
		if !exists[id] {
			return nil, appError{404, "Generation not found: " + id}
		}
		if err := os.RemoveAll(filepath.Join(s.generationRoot, id)); err != nil {
			return nil, err
		}
	}
	latest, err := s.loadGenerations()
	if err != nil {
		return nil, err
	}
	nextGens := latest["generations"].([]generation)
	resolved := active
	deleted := map[string]bool{}
	for _, id := range ids {
		deleted[id] = true
	}
	if resolved == "" || deleted[resolved] {
		resolved = nextGens[0].ID
	}
	latest["deletedGenerationIds"] = ids
	latest["remainingGenerationIds"] = generationIDs(nextGens)
	latest["resolvedActiveGenerationId"] = resolved
	latest["diagnostics"] = []any{}
	return latest, nil
}

func (s *server) duplicateGenerations(ids []string) (map[string]any, error) {
	if len(ids) == 0 {
		return nil, appError{400, "sourceGenerationIds must contain at least 1 generation id(s)."}
	}
	current, err := s.loadGenerations()
	if err != nil {
		return nil, err
	}
	settings := current["settings"].(generationSettings)
	gens := current["generations"].([]generation)
	byID := map[string]generation{}
	usedNames := map[string]bool{}
	maxIndex := -10
	for _, gen := range gens {
		byID[gen.ID] = gen
		usedNames[gen.PathName] = true
		if value, ok := toInt(gen.GenerationIndex); ok && value > maxIndex {
			maxIndex = value
		}
	}
	var created []string
	for _, id := range ids {
		source, ok := byID[id]
		if !ok {
			return nil, appError{404, "Generation not found: " + id}
		}
		maxIndex += 10
		pathName := source.PathName + "_copy"
		for i := 2; usedNames[pathName]; i++ {
			pathName = fmt.Sprintf("%s_copy%d", source.PathName, i)
		}
		usedNames[pathName] = true
		config := map[string]any{"generation_index": maxIndex, "output": source.Output, "path_name": pathName, "description": source.Description}
		gen, err := normalizeGenerationConfig(config, settings)
		if err != nil {
			return nil, err
		}
		folder := generationFolderName(gen, settings)
		if err := copyDir(filepath.Join(s.generationRoot, source.ID), filepath.Join(s.generationRoot, folder)); err != nil {
			return nil, err
		}
		if err := s.writeYAML(filepath.Join(s.generationRoot, folder, "_config.yaml"), gen); err != nil {
			return nil, err
		}
		created = append(created, folder)
	}
	latest, err := s.loadGenerations()
	if err != nil {
		return nil, err
	}
	latest["createdGenerationIds"] = created
	latest["generationIds"] = created
	if len(created) > 0 {
		latest["generationId"] = created[0]
	}
	latest["sourceGenerationIds"] = ids
	latest["diagnostics"] = []any{}
	return latest, nil
}

func (s *server) persistentMerge(ids []string, destination map[string]any) (map[string]any, error) {
	if len(ids) < 2 {
		return nil, appError{400, "sourceGenerationIds must contain at least 2 generation id(s)."}
	}
	current, err := s.loadGenerations()
	if err != nil {
		return nil, err
	}
	settings := current["settings"].(generationSettings)
	gen, err := normalizeGenerationConfig(destination, settings)
	if err != nil {
		return nil, err
	}
	folder := generationFolderName(gen, settings)
	dest := filepath.Join(s.generationRoot, folder)
	if err := os.Mkdir(dest, 0o755); err != nil {
		if os.IsExist(err) {
			return nil, appError{409, "Generation folder already exists: " + folder}
		}
		return nil, err
	}
	if err := s.writeYAML(filepath.Join(dest, "_config.yaml"), gen); err != nil {
		return nil, err
	}
	schemas, err := s.loadSchemas()
	if err != nil {
		return nil, err
	}
	for _, schema := range schemas {
		byKey := map[string]record{}
		order := []string{}
		for _, id := range ids {
			records, err := s.loadRecords(schema.TableID, id)
			if err != nil {
				return nil, err
			}
			for _, record := range records {
				key := normalizeComparable(record.Key)
				if _, ok := byKey[key]; !ok {
					order = append(order, key)
				}
				byKey[key] = record
			}
		}
		out := make([]record, 0, len(order))
		for _, key := range order {
			out = append(out, byKey[key])
		}
		if len(out) > 0 {
			if err := s.writeYAML(filepath.Join(dest, schema.TableID+".yaml"), map[string]any{schema.TableID: out}); err != nil {
				return nil, err
			}
		}
	}
	latest, err := s.loadGenerations()
	if err != nil {
		return nil, err
	}
	latest["generationId"] = folder
	latest["folderName"] = folder
	latest["sourceGenerationIds"] = ids
	latest["diagnostics"] = []any{}
	return latest, nil
}

func (s *server) analyzeGenerations(ids []string) (map[string]any, error) {
	if len(ids) == 0 {
		return nil, appError{400, "generationIds must contain at least 1 generation id(s)."}
	}
	schemas, err := s.loadSchemas()
	if err != nil {
		return nil, err
	}
	total := 0
	tables := map[string]any{}
	for _, schema := range schemas {
		counts := map[string]int{}
		tableTotal := 0
		for _, id := range ids {
			records, err := s.loadRecords(schema.TableID, id)
			if err != nil {
				return nil, err
			}
			counts[id] = len(records)
			tableTotal += len(records)
		}
		total += tableTotal
		tables[schema.TableID] = map[string]any{"recordCount": tableTotal, "generationRecordCounts": counts, "overriddenRecordCount": 0}
	}
	return map[string]any{
		"generationIds": ids, "orderedGenerationIds": ids,
		"summary": map[string]any{"generationCount": len(ids), "tableCount": len(schemas), "recordCount": total, "overriddenRecordCount": 0},
		"tables":  tables, "diagnostics": []any{},
	}, nil
}

func (s *server) saveSchemaList(rows []map[string]any) (map[string]any, error) {
	current, err := s.loadSchemas()
	if err != nil {
		return nil, err
	}
	byID := map[string]schema{}
	for _, item := range current {
		byID[item.TableID] = item
	}
	for _, row := range rows {
		systemName := strings.TrimSpace(stringValue(row["system_name"], ""))
		if systemName == "" {
			return nil, appError{422, "system_name is required."}
		}
		oldID := stringValue(row["table_id"], systemName)
		item, ok := byID[oldID]
		if !ok {
			item = normalizeSchema(systemName, schema{SystemName: systemName, BusinessName: stringValue(row["business_name"], systemName), PrimaryKey: []string{}, Export: boolValueDefault(row["export"], true), Fields: []field{}})
		}
		item.TableID = systemName
		item.SystemName = systemName
		item.BusinessName = stringValue(row["business_name"], item.BusinessName)
		item.Export = boolValueDefault(row["export"], true)
		item.Comment = stringValue(row["comment"], "")
		if oldID != systemName {
			if err := os.Rename(s.schemaFile(oldID), s.schemaFile(systemName)); err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			_ = s.renameTableDataFiles(oldID, systemName)
		}
		if err := s.writeYAML(s.schemaFile(systemName), schemaToYAML(item)); err != nil {
			return nil, err
		}
	}
	latest, err := s.loadSchemas()
	if err != nil {
		return nil, err
	}
	return map[string]any{"saved": true, "schemas": latest, "rows": schemaListRows(latest), "diagnostics": []any{}}, nil
}

func (s *server) saveSchemaDetail(table string, body map[string]any) (map[string]any, int, error) {
	current, err := s.loadSchema(table)
	if err != nil {
		return nil, 0, err
	}
	meta, _ := body["schema"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
	}
	rows := anyRows(sliceAny(body["fields"]))
	next := current
	next.SystemName = stringValue(meta["system_name"], current.SystemName)
	next.TableID = next.SystemName
	next.BusinessName = stringValue(meta["business_name"], current.BusinessName)
	next.Comment = stringValue(meta["comment"], current.Comment)
	next.Export = boolValueDefault(meta["export"], current.Export)
	next.PrimaryKey = nil
	next.Fields = nil
	for _, row := range rows {
		name := strings.TrimSpace(stringValue(row["system_name"], ""))
		if name == "" || stringValue(row["kind"], "") == "formula" {
			continue
		}
		kind := stringValue(row["kind"], "data")
		if kind == "primary_key" {
			next.PrimaryKey = append(next.PrimaryKey, name)
		}
		t := stringValue(row["type"], "string")
		f := field{SystemName: name, BusinessName: stringValue(row["business_name"], name), Type: t, Required: boolValue(row["required"]), Comment: stringValue(row["comment"], "")}
		exp := boolValueDefault(row["export"], true)
		f.Export = &exp
		if kind == "reference" {
			f.Type = "external_reference"
			f.Reference = map[string]any{"table": stringValue(row["reference_table"], "")}
		}
		next.Fields = append(next.Fields, f)
	}
	if len(next.PrimaryKey) == 0 {
		return map[string]any{"saved": false, "diagnostics": []map[string]any{{"severity": "error", "field": "primary_key", "message": "At least one primary key field is required."}}, "requiresForce": false}, 422, nil
	}
	if next.SystemName != table {
		_ = s.renameTableDataFiles(table, next.SystemName)
		_ = os.Remove(s.schemaFile(table))
	}
	if err := s.writeYAML(s.schemaFile(next.SystemName), schemaToYAML(next)); err != nil {
		return nil, 0, err
	}
	normalized := normalizeSchema(next.SystemName, next)
	return map[string]any{"saved": true, "schema": normalized, "fieldRows": schemaFieldRows(normalized), "changedFiles": []string{}, "diagnostics": []any{}}, 200, nil
}

func schemaToYAML(item schema) map[string]any {
	return map[string]any{
		"system_name": item.SystemName, "business_name": item.BusinessName, "primary_key": item.PrimaryKey,
		"export": item.Export, "dependent_tables": item.DependentTables, "comment": item.Comment, "fields": item.Fields,
	}
}

func (s *server) deleteSchemas(ids []string) (map[string]any, error) {
	if len(ids) == 0 {
		return nil, appError{400, "tableIds is required."}
	}
	var deleted []string
	for _, id := range ids {
		file := s.schemaFile(id)
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return nil, appError{404, "Schema not found: " + id}
		}
		if err := os.Remove(file); err != nil {
			return nil, err
		}
		deleted = append(deleted, file)
	}
	latest, err := s.loadSchemas()
	if err != nil {
		return nil, err
	}
	return map[string]any{"deletedTableIds": ids, "deletedPaths": deleted, "schemas": latest, "rows": schemaListRows(latest), "diagnostics": []any{}}, nil
}

func (s *server) renameTableDataFiles(oldTable, newTable string) error {
	entries, err := os.ReadDir(s.generationRoot)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		oldPath := filepath.Join(s.generationRoot, entry.Name(), oldTable+".yaml")
		newPath := filepath.Join(s.generationRoot, entry.Name(), newTable+".yaml")
		if _, err := os.Stat(oldPath); err == nil {
			if err := os.Rename(oldPath, newPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func generationIDs(gens []generation) []string {
	ids := make([]string, len(gens))
	for i, gen := range gens {
		ids[i] = gen.ID
	}
	return ids
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func hasErrorDiagnostics(items []map[string]any) bool {
	for _, item := range items {
		if item["severity"] == "error" {
			return true
		}
	}
	return false
}

func anyRows(items []any) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, row)
		}
	}
	return rows
}

func sliceAny(value any) []any {
	if value == nil {
		return nil
	}
	if items, ok := value.([]any); ok {
		return items
	}
	if rows, ok := value.([]map[string]any); ok {
		items := make([]any, 0, len(rows))
		for _, row := range rows {
			items = append(items, row)
		}
		return items
	}
	return nil
}

func stringValue(value any, fallback string) string {
	if value == nil {
		return fallback
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func stringMapValue(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	return stringValue(m[key], "")
}

func boolValue(value any) bool {
	if v, ok := value.(bool); ok {
		return v
	}
	return false
}

func boolValueDefault(value any, fallback bool) bool {
	if v, ok := value.(bool); ok {
		return v
	}
	return fallback
}

func intValue(value any) int {
	result, _ := toInt(value)
	return result
}

func toInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		if v == float64(int(v)) {
			return int(v), true
		}
	case string:
		i, err := strconv.Atoi(v)
		return i, err == nil
	}
	return 0, false
}

func toFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	}
	return 0, false
}

func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func first(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[0]
}

func isBlank(value any) bool {
	return value == nil || value == ""
}

func normalizeReferenceValue(value any) any {
	if m, ok := value.(map[string]any); ok {
		if v, ok := m["value"]; ok {
			return v
		}
	}
	return value
}

func defaultString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func validPathName(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			if i == 0 && (r == '_' || r == '-') {
				return false
			}
			continue
		}
		return false
	}
	return true
}
