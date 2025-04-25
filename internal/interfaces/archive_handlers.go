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
	// SuccessResponse := NewSuccessResponseBuilder()
	// Parse the multipart form
	var req UploadRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse(ResponseErrorValidRequest))
	}

	// 2. Validasi file di handler
	if err := h.validator.ValidateUpload(req.File); err != nil {
		return h.validator.MapDomainError(err)
	}

	src, err := req.File.Open()
	if err != nil {
		h.logger.Error("Gagal membuka file",
			zap.String("filename", req.File.Filename),
			zap.Error(err),
		)
		return c.JSON(http.StatusBadRequest, ErrorResponse(ResponseErrorOpenFile))
	}
	defer src.Close()
	userID := c.Get("user_id").(string)

	content := make([]byte, req.File.Size)
	_, err = src.Read(content)
	if err != nil {
		h.logger.Error("Gagal membaca file",
			zap.String("filename", req.File.Filename),
			zap.Error(err),
		)
		return c.JSON(http.StatusInternalServerError, ErrorResponse(ResponseErrorOpenFile))
	}

	archive, errUpload := h.service.UploadArchive(c.Request().Context(), domain.FileContent{
		Name:    req.File.Filename,
		Content: content,
		Size:    req.File.Size,
	}, domain.ArchiveMetadata{
		Category:    req.Category,
		Type:        req.Type,
		Tags:        req.Tags,
		Description: req.Description,
		OwnerID:     userID,
	})

	if errUpload != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse(ResponseErrorUploadToMongo))
	}

	h.logger.Info("Upload berhasil",
		zap.String("filename", req.File.Filename),
		zap.Duration("duration", time.Since(startTime)),
	)
	// Return success response
	SuccessResponseData := map[string]interface{}{
		"id":      archive.ID,
		"version": archive.Version,
		"isNew":   archive.Version == 1,
	}

	return c.JSON(http.StatusCreated, SuccessResponseData)
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
	ErrorResponse := NewErrorResponseBuilder()

	// Get IDs from query parameters
	ids := c.QueryParams()["id"]
	if len(ids) == 0 {
		return c.JSON(http.StatusBadRequest, ErrorResponse("Missing IDs parameter"))
	}

	archives, err := h.service.GetArchivesByIDs(c.Request().Context(), ids)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrArchiveNotFound):
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"status":  "error",
				"message": "file not found",
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "failed to retrieve files",
			})
		}
	}

	// Convert archives to response format
	var response []ArchiveResponse
	for _, a := range archives {
		response = append(response, ToArchiveResponse(&a))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"archives": response,
			"metadata": map[string]interface{}{
				"total_files": len(response),
				"ids":         ids,
			},
		},
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

func (h *ArchiveHandler) GetHistory(c echo.Context) error {
	id := c.Param("id")
	ErrorResponse := NewErrorResponseBuilder()

	history, err := h.service.GetHistory(c.Request().Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrArchiveNotFound):
			return c.JSON(http.StatusNotFound, ErrorResponse(ResponseErrorFileNotFound))
		default:
			return c.JSON(http.StatusInternalServerError, ErrorResponse(ResponseErrorGetArchive))
		}
	}

	// Format response with versioning info
	response := map[string]interface{}{
		"id":             history.ID,
		"file_name":      history.FileName,
		"total_versions": len(history.Logs),
		"history":        history.Logs,
	}

	return c.JSON(http.StatusOK, response)
}

func (h *ArchiveHandler) GetByCategory(c echo.Context) error {
	category := c.Param("category")
	ErrorResponse := NewErrorResponseBuilder()

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	archives, total, err := h.service.GetByCategory(c.Request().Context(), category, page, limit)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCategory):
			return c.JSON(http.StatusBadRequest, ErrorResponse("Invalid category"))
		default:
			return c.JSON(http.StatusInternalServerError, ErrorResponse(ResponseErrorGetArchive))
		}
	}

	// Convert archives to response format
	var response []ArchiveResponse
	for _, a := range archives {
		response = append(response, ToArchiveResponse(&a))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": response,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total_data":  total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
		"category": category,
	})
}

func (h *ArchiveHandler) GetByTags(c echo.Context) error {
	tags := c.QueryParams()["tag"]
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	archives, total, err := h.service.GetByTags(c.Request().Context(), tags, page, limit)
	if err != nil {
		return h.validator.MapDomainError(err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":  archives,
		"total": total,
		"page":  page,
		"limit": limit,
		"tags":  tags,
	})
}
