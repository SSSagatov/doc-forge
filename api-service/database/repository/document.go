package repository

import (
	"context"
	"errors"
	"fmt"

	"cloud-native-platform/api-service/entites"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDocumentNotFound = errors.New("document not found")

// DocumentRepository stores document metadata. The caller owns the pool.
type DocumentRepository struct {
	pool *pgxpool.Pool
}

func NewDocumentRepository(pool *pgxpool.Pool) *DocumentRepository {
	return &DocumentRepository{pool: pool}
}

// Create generates ID and CreatedAt in PostgreSQL and defaults an empty status
// to uploaded. The input is updated only after a successful INSERT.
func (r *DocumentRepository) Create(ctx context.Context, document *entites.Document) error {
	if document == nil {
		return errors.New("create document: document is nil")
	}
	status := document.Status
	if status == "" {
		status = entites.DocumentStatusUploaded
	}
	if !validDocumentStatus(status) {
		return fmt.Errorf("create document: invalid status %q", status)
	}
	if document.SizeBytes < 0 {
		return errors.New("create document: size must not be negative")
	}

	created, err := scanDocument(r.pool.QueryRow(ctx, `
		INSERT INTO documents (original_filename, storage_key, mime_type, size_bytes, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, original_filename, storage_key, mime_type, size_bytes, status, created_at`,
		document.OriginalFilename, document.StorageKey, document.MIMEType, document.SizeBytes, string(status)))
	if err != nil {
		return fmt.Errorf("create document: %w", err)
	}
	*document = *created
	return nil
}

func (r *DocumentRepository) GetByID(ctx context.Context, id string) (*entites.Document, error) {
	document, err := scanDocument(r.pool.QueryRow(ctx, `
		SELECT id::text, original_filename, storage_key, mime_type, size_bytes, status, created_at
		FROM documents WHERE id = $1::uuid`, id))
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}
	return document, nil
}

// List returns newest documents first, with ID as a stable tie-breaker.
// Limit must be between 1 and 100; offset must be non-negative.
func (r *DocumentRepository) List(ctx context.Context, limit, offset int) ([]entites.Document, error) {
	if limit < 1 || limit > 100 || offset < 0 {
		return nil, errors.New("list documents: limit must be 1..100 and offset must be non-negative")
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, original_filename, storage_key, mime_type, size_bytes, status, created_at
		FROM documents ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	documents := make([]entites.Document, 0)
	for rows.Next() {
		document, err := scanDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("list documents: %w", err)
		}
		documents = append(documents, *document)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	return documents, nil
}

func (r *DocumentRepository) UpdateStatus(ctx context.Context, id string, status entites.DocumentStatus) error {
	if !validDocumentStatus(status) {
		return fmt.Errorf("update document status: invalid status %q", status)
	}
	result, err := r.pool.Exec(ctx, `UPDATE documents SET status = $2 WHERE id = $1::uuid`, id, string(status))
	if err != nil {
		return fmt.Errorf("update document status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("update document status: %w", ErrDocumentNotFound)
	}
	return nil
}

// Delete removes metadata and cascades to related jobs and analysis_results.
// It does not delete the PDF from filesystem storage.
func (r *DocumentRepository) Delete(ctx context.Context, id string) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM documents WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete document: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("delete document: %w", ErrDocumentNotFound)
	}
	return nil
}

func scanDocument(row pgx.Row) (*entites.Document, error) {
	var document entites.Document
	var status string
	err := row.Scan(&document.ID, &document.OriginalFilename, &document.StorageKey,
		&document.MIMEType, &document.SizeBytes, &status, &document.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDocumentNotFound
	}
	if err != nil {
		return nil, err
	}
	document.Status = entites.DocumentStatus(status)
	return &document, nil
}

func validDocumentStatus(status entites.DocumentStatus) bool {
	switch status {
	case entites.DocumentStatusUploaded, entites.DocumentStatusProcessing,
		entites.DocumentStatusCompleted, entites.DocumentStatusFailed:
		return true
	default:
		return false
	}
}
