package backup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mysql-schema-sync/internal/logging"

	"github.com/stretchr/testify/assert"
)

func TestNewNotificationManager(t *testing.T) {
	logger := logging.NewDefaultLogger()

	config := NotificationConfig{
		Enabled: true,
		Email: &EmailConfig{
			SMTPHost: "smtp.example.com",
			SMTPPort: 587,
			To:       []string{"admin@example.com"},
		},
		Webhook: &WebhookConfig{
			URL: "https://example.com/webhook",
		},
	}

	nm := NewNotificationManager(logger, config)

	assert.NotNil(t, nm)
	assert.Equal(t, logger, nm.logger)
	assert.Equal(t, config, nm.config)
	assert.Len(t, nm.channels, 2) // Email and Webhook channels
}

func TestNotificationManager_SendNotification(t *testing.T) {
	logger := logging.NewDefaultLogger()
	tempDir := t.TempDir()

	config := NotificationConfig{
		Enabled: true,
		File: &FileConfig{
			Path:   filepath.Join(tempDir, "notifications.txt"),
			Format: "text",
		},
		Filters: NotificationFilters{
			MinSeverity: AlertSeverityWarning,
		},
	}

	nm := NewNotificationManager(logger, config)

	ctx := context.Background()
	alert := Alert{
		ID:        "test-alert-1",
		Type:      AlertTypeBackupFailure,
		Severity:  AlertSeverityCritical,
		Title:     "Test Alert",
		Message:   "This is a test alert",
		Timestamp: time.Now(),
	}

	err := nm.SendNotification(ctx, alert)
	assert.NoError(t, err)

	// Verify file was created and contains notification
	content, err := os.ReadFile(config.File.Path)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "Test Alert")
	assert.Contains(t, string(content), "CRITICAL")
}

