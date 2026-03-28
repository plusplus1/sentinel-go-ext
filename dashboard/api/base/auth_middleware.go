package base

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/plusplus1/sentinel-go-ext/dashboard/model"
	"github.com/plusplus1/sentinel-go-ext/dashboard/service"
)

// AuthMiddleware is a middleware to authenticate users
func AuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session token from cookie
		token, err := c.Cookie("session_token")
		if err != nil || token == "" {
			c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未登录"})
			c.Abort()
			return
		}

		// Validate session token
		user, err := authService.ValidateSession(token)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 401, "msg": err.Error()})
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("user", user)
		c.Next()
	}
}

// RequireRoleMiddleware is a middleware to check if user has the required role
func RequireRoleMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未登录"})
			c.Abort()
			return
		}

		// Check if user has the required role
		userRole := user.(*model.User).Role
		for _, role := range roles {
			if userRole == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "权限不足"})
		c.Abort()
	}
}

// RequirePermissionMiddleware is a middleware to check if user has permission on a resource
func RequirePermissionMiddleware(authService *service.AuthService, resourceType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "未登录"})
			c.Abort()
			return
		}

		// Get user ID and resource ID
		userID := user.(*model.User).UserID
		resourceID := c.Param("id")

		// Check if user has permission
		hasPermission, err := authService.CheckPermission(userID, resourceType, resourceID)
		if err != nil || !hasPermission {
			c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "无权访问该资源"})
			c.Abort()
			return
		}

		c.Next()
	}
}
