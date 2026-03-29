package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/plusplus1/sentinel-go-ext/dashboard/dao"
	"github.com/plusplus1/sentinel-go-ext/dashboard/model"
	"github.com/plusplus1/sentinel-go-ext/dashboard/provider"
)

var etcdMgr = provider.NewEtcdClientManager()

// getResourceNameByID looks up the resource name from its primary key
func getResourceNameByID(resourceID int64) string {
	var name string
	if err := mysqlDB.QueryRow("SELECT name FROM business_line_resources WHERE id = ?", resourceID).Scan(&name); err != nil {
		return fmt.Sprintf("unknown_%d", resourceID)
	}
	return name
}

// ListResources lists all resources for an app (MySQL-backed)
// ListResources lists all resources for an app (MySQL-backed)
func ListResources(c *gin.Context) {
	appId := c.Query("app") // 现在 appId 实际上是 app_id (数字)
	if appId == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "app_id is required"})
		return
	}

	// 获取数据库连接
	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	// 验证 app_id 是否存在（查询 business_line_apps 表）
	var count int
	err := mysqlDB.QueryRow("SELECT COUNT(*) FROM business_line_apps WHERE id = ?", appId).Scan(&count)
	if err != nil || count == 0 {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "App Not Found"})
		return
	}

	// 获取该 app 下的所有 groups
	groupRows, err := mysqlDB.Query("SELECT id FROM business_line_app_groups WHERE app_id = ?", appId)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	defer groupRows.Close()

	var groupIDs []string
	for groupRows.Next() {
		var groupID string
		groupRows.Scan(&groupID)
		groupIDs = append(groupIDs, groupID)
	}

	if len(groupIDs) == 0 {
		c.JSON(http.StatusOK, appResp{Data: []interface{}{}})
		return
	}

	// 构建 IN 子句
	placeholders := ""
	args := []interface{}{}
	for i, gid := range groupIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, gid)
	}

	// 查询这些 groups 下的所有 resources
	query := fmt.Sprintf(`SELECT r.id, r.name, r.description, r.group_id, r.created_at, r.updated_at, 
		COALESCE(g.name, '') as group_name, COALESCE(g.description, '') as group_description 
		FROM business_line_resources r 
		LEFT JOIN business_line_app_groups g ON r.group_id = g.id 
		WHERE r.group_id IN (%s) 
		ORDER BY r.name`, placeholders)
	resourceRows, err := mysqlDB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}
	defer resourceRows.Close()

	var resources []*model.Resource
	for resourceRows.Next() {
		var r model.Resource
		var id int64
		var groupID sql.NullInt64
		var groupName string
		var groupDescription string
		if err := resourceRows.Scan(&id, &r.Name, &r.Description, &groupID, &r.CreatedAt, &r.UpdatedAt, &groupName, &groupDescription); err != nil {
			continue
		}
		r.ID = fmt.Sprintf("%d", id)
		r.GroupName = groupName
		r.GroupDescription = groupDescription
		if groupID.Valid {
			gid := fmt.Sprintf("%d", groupID.Int64)
			r.GroupID = &gid
		}
		resources = append(resources, &r)
	}

	// 规则计数部分暂时返回空，后续需要调整（规则表仍使用 app_key 和 env）
	// TODO: 实现规则计数逻辑
	flowCounts := make(map[string]int)
	cbCounts := make(map[string]int)

	publishStatusMap := make(map[string]map[string]interface{})
	pubRows, err := mysqlDB.Query(`
		SELECT pr.resource, pr.created_at, pv.version_number
		FROM publish_records pr
		LEFT JOIN publish_versions pv ON pv.app_id = pr.app_id AND pv.version_number = (
			SELECT MAX(version_number) FROM publish_versions WHERE app_id = pr.app_id AND created_at <= pr.created_at
		)
		WHERE pr.app_id = ? AND pr.status = 'success'
		ORDER BY pr.created_at DESC`, appId)
	if err == nil && pubRows != nil {
		defer pubRows.Close()
		for pubRows.Next() {
			var resName string
			var pubTime sql.NullString
			var versionNum sql.NullInt64
			pubRows.Scan(&resName, &pubTime, &versionNum)
			if _, exists := publishStatusMap[resName]; !exists {
				info := map[string]interface{}{"last_publish_at": pubTime.String}
				if versionNum.Valid {
					info["running_version"] = versionNum.Int64
				}
				publishStatusMap[resName] = info
			}
		}
	}

	latestVersion := 0
	if v, err := versionDAOMy.GetLatestVersionNumber(appId); err == nil {
		latestVersion = v
	}

	type ResourceWithCounts struct {
		*model.Resource
		FlowRuleCount  int         `json:"flow_rule_count"`
		CBRuleCount    int         `json:"cb_rule_count"`
		LastPublishAt  string      `json:"last_publish_at,omitempty"`
		RunningVersion interface{} `json:"running_version,omitempty"`
		LatestVersion  int         `json:"latest_version"`
	}
	var result []ResourceWithCounts
	result = []ResourceWithCounts{}
	for _, r := range resources {
		rc := ResourceWithCounts{
			Resource:      r,
			FlowRuleCount: flowCounts[r.Name],
			CBRuleCount:   cbCounts[r.Name],
			LatestVersion: latestVersion,
		}
		if pub, ok := publishStatusMap[r.Name]; ok {
			if t, ok := pub["last_publish_at"].(string); ok {
				rc.LastPublishAt = t
			}
			rc.RunningVersion = pub["running_version"]
		}
		result = append(result, rc)
	}

	c.JSON(http.StatusOK, appResp{Data: result})
}

