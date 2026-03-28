package app

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/plusplus1/sentinel-go-ext/dashboard/dao"
	"github.com/plusplus1/sentinel-go-ext/dashboard/model"
)

// ListFlowRules lists flow rules (MySQL-backed)
func ListFlowRules(c *gin.Context) {
	appId := c.Query("app")
	if appId == "" {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "app is required"})
		return
	}

	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL not available: " + err.Error()})
		return
	}

	resourceIDStr := c.Query("res")
	var resourceID int64
	if resourceIDStr != "" {
		rid, err := strconv.ParseInt(resourceIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusOK, appResp{Code: 100, Msg: "invalid resource_id"})
			return
		}
		resourceID = rid
	}
	rules, err := flowRuleDAOMy.ListRules(appId, resourceID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	// Convert to legacy format
	var result []map[string]interface{}
	for _, r := range rules {
		resourceName := getResourceNameByID(r.ResourceID)
		result = append(result, map[string]interface{}{
			"id":                     r.ID,
			"resource_id":            r.ResourceID,
			"resource":               resourceName,
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
			"lowMemUsageThreshold":   0.8,
			"highMemUsageThreshold":  0.9,
			"memLowWaterMarkBytes":   0,
			"memHighWaterMarkBytes":  0,
		})
	}

	c.JSON(http.StatusOK, appResp{Data: result})
}

// SaveOrUpdateFlowRule creates or updates a flow rule (MySQL-backed)
func SaveOrUpdateFlowRule(c *gin.Context) {
	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL not available: " + err.Error()})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "参数错误"})
		return
	}

	appId, _ := req["appId"].(string)
	ruleIDFloat, _ := req["id"].(float64)
	ruleID := int64(ruleIDFloat)
	resourceIDFloat, _ := req["resource_id"].(float64)
	resourceID := int64(resourceIDFloat)
	if appId == "" || resourceID == 0 {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "参数错误"})
		return
	}

	threshold, _ := req["threshold"].(float64)
	controlBehavior := intVal(req, "controlBehavior")
	metricType := intVal(req, "metricType")
	warmUpPeriodSec := intVal(req, "warmUpPeriodSec")
	maxQueueingTimeMs := intVal(req, "maxQueueingTimeMs")
	clusterMode, _ := req["clusterMode"].(bool)
	tokenCalculateStrategy := intVal(req, "tokenCalculateStrategy")
	relationStrategy := intVal(req, "relationStrategy")
	refResource, _ := req["refResource"].(string)
	warmUpColdFactor := intVal(req, "warmUpColdFactor")
	if warmUpColdFactor == 0 {
		warmUpColdFactor = 3
	}
	statIntervalInMs := intVal(req, "statIntervalInMs")
	if statIntervalInMs == 0 {
		statIntervalInMs = 1000
	}

	rule := &dao.FlowRuleRecord{
		ID:                     ruleID,
		AppID:                  appId,
		ResourceID:             resourceID,
		Threshold:              threshold,
		MetricType:             metricType,
		ControlBehavior:        controlBehavior,
		WarmUpPeriodSec:        warmUpPeriodSec,
		MaxQueueingTimeMs:      maxQueueingTimeMs,
		ClusterMode:            clusterMode,
		TokenCalculateStrategy: tokenCalculateStrategy,
		RelationStrategy:       relationStrategy,
		RefResource:            refResource,
		WarmUpColdFactor:       warmUpColdFactor,
		StatIntervalMs:         statIntervalInMs,
		Enabled:                true,
	}

	if err := flowRuleDAOMy.CreateOrUpdateRule(rule); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	// Audit log
	if svc, err := GetAuthService(); err == nil {
		if u, exists := c.Get("user"); exists {
			user := u.(*model.User)
			action := "rule_create"
			if ruleID > 0 {
				action = "rule_update"
			}
			svc.LogAudit(user.UserID, action, "flow_rule", fmt.Sprintf("%d", resourceID),
				fmt.Sprintf("threshold=%.1f, metricType=%d, controlBehavior=%d", threshold, metricType, controlBehavior), c.ClientIP())
		}
	}

	c.JSON(http.StatusOK, appResp{Msg: "更新成功"})
}

// DeleteFlowRule deletes a flow rule (MySQL-backed)
func DeleteFlowRule(c *gin.Context) {
	if _, err := getMySQLDB(); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: "MySQL not available: " + err.Error()})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "参数错误"})
		return
	}

	ruleIDFloat, _ := req["id"].(float64)
	ruleID := int64(ruleIDFloat)
	if ruleID == 0 {
		c.JSON(http.StatusOK, appResp{Code: 100, Msg: "参数错误"})
		return
	}

	if err := flowRuleDAOMy.DeleteRule(ruleID); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	// Audit log
	if svc, err := GetAuthService(); err == nil {
		if u, exists := c.Get("user"); exists {
			user := u.(*model.User)
			svc.LogAudit(user.UserID, "rule_delete", "flow_rule", fmt.Sprintf("%d", ruleID), "", c.ClientIP())
		}
	}

	c.JSON(http.StatusOK, appResp{Msg: "删除成功"})
}

// intVal extracts int from map[string]interface{} (handles float64 from JSON)
func intVal(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}
