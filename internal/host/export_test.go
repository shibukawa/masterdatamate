package host

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportValidationFindsMissingExternalReference(t *testing.T) {
	root := testExportWorkspace(t, "missing")
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := s.buildExportDataset([]string{"0000_initial"}, "csv_zip", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if dataset.Exportable {
		t.Fatalf("expected export to be blocked")
	}
	if len(dataset.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(dataset.Diagnostics))
	}
	if got := dataset.Diagnostics[0]["field"]; got != "org_id" {
		t.Fatalf("expected org_id diagnostic, got %v", got)
	}
}

func TestExcelCSVUsesBOMAndUppercaseBooleans(t *testing.T) {
	root := testExportWorkspace(t, "valid")
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := s.buildExportDataset([]string{"0000_initial"}, "excel_csv_zip", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	files, err := buildMultiFiles(dataset)
	if err != nil {
		t.Fatal(err)
	}
	var org []byte
	for _, file := range files {
		if file.Name == "org.csv" {
			org = file.Data
		}
	}
	if !bytes.HasPrefix(org, []byte{0xef, 0xbb, 0xbf}) {
		t.Fatalf("expected UTF-8 BOM in Excel CSV")
	}
	if !strings.Contains(string(org), "TRUE") {
		t.Fatalf("expected TRUE boolean in Excel CSV: %q", string(org))
	}
}

func TestCLIOmittedTimeFormatReadsExportSettings(t *testing.T) {
	root := testExportWorkspace(t, "valid")
	writeExportTestFile(t, filepath.Join(root, "masterdata", "export_settings.yaml"), `version: 1
formats:
  csv:
    time_format: epoch-ms
`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunExportCommand([]string{"--workspace", root, "--format", "csv", "--check-only", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s", code, stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	options := payload["options"].(map[string]any)
	if got := options["TimeFormat"]; got != "epoch-ms" {
		t.Fatalf("expected settings time format, got %v", got)
	}
}

func TestSaveExportSettingsNormalizesHTTPFormat(t *testing.T) {
	root := testExportWorkspace(t, "valid")
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.saveExportSettings(exportSettings{
		Version: 1,
		Formats: map[string]exportFormatOptions{
			"excel_csv_zip": {TimeFormat: "iso-local", Timezone: "Asia/Tokyo"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	settings, err := s.loadExportSettings()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := settings.Formats["excel_csv_zip"]; ok {
		t.Fatalf("expected HTTP format alias to be normalized")
	}
	if got := settings.Formats["excel-csv"].Timezone; got != "Asia/Tokyo" {
		t.Fatalf("expected saved timezone, got %q", got)
	}
}

func TestGenerateRendersSelectedDefinitions(t *testing.T) {
	root := testExportWorkspace(t, "valid")
	writeExportTestFile(t, filepath.Join(root, "masterdata", "generate_definitions.yaml"), `version: 1
output_root: generated
definitions:
  - id: user_constants
    name: User constants
    enabled: true
    scope: table
    table: user
    template: |
      package generated

      {% for row in records %}
      const {{ row.user_id|go_ident }}Org = {{ row.org_id|go_string }}
      {% endfor %}
    output_path: users/constants.go
    formatter: gofmt
  - id: user_record
    name: User record
    enabled: true
    scope: record
    table: user
    template: "{{ record.user_id }} -> {{ record.org_id }}\n"
    output_path: users/{{ record.user_id }}.txt
`)
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	result, files, _, err := s.buildGenerateResult([]string{"0000_initial"}, []string{"user_constants", "user_record"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if result["generatable"] != true {
		t.Fatalf("expected generatable diagnostics=%v", result["diagnostics"])
	}
	byName := map[string]string{}
	for _, file := range files {
		byName[file.Name] = string(file.Data)
	}
	if got := byName["users/constants.go"]; !strings.Contains(got, `const User1Org = "org-1"`) {
		t.Fatalf("expected generated Go constant, got:\n%s", got)
	}
	if got := byName["users/user-1.txt"]; got != "user-1 -> org-1\n" {
		t.Fatalf("expected record file, got %q", got)
	}
}

func TestCLIGenerateWritesConfiguredOutputRoot(t *testing.T) {
	root := testExportWorkspace(t, "valid")
	writeExportTestFile(t, filepath.Join(root, "masterdata", "generate_templates", "users.txt.pongo2"), `{% for row in records %}{{ row.user_id }}={{ row.org_id }}
{% endfor %}`)
	writeExportTestFile(t, filepath.Join(root, "masterdata", "generate_definitions.yaml"), `version: 1
output_root: generated
definitions:
  - id: users_txt
    name: Users text
    enabled: true
    scope: table
    table: user
    template_file: users.txt.pongo2
    output_path: users.txt
`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunGenerateCommand([]string{"--workspace", root, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	data, err := os.ReadFile(filepath.Join(root, "generated", "users.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "user-1=org-1\n" {
		t.Fatalf("expected template output, got %q", got)
	}
}

func testExportWorkspace(t *testing.T, mode string) string {
	t.Helper()
	root := t.TempDir()
	writeExportTestFile(t, filepath.Join(root, "masterdata", "generations", "_config.yaml"), "ordering_mode: numeric\nnumeric_digits: 4\n")
	writeExportTestFile(t, filepath.Join(root, "masterdata", "generations", "0000_initial", "_config.yaml"), "generation_index: 0\noutput: true\npath_name: initial\n")
	writeExportTestFile(t, filepath.Join(root, "masterdata", "schema", "org.yaml"), `system_name: org
business_name: Organizations
primary_key: [org_id]
export: true
fields:
  - system_name: org_id
    business_name: Org ID
    type: string
    required: true
    export: true
  - system_name: active
    business_name: Active
    type: boolean
    required: true
    export: true
`)
	writeExportTestFile(t, filepath.Join(root, "masterdata", "schema", "user.yaml"), `system_name: user
business_name: Users
primary_key: [user_id]
export: true
fields:
  - system_name: user_id
    business_name: User ID
    type: string
    required: true
    export: true
  - system_name: org_id
    business_name: Organization
    type: external_reference
    required: true
    export: true
    reference:
      table: org
`)
	writeExportTestFile(t, filepath.Join(root, "masterdata", "generations", "0000_initial", "org.yaml"), `org:
  - key: org-1
    data:
      active: true
`)
	orgID := "org-1"
	if mode == "missing" {
		orgID = "missing-org"
	}
	writeExportTestFile(t, filepath.Join(root, "masterdata", "generations", "0000_initial", "user.yaml"), `user:
  - key: user-1
    data:
      org_id: `+orgID+`
`)
	return root
}

func writeExportTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
