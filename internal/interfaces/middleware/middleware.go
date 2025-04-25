package middlewares

import "github.com/labstack/echo/v4"

func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Implementasi auth logic
		userID := "user123" // Contoh
		c.Set("user_id", userID)
		return next(c)
	}
}
