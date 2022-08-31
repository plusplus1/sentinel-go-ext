package app

import (
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/plusplus1/sentinel-go-ext/dashboard/model"
)

// === Join table helper functions ===

// isLineAdmin checks if a user is an admin of a business line
func isLineAdmin(userID, lineID string) (bool, error) {
	if _, err := getAuthDB(); err != nil {
		return false, err
	}
	var count int
	err := authDB.QueryRow("SELECT COUNT(*) FROM business_line_admins WHERE user_id = (SELECT id FROM users WHERE user_id = ?) AND business_line_id = ?", userID, lineID).Scan(&count)
	return count > 0, err
}

// isLineMember checks if a user is a member of a business line
func isLineMember(userID, lineID string) (bool, error) {
	if _, err := getAuthDB(); err != nil {
		return false, err
	}
	var count int
	err := authDB.QueryRow("SELECT COUNT(*) FROM business_line_members WHERE user_id = (SELECT id FROM users WHERE user_id = ?) AND business_line_id = ?", userID, lineID).Scan(&count)
	return count > 0, err
}

// hasLineAccess checks if user has any access to a business line (admin or member)
func hasLineAccess(userID, lineID string) (bool, error) {
	isAdmin, err := isLineAdmin(userID, lineID)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return true, nil
	}
	return isLineMember(userID, lineID)
}

// getAdminLineIDs returns all business line IDs where user is admin
func getAdminLineIDs(userID string) ([]string, error) {
	if _, err := getAuthDB(); err != nil {
		return nil, err
	}
	rows, err := authDB.Query("SELECT business_line_id FROM business_line_admins WHERE user_id = (SELECT id FROM users WHERE user_id = ?)", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, nil
}

// getMemberLineIDs returns all business line IDs where user is member
func getMemberLineIDs(userID string) ([]string, error) {
	if _, err := getAuthDB(); err != nil {
		return nil, err
	}
	rows, err := authDB.Query("SELECT business_line_id FROM business_line_members WHERE user_id = (SELECT id FROM users WHERE user_id = ?)", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, nil
}

// === 用户管理 API（仅超级管理员）===

// ListUsers returns all users
func ListUsers(c *gin.Context) {
	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	rows, err := authDB.Query("SELECT user_id, name, email, role, status, created_at, updated_at FROM users ORDER BY created_at DESC")
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	defer rows.Close()

	var users []map[string]string
	for rows.Next() {
		var userID, name, email, role, status, createdAt, updatedAt string
		rows.Scan(&userID, &name, &email, &role, &status, &createdAt, &updatedAt)
		users = append(users, map[string]string{
			"user_id": userID, "name": name, "email": email,
			"role": role, "status": status,
			"created_at": createdAt, "updated_at": updatedAt,
		})
	}

	c.JSON(http.StatusOK, appResp{Data: users})
}

// CreateUser creates a new user (super admin only)
func CreateUser(c *gin.Context) {
	// Check permission: only super admin
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusOK, appResp{Code: 401, Msg: "未登录"})
		return
	}
	user := currentUser.(*model.User)
	if user.Role != "super_admin" {
		c.JSON(http.StatusOK, appResp{Code: 403, Msg: "仅超级管理员可创建用户"})
		return
	}

	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "参数错误"})
		return
	}

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	// Check if email already exists
	var count int
	authDB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", req.Email).Scan(&count)
	if count > 0 {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "邮箱已存在"})
		return
	}

	// Generate user_id
	userID := fmt.Sprintf("u_%d", len(req.Email)+len(req.Name))

	// Insert user
	_, err := authDB.Exec("INSERT INTO users (user_id, name, email, password, role, status) VALUES (?, ?, ?, ?, ?, 'active')",
		userID, req.Name, req.Email, req.Password, req.Role)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, appResp{Data: map[string]string{"user_id": userID}})
}

