package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"cloud-native-platform/api-service/entites"
	"cloud-native-platform/api-service/templates"
	"cloud-native-platform/api-service/usecases"
)

const maxPDFSize int64 = 10 << 20

type PDFProcessor interface {
	Execute(context.Context, *entites.Document, []byte) (*usecases.PDFAnalysis, error)
}

// NewAnalysisHandler enables persistent Gemini analysis for uploads.
func NewAnalysisHandler(uploadDir string, processor PDFProcessor) (http.Handler, error) {
	if processor == nil {
		return nil, errors.New("PDF processor is required")
	}
	return newHandler(uploadDir, processor)
}

// NewHandler serves the upload page and saves PDFs in uploadDir.
func NewHandler(uploadDir string) (http.Handler, error) {
	return newHandler(uploadDir, nil)
}

func newHandler(uploadDir string, processor PDFProcessor) (http.Handler, error) {
	page, err := template.ParseFS(templates.Files, "index.html")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(uploadDir, 0700); err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page.Execute(w, nil)
	})
	mux.HandleFunc("POST /doc/pdf/upload", uploadPDF(uploadDir, processor))
	return mux, nil
}

func uploadPDF(uploadDir string, processor PDFProcessor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow multipart headers in addition to the PDF itself.
		r.Body = http.MaxBytesReader(w, r.Body, maxPDFSize+(1<<20))
		err := r.ParseMultipartForm(1 << 20)
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				respond(w, http.StatusRequestEntityTooLarge, "Файл слишком большой. Максимум — 10 МБ.")
			} else {
				respond(w, http.StatusBadRequest, "Не удалось прочитать файл. Отправьте PDF через форму.")
			}
			return
		}
		files := r.MultipartForm.File["file"]
		if len(files) != 1 || len(r.MultipartForm.File) != 1 {
			respond(w, http.StatusBadRequest, "Выберите один PDF-файл.")
			return
		}
		header := files[0]
		if header.Size > maxPDFSize {
			respond(w, http.StatusRequestEntityTooLarge, "Файл слишком большой. Максимум — 10 МБ.")
			return
		}
		file, err := header.Open()
		if err != nil {
			respond(w, http.StatusInternalServerError, "Не удалось прочитать загруженный файл.")
			return
		}
		defer file.Close()
		var signature [5]byte
		_, err = io.ReadFull(file, signature[:])
		if err != nil || string(signature[:]) != "%PDF-" || !strings.EqualFold(filepath.Ext(header.Filename), ".pdf") {
			respond(w, http.StatusUnsupportedMediaType, "Поддерживаются только PDF-файлы.")
			return
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			respond(w, http.StatusInternalServerError, "Не удалось прочитать загруженный файл.")
			return
		}
		// Never use a client-provided filename as a storage path.
		destination, err := os.CreateTemp(uploadDir, "document-*.pdf")
		if err != nil {
			respond(w, http.StatusInternalServerError, "Не удалось сохранить файл. Попробуйте позже.")
			return
		}
		_, copyErr := io.Copy(destination, file)
		closeErr := destination.Close()
		if copyErr != nil || closeErr != nil {
			_ = os.Remove(destination.Name())
			respond(w, http.StatusInternalServerError, "Не удалось сохранить файл. Попробуйте позже.")
			return
		}
		if processor != nil {
			pdf, err := os.ReadFile(destination.Name())
			if err != nil {
				_ = os.Remove(destination.Name())
				respond(w, http.StatusInternalServerError, "Не удалось прочитать сохранённый PDF.")
				return
			}
			document := &entites.Document{OriginalFilename: header.Filename, StorageKey: destination.Name(), MIMEType: "application/pdf", SizeBytes: int64(len(pdf))}
			result, err := processor.Execute(r.Context(), document, pdf)
			if err != nil {
				// Retain the file if a persistent job was created, including failed jobs.
				if result == nil {
					_ = os.Remove(destination.Name())
				}
				status := http.StatusInternalServerError
				if errors.Is(err, usecases.ErrAnalysisFailed) {
					status = http.StatusBadGateway
				}
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "Не удалось выполнить анализ. Проверьте БД, доступность и квоту Gemini.", "data": result})
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "PDF проанализирован, результат сохранён.", "data": result})
			return
		}
		respond(w, http.StatusCreated, "PDF-файл успешно загружен.")
	}
}

func respond(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}
