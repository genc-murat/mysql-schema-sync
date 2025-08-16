package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"mysql-schema-sync/internal/logging"
)

// NotificationManager handles sending notifications for backup system alerts
type NotificationManager struct {
	logger   *logging.Logger
	config   NotificationConfig
	channels []NotificationChannel
}

// NotificationConfig holds configuration for notifications
type NotificationConfig struct {
	Enabled   bool                `yaml:"enabled"`
	Email     *EmailConfig        `yaml:"email,omitempty"`
	Webhook   *WebhookConfig      `yaml:"webhook,omitempty"`
	Slack     *SlackConfig        `yaml:"slack,omitempty"`
	Teams     *TeamsConfig        `yaml:"teams,omitempty"`
	File      *FileConfig         `yaml:"file,omitempty"`
	Filters   NotificationFilters `yaml:"filters"`
	RateLimit RateLimitConfig     `yaml:"rate_limit"`
}

// EmailConfig for email notifications
type EmailConfig struct {
	SMTPHost string   `yaml:"smtp_host"`
	SMTPPort int      `yaml:"smtp_port"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	From     string   `yaml:"from"`
	To       []string `yaml:"to"`
	Subject  string   `yaml:"subject"`
	UseTLS   bool     `yaml:"use_tls"`
}

// WebhookConfig for generic webhook notifications
type WebhookConfig struct {
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Headers map[string]string `yaml:"headers"`
	Timeout time.Duration     `yaml:"timeout"`
}

// SlackConfig for Slack notifications
type SlackConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
	Username   string `yaml:"username"`
	IconEmoji  string `yaml:"icon_emoji"`
}

// TeamsConfig for Microsoft Teams notifications
type TeamsConfig struct {
	WebhookURL string `yaml:"webhook_url"`
}

// FileConfig for file-based notifications
type FileConfig struct {
	Path   string `yaml:"path"`
	Format string `yaml:"format"` // json, text
}

// NotificationFilters define which alerts should trigger notifications
type NotificationFilters struct {
	MinSeverity   AlertSeverity `yaml:"min_severity"`
	AlertTypes    []AlertType   `yaml:"alert_types"`
	ExcludeTypes  []AlertType   `yaml:"exclude_types"`
	BusinessHours bool          `yaml:"business_hours_only"`
	Weekdays      bool          `yaml:"weekdays_only"`
}

// RateLimitConfig prevents notification spam
type RateLimitConfig struct {
	MaxPerHour   int           `yaml:"max_per_hour"`
	MaxPerDay    int           `yaml:"max_per_day"`
	CooldownTime time.Duration `yaml:"cooldown_time"`
}

// NotificationChannel interface for different notification methods
type NotificationChannel interface {
	Send(ctx context.Context, alert Alert) error
	GetType() string
	IsEnabled() bool
}

// NotificationMessage represents a formatted notification message
type NotificationMessage struct {
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Severity  AlertSeverity          `json:"severity"`
	Timestamp time.Time              `json:"timestamp"`
	AlertID   string                 `json:"alert_id"`
	AlertType AlertType              `json:"alert_type"`
	Metadata  map[string]interface{} `json:"metadata"`
	Color     string                 `json:"color,omitempty"`      // For visual notifications
	IconEmoji string                 `json:"icon_emoji,omitempty"` // For Slack/Teams
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(logger *logging.Logger, config NotificationConfig) *NotificationManager {
	nm := &NotificationManager{
		logger:   logger,
		config:   config,
		channels: make([]NotificationChannel, 0),
	}

	// Initialize notification channels based on configuration
	if config.Email != nil {
		nm.channels = append(nm.channels, NewEmailChannel(logger, *config.Email))
	}
	if config.Webhook != nil {
		nm.channels = append(nm.channels, NewWebhookChannel(logger, *config.Webhook))
	}
	if config.Slack != nil {
		nm.channels = append(nm.channels, NewSlackChannel(logger, *config.Slack))
	}
	if config.Teams != nil {
		nm.channels = append(nm.channels, NewTeamsChannel(logger, *config.Teams))
	}
	if config.File != nil {
		nm.channels = append(nm.channels, NewFileChannel(logger, *config.File))
	}

	return nm
}

// SendNotification sends a notification for an alert through all configured channels
func (nm *NotificationManager) SendNotification(ctx context.Context, alert Alert) error {
	if !nm.config.Enabled {
		return nil
	}

	// Check if alert should be filtered out
	if !nm.shouldNotify(alert) {
		nm.logger.WithFields(map[string]interface{}{
			"alert_id":   alert.ID,
			"alert_type": string(alert.Type),
			"severity":   string(alert.Severity),
		}).Debug("Alert filtered out, not sending notification")
		return nil
	}

	// Check rate limits
	if !nm.checkRateLimit(alert) {
		nm.logger.WithFields(map[string]interface{}{
			"alert_id":   alert.ID,
			"alert_type": string(alert.Type),
		}).Warn("Notification rate limit exceeded, skipping")
		return nil
	}

	var errors []string
	successCount := 0

	// Send through all enabled channels
	for _, channel := range nm.channels {
		if !channel.IsEnabled() {
			continue
		}

		err := channel.Send(ctx, alert)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", channel.GetType(), err))
			nm.logger.WithFields(map[string]interface{}{
				"channel":  channel.GetType(),
				"alert_id": alert.ID,
				"error":    err.Error(),
			}).Error("Failed to send notification")
		} else {
			successCount++
			nm.logger.WithFields(map[string]interface{}{
				"channel":  channel.GetType(),
				"alert_id": alert.ID,
			}).Info("Notification sent successfully")
		}
	}

	if len(errors) > 0 && successCount == 0 {
		return fmt.Errorf("all notification channels failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// shouldNotify checks if an alert should trigger notifications based on filters
func (nm *NotificationManager) shouldNotify(alert Alert) bool {
	filters := nm.config.Filters

	// Check minimum severity
	if !nm.severityMeetsThreshold(alert.Severity, filters.MinSeverity) {
		return false
	}

	// Check alert type inclusion
	if len(filters.AlertTypes) > 0 {
		found := false
		for _, alertType := range filters.AlertTypes {
			if alert.Type == alertType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check alert type exclusion
	for _, excludeType := range filters.ExcludeTypes {
		if alert.Type == excludeType {
			return false
		}
	}

	// Check business hours filter
	if filters.BusinessHours && !nm.isBusinessHours(alert.Timestamp) {
		return false
	}

	// Check weekdays filter
	if filters.Weekdays && nm.isWeekend(alert.Timestamp) {
		return false
	}

	return true
}

// severityMeetsThreshold checks if alert severity meets the minimum threshold
func (nm *NotificationManager) severityMeetsThreshold(alertSeverity, minSeverity AlertSeverity) bool {
	severityLevels := map[AlertSeverity]int{
		AlertSeverityInfo:     1,
		AlertSeverityWarning:  2,
		AlertSeverityCritical: 3,
	}

	alertLevel, exists := severityLevels[alertSeverity]
	if !exists {
		return false
	}

	minLevel, exists := severityLevels[minSeverity]
	if !exists {
		return true // If min severity is not set, allow all
	}

	return alertLevel >= minLevel
}

// isBusinessHours checks if the timestamp falls within business hours (9 AM - 5 PM)
func (nm *NotificationManager) isBusinessHours(timestamp time.Time) bool {
	hour := timestamp.Hour()
	return hour >= 9 && hour < 17
}

// isWeekend checks if the timestamp falls on a weekend
func (nm *NotificationManager) isWeekend(timestamp time.Time) bool {
	weekday := timestamp.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

// checkRateLimit checks if the notification is within rate limits
func (nm *NotificationManager) checkRateLimit(alert Alert) bool {
	// This is a simplified rate limiting implementation
	// In a production system, you would use a more sophisticated approach
	// with persistent storage to track notification counts
	return true
}

// formatMessage formats an alert into a notification message
func (nm *NotificationManager) formatMessage(alert Alert) NotificationMessage {
	message := NotificationMessage{
		Title:     alert.Title,
		Message:   alert.Message,
		Severity:  alert.Severity,
		Timestamp: alert.Timestamp,
		AlertID:   alert.ID,
		AlertType: alert.Type,
		Metadata:  alert.Metadata,
	}

	// Set color based on severity
	switch alert.Severity {
	case AlertSeverityInfo:
		message.Color = "#36a64f" // Green
		message.IconEmoji = ":information_source:"
	case AlertSeverityWarning:
		message.Color = "#ff9900" // Orange
		message.IconEmoji = ":warning:"
	case AlertSeverityCritical:
		message.Color = "#ff0000" // Red
		message.IconEmoji = ":rotating_light:"
	}

	return message
}

// EmailChannel implements email notifications
type EmailChannel struct {
	logger *logging.Logger
	config EmailConfig
}

// NewEmailChannel creates a new email notification channel
func NewEmailChannel(logger *logging.Logger, config EmailConfig) *EmailChannel {
	return &EmailChannel{
		logger: logger,
		config: config,
	}
}

// Send sends an email notification
func (ec *EmailChannel) Send(ctx context.Context, alert Alert) error {
	if ec.config.SMTPHost == "" || len(ec.config.To) == 0 {
		return fmt.Errorf("email configuration incomplete")
	}

	subject := ec.config.Subject
	if subject == "" {
		subject = fmt.Sprintf("Backup System Alert: %s", alert.Title)
	}

	body := fmt.Sprintf(`
Backup System Alert

Alert ID: %s
Type: %s
Severity: %s
Time: %s

%s

Details:
%s

This is an automated message from the MySQL Schema Sync backup system.
`, alert.ID, alert.Type, alert.Severity, alert.Timestamp.Format(time.RFC3339), alert.Title, alert.Message)

	// Create message
	message := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s",
		strings.Join(ec.config.To, ","), subject, body)

	// Send email
	auth := smtp.PlainAuth("", ec.config.Username, ec.config.Password, ec.config.SMTPHost)
	addr := fmt.Sprintf("%s:%d", ec.config.SMTPHost, ec.config.SMTPPort)

	err := smtp.SendMail(addr, auth, ec.config.From, ec.config.To, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// GetType returns the channel type
func (ec *EmailChannel) GetType() string {
	return "email"
}

// IsEnabled checks if the channel is enabled
func (ec *EmailChannel) IsEnabled() bool {
	return ec.config.SMTPHost != "" && len(ec.config.To) > 0
}

// WebhookChannel implements generic webhook notifications
type WebhookChannel struct {
	logger *logging.Logger
	config WebhookConfig
	client *http.Client
}

// NewWebhookChannel creates a new webhook notification channel
func NewWebhookChannel(logger *logging.Logger, config WebhookConfig) *WebhookChannel {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &WebhookChannel{
		logger: logger,
		config: config,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Send sends a webhook notification
func (wc *WebhookChannel) Send(ctx context.Context, alert Alert) error {
	if wc.config.URL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	// Create notification message
	nm := &NotificationManager{}
	message := nm.formatMessage(alert)

	// Marshal to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create request
	method := wc.config.Method
	if method == "" {
		method = "POST"
	}

	req, err := http.NewRequestWithContext(ctx, method, wc.config.URL, strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range wc.config.Headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := wc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
	}

	return nil
}

// GetType returns the channel type
func (wc *WebhookChannel) GetType() string {
	return "webhook"
}

// IsEnabled checks if the channel is enabled
func (wc *WebhookChannel) IsEnabled() bool {
	return wc.config.URL != ""
}

// SlackChannel implements Slack notifications
type SlackChannel struct {
	logger *logging.Logger
	config SlackConfig
	client *http.Client
}

// NewSlackChannel creates a new Slack notification channel
func NewSlackChannel(logger *logging.Logger, config SlackConfig) *SlackChannel {
	return &SlackChannel{
		logger: logger,
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send sends a Slack notification
func (sc *SlackChannel) Send(ctx context.Context, alert Alert) error {
	if sc.config.WebhookURL == "" {
		return fmt.Errorf("Slack webhook URL not configured")
	}

	// Create notification message
	nm := &NotificationManager{}
	message := nm.formatMessage(alert)

	// Create Slack payload
	payload := map[string]interface{}{
		"text": fmt.Sprintf("%s %s", message.IconEmoji, alert.Title),
		"attachments": []map[string]interface{}{
			{
				"color":     message.Color,
				"title":     alert.Title,
				"text":      alert.Message,
				"timestamp": alert.Timestamp.Unix(),
				"fields": []map[string]interface{}{
					{
						"title": "Alert ID",
						"value": alert.ID,
						"short": true,
					},
					{
						"title": "Type",
						"value": string(alert.Type),
						"short": true,
					},
					{
						"title": "Severity",
						"value": string(alert.Severity),
						"short": true,
					},
				},
			},
		},
	}

	if sc.config.Channel != "" {
		payload["channel"] = sc.config.Channel
	}
	if sc.config.Username != "" {
		payload["username"] = sc.config.Username
	}

	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	// Send request
	req, err := http.NewRequestWithContext(ctx, "POST", sc.config.WebhookURL, strings.NewReader(string(jsonPayload)))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := sc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Slack returned error status: %d", resp.StatusCode)
	}

	return nil
}

// GetType returns the channel type
func (sc *SlackChannel) GetType() string {
	return "slack"
}

// IsEnabled checks if the channel is enabled
func (sc *SlackChannel) IsEnabled() bool {
	return sc.config.WebhookURL != ""
}

// TeamsChannel implements Microsoft Teams notifications
type TeamsChannel struct {
	logger *logging.Logger
	config TeamsConfig
	client *http.Client
}

// NewTeamsChannel creates a new Teams notification channel
func NewTeamsChannel(logger *logging.Logger, config TeamsConfig) *TeamsChannel {
	return &TeamsChannel{
		logger: logger,
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send sends a Teams notification
func (tc *TeamsChannel) Send(ctx context.Context, alert Alert) error {
	if tc.config.WebhookURL == "" {
		return fmt.Errorf("Teams webhook URL not configured")
	}

	// Create notification message
	nm := &NotificationManager{}
	message := nm.formatMessage(alert)

	// Create Teams payload
	payload := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "http://schema.org/extensions",
		"summary":    alert.Title,
		"themeColor": message.Color,
		"sections": []map[string]interface{}{
			{
				"activityTitle":    alert.Title,
				"activitySubtitle": fmt.Sprintf("Alert ID: %s", alert.ID),
				"text":             alert.Message,
				"facts": []map[string]interface{}{
					{
						"name":  "Type",
						"value": string(alert.Type),
					},
					{
						"name":  "Severity",
						"value": string(alert.Severity),
					},
					{
						"name":  "Time",
						"value": alert.Timestamp.Format(time.RFC3339),
					},
				},
			},
		},
	}

	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Teams payload: %w", err)
	}

	// Send request
	req, err := http.NewRequestWithContext(ctx, "POST", tc.config.WebhookURL, strings.NewReader(string(jsonPayload)))
	if err != nil {
		return fmt.Errorf("failed to create Teams request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := tc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Teams notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Teams returned error status: %d", resp.StatusCode)
	}

	return nil
}

// GetType returns the channel type
func (tc *TeamsChannel) GetType() string {
	return "teams"
}

// IsEnabled checks if the channel is enabled
func (tc *TeamsChannel) IsEnabled() bool {
	return tc.config.WebhookURL != ""
}

// FileChannel implements file-based notifications
type FileChannel struct {
	logger *logging.Logger
	config FileConfig
}

// NewFileChannel creates a new file notification channel
func NewFileChannel(logger *logging.Logger, config FileConfig) *FileChannel {
	return &FileChannel{
		logger: logger,
		config: config,
	}
}

// Send writes a notification to a file
func (fc *FileChannel) Send(ctx context.Context, alert Alert) error {
	if fc.config.Path == "" {
		return fmt.Errorf("file path not configured")
	}

	// Create notification message
	nm := &NotificationManager{}
	message := nm.formatMessage(alert)

	var content string
	var err error

	switch fc.config.Format {
	case "json":
		jsonData, err := json.MarshalIndent(message, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal notification to JSON: %w", err)
		}
		content = string(jsonData) + "\n"
	default: // text format
		content = fmt.Sprintf("[%s] %s - %s: %s\n",
			alert.Timestamp.Format(time.RFC3339),
			alert.Severity,
			alert.Type,
			alert.Title)
	}

	// Append to file
	file, err := os.OpenFile(fc.config.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open notification file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write notification to file: %w", err)
	}

	return nil
}

// GetType returns the channel type
func (fc *FileChannel) GetType() string {
	return "file"
}

// IsEnabled checks if the channel is enabled
func (fc *FileChannel) IsEnabled() bool {
	return fc.config.Path != ""
}
