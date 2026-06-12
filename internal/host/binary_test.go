package host

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBinaryAssetUploadDownloadAndDelete(t *testing.T) {
	root := testBinaryWorkspace(t)
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	handler := s.routes()

	upload := multipartRequest(t, "/api/binaries/product/prod-core", "asset.txt", "text/plain", []byte("asset bytes\n"))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, upload)
	if resp.Code != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", resp.Code, resp.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "masterdata", "binaries", "product", "prod-core.txt")); err != nil {
		t.Fatalf("expected uploaded file: %v", err)
	}

	download := httptest.NewRequest(http.MethodGet, "/api/binaries/product/prod-core", nil)
	resp = httptest.NewRecorder()
	handler.ServeHTTP(resp, download)
	if resp.Code != http.StatusOK {
		t.Fatalf("download status=%d body=%s", resp.Code, resp.Body.String())
	}
	if got := resp.Body.String(); got != "asset bytes\n" {
		t.Fatalf("download body=%q", got)
	}

	del := httptest.NewRequest(http.MethodDelete, "/api/binaries/product/prod-core", nil)
	resp = httptest.NewRecorder()
	handler.ServeHTTP(resp, del)
	if resp.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", resp.Code, resp.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "masterdata", "binaries", "product", "prod-core.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected file deletion, stat err=%v", err)
	}
}

func TestBinaryAssetUploadRejectsUnknownRecord(t *testing.T) {
	root := testBinaryWorkspace(t)
	s, err := NewData(root)
	if err != nil {
		t.Fatal(err)
	}
	resp := httptest.NewRecorder()
	s.routes().ServeHTTP(resp, multipartRequest(t, "/api/binaries/product/missing", "asset.txt", "text/plain", []byte("asset bytes\n")))
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "Record not found") {
		t.Fatalf("expected missing record error, got %s", resp.Body.String())
	}
}

func multipartRequest(t *testing.T, url, filename, contentType string, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("field", "image"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("generationId", "0000_initial"); err != nil {
		t.Fatal(err)
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, url, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func testBinaryWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "masterdata", "schema"))
	mkdirAll(t, filepath.Join(root, "masterdata", "generations", "0000_initial"))
	writeFile(t, filepath.Join(root, "masterdata", "generations", "0000_initial", "_config.yaml"), "generation_index: 0\noutput: true\npath_name: initial\ndescription: Initial\n")
	writeFile(t, filepath.Join(root, "masterdata", "schema", "product.yaml"), `system_name: product
business_name: Products
primary_key: [product_id]
export: true
fields:
  - system_name: product_id
    business_name: Product ID
    type: string
    required: true
    export: true
  - system_name: image
    business_name: Image
    type: binary_file
    required: false
    export: false
    binary:
      allowed_extensions: [txt]
      allowed_mime_types: [text/plain]
      max_size_bytes: 1024
`)
	writeFile(t, filepath.Join(root, "masterdata", "generations", "0000_initial", "product.yaml"), `product:
  - key: prod-core
    data: {}
`)
	return root
}
