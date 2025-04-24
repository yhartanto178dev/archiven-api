package interfaces

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/yhartanto178dev/archiven-api/internal/archive/application"
	"github.com/yhartanto178dev/archiven-api/internal/archive/domain"
	"go.uber.org/zap"

	"github.com/labstack/echo/v4"
)

// Error response

type ArchiveHandler struct {
	service   *application.ArchiveService
	validator *FileValidator
	logger    *zap.Logger
}

func NewArchiveHandler(service *application.ArchiveService, validator *FileValidator,
	logger *zap.Logger) *ArchiveHandler {
	return &ArchiveHandler{service: service, validator: validator,
		logger: logger}
}

func (h *ArchiveHandler) Upload(c echo.Context) error {
	startTime := time.Now()
	// Error NewErrorResponseBuilder
	ErrorResponse := NewErrorResponseBuilder()
	//SuccessResponse
	SuccessResponse := NewSuccessResponseBuilder()
	// Parse the multipart form

	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse(ResponseErrorFileNotFound))
	}

	// 2. Validasi file di handler
	if err := h.validator.ValidateUpload(file); err != nil {
		return h.validator.MapDomainError(err)
	}

	src, err := file.Open()
	if err != nil {
		h.logger.Error("Gagal membuka file",
			zap.String("filename", file.Filename),
			zap.Error(err),
		)
		return c.JSON(http.StatusBadRequest, ErrorResponse(ResponseErrorOpenFile))
	}
	defer src.Close()

	content := make([]byte, file.Size)
	_, err = src.Read(content)
	if err != nil {
		h.logger.Error("Gagal membaca file",
			zap.String("filename", file.Filename),
			zap.Error(err),
		)
		return c.JSON(http.StatusInternalServerError, ErrorResponse(ResponseErrorOpenFile))
	}

	errUpload := h.service.UploadArchive(c.Request().Context(), domain.FileContent{
		Name:    file.Filename,
		Content: content,
	})

	if errUpload != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse(ResponseErrorUploadToMongo))
	}

	h.logger.Info("Upload berhasil",
		zap.String("filename", file.Filename),
		zap.Duration("duration", time.Since(startTime)),
	)
	// Return success response

	return c.JSON(http.StatusCreated, SuccessResponse(ResponseSuccessUpload))
}
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (h *ArchiveHandler) List(c echo.Context) error {
	// Error NewErrorResponseBuilder
	ErrorResponse := NewErrorResponseBuilder()

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	archives, total, err := h.service.ListArchives(c.Request().Context(), page, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse(ResponseErrorListArchive))
	}

	// Convert archives to response format
	var response []ArchiveResponse
	for _, a := range archives {
		response = append(response, ToArchiveResponse(&a))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": response,
		"pagination": map[string]interface{}{
			"page":       page,
			"limit":      limit,
			"totalData":  total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

func (h *ArchiveHandler) Download(c echo.Context) error {
	id := c.Param("id")

	_, buf, err := h.service.GetArchive(c.Request().Context(), id)
	if err != nil {
		errBuilder := NewErrorResponseBuilder()
		switch {
		case errors.Is(err, domain.ErrArchiveNotFound):
			return c.JSON(404, errBuilder(ResponseErrorFileNotFound))
		case errors.Is(err, domain.ErrAlreadyDeleted):
			return c.JSON(403, errBuilder("File has been deleted"))
		case errors.Is(err, domain.ErrAlreadyExpire):
			return c.JSON(403, errBuilder("File has expired"))
		default:
			return c.JSON(500, errBuilder(ResponseErrorGetArchive))
		}
	}

	return c.Blob(200, "application/octet-stream", buf)
}

// Tambahkan handler baru
func (h *ArchiveHandler) GetByIDs(c echo.Context) error {
	// Error NewErrorResponseBuilder
	ErrorResponse := NewErrorResponseBuilder()
	ids := c.QueryParams()["id"]
	if len(ids) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing ids parameter"})
	}

	archives, err := h.service.GetArchivesByIDs(c.Request().Context(), ids)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse(ResponseErrorGetArchive))
	}

	var response []ArchiveResponse
	for _, a := range archives {
		response = append(response, ToArchiveResponse(&a))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":  response,
		"count": len(response),
	})
}

func (h *ArchiveHandler) DeleteArchive(c echo.Context) error {
	id := c.Param("id")
	deleteType := getDeleteTypeFromParam(c)

	if err := h.service.DeleteArchive(c.Request().Context(), id, deleteType); err != nil {
		return h.validator.MapDomainError(err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "File deleted successfully",
		"type":    deleteType.String(),
	})
}

func (h *ArchiveHandler) RestoreArchive(c echo.Context) error {
	id := c.Param("id")

	if err := h.service.RestoreArchive(c.Request().Context(), id); err != nil {
		return h.validator.MapDomainError(err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "File restored successfully",
	})
}

func getDeleteTypeFromParam(c echo.Context) domain.DeleteType {
	switch c.QueryParam("type") {
	case "permanent":
		return domain.HardDelete
	case "temporary":
		return domain.TempDelete
	default:
		return domain.SoftDelete
	}
}
