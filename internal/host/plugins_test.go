package host

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditorPluginCommitPreservesRows(t *testing.T) {
	root := testPluginWorkspace(t)
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]any{
		"activeGenerationId": "0000_initial",
		"mode":               "active_only",
		"entryPointId":       "enemy-record",
		"entry": map[string]any{
			"kind":  "record",
			"table": "enemy",
			"key":   "slime",
		},
		"changes": map[string]any{
			"tables": map[string]any{
				"enemy": map[string]any{
					"updates": []any{
						map[string]any{
							"previousKey": "slime",
							"record": map[string]any{
								"key":  "slime",
								"name": "Slime",
								"data": map[string]any{"name": "Slime", "hp": 30},
							},
						},
					},
				},
			},
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/editor-plugins/enemy-status-editor/changes/commit", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	s.routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	var payload struct {
		Tables map[string]struct {
			Rows []map[string]any `json:"rows"`
		} `json:"tables"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if got := len(payload.Tables["enemy"].Rows); got != 2 {
		t.Fatalf("committed row count=%d, want 2; body=%s", got, resp.Body.String())
	}
	content := readFile(t, filepath.Join(root, "masterdata", "generations", "0000_initial", "enemy.yaml"))
	if !strings.Contains(content, "key: slime") || !strings.Contains(content, "key: bat") || !strings.Contains(content, "hp: 30") {
		t.Fatalf("unexpected enemy.yaml:\n%s", content)
	}
}

func testPluginWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "masterdata", "schema"))
	mkdirAll(t, filepath.Join(root, "masterdata", "generations", "0000_initial"))
	mkdirAll(t, filepath.Join(root, "masterdata", "plugins", "enemy-status-editor"))
	writeFile(t, filepath.Join(root, "masterdata", "generations", "0000_initial", "_config.yaml"), "generation_index: 0\noutput: true\npath_name: initial\ndescription: Initial\n")
	writeFile(t, filepath.Join(root, "masterdata", "plugins", "enemy-status-editor", "index.html"), "<!doctype html>\n")
	writeFile(t, filepath.Join(root, "masterdata", "editor_plugins.yaml"), `plugins:
  - plugin_id: enemy-status-editor
    display_name: Enemy status editor
    entry_html: plugins/enemy-status-editor/index.html
    open_mode: record
    entry_points:
      - entry_id: enemy-record
        placement: record_action
        table: enemy
        open_mode: record
    target_tables:
      - role: primary
        table: enemy
        required: true
        write: true
    permissions:
      read_tables: [enemy]
      write_tables: [enemy]
`)
	writeFile(t, filepath.Join(root, "masterdata", "schema", "enemy.yaml"), `system_name: enemy
business_name: Enemies
primary_key: [enemy_id]
export: true
fields:
  - system_name: enemy_id
    business_name: Enemy ID
    type: string
    required: true
    export: true
  - system_name: name
    business_name: Name
    type: string
    required: true
    export: true
  - system_name: hp
    business_name: HP
    type: integer
    required: true
    export: true
`)
	writeFile(t, filepath.Join(root, "masterdata", "generations", "0000_initial", "enemy.yaml"), `enemy:
  - key: slime
    name: Slime
    data:
      name: Slime
      hp: 24
  - key: bat
    name: Cave Bat
    data:
      name: Cave Bat
      hp: 18
`)
	return root
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
