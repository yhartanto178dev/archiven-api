package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Archive struct {
	ID          primitive.ObjectID `bson:"_id" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Size        int64              `bson:"size" json:"size"`
	SizeMB      string             `bson:"-" json:"size_mb"`
	Category    string             `bson:"category" json:"category"`
	Type        string             `bson:"type" json:"type"`
	Tags        []string           `bson:"tags" json:"tags"`
	Description string             `bson:"description" json:"description"`
	OwnerID     string             `bson:"owner_id" json:"owner_id"`
	Version     int                `bson:"version" json:"version"`
	DeletedAt   *time.Time         `bson:"deleted_at,omitempty" json:"deleted_at"`
	ExpiresAt   *time.Time         `bson:"expires_at,omitempty" json:"expires_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	IsTemp      bool               `bson:"is_temp" json:"is_temp"`
	ChangeLogs  []ChangeLog        `bson:"change_logs" json:"change_logs"`
}

type ChangeLog struct {
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
	Action    string    `bson:"action" json:"action"` // upload, update, delete, restore
	UserID    string    `bson:"user_id" json:"user_id"`
	Changes   []Change  `bson:"changes" json:"changes"`
}

type HistoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	User      string    `json:"user"`
	Changes   []Change  `json:"changes"`
}
type History struct {
	ID       string         `json:"id"`
	FileName string         `json:"file_name"`
	Logs     []HistoryEntry `json:"logs"`
}

type Change struct {
	Field    string      `bson:"field" json:"field"`
	OldValue interface{} `bson:"old_value" json:"old_value"`
	NewValue interface{} `bson:"new_value" json:"new_value"`
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
	RestoreArchive(ctx context.Context, id string) error
	Exists(ctx context.Context, id string) (bool, error)
	DeleteExpiredTempFiles(ctx context.Context) error
	FindExistingArchive(ctx context.Context, archive Archive) (*Archive, error)
	SaveWithVersioning(context.Context, Archive, []byte) (*Archive, error)
	GetHistory(ctx context.Context, id string) (*History, error)
	GetByCategory(ctx context.Context, category string, page, limit int) ([]Archive, int64, error)
	GetByTags(ctx context.Context, tags []string, page, limit int) ([]Archive, int64, error)
	DeleteExpiredFiles(ctx context.Context) (int64, error)
	DeleteByFilter(ctx context.Context, filter bson.M) (int64, error)
}

type FileContent struct {
	Name      string
	Content   []byte
	Size      int64
	MimeType  string
	Extension string
	CreatedAt time.Time
}
type ArchiveMetadata struct {
	Name        string   `bson:"name" json:"name"`
	Category    string   `bson:"category" json:"category" validate:"required"`
	Type        string   `bson:"type" json:"type" validate:"required"`
	Tags        []string `bson:"tags" json:"tags" validate:"required,min=1,max=5"`
	Description string   `bson:"description" json:"description"`
	OwnerID     string   `bson:"owner_id" json:"owner_id" validate:"required"`
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
	ErrInvalidCategory   = errors.New("invalid category")
	ErrTagsRequired      = errors.New("tags are required")
	ErrNotDeleted        = errors.New("archive not deleted")
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
