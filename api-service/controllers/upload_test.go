package controllers

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUploadPDF(t *testing.T) {
	for _, tc := range []struct {
		name, filename, content string
		want                    int
	}{
		{"pdf", "document.pdf", "%PDF-1.4\n%%EOF\n", http.StatusCreated},
		{"uppercase", "document.PDF", "%PDF-1.4\n%%EOF\n", http.StatusCreated},
		{"disguised text", "document.pdf", "not a PDF", http.StatusUnsupportedMediaType},
		{"wrong extension", "document.txt", "%PDF-1.4\n", http.StatusUnsupportedMediaType},
		{"empty", "document.pdf", "", http.StatusUnsupportedMediaType},
		{"oversized file", "document.pdf", "%PDF-" + strings.Repeat("x", int(maxPDFSize)), http.StatusRequestEntityTooLarge},
		{"oversized request", "document.pdf", "%PDF-" + strings.Repeat("x", int(maxPDFSize)+(2<<20)), http.StatusRequestEntityTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			handler, err := NewHandler(dir)
			if err != nil {
				t.Fatal(err)
			}
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("file", tc.filename)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.WriteString(part, tc.content); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/doc/pdf/upload", &body)
			request.Header.Set("Content-Type", writer.FormDataContentType())
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.want {
				t.Fatalf("status = %d, want %d: %s", response.Code, tc.want, response.Body)
			}
			files, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if tc.want != http.StatusCreated {
				if len(files) != 0 {
					t.Fatal("rejected upload was saved")
				}
				return
			}
			if len(files) != 1 {
				t.Fatalf("saved %d files, want 1", len(files))
			}
			if files[0].Name() == tc.filename {
				t.Fatal("client filename used as storage path")
			}
			content, err := os.ReadFile(filepath.Join(dir, files[0].Name()))
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != tc.content {
				t.Fatal("saved content differs from uploaded content")
			}
		})
	}
}

func TestRoutes(t *testing.T) {
	handler, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{"GET", "/", http.StatusOK},
		{"GET", "/missing", http.StatusNotFound},
		{"GET", "/doc/pdf/upload", http.StatusMethodNotAllowed},
		{"POST", "/doc/pdf/upload", http.StatusBadRequest},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
		if response.Code != tc.want {
			t.Fatalf("%s %s: status %d, want %d", tc.method, tc.path, response.Code, tc.want)
		}
		if tc.path == "/" && !strings.Contains(response.Body.String(), "Загрузить PDF файл") {
			t.Fatal("upload button missing")
		}
	}
}
