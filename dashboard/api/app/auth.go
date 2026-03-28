package app

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plusplus1/sentinel-go-ext/dashboard/config"
	"github.com/plusplus1/sentinel-go-ext/dashboard/dao"
	"github.com/plusplus1/sentinel-go-ext/dashboard/service"
)

type appResp struct {
	Code int         `json:"code"`
	Msg  string      `json:"message,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

var (
	authMutex   sync.Mutex
	authDB      *sql.DB
	authService *service.AuthService
)

func getAuthDB() (*sql.DB, error) {
	if authDB != nil {
		if err := authDB.Ping(); err == nil {
			return authDB, nil
		}
		authDB.Close()
	}

	// Load config and create MySQL connection
	cfg := config.AppSettings()
	dbCfg := dao.MySQLConfig{
		Host:     cfg.MySQL.Host,
		Port:     cfg.MySQL.Port,
		User:     cfg.MySQL.User,
		Password: cfg.MySQL.Password,
		Database: cfg.MySQL.Database,
	}
	db, err := dao.NewMySQLDB(dbCfg)
	if err != nil {
		log.Printf("Failed to connect to auth database: %v", err)
		return nil, err
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		log.Printf("Failed to ping auth database: %v", err)
		return nil, err
	}

	authDB = db
	log.Println("Database connection established successfully")
	return authDB, nil
}

// GetAuthService returns the auth service instance
func GetAuthService() (*service.AuthService, error) {
	// First check without lock for fast path
	if authService != nil {
		return authService, nil
	}

	// Acquire lock for initialization
	authMutex.Lock()
	defer authMutex.Unlock()

	// Double-check after acquiring lock
	if authService != nil {
		return authService, nil
	}

	db, err := getAuthDB()
	if err != nil {
		return nil, err
	}

	authService = service.NewAuthService(db)
	return authService, nil
}

// GetFeishuService returns the Feishu service instance
func GetFeishuService() (*service.FeishuService, error) {
	db, err := getAuthDB()
	if err != nil {
		return nil, err
	}

	// Load from config file first, then environment variables, then defaults
	appID := ""
	appSecret := ""
	redirectURI := ""

	// Try to load from config
	cfg := config.AppSettings()
	appID = cfg.Feishu.AppID
	appSecret = cfg.Feishu.AppSecret
	redirectURI = cfg.Feishu.RedirectURI

	// Fallback to environment variables
	if appID == "" {
		appID = os.Getenv("FEISHU_APP_ID")
	}
	if appSecret == "" {
		appSecret = os.Getenv("FEISHU_APP_SECRET")
	}
	if redirectURI == "" {
		redirectURI = os.Getenv("FEISHU_REDIRECT_URI")
	}

	// Final fallback to defaults for testing
	if appID == "" {
		appID = "cli_test_app_id"
	}
	if appSecret == "" {
		appSecret = "test_app_secret"
	}
	if redirectURI == "" {
		redirectURI = "http://localhost:6111/api/auth/feishu/callback"
	}

	return service.NewFeishuService(db, appID, appSecret, redirectURI), nil
}

// FeishuLogin redirects to Feishu OAuth
func FeishuLogin(c *gin.Context) {
	feishuService, err := GetFeishuService()
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "获取飞书服务失败"})
		return
	}

	// Generate state for CSRF protection
	state, err := generateState()
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "生成状态失败"})
		return
	}

	// Set state cookie (5 minute expiry)
	c.SetCookie("feishu_state", state, 300, "/", "", false, true)

	// Get OAuth URL and redirect
	oauthURL := feishuService.GetOAuthURL(state)
	c.Redirect(http.StatusTemporaryRedirect, oauthURL)
}

// FeishuCallback handles Feishu OAuth callback
func FeishuCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "缺少必要参数"})
		return
	}

	// Verify state to prevent CSRF
	cookieState, _ := c.Cookie("feishu_state")
	if cookieState != state {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "无效的状态参数"})
		return
	}

	// Clear state cookie
	c.SetCookie("feishu_state", "", -1, "/", "", false, true)

	feishuService, err := GetFeishuService()
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "获取飞书服务失败"})
		return
	}

	// Handle callback
	userID, userName, err := feishuService.HandleCallback(code, state)
	if err != nil {
		log.Printf("Feishu callback error: %v", err)
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "飞书登录失败: " + err.Error()})
		return
	}

	svc, err := GetAuthService()
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "获取认证服务失败"})
		return
	}

	token, err := svc.CreateSessionForUser(userID, 7*24*time.Hour)
	if err != nil {
		log.Printf("Error creating session: %v", err)
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "创建会话失败"})
		return
	}

	// Set session cookie (7 days)
	c.SetCookie("session_token", token, 7*24*60*60, "/", "", false, true)

	// Redirect to frontend
	c.Redirect(http.StatusTemporaryRedirect, "/web/login?feishu_success=true&user_id="+userID+"&name="+userName)
}

// generateState generates random state for OAuth
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Login handles user login
func Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "参数错误"})
		return
	}

	svc, err := GetAuthService()
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "获取认证服务失败"})
		return
	}

	user, token, err := svc.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "账号或密码错误"})
		return
	}

	// Set session cookie (24 hours)
	c.SetCookie("session_token", token, 24*60*60, "/", "", false, true)

	c.JSON(http.StatusOK, appResp{
		Data: map[string]interface{}{
			"token": token,
			"user": map[string]string{
				"user_id": user.UserID,
				"name":    user.Name,
				"email":   user.Email,
				"role":    user.Role,
			},
		},
	})
}

// Logout handles user logout
func Logout(c *gin.Context) {
	// Get token from cookie
	token, _ := c.Cookie("session_token")
	if token != "" {
		svc, err := GetAuthService()
		if err == nil {
			svc.DeleteSessionByToken(token)
		}
	}

	// Clear cookie
	c.SetCookie("session_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, appResp{Msg: "登出成功"})
}

// GetCurrentUser returns current authenticated user
func GetCurrentUser(c *gin.Context) {
	// Get token from cookie or header
	token, _ := c.Cookie("session_token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	if token == "" {
		c.JSON(http.StatusOK, appResp{Code: 401, Msg: "未登录"})
		return
	}

	// Get user from token
	svc, err := GetAuthService()
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "获取认证服务失败"})
		return
	}

	user, err := svc.ValidateSession(token)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 401, Msg: "无效的令牌"})
		return
	}

	c.JSON(http.StatusOK, appResp{
		Data: map[string]string{
			"user_id": user.UserID,
			"name":    user.Name,
			"email":   user.Email,
			"role":    user.Role,
		},
	})
}

// Helper function for generating random hex string
func generateHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// init initializes the auth module
func init() {
	// Log auth module initialization
	log.Println("Auth module initialized")
	fmt.Println("Auth module initialized")
}
