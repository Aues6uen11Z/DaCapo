package service

import "sync"

// ServiceManager manages all service instances and their dependencies
type ServiceManager struct {
	instanceService        *InstanceService
	schedulerService       *SchedulerService
	instanceUpdaterService *InstanceUpdaterService
	wsService              *WebSocketService

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
		// Create WebSocket service first (no dependencies)
		sm.wsService = &WebSocketService{}

		// Create scheduler service with WebSocket dependency
		sm.schedulerService = &SchedulerService{
			wsService: sm.wsService,
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
