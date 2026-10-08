package usecases

import (
	"cloud-native-platform/api-service/entites"
	"context"
)

// DocumentRepository is the storage contract required by DocumentUseCase.
type DocumentRepository interface {
	Create(context.Context, *entites.Document) error
	GetByID(context.Context, string) (*entites.Document, error)
	List(context.Context, int, int) ([]entites.Document, error)
	UpdateStatus(context.Context, string, entites.DocumentStatus) error
	Delete(context.Context, string) error
}

type DocumentUseCase struct{ repository DocumentRepository }

func NewDocumentUseCase(repository DocumentRepository) *DocumentUseCase {
	return &DocumentUseCase{repository: repository}
}

// Create validates input and fills database-generated fields after success.
func (u *DocumentUseCase) Create(ctx context.Context, value *entites.Document) error {
	if err := validateDocument(value); err != nil {
		return err
	}
	copy := *value
	if copy.Status == "" {
		copy.Status = entites.DocumentStatusUploaded
	}
	if err := u.repository.Create(ctx, &copy); err != nil {
		return err
	}
	*value = copy
	return nil
}

func (u *DocumentUseCase) GetByID(ctx context.Context, id string) (*entites.Document, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	return u.repository.GetByID(ctx, id)
}

// List accepts a limit of 1..100 and a non-negative offset.
func (u *DocumentUseCase) List(ctx context.Context, limit, offset int) ([]entites.Document, error) {
	if err := validatePagination(limit, offset); err != nil {
		return nil, err
	}
	return u.repository.List(ctx, limit, offset)
}

func (u *DocumentUseCase) UpdateStatus(ctx context.Context, id string, status entites.DocumentStatus) error {
	if err := validateID(id); err != nil {
		return err
	}
	if !validStatus(status) {
		return invalid("unknown document status")
	}
	return u.repository.UpdateStatus(ctx, id, status)
}

// Delete removes database metadata and any cascading child records.
// File storage is not modified.
func (u *DocumentUseCase) Delete(ctx context.Context, id string) error {
	if err := validateID(id); err != nil {
		return err
	}
	return u.repository.Delete(ctx, id)
}