// GetResourceWithRules gets a resource with its rules (MySQL-backed for rules, etcd for live data)
func GetResourceWithRules(c *gin.Context) {
	appId := c.Query("app")
	resourceIdentifier := c.Param("id")

	if appId == "" || resourceIdentifier == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "app and resource id are required"})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL not available: " + err.Error()})
		return
	}

	// Look up resource by ID to get name and ID for rule queries
	resourceName := resourceIdentifier
	var resourceID int64
	if res, err := resourceDAOMy.GetResourceByID(resourceIdentifier); err == nil {
		resourceName = res.Name
		fmt.Sscanf(res.ID, "%d", &resourceID)
	}

	flowRules, _ := flowRuleDAOMy.ListRules(appId, resourceID)
	cbRules, _ := cbRuleDAOMy.ListRules(appId, resourceID)

	// Convert to response format (camelCase for frontend compatibility)
	var flowRulesInterface []interface{}
	flowRulesInterface = []interface{}{}
	for _, r := range flowRules {
		flowRulesInterface = append(flowRulesInterface, map[string]interface{}{
			"id":                     r.ID,
			"resource_id":            r.ResourceID,
			"resource":               getResourceNameByID(r.ResourceID),
			"threshold":              r.Threshold,
			"metricType":             r.MetricType,
			"controlBehavior":        r.ControlBehavior,
			"warmUpPeriodSec":        r.WarmUpPeriodSec,
			"maxQueueingTimeMs":      r.MaxQueueingTimeMs,
			"clusterMode":            r.ClusterMode,
			"tokenCalculateStrategy": r.TokenCalculateStrategy,
			"relationStrategy":       r.RelationStrategy,
			"refResource":            r.RefResource,
			"warmUpColdFactor":       r.WarmUpColdFactor,
			"statIntervalInMs":       r.StatIntervalMs,
			"enabled":                r.Enabled,
		})
	}
	var cbRulesInterface []interface{}
	cbRulesInterface = []interface{}{}
	for _, r := range cbRules {
		cbRulesInterface = append(cbRulesInterface, map[string]interface{}{
			"id":                           r.ID,
			"resource_id":                  r.ResourceID,
			"resource":                     getResourceNameByID(r.ResourceID),
			"strategy":                     r.Strategy,
			"threshold":                    r.Threshold,
			"retryTimeoutMs":               r.RetryTimeoutMs,
			"minRequestAmount":             r.MinRequestAmount,
			"statIntervalMs":               r.StatIntervalMs,
			"statSlidingWindowBucketCount": r.StatSlidingWindowBucketCount,
			"maxAllowedRtMs":               r.MaxAllowedRtMs,
			"probeNum":                     r.ProbeNum,
			"enabled":                      r.Enabled,
		})
	}

	result := map[string]interface{}{
		"resource": map[string]interface{}{
			"name":   resourceName,
			"app_id": appId,
		},
		"flow_rules":            flowRulesInterface,
		"circuit_breaker_rules": cbRulesInterface,
		"summary": map[string]interface{}{
			"total_rules":     len(flowRules) + len(cbRules),
			"enabled_rules":   len(flowRules) + len(cbRules),
			"triggered_rules": 0,
			"health_status":   "healthy",
		},
	}
	c.JSON(http.StatusOK, appResp{Data: result})
}

// GetResourceMetadata gets resource metadata (MySQL-backed)
func GetResourceMetadata(c *gin.Context) {
	resourceIdentifier := c.Param("id")
	if resourceIdentifier == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "resource id is required"})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	resource, err := resourceDAOMy.GetResourceByID(resourceIdentifier)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "Resource not found"})
		return
	}

	c.JSON(http.StatusOK, appResp{Data: resource})
}

// UpdateResourceGroup updates the resource group assignment (module)
func UpdateResourceGroup(c *gin.Context) {
	resourceName := c.Param("id")
	appId := c.Query("app")
	if resourceName == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "resource id is required"})
		return
	}
	if appId == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "app is required"})
		return
	}

	var req struct {
		GroupID     string `json:"group_id"`
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

	// Build SET clause dynamically
	var sets []string
	if req.GroupID != "" {
		sets = append(sets, "group_id = VALUES(group_id)")
	}
	if req.Description != "" {
		sets = append(sets, "description = VALUES(description)")
	}
	if len(sets) == 0 {
		sets = append(sets, "description = VALUES(description)")
	}

	query := "INSERT INTO business_line_resources (app_id, name, description, group_id) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE " + strings.Join(sets, ", ") + ", updated_at = NOW()"
	var groupID interface{}
	if req.GroupID != "" {
		groupID = req.GroupID
	}
	result, err := mysqlDB.Exec(query, appId, resourceName, req.Description, groupID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "Resource not found"})
		return
	}

	c.JSON(http.StatusOK, appResp{Msg: "Resource updated successfully"})
}

