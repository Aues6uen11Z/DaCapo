package service

import (
	"bufio"
	"bytes"
	"dacapo/backend/model"
	"dacapo/backend/utils"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/autobrr/go-shellwords"
	"golang.org/x/text/encoding/simplifiedchinese"
)

var ErrManualStop = errors.New("task manually stopped")

type SchedulerService struct {
	wsService *WebSocketService
}

// UpdateTaskQueue updates the task queue
func (s *SchedulerService) UpdateTaskQueue(queues map[string]model.TaskQueue) {
	scheduler := model.GetScheduler()
	scheduler.UpdateQueue(queues)
}

// UpdateSchedulerState updates the scheduler state
func (s *SchedulerService) UpdateSchedulerState(actionType, instanceName string) {
	if actionType == "start" {
		if instanceName == "" {
			go s.StartAll()
		} else {
			go s.StartOne(instanceName)
		}
	} else if actionType == "stop" {
		if instanceName == "" {
			scheduler := model.GetScheduler()
			scheduler.AutoClose = false
			s.StopAll()
		} else {
			s.stopOne(instanceName, nil)
		}
	}
}

// GetTaskQueue broadcasts task queue for an instance
func (s *SchedulerService) GetTaskQueue(instanceName string) {
	s.wsService.BroadcastQueue(instanceName)
}

// SetSchedulerCron sets the scheduler cron expression
func (s *SchedulerService) SetSchedulerCron(cronExpr string) {
	scheduler := model.GetScheduler()
	scheduler.CronExpr = cronExpr
}

// stopOne stops the instance-level task manager
func (s *SchedulerService) stopOne(instanceName string, err error) {
	scheduler := model.GetScheduler()

	if err == nil {
		// User manually stopped, cancel task execution
		scheduler.CancelTask(instanceName)
		utils.Logger.Infof("[%s]: stopped manually", instanceName)
	} else {
		utils.Logger.Errorf("[%s]: task execution failed: %v", instanceName, err)
	}

	status := model.StatusPending
	if err != nil {
		status = model.StatusFailed
	}

	scheduler.UpdateTaskManagerStatus(instanceName, status)
	s.wsService.BroadcastState(instanceName, status)
	scheduler.TaskManagers[instanceName].RemoveRun()
	s.wsService.BroadcastQueue(instanceName)
	if err != nil {
		s.wsService.BroadcastLog(instanceName, err.Error())
	}
}

// getVenvPython gets the Python executable path for a virtual environment
func (s *SchedulerService) getVenvPython(envName string) string {
	if envName == "" {
		return ""
	}

	venvPath := filepath.Join("envs", envName)
	if runtime.GOOS == "windows" {
		return filepath.Join(venvPath, "Scripts", "python.exe")
	}
	return filepath.Join(venvPath, "bin", "python")
}

// runCommand executes a command and processes the output
func (s *SchedulerService) RunCommand(tm *model.TaskManager, command string, workDir string) error {
	// Create command
	args, err := shellwords.Parse(command)
	if len(args) == 0 {
		return fmt.Errorf("empty command after parsing")
	}
	if err != nil {
		return fmt.Errorf("failed to parse command: %w", err)
	}
	cmd := exec.Command(args[0], args[1:]...)
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: 0x08000000,
		}
	}

	// Set environment variables to force color output
	cmd.Env = append(os.Environ(),
		"PYTHONUNBUFFERED=1",  // Disable Python output buffering
		"FORCE_COLOR=1",       // Force enable color
		"TERM=xterm-256color", // Set terminal type to support color
		"COLORTERM=truecolor", // Enable true color support
		"CLICOLOR=1",          // Enable CLI color
		"CLICOLOR_FORCE=1",    // Force enable CLI color
	)

	// Set working directory
	if workDir != "" {
		cmd.Dir = workDir
	}

	// Create pipes for stdout and stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	tm.Cmd = cmd

	// Create wait group to ensure both goroutines complete
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		s.processOutput(stdoutPipe, tm.InstanceName, false)
	}()
	go func() {
		defer wg.Done()
		s.processOutput(stderrPipe, tm.InstanceName, true)
	}()

	// Wait for command to complete
	err = cmd.Wait()
	wg.Wait()

	if err != nil {
		if tm.ManualStop {
			tm.ManualStop = false
			return ErrManualStop
		}
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

// detectAndConvert detects encoding and converts to UTF-8
func (s *SchedulerService) detectAndConvert(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}

	// Try to decode using GBK encoding, which is common for Chinese text
	if decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(data); err == nil && utf8.Valid(decoded) {
		return string(decoded)
	}

	return string(bytes.ToValidUTF8(data, []byte("�")))
}

