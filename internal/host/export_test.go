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