// ToggleRule toggles a flow or circuit breaker rule (MySQL-backed)
func ToggleRule(c *gin.Context) {
	appId := c.Query("app")
	resourceIDStr := c.Query("resource")
	ruleIDStr := c.Param("id")
	ruleType := c.Param("type")

	if appId == "" {
		appId = c.Query("app")
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	var ruleID int64
	fmt.Sscanf(ruleIDStr, "%d", &ruleID)
	var resourceID int64
	fmt.Sscanf(resourceIDStr, "%d", &resourceID)

	isFlowRule := ruleType == "flow"

	if isFlowRule {
		// Get current state and toggle
		rule, err := flowRuleDAOMy.GetRule(ruleID)
		if err != nil {
			c.JSON(http.StatusOK, appResp{Code: 999, Msg: "Rule not found: " + err.Error()})
			return
		}
		if err := flowRuleDAOMy.ToggleRule(ruleID, !rule.Enabled); err != nil {
			c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
			return
		}
	} else {
		// For CB rules, we need to get the list to find the rule
		rules, _ := cbRuleDAOMy.ListRules(appId, resourceID)
		var currentEnabled bool
		found := false
		for _, r := range rules {
			if r.ID == ruleID {
				currentEnabled = r.Enabled
				found = true
				break
			}
		}
		if !found {
			c.JSON(http.StatusOK, appResp{Code: 999, Msg: "Rule not found"})
			return
		}
		if err := cbRuleDAOMy.ToggleRule(ruleID, !currentEnabled); err != nil {
			c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
			return
		}
	}

	if svc, err := GetAuthService(); err == nil {
		if u, exists := c.Get("user"); exists {
			user := u.(*model.User)
			svc.LogAudit(user.UserID, "rule_toggle", ruleType+"_rule", ruleIDStr,
				fmt.Sprintf("resource=%s", resourceIDStr), c.ClientIP())
		}
	}

	c.JSON(http.StatusOK, appResp{Msg: "Rule toggled successfully"})
}

// --- Rule APIs are in rule_flow.go and rule_circuitbreaker.go (modified to use MySQL) ---

// DeleteResource deletes a resource (MySQL-backed, by ID or name)
// DeleteResource deletes a resource (MySQL-backed, by ID)
func DeleteResource(c *gin.Context) {
	resourceIdentifier := c.Query("id")
	if resourceIdentifier == "" {
		resourceIdentifier = c.Param("id")
	}
	if resourceIdentifier == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "resource id is required"})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	mysqlDB.Exec("DELETE FROM business_line_resource_flowrules WHERE resource_id = ?", resourceIdentifier)
	mysqlDB.Exec("DELETE FROM business_line_resource_circuitbreakerrules WHERE resource_id = ?", resourceIdentifier)
	_, err := mysqlDB.Exec("DELETE FROM business_line_resources WHERE id = ?", resourceIdentifier)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	if svc, err := GetAuthService(); err == nil {
		if u, exists := c.Get("user"); exists {
			user := u.(*model.User)
			svc.LogAudit(user.UserID, "resource_delete", "resource", resourceIdentifier, "", c.ClientIP())
		}
	}

	c.JSON(http.StatusOK, appResp{Msg: "Resource deleted successfully"})
}

