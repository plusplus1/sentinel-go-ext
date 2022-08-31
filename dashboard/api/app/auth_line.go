package app

import (
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/plusplus1/sentinel-go-ext/dashboard/model"
)

// === Line Admin 专用 API ===

// ListMyBusinessLines returns business lines where the current user is admin
func ListMyBusinessLines(c *gin.Context) {
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

	rows, err := authDB.Query(`
		SELECT 
			bl.id, bl.name, COALESCE(bl.description,''), bl.status, 
			COALESCE(bl.owner_id,''), bl.updated_at 
		FROM business_lines bl 
		JOIN business_line_admins blad ON bl.id = blad.business_line_id
		WHERE blad.user_id = (SELECT id FROM users WHERE user_id = ?)
		ORDER BY bl.updated_at DESC`, user.UserID)
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

	// Get all admins for these lines
	adminMap := make(map[string][]map[string]string)
	if len(lineList) > 0 {
		lineIDs := make([]string, len(lineList))
		for i, li := range lineList {
			lineIDs[i] = li.ID
		}
		placeholders := strings.Repeat("?,", len(lineIDs))
		placeholders = placeholders[:len(placeholders)-1]
		adminQuery := fmt.Sprintf(`SELECT 
			blad.business_line_id, u.user_id, COALESCE(u.name,''), COALESCE(u.email,''), COALESCE(u.status,'')
		FROM business_line_admins blad
		JOIN users u ON blad.user_id = u.id
		WHERE blad.business_line_id IN (%s)`, placeholders)
		adminArgs := make([]interface{}, len(lineIDs))
		for i, id := range lineIDs {
			adminArgs[i] = id
		}
		adminRows, err := authDB.Query(adminQuery, adminArgs...)
		if err == nil {
			defer adminRows.Close()
			for adminRows.Next() {
				var lineID, adminUserID, adminName, adminEmail, adminStatus string
				adminRows.Scan(&lineID, &adminUserID, &adminName, &adminEmail, &adminStatus)
				adminMap[lineID] = append(adminMap[lineID], map[string]string{
					"user_id": adminUserID, "user_name": adminName, "user_email": adminEmail, "user_status": adminStatus,
				})
			}
		}
	}

	// Build response with admins array
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

// UpdateMyBusinessLine updates description of a business line owned by the current line admin
func UpdateMyBusinessLine(c *gin.Context) {
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusOK, appResp{Code: 401, Msg: "未登录"})
		return
	}
	user := currentUser.(*model.User)
	lineID := c.Param("id")

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	// Verify user is admin of this business line
	isAdmin, err := isLineAdmin(user.UserID, lineID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	if !isAdmin {
		c.JSON(http.StatusOK, appResp{Code: 403, Msg: "无权限修改此业务线"})
		return
	}

	var req struct {
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "参数错误"})
		return
	}

	// Only update description
	_, err = authDB.Exec("UPDATE business_lines SET description = ?, updated_at = NOW() WHERE id = ?", req.Description, lineID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, appResp{Msg: "业务线描述更新成功"})
}

// CreateBusinessLineApp creates a new app within a business line (for line admin)
func CreateBusinessLineApp(c *gin.Context) {
	lineID := c.Param("id")
	if lineID == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线ID不能为空"})
		return
	}

	// Check permission
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
		AppKey      string `json:"app_key"`     // 英文、数字、下划线
		Description string `json:"description"` // 中文描述
		Settings    string `json:"settings"`    // etcd 配置
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
	result, err := authDB.Exec("INSERT INTO business_line_apps (business_line_id, app_key, description, settings, status) VALUES (?, ?, ?, ?, 'active')",
		lineID, req.AppKey, req.Description, req.Settings)
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
		"id": appID, "app_key": req.AppKey, "description": req.Description,
	}})
}