// SearchUsers searches users by name or email (super admin only)
func SearchUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "关键词不能为空"})
		return
	}

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	rows, err := authDB.Query(`
		SELECT user_id, name, email 
		FROM users 
		WHERE (name LIKE ? OR email LIKE ?) AND status = 'active'
		LIMIT 20`, "%"+keyword+"%", "%"+keyword+"%")
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	defer rows.Close()

	var users []map[string]string
	for rows.Next() {
		var userID, name, email string
		rows.Scan(&userID, &name, &email)
		users = append(users, map[string]string{
			"user_id": userID, "name": name, "email": email,
		})
	}

	c.JSON(http.StatusOK, appResp{Data: users})
}

// === 业务线管理 API（超级管理员 + 业务线管理员）===

// ListBusinessLines returns all business lines (super admin only)
func ListBusinessLines(c *gin.Context) {
	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	// Get all business lines
	rows, err := authDB.Query(`SELECT 
		bl.id, bl.name, COALESCE(bl.description,''), bl.status, 
		COALESCE(bl.owner_id,''), bl.updated_at 
	FROM business_lines bl 
	ORDER BY bl.updated_at DESC`)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	defer rows.Close()

	type lineInfo struct {
		ID, Name, Desc, Status, OwnerID, UpdatedAt string
	}
	var lineList []lineInfo
	lineMap := make(map[string]*lineInfo)
	for rows.Next() {
		var li lineInfo
		rows.Scan(&li.ID, &li.Name, &li.Desc, &li.Status, &li.OwnerID, &li.UpdatedAt)
		lineList = append(lineList, li)
		lineMap[li.ID] = &lineList[len(lineList)-1]
	}

	// Get all admins from business_line_admins JOIN users
	adminRows, err := authDB.Query(`SELECT 
		blad.business_line_id, u.user_id, COALESCE(u.name,''), COALESCE(u.email,''), COALESCE(u.status,'')
	FROM business_line_admins blad
	JOIN users u ON blad.user_id = u.id`)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	defer adminRows.Close()

	// Build admin map: lineID -> []admin
	adminMap := make(map[string][]map[string]string)
	for adminRows.Next() {
		var lineID, userID, userName, userEmail, userStatus string
		adminRows.Scan(&lineID, &userID, &userName, &userEmail, &userStatus)
		adminMap[lineID] = append(adminMap[lineID], map[string]string{
			"user_id": userID, "user_name": userName, "user_email": userEmail, "user_status": userStatus,
		})
	}

	// Build response
	var lines []map[string]interface{}
	for _, li := range lineList {
		admins := adminMap[li.ID]
		if admins == nil {
			admins = []map[string]string{}
		}
		lines = append(lines, map[string]interface{}{
			"id": li.ID, "name": li.Name, "description": li.Desc, "status": li.Status,
			"owner_id": li.OwnerID, "admins": admins,
			"updated_at": li.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, appResp{Data: lines})
}

// CreateBusinessLine creates a new business line (super admin only)
func CreateBusinessLine(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		OwnerID     string `json:"owner_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "参数错误"})
		return
	}

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	// Insert business line
	var result sql.Result
	var err error
	if req.OwnerID != "" {
		result, err = authDB.Exec("INSERT INTO business_lines (name, description, owner_id, status) VALUES (?, ?, ?, 'active')",
			req.Name, req.Description, req.OwnerID)
	} else {
		result, err = authDB.Exec("INSERT INTO business_lines (name, description, status) VALUES (?, ?, 'active')",
			req.Name, req.Description)
	}
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线名称已存在"})
			return
		}
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	lineID, _ := result.LastInsertId()

	// If owner specified, update user role to line_admin
	if req.OwnerID != "" {
		_, err = authDB.Exec("UPDATE users SET role = 'line_admin', updated_at = NOW() WHERE user_id = ?", req.OwnerID)
		if err != nil {
			fmt.Printf("Error updating user role: %v\n", err)
		}
	}

	// Create default app under this business line
	_, err = authDB.Exec("INSERT INTO business_line_apps (business_line_id, app_key, description, status) VALUES (?, 'default', '默认应用', 'active')", lineID)
	if err != nil {
		fmt.Printf("Error creating default app for business line %d: %v\n", lineID, err)
	}

	c.JSON(http.StatusOK, appResp{Data: map[string]interface{}{
		"id": lineID,
	}})
}

// UpdateBusinessLine updates business line description/status (super admin only)
func UpdateBusinessLine(c *gin.Context) {
	// Get user from context (already authenticated and authorized by middleware)
	lineID := c.Param("id")

	// Check if business line exists
	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	var businessLine struct {
		ID     string
		Name   string
		Status string
	}
	err := authDB.QueryRow("SELECT id, name, status FROM business_lines WHERE id = ?", lineID).Scan(&businessLine.ID, &businessLine.Name, &businessLine.Status)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	var req struct {
		Description string `json:"description"`
		Status      string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "参数错误"})
		return
	}

	// Dynamic update: only update fields that are provided
	setClauses := []string{}
	args := []interface{}{}

	if req.Description != "" {
		setClauses = append(setClauses, "description = ?")
		args = append(args, req.Description)
	}
	if req.Status != "" {
		setClauses = append(setClauses, "status = ?")
		args = append(args, req.Status)
	}

	if len(setClauses) == 0 {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "没有需要更新的字段"})
		return
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, lineID)

	query := fmt.Sprintf("UPDATE business_lines SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err = authDB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, appResp{Msg: "业务线更新成功"})
}

// DeleteBusinessLine deletes a business line (super admin only, soft delete)
func DeleteBusinessLine(c *gin.Context) {
	lineID := c.Param("id")

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	// Check if business line exists
	var status string
	err := authDB.QueryRow("SELECT status FROM business_lines WHERE id = ?", lineID).Scan(&status)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	// Soft delete: set status to deleted
	_, err = authDB.Exec("UPDATE business_lines SET status = 'deleted', updated_at = NOW() WHERE id = ?", lineID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, appResp{Msg: "业务线删除成功"})
}

// AddBusinessLineAdmin adds a user as business line admin (super admin only)
func AddBusinessLineAdmin(c *gin.Context) {
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusOK, appResp{Code: 401, Msg: "未登录"})
		return
	}
	user := currentUser.(*model.User)

	lineID := c.Param("id")
	if lineID == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线ID不能为空"})
		return
	}

	var req struct {
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "参数错误"})
		return
	}

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	// Check if business line exists and is active
	var lineStatus string
	err := authDB.QueryRow("SELECT status FROM business_lines WHERE id = ?", lineID).Scan(&lineStatus)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	if lineStatus != "active" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线已下线，无法绑定管理员"})
		return
	}

	// Check if user exists and is active
	var userStatus string
	err = authDB.QueryRow("SELECT status FROM users WHERE user_id = ?", req.UserID).Scan(&userStatus)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "用户不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	if userStatus != "active" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "用户已禁用"})
		return
	}

	// Check if already admin (duplicate)
	alreadyAdmin, err := isLineAdmin(req.UserID, lineID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	if alreadyAdmin {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "用户已是该业务线管理员"})
		return
	}

	// Insert into business_line_admins
	_, err = authDB.Exec("INSERT INTO business_line_admins (business_line_id, user_id, added_by) VALUES (?, (SELECT id FROM users WHERE user_id = ?), (SELECT id FROM users WHERE user_id = ?))",
		lineID, req.UserID, user.UserID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	// Update user role to line_admin
	_, err = authDB.Exec("UPDATE users SET role = 'line_admin', updated_at = NOW() WHERE user_id = ?", req.UserID)
	if err != nil {
		fmt.Printf("Error updating user role: %v\n", err)
	}

	// Keep owner_id column updated for backward compat (first admin)
	var ownerID string
	authDB.QueryRow("SELECT COALESCE(owner_id, '') FROM business_lines WHERE id = ?", lineID).Scan(&ownerID)
	if ownerID == "" {
		authDB.Exec("UPDATE business_lines SET owner_id = ?, updated_at = NOW() WHERE id = ?", req.UserID, lineID)
	}

	c.JSON(http.StatusOK, appResp{Msg: "管理员添加成功"})
}

// RemoveBusinessLineAdmin removes a user from business line admins (super admin only)
func RemoveBusinessLineAdmin(c *gin.Context) {
	lineID := c.Param("id")
	userID := c.Param("user_id")
	if lineID == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线ID不能为空"})
		return
	}
	if userID == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "用户ID不能为空"})
		return
	}

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	// Check if user is admin of this business line
	isAdmin, err := isLineAdmin(userID, lineID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	if !isAdmin {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "用户不是该业务线管理员"})
		return
	}

	_, err = authDB.Exec("DELETE FROM business_line_admins WHERE business_line_id = ? AND user_id = (SELECT id FROM users WHERE user_id = ?)", lineID, userID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	// Check if user still has other business lines as admin
	remainingIDs, err := getAdminLineIDs(userID)
	if err != nil {
		fmt.Printf("Error checking other business lines: %v\n", err)
	}
	if len(remainingIDs) == 0 {
		// User has no other business lines, revert role to member
		_, err = authDB.Exec("UPDATE users SET role = 'member', updated_at = NOW() WHERE user_id = ?", userID)
		if err != nil {
			fmt.Printf("Error reverting user role: %v\n", err)
		}
	}

	c.JSON(http.StatusOK, appResp{Msg: "管理员移除成功"})
}

// === 应用管理 API（超级管理员 + 业务线管理员）===

// ListApps returns apps (super admin sees all, line admin sees own business line apps)
func ListApps(c *gin.Context) {
	lineID := c.Query("line_id")

	// Get current user
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusOK, appResp{Code: 401, Msg: "未登录"})
		return
	}
	user := currentUser.(*model.User)

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	// Build query based on user role
	var query string
	var args []interface{}
	var result []map[string]interface{}

	if user.Role == "super_admin" {
		// Super admin sees all apps, organized by business lines
		query = `
			SELECT 
				bl.id as line_id, bl.name as line_name, COALESCE(bl.description, '') as line_desc,
				bla.id as app_id, bla.app_key as app_name, COALESCE(bla.description,'') as app_desc, bla.status as app_status
			FROM business_line_apps bla
			JOIN business_lines bl ON bla.business_line_id = bl.id
			WHERE bl.status = 'active'`
		if lineID != "" {
			query += " AND bl.id = ?"
			args = append(args, lineID)
		}
		query += " ORDER BY bl.name, bla.app_key"
	} else if user.Role == "line_admin" {
		query = `
			SELECT 
				bl.id as line_id, bl.name as line_name, COALESCE(bl.description, '') as line_desc,
				bla.id as app_id, bla.app_key as app_name, COALESCE(bla.description,'') as app_desc, bla.status as app_status
			FROM business_line_apps bla
			JOIN business_lines bl ON bla.business_line_id = bl.id
			WHERE bl.status = 'active' AND bl.id IN (SELECT business_line_id FROM business_line_admins WHERE user_id = (SELECT id FROM users WHERE user_id = ?))`
		args = append(args, user.UserID)
		if lineID != "" {
			query += " AND bl.id = ?"
			args = append(args, lineID)
		}
		query += " ORDER BY bl.name, bla.app_key"
	} else {
		query = `
			SELECT 
				bl.id as line_id, bl.name as line_name, COALESCE(bl.description, '') as line_desc,
				bla.id as app_id, bla.app_key as app_name, COALESCE(bla.description,'') as app_desc, bla.status as app_status
			FROM business_line_members blm
			JOIN business_lines bl ON blm.business_line_id = bl.id
			JOIN business_line_apps bla ON bla.business_line_id = bl.id
			WHERE blm.user_id = (SELECT id FROM users WHERE user_id = ?) AND bl.status = 'active' AND bla.status = 'active'`
		args = append(args, user.UserID)
		if lineID != "" {
			query += " AND bl.id = ?"
			args = append(args, lineID)
		}
		query += " ORDER BY bl.name, bla.app_key"
	}

	rows, err := authDB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	defer rows.Close()

	// Organize by business line
	lineMap := make(map[string]map[string]interface{})
	for rows.Next() {
		var lineID, lineName, lineDesc, appID, appName, appDesc, appStatus string
		rows.Scan(&lineID, &lineName, &lineDesc, &appID, &appName, &appDesc, &appStatus)

		if _, ok := lineMap[lineID]; !ok {
			lineMap[lineID] = map[string]interface{}{
				"id":          lineID,
				"name":        lineName,
				"description": lineDesc,
				"children":    []map[string]interface{}{},
			}
		}

		apps := lineMap[lineID]["children"].([]map[string]interface{})
		lineMap[lineID]["children"] = append(apps, map[string]interface{}{
			"id":          appID,
			"name":        appName,
			"description": appDesc,
			"status":      appStatus,
		})
	}

	// Convert map to slice
	for _, v := range lineMap {
		result = append(result, v)
	}

	c.JSON(http.StatusOK, appResp{Data: result})
}

// CreateApp creates a new app (super admin only, or line admin within their business line)
func CreateApp(c *gin.Context) {
	// Check permission
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusOK, appResp{Code: 401, Msg: "未登录"})
		return
	}
	user := currentUser.(*model.User)

	var req struct {
		AppKey      string `json:"app_key"`     // 英文、数字、下划线
		Description string `json:"description"` // 中文描述
		LineID      string `json:"line_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "参数错误"})
		return
	}

	// Validate app_key: only English, numbers, underscores
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]{3,50}$`, req.AppKey); !matched {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "应用标识只能包含英文、数字、下划线，长度3-50字符"})
		return
	}

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	// For line admin, verify they are admin of the business line
	if user.Role == "line_admin" {
		if req.LineID == "" {
			c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线ID不能为空"})
			return
		}
		isAdmin, err := isLineAdmin(user.UserID, req.LineID)
		if err != nil || !isAdmin {
			c.JSON(http.StatusOK, appResp{Code: 403, Msg: "无权限在此业务线下创建应用"})
			return
		}
	}

	// LineID is required
	if req.LineID == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线ID不能为空"})
		return
	}

	// Insert app directly into business_line_apps
	result, err := authDB.Exec("INSERT INTO business_line_apps (business_line_id, app_key, description, status) VALUES (?, ?, ?, 'active')",
		req.LineID, req.AppKey, req.Description)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			c.JSON(http.StatusOK, appResp{Code: 100, Msg: "应用标识已存在"})
			return
		}
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	appID, _ := result.LastInsertId()

	c.JSON(http.StatusOK, appResp{Data: map[string]interface{}{
		"id": appID,
	}})
}

// UpdateApp updates an app (super admin or line admin for own business line)
func UpdateApp(c *gin.Context) {
	appID := c.Param("app_id")

	// Check permission
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusOK, appResp{Code: 401, Msg: "未登录"})
		return
	}
	user := currentUser.(*model.User)

	var req struct {
		AppKey      string `json:"app_key"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "参数错误"})
		return
	}

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	// Check if app exists and get business_line_id for permission check
	var businessLineID int
	err := authDB.QueryRow("SELECT business_line_id FROM business_line_apps WHERE id = ?", appID).Scan(&businessLineID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "应用不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	// For line admin, verify they are admin of the business line
	if user.Role == "line_admin" {
		isAdmin, err := isLineAdmin(user.UserID, fmt.Sprintf("%d", businessLineID))
		if err != nil || !isAdmin {
			c.JSON(http.StatusOK, appResp{Code: 403, Msg: "无权限修改此应用"})
			return
		}
	}

	// Dynamic update
	setClauses := []string{}
	args := []interface{}{}

	if req.AppKey != "" {
		// Validate app_key
		if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]{3,50}$`, req.AppKey); !matched {
			c.JSON(http.StatusOK, appResp{Code: 100, Msg: "应用标识只能包含英文、数字、下划线，长度3-50字符"})
			return
		}
		setClauses = append(setClauses, "app_key = ?")
		args = append(args, req.AppKey)
	}
	if req.Description != "" {
		setClauses = append(setClauses, "description = ?")
		args = append(args, req.Description)
	}
	if req.Status != "" {
		setClauses = append(setClauses, "status = ?")
		args = append(args, req.Status)
	}

	if len(setClauses) == 0 {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "没有需要更新的字段"})
		return
	}

	args = append(args, appID)
	query := fmt.Sprintf("UPDATE business_line_apps SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err = authDB.Exec(query, args...)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			c.JSON(http.StatusOK, appResp{Code: 100, Msg: "应用标识已存在"})
			return
		}
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, appResp{Msg: "应用更新成功"})
}

// DeleteApp deletes an app (super admin only, soft delete)
func DeleteApp(c *gin.Context) {
	appID := c.Param("app_id")

	// Check permission: only super admin
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusOK, appResp{Code: 401, Msg: "未登录"})
		return
	}
	user := currentUser.(*model.User)
	if user.Role != "super_admin" {
		c.JSON(http.StatusOK, appResp{Code: 403, Msg: "仅超级管理员可删除应用"})
		return
	}

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	// Soft delete
	_, err := authDB.Exec("UPDATE business_line_apps SET status = 'deleted' WHERE id = ?", appID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, appResp{Msg: "应用删除成功"})
}

// === 业务线应用关联 API（超级管理员）===

// ListBusinessLineApps returns apps associated with a business line
func ListBusinessLineApps(c *gin.Context) {
	lineID := c.Param("id")
	if lineID == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线ID不能为空"})
		return
	}

	// Check permission: only super admin or line admin of this business line
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusOK, appResp{Code: 401, Msg: "未登录"})
		return
	}
	user := currentUser.(*model.User)
	if user.Role != "super_admin" {
		isAdmin, err := isLineAdmin(user.UserID, lineID)
		if err != nil || !isAdmin {
			c.JSON(http.StatusOK, appResp{Code: 403, Msg: "无权限查看此业务线的应用"})
			return
		}
	}

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	rows, err := authDB.Query(`
		SELECT 
			bla.id, bla.app_key, COALESCE(bla.description,''), COALESCE(bla.settings,''), bla.status, bla.created_at 
		FROM business_line_apps bla
		WHERE bla.business_line_id = ?
		ORDER BY bla.created_at DESC`, lineID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	defer rows.Close()

	var apps []map[string]interface{}
	for rows.Next() {
		var id int
		var appKey, description, settings, status, createdAt string
		rows.Scan(&id, &appKey, &description, &settings, &status, &createdAt)
		apps = append(apps, map[string]interface{}{
			"id": id, "app_key": appKey,
			"description": description, "settings": settings, "status": status, "created_at": createdAt,
		})
	}

	c.JSON(http.StatusOK, appResp{Data: apps})
}

// AssociateAppWithBusinessLine creates a new app and associates it with a business line
func AssociateAppWithBusinessLine(c *gin.Context) {
	lineID := c.Param("id")
	if lineID == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线ID不能为空"})
		return
	}

	// Check permission: only super admin or line admin of this business line
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusOK, appResp{Code: 401, Msg: "未登录"})
		return
	}
	user := currentUser.(*model.User)
	if user.Role != "super_admin" {
		isAdmin, err := isLineAdmin(user.UserID, lineID)
		if err != nil || !isAdmin {
			c.JSON(http.StatusOK, appResp{Code: 403, Msg: "无权限操作此业务线的应用"})
			return
		}
	}

	var req struct {
		AppKey      string `json:"app_key"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "参数错误"})
		return
	}

	// Validate app_key
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]{3,50}$`, req.AppKey); !matched {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "应用标识只能包含英文、数字、下划线，长度3-50字符"})
		return
	}

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	// Check if business line exists and is active
	var lineStatus string
	err := authDB.QueryRow("SELECT status FROM business_lines WHERE id = ?", lineID).Scan(&lineStatus)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	if lineStatus != "active" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线已下线，无法创建应用"})
		return
	}

	// Insert app directly into business_line_apps
	result, err := authDB.Exec("INSERT INTO business_line_apps (business_line_id, app_key, description, status) VALUES (?, ?, ?, 'active')",
		lineID, req.AppKey, req.Description)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			c.JSON(http.StatusOK, appResp{Code: 100, Msg: "应用标识已存在"})
			return
		}
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	appID, _ := result.LastInsertId()
	c.JSON(http.StatusOK, appResp{Msg: "应用创建成功", Data: map[string]interface{}{"id": appID}})
}

// DisassociateAppFromBusinessLine removes an app from a business line
func DisassociateAppFromBusinessLine(c *gin.Context) {
	lineID := c.Param("id")
	appID := c.Param("app_id")
	if lineID == "" || appID == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线ID和应用ID不能为空"})
		return
	}

	// Check permission: only super admin or line admin of this business line
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusOK, appResp{Code: 401, Msg: "未登录"})
		return
	}
	user := currentUser.(*model.User)
	if user.Role != "super_admin" {
		isAdmin, err := isLineAdmin(user.UserID, lineID)
		if err != nil || !isAdmin {
			c.JSON(http.StatusOK, appResp{Code: 403, Msg: "无权限操作此业务线的应用"})
			return
		}
	}

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	// Delete app from business_line_apps
	_, err := authDB.Exec("DELETE FROM business_line_apps WHERE id = ? AND business_line_id = ?", appID, lineID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, appResp{Msg: "应用关联已解除"})
}

// === 权限管理 API（超级管理员）===

// GrantPermission grants permission to user (super admin or line admin)
func GrantPermission(c *gin.Context) {
	// Check permission
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusOK, appResp{Code: 401, Msg: "未登录"})
		return
	}
	user := currentUser.(*model.User)

	var req struct {
		UserID       string `json:"user_id"`
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
		Role         string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "参数错误"})
		return
	}

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	// For line admin, verify they have permission to grant access to this resource
	if user.Role == "line_admin" {
		// Line admin can only grant permissions for apps within their business lines
		if req.ResourceType != "app" {
			c.JSON(http.StatusOK, appResp{Code: 403, Msg: "业务线管理员只能授权应用权限"})
			return
		}
		// Check if app is in user's business line
		var count int
		err := authDB.QueryRow(`
			SELECT COUNT(*) FROM business_line_apps bla
			JOIN business_line_admins blad ON bla.business_line_id = blad.business_line_id
			WHERE blad.user_id = (SELECT id FROM users WHERE user_id = ?) AND bla.app_id = ?`, user.UserID, req.ResourceID).Scan(&count)
		if err != nil || count == 0 {
			c.JSON(http.StatusOK, appResp{Code: 403, Msg: "无权限授权此应用"})
			return
		}
	}

	// Insert permission
	_, err := authDB.Exec("INSERT INTO user_permissions (user_id, resource_type, resource_id, role, granted_by) VALUES (?, ?, ?, ?, ?)",
		req.UserID, req.ResourceType, req.ResourceID, req.Role, user.UserID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, appResp{Msg: "授权成功"})
}

// ListPermissions returns permissions list (super admin sees all, line admin sees own business line)
func ListPermissions(c *gin.Context) {
	// Get current user
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusOK, appResp{Code: 401, Msg: "未登录"})
		return
	}
	user := currentUser.(*model.User)

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	var query string
	var args []interface{}

	if user.Role == "super_admin" {
		// Super admin sees all permissions
		query = `
			SELECT 
				p.id, p.user_id, COALESCE(u.name,''), COALESCE(u.email,''),
				p.resource_type, p.resource_id, p.role,
				COALESCE(p.granted_by,''), p.created_at
			FROM user_permissions p
			LEFT JOIN users u ON p.user_id = u.user_id
			ORDER BY p.created_at DESC`
	} else if user.Role == "line_admin" {
		query = `
			SELECT 
				p.id, p.user_id, COALESCE(u.name,''), COALESCE(u.email,''),
				p.resource_type, p.resource_id, p.role,
				COALESCE(p.granted_by,''), p.created_at
			FROM user_permissions p
			LEFT JOIN users u ON p.user_id = u.user_id
			WHERE p.resource_type = 'app' AND p.resource_id IN (
				SELECT bla.app_id FROM business_line_apps bla
				JOIN business_line_admins blad ON bla.business_line_id = blad.business_line_id
				WHERE blad.user_id = (SELECT id FROM users WHERE user_id = ?)
			)
			ORDER BY p.created_at DESC`
		args = append(args, user.UserID)
	} else {
		// Regular user sees only their own permissions
		query = `
			SELECT 
				p.id, p.user_id, COALESCE(u.name,''), COALESCE(u.email,''),
				p.resource_type, p.resource_id, p.role,
				COALESCE(p.granted_by,''), p.created_at
			FROM user_permissions p
			LEFT JOIN users u ON p.user_id = u.user_id
			WHERE p.user_id = ?
			ORDER BY p.created_at DESC`
		args = append(args, user.UserID)
	}

	rows, err := authDB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	defer rows.Close()

	var permissions []map[string]string
	for rows.Next() {
		var id, userID, userName, userEmail, resourceType, resourceID, role, grantedBy, createdAt string
		rows.Scan(&id, &userID, &userName, &userEmail, &resourceType, &resourceID, &role, &grantedBy, &createdAt)
		permissions = append(permissions, map[string]string{
			"id": id, "user_id": userID, "user_name": userName, "user_email": userEmail,
			"resource_type": resourceType, "resource_id": resourceID, "role": role,
			"granted_by": grantedBy, "created_at": createdAt,
		})
	}

	c.JSON(http.StatusOK, appResp{Data: permissions})
}

// RevokePermission revokes permission (super admin or line admin for own business line)
func RevokePermission(c *gin.Context) {
	permissionID := c.Param("id")

	// Check permission
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusOK, appResp{Code: 401, Msg: "未登录"})
		return
	}
	user := currentUser.(*model.User)

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	// For line admin, check if permission is for app in their business line
	if user.Role == "line_admin" {
		var resourceType, resourceID string
		err := authDB.QueryRow("SELECT resource_type, resource_id FROM user_permissions WHERE id = ?", permissionID).Scan(&resourceType, &resourceID)
		if err != nil {
			c.JSON(http.StatusOK, appResp{Code: 100, Msg: "权限不存在"})
			return
		}

		if resourceType != "app" {
			c.JSON(http.StatusOK, appResp{Code: 403, Msg: "业务线管理员只能撤销应用权限"})
			return
		}

		var count int
		err = authDB.QueryRow(`
			SELECT COUNT(*) FROM business_line_apps bla
			JOIN business_line_admins blad ON bla.business_line_id = blad.business_line_id
			WHERE blad.user_id = (SELECT id FROM users WHERE user_id = ?) AND bla.app_id = ?`, user.UserID, resourceID).Scan(&count)
		if err != nil || count == 0 {
			c.JSON(http.StatusOK, appResp{Code: 403, Msg: "无权限撤销此权限"})
			return
		}
	}

	// Delete permission
	_, err := authDB.Exec("DELETE FROM user_permissions WHERE id = ?", permissionID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, appResp{Msg: "权限撤销成功"})
}

func ListAuditLogs(c *gin.Context) {
	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	userID := c.Query("user_id")
	action := c.Query("action")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	where := "1=1"
	args := []interface{}{}
	if userID != "" {
		where += " AND al.user_id = ?"
		args = append(args, userID)
	}
	if action != "" {
		where += " AND al.action = ?"
		args = append(args, action)
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM user_audit_logs al WHERE %s", where)
	authDB.QueryRow(countQuery, args...).Scan(&total)

	query := fmt.Sprintf(`
		SELECT al.id, al.user_id, COALESCE(u.name, ''), al.action, al.resource_type,
		       al.resource_id, COALESCE(al.detail, ''), COALESCE(al.ip_address, ''), al.created_at
		FROM user_audit_logs al
		LEFT JOIN users u ON al.user_id = u.user_id
		WHERE %s
		ORDER BY al.created_at DESC
		LIMIT ? OFFSET ?`, where)
	args = append(args, pageSize, offset)

	rows, err := authDB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id, uid, userName, act, resType, resID, detail, ip, createdAt string
		rows.Scan(&id, &uid, &userName, &act, &resType, &resID, &detail, &ip, &createdAt)
		logs = append(logs, map[string]interface{}{
			"id": id, "user_id": uid, "user_name": userName,
			"action": act, "resource_type": resType, "resource_id": resID,
			"detail": detail, "ip_address": ip, "created_at": createdAt,
		})
	}

	c.JSON(http.StatusOK, appResp{Data: map[string]interface{}{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"items":     logs,
	}})
}
