package interfaces

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/yhartanto178dev/archiven-api/internal/archive/domain"
	"github.com/yhartanto178dev/archiven-api/internal/configs"
)

// Error response
const (
	ResponseErrorMultiprt      = "failed to parse multipart form"
	ResponseErrorOpenFile      = "failed to open file"
	ResponseErrorUploadToMongo = "failed to upload file to MongoDB"
	ResponseErrorListArchive   = "failed to get list archives"
	ResponseErrorGetArchive    = "failed to get archive"
	ResponseErrorLimitUpload   = "file size exceeds the limit"
	ResponseErrorFileType      = "file type not allowed"
	ResponseErrorFileNotFound  = "file not found"
)

// Success response
const (
	ResponseSuccessUpload = "File uploaded successfully"
)

type ArchiveResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SizeMB      string `json:"size_mb"`
	DownloadURL string `json:"download_url"`
	CreatedAt   string `json:"created_at"`
}

func ToArchiveResponse(archive *domain.Archive) ArchiveResponse {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, using system environment variables")
	}

	cfg := configs.Load()
	// Convert bytes to MB with 2 decimal places
	sizeMB := fmt.Sprintf("%.2f MB", float64(archive.Size)/(1024*1024))

	return ArchiveResponse{
		ID:          archive.ID,
		Name:        archive.Name,
		SizeMB:      sizeMB,
		DownloadURL: fmt.Sprintf("http://%s:%s/download/%s", cfg.Host, strconv.Itoa(cfg.ServerPort), archive.ID),
		CreatedAt:   archive.CreatedAt.Format(time.RFC3339),
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
