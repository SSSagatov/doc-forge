package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"cloud-native-platform/api-service/database/repository"
	"cloud-native-platform/api-service/entites"
)

var (
	_ DocumentRepository       = (*repository.DocumentRepository)(nil)
	_ JobRepository            = (*repository.JobRepository)(nil)
	_ AnalysisResultRepository = (*repository.AnalysisResultRepository)(nil)
)

const testID = "12345678-1234-1234-1234-123456789abc"

type documentStore struct {
	DocumentRepository
	create func(context.Context, *entites.Document) error
}

func (s documentStore) Create(ctx context.Context, v *entites.Document) error {
	return s.create(ctx, v)
}

func TestDocumentCreate(t *testing.T) {
	ctx := context.Background()
	document := entites.Document{OriginalFilename: "a.pdf", StorageKey: "key", MIMEType: "application/pdf"}
	u := NewDocumentUseCase(documentStore{create: func(got context.Context, value *entites.Document) error {
		if got != ctx || value.Status != entites.DocumentStatusUploaded {
			t.Fatal("context or default status not forwarded")
		}
		value.ID = testID
		return nil
	}})
	if err := u.Create(ctx, &document); err != nil {
		t.Fatal(err)
	}
	if document.ID != testID || document.Status != entites.DocumentStatusUploaded {
		t.Fatal("created fields not returned")
	}

	failure := errors.New("storage failed")
	before := document
	u = NewDocumentUseCase(documentStore{create: func(_ context.Context, value *entites.Document) error {
		value.ID = "changed"
		return failure
	}})
	if err := u.Create(ctx, &document); !errors.Is(err, failure) {
		t.Fatalf("error = %v", err)
	}
	if document != before {
		t.Fatal("failed create changed input")
	}
}

func TestInvalidInputDoesNotReachStorage(t *testing.T) {
	ctx := context.Background()
	// Nil stores ensure rejected inputs cannot silently reach persistence.
	documents := NewDocumentUseCase(nil)
	jobs := NewJobUseCase(nil)
	results := NewAnalysisResultUseCase(nil)
	for name, call := range map[string]func() error{
		"nil document":      func() error { return documents.Create(ctx, nil) },
		"missing metadata":  func() error { return documents.Create(ctx, &entites.Document{}) },
		"invalid status":    func() error { return documents.UpdateStatus(ctx, testID, "unknown") },
		"nil job":           func() error { return jobs.Create(ctx, nil) },
		"invalid parent":    func() error { return jobs.Create(ctx, &entites.Job{DocumentID: "bad"}) },
		"negative attempts": func() error { return jobs.Create(ctx, &entites.Job{DocumentID: testID, Attempts: -1}) },
		"missing update ID": func() error { return jobs.Update(ctx, &entites.Job{DocumentID: testID}) },
		"nil result":        func() error { return results.Create(ctx, nil) },
		"malformed JSON": func() error {
			return results.Create(ctx, &entites.AnalysisResult{JobID: testID, Risks: json.RawMessage(`{`)})
		},
		"invalid delete ID":  func() error { return results.Delete(ctx, "bad") },
		"invalid pagination": func() error { _, err := jobs.ListByDocumentID(ctx, testID, 101, 0); return err },
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestUUIDValidation(t *testing.T) {
	for _, id := range []string{testID, "ABCDEF12-1234-5678-90AB-123456789ABC"} {
		if err := validateID(id); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"", "12345678123412341234123456789abc", "zzzzzzzz-1234-1234-1234-123456789abc"} {
		if !errors.Is(validateID(id), ErrInvalidInput) {
			t.Fatalf("invalid UUID accepted: %q", id)
		}
	}
}
