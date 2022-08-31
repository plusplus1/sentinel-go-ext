package app

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/plusplus1/sentinel-go-ext/dashboard/dao"
	"github.com/plusplus1/sentinel-go-ext/dashboard/model"
)

// ListCircuitbreakerRules lists circuit breaker rules (MySQL-backed)
func ListCircuitbreakerRules(c *gin.Context) {
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
	rules, err := cbRuleDAOMy.ListRules(appId, resourceID)
	if err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	var result []map[string]interface{}
	for _, r := range rules {
		resourceName := getResourceNameByID(r.ResourceID)
		result = append(result, map[string]interface{}{
			"id":                           r.ID,
			"resource_id":                  r.ResourceID,
			"resource":                     resourceName,
			"strategy":                     r.Strategy,
			"threshold":                    r.Threshold,
			"retryTimeoutMs":               r.RetryTimeoutMs,
			"minRequestAmount":             r.MinRequestAmount,
			"statIntervalMs":               r.StatIntervalMs,
			"statSlidingWindowBucketCount": r.StatSlidingWindowBucketCount,
			"maxAllowedRtMs":               r.MaxAllowedRtMs,
			"probeNum":                     r.ProbeNum,
		})
	}

	c.JSON(http.StatusOK, appResp{Data: result})
}

// SaveOrUpdateCircuitbreakerRule creates or updates a CB rule (MySQL-backed)
func SaveOrUpdateCircuitbreakerRule(c *gin.Context) {
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

	rule := &dao.CBRuleRecord{
		ID:                           ruleID,
		AppID:                        appId,
		ResourceID:                   resourceID,
		Strategy:                     intVal(req, "strategy"),
		Threshold:                    floatVal(req, "threshold"),
		RetryTimeoutMs:               int64Val(req, "retryTimeoutMs"),
		MinRequestAmount:             intVal(req, "minRequestAmount"),
		StatIntervalMs:               intVal(req, "statIntervalMs"),
		StatSlidingWindowBucketCount: intVal(req, "statSlidingWindowBucketCount"),
		MaxAllowedRtMs:               int64Val(req, "maxAllowedRtMs"),
		ProbeNum:                     intVal(req, "probeNum"),
		Enabled:                      true,
	}

	if err := cbRuleDAOMy.CreateOrUpdateRule(rule); err != nil {
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
			svc.LogAudit(user.UserID, action, "cb_rule", fmt.Sprintf("%d", resourceID),
				fmt.Sprintf("strategy=%d, threshold=%.1f", rule.Strategy, rule.Threshold), c.ClientIP())
		}
	}

	c.JSON(http.StatusOK, appResp{Msg: "更新成功"})
}

// DeleteCircuitbreakerRule deletes a CB rule (MySQL-backed)
func DeleteCircuitbreakerRule(c *gin.Context) {
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

	if err := cbRuleDAOMy.DeleteRule(ruleID); err != nil {
		c.JSON(http.StatusOK, appResp{Code: 999, Msg: err.Error()})
		return
	}

	// Audit log
	if svc, err := GetAuthService(); err == nil {
		if u, exists := c.Get("user"); exists {
			user := u.(*model.User)
			svc.LogAudit(user.UserID, "rule_delete", "cb_rule", fmt.Sprintf("%d", ruleID), "", c.ClientIP())
		}
	}

	c.JSON(http.StatusOK, appResp{Msg: "删除成功"})
}

func floatVal(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func int64Val(m map[string]interface{}, key string) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	return 0
}
