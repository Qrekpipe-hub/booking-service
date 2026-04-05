package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// apiError wraps the error shape defined in the OpenAPI spec.
func apiError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

func badRequest(c *gin.Context, message string) {
	apiError(c, http.StatusBadRequest, "INVALID_REQUEST", message)
}

func unauthorized(c *gin.Context, message string) {
	apiError(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func forbidden(c *gin.Context, message string) {
	apiError(c, http.StatusForbidden, "FORBIDDEN", message)
}

func notFound(c *gin.Context, code, message string) {
	apiError(c, http.StatusNotFound, code, message)
}

func conflict(c *gin.Context, code, message string) {
	apiError(c, http.StatusConflict, code, message)
}

func internalError(c *gin.Context) {
	apiError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
