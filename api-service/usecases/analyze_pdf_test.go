package usecases

import (
	"context"
	"errors"
	"testing"

	"cloud-native-platform/api-service/entites"
)

type fakeAnalysisStore struct {
	completed, failed     bool
	beginErr, completeErr error
	cleanupActive         bool
}

func (s *fakeAnalysisStore) Begin(_ context.Context, d *entites.Document) (*entites.Job, error) {
	if s.beginErr != nil {
		return nil, s.beginErr
	}
	d.ID = testID
	d.Status = entites.DocumentStatusProcessing
	return &entites.Job{ID: testID, DocumentID: d.ID}, nil
}
func (s *fakeAnalysisStore) Complete(_ context.Context, _ *entites.Job, r *entites.AnalysisResult) error {
	s.completed = true
	if r.JobID != testID {
		panic("missing job ID")
	}
	return s.completeErr
}
func (s *fakeAnalysisStore) Fail(ctx context.Context, _ *entites.Job, _ string) error {
	s.failed = true
	s.cleanupActive = ctx.Err() == nil
	return nil
}

type fakeAnalyzer struct {
	err    error
	cancel context.CancelFunc
	called bool
}

func (a *fakeAnalyzer) Analyze(_ context.Context, _ []byte) (*entites.AnalysisResult, error) {
	a.called = true
	if a.cancel != nil {
		a.cancel()
	}
	return &entites.AnalysisResult{}, a.err
}

func TestAnalysisWorkflow(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		begin, analysis, complete bool
	}{
		{"success", false, false, false}, {"begin failure", true, false, false}, {"upstream failure", false, true, false}, {"save failure", false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			failure := errors.New("test failure")
			store := &fakeAnalysisStore{}
			ai := &fakeAnalyzer{}
			if tc.begin {
				store.beginErr = failure
			}
			if tc.analysis {
				ai.err = failure
			}
			if tc.complete {
				store.completeErr = failure
			}
			u := NewAnalyzePDFUseCase(store, ai)
			d := &entites.Document{OriginalFilename: "a.pdf", StorageKey: "key", MIMEType: "application/pdf"}
			out, err := u.Execute(context.Background(), d, []byte("%PDF-test"))
			if tc.begin {
				if err == nil || out != nil || ai.called {
					t.Fatal("begin failure not isolated")
				}
				return
			}
			if tc.analysis || tc.complete {
				if !errors.Is(err, ErrAnalysisFailed) || !store.failed || d.Status != entites.DocumentStatusFailed {
					t.Fatal("failure not persisted")
				}
				return
			}
			if err != nil || out.Result == nil || !store.completed || d.Status != entites.DocumentStatusCompleted {
				t.Fatal("incomplete success")
			}
		})
	}
}

func TestCancelledAnalysisStillPersistsFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := &fakeAnalysisStore{}
	ai := &fakeAnalyzer{err: context.Canceled, cancel: cancel}
	_, err := NewAnalyzePDFUseCase(store, ai).Execute(ctx, &entites.Document{OriginalFilename: "a.pdf", StorageKey: "key", MIMEType: "application/pdf"}, []byte("%PDF-test"))
	if !errors.Is(err, context.Canceled) || !store.cleanupActive {
		t.Fatal("failure cleanup used cancelled context")
	}
}
