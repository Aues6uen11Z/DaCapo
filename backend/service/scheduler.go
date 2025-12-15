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

// Constants for scheduler configuration
const (
	MaxErrorLength      = 2000             // Maximum length of error message in notification
	UpdateCheckInterval = 10 * time.Second // Interval to check for instance updates
)

type SchedulerService struct {
	wsService    *WebSocketService
	notifService *NotificationService
}

// UpdateTaskQueue updates the task queue
func (s *SchedulerService) UpdateTaskQueue(queues map[string]model.TaskQueue) {
	scheduler := model.GetScheduler()
	scheduler.UpdateQueue(queues)
}

// UpdateInstanceStatus updates the instance status and broadcasts it
func (s *SchedulerService) UpdateInstanceStatus(instanceName, status string) {
	scheduler := model.GetScheduler()
	scheduler.UpdateTaskManagerStatus(instanceName, status)
	s.wsService.BroadcastState(instanceName, status)
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
	tm := scheduler.GetTaskManager(instanceName)

	if err == nil {
		// User manually stopped, cancel task execution
		scheduler.CancelTask(instanceName)
		utils.Logger.Infof("[%s]: stopped manually", instanceName)
		if tm != nil {
			tm.LastError = ""
		}
	} else {
		utils.Logger.Errorf("[%s]: task execution failed: %v", instanceName, err)
		// Store the error message in TaskManager
		if tm != nil {
			tm.LastError = err.Error()
		}
	}

	status := model.StatusPending
	if err != nil {
		status = model.StatusFailed
	}

	s.UpdateInstanceStatus(instanceName, status)
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
	var stderrBuf bytes.Buffer

	wg.Add(2)
	go func() {
		defer wg.Done()
		s.processOutput(stdoutPipe, tm.InstanceName, false, nil)
	}()
	go func() {
		defer wg.Done()
		s.processOutput(stderrPipe, tm.InstanceName, true, &stderrBuf)
	}()

	// Wait for command to complete
	err = cmd.Wait()
	wg.Wait()

	if err != nil {
		if tm.ManualStop {
			tm.ManualStop = false
			return ErrManualStop
		}

		// Build detailed error message with stderr content
		stderrContent := stderrBuf.String()
		if stderrContent != "" {
			// Limit error message length to avoid excessive size
			if len(stderrContent) > MaxErrorLength {
				// Keep the last part which usually contains the actual error
				stderrContent = "...\n" + stderrContent[len(stderrContent)-MaxErrorLength:]
			}
			return fmt.Errorf("command failed with exit code %v:\n%s", err, stderrContent)
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
// If buf is provided, it will also capture the output
func (s *SchedulerService) processOutput(pipe io.ReadCloser, instanceName string, isError bool, buf *bytes.Buffer) {
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
			if buf != nil {
				buf.WriteString("\n")
			}
			continue
		}

		// Detect encoding and convert
		text := s.detectAndConvert(line)
		s.wsService.BroadcastLog(instanceName, text)

		// Capture to buffer if provided
		if buf != nil {
			buf.WriteString(text)
			buf.WriteString("\n")
		}

		if isError {
			utils.Logger.Errorf("[%s]: %s", instanceName, text)
		}
	}
}

// validateTaskManager validates TaskManager state and returns an error result if invalid
func (s *SchedulerService) validateTaskManager(instanceName string) (tm *model.TaskManager, result *model.InstanceResult) {
	scheduler := model.GetScheduler()
	tm = scheduler.GetTaskManager(instanceName)

	if tm == nil {
		return nil, &model.InstanceResult{
			Name:     instanceName,
			TaskName: "",
			Success:  false,
			Error:    "TaskManager not found",
		}
	}

	if tm.Status == model.StatusFailed {
		errMsg := tm.LastError
		if errMsg == "" {
			errMsg = "Instance in failed state"
		}
		return tm, &model.InstanceResult{
			Name:     instanceName,
			TaskName: "",
			Success:  false,
			Error:    errMsg,
		}
	}

	return tm, nil
}

// StartOne runs tasks for a single instance (public wrapper)
func (s *SchedulerService) StartOne(instanceName string) {
	s.startOne(instanceName)
}

// startOne runs tasks for a single instance and returns result
func (s *SchedulerService) startOne(instanceName string) model.InstanceResult {
	tm, errResult := s.validateTaskManager(instanceName)
	if errResult != nil {
		return *errResult
	}

	// Helper function to create error result
	failWithError := func(err error, taskName string) model.InstanceResult {
		s.stopOne(instanceName, err)
		return model.InstanceResult{
			Name:     instanceName,
			TaskName: taskName,
			Success:  false,
			Error:    err.Error(),
		}
	}

	if tm.Status == model.StatusUpdating {
		utils.Logger.Warnf("[%s]: Cannot start - instance is updating", instanceName)
		return model.InstanceResult{
			Name:     instanceName,
			TaskName: "",
			Success:  false,
			Error:    "Instance is updating",
		}
	}

	// Clear previous error message when starting a new run
	tm.LastError = ""
	s.UpdateInstanceStatus(instanceName, model.StatusRunning)
	s.wsService.BroadcastQueue(instanceName)

	for len(tm.Queue.Waiting) > 0 {
		taskName := tm.SwitchRun()
		s.wsService.BroadcastQueue(instanceName)

		var istInfo model.InstanceInfo
		if err := istInfo.GetByName(instanceName); err != nil {
			return failWithError(fmt.Errorf("failed to get instance info: %w", err), taskName)
		}

		task := istInfo.GetTaskByName(taskName)
		if task == nil {
			return failWithError(fmt.Errorf("task not found: %s", taskName), taskName)
		}

		cmd := task.Command
		if strings.HasPrefix(task.Command, "py ") {
			pythonExec := s.getVenvPython(istInfo.EnvName)
			if pythonExec == "" {
				return failWithError(fmt.Errorf("failed to find python executable in venv: %s", istInfo.EnvName), taskName)
			}
			pythonExec, err := filepath.Abs(pythonExec)
			if err != nil {
				return failWithError(fmt.Errorf("failed to get absolute path: %w", err), taskName)
			}
			cmd = strings.Replace(task.Command, "py ", "\""+pythonExec+"\" ", 1)
		}

		utils.Logger.Infof("[%s]: Running task <%s>: %s", instanceName, taskName, cmd)
		if err := s.RunCommand(tm, cmd, istInfo.WorkDir); err != nil {
			if errors.Is(err, ErrManualStop) {
				return model.InstanceResult{
					Name:     instanceName,
					TaskName: taskName,
					Success:  false,
					Error:    "Manually stopped",
				}
			}
			return failWithError(err, taskName)
		}

		utils.Logger.Infof("[%s]: task %s finished", instanceName, taskName)
	}

	tm.SwitchRun() // Clean up the last task
	s.UpdateInstanceStatus(instanceName, model.StatusPending)
	s.wsService.BroadcastQueue(instanceName)

	return model.InstanceResult{
		Name:     instanceName,
		TaskName: "", // Success - no specific failing task
		Success:  true,
		Error:    "",
	}
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
		scheduler.Stop()
		s.wsService.BroadcastState("", model.StatusPending)
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

	// Initialize result tracking
	resultChan := make(chan model.InstanceResult, len(backgroundTasks)+len(foregroundTasks))

	// Handle background tasks
	if len(backgroundTasks) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.runBackgroundTasks(backgroundTasks, resultChan)
		}()
	}

	// Handle foreground tasks
	if len(foregroundTasks) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.runForegroundTasks(foregroundTasks, resultChan)
		}()
	}

	// Wait for all tasks to complete before stopping the scheduler
	go func() {
		wg.Wait()
		close(resultChan)

		// Collect results
		results := make([]model.InstanceResult, 0)
		for result := range resultChan {
			results = append(results, result)
		}

		// Build notification result
		schedulerResult := s.buildSchedulerResult(results)

		// Send notification using injected service
		if s.notifService != nil {
			s.notifService.SendSchedulerNotification(&schedulerResult)
		}

		utils.Logger.Info("All tasks completed, stopping scheduler")
		scheduler.Stop()
		scheduler.TriggerCloseFunc()
		s.wsService.BroadcastState("", model.StatusPending)
	}()
}

