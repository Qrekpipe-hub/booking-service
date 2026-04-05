package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/example/booking-service/internal/model"
	"github.com/example/booking-service/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// POST /dummyLogin
func (h *AuthHandler) DummyLogin(c *gin.Context) {
	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "role is required")
		return
	}

	role := model.Role(req.Role)
	if role != model.RoleAdmin && role != model.RoleUser {
		badRequest(c, "role must be 'admin' or 'user'")
		return
	}

	token, err := h.auth.DummyLogin(role)
	if err != nil {
		internalError(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// POST /register  (optional feature)
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email"    binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		Role     string `json:"role"     binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	role := model.Role(req.Role)
	if role != model.RoleAdmin && role != model.RoleUser {
		badRequest(c, "role must be 'admin' or 'user'")
		return
	}

	user, err := h.auth.Register(c.Request.Context(), req.Email, req.Password, role)
	if err != nil {
		if err == service.ErrEmailTaken {
			badRequest(c, "email already taken")
			return
		}
		internalError(c)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": user})
}

// POST /login  (optional feature)
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"    binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	token, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid credentials"},
			})
			return
		}
		internalError(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}