// processOutput handles reading from a pipe and broadcasting/logging the output
func (s *SchedulerService) processOutput(pipe io.ReadCloser, instanceName string, isError bool) {
	defer pipe.Close()

	reader := bufio.NewReader(pipe)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				utils.Logger.Errorf("[%s]: Error reading output: %v", instanceName, err)
			}
			break
		}

		// Remove line breaks and handle empty lines
		line = bytes.TrimRight(line, "\r\n")
		if len(line) == 0 {
			s.wsService.BroadcastLog(instanceName, "")
			continue
		}

		// Detect encoding and convert
		text := s.detectAndConvert(line)
		s.wsService.BroadcastLog(instanceName, text)

		if isError {
			utils.Logger.Errorf("[%s]: %s", instanceName, text)
		}
	}
}

// StartOne runs tasks for a single instance
func (s *SchedulerService) StartOne(instanceName string) {
	scheduler := model.GetScheduler()
	tm := scheduler.GetTaskManager(instanceName)
	if tm == nil {
		return
	}

	scheduler.UpdateTaskManagerStatus(instanceName, model.StatusRunning)
	s.wsService.BroadcastState(instanceName, model.StatusRunning)
	s.wsService.BroadcastQueue(instanceName)

	for len(tm.Queue.Waiting) > 0 {
		// Get information about the running task
		taskName := tm.SwitchRun()
		s.wsService.BroadcastQueue(instanceName)

		var istInfo model.InstanceInfo
		if err := istInfo.GetByName(instanceName); err != nil {
			s.stopOne(instanceName, fmt.Errorf("failed to get instance info: %w", err))
			return
		}

		task := istInfo.GetTaskByName(taskName)
		if task == nil {
			s.stopOne(instanceName, fmt.Errorf("task not found: %s", taskName))
			return
		}

		// Execute command
		cmd := task.Command
		if strings.HasPrefix(task.Command, "py ") {
			pythonExec := s.getVenvPython(istInfo.EnvName)
			if pythonExec == "" {
				s.stopOne(instanceName, fmt.Errorf("failed to find python executable in venv: %s", istInfo.EnvName))
				return
			}
			pythonExec, err := filepath.Abs(pythonExec)
			if err != nil {
				s.stopOne(instanceName, fmt.Errorf("failed to get absolute path: %w", err))
				return
			}
			cmd = strings.Replace(task.Command, "py ", "\""+pythonExec+"\" ", 1)
		}

		utils.Logger.Infof("[%s]: Running task <%s>: %s", instanceName, taskName, cmd)
		if err := s.RunCommand(tm, cmd, istInfo.WorkDir); err != nil {
			if errors.Is(err, ErrManualStop) {
				return
			}
			s.stopOne(instanceName, err)
			return
		}

		utils.Logger.Infof("[%s]: task %s finished", instanceName, taskName)
	}

	tm.SwitchRun() // Clean up the last task
	scheduler.UpdateTaskManagerStatus(instanceName, model.StatusPending)
	s.wsService.BroadcastState(instanceName, model.StatusPending)
	s.wsService.BroadcastQueue(instanceName)
}