// runWithCheck waits for instance updates to complete before execution and returns result
func (s *SchedulerService) runWithCheck(istName string) model.InstanceResult {
	scheduler := model.GetScheduler()
	for {
		if !scheduler.IsRunning {
			return model.InstanceResult{
				Name:     istName,
				TaskName: "",
				Success:  false,
				Error:    "Scheduler stopped",
			}
		}

		tm, errResult := s.validateTaskManager(istName)
		if errResult != nil {
			utils.Logger.Warnf("Skip instance %s: %s", istName, errResult.Error)
			return *errResult
		}

		if tm.Status == model.StatusUpdating {
			time.Sleep(UpdateCheckInterval)
			continue
		}

		return s.startOne(istName)
	}
}

// runBackgroundTasks executes background tasks concurrently with limit
func (s *SchedulerService) runBackgroundTasks(tasks []string, resultChan chan<- model.InstanceResult) {
	utils.Logger.Info("Starting background tasks")
	var wg sync.WaitGroup

	// Load max concurrent setting
	settings, err := model.LoadSettings()
	if err != nil {
		utils.Logger.Warn("Failed to load settings, using default concurrency")
		settings = &model.AppSettings{MaxBgConcurrent: 0}
	}
	maxConcurrent := settings.MaxBgConcurrent
	if maxConcurrent < 0 {
		maxConcurrent = 0 // Fallback to no limit
	}

	if maxConcurrent > 0 {
		utils.Logger.Infof("Max concurrent background tasks: %d", maxConcurrent)
	} else {
		utils.Logger.Info("Max concurrent background tasks: unlimited")
	}

	var semaphore chan struct{}
	if maxConcurrent > 0 {
		semaphore = make(chan struct{}, maxConcurrent)
	}

	for _, istName := range tasks {
		if !model.GetScheduler().IsRunning {
			return
		}

		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			if semaphore != nil {
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
			}

			result := s.runWithCheck(name)
			resultChan <- result
		}(istName)
	}

	wg.Wait()
	utils.Logger.Info("All background tasks completed")
}

