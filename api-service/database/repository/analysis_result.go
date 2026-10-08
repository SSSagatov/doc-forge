package repository

import (
	"cloud-native-platform/api-service/entites"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAnalysisResultNotFound = errors.New("analysis_result not found")

// AnalysisResultRepository persists analysis_results. The caller owns the pool.
type AnalysisResultRepository struct{ pool *pgxpool.Pool }

func NewAnalysisResultRepository(pool *pgxpool.Pool) *AnalysisResultRepository {
	return &AnalysisResultRepository{pool: pool}
}

// Create generates ID and CreatedAt in PostgreSQL.
// The input is updated only after the INSERT succeeds.
func (r *AnalysisResultRepository) Create(ctx context.Context, value *entites.AnalysisResult) error {
	if err := validateAnalysisResult(value); err != nil {
		return fmt.Errorf("create analysis_result: %w", err)
	}
	created, err := scanAnalysisResult(r.pool.QueryRow(ctx, `
	INSERT INTO analysis_results (job_id, summary, requirements, entities, risks, raw_response, prompt_tokens, completion_tokens)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING id::text, job_id::text, summary, requirements, entities, risks, raw_response, prompt_tokens, completion_tokens, created_at`,
		value.JobID, value.Summary, value.Requirements, value.Entities, value.Risks, value.RawResponse, value.PromptTokens, value.CompletionTokens,
	))
	if err != nil {
		return fmt.Errorf("create analysis_result: %w", err)
	}
	*value = *created
	return nil
}

func (r *AnalysisResultRepository) GetByID(ctx context.Context, id string) (*entites.AnalysisResult, error) {
	value, err := scanAnalysisResult(r.pool.QueryRow(ctx, `
	SELECT id::text, job_id::text, summary, requirements, entities, risks, raw_response, prompt_tokens, completion_tokens, created_at 
	FROM analysis_results WHERE id = $1::uuid`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("get analysis_result: %w", err)
	}
	return value, nil
}

// List returns newest records first. Limit is 1..100; offset is non-negative.
func (r *AnalysisResultRepository) List(ctx context.Context, limit, offset int) ([]entites.AnalysisResult, error) {
	return r.list(ctx, nil, limit, offset)
}

// ListByJobID returns a page of records belonging to the parent.
// An absent parent or a parent without records yields an empty slice.
func (r *AnalysisResultRepository) ListByJobID(ctx context.Context, id string, limit, offset int) ([]entites.AnalysisResult, error) {
	return r.list(ctx, &id, limit, offset)
}

func (r *AnalysisResultRepository) list(ctx context.Context, parentID *string, limit, offset int) ([]entites.AnalysisResult, error) {
	if limit < 1 || limit > 100 || offset < 0 {
		return nil, errors.New("list analysis_results: limit must be 1..100 and offset must be non-negative")
	}
	query := `
	SELECT id::text, job_id::text, summary, requirements, entities, risks, raw_response, prompt_tokens, completion_tokens, created_at 
	FROM analysis_results`
	args := []any{limit, offset}
	if parentID != nil {
		query += " WHERE job_id = $3::uuid"
		args = append(args, *parentID)
	}
	query += " ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2"
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list analysis_results: %w", err)
	}
	defer rows.Close()
	values := make([]entites.AnalysisResult, 0)
	for rows.Next() {
		value, err := scanAnalysisResult(rows)
		if err != nil {
			return nil, fmt.Errorf("list analysis_results: %w", err)
		}
		values = append(values, *value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list analysis_results: %w", err)
	}
	return values, nil
}

// Update replaces all mutable fields, including nullable fields.
// ID and CreatedAt are preserved; nil fields are written as SQL NULL.
func (r *AnalysisResultRepository) Update(ctx context.Context, value *entites.AnalysisResult) error {
	if err := validateAnalysisResult(value); err != nil {
		return fmt.Errorf("update analysis_result: %w", err)
	}
	updated, err := scanAnalysisResult(r.pool.QueryRow(ctx, `
	UPDATE analysis_results 
	SET job_id = $2, summary = $3, requirements = $4, entities = $5, risks = $6, raw_response = $7, prompt_tokens = $8, completion_tokens = $9
	WHERE id = $1::uuid 
	RETURNING id::text, job_id::text, summary, requirements, entities, risks, raw_response, prompt_tokens, completion_tokens, created_at`,
		value.ID, value.JobID, value.Summary, value.Requirements, value.Entities, value.Risks, value.RawResponse, value.PromptTokens, value.CompletionTokens,
	))
	if err != nil {
		return fmt.Errorf("update analysis_result: %w", err)
	}
	*value = *updated
	return nil
}

// Delete removes the record.
func (r *AnalysisResultRepository) Delete(ctx context.Context, id string) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM analysis_results WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete analysis_result: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("delete analysis_result: %w", ErrAnalysisResultNotFound)
	}
	return nil
}

func scanAnalysisResult(row pgx.Row) (*entites.AnalysisResult, error) {
	var value entites.AnalysisResult
	err := row.Scan(
		&value.ID, &value.JobID, &value.Summary, &value.Requirements, &value.Entities, &value.Risks,
		&value.RawResponse, &value.PromptTokens, &value.CompletionTokens, &value.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAnalysisResultNotFound
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func validateAnalysisResult(value *entites.AnalysisResult) error {
	if value == nil {
		return errors.New("analysis_result is nil")
	}
	if value.PromptTokens != nil && *value.PromptTokens < 0 {
		return errors.New("prompt tokens must not be negative")
	}
	if value.CompletionTokens != nil && *value.CompletionTokens < 0 {
		return errors.New("completion tokens must not be negative")
	}
	for _, field := range []struct {
		name string
		data json.RawMessage
	}{
		{"requirements", value.Requirements}, {"entities", value.Entities}, {"risks", value.Risks}, {"raw_response", value.RawResponse},
	} {
		if field.data != nil && !json.Valid(field.data) {
			return fmt.Errorf("%s must contain valid JSON or be nil", field.name)
		}
	}
	return nil
}
