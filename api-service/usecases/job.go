package usecases

import (
	"cloud-native-platform/api-service/entites"
	"context"
)

// JobRepository is the storage contract required by JobUseCase.
type JobRepository interface {
	Create(context.Context, *entites.Job) error
	GetByID(context.Context, string) (*entites.Job, error)
	List(context.Context, int, int) ([]entites.Job, error)
	Update(context.Context, *entites.Job) error
	Delete(context.Context, string) error
	ListByDocumentID(context.Context, string, int, int) ([]entites.Job, error)
}

type JobUseCase struct{ repository JobRepository }

func NewJobUseCase(repository JobRepository) *JobUseCase {
	return &JobUseCase{repository: repository}
}

// Create validates input and fills database-generated fields after success.
func (u *JobUseCase) Create(ctx context.Context, value *entites.Job) error {
	if err := validateJob(value); err != nil {
		return err
	}
	copy := *value

	if err := u.repository.Create(ctx, &copy); err != nil {
		return err
	}
	*value = copy
	return nil
}

func (u *JobUseCase) GetByID(ctx context.Context, id string) (*entites.Job, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	return u.repository.GetByID(ctx, id)
}

// List accepts a limit of 1..100 and a non-negative offset.
func (u *JobUseCase) List(ctx context.Context, limit, offset int) ([]entites.Job, error) {
	if err := validatePagination(limit, offset); err != nil {
		return nil, err
	}
	return u.repository.List(ctx, limit, offset)
}

func (u *JobUseCase) ListByDocumentID(ctx context.Context, id string, limit, offset int) ([]entites.Job, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	if err := validatePagination(limit, offset); err != nil {
		return nil, err
	}
	return u.repository.ListByDocumentID(ctx, id, limit, offset)
}

// Update replaces all mutable fields, including nullable fields.
func (u *JobUseCase) Update(ctx context.Context, value *entites.Job) error {
	if err := validateJob(value); err != nil {
		return err
	}
	if err := validateID(value.ID); err != nil {
		return err
	}
	copy := *value
	if err := u.repository.Update(ctx, &copy); err != nil {
		return err
	}
	*value = copy
	return nil
}

// Delete removes database metadata and any cascading child records.
// File storage is not modified.
func (u *JobUseCase) Delete(ctx context.Context, id string) error {
	if err := validateID(id); err != nil {
		return err
	}
	return u.repository.Delete(ctx, id)
}
