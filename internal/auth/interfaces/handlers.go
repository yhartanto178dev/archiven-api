package interfaces

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/yhartanto178dev/archiven-api/internal/auth/application"
	"github.com/yhartanto178dev/archiven-api/internal/auth/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuthHandler struct {
	authService *application.AuthService
}

func NewAuthHandler(authService *application.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req domain.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "Invalid request body",
		})
	}

	userAgent := c.Request().Header.Get("User-Agent")
	response, err := h.authService.Login(c.Request().Context(), req, userAgent)
	if err != nil {
		status := http.StatusUnauthorized
		message := "Invalid credentials"

		switch err {
		case domain.ErrUserNotFound:
			message = "User not found"
		case domain.ErrInvalidPassword:
			message = "Invalid password"
		case domain.ErrUserInactive:
			message = "User account is inactive"
		default:
			status = http.StatusInternalServerError
			message = "Internal server error"
		}

		return c.JSON(status, map[string]interface{}{
			"status":  "error",
			"message": message,
		})
	}

	// Set refresh token as HTTP-only cookie
	h.setRefreshTokenCookie(c, response.RefreshToken)

	// Set access token as secure cookie (optional, can also be sent in response)
	h.setAccessTokenCookie(c, response.AccessToken)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"access_token": response.AccessToken,
			"expires_in":   response.ExpiresIn,
			"token_type":   response.TokenType,
			"user":         response.User,
		},
	})
}

func (h *AuthHandler) RefreshToken(c echo.Context) error {
	// Try to get refresh token from cookie first
	refreshToken := h.getRefreshTokenFromCookie(c)

	// If not in cookie, try to get from request body
	if refreshToken == "" {
		var req domain.RefreshRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  "error",
				"message": "Invalid request",
			})
		}
		refreshToken = req.RefreshToken
	}

	if refreshToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "Refresh token required",
		})
	}

	response, err := h.authService.RefreshToken(c.Request().Context(), domain.RefreshRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		status := http.StatusUnauthorized
		message := "Invalid refresh token"

		switch err {
		case domain.ErrTokenExpired:
			message = "Refresh token expired"
		case domain.ErrTokenRevoked:
			message = "Refresh token revoked"
		default:
			status = http.StatusInternalServerError
			message = "Internal server error"
		}

		return c.JSON(status, map[string]interface{}{
			"status":  "error",
			"message": message,
		})
	}

	// Update cookies with new tokens
	h.setRefreshTokenCookie(c, response.RefreshToken)
	h.setAccessTokenCookie(c, response.AccessToken)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"access_token": response.AccessToken,
			"expires_in":   response.ExpiresIn,
			"token_type":   response.TokenType,
			"user":         response.User,
		},
	})
}

func (h *AuthHandler) Logout(c echo.Context) error {
	refreshToken := h.getRefreshTokenFromCookie(c)

	if refreshToken != "" {
		h.authService.Logout(c.Request().Context(), refreshToken)
	}

	// Clear cookies
	h.clearAuthCookies(c)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Successfully logged out",
	})
}

func (h *AuthHandler) LogoutAll(c echo.Context) error {
	userID := c.Get("user_id").(string)
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "Invalid user ID",
		})
	}

	if err := h.authService.LogoutAll(c.Request().Context(), objID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  "error",
			"message": "Failed to logout from all devices",
		})
	}

	// Clear cookies
	h.clearAuthCookies(c)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Successfully logged out from all devices",
	})
}

func (h *AuthHandler) GetProfile(c echo.Context) error {
	userID := c.Get("user_id").(string)
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "Invalid user ID",
		})
	}

	user, err := h.authService.GetUser(c.Request().Context(), objID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"status":  "error",
			"message": "User not found",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   user,
	})
}

// Cookie helper methods
func (h *AuthHandler) setRefreshTokenCookie(c echo.Context, token string) {
	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		MaxAge:   int((7 * 24 * time.Hour).Seconds()), // 7 days
		HttpOnly: true,
		Secure:   true, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
	}
	c.SetCookie(cookie)
}

func (h *AuthHandler) setAccessTokenCookie(c echo.Context, token string) {
	cookie := &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		MaxAge:   int((15 * time.Minute).Seconds()), // 15 minutes
		HttpOnly: true,
		Secure:   true, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
	}
	c.SetCookie(cookie)
}

func (h *AuthHandler) getRefreshTokenFromCookie(c echo.Context) string {
	cookie, err := c.Cookie("refresh_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (h *AuthHandler) getAccessTokenFromCookie(c echo.Context) string {
	cookie, err := c.Cookie("access_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (h *AuthHandler) clearAuthCookies(c echo.Context) {
	// Clear refresh token cookie
	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	c.SetCookie(refreshCookie)

	// Clear access token cookie
	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	c.SetCookie(accessCookie)
}
