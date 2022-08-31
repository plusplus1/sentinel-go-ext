package dashboard

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/plusplus1/sentinel-go-ext/dashboard/api/app"
	"github.com/plusplus1/sentinel-go-ext/dashboard/api/base"
)

func InstallApi(router gin.IRouter) {
	// Compute frontend path (new frontend in frontend/dist)
	frontendPath := resolveFrontendPath()
	// Compute dist directory path (dashboard/dist) for version file
	distPath := resolveDistPath()

	// Version from git short tag (set at build time)
	router.GET("/version", func(c *gin.Context) {
		c.String(200, Version)
	})

	// Serve favicon from frontend/dist/favicon.svg as /favicon.ico
	router.GET("/favicon.ico", func(c *gin.Context) {
		if frontendPath != "" {
			// Serve frontend's favicon.svg as favicon.ico (with proper content-type)
			c.File(filepath.Join(frontendPath, "favicon.svg"))
		} else {
			// No frontend, return empty favicon
			c.Data(200, "image/x-icon", []byte{})
		}
	})

	// Get auth service for middleware
	authService, err := app.GetAuthService()
	if err != nil {
		// Log error but continue - auth will fail for protected routes
		// This allows the app to start even if MySQL is down
		// Protected routes will return 401/500 when auth is needed
		_ = err // Suppress unused variable warning
		authService = nil
	}

	// API routes first
	apiGroup := router.Group("/api/")
	{
		// === Public routes (no authentication required) ===
		apiGroup.POST("/auth/login", app.Login)
		apiGroup.POST("/auth/logout", app.Logout)
		apiGroup.GET("/auth/feishu", app.FeishuLogin)
		apiGroup.GET("/auth/feishu/callback", app.FeishuCallback)

		// === Protected routes (authentication required) ===
		// Apply AuthMiddleware to all routes below
		protectedGroup := apiGroup.Group("", base.AuthMiddleware(authService))
		{
			// Auth & User Management API (protected)
			protectedGroup.GET("/auth/me", app.GetCurrentUser)

			// Rule APIs
			protectedGroup.GET("/app/rule/flow/list", app.ListFlowRules)
			protectedGroup.POST("/app/rule/flow/del", app.DeleteFlowRule)
			protectedGroup.POST("/app/rule/flow/update", app.SaveOrUpdateFlowRule)
			protectedGroup.GET("/app/rule/circuitbreaker/list", app.ListCircuitbreakerRules)
			protectedGroup.POST("/app/rule/circuitbreaker/del", app.DeleteCircuitbreakerRule)
			protectedGroup.POST("/app/rule/circuitbreaker/update", app.SaveOrUpdateCircuitbreakerRule)

			// Group management APIs
			protectedGroup.GET("/groups", app.ListGroups)
			protectedGroup.POST("/groups", app.CreateGroup)
			protectedGroup.GET("/groups/:id", app.GetGroup)
			protectedGroup.PUT("/groups/:id", app.UpdateGroup)
			protectedGroup.DELETE("/groups/:id", app.DeleteGroup)
			protectedGroup.GET("/groups/:id/members", app.ListGroupMembers)
			protectedGroup.POST("/groups/:id/members", app.AddResourceToGroup)
			protectedGroup.DELETE("/groups/:id/members/:resource", app.RemoveResourceFromGroup)

			// Resource APIs
			protectedGroup.GET("/resources", app.ListResources)
			protectedGroup.GET("/resource/:id/rules", app.GetResourceWithRules)
			protectedGroup.GET("/resource/:id/diff", app.GetResourceDiff)
			protectedGroup.GET("/resource/:id", app.GetResourceMetadata)
			protectedGroup.PUT("/resource/:id", app.UpdateResourceGroup)
			protectedGroup.PUT("/rule/:type/:id/toggle", app.ToggleRule)
			protectedGroup.DELETE("/resource", app.DeleteResource)
			protectedGroup.DELETE("/resource/:id", app.DeleteResource)

			// Publish API (MySQL → etcd)
			protectedGroup.POST("/publish", app.PublishRules)
			protectedGroup.GET("/publish/records", app.ListPublishRecords)

			// Version management API
			protectedGroup.GET("/versions", app.ListVersions)
			protectedGroup.GET("/versions/:id", app.GetVersion)
			protectedGroup.POST("/versions/:id/rollback", app.RollbackVersion)

			// === Super Admin routes (require super_admin role) ===
			superAdminGroup := protectedGroup.Group("/admin", base.RequireRoleMiddleware("super_admin"))
			{
				superAdminGroup.GET("/users", app.ListUsers)
				superAdminGroup.GET("/lines", app.ListBusinessLines)
				superAdminGroup.POST("/lines", app.CreateBusinessLine)
				superAdminGroup.PUT("/lines/:id", app.UpdateBusinessLine)
				superAdminGroup.DELETE("/lines/:id", app.DeleteBusinessLine)
				superAdminGroup.POST("/lines/:id/admins", app.AddBusinessLineAdmin)
				superAdminGroup.DELETE("/lines/:id/admins/:user_id", app.RemoveBusinessLineAdmin)
				superAdminGroup.GET("/audit-logs", app.ListAuditLogs)
			}

			// === User search (any authenticated user) ===
			protectedGroup.GET("/users/search", app.SearchUsers)

			// === Line Admin routes (require line_admin role) ===
			lineAdminGroup := protectedGroup.Group("/line-admin", base.RequireRoleMiddleware("line_admin"))
			{
				// Line admin can view their own business lines
				lineAdminGroup.GET("/lines", app.ListMyBusinessLines)
				// Line admin can update description of their own business lines
				lineAdminGroup.PUT("/lines/:id", app.UpdateMyBusinessLine)
				// Line admin can manage apps within their business lines
				lineAdminGroup.GET("/lines/:id/apps", app.ListBusinessLineApps)
				lineAdminGroup.POST("/lines/:id/apps", app.CreateBusinessLineApp)
				lineAdminGroup.PUT("/lines/:id/apps/:app_id", app.UpdateBusinessLineApp)
				lineAdminGroup.DELETE("/lines/:id/apps/:app_id", app.DeleteBusinessLineApp)
				// Line admin can manage members within their business lines
				lineAdminGroup.GET("/lines/:id/members", app.ListBusinessLineMembers)
				lineAdminGroup.POST("/lines/:id/members", app.AddBusinessLineMember)
				lineAdminGroup.DELETE("/lines/:id/members/:user_id", app.RemoveBusinessLineMember)
			}

			// === Permission routes (protected, but not admin-only) ===
			protectedGroup.POST("/permissions", app.GrantPermission)
			protectedGroup.GET("/permissions", app.ListPermissions)
			protectedGroup.DELETE("/permissions/:id", app.RevokePermission)
			protectedGroup.GET("/apps", app.ListApps)
			protectedGroup.POST("/apps", app.CreateApp)
			protectedGroup.PUT("/apps/:app_id", app.UpdateApp)
			protectedGroup.DELETE("/apps/:app_id", app.DeleteApp)
		}
	}

	// Redirect root to /web (frontend)
	router.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/web")
	})

	if frontendPath != "" {
		// Serve static files from frontendPath using http.FS (fsserver scheme)
		router.StaticFS("/web", http.Dir(frontendPath))
		// SPA fallback: serve index.html for all /web/* routes (non-API)
		if eg := router.(*gin.Engine); eg != nil {
			eg.NoRoute(func(c *gin.Context) {
				// Check if it's an API route
				if len(c.Request.URL.Path) >= 5 && c.Request.URL.Path[:5] == "/api/" {
					c.JSON(404, gin.H{"error": "Not Found"})
					return
				}
				// If it's under /web, serve index.html from frontendPath
				if len(c.Request.URL.Path) >= 5 && c.Request.URL.Path[:5] == "/web/" {
					c.File(filepath.Join(frontendPath, "index.html"))
					return
				}
				// Otherwise 404
				c.JSON(404, gin.H{"error": "Not Found"})
			})
		}
	} else {
		// No frontend found, return error
		router.GET("/web", func(c *gin.Context) {
			c.JSON(500, gin.H{"error": "Frontend not found. Please run 'npm run build' in frontend directory."})
		})
		// Also handle other routes
		if eg := router.(*gin.Engine); eg != nil {
			eg.NoRoute(func(c *gin.Context) {
				c.JSON(404, gin.H{"error": "Not Found"})
			})
		}
	}

	//debug - scan real dist directory (dashboard/dist)
	router.GET("/fs/", func(c *gin.Context) {
		scanRealFs(c, distPath)
	})

}

