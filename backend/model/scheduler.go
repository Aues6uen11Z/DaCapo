package model

import (
	"dacapo/backend/utils"
	"os/exec"
	"sync"
)

// Task status constants
const (
	StatusPending  string = "pending"
	StatusRunning  string = "running"
	StatusUpdating string = "updating"
	StatusFailed   string = "failed"
)

// TaskQueue represents the task queue status for an instance
type TaskQueue struct {
	Running string   `json:"running"`
	Waiting []string `json:"waiting"`
	Stopped []string `json:"stopped"`
}

// TaskManager handles task execution for a specific instance
type TaskManager struct {
	InstanceName string
	Status       string
	Queue        TaskQueue
	Cmd          *exec.Cmd // Current executing command
	ManualStop   bool
}

// SwitchRun switches to the next task in the queue and returns its name
func (tm *TaskManager) SwitchRun() string {
	tm.RemoveRun()

	if len(tm.Queue.Waiting) == 0 {
		return ""
	}
	nextTask := tm.Queue.Waiting[0]
	tm.Queue.Waiting = tm.Queue.Waiting[1:]
	tm.Queue.Running = nextTask

	return nextTask
}

// RemoveRun moves the currently running task to the stopped list
func (tm *TaskManager) RemoveRun() {
	if tm.Queue.Running != "" {
		tm.Queue.Stopped = append(tm.Queue.Stopped, tm.Queue.Running)
		tm.Queue.Running = ""
	}
}

// Cancel terminates the current task execution
func (tm *TaskManager) Cancel() {
	if tm.Cmd != nil && tm.Cmd.Process != nil {
		if err := tm.Cmd.Process.Kill(); err != nil {
			utils.Logger.Errorf("[%s]: Failed to kill process: %v", tm.InstanceName, err)
		}
	}
	tm.ManualStop = true
}

// Scheduler manages task execution across multiple instances
type Scheduler struct {
	TaskManagers map[string]*TaskManager
	IsRunning    bool
	CronExpr     string
	AutoClose    bool
	CloseFunc    func()
	mu           sync.Mutex // Used only when updating queues and switching tasks
}

var scheduler *Scheduler

// GetScheduler initializes or returns the existing scheduler
func GetScheduler() *Scheduler {
	if scheduler == nil {
		scheduler = &Scheduler{
			TaskManagers: make(map[string]*TaskManager),
		}
	}

	// Synchronize with database state
	scheduler.SyncWithDatabase()
	return scheduler
}

func (s *Scheduler) TriggerCloseFunc() {
	if s.AutoClose && s.CloseFunc != nil {
		s.CloseFunc()
	}
}

// Stop halts the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	s.IsRunning = false
	s.mu.Unlock()
}

// Start activates the scheduler
func (s *Scheduler) Start() {
	s.mu.Lock()
	s.IsRunning = true
	s.mu.Unlock()
}

// SyncWithDatabase synchronizes scheduler state with the database
func (s *Scheduler) SyncWithDatabase() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get all instances from database
	instanceNames, err := GetAllIstNames()
	if err != nil {
		utils.Logger.Error("Failed to get instances from database: ", err)
		return
	}

	// Track instances that exist in the database
	existingInstances := make(map[string]bool)
	for _, istName := range instanceNames {
		existingInstances[istName] = true

		// Create TaskManager for new instances
		if _, exists := s.TaskManagers[istName]; !exists {
			var istInfo InstanceInfo
			if err := istInfo.GetByName(istName); err != nil {
				utils.Logger.Errorf("[%s]: Failed to get instance info: %v", istName, err)
				continue
			}

			waitingQueue, stoppedQueue := istInfo.GetTaskQueue()
			s.TaskManagers[istName] = &TaskManager{
				InstanceName: istName,
				Status:       StatusPending,
				Queue: TaskQueue{
					Running: "",
					Waiting: waitingQueue,
					Stopped: stoppedQueue,
				},
			}
		}
	}

	// Remove instances that no longer exist in the database
	for istName := range s.TaskManagers {
		if !existingInstances[istName] {
			utils.Logger.Infof("[%s]: Removing deleted instance from scheduler", istName)
			delete(s.TaskManagers, istName)
		}
	}
}

// GetTaskManager returns the task manager for a specific instance
func (s *Scheduler) GetTaskManager(istName string) *TaskManager {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.TaskManagers[istName]
}

// GetTaskQueues returns a snapshot of all task queues
func (s *Scheduler) GetTaskQueues() map[string]TaskQueue {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string]TaskQueue, len(s.TaskManagers))
	for name, tm := range s.TaskManagers {
		result[name] = TaskQueue{
			Running: tm.Queue.Running,
			Waiting: append([]string{}, tm.Queue.Waiting...),
			Stopped: append([]string{}, tm.Queue.Stopped...),
		}
	}
	return result
}

// UpdateQueue updates task queues
func (s *Scheduler) UpdateQueue(queues map[string]TaskQueue) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// utils.Logger.Debugf("Updating queues: %+v", queues)
	for istName, queue := range queues {
		if tm, ok := s.TaskManagers[istName]; ok {
			tm.Queue = TaskQueue{
				Running: queue.Running,
				Waiting: append([]string{}, queue.Waiting...),
				Stopped: append([]string{}, queue.Stopped...),
			}
		}
	}
}

// UpdateTaskManagerStatus updates the status of a task manager
func (s *Scheduler) UpdateTaskManagerStatus(istName, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tm, ok := s.TaskManagers[istName]; ok {
		tm.Status = status
		utils.Logger.Infof("[%s]: status: %s", istName, status)
	}
}

// CancelTask cancels task execution for an instance
func (s *Scheduler) CancelTask(istName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tm, ok := s.TaskManagers[istName]; ok {
		tm.Cancel()
	}
}