// UpdateBusinessLineApp updates an app within a business line (for line admin)
func UpdateBusinessLineApp(c *gin.Context) {
	lineID := c.Param("id")
	appID := c.Param("app_id")
	if lineID == "" || appID == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线ID和应用ID不能为空"})
		return
	}

	// Check permission
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
		Settings    string `json:"settings"`
		Status      string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "参数错误"})
		return
	}

	// Verify app exists and belongs to this business line
	var currentLineID int
	err := authDB.QueryRow("SELECT business_line_id FROM business_line_apps WHERE id = ?", appID).Scan(&currentLineID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "应用不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	var lineIDInt int
	fmt.Sscanf(lineID, "%d", &lineIDInt)
	if currentLineID != lineIDInt {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "应用不属于此业务线"})
		return
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
	if req.Settings != "" {
		setClauses = append(setClauses, "settings = ?")
		args = append(args, req.Settings)
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

// DeleteBusinessLineApp deletes an app from a business line (for line admin)
func DeleteBusinessLineApp(c *gin.Context) {
	lineID := c.Param("id")
	appID := c.Param("app_id")
	if lineID == "" || appID == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "业务线ID和应用ID不能为空"})
		return
	}

	// Check permission
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

	// Verify app exists and belongs to this business line
	var currentLineID int
	err := authDB.QueryRow("SELECT business_line_id FROM business_line_apps WHERE id = ?", appID).Scan(&currentLineID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "应用不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	// Convert lineID to int for comparison
	var lineIDInt int
	fmt.Sscanf(lineID, "%d", &lineIDInt)
	if currentLineID != lineIDInt {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "应用不属于此业务线"})
		return
	}

	// Delete the app
	_, err = authDB.Exec("DELETE FROM business_line_apps WHERE id = ?", appID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, appResp{Msg: "应用已从业务线移除"})
}

// ListBusinessLineMembers returns members of a business line (line admin only)
func ListBusinessLineMembers(c *gin.Context) {
	lineID := c.Param("id")
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

	// Verify user is admin of this line (or super_admin)
	if user.Role != "super_admin" {
		isAdmin, err := isLineAdmin(user.UserID, lineID)
		if err != nil || !isAdmin {
			c.JSON(http.StatusOK, appResp{Code: 403, Msg: "无权限查看此业务线成员"})
			return
		}
	}

	rows, err := authDB.Query(`SELECT blm.id, u.user_id, COALESCE(u.name,''), COALESCE(u.email,''), COALESCE(u.status,''), COALESCE(ab.name,''), blm.created_at
		FROM business_line_members blm
		JOIN users u ON blm.user_id = u.id
		LEFT JOIN users ab ON blm.added_by = ab.id
		WHERE blm.business_line_id = ? ORDER BY blm.created_at DESC`, lineID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	defer rows.Close()

	var members []map[string]interface{}
	for rows.Next() {
		var id, userID, userName, userEmail, userStatus, addedBy string
		var createdAt string
		rows.Scan(&id, &userID, &userName, &userEmail, &userStatus, &addedBy, &createdAt)
		members = append(members, map[string]interface{}{
			"id": id, "user_id": userID, "user_name": userName, "user_email": userEmail,
			"user_status": userStatus, "added_by": addedBy, "created_at": createdAt,
		})
	}
	if members == nil {
		members = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, appResp{Data: members})
}

// AddBusinessLineMember adds a regular member to a business line (line admin only)
func AddBusinessLineMember(c *gin.Context) {
	lineID := c.Param("id")
	currentUser, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusOK, appResp{Code: 401, Msg: "未登录"})
		return
	}
	user := currentUser.(*model.User)

	var req struct {
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.UserID == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "user_id is required"})
		return
	}

	if _, err := getAuthDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "数据库连接失败"})
		return
	}

	// Verify user is admin of this line (or super_admin)
	if user.Role != "super_admin" {
		isAdmin, err := isLineAdmin(user.UserID, lineID)
		if err != nil || !isAdmin {
			c.JSON(http.StatusOK, appResp{Code: 403, Msg: "无权限添加成员"})
			return
		}
	}

	// Check if already member
	var count int
	authDB.QueryRow("SELECT COUNT(*) FROM business_line_members WHERE business_line_id = ? AND user_id = (SELECT id FROM users WHERE user_id = ?)", lineID, req.UserID).Scan(&count)
	if count > 0 {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "用户已是该业务线成员"})
		return
	}

	_, err := authDB.Exec("INSERT INTO business_line_members (business_line_id, user_id, added_by) VALUES (?, (SELECT id FROM users WHERE user_id = ?), (SELECT id FROM users WHERE user_id = ?))", lineID, req.UserID, user.UserID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, appResp{Msg: "成员添加成功"})
}

// RemoveBusinessLineMember removes a member from a business line (line admin only)
func RemoveBusinessLineMember(c *gin.Context) {
	lineID := c.Param("id")
	memberUserID := c.Param("user_id")
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

	// Verify user is admin of this line (or super_admin)
	if user.Role != "super_admin" {
		isAdmin, err := isLineAdmin(user.UserID, lineID)
		if err != nil || !isAdmin {
			c.JSON(http.StatusOK, appResp{Code: 403, Msg: "无权限移除成员"})
			return
		}
	}

	_, err := authDB.Exec("DELETE FROM business_line_members WHERE business_line_id = ? AND user_id = (SELECT id FROM users WHERE user_id = ?)", lineID, memberUserID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, appResp{Msg: "成员已移除"})
}
