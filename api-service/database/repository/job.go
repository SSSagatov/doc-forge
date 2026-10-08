package repository

import (
	"context"
	"errors"
	"fmt"

	"cloud-native-platform/api-service/entites"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrJobNotFound = errors.New("job not found")

// JobRepository persists jobs. The caller owns the pool.
type JobRepository struct{ pool *pgxpool.Pool }

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{pool: pool}
}

// Create generates ID and CreatedAt in PostgreSQL.
// The input is updated only after the INSERT succeeds.
func (r *JobRepository) Create(ctx context.Context, value *entites.Job) error {
	if err := validateJob(value); err != nil {
		return fmt.Errorf("create job: %w", err)
	}
	created, err := scanJob(r.pool.QueryRow(ctx, `
		INSERT INTO jobs (document_id, attempts, last_error, started_at, finished_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, document_id::text, attempts, last_error, started_at, finished_at, created_at`,
		value.DocumentID, value.Attempts, value.LastError, value.StartedAt, value.FinishedAt,
	))
	if err != nil {
		return fmt.Errorf("create job: %w", err)
	}
	*value = *created
	return nil
}

func (r *JobRepository) GetByID(ctx context.Context, id string) (*entites.Job, error) {
	value, err := scanJob(r.pool.QueryRow(ctx,
		`SELECT id::text, document_id::text, attempts, last_error, started_at, finished_at, created_at 
		FROM jobs WHERE id = $1::uuid`,
		id,
	))
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return value, nil
}

// List returns newest records first. Limit is 1..100; offset is non-negative.
func (r *JobRepository) List(ctx context.Context, limit, offset int) ([]entites.Job, error) {
	return r.list(ctx, nil, limit, offset)
}

// ListByDocumentID returns a page of records belonging to the parent.
// An absent parent or a parent without records yields an empty slice.
func (r *JobRepository) ListByDocumentID(ctx context.Context, id string, limit, offset int) ([]entites.Job, error) {
	return r.list(ctx, &id, limit, offset)
}

func (r *JobRepository) list(ctx context.Context, parentID *string, limit, offset int) ([]entites.Job, error) {
	if limit < 1 || limit > 100 || offset < 0 {
		return nil, errors.New("list jobs: limit must be 1..100 and offset must be non-negative")
	}
	query := `
	SELECT id::text, document_id::text, attempts, last_error, started_at, finished_at, created_at 
	FROM jobs`
	args := []any{limit, offset}
	if parentID != nil {
		query += " WHERE document_id = $3::uuid"
		args = append(args, *parentID)
	}
	query += " ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2"
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()
	values := make([]entites.Job, 0)
	for rows.Next() {
		value, err := scanJob(rows)
		if err != nil {
			return nil, fmt.Errorf("list jobs: %w", err)
		}
		values = append(values, *value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	return values, nil
}

// Update replaces all mutable fields, including nullable fields.
// ID and CreatedAt are preserved; nil fields are written as SQL NULL.
func (r *JobRepository) Update(ctx context.Context, value *entites.Job) error {
	if err := validateJob(value); err != nil {
		return fmt.Errorf("update job: %w", err)
	}
	updated, err := scanJob(r.pool.QueryRow(ctx, `
		UPDATE jobs SET document_id = $2, attempts = $3, last_error = $4, started_at = $5, finished_at = $6
		WHERE id = $1::uuid RETURNING id::text, document_id::text, attempts, last_error, started_at, finished_at, created_at`,
		value.ID, value.DocumentID, value.Attempts, value.LastError, value.StartedAt, value.FinishedAt,
	))
	if err != nil {
		return fmt.Errorf("update job: %w", err)
	}
	*value = *updated
	return nil
}

// Delete removes the record. Related analysis results are deleted by ON DELETE CASCADE.
func (r *JobRepository) Delete(ctx context.Context, id string) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM jobs WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete job: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("delete job: %w", ErrJobNotFound)
	}
	return nil
}

func scanJob(row pgx.Row) (*entites.Job, error) {
	var value entites.Job
	err := row.Scan(&value.ID, &value.DocumentID, &value.Attempts, &value.LastError, &value.StartedAt, &value.FinishedAt, &value.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrJobNotFound
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func validateJob(value *entites.Job) error {
	if value == nil {
		return errors.New("job is nil")
	}
	if value.Attempts < 0 {
		return errors.New("attempts must not be negative")
	}
	return nil
}