func TestNotificationManager_ShouldNotify(t *testing.T) {
	logger := logging.NewDefaultLogger()

	tests := []struct {
		name     string
		filters  NotificationFilters
		alert    Alert
		expected bool
	}{
		{
			name: "meets minimum severity",
			filters: NotificationFilters{
				MinSeverity: AlertSeverityWarning,
			},
			alert: Alert{
				Severity: AlertSeverityCritical,
			},
			expected: true,
		},
		{
			name: "below minimum severity",
			filters: NotificationFilters{
				MinSeverity: AlertSeverityWarning,
			},
			alert: Alert{
				Severity: AlertSeverityInfo,
			},
			expected: false,
		},
		{
			name: "included alert type",
			filters: NotificationFilters{
				AlertTypes: []AlertType{AlertTypeBackupFailure, AlertTypeStorageQuota},
			},
			alert: Alert{
				Type:     AlertTypeBackupFailure,
				Severity: AlertSeverityInfo,
			},
			expected: true,
		},
		{
			name: "excluded alert type",
			filters: NotificationFilters{
				ExcludeTypes: []AlertType{AlertTypeBackupFailure},
			},
			alert: Alert{
				Type:     AlertTypeBackupFailure,
				Severity: AlertSeverityCritical,
			},
			expected: false,
		},
		{
			name: "business hours filter - during business hours",
			filters: NotificationFilters{
				BusinessHours: true,
			},
			alert: Alert{
				Severity:  AlertSeverityWarning,
				Timestamp: time.Date(2023, 1, 2, 10, 0, 0, 0, time.UTC), // Monday 10 AM
			},
			expected: true,
		},
		{
			name: "business hours filter - outside business hours",
			filters: NotificationFilters{
				BusinessHours: true,
			},
			alert: Alert{
				Severity:  AlertSeverityWarning,
				Timestamp: time.Date(2023, 1, 2, 20, 0, 0, 0, time.UTC), // Monday 8 PM
			},
			expected: false,
		},
		{
			name: "weekdays filter - weekday",
			filters: NotificationFilters{
				Weekdays: true,
			},
			alert: Alert{
				Severity:  AlertSeverityWarning,
				Timestamp: time.Date(2023, 1, 2, 10, 0, 0, 0, time.UTC), // Monday
			},
			expected: true,
		},
		{
			name: "weekdays filter - weekend",
			filters: NotificationFilters{
				Weekdays: true,
			},
			alert: Alert{
				Severity:  AlertSeverityWarning,
				Timestamp: time.Date(2023, 1, 7, 10, 0, 0, 0, time.UTC), // Saturday
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NotificationConfig{
				Enabled: true,
				Filters: tt.filters,
			}

			nm := NewNotificationManager(logger, config)
			result := nm.shouldNotify(tt.alert)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEmailChannel(t *testing.T) {
	logger := logging.NewDefaultLogger()

	config := EmailConfig{
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
		Username: "user@example.com",
		Password: "password",
		From:     "backup@example.com",
		To:       []string{"admin@example.com"},
		Subject:  "Test Alert",
	}

	channel := NewEmailChannel(logger, config)

	assert.Equal(t, "email", channel.GetType())
	assert.True(t, channel.IsEnabled())

	// Test with incomplete config
	incompleteConfig := EmailConfig{
		SMTPHost: "smtp.example.com",
		// Missing To field
	}
	incompleteChannel := NewEmailChannel(logger, incompleteConfig)
	assert.False(t, incompleteChannel.IsEnabled())
}

func TestWebhookChannel(t *testing.T) {
	logger := logging.NewDefaultLogger()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var message NotificationMessage
		err := json.NewDecoder(r.Body).Decode(&message)
		assert.NoError(t, err)
		assert.Equal(t, "Test Alert", message.Title)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := WebhookConfig{
		URL:     server.URL,
		Method:  "POST",
		Headers: map[string]string{"Authorization": "Bearer token"},
		Timeout: 10 * time.Second,
	}

	channel := NewWebhookChannel(logger, config)

	assert.Equal(t, "webhook", channel.GetType())
	assert.True(t, channel.IsEnabled())

	ctx := context.Background()
	alert := Alert{
		ID:        "test-alert-1",
		Type:      AlertTypeBackupFailure,
		Severity:  AlertSeverityWarning,
		Title:     "Test Alert",
		Message:   "This is a test alert",
		Timestamp: time.Now(),
	}

	err := channel.Send(ctx, alert)
	assert.NoError(t, err)
}

func TestSlackChannel(t *testing.T) {
	logger := logging.NewDefaultLogger()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		assert.NoError(t, err)
		assert.Contains(t, payload["text"], "Test Alert")
		assert.Contains(t, payload, "attachments")

		w.Write([]byte("ok"))
	}))
	defer server.Close()

	config := SlackConfig{
		WebhookURL: server.URL,
		Channel:    "#alerts",
		Username:   "backup-bot",
		IconEmoji:  ":warning:",
	}

	channel := NewSlackChannel(logger, config)

	assert.Equal(t, "slack", channel.GetType())
	assert.True(t, channel.IsEnabled())

	ctx := context.Background()
	alert := Alert{
		ID:        "test-alert-1",
		Type:      AlertTypeBackupFailure,
		Severity:  AlertSeverityWarning,
		Title:     "Test Alert",
		Message:   "This is a test alert",
		Timestamp: time.Now(),
	}

	err := channel.Send(ctx, alert)
	assert.NoError(t, err)
}

func TestTeamsChannel(t *testing.T) {
	logger := logging.NewDefaultLogger()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		assert.NoError(t, err)
		assert.Equal(t, "MessageCard", payload["@type"])
		assert.Equal(t, "Test Alert", payload["summary"])
		assert.Contains(t, payload, "sections")

		w.Write([]byte("1"))
	}))
	defer server.Close()

	config := TeamsConfig{
		WebhookURL: server.URL,
	}

	channel := NewTeamsChannel(logger, config)

	assert.Equal(t, "teams", channel.GetType())
	assert.True(t, channel.IsEnabled())

	ctx := context.Background()
	alert := Alert{
		ID:        "test-alert-1",
		Type:      AlertTypeBackupFailure,
		Severity:  AlertSeverityWarning,
		Title:     "Test Alert",
		Message:   "This is a test alert",
		Timestamp: time.Now(),
	}

	err := channel.Send(ctx, alert)
	assert.NoError(t, err)
}

func TestFileChannel(t *testing.T) {
	logger := logging.NewDefaultLogger()
	tempDir := t.TempDir()

	tests := []struct {
		name   string
		format string
		verify func(t *testing.T, content string)
	}{
		{
			name:   "text format",
			format: "text",
			verify: func(t *testing.T, content string) {
				assert.Contains(t, content, "Test Alert")
				assert.Contains(t, content, "WARNING")
				assert.Contains(t, content, "BACKUP_FAILURE")
			},
		},
		{
			name:   "json format",
			format: "json",
			verify: func(t *testing.T, content string) {
				var message NotificationMessage
				err := json.Unmarshal([]byte(strings.TrimSpace(content)), &message)
				assert.NoError(t, err)
				assert.Equal(t, "Test Alert", message.Title)
				assert.Equal(t, AlertSeverityWarning, message.Severity)
				assert.Equal(t, AlertTypeBackupFailure, message.AlertType)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, "notifications_"+tt.name+".log")

			config := FileConfig{
				Path:   filePath,
				Format: tt.format,
			}

			channel := NewFileChannel(logger, config)

			assert.Equal(t, "file", channel.GetType())
			assert.True(t, channel.IsEnabled())

			ctx := context.Background()
			alert := Alert{
				ID:        "test-alert-1",
				Type:      AlertTypeBackupFailure,
				Severity:  AlertSeverityWarning,
				Title:     "Test Alert",
				Message:   "This is a test alert",
				Timestamp: time.Now(),
			}

			err := channel.Send(ctx, alert)
			assert.NoError(t, err)

			// Verify file content
			content, err := os.ReadFile(filePath)
			assert.NoError(t, err)
			tt.verify(t, string(content))
		})
	}
}

