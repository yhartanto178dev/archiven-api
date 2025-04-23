package domain

import (
	"context"
	"time"
)

type Archive struct {
	ID        string
	Name      string
	Size      int64
	CreatedAt time.Time
}

type ArchiveRepository interface {
	Save(ctx context.Context, file FileContent) error
	FindByID(ctx context.Context, id string) (*Archive, []byte, error)
	FindAll(ctx context.Context, page, limit int) ([]Archive, int64, error)
	FindByIDs(ctx context.Context, ids []string) ([]Archive, error)
}

type FileContent struct {
	Name    string
	Content []byte
}
