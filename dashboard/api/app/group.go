package app

import (
	"database/sql"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/plusplus1/sentinel-go-ext/dashboard/dao"
	"github.com/plusplus1/sentinel-go-ext/dashboard/model"
	"github.com/plusplus1/sentinel-go-ext/dashboard/source/reg"
)

// MySQL connection
var (
	mysqlDB         *sql.DB
	groupDAOMy      *dao.MySQLGroupDAO
	resourceDAOMy   *dao.MySQLResourceDAO
	publishDAOMy    *dao.MySQLPublishRecordDAO
	versionDAOMy    *dao.MySQLPublishVersionDAO
	etcdClientMap   = make(map[string]*clientv3.Client)
	etcdClientMutex sync.Mutex
)

func getMySQLDB() (*sql.DB, error) {
	if mysqlDB == nil {
		db, err := dao.NewMySQLDB(dao.DefaultMySQLConfig())
		if err != nil {
			return nil, err
		}
		mysqlDB = db
		groupDAOMy = dao.NewMySQLGroupDAO(mysqlDB)
		resourceDAOMy = dao.NewMySQLResourceDAO(mysqlDB)
		publishDAOMy = dao.NewMySQLPublishRecordDAO(mysqlDB)
		versionDAOMy = dao.NewMySQLPublishVersionDAO(mysqlDB)
		flowRuleDAOMy = dao.NewMySQLFlowRuleDAO(mysqlDB)
		cbRuleDAOMy = dao.NewMySQLCBRuleDAO(mysqlDB)
	}
	return mysqlDB, nil
}

// Publish types
type PublishRecord = dao.PublishRecord
type PublishVersion = dao.PublishVersion
type RuleSnapshot = dao.RuleSnapshot

// DAO instances
var (
	flowRuleDAOMy *dao.MySQLFlowRuleDAO
	cbRuleDAOMy   *dao.MySQLCBRuleDAO
)

// === Group API (MySQL-backed) ===

// ListGroups lists all groups for an app (MySQL-backed)
func ListGroups(c *gin.Context) {
	appId := c.Query("app")
	if appId == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "app parameter is required"})
		return
	}

	// 验证 app_id 是否存在（查询 business_line_apps 表）
	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	var count int
	err := mysqlDB.QueryRow("SELECT COUNT(*) FROM business_line_apps WHERE id = ?", appId).Scan(&count)
	if err != nil || count == 0 {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "App Not Found"})
		return
	}

	groups, err := groupDAOMy.ListGroups(appId)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	// Ensure empty array instead of null
	if groups == nil {
		groups = []*model.Group{}
	}

	c.JSON(http.StatusOK, appResp{Data: groups})
}

// CreateGroup creates a new group (MySQL-backed)
func CreateGroup(c *gin.Context) {
	var req struct {
		AppID       string `json:"app_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "Invalid request: " + err.Error()})
		return
	}
	if req.AppID == "" || req.Name == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "app_id and name are required"})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	group := &model.Group{
		AppID:       req.AppID,
		Name:        req.Name,
		Description: req.Description,
	}
	if err := groupDAOMy.CreateGroup(group); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, appResp{Data: group})
}

// GetGroup gets a group by ID (MySQL-backed)
func GetGroup(c *gin.Context) {
	appId := c.Query("app")
	groupId := c.Param("id")
	if appId == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "app parameter is required"})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	group, err := groupDAOMy.GetGroup(appId, groupId)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, appResp{Data: group})
}

// UpdateGroup updates a group (MySQL-backed)
func UpdateGroup(c *gin.Context) {
	groupId := c.Param("id")
	var req struct {
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "Invalid request: " + err.Error()})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	if err := groupDAOMy.UpdateGroup(groupId, &model.Group{Description: req.Description}); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, appResp{Msg: "Group updated successfully"})
}

// DeleteGroup deletes a group (MySQL-backed)
func DeleteGroup(c *gin.Context) {
	groupId := c.Param("id")
	if groupId == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "group id is required"})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	// Check if module has bound resources - cannot delete if resources exist
	var count int
	err := mysqlDB.QueryRow(
		"SELECT COUNT(*) FROM business_line_resources WHERE group_id = ?",
		groupId,
	).Scan(&count)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "检查模块资源失败: " + err.Error()})
		return
	}
	if count > 0 {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: fmt.Sprintf("该模块下还有 %d 个资源，请先移除资源后再删除", count)})
		return
	}

	if err := groupDAOMy.DeleteGroup(groupId); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, appResp{Msg: "Group deleted successfully"})
}

// ListGroupMembers lists members of a group
func ListGroupMembers(c *gin.Context) {
	appId := c.Query("app")
	groupId := c.Param("id")
	if appId == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "app parameter is required"})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	resources, err := resourceDAOMy.ListGroupResources(groupId, appId)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	// Ensure empty array instead of null
	if resources == nil {
		resources = []*model.Resource{}
	}

	c.JSON(http.StatusOK, appResp{Data: resources})
}

// AddResourceToGroup adds a resource to a group
func AddResourceToGroup(c *gin.Context) {
	appId := c.Query("app")
	groupId := c.Param("id")
	if appId == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "app parameter is required"})
		return
	}

	var req struct {
		Resource string `json:"resource"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "Invalid request: " + err.Error()})
		return
	}
	if req.Resource == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "resource is required"})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	// Get or create resource
	if _, err := resourceDAOMy.GetOrCreateResource(appId, req.Resource); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	if err := groupDAOMy.AddResourceToGroup(appId, groupId, req.Resource); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, appResp{Msg: "Resource added to group successfully"})
}

// RemoveResourceFromGroup removes a resource from a group
func RemoveResourceFromGroup(c *gin.Context) {
	appId := c.Query("app")
	groupId := c.Param("id")
	resource := c.Param("resource")
	if appId == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "app parameter is required"})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	if err := groupDAOMy.RemoveResourceFromGroup(appId, groupId, resource); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, appResp{Msg: "Resource removed from group successfully"})
}

// GetEtcdClient gets or creates an etcd client for an app
func getEtcdClient(appInfo reg.AppInfo) (*clientv3.Client, error) {
	etcdClientMutex.Lock()
	defer etcdClientMutex.Unlock()

	if client, ok := etcdClientMap[appInfo.Id]; ok {
		return client, nil
	}

	// Get etcd endpoints from app info
	var endpoints []string
	if len(appInfo.Endpoints) > 0 {
		endpoints = appInfo.Endpoints
	} else {
		endpoints = []string{"http://127.0.0.1:2379"}
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5e9, // 5 seconds
	})
	if err != nil {
		return nil, err
	}

	etcdClientMap[appInfo.Id] = client
	return client, nil
}

// EnsureBuild ensures the build cache is populated
func EnsureBuild() {
	// This is a placeholder for the base package
	// The actual implementation is in api/base/apps.go
}
