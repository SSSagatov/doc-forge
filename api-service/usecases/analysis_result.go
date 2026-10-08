package usecases

import (
	"cloud-native-platform/api-service/entites"
	"context"
)

// AnalysisResultRepository is the storage contract required by AnalysisResultUseCase.
type AnalysisResultRepository interface {
	Create(context.Context, *entites.AnalysisResult) error
	GetByID(context.Context, string) (*entites.AnalysisResult, error)
	List(context.Context, int, int) ([]entites.AnalysisResult, error)
	Update(context.Context, *entites.AnalysisResult) error
	Delete(context.Context, string) error
	ListByJobID(context.Context, string, int, int) ([]entites.AnalysisResult, error)
}

type AnalysisResultUseCase struct{ repository AnalysisResultRepository }

func NewAnalysisResultUseCase(repository AnalysisResultRepository) *AnalysisResultUseCase {
	return &AnalysisResultUseCase{repository: repository}
}

// Create validates input and fills database-generated fields after success.
func (u *AnalysisResultUseCase) Create(ctx context.Context, value *entites.AnalysisResult) error {
	if err := validateAnalysisResult(value); err != nil {
		return err
	}
	copy := *value

	if err := u.repository.Create(ctx, &copy); err != nil {
		return err
	}
	*value = copy
	return nil
}

func (u *AnalysisResultUseCase) GetByID(ctx context.Context, id string) (*entites.AnalysisResult, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	return u.repository.GetByID(ctx, id)
}

// List accepts a limit of 1..100 and a non-negative offset.
func (u *AnalysisResultUseCase) List(ctx context.Context, limit, offset int) ([]entites.AnalysisResult, error) {
	if err := validatePagination(limit, offset); err != nil {
		return nil, err
	}
	return u.repository.List(ctx, limit, offset)
}

func (u *AnalysisResultUseCase) ListByJobID(ctx context.Context, id string, limit, offset int) ([]entites.AnalysisResult, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	if err := validatePagination(limit, offset); err != nil {
		return nil, err
	}
	return u.repository.ListByJobID(ctx, id, limit, offset)
}

// Update replaces all mutable fields, including nullable fields.
func (u *AnalysisResultUseCase) Update(ctx context.Context, value *entites.AnalysisResult) error {
	if err := validateAnalysisResult(value); err != nil {
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
func (u *AnalysisResultUseCase) Delete(ctx context.Context, id string) error {
	if err := validateID(id); err != nil {
		return err
	}
	return u.repository.Delete(ctx, id)
}