// PublishRules publishes rules from MySQL to etcd
// etcd key path: /sentinel/{business_line}/{app_key}/{group}/{resource}/{rule_type}
func PublishRules(c *gin.Context) {
	var req struct {
		AppID    string `json:"app_key"`
		RuleType string `json:"rule_type"`
		Resource string `json:"resource"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "Invalid request: " + err.Error()})
		return
	}
	if req.AppID == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "app_id is required"})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	if req.Resource != "" {
		if changed, err := hasResourceChanged(req.AppID, req.Resource); err == nil && !changed {
			c.JSON(http.StatusOK, appResp{Msg: "配置无变更，跳过发布", Data: map[string]interface{}{"skipped": true}})
			return
		}
	}

	var appKey, settingsJSON, lineName string
	var businessLineID int64
	err := mysqlDB.QueryRow(`
		SELECT bla.app_key, bla.business_line_id, COALESCE(bla.settings, ''), bl.name
		FROM business_line_apps bla
		JOIN business_lines bl ON bla.business_line_id = bl.id
		WHERE bla.id = ?`, req.AppID).Scan(&appKey, &businessLineID, &settingsJSON, &lineName)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "App not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	client, err := etcdMgr.GetOrCreateClient(req.AppID, settingsJSON)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "Failed to connect to etcd: " + err.Error()})
		return
	}
	publisher := provider.NewEtcdRulePublisher(client)
	pathBuilder := provider.NewEtcdPathBuilder()

	groupMap := make(map[int64]string)
	rows, _ := mysqlDB.Query("SELECT id, name FROM business_line_app_groups WHERE app_id = ?", req.AppID)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var gid int64
			var gname string
			rows.Scan(&gid, &gname)
			groupMap[gid] = gname
		}
	}

	resourceGroupMap := make(map[int64]string)
	resRows, _ := mysqlDB.Query("SELECT id, COALESCE(group_id, 0) FROM business_line_resources WHERE app_id = ?", req.AppID)
	if resRows != nil {
		defer resRows.Close()
		for resRows.Next() {
			var rid, gid int64
			resRows.Scan(&rid, &gid)
			if gname, ok := groupMap[gid]; ok {
				resourceGroupMap[rid] = gname
			} else {
				resourceGroupMap[rid] = "default"
			}
		}
	}

	var published int
	versionNum := 1
	if v, err := versionDAOMy.GetLatestVersionNumber(req.AppID); err == nil {
		versionNum = v + 1
	}

	var allFlowRules []interface{}
	var allCBRules []interface{}

	if req.RuleType == "flow" || req.RuleType == "all" {
		var resourceID int64
		if req.Resource != "" {
			fmt.Sscanf(req.Resource, "%d", &resourceID)
		}
		rules, _ := flowRuleDAOMy.ListRules(req.AppID, resourceID)
		type ruleKey struct {
			group    string
			resource string
		}
		grouped := make(map[ruleKey][]interface{})
		for _, r := range rules {
			if !r.Enabled {
				continue
			}
			resourceName := getResourceNameByID(r.ResourceID)
			groupName := resourceGroupMap[r.ResourceID]
			if groupName == "" {
				groupName = "default"
			}
			rule := map[string]interface{}{
				"resource": resourceName, "threshold": r.Threshold,
				"metricType": r.MetricType, "controlBehavior": r.ControlBehavior,
				"warmUpPeriodSec": r.WarmUpPeriodSec, "maxQueueingTimeMs": r.MaxQueueingTimeMs,
				"tokenCalculateStrategy": r.TokenCalculateStrategy, "relationStrategy": r.RelationStrategy,
				"refResource": r.RefResource, "warmUpColdFactor": r.WarmUpColdFactor,
				"statIntervalInMs": r.StatIntervalMs, "clusterMode": r.ClusterMode,
			}
			k := ruleKey{group: groupName, resource: resourceName}
			grouped[k] = append(grouped[k], rule)
			allFlowRules = append(allFlowRules, rule)
			published++
		}
		for k, rr := range grouped {
			data, _ := json.Marshal(rr)
			key := pathBuilder.BuildPath(lineName, appKey, k.group, k.resource, "flow")
			publisher.PublishRules(key, data)
		}
	}

	if req.RuleType == "circuitbreaker" || req.RuleType == "all" {
		var resourceID int64
		if req.Resource != "" {
			fmt.Sscanf(req.Resource, "%d", &resourceID)
		}
		rules, _ := cbRuleDAOMy.ListRules(req.AppID, resourceID)
		type ruleKey struct {
			group    string
			resource string
		}
		grouped := make(map[ruleKey][]interface{})
		for _, r := range rules {
			if !r.Enabled {
				continue
			}
			resourceName := getResourceNameByID(r.ResourceID)
			groupName := resourceGroupMap[r.ResourceID]
			if groupName == "" {
				groupName = "default"
			}
			rule := map[string]interface{}{
				"resource": resourceName, "strategy": r.Strategy, "threshold": r.Threshold,
				"retryTimeoutMs": r.RetryTimeoutMs, "minRequestAmount": r.MinRequestAmount,
				"statIntervalMs": r.StatIntervalMs, "statSlidingWindowBucketCount": r.StatSlidingWindowBucketCount,
				"maxAllowedRtMs": r.MaxAllowedRtMs, "probeNum": r.ProbeNum,
			}
			k := ruleKey{group: groupName, resource: resourceName}
			grouped[k] = append(grouped[k], rule)
			allCBRules = append(allCBRules, rule)
			published++
		}
		for k, rr := range grouped {
			data, _ := json.Marshal(rr)
			key := pathBuilder.BuildPath(lineName, appKey, k.group, k.resource, "circuitbreaker")
			publisher.PublishRules(key, data)
		}
	}

	snapshot := map[string]interface{}{
		"flow_rules": allFlowRules, "circuit_breaker_rules": allCBRules,
	}
	snapshotJSON, _ := json.Marshal(snapshot)

	// Begin MySQL transaction for version and publish record
	tx, err := mysqlDB.Begin()
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "Failed to begin transaction: " + err.Error()})
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`INSERT INTO publish_versions 
		(app_id, version_number, description, operator, rule_count, snapshot, status, error_msg)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		req.AppID, versionNum,
		fmt.Sprintf("发布 %d 条规则", published),
		"user", published, string(snapshotJSON), "success", ""); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "Failed to create version record: " + err.Error()})
		return
	}

	if _, err := tx.Exec(`INSERT INTO publish_records (app_id, rule_type, resource, rule_count, status, error_msg, operator)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		req.AppID, req.RuleType, req.Resource, published, "success", "", "user"); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "Failed to create publish record: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "Failed to commit transaction: " + err.Error()})
		return
	}

	if svc, err := GetAuthService(); err == nil {
		if u, exists := c.Get("user"); exists {
			user := u.(*model.User)
			svc.LogAudit(user.UserID, "publish", "rules", req.AppID,
				fmt.Sprintf("v%d, %d rules, type=%s", versionNum, published, req.RuleType), c.ClientIP())
		}
	}

	c.JSON(http.StatusOK, appResp{Msg: fmt.Sprintf("已发布 v%d，共 %d 条规则", versionNum, published)})
}

// FieldDiff represents a field-level diff between current and published rule
type FieldDiff struct {
	Field    string      `json:"field"`
	Label    string      `json:"label"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
	Changed  bool        `json:"changed"`
}

func compareFlowRule(current *dao.FlowRuleRecord, published map[string]interface{}) []FieldDiff {
	if published == nil {
		return []FieldDiff{
			{Field: "_new", Label: "规则", OldValue: nil, NewValue: "新增", Changed: true},
		}
	}

	fields := []struct {
		field  string
		label  string
		newVal interface{}
		oldKey string
	}{
		{"threshold", "阈值", current.Threshold, "threshold"},
		{"metricType", "指标类型", current.MetricType, "metricType"},
		{"controlBehavior", "控制行为", current.ControlBehavior, "controlBehavior"},
		{"warmUpPeriodSec", "预热时长(秒)", current.WarmUpPeriodSec, "warmUpPeriodSec"},
		{"maxQueueingTimeMs", "最大排队时间(ms)", current.MaxQueueingTimeMs, "maxQueueingTimeMs"},
		{"tokenCalculateStrategy", "Token计算策略", current.TokenCalculateStrategy, "tokenCalculateStrategy"},
		{"relationStrategy", "关联策略", current.RelationStrategy, "relationStrategy"},
		{"refResource", "关联资源", current.RefResource, "refResource"},
		{"warmUpColdFactor", "冷启动因子", current.WarmUpColdFactor, "warmUpColdFactor"},
		{"statIntervalInMs", "统计窗口(ms)", current.StatIntervalMs, "statIntervalInMs"},
		{"clusterMode", "集群模式", current.ClusterMode, "clusterMode"},
	}

	var diffs []FieldDiff
	for _, f := range fields {
		oldVal := published[f.oldKey]
		changed := false
		if oldVal == nil {
			changed = true
		} else {
			switch v := f.newVal.(type) {
			case float64:
				if ov, ok := oldVal.(float64); ok {
					changed = v != ov
				} else {
					changed = true
				}
			case int:
				if ov, ok := oldVal.(float64); ok {
					changed = float64(v) != ov
				} else {
					changed = true
				}
			default:
				changed = fmt.Sprintf("%v", f.newVal) != fmt.Sprintf("%v", oldVal)
			}
		}
		diffs = append(diffs, FieldDiff{
			Field: f.field, Label: f.label,
			OldValue: oldVal, NewValue: f.newVal, Changed: changed,
		})
	}
	return diffs
}

