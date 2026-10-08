package controllers

import (
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"cloud-native-platform/api-service/templates"
)

const maxPDFSize int64 = 10 << 20

// NewHandler serves the upload page and saves PDFs in uploadDir.
func NewHandler(uploadDir string) (http.Handler, error) {
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
	mux.HandleFunc("POST /doc/pdf/upload", uploadPDF(uploadDir))
	return mux, nil
}

func uploadPDF(uploadDir string) http.HandlerFunc {
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
		respond(w, http.StatusCreated, "PDF-файл успешно загружен.")
	}
}

func respond(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}
