package usecases

import (
	"context"
	"errors"
	"time"

	"cloud-native-platform/api-service/entites"
)

var ErrAnalysisFailed = errors.New("PDF analysis failed")

type PDFAnalyzer interface {
	Analyze(context.Context, []byte) (*entites.AnalysisResult, error)
}

// AnalysisStore makes each database state transition atomic.
type AnalysisStore interface {
	Begin(context.Context, *entites.Document) (*entites.Job, error)
	Complete(context.Context, *entites.Job, *entites.AnalysisResult) error
	Fail(context.Context, *entites.Job, string) error
}

type PDFAnalysis struct {
	Document *entites.Document       `json:"document"`
	Job      *entites.Job            `json:"job"`
	Result   *entites.AnalysisResult `json:"analysis"`
}

type AnalyzePDFUseCase struct {
	store    AnalysisStore
	analyzer PDFAnalyzer
}

func NewAnalyzePDFUseCase(store AnalysisStore, analyzer PDFAnalyzer) *AnalyzePDFUseCase {
	return &AnalyzePDFUseCase{store: store, analyzer: analyzer}
}

// Execute performs one bounded synchronous analysis. The stored file is retained
// if a job exists, including when analysis fails.
func (u *AnalyzePDFUseCase) Execute(ctx context.Context, document *entites.Document, pdf []byte) (*PDFAnalysis, error) {
	if err := validateDocument(document); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	job, err := u.store.Begin(ctx, document)
	if err != nil {
		return nil, err
	}
	out := &PDFAnalysis{Document: document, Job: job}
	result, err := u.analyzer.Analyze(ctx, pdf)
	if err == nil && result == nil {
		err = ErrAnalysisFailed
	}
	if err == nil {
		result.JobID = job.ID
		err = u.store.Complete(ctx, job, result)
	}
	if err != nil {
		// Persist failure even if the HTTP request was cancelled. Never store raw
		// upstream errors, which may contain document contents or credentials.
		failureCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer stop()
		failureErr := u.store.Fail(failureCtx, job, "Analysis failed; retry with a new upload.")
		if failureErr == nil {
			document.Status = entites.DocumentStatusFailed
		}
		return out, errors.Join(ErrAnalysisFailed, err, failureErr)
	}
	document.Status = entites.DocumentStatusCompleted
	out.Result = result
	return out, nil
}
