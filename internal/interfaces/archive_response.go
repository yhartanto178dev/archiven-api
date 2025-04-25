package interfaces

import (
	"errors"
	"time"

	"github.com/yhartanto178dev/archiven-api/internal/archive/domain"
)

// Error response
const (
	ResponseErrorFileNotFound     = "failed file not found"
	ResponseErrorOpenFile         = "failed to open file"
	ResponseErrorUploadToMongo    = "failed to upload file to MongoDB"
	ResponseErrorListArchive      = "failed to get list archives"
	ResponseErrorGetArchive       = "failed to get archive"
	ResponseErrorLimitUpload      = "file size exceeds the limit"
	ResponseErrorFileType         = "file type not allowed"
	ResponseErrorValidRequest     = "failed to bind request"
	ResponseErrorUploadFile       = "failed to upload file"
	ResponseErrorHeaderRead       = "failed to read header"
	ResponseErrorValidationStages = "failed validation stage"
)

var (
	ErrFileTooLarge      = errors.New("file too large")
	ErrInvalidFileType   = errors.New("invalid file type")
	ErrInvalidExtension  = errors.New("invalid file extension")
	ErrInvalidPDF        = errors.New("invalid PDF structure")
	ErrVirusDetected     = errors.New("virus detected")
	ErrValidationTimeout = errors.New("validation timeout")
	ErrArchiveNotFound   = errors.New("archive not found")
	ErrDeleteNotAllowed  = errors.New("delete operation not allowed")
	ErrAlreadyDeleted    = errors.New("archive already deleted")
	ErrRestoreNotAllowed = errors.New("restore operation not allowed")
	ErrCategoryRequired  = errors.New("category is required")
	ErrTooManyTags       = errors.New("too many tags, maximum 5 allowed")
	ErrTypeRequired      = errors.New("type is required")
	ErrTagsRequired      = errors.New("tags are required")
)

// Success response
const (
	ResponseSuccessUpload = "File uploaded successfully"
)

type ArchiveResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	SizeMB      string    `json:"size_mb"`
	Category    string    `json:"category"`
	Type        string    `json:"type"`
	Tags        []string  `json:"tags"`
	Description string    `json:"description"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func ToArchiveResponse(a *domain.Archive) ArchiveResponse {
	return ArchiveResponse{
		ID:          a.ID.Hex(),
		Name:        a.Name,
		Size:        a.Size,
		SizeMB:      a.SizeMB,
		Category:    a.Category,
		Type:        a.Type,
		Tags:        a.Tags,
		Description: a.Description,
		Version:     a.Version,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

type SuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ErrorResponseBuilder func(message string) ErrorResponse

type SuccessResponseBuilder func(message string) SuccessResponse

func NewErrorResponseBuilder() ErrorResponseBuilder {
	return func(message string) ErrorResponse {
		return ErrorResponse{
			Status:  "error",
			Message: message,
		}
	}
}
func NewSuccessResponseBuilder() SuccessResponseBuilder {
	return func(message string) SuccessResponse {
		return SuccessResponse{
			Status:  "success",
			Message: message,
		}
	}
}

type SuccessResponseWithDataVersion struct {
	Status string                 `json:"status"`
	Data   map[string]interface{} `json:"data"`
}

// type DataResponseVersion struct {
// 	ID        primitive.ObjectID `json:"archive_id"`
// 	Version   int                `json:"version"`
// 	IsNew    bool               `json:"is_new"`
// }

func NewSuccessResponseWithDataVersion(data map[string]interface{}) SuccessResponseWithDataVersion {
	return SuccessResponseWithDataVersion{
		Status: "success",
		Data:   data,
	}
}
