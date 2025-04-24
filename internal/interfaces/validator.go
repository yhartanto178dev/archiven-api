package interfaces

import (
	"errors"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type FileValidator struct {
	MaxSize      int64
	AllowedTypes []string
	AllowedExt   string
	logger       *zap.Logger
}

func NewFileValidator(maxSize int64, allowedTypes []string, ext string, logger *zap.Logger) *FileValidator {
	return &FileValidator{
		MaxSize:      maxSize,
		AllowedTypes: allowedTypes,
		AllowedExt:   ext,
		logger:       logger,
	}
}

func (v *FileValidator) ValidateUpload(file *multipart.FileHeader) error {
	// Validasi ukuran file
	if file.Size > v.MaxSize {
		v.logger.Warn("File terlalu besar",
			zap.String("filename", file.Filename),
			zap.Int64("size", file.Size),
		)
		return ErrFileTooLarge
	}

	// Validasi ekstensi file
	ext := filepath.Ext(file.Filename)
	if ext != v.AllowedExt {
		v.logger.Warn("Ekstensi file tidak valid",
			zap.String("filename", file.Filename),
			zap.String("ext", ext),
		)
		return ErrInvalidExtension
	}

	// Validasi MIME type
	src, err := file.Open()
	if err != nil {
		v.logger.Error("Gagal membuka file",
			zap.String("filename", file.Filename),
			zap.Error(err),
		)
		return err
	}
	defer src.Close()

	header := make([]byte, 512)
	if _, err = src.Read(header); err != nil {
		v.logger.Error("Gagal membaca header file",
			zap.String("filename", file.Filename),
			zap.Error(err),
		)
		return err
	}

	mimeType := http.DetectContentType(header)
	if !contains(v.AllowedTypes, mimeType) {
		v.logger.Warn("Tipe file tidak diizinkan",
			zap.String("filename", file.Filename),
			zap.String("mime", mimeType),
		)
		return ErrInvalidFileType
	}

	return nil
}

func (v *FileValidator) MapDomainError(err error) *echo.HTTPError {
	switch {
	case errors.Is(err, ErrFileTooLarge):
		return echo.NewHTTPError(http.StatusBadRequest, "Ukuran file melebihi batas")
	case errors.Is(err, ErrInvalidExtension):
		return echo.NewHTTPError(http.StatusBadRequest, "Ekstensi file tidak valid")
	case errors.Is(err, ErrInvalidFileType):
		return echo.NewHTTPError(http.StatusBadRequest, "Tipe file tidak diizinkan")
	case errors.Is(err, ErrInvalidPDF):
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "File PDF tidak valid")
	case errors.Is(err, ErrArchiveNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "Archive not found")
	case errors.Is(err, ErrDeleteNotAllowed):
		return echo.NewHTTPError(http.StatusForbidden, "Delete operation not allowed")
	case errors.Is(err, ErrAlreadyDeleted):
		return echo.NewHTTPError(http.StatusConflict, "Archive already deleted")
	case errors.Is(err, ErrRestoreNotAllowed):
		return echo.NewHTTPError(http.StatusForbidden, "Restore operation not allowed")
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, "Terjadi kesalahan server")
	}
}
