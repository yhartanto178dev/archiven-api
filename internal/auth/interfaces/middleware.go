package interfaces

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/yhartanto178dev/archiven-api/internal/auth/application"
	"github.com/yhartanto178dev/archiven-api/internal/auth/domain"
)

type AuthMiddleware struct {
	authService *application.AuthService
}

func NewAuthMiddleware(authService *application.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

func (m *AuthMiddleware) JWTAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get token from Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			var token string

			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			} else {
				// Try to get token from cookie
				if cookie, err := c.Cookie("access_token"); err == nil {
					token = cookie.Value
				}
			}

			if token == "" {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"status":  "error",
					"message": "Access token required",
				})
			}

			// Validate token
			claims, err := m.authService.ValidateToken(c.Request().Context(), token)
			if err != nil {
				status := http.StatusUnauthorized
				message := "Invalid token"

				switch err {
				case domain.ErrTokenExpired:
					message = "Token expired"
				case domain.ErrTokenRevoked:
					message = "Token revoked"
				case domain.ErrInvalidToken:
					message = "Invalid token"
				}

				return c.JSON(status, map[string]interface{}{
					"status":  "error",
					"message": message,
				})
			}

			// Set user information in context
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("email", claims.Email)
			c.Set("role", claims.Role)
			c.Set("token_id", claims.TokenID)

			return next(c)
		}
	}
}

func (m *AuthMiddleware) RequireRole(role string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole := c.Get("role").(string)
			if userRole != role {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"status":  "error",
					"message": "Insufficient permissions",
				})
			}
			return next(c)
		}
	}
}

func (m *AuthMiddleware) RequireAnyRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole := c.Get("role").(string)
			for _, role := range roles {
				if userRole == role {
					return next(c)
				}
			}
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"status":  "error",
				"message": "Insufficient permissions",
			})
		}
	}
}
