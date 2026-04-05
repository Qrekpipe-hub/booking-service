package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Qrekpipe-hub/booking-service/internal/model"
	"github.com/Qrekpipe-hub/booking-service/internal/service"
)

const (
	ContextUserID = "user_id"
	ContextRole   = "role"
)

// Auth validates the Bearer JWT and sets user_id + role in the Gin context.
func Auth(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errResp("UNAUTHORIZED", "missing or invalid Authorization header"))
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		claims, err := authSvc.ParseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errResp("UNAUTHORIZED", "invalid token"))
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errResp("UNAUTHORIZED", "invalid user_id in token"))
			return
		}

		c.Set(ContextUserID, userID)
		c.Set(ContextRole, model.Role(claims.Role))
		c.Next()
	}
}

// RequireRole aborts with 403 if the authenticated user does not have one of the given roles.
func RequireRole(roles ...model.Role) gin.HandlerFunc {
	allowed := make(map[model.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		role, _ := c.Get(ContextRole)
		if !allowed[role.(model.Role)] {
			c.AbortWithStatusJSON(http.StatusForbidden, errResp("FORBIDDEN", "access denied"))
			return
		}
		c.Next()
	}
}

// GetUserID extracts the authenticated user's UUID from the Gin context.
func GetUserID(c *gin.Context) uuid.UUID {
	v, _ := c.Get(ContextUserID)
	return v.(uuid.UUID)
}

// GetRole extracts the authenticated user's role from the Gin context.
func GetRole(c *gin.Context) model.Role {
	v, _ := c.Get(ContextRole)
	return v.(model.Role)
}

func errResp(code, message string) gin.H {
	return gin.H{"error": gin.H{"code": code, "message": message}}
}
