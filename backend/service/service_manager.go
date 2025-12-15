package service

import (
	"dacapo/backend/model"
	"sync"
)

// ServiceManager manages all service instances and their dependencies
type ServiceManager struct {
	instanceService        *InstanceService
	schedulerService       *SchedulerService
	instanceUpdaterService *InstanceUpdaterService
	wsService              *WebSocketService
	notificationService    *NotificationService

	once sync.Once
}

var serviceManager *ServiceManager

// GetServiceManager returns the singleton service manager
func GetServiceManager() *ServiceManager {
	if serviceManager == nil {
		serviceManager = &ServiceManager{}
		serviceManager.init()
	}
	return serviceManager
}

// Initialize all services with proper dependency injection
func (sm *ServiceManager) init() {
	sm.once.Do(func() {
		// Load settings for notification configuration
		settings, err := model.LoadSettings()
		if err != nil {
			// Log warning but continue - notification is optional
			// utils.Logger can't be used here as it may not be initialized
		}

		// Create WebSocket service first (no dependencies)
		sm.wsService = &WebSocketService{}

		// Create notification service (optional)
		if err == nil && settings.ServerChanSendKey != "" {
			sm.notificationService = NewNotificationService(settings.ServerChanSendKey)
		}

		// Create scheduler service with dependencies
		sm.schedulerService = &SchedulerService{
			wsService:    sm.wsService,
			notifService: sm.notificationService,
		}

		// Create instance service (no dependencies)
		sm.instanceService = &InstanceService{}

		// Create instance updater service with dependencies
		sm.instanceUpdaterService = &InstanceUpdaterService{
			schedulerService: sm.schedulerService,
			wsService:        sm.wsService,
		}
	})
}

// Getters for each service
func (sm *ServiceManager) InstanceService() *InstanceService {
	return sm.instanceService
}

func (sm *ServiceManager) SchedulerService() *SchedulerService {
	return sm.schedulerService
}

func (sm *ServiceManager) InstanceUpdaterService() *InstanceUpdaterService {
	return sm.instanceUpdaterService
}

func (sm *ServiceManager) WebSocketService() *WebSocketService {
	return sm.wsService
}

func (sm *ServiceManager) NotificationService() *NotificationService {
	return sm.notificationService
}

// ReloadNotificationService reloads the notification service with new settings
func (sm *ServiceManager) ReloadNotificationService() error {
	settings, err := model.LoadSettings()
	if err != nil {
		return err
	}

	if settings.ServerChanSendKey != "" {
		sm.notificationService = NewNotificationService(settings.ServerChanSendKey)
	} else {
		sm.notificationService = nil
	}

	// Update scheduler's notification service
	sm.schedulerService.notifService = sm.notificationService
	return nil
}
