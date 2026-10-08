// Package entites defines the records described by the SQL migrations.
// Database defaults are applied on INSERT, not when creating a Go struct.
package entites

import "time"

// DocumentStatus describes the document processing state.
type DocumentStatus string

const (
	DocumentStatusUploaded   DocumentStatus = "uploaded"
	DocumentStatusProcessing DocumentStatus = "processing"
	DocumentStatusCompleted  DocumentStatus = "completed"
	DocumentStatusFailed     DocumentStatus = "failed"
)

// Document represents a row in documents. ID contains a UUID string.
type Document struct {
	ID               string         `db:"id" json:"id"`
	OriginalFilename string         `db:"original_filename" json:"original_filename"`
	StorageKey       string         `db:"storage_key" json:"storage_key"`
	MIMEType         string         `db:"mime_type" json:"mime_type"`
	SizeBytes        int64          `db:"size_bytes" json:"size_bytes"`
	Status           DocumentStatus `db:"status" json:"status"`
	CreatedAt        time.Time      `db:"created_at" json:"created_at"`
}