func compareCBRule(current *dao.CBRuleRecord, published map[string]interface{}) []FieldDiff {
	if published == nil {
		return []FieldDiff{
			{Field: "_new", Label: "规则", OldValue: nil, NewValue: "新增", Changed: true},
		}
	}

	fields := []struct {
		field  string
		label  string
		newVal interface{}
		oldKey string
	}{
		{"strategy", "熔断策略", current.Strategy, "strategy"},
		{"threshold", "阈值", current.Threshold, "threshold"},
		{"retryTimeoutMs", "重试超时(ms)", current.RetryTimeoutMs, "retryTimeoutMs"},
		{"minRequestAmount", "最小请求数", current.MinRequestAmount, "minRequestAmount"},
		{"statIntervalMs", "统计窗口(ms)", current.StatIntervalMs, "statIntervalMs"},
		{"statSlidingWindowBucketCount", "滑动窗口桶数", current.StatSlidingWindowBucketCount, "statSlidingWindowBucketCount"},
		{"maxAllowedRtMs", "最大允许RT(ms)", current.MaxAllowedRtMs, "maxAllowedRtMs"},
		{"probeNum", "探测数量", current.ProbeNum, "probeNum"},
	}

	var diffs []FieldDiff
	for _, f := range fields {
		oldVal := published[f.oldKey]
		changed := false
		if oldVal == nil {
			changed = true
		} else {
			switch v := f.newVal.(type) {
			case float64:
				if ov, ok := oldVal.(float64); ok {
					changed = v != ov
				} else {
					changed = true
				}
			case int:
				if ov, ok := oldVal.(float64); ok {
					changed = float64(v) != ov
				} else {
					changed = true
				}
			case int64:
				if ov, ok := oldVal.(float64); ok {
					changed = float64(v) != ov
				} else {
					changed = true
				}
			default:
				changed = fmt.Sprintf("%v", f.newVal) != fmt.Sprintf("%v", oldVal)
			}
		}
		diffs = append(diffs, FieldDiff{
			Field: f.field, Label: f.label,
			OldValue: oldVal, NewValue: f.newVal, Changed: changed,
		})
	}
	return diffs
}

func rulesChanged(currentFlow, publishedFlow, currentCB, publishedCB []map[string]interface{}) bool {
	if len(currentFlow) != len(publishedFlow) || len(currentCB) != len(publishedCB) {
		return true
	}
	for i, current := range currentFlow {
		if !mapEqual(current, publishedFlow[i]) {
			return true
		}
	}
	for i, current := range currentCB {
		if !mapEqual(current, publishedCB[i]) {
			return true
		}
	}
	return false
}

func mapEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, exists := b[k]
		if !exists || !valueEqual(av, bv) {
			return false
		}
	}
	return true
}

func valueEqual(a, b interface{}) bool {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if aOk && bOk {
		return af == bf
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	default:
		return 0, false
	}
}

func hasResourceChanged(appID, resourceStr string) (bool, error) {
	var resourceID int64
	fmt.Sscanf(resourceStr, "%d", &resourceID)
	resourceName := getResourceNameByID(resourceID)

	currentFlow, _ := flowRuleDAOMy.ListRules(appID, resourceID)
	currentCB, _ := cbRuleDAOMy.ListRules(appID, resourceID)

	var currentFlowMaps []map[string]interface{}
	for _, r := range currentFlow {
		if !r.Enabled {
			continue
		}
		rule := map[string]interface{}{
			"resource":               resourceName,
			"threshold":              r.Threshold,
			"metricType":             r.MetricType,
			"controlBehavior":        r.ControlBehavior,
			"warmUpPeriodSec":        r.WarmUpPeriodSec,
			"maxQueueingTimeMs":      r.MaxQueueingTimeMs,
			"tokenCalculateStrategy": r.TokenCalculateStrategy,
			"relationStrategy":       r.RelationStrategy,
			"refResource":            r.RefResource,
			"warmUpColdFactor":       r.WarmUpColdFactor,
			"statIntervalInMs":       r.StatIntervalMs,
			"clusterMode":            r.ClusterMode,
		}
		currentFlowMaps = append(currentFlowMaps, rule)
	}

	var currentCBMaps []map[string]interface{}
	for _, r := range currentCB {
		if !r.Enabled {
			continue
		}
		rule := map[string]interface{}{
			"resource":                     resourceName,
			"strategy":                     r.Strategy,
			"threshold":                    r.Threshold,
			"retryTimeoutMs":               r.RetryTimeoutMs,
			"minRequestAmount":             r.MinRequestAmount,
			"statIntervalMs":               r.StatIntervalMs,
			"statSlidingWindowBucketCount": r.StatSlidingWindowBucketCount,
			"maxAllowedRtMs":               r.MaxAllowedRtMs,
			"probeNum":                     r.ProbeNum,
		}
		currentCBMaps = append(currentCBMaps, rule)
	}

	var snapshotJSON string
	err := mysqlDB.QueryRow(`
		SELECT snapshot FROM publish_versions
		WHERE app_id = ?
		ORDER BY version_number DESC LIMIT 1`, appID).Scan(&snapshotJSON)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return true, err
	}

	var snapshot struct {
		FlowRules []map[string]interface{} `json:"flow_rules"`
		CBRules   []map[string]interface{} `json:"circuit_breaker_rules"`
	}
	if err := json.Unmarshal([]byte(snapshotJSON), &snapshot); err != nil {
		return true, err
	}

	var publishedFlowMaps []map[string]interface{}
	for _, r := range snapshot.FlowRules {
		if name, _ := r["resource"].(string); name == resourceName {
			publishedFlowMaps = append(publishedFlowMaps, r)
		}
	}

	var publishedCBMaps []map[string]interface{}
	for _, r := range snapshot.CBRules {
		if name, _ := r["resource"].(string); name == resourceName {
			publishedCBMaps = append(publishedCBMaps, r)
		}
	}

	return rulesChanged(currentFlowMaps, publishedFlowMaps, currentCBMaps, publishedCBMaps), nil
}

