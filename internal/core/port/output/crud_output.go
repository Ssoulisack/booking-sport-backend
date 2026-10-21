package output

import "context"

// ListQuery is a paginated list request. Filters are exact-match column filters
// chosen by the service, never taken straight from user input.
type ListQuery struct {
	Page    int
	Limit   int
	Keyword string
	Filters map[string]any
}

// CRUDRepository is the basic storage contract shared by simple entities
type CRUDRepository[T any] interface {
	Create(ctx context.Context, m *T) error
	GetByID(ctx context.Context, id uint) (*T, error)
	List(ctx context.Context, q ListQuery) ([]T, int64, error)
	Update(ctx context.Context, m *T) error
	Delete(ctx context.Context, id uint) error
}
