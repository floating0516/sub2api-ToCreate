package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

// AuthSubject is the minimal authenticated identity stored in gin context.
// AuthenticatedAt is the API JWT issue time used by interactive OIDC prompts.
type AuthSubject struct {
	UserID          int64
	Concurrency     int
	AuthenticatedAt time.Time
}

func GetAuthSubjectFromContext(c *gin.Context) (AuthSubject, bool) {
	value, exists := c.Get(string(ContextKeyUser))
	if !exists {
		return AuthSubject{}, false
	}
	subject, ok := value.(AuthSubject)
	return subject, ok
}

func GetUserRoleFromContext(c *gin.Context) (string, bool) {
	value, exists := c.Get(string(ContextKeyUserRole))
	if !exists {
		return "", false
	}
	role, ok := value.(string)
	return role, ok
}
