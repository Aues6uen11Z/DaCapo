package service

import (
	"dacapo/backend/model"
	"dacapo/backend/utils"
	"fmt"
	"strings"

	serverchan "github.com/easychen/serverchan-sdk-golang"
)

// NotificationService handles push notifications
type NotificationService struct {
	sendKey string
}

// NewNotificationService creates a new notification service
func NewNotificationService(sendKey string) *NotificationService {
	return &NotificationService{
		sendKey: sendKey,
	}
}

// SendSchedulerNotification sends a notification about scheduler execution results
func (n *NotificationService) SendSchedulerNotification(result *model.SchedulerResult) error {
	if n.sendKey == "" {
		utils.Logger.Info("ServerChan SendKey is not configured, skipping notification")
		return nil
	}

	// Build title
	title := n.buildTitle(result)

	// Build markdown content
	content := n.buildContent(result)

	// Send notification using ServerChan
	_, err := serverchan.ScSend(n.sendKey, title, content, nil)
	if err != nil {
		utils.Logger.Errorf("Failed to send ServerChan notification: %v", err)
		return fmt.Errorf("failed to send notification: %w", err)
	}

	utils.Logger.Infof("ServerChan notification sent successfully: %s", title)
	return nil
}

// buildTitle builds the notification title
func (n *NotificationService) buildTitle(result *model.SchedulerResult) string {
	if result.Success {
		return fmt.Sprintf("DaCapo运行成功 (%d/%d)", result.SuccessCount, result.TotalCount)
	}

	// If there are failures, list failed instance names
	if len(result.FailedNames) > 0 {
		failedList := strings.Join(result.FailedNames, "、")
		return fmt.Sprintf("DaCapo运行失败 - %s", failedList)
	}

	return fmt.Sprintf("DaCapo运行失败 (%d/%d)", result.FailedCount, result.TotalCount)
}

// buildContent builds the notification content in markdown format
func (n *NotificationService) buildContent(result *model.SchedulerResult) string {
	var builder strings.Builder

	// Summary
	builder.WriteString("## 📊 运行概况\n\n")
	builder.WriteString(fmt.Sprintf("- **总实例数**: %d\n", result.TotalCount))
	builder.WriteString(fmt.Sprintf("- **成功**: %d\n", result.SuccessCount))
	builder.WriteString(fmt.Sprintf("- **失败**: %d\n", result.FailedCount))
	builder.WriteString("\n---\n\n")

	// Success instances
	if result.SuccessCount > 0 {
		builder.WriteString("## ✅ 成功实例\n\n")
		for _, r := range result.Results {
			if r.Success {
				builder.WriteString(fmt.Sprintf("- **%s**\n", r.Name))
			}
		}
		builder.WriteString("\n")
	}

	// Failed instances with error details
	if result.FailedCount > 0 {
		builder.WriteString("## ❌ 失败实例\n\n")
		for _, r := range result.Results {
			if !r.Success {
				// Show instance name and task name if available
				if r.TaskName != "" {
					builder.WriteString(fmt.Sprintf("### %s - 任务: %s\n\n", r.Name, r.TaskName))
				} else {
					builder.WriteString(fmt.Sprintf("### %s\n\n", r.Name))
				}
				builder.WriteString("```\n")
				if r.Error != "" {
					builder.WriteString(r.Error)
				} else {
					builder.WriteString("未知错误")
				}
				builder.WriteString("\n```\n\n")
			}
		}
	}

	return builder.String()
}
