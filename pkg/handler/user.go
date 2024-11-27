package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Signup create a user
func (h *Handler) Signup(c *gin.Context) {
	var req signupRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, req)
}

func (h *Handler) GetUserFromCookie(c *gin.Context) (string, error) {
	cookie, err := c.Request.Cookie("user_id")
	if err != nil {
		if err == http.ErrNoCookie {
			return "", err
		}
		return "", err
	}
	return cookie.Value, nil
}

func (h *Handler) SetUserCookie(c *gin.Context) {
	userID := uuid.New().String()
	cookie := &http.Cookie{
		Name:     "user_id",
		Value:    userID,
		Expires:  time.Now().AddDate(5, 0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(c.Writer, cookie)
}