// GetResourceDiff returns field-level diff between current rules and last published version
func GetResourceDiff(c *gin.Context) {
	resourceIDStr := c.Param("id")
	appId := c.Query("app")

	if resourceIDStr == "" || appId == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "resource id and app are required"})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL not available"})
		return
	}

	var resourceID int64
	fmt.Sscanf(resourceIDStr, "%d", &resourceID)
	resourceName := getResourceNameByID(resourceID)

	currentFlow, _ := flowRuleDAOMy.ListRules(appId, resourceID)
	currentCB, _ := cbRuleDAOMy.ListRules(appId, resourceID)

	var snapshotJSON string
	err := mysqlDB.QueryRow(`
		SELECT snapshot FROM publish_versions
		WHERE app_id = ?
		ORDER BY version_number DESC LIMIT 1`, appId).Scan(&snapshotJSON)

	var publishedFlow map[string]interface{}
	var publishedCB map[string]interface{}

	if err == nil && snapshotJSON != "" {
		var snapshot struct {
			FlowRules []map[string]interface{} `json:"flow_rules"`
			CBRules   []map[string]interface{} `json:"circuit_breaker_rules"`
		}
		json.Unmarshal([]byte(snapshotJSON), &snapshot)

		for _, r := range snapshot.FlowRules {
			if name, _ := r["resource"].(string); name == resourceName {
				publishedFlow = r
				break
			}
		}
		for _, r := range snapshot.CBRules {
			if name, _ := r["resource"].(string); name == resourceName {
				publishedCB = r
				break
			}
		}
	}

	var flowDiffs []FieldDiff
	var cbDiffs []FieldDiff

	if len(currentFlow) > 0 {
		flowDiffs = compareFlowRule(currentFlow[0], publishedFlow)
	} else if publishedFlow != nil {
		flowDiffs = []FieldDiff{{Field: "_deleted", Label: "规则", OldValue: "已配置", NewValue: "已删除", Changed: true}}
	} else {
		flowDiffs = []FieldDiff{}
	}

	if len(currentCB) > 0 {
		cbDiffs = compareCBRule(currentCB[0], publishedCB)
	} else if publishedCB != nil {
		cbDiffs = []FieldDiff{{Field: "_deleted", Label: "规则", OldValue: "已配置", NewValue: "已删除", Changed: true}}
	} else {
		cbDiffs = []FieldDiff{}
	}

	changedCount := 0
	for _, d := range flowDiffs {
		if d.Changed {
			changedCount++
		}
	}
	for _, d := range cbDiffs {
		if d.Changed {
			changedCount++
		}
	}

	c.JSON(http.StatusOK, appResp{Data: map[string]interface{}{
		"resource":      resourceName,
		"flow_diffs":    flowDiffs,
		"cb_diffs":      cbDiffs,
		"change_count":  changedCount,
		"has_published": publishedFlow != nil || publishedCB != nil,
	}})
}

// appResp is defined in list_app.go

// ListPublishRecords lists publish history
func ListPublishRecords(c *gin.Context) {
	appId := c.Query("app")

	if appId == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "app is required"})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	// 根据 app_id 从 business_line_apps 获取 app_key
	var appKey string
	err := mysqlDB.QueryRow("SELECT app_key FROM business_line_apps WHERE id = ?", appId).Scan(&appKey)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "App not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	records, err := publishDAOMy.ListRecords(appId, 20)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, appResp{Data: records})
}

// ListVersions lists publish versions
func ListVersions(c *gin.Context) {
	appId := c.Query("app")

	if appId == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "app is required"})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	// 根据 app_id 从 business_line_apps 获取 app_key
	var appKey string
	err := mysqlDB.QueryRow("SELECT app_key FROM business_line_apps WHERE id = ?", appId).Scan(&appKey)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "App not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	versions, err := versionDAOMy.ListVersions(appId, 20)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, appResp{Data: versions})
}

// GetVersion gets a version detail with snapshot
func GetVersion(c *gin.Context) {
	versionId := c.Param("id")
	if versionId == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "version id is required"})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	var id int64
	fmt.Sscanf(versionId, "%d", &id)
	version, err := versionDAOMy.GetVersion(id)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "Version not found: " + err.Error()})
		return
	}

	// Parse snapshot
	var snapshot map[string]interface{}
	json.Unmarshal([]byte(version.Snapshot), &snapshot)

	result := map[string]interface{}{
		"version":  version,
		"snapshot": snapshot,
	}

	c.JSON(http.StatusOK, appResp{Data: result})
}

