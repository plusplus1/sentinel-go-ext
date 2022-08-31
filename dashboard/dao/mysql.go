package dao

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/plusplus1/sentinel-go-ext/dashboard/config"
	"github.com/plusplus1/sentinel-go-ext/dashboard/model"
)

// MySQLConfig holds MySQL connection configuration
type MySQLConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

// DefaultMySQLConfig returns MySQL configuration from settings file, or default if not configured
func DefaultMySQLConfig() MySQLConfig {
	// Get settings from configuration file
	settings := config.AppSettings()
	mysqlCfg := settings.MySQL

	// Use values from config if available, otherwise use defaults
	cfg := MySQLConfig{
		Host: "127.0.0.1",
		Port: 3306,
	}

	// Override with config values if provided
	if mysqlCfg.Host != "" {
		cfg.Host = mysqlCfg.Host
	}
	if mysqlCfg.Port != 0 {
		cfg.Port = mysqlCfg.Port
	}
	if mysqlCfg.User != "" {
		cfg.User = mysqlCfg.User
	}
	if mysqlCfg.Password != "" {
		cfg.Password = mysqlCfg.Password
	}
	if mysqlCfg.Database != "" {
		cfg.Database = mysqlCfg.Database
	}

	return cfg
}

// NewMySQLDB creates a new MySQL database connection
func NewMySQLDB(cfg MySQLConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping mysql: %v", err)
	}

	return db, nil
}

// ============================================
// Group DAO (MySQL)
// ============================================

type MySQLGroupDAO struct {
	db *sql.DB
}

func NewMySQLGroupDAO(db *sql.DB) *MySQLGroupDAO {
	return &MySQLGroupDAO{db: db}
}

func (d *MySQLGroupDAO) CreateGroup(group *model.Group) error {
	query := `INSERT INTO ` + "business_line_app_groups" + ` (app_id, name, description, is_default) VALUES (?, ?, ?, ?)`
	result, err := d.db.Exec(query, group.AppID, group.Name, group.Description, group.IsDefault)
	if err != nil {
		return fmt.Errorf("failed to create group: %v", err)
	}
	id, _ := result.LastInsertId()
	group.ID = fmt.Sprintf("%d", id)
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()
	return nil
}

func (d *MySQLGroupDAO) GetGroup(appID, groupID string) (*model.Group, error) {
	query := `SELECT id, app_id, name, description, is_default, created_at, updated_at 
			  FROM ` + "business_line_app_groups" + ` WHERE id = ? AND app_id = ?`
	var g model.Group
	var id int64
	err := d.db.QueryRow(query, groupID, appID).Scan(
		&id, &g.AppID, &g.Name, &g.Description, &g.IsDefault, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("group not found: %v", err)
	}
	g.ID = fmt.Sprintf("%d", id)

	// Get member count
	d.db.QueryRow("SELECT COUNT(*) FROM business_line_resources WHERE group_id = ?", id).Scan(&g.MemberCount)

	return &g, nil
}

func (d *MySQLGroupDAO) GetGroupByName(appID, name string) (*model.Group, error) {
	query := `SELECT id, app_id, name, description, is_default, created_at, updated_at 
			  FROM ` + "business_line_app_groups" + ` WHERE app_id = ? AND name = ?`
	var g model.Group
	var id int64
	err := d.db.QueryRow(query, appID, name).Scan(
		&id, &g.AppID, &g.Name, &g.Description, &g.IsDefault, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, err
	}
	g.ID = fmt.Sprintf("%d", id)
	return &g, nil
}

func (d *MySQLGroupDAO) UpdateGroup(groupID string, group *model.Group) error {
	query := `UPDATE ` + "business_line_app_groups" + ` SET description = ?, updated_at = NOW() WHERE id = ?`
	_, err := d.db.Exec(query, group.Description, groupID)
	if err != nil {
		return fmt.Errorf("failed to update group: %v", err)
	}
	return nil
}

func (d *MySQLGroupDAO) DeleteGroup(groupID string) error {
	_, err := d.db.Exec("DELETE FROM business_line_app_groups WHERE id = ?", groupID)
	return err
}