// runForegroundTasks executes foreground tasks sequentially
func (s *SchedulerService) runForegroundTasks(tasks []string, resultChan chan<- model.InstanceResult) {
	utils.Logger.Info("Starting foreground tasks")
	scheduler := model.GetScheduler()

	for len(tasks) > 0 {
		allUpdating := true
		remainingTasks := make([]string, 0, len(tasks))

		for _, istName := range tasks {
			if !scheduler.IsRunning {
				return
			}

			tm, errResult := s.validateTaskManager(istName)
			if errResult != nil {
				utils.Logger.Warnf("Skip instance %s: %s", istName, errResult.Error)
				resultChan <- *errResult
				continue
			}

			if tm.Status == model.StatusUpdating {
				remainingTasks = append(remainingTasks, istName)
			} else {
				allUpdating = false
				result := s.startOne(istName)
				resultChan <- result
			}
		}

		tasks = remainingTasks
		if allUpdating && len(tasks) > 0 {
			utils.Logger.Debug("All remaining foreground tasks are updating, waiting...")
			time.Sleep(UpdateCheckInterval)
		}
	}

	utils.Logger.Info("All foreground tasks completed")
}

// StopAll stops the scheduler and all tasks
func (s *SchedulerService) StopAll() {
	utils.Logger.Info("Stopping all running instances...")
	scheduler := model.GetScheduler()
	if !scheduler.IsRunning {
		utils.Logger.Info("Scheduler is not running")
		return
	}

	// Stop the scheduler
	scheduler.Stop()
	s.wsService.BroadcastState("", model.StatusPending)

	// Get all instances
	var instances []model.InstanceInfo
	if err := model.GetAllInstances(&instances); err != nil {
		utils.Logger.Error("Failed to get all instances:", err)
		return
	}
	// Stop all running instances
	for _, ist := range instances {
		tm := scheduler.GetTaskManager(ist.Name)
		if tm != nil && tm.Status == model.StatusRunning {
			s.stopOne(ist.Name, nil)
		}
	}

	scheduler.TriggerCloseFunc()
}

// buildSchedulerResult builds the final scheduler result from instance results
func (s *SchedulerService) buildSchedulerResult(results []model.InstanceResult) model.SchedulerResult {
	schedulerResult := model.SchedulerResult{
		Success:      true,
		FailedCount:  0,
		SuccessCount: 0,
		TotalCount:   len(results),
		FailedNames:  make([]string, 0),
		Results:      results,
	}

	for _, result := range results {
		if result.Success {
			schedulerResult.SuccessCount++
		} else {
			schedulerResult.FailedCount++
			schedulerResult.FailedNames = append(schedulerResult.FailedNames, result.Name)
			schedulerResult.Success = false
		}
	}

	return schedulerResult
}
