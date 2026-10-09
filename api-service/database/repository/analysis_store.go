package repository

import (
	"context"
	"errors"
	"time"

	"cloud-native-platform/api-service/entites"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AnalysisStore commits related document/job/result changes atomically.
type AnalysisStore struct{ pool *pgxpool.Pool }

func NewAnalysisStore(pool *pgxpool.Pool) *AnalysisStore { return &AnalysisStore{pool: pool} }

func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}

func (s *AnalysisStore) Begin(ctx context.Context, document *entites.Document) (*entites.Job, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	created, err := scanDocument(tx.QueryRow(ctx, `INSERT INTO documents
	(original_filename, storage_key, mime_type, size_bytes, status) VALUES ($1,$2,$3,$4,'processing')
	RETURNING id::text, original_filename, storage_key, mime_type, size_bytes, status, created_at`,
		document.OriginalFilename, document.StorageKey, document.MIMEType, document.SizeBytes))
	if err != nil {
		return nil, err
	}
	job, err := scanJob(tx.QueryRow(ctx, `INSERT INTO jobs (document_id, attempts, started_at)
	VALUES ($1::uuid,1,now()) RETURNING id::text, document_id::text, attempts, last_error, started_at, finished_at, created_at`, created.ID))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	*document = *created
	return job, nil
}

func (s *AnalysisStore) Complete(ctx context.Context, job *entites.Job, result *entites.AnalysisResult) error {
	if err := validateAnalysisResult(result); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	// Lock the unfinished job before writing a result, preventing duplicate completion.
	var documentID string
	if err := tx.QueryRow(ctx, `SELECT document_id::text FROM jobs WHERE id=$1::uuid AND finished_at IS NULL FOR UPDATE`, job.ID).Scan(&documentID); err != nil {
		return err
	}
	created, err := scanAnalysisResult(tx.QueryRow(ctx, `INSERT INTO analysis_results
	(job_id, summary, requirements, entities, risks, raw_response, prompt_tokens, completion_tokens)
	VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8)
	RETURNING id::text, job_id::text, summary, requirements, entities, risks, raw_response, prompt_tokens, completion_tokens, created_at`,
		job.ID, result.Summary, result.Requirements, result.Entities, result.Risks, result.RawResponse, result.PromptTokens, result.CompletionTokens))
	if err != nil {
		return err
	}
	var finished time.Time
	if err := tx.QueryRow(ctx, `UPDATE jobs SET finished_at=now(), last_error=NULL WHERE id=$1::uuid RETURNING finished_at`, job.ID).Scan(&finished); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE documents SET status='completed' WHERE id=$1::uuid`, documentID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	*result = *created
	job.FinishedAt, job.LastError = &finished, nil
	return nil
}

func (s *AnalysisStore) Fail(ctx context.Context, job *entites.Job, message string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var finished time.Time
	var documentID string
	err = tx.QueryRow(ctx, `UPDATE jobs SET finished_at=now(), last_error=$2
	WHERE id=$1::uuid AND finished_at IS NULL RETURNING finished_at, document_id::text`, job.ID, message).Scan(&finished, &documentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("job already finished or missing")
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE documents SET status='failed' WHERE id=$1::uuid`, documentID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	job.FinishedAt, job.LastError = &finished, &message
	return nil
}