func TestNotificationMessage_FormatMessage(t *testing.T) {
	nm := &NotificationManager{}

	alert := Alert{
		ID:        "test-alert-1",
		Type:      AlertTypeBackupFailure,
		Severity:  AlertSeverityWarning,
		Title:     "Test Alert",
		Message:   "This is a test alert",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"backup_id": "backup-123",
		},
	}

	message := nm.formatMessage(alert)

	assert.Equal(t, alert.Title, message.Title)
	assert.Equal(t, alert.Message, message.Message)
	assert.Equal(t, alert.Severity, message.Severity)
	assert.Equal(t, alert.ID, message.AlertID)
	assert.Equal(t, alert.Type, message.AlertType)
	assert.Equal(t, alert.Metadata, message.Metadata)
	assert.Equal(t, "#ff9900", message.Color) // Orange for warning
	assert.Equal(t, ":warning:", message.IconEmoji)
}

func TestNotificationFiltering(t *testing.T) {
	logger := logging.NewDefaultLogger()
	tempDir := t.TempDir()

	config := NotificationConfig{
		Enabled: true,
		File: &FileConfig{
			Path:   filepath.Join(tempDir, "filtered_notifications.txt"),
			Format: "text",
		},
		Filters: NotificationFilters{
			MinSeverity:  AlertSeverityWarning,
			ExcludeTypes: []AlertType{AlertTypeSystemHealth},
		},
	}

	nm := NewNotificationManager(logger, config)
	ctx := context.Background()

	tests := []struct {
		name         string
		alert        Alert
		shouldNotify bool
	}{
		{
			name: "critical alert should notify",
			alert: Alert{
				ID:       "alert-1",
				Type:     AlertTypeBackupFailure,
				Severity: AlertSeverityCritical,
				Title:    "Critical Alert",
			},
			shouldNotify: true,
		},
		{
			name: "info alert should not notify",
			alert: Alert{
				ID:       "alert-2",
				Type:     AlertTypeBackupFailure,
				Severity: AlertSeverityInfo,
				Title:    "Info Alert",
			},
			shouldNotify: false,
		},
		{
			name: "excluded type should not notify",
			alert: Alert{
				ID:       "alert-3",
				Type:     AlertTypeSystemHealth,
				Severity: AlertSeverityCritical,
				Title:    "System Health Alert",
			},
			shouldNotify: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.alert.Timestamp = time.Now()
			err := nm.SendNotification(ctx, tt.alert)
			assert.NoError(t, err)
		})
	}

	// Check file content
	content, err := os.ReadFile(config.File.Path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("Error reading file: %v", err)
	}

	contentStr := string(content)

	// Should contain critical alert
	assert.Contains(t, contentStr, "Critical Alert")

	// Should not contain info alert or excluded type
	assert.NotContains(t, contentStr, "Info Alert")
	assert.NotContains(t, contentStr, "System Health Alert")
}

func TestChannelErrorHandling(t *testing.T) {
	logger := logging.NewDefaultLogger()

	// Test webhook with invalid URL
	t.Run("webhook with invalid URL", func(t *testing.T) {
		config := WebhookConfig{
			URL: "invalid-url",
		}

		channel := NewWebhookChannel(logger, config)

		ctx := context.Background()
		alert := Alert{
			ID:        "test-alert",
			Title:     "Test Alert",
			Timestamp: time.Now(),
		}

		err := channel.Send(ctx, alert)
		assert.Error(t, err)
	})

	// Test file channel with invalid path
	t.Run("file channel with invalid path", func(t *testing.T) {
		config := FileConfig{
			Path: "/invalid/path/notifications.log",
		}

		channel := NewFileChannel(logger, config)

		ctx := context.Background()
		alert := Alert{
			ID:        "test-alert",
			Title:     "Test Alert",
			Timestamp: time.Now(),
		}

		err := channel.Send(ctx, alert)
		assert.Error(t, err)
	})
}

func TestMultipleChannels(t *testing.T) {
	logger := logging.NewDefaultLogger()
	tempDir := t.TempDir()

	// Create test webhook server
	webhookCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := NotificationConfig{
		Enabled: true,
		File: &FileConfig{
			Path:   filepath.Join(tempDir, "notifications.txt"),
			Format: "text",
		},
		Webhook: &WebhookConfig{
			URL: server.URL,
		},
	}

	nm := NewNotificationManager(logger, config)
	assert.Len(t, nm.channels, 2) // File and Webhook channels

	ctx := context.Background()
	alert := Alert{
		ID:        "test-alert-1",
		Type:      AlertTypeBackupFailure,
		Severity:  AlertSeverityWarning,
		Title:     "Test Alert",
		Message:   "This is a test alert",
		Timestamp: time.Now(),
	}

	err := nm.SendNotification(ctx, alert)
	assert.NoError(t, err)

	// Verify both channels were called
	assert.True(t, webhookCalled, "Webhook should have been called")

	content, err := os.ReadFile(config.File.Path)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "Test Alert")
}
