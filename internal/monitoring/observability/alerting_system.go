// Package observability provides alerting system capabilities
package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// AlertSeverity represents alert severity level
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents alert status
type AlertStatus string

const (
	AlertStatusActive   AlertStatus = "active"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusSilenced AlertStatus = "silenced"
)

// Alert represents an alert
type Alert struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Severity      AlertSeverity          `json:"severity"`
	Status        AlertStatus            `json:"status"`
	Source        string                 `json:"source"`
	Component     string                 `json:"component"`
	Timestamp     time.Time              `json:"timestamp"`
	UpdatedAt     time.Time              `json:"updated_at"`
	ResolvedAt    *time.Time             `json:"resolved_at,omitempty"`
	Labels        map[string]string      `json:"labels"`
	Annotations   map[string]string      `json:"annotations"`
	SilencedUntil *time.Time             `json:"silenced_until,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AlertRule defines a rule for generating alerts
type AlertRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Condition   string                 `json:"condition"`
	Severity    AlertSeverity          `json:"severity"`
	Duration    time.Duration          `json:"duration"`
	Source      string                 `json:"source"`
	Component   string                 `json:"component"`
	Labels      map[string]string      `json:"labels"`
	Annotations map[string]string      `json:"annotations"`
	Enabled     bool                   `json:"enabled"`
	LastEval    time.Time              `json:"last_eval"`
	ActiveSince *time.Time             `json:"active_since,omitempty"`
}

// AlertHandler handles alert notifications
type AlertHandler interface {
	Handle(ctx context.Context, alert *Alert) error
	Name() string
}

// AlertingSystem manages alerts and alert rules
type AlertingSystem struct {
	mu            sync.RWMutex
	rules         map[string]*AlertRule
	alerts        map[string]*Alert
	handlers      []AlertHandler
	checkInterval time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
}

// AlertStats provides alert statistics
type AlertStats struct {
	TotalRules      int                     `json:"total_rules"`
	ActiveRules     int                     `json:"active_rules"`
	TotalAlerts     int                     `json:"total_alerts"`
	ActiveAlerts    int                     `json:"active_alerts"`
	ResolvedAlerts  int                     `json:"resolved_alerts"`
	CriticalAlerts  int                     `json:"critical_alerts"`
	WarningAlerts   int                     `json:"warning_alerts"`
	ErrorAlerts     int                     `json:"error_alerts"`
	InfoAlerts      int                     `json:"info_alerts"`
	AlertsBySeverity map[AlertSeverity]int   `json:"alerts_by_severity"`
	LastUpdate      time.Time               `json:"last_update"`
}

// AlertConfig represents alerting system configuration
type AlertConfig struct {
	CheckInterval time.Duration `json:"check_interval"`
	EnableAlerts  bool          `json:"enable_alerts"`
	DefaultSeverity AlertSeverity `json:"default_severity"`
}

// DefaultAlertConfig returns default alert configuration
func DefaultAlertConfig() *AlertConfig {
	return &AlertConfig{
		CheckInterval:  30 * time.Second,
		EnableAlerts:   true,
		DefaultSeverity: AlertSeverityWarning,
	}
}

// NewAlertingSystem creates a new alerting system
func NewAlertingSystem(config *AlertConfig) *AlertingSystem {
	if config == nil {
		config = DefaultAlertConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	system := &AlertingSystem{
		rules:         make(map[string]*AlertRule),
		alerts:        make(map[string]*Alert),
		handlers:      make([]AlertHandler, 0),
		checkInterval: config.CheckInterval,
		ctx:           ctx,
		cancel:        cancel,
	}

	system.initDefaultRules()

	logger.Info("Alerting system initialized",
		zap.Duration("check_interval", config.CheckInterval),
		zap.Bool("enabled", config.EnableAlerts))

	return system
}

// initDefaultRules initializes default alert rules
func (as *AlertingSystem) initDefaultRules() {
	defaultRules := []AlertRule{
		{
			ID:          "high_cpu",
			Name:        "High CPU Usage",
			Description: "CPU usage is above 90% for more than 5 minutes",
			Condition:   "cpu > 90",
			Severity:    AlertSeverityWarning,
			Duration:    5 * time.Minute,
			Source:      "resource_monitor",
			Component:   "system",
			Labels: map[string]string{
				"component": "system",
				"metric":    "cpu",
			},
			Annotations: map[string]string{
				"description": "CPU usage is above 90% for more than 5 minutes",
				"runbook":     "check_system_cpu",
			},
			Enabled: true,
		},
		{
			ID:          "high_memory",
			Name:        "High Memory Usage",
			Description: "Memory usage is above 85% for more than 3 minutes",
			Condition:   "memory > 85",
			Severity:    AlertSeverityCritical,
			Duration:    3 * time.Minute,
			Source:      "resource_monitor",
			Component:   "system",
			Labels: map[string]string{
				"component": "system",
				"metric":    "memory",
			},
			Annotations: map[string]string{
				"description": "Memory usage is above 85% for more than 3 minutes",
				"runbook":     "check_system_memory",
			},
			Enabled: true,
		},
		{
			ID:          "high_disk",
			Name:        "High Disk Usage",
			Description: "Disk usage is above 95%",
			Condition:   "disk > 95",
			Severity:    AlertSeverityCritical,
			Duration:    1 * time.Minute,
			Source:      "resource_monitor",
			Component:   "system",
			Labels: map[string]string{
				"component": "system",
				"metric":    "disk",
			},
			Annotations: map[string]string{
				"description": "Disk usage is above 95%",
				"runbook":     "check_disk_space",
			},
			Enabled: true,
		},
	}

	for _, rule := range defaultRules {
		as.rules[rule.ID] = &rule
	}

	logger.Info("Default alert rules initialized", zap.Int("rules", len(defaultRules)))
}

// Start begins alert monitoring
func (as *AlertingSystem) Start() {
	logger.Info("Starting alerting system",
		zap.Duration("check_interval", as.checkInterval))

	go func() {
		ticker := time.NewTicker(as.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				as.evaluateRules()
			case <-as.ctx.Done():
				return
			}
		}
	}()
}

// Stop stops alert monitoring
func (as *AlertingSystem) Stop() {
	logger.Info("Stopping alerting system")
	as.cancel()
}

// CreateAlert creates a new alert
func (as *AlertingSystem) CreateAlert(alert *Alert) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if alert.ID == "" {
		alert.ID = generateAlertID()
	}

	alert.Timestamp = time.Now()
	alert.UpdatedAt = time.Now()
	alert.Status = AlertStatusActive

	as.alerts[alert.ID] = alert

	logger.Warn("Alert created",
		zap.String("id", alert.ID),
		zap.String("name", alert.Name),
		zap.String("severity", string(alert.Severity)),
		zap.String("source", alert.Source))

	as.sendNotification(alert)

	return nil
}

// ResolveAlert resolves an alert
func (as *AlertingSystem) ResolveAlert(alertID string) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	alert, exists := as.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	now := time.Now()
	alert.Status = AlertStatusResolved
	alert.UpdatedAt = now
	alert.ResolvedAt = &now

	logger.Info("Alert resolved",
		zap.String("id", alertID),
		zap.String("name", alert.Name))

	as.sendNotification(alert)

	return nil
}

// SilenceAlert silences an alert until a specific time
func (as *AlertingSystem) SilenceAlert(alertID string, until time.Time) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	alert, exists := as.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	alert.Status = AlertStatusSilenced
	alert.SilencedUntil = &until
	alert.UpdatedAt = time.Now()

	logger.Info("Alert silenced",
		zap.String("id", alertID),
		zap.Time("until", until))

	return nil
}

// AddRule adds an alert rule
func (as *AlertingSystem) AddRule(rule *AlertRule) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if rule.ID == "" {
		rule.ID = generateRuleID()
	}

	as.rules[rule.ID] = rule

	logger.Info("Alert rule added",
		zap.String("id", rule.ID),
		zap.String("name", rule.Name),
		zap.Bool("enabled", rule.Enabled))

	return nil
}

// RemoveRule removes an alert rule
func (as *AlertingSystem) RemoveRule(ruleID string) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	delete(as.rules, ruleID)

	logger.Info("Alert rule removed", zap.String("id", ruleID))

	return nil
}

// AddHandler adds an alert handler
func (as *AlertingSystem) AddHandler(handler AlertHandler) {
	as.mu.Lock()
	defer as.mu.Unlock()

	as.handlers = append(as.handlers, handler)

	logger.Info("Alert handler added", zap.String("handler", handler.Name()))
}

// GetAlert returns an alert by ID
func (as *AlertingSystem) GetAlert(alertID string) (*Alert, bool) {
	as.mu.RLock()
	defer as.mu.RUnlock()

	alert, exists := as.alerts[alertID]
	if !exists {
		return nil, false
	}

	copy := *alert
	return &copy, true
}

// GetAlerts returns all alerts
func (as *AlertingSystem) GetAlerts() map[string]*Alert {
	as.mu.RLock()
	defer as.mu.RUnlock()

	alerts := make(map[string]*Alert)
	for k, v := range as.alerts {
		copy := *v
		alerts[k] = &copy
	}

	return alerts
}

// GetRules returns all alert rules
func (as *AlertingSystem) GetRules() map[string]*AlertRule {
	as.mu.RLock()
	defer as.mu.RUnlock()

	rules := make(map[string]*AlertRule)
	for k, v := range as.rules {
		copy := *v
		rules[k] = &copy
	}

	return rules
}

// GetStats returns alert statistics
func (as *AlertingSystem) GetStats() AlertStats {
	as.mu.RLock()
	defer as.mu.RUnlock()

	stats := AlertStats{
		TotalRules:      len(as.rules),
		TotalAlerts:     len(as.alerts),
		AlertsBySeverity: make(map[AlertSeverity]int),
		LastUpdate:      time.Now(),
	}

	for _, rule := range as.rules {
		if rule.Enabled {
			stats.ActiveRules++
		}
	}

	for _, alert := range as.alerts {
		switch alert.Status {
		case AlertStatusActive:
			stats.ActiveAlerts++
		case AlertStatusResolved:
			stats.ResolvedAlerts++
		}

		stats.AlertsBySeverity[alert.Severity]++

		switch alert.Severity {
		case AlertSeverityCritical:
			stats.CriticalAlerts++
		case AlertSeverityWarning:
			stats.WarningAlerts++
		case AlertSeverityError:
			stats.ErrorAlerts++
		case AlertSeverityInfo:
			stats.InfoAlerts++
		}
	}

	return stats
}

// evaluateRules evaluates all enabled alert rules
func (as *AlertingSystem) evaluateRules() {
	as.mu.Lock()
	defer as.mu.Unlock()

	for _, rule := range as.rules {
		if !rule.Enabled {
			continue
		}

		shouldAlert := as.evaluateCondition(rule.Condition)
		rule.LastEval = time.Now()

		if shouldAlert {
			as.handleActiveRule(rule)
		} else {
			as.handleInactiveRule(rule)
		}
	}
}

// evaluateCondition evaluates a condition expression (simplified)
func (as *AlertingSystem) evaluateCondition(condition string) bool {
	// Simplified evaluation - in production, use proper expression evaluator
	// This is a placeholder that would integrate with metrics collection
	return false // Default to false, actual evaluation would check metrics
}

// handleActiveRule handles when a rule becomes active
func (as *AlertingSystem) handleActiveRule(rule *AlertRule) {
	if rule.ActiveSince == nil {
		now := time.Now()
		rule.ActiveSince = &now
		logger.Debug("Alert rule activated", zap.String("rule", rule.ID))
		return
	}

	if time.Since(*rule.ActiveSince) >= rule.Duration {
		as.createOrUpdateAlert(rule)
	}
}

// handleInactiveRule handles when a rule becomes inactive
func (as *AlertingSystem) handleInactiveRule(rule *AlertRule) {
	if rule.ActiveSince != nil {
		rule.ActiveSince = nil
		as.resolveAlertByRule(rule.ID)
		logger.Debug("Alert rule resolved", zap.String("rule", rule.ID))
	}
}

// createOrUpdateAlert creates or updates an alert from a rule
func (as *AlertingSystem) createOrUpdateAlert(rule *AlertRule) {
	alert, exists := as.alerts[rule.ID]

	if !exists {
		alert = &Alert{
			ID:          rule.ID,
			Name:        rule.Name,
			Description: rule.Description,
			Severity:    rule.Severity,
			Status:      AlertStatusActive,
			Source:      rule.Source,
			Component:   rule.Component,
			Timestamp:   time.Now(),
			UpdatedAt:   time.Now(),
			Labels:      rule.Labels,
			Annotations: rule.Annotations,
		}

		as.alerts[rule.ID] = alert

		logger.Warn("Alert created from rule",
			zap.String("id", alert.ID),
			zap.String("name", alert.Name),
			zap.String("severity", string(alert.Severity)))

		as.sendNotification(alert)
	} else {
		alert.UpdatedAt = time.Now()
		alert.Status = AlertStatusActive
	}
}

// resolveAlertByRule resolves an alert by rule ID
func (as *AlertingSystem) resolveAlertByRule(ruleID string) {
	if alert, exists := as.alerts[ruleID]; exists {
		now := time.Now()
		alert.Status = AlertStatusResolved
		alert.UpdatedAt = now
		alert.ResolvedAt = &now

		as.sendNotification(alert)
	}
}

// sendNotification sends alert to all handlers
func (as *AlertingSystem) sendNotification(alert *Alert) {
	for _, handler := range as.handlers {
		go func(h AlertHandler) {
			if err := h.Handle(as.ctx, alert); err != nil {
				logger.Error("Alert handler failed",
					zap.String("handler", h.Name()),
					zap.String("alert", alert.ID),
					zap.Error(err))
			}
		}(handler)
	}
}

// Helper functions

func generateAlertID() string {
	return fmt.Sprintf("alert-%d", time.Now().UnixNano())
}

func generateRuleID() string {
	return fmt.Sprintf("rule-%d", time.Now().UnixNano())
}
