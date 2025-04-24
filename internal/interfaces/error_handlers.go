package interfaces

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// Custom error handler
func CreateErrorHandler(l *zap.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		ErrorResponse := NewErrorResponseBuilder()
		code := http.StatusInternalServerError
		message := "Internal Server Error"

		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
			message = fmt.Sprintf("%v", he.Message)
		}

		l.Error("server error",
			zap.Error(err),
			zap.Int("status", code),
			zap.String("path", c.Path()),
		)

		c.JSON(code, ErrorResponse(message))
	}
}
