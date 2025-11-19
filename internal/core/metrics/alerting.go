// Package metrics provides alerting capabilities
package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityError    AlertSeverity = "error"
	SeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents the current status of an alert
type AlertStatus string

const (
	StatusActive   AlertStatus = "active"
	StatusResolved AlertStatus = "resolved"
	StatusSilenced AlertStatus = "silenced"
)

// Alert represents an alert condition
type Alert struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Severity    AlertSeverity          `json:"severity"`
	Status      AlertStatus            `json:"status"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
	Updated     time.Time              `json:"updated"`
	Labels      map[string]string      `json:"labels"`
	Annotations map[string]string      `json:"annotations"`
	SilencedUntil *time.Time          `json:"silenced_until,omitempty"`
}

// AlertRule defines a rule for generating alerts
type AlertRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Condition   string                 `json:"condition"`
	Severity    AlertSeverity          `json:"severity"`
	Duration    time.Duration          `json:"duration"`
	Source      string                 `json:"source"`
	Labels      map[string]string      `json:"labels"`
	Annotations map[string]string      `json:"annotations"`
	Enabled     bool                   `json:"enabled"`
	LastEval    time.Time              `json:"last_eval"`
	ActiveSince *time.Time             `json:"active_since,omitempty"`
}

// AlertManager manages alert rules and notifications
type AlertManager struct {
	mu            sync.RWMutex
	rules         map[string]*AlertRule
	alerts        map[string]*Alert
	checkInterval time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
	handlers      []AlertHandler
}

// AlertHandler handles alert notifications
type AlertHandler interface {
	Handle(ctx context.Context, alert *Alert) error
	Name() string
}

// AlertStats provides alert statistics
type AlertStats struct {
	TotalRules      int                     `json:"total_rules"`
	ActiveRules     int                     `json:"active_rules"`
	TotalAlerts     int                     `json:"total_alerts"`
	ActiveAlerts    int                     `json:"active_alerts"`
	CriticalAlerts   int                     `json:"critical_alerts"`
	WarningAlerts    int                     `json:"warning_alerts"`
	AlertsBySeverity map[AlertSeverity]int   `json:"alerts_by_severity"`
	LastUpdate      time.Time               `json:"last_update"`
}

// NewAlertManager creates a new alert manager
func NewAlertManager(checkInterval time.Duration) *AlertManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &AlertManager{
		rules:         make(map[string]*AlertRule),
		alerts:        make(map[string]*Alert),
		checkInterval: checkInterval,
		ctx:           ctx,
		cancel:        cancel,
		handlers:      make([]AlertHandler, 0),
	}
}

// Start begins alert monitoring
func (am *AlertManager) Start() {
	logger.Info("Starting alert manager", zap.Duration("interval", am.checkInterval))

	am.initDefaultRules()

	go func() {
		ticker := time.NewTicker(am.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				am.evaluateRules()
			case <-am.ctx.Done():
				return
			}
		}
	}()
}

// Stop stops alert monitoring
func (am *AlertManager) Stop() {
	logger.Info("Stopping alert manager")
	am.cancel()
}

// initDefaultRules sets up default alert rules
func (am *AlertManager) initDefaultRules() {
	defaultRules := []AlertRule{
		{
			ID:        "high_cpu",
			Name:      "High CPU Usage",
			Condition: "cpu > 90",
			Severity:  SeverityWarning,
			Duration:  5 * time.Minute,
			Source:    "resource_tracker",
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
			ID:        "high_memory",
			Name:      "High Memory Usage",
			Condition: "memory > 85",
			Severity:  SeverityCritical,
			Duration:  3 * time.Minute,
			Source:    "resource_tracker",
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
			ID:        "high_disk",
			Name:      "High Disk Usage",
			Condition: "disk > 95",
			Severity:  SeverityCritical,
			Duration:  1 * time.Minute,
			Source:    "resource_tracker",
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
		am.rules[rule.ID] = &rule
	}

	logger.Info("Default alert rules initialized", zap.Int("rules", len(defaultRules)))
}

// evaluateRules evaluates all enabled alert rules
func (am *AlertManager) evaluateRules() {
	am.mu.Lock()
	defer am.mu.Unlock()

	for _, rule := range am.rules {
		if !rule.Enabled {
			continue
		}

		shouldAlert := am.evaluateCondition(rule.Condition)
		rule.LastEval = time.Now()

		if shouldAlert {
			am.handleActiveRule(rule)
		} else {
			am.handleInactiveRule(rule)
		}
	}
}

// evaluateCondition evaluates a simple condition expression
func (am *AlertManager) evaluateCondition(condition string) bool {
	// Simulated evaluation - in production, use proper expression evaluator
	now := time.Now().Unix()
	
	switch condition {
	case "cpu > 90":
		return float64(now%100) > 90
	case "memory > 85":
		return float64(now%100) > 85
	case "disk > 95":
		return float64(now%100) > 95
	default:
		return false
	}
}

// handleActiveRule handles when a rule becomes active
func (am *AlertManager) handleActiveRule(rule *AlertRule) {
	if rule.ActiveSince == nil {
		now := time.Now()
		rule.ActiveSince = &now
		logger.Debug("Alert rule activated", zap.String("rule", rule.ID))
		return
	}

	if time.Since(*rule.ActiveSince) >= rule.Duration {
		am.createOrUpdateAlert(rule)
	}
}

// handleInactiveRule handles when a rule becomes inactive
func (am *AlertManager) handleInactiveRule(rule *AlertRule) {
	if rule.ActiveSince != nil {
		rule.ActiveSince = nil
		am.resolveAlert(rule.ID)
		logger.Debug("Alert rule resolved", zap.String("rule", rule.ID))
	}
}

// createOrUpdateAlert creates or updates an alert
func (am *AlertManager) createOrUpdateAlert(rule *AlertRule) {
	alert, exists := am.alerts[rule.ID]
	
	if !exists {
		alert = &Alert{
			ID:          rule.ID,
			Name:        rule.Name,
			Description: rule.Annotations["description"],
			Severity:    rule.Severity,
			Status:      StatusActive,
			Source:      rule.Source,
			Timestamp:   time.Now(),
			Updated:     time.Now(),
			Labels:      rule.Labels,
			Annotations: rule.Annotations,
		}
		
		am.alerts[rule.ID] = alert
		
		logger.Warn("Alert created",
			zap.String("id", alert.ID),
			zap.String("name", alert.Name),
			zap.String("severity", string(alert.Severity)),
		)
		
		am.sendNotification(alert)
	} else {
		alert.Updated = time.Now()
		alert.Status = StatusActive
		
		logger.Debug("Alert updated",
			zap.String("id", alert.ID),
			zap.String("status", string(alert.Status)),
		)
	}
}

// resolveAlert resolves an alert
func (am *AlertManager) resolveAlert(ruleID string) {
	if alert, exists := am.alerts[ruleID]; exists {
		alert.Status = StatusResolved
		alert.Updated = time.Now()
		alert.SilencedUntil = nil
		
		logger.Info("Alert resolved",
			zap.String("id", alert.ID),
			zap.String("name", alert.Name),
		)
		
		am.sendNotification(alert)
	}
}

// sendNotification sends alert to all handlers
func (am *AlertManager) sendNotification(alert *Alert) {
	for _, handler := range am.handlers {
		go func(h AlertHandler) {
			if err := h.Handle(am.ctx, alert); err != nil {
				logger.Error("Alert handler failed",
					zap.String("handler", h.Name()),
					zap.String("alert", alert.ID),
					zap.Error(err),
				)
			}
		}(handler)
	}
}

// AddHandler adds an alert handler
func (am *AlertManager) AddHandler(handler AlertHandler) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	am.handlers = append(am.handlers, handler)
	logger.Info("Alert handler added", zap.String("handler", handler.Name()))
}

// GetAlerts returns all current alerts
func (am *AlertManager) GetAlerts() map[string]*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	alerts := make(map[string]*Alert)
	for k, v := range am.alerts {
		alerts[k] = v
	}
	
	return alerts
}

// GetRules returns all alert rules
func (am *AlertManager) GetRules() map[string]*AlertRule {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	rules := make(map[string]*AlertRule)
	for k, v := range am.rules {
		rules[k] = v
	}
	
	return rules
}

// GetStats returns alert statistics
func (am *AlertManager) GetStats() AlertStats {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	stats := AlertStats{
		TotalRules:      len(am.rules),
		TotalAlerts:     len(am.alerts),
		AlertsBySeverity: make(map[AlertSeverity]int),
		LastUpdate:      time.Now(),
	}
	
	for _, rule := range am.rules {
		if rule.Enabled {
			stats.ActiveRules++
		}
	}
	
	for _, alert := range am.alerts {
		if alert.Status == StatusActive {
			stats.ActiveAlerts++
		}
		
		stats.AlertsBySeverity[alert.Severity]++
		
		switch alert.Severity {
		case SeverityCritical:
			stats.CriticalAlerts++
		case SeverityWarning:
			stats.WarningAlerts++
		}
	}
	
	return stats
}

// LogHandler is a simple alert handler that logs alerts
type LogHandler struct{}

// Handle logs an alert
func (lh *LogHandler) Handle(ctx context.Context, alert *Alert) error {
	if alert.Status == StatusActive {
		logger.Error("ALERT",
			zap.String("id", alert.ID),
			zap.String("name", alert.Name),
			zap.String("severity", string(alert.Severity)),
			zap.String("description", alert.Description),
			zap.Time("timestamp", alert.Timestamp),
		)
	} else {
		logger.Info("ALERT RESOLVED",
			zap.String("id", alert.ID),
			zap.String("name", alert.Name),
			zap.Time("resolved", alert.Updated),
		)
	}
	
	return nil
}

// Name returns handler name
func (lh *LogHandler) Name() string {
	return "log"
}