func resolveFrontendPath() string {
	possiblePaths := []string{
		"./frontend/dist",  // relative to CWD (project root)
		"../frontend/dist", // relative to CWD if in dashboard/
		"frontend/dist",    // relative to CWD (no dot)
	}

	// Also try relative to executable location
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		possiblePaths = append(possiblePaths,
			filepath.Join(execDir, "../frontend/dist"), // binary in dashboard/
			filepath.Join(execDir, "frontend/dist"),    // binary in project root
		)
	}

	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func resolveDistPath() string {
	possiblePaths := []string{
		"./dashboard/dist", // relative to CWD (project root)
		"dashboard/dist",   // relative to CWD (no dot)
		"dist",             // relative to CWD if already in dashboard/
	}

	// Also try relative to executable location
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		possiblePaths = append(possiblePaths,
			filepath.Join(execDir, "dashboard/dist"), // binary in project root
			filepath.Join(execDir, "dist"),           // binary in dashboard/
		)
	}

	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	return ""
}

func scanRealFs(c *gin.Context, distPath string) {
	var (
		ret     = gin.H{}
		items   []gin.H
		path, _ = c.GetQuery("p")
	)

	if path == `` {
		path = "/"
	}

	// Security: ensure path is within distPath
	fullPath := filepath.Join(distPath, path)
	absFullPath, _ := filepath.Abs(fullPath)
	absDistPath, _ := filepath.Abs(distPath)
	if !strings.HasPrefix(absFullPath, absDistPath) {
		c.JSON(400, gin.H{"error": "Invalid path"})
		return
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		c.JSONP(200, gin.H{
			"items": []gin.H{},
			"path":  path,
			"error": err.Error(),
		})
		return
	}

	for _, d := range entries {
		items = append(items, gin.H{
			`name`:   d.Name(),
			`is_dir`: d.IsDir(),
			`type`:   d.Type(),
		})
	}
	ret[`items`] = items
	ret[`path`] = path
	c.JSONP(200, ret)
}