// RollbackVersion rolls back to a specific version
func RollbackVersion(c *gin.Context) {
	versionId := c.Param("id")
	var req struct {
		AppID string `json:"app_key"` // 前端传的是 app_key，实际是 app_id

	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "Invalid request: " + err.Error()})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL connection failed: " + err.Error()})
		return
	}

	var id int64
	fmt.Sscanf(versionId, "%d", &id)

	// Get target version
	version, err := versionDAOMy.GetVersion(id)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "Version not found"})
		return
	}

	// Parse snapshot
	var snapshot dao.RuleSnapshot
	if err := json.Unmarshal([]byte(version.Snapshot), &snapshot); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "Failed to parse snapshot"})
		return
	}

	var appKey, settingsJSON, lineName string
	var businessLineID int64
	err = mysqlDB.QueryRow(`
		SELECT bla.app_key, bla.business_line_id, COALESCE(bla.settings, ''), bl.name
		FROM business_line_apps bla
		JOIN business_lines bl ON bla.business_line_id = bl.id
		WHERE bla.id = ?`, req.AppID).Scan(&appKey, &businessLineID, &settingsJSON, &lineName)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "App not found: " + err.Error()})
		return
	}

	client, cerr := etcdMgr.GetOrCreateClient(req.AppID, settingsJSON)
	if cerr != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: cerr.Error()})
		return
	}
	rollbackPublisher := provider.NewEtcdRulePublisher(client)
	rollbackPathBuilder := provider.NewEtcdPathBuilder()

	// Get version number for new record
	newVersionNum := 1
	if v, err := versionDAOMy.GetLatestVersionNumber(req.AppID); err == nil {
		newVersionNum = v + 1
	}

	// Begin MySQL transaction for all database operations
	tx, err := mysqlDB.Begin()
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "Failed to begin transaction: " + err.Error()})
		return
	}
	defer tx.Rollback() // Rollback if not committed

	// Step 1: Clear current rules (within transaction)
	if _, err := tx.Exec("DELETE FROM business_line_resource_flowrules WHERE app_id = ?", req.AppID); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "Failed to clear current flow rules: " + err.Error()})
		return
	}
	if _, err := tx.Exec("DELETE FROM business_line_resource_circuitbreakerrules WHERE app_id = ?", req.AppID); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "Failed to clear current circuit breaker rules: " + err.Error()})
		return
	}

	// Step 2: Restore rules from snapshot (within transaction)
	restored := 0
	for _, r := range snapshot.FlowRules {
		resource, _ := r["resource"].(string)
		if resource == "" {
			continue
		}
		// Look up resource_id from resource name
		var resourceID int64
		tx.QueryRow("SELECT id FROM business_line_resources WHERE app_id = ? AND name = ?",
			req.AppID, resource).Scan(&resourceID)
		if resourceID == 0 {
			continue
		}
		threshold := getFloat(r, "threshold")
		metricType := getInt(r, "metricType")
		controlBehavior := getInt(r, "controlBehavior")
		warmUpPeriodSec := getInt(r, "warmUpPeriodSec")
		maxQueueingTimeMs := getInt(r, "maxQueueingTimeMs")
		clusterMode := r["clusterMode"].(bool)
		tokenCalculateStrategy := getInt(r, "tokenCalculateStrategy")
		relationStrategy := getInt(r, "relationStrategy")
		refResource, _ := r["refResource"].(string)
		warmUpColdFactor := getInt(r, "warmUpColdFactor")
		statIntervalMs := getInt(r, "statIntervalMs")

		_, err := tx.Exec(`INSERT INTO business_line_resource_flowrules
			(app_id, resource_id, threshold, metric_type, control_behavior,
			 warm_up_period_sec, max_queueing_time_ms, cluster_mode, cluster_config,
			 token_calculate_strategy, relation_strategy, ref_resource, warm_up_cold_factor,
			 stat_interval_ms, enabled)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				threshold=VALUES(threshold), metric_type=VALUES(metric_type), control_behavior=VALUES(control_behavior),
				warm_up_period_sec=VALUES(warm_up_period_sec), max_queueing_time_ms=VALUES(max_queueing_time_ms),
				cluster_mode=VALUES(cluster_mode), cluster_config=VALUES(cluster_config),
				token_calculate_strategy=VALUES(token_calculate_strategy), relation_strategy=VALUES(relation_strategy),
				ref_resource=VALUES(ref_resource), warm_up_cold_factor=VALUES(warm_up_cold_factor),
				stat_interval_ms=VALUES(stat_interval_ms), enabled=VALUES(enabled), updated_at=NOW()`,
			req.AppID, resourceID, threshold, metricType, controlBehavior,
			warmUpPeriodSec, maxQueueingTimeMs, clusterMode, nil,
			tokenCalculateStrategy, relationStrategy, refResource,
			warmUpColdFactor, statIntervalMs, true)
		if err == nil {
			restored++
		}
	}

	for _, r := range snapshot.CBRules {
		resource, _ := r["resource"].(string)
		if resource == "" {
			continue
		}
		// Look up resource_id from resource name
		var resourceID int64
		tx.QueryRow("SELECT id FROM business_line_resources WHERE app_id = ? AND name = ?",
			req.AppID, resource).Scan(&resourceID)
		if resourceID == 0 {
			continue
		}
		strategy := getInt(r, "strategy")
		threshold := getFloat(r, "threshold")
		retryTimeoutMs := int64(getInt(r, "retryTimeoutMs"))
		minRequestAmount := getInt(r, "minRequestAmount")
		statIntervalMs := getInt(r, "statIntervalMs")
		statSlidingWindowBucketCount := getInt(r, "statSlidingWindowBucketCount")
		maxAllowedRtMs := int64(getInt(r, "maxAllowedRtMs"))
		probeNum := getInt(r, "probeNum")

		_, err := tx.Exec(`INSERT INTO business_line_resource_circuitbreakerrules
			(app_id, resource_id, strategy, threshold, retry_timeout_ms,
			 min_request_amount, stat_interval_ms, stat_sliding_window_bucket_count, max_allowed_rt_ms, probe_num, enabled)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				strategy=VALUES(strategy), threshold=VALUES(threshold), retry_timeout_ms=VALUES(retry_timeout_ms),
				min_request_amount=VALUES(min_request_amount), stat_interval_ms=VALUES(stat_interval_ms),
				stat_sliding_window_bucket_count=VALUES(stat_sliding_window_bucket_count),
				max_allowed_rt_ms=VALUES(max_allowed_rt_ms), probe_num=VALUES(probe_num),
				enabled=VALUES(enabled), updated_at=NOW()`,
			req.AppID, resourceID, strategy, threshold, retryTimeoutMs,
			minRequestAmount, statIntervalMs, statSlidingWindowBucketCount,
			maxAllowedRtMs, probeNum, true)
		if err == nil {
			restored++
		}
	}

	// Create rollback version record (within transaction)
	rollbackSnapshot, _ := json.Marshal(snapshot)
	if _, err := tx.Exec(`INSERT INTO publish_versions 
		(app_id, version_number, description, operator, rule_count, snapshot, status, error_msg)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		req.AppID, newVersionNum,
		fmt.Sprintf("回滚到 v%d（恢复 %d 条规则）", version.VersionNumber, restored),
		"user", restored, string(rollbackSnapshot), "success", ""); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "Failed to create version record: " + err.Error()})
		return
	}

	// Record publish (within transaction)
	if _, err := tx.Exec(`INSERT INTO publish_records (app_id, rule_type, resource, rule_count, status, error_msg, operator)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		req.AppID, "all", "", restored, "success", "", "user"); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "Failed to create publish record: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "Failed to commit transaction: " + err.Error()})
		return
	}

	groupMap := make(map[int64]string)
	gRows, _ := mysqlDB.Query("SELECT id, name FROM business_line_app_groups WHERE app_id = ?", req.AppID)
	if gRows != nil {
		defer gRows.Close()
		for gRows.Next() {
			var gid int64
			var gname string
			gRows.Scan(&gid, &gname)
			groupMap[gid] = gname
		}
	}

	resourceGroupMap := make(map[int64]string)
	rgRows, _ := mysqlDB.Query("SELECT id, COALESCE(group_id, 0) FROM business_line_resources WHERE app_id = ?", req.AppID)
	if rgRows != nil {
		defer rgRows.Close()
		for rgRows.Next() {
			var rid, gid int64
			rgRows.Scan(&rid, &gid)
			if gname, ok := groupMap[gid]; ok {
				resourceGroupMap[rid] = gname
			} else {
				resourceGroupMap[rid] = "default"
			}
		}
	}

	type rollbackRuleKey struct {
		group    string
		resource string
	}

	flowRules, _ := flowRuleDAOMy.ListRules(req.AppID)
	flowGrouped := make(map[rollbackRuleKey][]interface{})
	for _, r := range flowRules {
		if !r.Enabled {
			continue
		}
		resourceName := getResourceNameByID(r.ResourceID)
		groupName := resourceGroupMap[r.ResourceID]
		if groupName == "" {
			groupName = "default"
		}
		rule := map[string]interface{}{
			"resource": resourceName, "threshold": r.Threshold,
			"metricType": r.MetricType, "controlBehavior": r.ControlBehavior,
			"warmUpPeriodSec": r.WarmUpPeriodSec, "maxQueueingTimeMs": r.MaxQueueingTimeMs,
			"tokenCalculateStrategy": r.TokenCalculateStrategy, "relationStrategy": r.RelationStrategy,
			"refResource": r.RefResource, "warmUpColdFactor": r.WarmUpColdFactor,
			"statIntervalInMs": r.StatIntervalMs, "clusterMode": r.ClusterMode,
		}
		k := rollbackRuleKey{group: groupName, resource: resourceName}
		flowGrouped[k] = append(flowGrouped[k], rule)
	}
	for k, rr := range flowGrouped {
		data, _ := json.Marshal(rr)
		rollbackPublisher.PublishRules(rollbackPathBuilder.BuildPath(lineName, appKey, k.group, k.resource, "flow"), data)
	}

	cbRules, _ := cbRuleDAOMy.ListRules(req.AppID)
	cbGrouped := make(map[rollbackRuleKey][]interface{})
	for _, r := range cbRules {
		if !r.Enabled {
			continue
		}
		resourceName := getResourceNameByID(r.ResourceID)
		groupName := resourceGroupMap[r.ResourceID]
		if groupName == "" {
			groupName = "default"
		}
		rule := map[string]interface{}{
			"resource": resourceName, "strategy": r.Strategy, "threshold": r.Threshold,
			"retryTimeoutMs": r.RetryTimeoutMs, "minRequestAmount": r.MinRequestAmount,
			"statIntervalMs": r.StatIntervalMs, "statSlidingWindowBucketCount": r.StatSlidingWindowBucketCount,
			"maxAllowedRtMs": r.MaxAllowedRtMs, "probeNum": r.ProbeNum,
		}
		k := rollbackRuleKey{group: groupName, resource: resourceName}
		cbGrouped[k] = append(cbGrouped[k], rule)
	}
	for k, rr := range cbGrouped {
		data, _ := json.Marshal(rr)
		rollbackPublisher.PublishRules(rollbackPathBuilder.BuildPath(lineName, appKey, k.group, k.resource, "circuitbreaker"), data)
	}

	if svc, err := GetAuthService(); err == nil {
		if u, exists := c.Get("user"); exists {
			user := u.(*model.User)
			svc.LogAudit(user.UserID, "rollback", "version", versionId,
				fmt.Sprintf("rollback to v%d, restored %d rules", version.VersionNumber, restored), c.ClientIP())
		}
	}

	c.JSON(http.StatusOK, appResp{Msg: fmt.Sprintf("已回滚到 v%d，恢复 %d 条规则并发布到 etcd", version.VersionNumber, restored)})
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}