// StartAll starts tasks for all instances
func (s *SchedulerService) StartAll() {
	scheduler := model.GetScheduler()
	scheduler.Start()
	s.wsService.BroadcastState("", model.StatusRunning)
	utils.Logger.Info("Scheduler started")

	var wg sync.WaitGroup

	// Get information for all instances
	var instances []model.InstanceInfo
	if err := model.GetAllInstances(&instances); err != nil {
		utils.Logger.Error("Failed to get all instances:", err)
		return
	}

	// Divide instances into foreground and background categories
	var foregroundTasks []string
	var backgroundTasks []string
	for _, ist := range instances {
		if !ist.Ready {
			continue
		}
		tm := scheduler.GetTaskManager(ist.Name)
		if tm == nil || tm.Status == model.StatusFailed {
			utils.Logger.Infof("Skip failed/missing instance: %s", ist.Name)
			continue
		}

		if ist.Background {
			backgroundTasks = append(backgroundTasks, ist.Name)
		} else {
			foregroundTasks = append(foregroundTasks, ist.Name)
		}
	}
	utils.Logger.Infof("Background tasks (%d): %v", len(backgroundTasks), backgroundTasks)
	utils.Logger.Infof("Foreground tasks (%d): %v", len(foregroundTasks), foregroundTasks)

	// Handle background tasks
	if len(backgroundTasks) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.runBackgroundTasks(backgroundTasks)
		}()
	}

	// Handle foreground tasks
	if len(foregroundTasks) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.runForegroundTasks(foregroundTasks)
		}()
	}

	// Wait for all tasks to complete before stopping the scheduler
	go func() {
		wg.Wait()
		utils.Logger.Info("All tasks completed, stopping scheduler")
		scheduler.Stop()
		s.wsService.BroadcastState("", model.StatusPending)
	}()
}

// runWithCheck waits for instance updates to complete before execution
func (s *SchedulerService) runWithCheck(istName string) {
	scheduler := model.GetScheduler()
	for {
		if !scheduler.IsRunning {
			return
		}

		tm := scheduler.GetTaskManager(istName)
		if tm == nil {
			utils.Logger.Warnf("TaskManager not found for instance: %s", istName)
			return
		}

		if tm.Status == model.StatusUpdating {
			utils.Logger.Debugf("Instance %s is still updating, waiting...", istName)
			time.Sleep(10 * time.Second)
			continue
		}

		if tm.Status == model.StatusFailed {
			utils.Logger.Warnf("Skip failed instance: %s", istName)
			return
		}

		if tm.Status == model.StatusPending {
			s.StartOne(istName)
		}
		return
	}
}

// runBackgroundTasks executes background tasks concurrently
func (s *SchedulerService) runBackgroundTasks(tasks []string) {
	utils.Logger.Info("Starting background tasks")
	var wg sync.WaitGroup

	for _, istName := range tasks {
		if !model.GetScheduler().IsRunning {
			return
		}

		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			s.runWithCheck(name)
		}(istName)
	}

	wg.Wait()
	utils.Logger.Info("All background tasks completed")
}

// runForegroundTasks executes foreground tasks sequentially
func (s *SchedulerService) runForegroundTasks(tasks []string) {
	utils.Logger.Info("Starting foreground tasks")
	scheduler := model.GetScheduler()

	for len(tasks) > 0 {
		allUpdating := true
		remainingTasks := make([]string, 0, len(tasks))

		for _, istName := range tasks {
			if !scheduler.IsRunning {
				return
			}

			tm := scheduler.GetTaskManager(istName)
			if tm == nil {
				continue
			}

			switch tm.Status {
			case model.StatusUpdating:
				remainingTasks = append(remainingTasks, istName)
			case model.StatusFailed:
				utils.Logger.Warnf("Skip failed instance: %s", istName)
			default:
				allUpdating = false
				s.StartOne(istName)
			}
		}

		tasks = remainingTasks
		if allUpdating && len(tasks) > 0 {
			utils.Logger.Debug("All remaining foreground tasks are updating, waiting...")
			time.Sleep(10 * time.Second)
		}
	}

	utils.Logger.Info("All foreground tasks completed")
}

// StopAll stops the scheduler and all tasks
func (s *SchedulerService) StopAll() {
	utils.Logger.Info("Stopping all running instances...")

	// Get all instances
	var instances []model.InstanceInfo
	if err := model.GetAllInstances(&instances); err != nil {
		utils.Logger.Error("Failed to get all instances:", err)
		return
	}

	// Stop all running instances
	scheduler := model.GetScheduler()
	for _, ist := range instances {
		tm := scheduler.GetTaskManager(ist.Name)
		if tm != nil && tm.Status == model.StatusRunning {
			s.stopOne(ist.Name, nil)
		}
	}

	// Prevent loop calls in CloseApp
	if !scheduler.IsRunning {
		utils.Logger.Info("Scheduler is not running")
		return
	}

	// Stop the scheduler
	scheduler.Stop()
	s.wsService.BroadcastState("", model.StatusPending)
}
