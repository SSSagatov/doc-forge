package controllers

import (
	"bytes"
	"cloud-native-platform/api-service/entites"
	"cloud-native-platform/api-service/usecases"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"testing"
)

type processorFunc func(context.Context, *entites.Document, []byte) (*usecases.PDFAnalysis, error)

func (f processorFunc) Execute(ctx context.Context, d *entites.Document, b []byte) (*usecases.PDFAnalysis, error) {
	return f(ctx, d, b)
}

func TestAnalysisUpload(t *testing.T) {
	for _, tc := range []struct {
		name          string
		status, files int
	}{{"success", 201, 1}, {"AI failure", 502, 1}, {"database failure", 500, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			handler, err := NewAnalysisHandler(dir, processorFunc(func(_ context.Context, d *entites.Document, b []byte) (*usecases.PDFAnalysis, error) {
				if string(b) != "%PDF-test" || d.SizeBytes != int64(len(b)) || d.OriginalFilename != "test.pdf" {
					t.Fatal("incorrect input")
				}
				if _, err := os.Stat(d.StorageKey); err != nil {
					t.Fatal(err)
				}
				if tc.status == 500 {
					return nil, errors.New("secret database detail")
				}
				out := &usecases.PDFAnalysis{Document: d, Job: &entites.Job{}}
				if tc.status == 502 {
					return out, usecases.ErrAnalysisFailed
				}
				summary := "summary"
				out.Result = &entites.AnalysisResult{Summary: &summary, Requirements: json.RawMessage(`[]`), Entities: json.RawMessage(`[]`), Risks: json.RawMessage(`[]`)}
				return out, nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("file", "test.pdf")
			if err != nil {
				t.Fatal(err)
			}
			_, _ = part.Write([]byte("%PDF-test"))
			_ = writer.Close()
			req := httptest.NewRequest("POST", "/doc/pdf/upload", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != tc.status {
				t.Fatalf("status %d: %s", response.Code, response.Body)
			}
			if bytes.Contains(response.Body.Bytes(), []byte("secret database")) {
				t.Fatal("error leaked")
			}
			files, err := os.ReadDir(dir)
			if err != nil || len(files) != tc.files {
				t.Fatal("incorrect file cleanup")
			}
			if tc.status == 201 && !bytes.Contains(response.Body.Bytes(), []byte(`"summary":"summary"`)) {
				t.Fatal("missing analysis")
			}
		})
	}
}