func (d *MySQLGroupDAO) ListGroups(appID string) ([]*model.Group, error) {
	query := `SELECT id, app_id, name, description, is_default, created_at, updated_at 
			  FROM ` + "business_line_app_groups" + ` WHERE app_id = ? ORDER BY created_at ASC`
	rows, err := d.db.Query(query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*model.Group
	for rows.Next() {
		var g model.Group
		var id int64
		if err := rows.Scan(&id, &g.AppID, &g.Name, &g.Description, &g.IsDefault, &g.CreatedAt, &g.UpdatedAt); err != nil {
			continue
		}
		g.ID = fmt.Sprintf("%d", id)
		d.db.QueryRow("SELECT COUNT(*) FROM business_line_resources WHERE group_id = ?", id).Scan(&g.MemberCount)
		groups = append(groups, &g)
	}
	return groups, nil
}

func (d *MySQLGroupDAO) GetOrCreateDefaultGroup(appID string) (*model.Group, error) {
	g, err := d.GetGroupByName(appID, "默认模块")
	if err == nil {
		return g, nil
	}
	// Create default group
	defaultGroup := &model.Group{
		AppID: appID,

		Name:        "默认模块",
		Description: "未分配到任何模块的资源自动归属此模块",
		IsDefault:   true,
	}
	if err := d.CreateGroup(defaultGroup); err != nil {
		return nil, err
	}
	return defaultGroup, nil
}

func (d *MySQLGroupDAO) AddResourceToGroup(appID, groupID, resourceName string) error {
	// Update resource's group_id
	_, err := d.db.Exec("UPDATE business_line_resources SET group_id = ? WHERE app_id = ? AND name = ?",
		groupID, appID, resourceName)
	return err
}

func (d *MySQLGroupDAO) RemoveResourceFromGroup(appID, groupID, resourceName string) error {
	_, err := d.db.Exec("UPDATE business_line_resources SET group_id = NULL WHERE app_id = ? AND name = ? AND group_id = ?",
		appID, resourceName, groupID)
	return err
}

// ============================================
// Resource DAO (MySQL)
// ============================================

type MySQLResourceDAO struct {
	db *sql.DB
}

func NewMySQLResourceDAO(db *sql.DB) *MySQLResourceDAO {
	return &MySQLResourceDAO{db: db}
}

func (d *MySQLResourceDAO) GetOrCreateResource(appID, resourceName string) (*model.Resource, error) {
	r, err := d.GetResource(appID, resourceName)
	if err == nil {
		return r, nil
	}
	// Create resource
	res := &model.Resource{
		Name:        resourceName,
		Description: "",
		AppID:       appID,
	}
	query := `INSERT INTO business_line_resources (app_id, name, description, group_id) VALUES (?, ?, '', NULL)`
	result, err := d.db.Exec(query, appID, resourceName)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	res.ID = fmt.Sprintf("%d", id)
	res.CreatedAt = time.Now()
	res.UpdatedAt = time.Now()
	return res, nil
}

func (d *MySQLResourceDAO) GetResource(appID, resourceName string) (*model.Resource, error) {
	query := `SELECT r.id, r.app_id, r.name, r.description, r.group_id, r.created_at, r.updated_at,
			  COALESCE(g.name, '') as group_name
			  FROM business_line_resources r
			  LEFT JOIN ` + "business_line_app_groups" + ` g ON r.group_id = g.id
			  WHERE r.app_id = ? AND r.name = ?`
	var r model.Resource
	var id int64
	var groupID sql.NullInt64
	var groupName string
	err := d.db.QueryRow(query, appID, resourceName).Scan(
		&id, &r.AppID, &r.Name, &r.Description, &groupID, &r.CreatedAt, &r.UpdatedAt, &groupName)
	if err != nil {
		return nil, err
	}
	r.ID = fmt.Sprintf("%d", id)
	r.GroupName = groupName
	if groupID.Valid {
		gid := fmt.Sprintf("%d", groupID.Int64)
		r.GroupID = &gid
	}
	return &r, nil
}

// GetResourceByID queries a resource by its primary ID
func (d *MySQLResourceDAO) GetResourceByID(id string) (*model.Resource, error) {
	query := `SELECT r.id, r.app_id, r.name, r.description, r.group_id, r.created_at, r.updated_at,
			  COALESCE(g.name, '') as group_name
			  FROM business_line_resources r
			  LEFT JOIN ` + "business_line_app_groups" + ` g ON r.group_id = g.id
			  WHERE r.id = ?`
	var r model.Resource
	var rid int64
	var groupID sql.NullInt64
	var groupName string
	err := d.db.QueryRow(query, id).Scan(
		&rid, &r.AppID, &r.Name, &r.Description, &groupID, &r.CreatedAt, &r.UpdatedAt, &groupName)
	if err != nil {
		return nil, err
	}
	r.ID = fmt.Sprintf("%d", rid)
	r.GroupName = groupName
	if groupID.Valid {
		gid := fmt.Sprintf("%d", groupID.Int64)
		r.GroupID = &gid
	}
	return &r, nil
}

func (d *MySQLResourceDAO) ListResources(appID string) ([]*model.Resource, error) {
	query := `SELECT r.id, r.app_id, r.name, r.description, r.group_id, r.created_at, r.updated_at,
		  COALESCE(g.name, '') as group_name, COALESCE(g.description, '') as group_description
		  FROM business_line_resources r
		  LEFT JOIN ` + "business_line_app_groups" + ` g ON r.group_id = g.id
		  WHERE r.app_id = ?
		  ORDER BY r.created_at DESC`
	rows, err := d.db.Query(query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []*model.Resource
	for rows.Next() {
		var r model.Resource
		var id int64
		var groupID sql.NullInt64
		var groupName string
		var groupDescription string
		if err := rows.Scan(&id, &r.AppID, &r.Name, &r.Description, &groupID, &r.CreatedAt, &r.UpdatedAt, &groupName, &groupDescription); err != nil {
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
	return resources, nil
}

func (d *MySQLResourceDAO) UpdateResourceGroup(appID, resourceName, groupID string) error {
	_, err := d.db.Exec("UPDATE business_line_resources SET group_id = ? WHERE app_id = ? AND name = ?",
		groupID, appID, resourceName)
	return err
}

// UpdateResourceDescription updates the description of a resource
func (d *MySQLResourceDAO) UpdateResourceDescription(appID, resourceName, description string) error {
	_, err := d.db.Exec("UPDATE business_line_resources SET description = ? WHERE app_id = ? AND name = ?",
		description, appID, resourceName)
	return err
}

func (d *MySQLResourceDAO) DeleteResource(appID, resourceName string) error {
	var resourceID int64
	d.db.QueryRow("SELECT id FROM business_line_resources WHERE app_id = ? AND name = ?",
		appID, resourceName).Scan(&resourceID)
	if resourceID > 0 {
		d.db.Exec("DELETE FROM business_line_resource_flowrules WHERE resource_id = ?", resourceID)
		d.db.Exec("DELETE FROM business_line_resource_circuitbreakerrules WHERE resource_id = ?", resourceID)
	}
	_, err := d.db.Exec("DELETE FROM business_line_resources WHERE app_id = ? AND name = ?",
		appID, resourceName)
	return err
}

// ============================================
// Flow Rule DAO (MySQL)
// ============================================

type MySQLFlowRuleDAO struct {
	db *sql.DB
}

func NewMySQLFlowRuleDAO(db *sql.DB) *MySQLFlowRuleDAO {
	return &MySQLFlowRuleDAO{db: db}
}

func (d *MySQLFlowRuleDAO) CreateOrUpdateRule(rule *FlowRuleRecord) error {
	if rule.ID > 0 {
		query := `UPDATE business_line_resource_flowrules SET
			threshold=?, metric_type=?, control_behavior=?, warm_up_period_sec=?,
			max_queueing_time_ms=?, cluster_mode=?, cluster_config=?,
			token_calculate_strategy=?, relation_strategy=?, ref_resource=?,
			warm_up_cold_factor=?, stat_interval_ms=?, enabled=?, updated_at=NOW()
			WHERE id=?`
		_, err := d.db.Exec(query,
			rule.Threshold, rule.MetricType, rule.ControlBehavior,
			rule.WarmUpPeriodSec, rule.MaxQueueingTimeMs, rule.ClusterMode, rule.ClusterConfig,
			rule.TokenCalculateStrategy, rule.RelationStrategy, rule.RefResource,
			rule.WarmUpColdFactor, rule.StatIntervalMs, rule.Enabled,
			rule.ID)
		return err
	}

	query := `INSERT INTO business_line_resource_flowrules
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
			stat_interval_ms=VALUES(stat_interval_ms), enabled=VALUES(enabled), updated_at=NOW()`
	result, err := d.db.Exec(query,
		rule.AppID, rule.ResourceID,
		rule.Threshold, rule.MetricType, rule.ControlBehavior,
		rule.WarmUpPeriodSec, rule.MaxQueueingTimeMs, rule.ClusterMode, rule.ClusterConfig,
		rule.TokenCalculateStrategy, rule.RelationStrategy, rule.RefResource,
		rule.WarmUpColdFactor, rule.StatIntervalMs, rule.Enabled)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	rule.ID = id
	return nil
}

func (d *MySQLFlowRuleDAO) DeleteRule(id int64) error {
	_, err := d.db.Exec("DELETE FROM business_line_resource_flowrules WHERE id = ?", id)
	return err
}

func (d *MySQLFlowRuleDAO) GetRule(id int64) (*FlowRuleRecord, error) {
	query := `SELECT id, app_id, resource_id, threshold, metric_type, control_behavior,
			  warm_up_period_sec, max_queueing_time_ms, cluster_mode, cluster_config,
			  token_calculate_strategy, relation_strategy, ref_resource, warm_up_cold_factor,
			  stat_interval_ms, enabled, created_at, updated_at
			  FROM business_line_resource_flowrules WHERE id = ?`
	var r FlowRuleRecord
	err := d.db.QueryRow(query, id).Scan(
		&r.ID, &r.AppID, &r.ResourceID,
		&r.Threshold, &r.MetricType, &r.ControlBehavior,
		&r.WarmUpPeriodSec, &r.MaxQueueingTimeMs, &r.ClusterMode, &r.ClusterConfig,
		&r.TokenCalculateStrategy, &r.RelationStrategy, &r.RefResource,
		&r.WarmUpColdFactor, &r.StatIntervalMs, &r.Enabled,
		&r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (d *MySQLFlowRuleDAO) ListRules(appID string, resourceID ...int64) ([]*FlowRuleRecord, error) {
	var query string
	var args []interface{}

	if len(resourceID) > 0 && resourceID[0] > 0 {
		query = `SELECT id, app_id, resource_id, threshold, metric_type, control_behavior,
				 warm_up_period_sec, max_queueing_time_ms, cluster_mode, cluster_config,
				 token_calculate_strategy, relation_strategy, ref_resource, warm_up_cold_factor,
				 stat_interval_ms, enabled, created_at, updated_at
				 FROM business_line_resource_flowrules WHERE app_id = ? AND resource_id = ? ORDER BY created_at DESC`
		args = []interface{}{appID, resourceID[0]}
	} else {
		query = `SELECT id, app_id, resource_id, threshold, metric_type, control_behavior,
				 warm_up_period_sec, max_queueing_time_ms, cluster_mode, cluster_config,
				 token_calculate_strategy, relation_strategy, ref_resource, warm_up_cold_factor,
				 stat_interval_ms, enabled, created_at, updated_at
				 FROM business_line_resource_flowrules WHERE app_id = ? ORDER BY created_at DESC`
		args = []interface{}{appID}
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*FlowRuleRecord
	for rows.Next() {
		var r FlowRuleRecord
		if err := rows.Scan(
			&r.ID, &r.AppID, &r.ResourceID,
			&r.Threshold, &r.MetricType, &r.ControlBehavior,
			&r.WarmUpPeriodSec, &r.MaxQueueingTimeMs, &r.ClusterMode, &r.ClusterConfig,
			&r.TokenCalculateStrategy, &r.RelationStrategy, &r.RefResource,
			&r.WarmUpColdFactor, &r.StatIntervalMs, &r.Enabled,
			&r.CreatedAt, &r.UpdatedAt); err != nil {
			continue
		}
		rules = append(rules, &r)
	}
	return rules, nil
}

func (d *MySQLFlowRuleDAO) ToggleRule(id int64, enabled bool) error {
	_, err := d.db.Exec("UPDATE business_line_resource_flowrules SET enabled = ?, updated_at = NOW() WHERE id = ?", enabled, id)
	return err
}

// ============================================
// Circuit Breaker Rule DAO (MySQL)
// ============================================

type MySQLCBRuleDAO struct {
	db *sql.DB
}

func NewMySQLCBRuleDAO(db *sql.DB) *MySQLCBRuleDAO {
	return &MySQLCBRuleDAO{db: db}
}

func (d *MySQLCBRuleDAO) CreateOrUpdateRule(rule *CBRuleRecord) error {
	if rule.ID > 0 {
		query := `UPDATE business_line_resource_circuitbreakerrules SET
			strategy=?, threshold=?, retry_timeout_ms=?,
			min_request_amount=?, stat_interval_ms=?, stat_sliding_window_bucket_count=?,
			max_allowed_rt_ms=?, probe_num=?, enabled=?, updated_at=NOW()
			WHERE id=?`
		_, err := d.db.Exec(query,
			rule.Strategy, rule.Threshold, rule.RetryTimeoutMs,
			rule.MinRequestAmount, rule.StatIntervalMs, rule.StatSlidingWindowBucketCount,
			rule.MaxAllowedRtMs, rule.ProbeNum, rule.Enabled,
			rule.ID)
		return err
	}

	query := `INSERT INTO business_line_resource_circuitbreakerrules
		(app_id, resource_id, strategy, threshold, retry_timeout_ms,
		 min_request_amount, stat_interval_ms, stat_sliding_window_bucket_count, max_allowed_rt_ms, probe_num, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			strategy=VALUES(strategy), threshold=VALUES(threshold), retry_timeout_ms=VALUES(retry_timeout_ms),
			min_request_amount=VALUES(min_request_amount), stat_interval_ms=VALUES(stat_interval_ms),
			stat_sliding_window_bucket_count=VALUES(stat_sliding_window_bucket_count),
			max_allowed_rt_ms=VALUES(max_allowed_rt_ms), probe_num=VALUES(probe_num),
			enabled=VALUES(enabled), updated_at=NOW()`
	result, err := d.db.Exec(query,
		rule.AppID, rule.ResourceID,
		rule.Strategy, rule.Threshold, rule.RetryTimeoutMs,
		rule.MinRequestAmount, rule.StatIntervalMs, rule.StatSlidingWindowBucketCount,
		rule.MaxAllowedRtMs, rule.ProbeNum, rule.Enabled)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	rule.ID = id
	return nil
}

func (d *MySQLCBRuleDAO) DeleteRule(id int64) error {
	_, err := d.db.Exec("DELETE FROM business_line_resource_circuitbreakerrules WHERE id = ?", id)
	return err
}

func (d *MySQLCBRuleDAO) ListRules(appID string, resourceID ...int64) ([]*CBRuleRecord, error) {
	var query string
	var args []interface{}

	if len(resourceID) > 0 && resourceID[0] > 0 {
		query = `SELECT id, app_id, resource_id, strategy, threshold, retry_timeout_ms,
				 min_request_amount, stat_interval_ms, stat_sliding_window_bucket_count, max_allowed_rt_ms, probe_num, enabled,
				 created_at, updated_at
				 FROM business_line_resource_circuitbreakerrules WHERE app_id = ? AND resource_id = ?
				 ORDER BY created_at DESC`
		args = []interface{}{appID, resourceID[0]}
	} else {
		query = `SELECT id, app_id, resource_id, strategy, threshold, retry_timeout_ms,
				 min_request_amount, stat_interval_ms, stat_sliding_window_bucket_count, max_allowed_rt_ms, probe_num, enabled,
				 created_at, updated_at
				 FROM business_line_resource_circuitbreakerrules WHERE app_id = ?
				 ORDER BY created_at DESC`
		args = []interface{}{appID}
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*CBRuleRecord
	for rows.Next() {
		var r CBRuleRecord
		if err := rows.Scan(
			&r.ID, &r.AppID, &r.ResourceID,
			&r.Strategy, &r.Threshold, &r.RetryTimeoutMs,
			&r.MinRequestAmount, &r.StatIntervalMs, &r.StatSlidingWindowBucketCount,
			&r.MaxAllowedRtMs, &r.ProbeNum, &r.Enabled,
			&r.CreatedAt, &r.UpdatedAt); err != nil {
			continue
		}
		rules = append(rules, &r)
	}
	return rules, nil
}

func (d *MySQLCBRuleDAO) ToggleRule(id int64, enabled bool) error {
	_, err := d.db.Exec("UPDATE business_line_resource_circuitbreakerrules SET enabled = ?, updated_at = NOW() WHERE id = ?", enabled, id)
	return err
}

// ============================================
// Publish Record DAO (MySQL)
// ============================================

type MySQLPublishRecordDAO struct {
	db *sql.DB
}

func NewMySQLPublishRecordDAO(db *sql.DB) *MySQLPublishRecordDAO {
	return &MySQLPublishRecordDAO{db: db}
}

func (d *MySQLPublishRecordDAO) CreateRecord(record *PublishRecord) error {
	query := `INSERT INTO publish_records (app_id, rule_type, resource, rule_count, status, error_msg, operator)
			  VALUES (?, ?, ?, ?, ?, ?, ?)`
	result, err := d.db.Exec(query, record.AppID, record.RuleType,
		record.Resource, record.RuleCount, record.Status, record.ErrorMsg, record.Operator)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	record.ID = id
	return nil
}

func (d *MySQLPublishRecordDAO) ListRecords(appID string, limit int) ([]*PublishRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `SELECT id, app_id, rule_type, resource, rule_count, status, error_msg, operator, created_at
			  FROM publish_records WHERE app_id = ?
			  ORDER BY created_at DESC LIMIT ?`
	rows, err := d.db.Query(query, appID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*PublishRecord
	for rows.Next() {
		var r PublishRecord
		if err := rows.Scan(&r.ID, &r.AppID, &r.RuleType, &r.Resource,
			&r.RuleCount, &r.Status, &r.ErrorMsg, &r.Operator, &r.CreatedAt); err != nil {
			continue
		}
		records = append(records, &r)
	}
	return records, nil
}

// ============================================
// Record Models (DB-specific)
// ============================================

type FlowRuleRecord struct {
	ID    int64  `json:"id"`
	AppID string `json:"app_id"`

	ResourceID             int64          `json:"resource_id"`
	Threshold              float64        `json:"threshold"`
	MetricType             int            `json:"metric_type"`
	ControlBehavior        int            `json:"control_behavior"`
	WarmUpPeriodSec        int            `json:"warm_up_period_sec"`
	MaxQueueingTimeMs      int            `json:"max_queueing_time_ms"`
	ClusterMode            bool           `json:"cluster_mode"`
	ClusterConfig          sql.NullString `json:"cluster_config"`
	TokenCalculateStrategy int            `json:"token_calculate_strategy"`
	RelationStrategy       int            `json:"relation_strategy"`
	RefResource            string         `json:"ref_resource"`
	WarmUpColdFactor       int            `json:"warm_up_cold_factor"`
	StatIntervalMs         int            `json:"stat_interval_ms"`
	Enabled                bool           `json:"enabled"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
}

type CBRuleRecord struct {
	ID    int64  `json:"id"`
	AppID string `json:"app_id"`

	ResourceID                   int64     `json:"resource_id"`
	Strategy                     int       `json:"strategy"`
	Threshold                    float64   `json:"threshold"`
	RetryTimeoutMs               int64     `json:"retry_timeout_ms"`
	MinRequestAmount             int       `json:"min_request_amount"`
	StatIntervalMs               int       `json:"stat_interval_ms"`
	StatSlidingWindowBucketCount int       `json:"stat_sliding_window_bucket_count"`
	MaxAllowedRtMs               int64     `json:"max_allowed_rt_ms"`
	ProbeNum                     int       `json:"probe_num"`
	Enabled                      bool      `json:"enabled"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`
}

type PublishRecord struct {
	ID    int64  `json:"id"`
	AppID string `json:"app_id"`

	RuleType  string    `json:"rule_type"`
	Resource  string    `json:"resource"`
	RuleCount int       `json:"rule_count"`
	Status    string    `json:"status"`
	ErrorMsg  string    `json:"error_msg"`
	Operator  string    `json:"operator"`
	CreatedAt time.Time `json:"created_at"`
}

// ============================================
// Publish Version DAO (MySQL)
// ============================================

type PublishVersion struct {
	ID            int64     `json:"id"`
	AppID         string    `json:"app_id"`
	VersionNumber int       `json:"version_number"`
	Description   string    `json:"description"`
	Operator      string    `json:"operator"`
	RuleCount     int       `json:"rule_count"`
	Snapshot      string    `json:"snapshot"`
	Status        string    `json:"status"`
	ErrorMsg      string    `json:"error_msg"`
	CreatedAt     time.Time `json:"created_at"`
}

type RuleSnapshot struct {
	FlowRules []map[string]interface{} `json:"flow_rules"`
	CBRules   []map[string]interface{} `json:"circuit_breaker_rules"`
}

type MySQLPublishVersionDAO struct {
	db *sql.DB
}

func NewMySQLPublishVersionDAO(db *sql.DB) *MySQLPublishVersionDAO {
	return &MySQLPublishVersionDAO{db: db}
}

func (d *MySQLPublishVersionDAO) GetLatestVersionNumber(appID string) (int, error) {
	var version sql.NullInt64
	err := d.db.QueryRow(
		"SELECT MAX(version_number) FROM publish_versions WHERE app_id = ?",
		appID).Scan(&version)
	if err != nil || !version.Valid {
		return 0, err
	}
	return int(version.Int64), nil
}

func (d *MySQLPublishVersionDAO) CreateVersion(v *PublishVersion) error {
	query := `INSERT INTO publish_versions 
		(app_id, version_number, description, operator, rule_count, snapshot, status, error_msg)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := d.db.Exec(query,
		v.AppID, v.VersionNumber, v.Description,
		v.Operator, v.RuleCount, v.Snapshot, v.Status, v.ErrorMsg)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	v.ID = id
	return nil
}

func (d *MySQLPublishVersionDAO) ListVersions(appID string, limit int) ([]*PublishVersion, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `SELECT id, app_id, version_number, description, operator, 
			  rule_count, status, created_at 
			  FROM publish_versions WHERE app_id = ? 
			  ORDER BY version_number DESC LIMIT ?`
	rows, err := d.db.Query(query, appID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*PublishVersion
	for rows.Next() {
		var v PublishVersion
		if err := rows.Scan(&v.ID, &v.AppID, &v.VersionNumber,
			&v.Description, &v.Operator, &v.RuleCount, &v.Status, &v.CreatedAt); err != nil {
			continue
		}
		versions = append(versions, &v)
	}
	return versions, nil
}

func (d *MySQLPublishVersionDAO) GetVersion(versionID int64) (*PublishVersion, error) {
	query := `SELECT id, app_id, version_number, description, operator, 
			  rule_count, snapshot, status, error_msg, created_at
			  FROM publish_versions WHERE id = ?`
	var v PublishVersion
	err := d.db.QueryRow(query, versionID).Scan(
		&v.ID, &v.AppID, &v.VersionNumber,
		&v.Description, &v.Operator, &v.RuleCount, &v.Snapshot,
		&v.Status, &v.ErrorMsg, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// DeleteAllFlowRules deletes all flow rules for an app/env (for rollback)
func (d *MySQLFlowRuleDAO) DeleteAll(appID string) error {
	_, err := d.db.Exec("DELETE FROM business_line_resource_flowrules WHERE app_id = ?", appID)
	return err
}

// DeleteAllCircuitBreakerRules deletes all CB rules for an app/env (for rollback)
func (d *MySQLCBRuleDAO) DeleteAll(appID string) error {
	_, err := d.db.Exec("DELETE FROM business_line_resource_circuitbreakerrules WHERE app_id = ?", appID)
	return err
}

func (d *MySQLResourceDAO) ListGroupResources(groupID, appID string) ([]*model.Resource, error) {
	query := `SELECT r.id, r.app_id, r.name, r.description, r.group_id, r.created_at, r.updated_at,
			  COALESCE(g.name, '') as group_name, COALESCE(g.description, '') as group_description
			  FROM business_line_resources r
			  LEFT JOIN business_line_app_groups g ON r.group_id = g.id
			  WHERE r.group_id = ? AND r.app_id = ?
			  ORDER BY r.created_at ASC`
	rows, err := d.db.Query(query, groupID, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []*model.Resource
	for rows.Next() {
		var r model.Resource
		var id int64
		var gID sql.NullString
		var groupName string
		var groupDescription string
		if err := rows.Scan(&id, &r.AppID, &r.Name, &r.Description, &gID, &r.CreatedAt, &r.UpdatedAt, &groupName, &groupDescription); err != nil {
			continue
		}
		r.ID = fmt.Sprintf("%d", id)
		r.GroupName = groupName
		r.GroupDescription = groupDescription
		if gID.Valid {
			r.GroupID = &gID.String
		}
		resources = append(resources, &r)
	}
	return resources, nil
}
