package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Archive struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	Name      string             `bson:"name" json:"name"`
	Size      int64              `bson:"size" json:"size"`
	SizeMB    string             `bson:"-" json:"size_mb"`
	DeletedAt *time.Time         `bson:"deleted_at,omitempty" json:"deleted_at"`
	ExpiresAt *time.Time         `bson:"expires_at,omitempty" json:"expires_at"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	IsTemp    bool               `bson:"is_temp" json:"is_temp"`
}

func (a *Archive) FormatSize() {
	const (
		KB = 1024
		MB = KB * 1024
	)

	// Always convert to MB
	a.SizeMB = fmt.Sprintf("%.2f MB", float64(a.Size)/float64(MB))
}

type ArchiveRepository interface {
	Save(ctx context.Context, file FileContent) error
	FindByID(ctx context.Context, id string) (*Archive, []byte, error)
	FindAll(ctx context.Context, page, limit int) ([]Archive, int64, error)
	FindByIDs(ctx context.Context, ids []string) ([]Archive, error)
	Delete(ctx context.Context, id string, deleteType DeleteType) error
	Restore(ctx context.Context, id string) error
	Exists(ctx context.Context, id string) (bool, error)
	DeleteExpiredTempFiles(ctx context.Context) error
}

type FileContent struct {
	Name    string
	Content []byte
}

type DeleteType int

const (
	SoftDelete DeleteType = iota
	HardDelete
	TempDelete
)

var (
	ErrArchiveNotFound   = errors.New("archive not found")
	ErrDeleteNotAllowed  = errors.New("delete operation not allowed")
	ErrAlreadyDeleted    = errors.New("archive already deleted")
	ErrRestoreNotAllowed = errors.New("restore operation not allowed")
	ErrAlreadyExpire     = errors.New("archive already expired")
)

func (dt DeleteType) String() string {
	switch dt {
	case SoftDelete:
		return "soft"
	case HardDelete:
		return "hard"
	case TempDelete:
		return "temporary"
	default:
		return "unknown"
	}
}
