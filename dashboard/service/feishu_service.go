package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FeishuService handles Feishu SSO OAuth flow
type FeishuService struct {
	db         *sql.DB
	appID      string
	appSecret  string
	redirectURI string
}

// NewFeishuService creates a new FeishuService
func NewFeishuService(db *sql.DB, appID, appSecret, redirectURI string) *FeishuService {
	return &FeishuService{
		db:         db,
		appID:      appID,
		appSecret:  appSecret,
		redirectURI: redirectURI,
	}
}

// GetOAuthURL generates the Feishu OAuth authorization URL
func (s *FeishuService) GetOAuthURL(state string) string {
	params := url.Values{}
	params.Set("app_id", s.appID)
	params.Set("redirect_uri", s.redirectURI)
	params.Set("state", state)
	params.Set("response_type", "code")
	return "https://accounts.feishu.cn/open-apis/authen/v1/authorize?" + params.Encode()
}

// HandleCallback handles the OAuth callback from Feishu
func (s *FeishuService) HandleCallback(code, state string) (string, string, error) {
	// State validation is performed in the API handler to prevent CSRF attacks
	// The state parameter is used to ensure the request came from our application

	// Exchange code for access token
	accessToken, err := s.exchangeCodeForToken(code)
	if err != nil {
		return "", "", fmt.Errorf("failed to exchange code for token: %v", err)
	}

	// Get user info from Feishu
	userInfo, err := s.getUserInfo(accessToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to get user info: %v", err)
	}

	// Create or update user in database
	userID, err := s.createOrUpdateUser(userInfo)
	if err != nil {
		return "", "", fmt.Errorf("failed to create/update user: %v", err)
	}

	return userID, userInfo.Name, nil
}

// exchangeCodeForToken exchanges authorization code for access token
func (s *FeishuService) exchangeCodeForToken(code string) (string, error) {
	tokenURL := "https://accounts.feishu.cn/open-apis/authen/v1/oidc/access_token"
	
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("app_id", s.appID)
	data.Set("app_secret", s.appSecret)
	
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	var tokenResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			ExpiresIn   int    `json:"expires_in"`
		} `json:"data"`
	}
	
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}
	
	if tokenResp.Code != 0 {
		return "", fmt.Errorf("token exchange failed: %s", tokenResp.Msg)
	}
	
	return tokenResp.Data.AccessToken, nil
}

// getUserInfo retrieves user information from Feishu API
func (s *FeishuService) getUserInfo(accessToken string) (*FeishuUserInfo, error) {
	userInfoURL := "https://open.feishu.cn/open-apis/authen/v1/user_info"
	
	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var userInfoResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			OpenID  string `json:"open_id"`
			UnionID string `json:"union_id"`
			UserID  string `json:"user_id"`
			Name    string `json:"name"`
			Email   string `json:"email"`
			Avatar  struct {
				Avatar72 string `json:"avatar_72"`
			} `json:"avatar"`
		} `json:"data"`
	}
	
	if err := json.Unmarshal(body, &userInfoResp); err != nil {
		return nil, err
	}
	
	if userInfoResp.Code != 0 {
		return nil, fmt.Errorf("get user info failed: %s", userInfoResp.Msg)
	}
	
	return &FeishuUserInfo{
		OpenID:  userInfoResp.Data.OpenID,
		UnionID: userInfoResp.Data.UnionID,
		Name:    userInfoResp.Data.Name,
		Email:   userInfoResp.Data.Email,
		Avatar:  userInfoResp.Data.Avatar.Avatar72,
	}, nil
}

// FeishuUserInfo contains user information from Feishu
type FeishuUserInfo struct {
	OpenID  string
	UnionID string
	Name    string
	Email   string
	Avatar  string
}

// createOrUpdateUser creates a new user or updates existing user based on Feishu info
func (s *FeishuService) createOrUpdateUser(userInfo *FeishuUserInfo) (string, error) {
	// Check if user already exists by Feishu OpenID
	var userID string
	var email string
	var name string
	var role string
	
	err := s.db.QueryRow(
		`SELECT user_id, email, name, role FROM users WHERE feishu_user_id = ?`,
		userInfo.OpenID,
	).Scan(&userID, &email, &name, &role)
	
	if err == nil {
		// User exists, update last login
		_, err = s.db.Exec(
			`UPDATE users SET last_login_at = NOW() WHERE user_id = ?`,
			userID,
		)
		if err != nil {
			log.Printf("Failed to update last login: %v", err)
		}
		return userID, nil
	}
	
	// User doesn't exist, create new user
	if err == sql.ErrNoRows {
		// Generate new user ID
		userID, err = generateUserID(s.db)
		if err != nil {
			return "", err
		}
		
		// Use Feishu email if available, otherwise generate placeholder
		email = userInfo.Email
		if email == "" {
			email = fmt.Sprintf("feishu_%s@feishu.local", userInfo.OpenID[:8])
		} else {
			// Check if email already exists
			var exists bool
			err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", email).Scan(&exists)
			if err != nil {
				return "", err
			}
			if exists {
				// Email already exists, generate a unique one
				email = fmt.Sprintf("feishu_%s@feishu.local", userInfo.OpenID[:8])
				log.Printf("Email %s already exists, using generated email: %s", userInfo.Email, email)
			}
		}
		
		// Default role is member
		role = "member"
		
		// Insert new user
		_, err = s.db.Exec(
			`INSERT INTO users (user_id, email, name, role, feishu_user_id, status, last_login_at)
			 VALUES (?, ?, ?, ?, ?, 'active', NOW())`,
			userID, email, userInfo.Name, role, userInfo.OpenID,
		)
		if err != nil {
			return "", err
		}
		
		log.Printf("Created new user from Feishu SSO: %s (%s)", userInfo.Name, email)
		return userID, nil
	}
	
	return "", err
}


