package controller

import (
	"bufio"
	"dacapo/backend/model"
	"dacapo/backend/utils"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/autobrr/go-shellwords"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var ErrManualStop = errors.New("task manually stopped")

// CreateWS establishes a WebSocket connection
func CreateWS(c *gin.Context) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		utils.Logger.Error("Failed to upgrade websocket: ", err)
		return
	}

	go handleConnection(conn)
}

// UpdateTaskQueue updates the task queue
func UpdateTaskQueue(c *gin.Context) {
	var req model.ReqUpdateQueue
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		utils.Logger.Error("Invalid request format\n", err)
		return
	}

	scheduler := model.GetScheduler()
	scheduler.UpdateQueue(req.Queues)

	c.JSON(http.StatusOK, gin.H{
		"code":    model.StatusSuccess.Code,
		"message": model.StatusSuccess.Message,
		"detail":  "",
	})
}

// UpdateSchedulerState updates the scheduler state
func UpdateSchedulerState(c *gin.Context) {
	var req model.ReqSchedulerState
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		utils.Logger.Error("Invalid request format\n", err)
		return
	}

	if req.Type == "start" {
		if req.InstanceName == "" {
			if req.AutoClose {
				scheduler := model.GetScheduler()
				scheduler.AutoClose = true
			}
			go StartAll()
		} else {
			go StartOne(req.InstanceName)
		}
	} else if req.Type == "stop" {
		if req.InstanceName == "" {
			scheduler := model.GetScheduler()
			scheduler.AutoClose = false
			StopAll()
		} else {
			stopOne(req.InstanceName, nil)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    model.StatusSuccess.Code,
		"message": model.StatusSuccess.Message,
		"detail":  "",
	})
}

func GetTaskQueue(c *gin.Context) {
	instanceName := c.Param("instance_name")
	broadcastQueue(instanceName)

	c.JSON(http.StatusOK, gin.H{
		"code":    model.StatusSuccess.Code,
		"message": model.StatusSuccess.Message,
		"detail":  "",
	})
}

func SetSchedulerCron(c *gin.Context) {
	var req model.ReqSchedulerCron
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		utils.Logger.Error("Invalid request format\n", err)
		return
	}

	scheduler := model.GetScheduler()
	scheduler.CronExpr = req.CronExpr

	c.JSON(http.StatusOK, gin.H{
		"code":    model.StatusSuccess.Code,
		"message": model.StatusSuccess.Message,
		"detail":  "",
	})
}

// handleConnection handles the WebSocket connection
func handleConnection(conn *websocket.Conn) {
	defer conn.Close()

	wsManager := utils.GetWSManager()
	wsManager.RegisterClient(conn)
	defer wsManager.RemoveClient(conn)

	// Send initial state
	sendQueue(conn)
	sendState(conn)

	// Listen for connection
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// sendQueue sends the task queue to a specific connection
func sendQueue(conn *websocket.Conn) {
	scheduler := model.GetScheduler()
	queues := scheduler.GetTaskQueues()
	for istName, queue := range queues {
		update := model.RspTaskQueue{
			Type:         "queue",
			InstanceName: istName,
			Queue:        queue,
		}
		if err := utils.GetWSManager().SendJSON(conn, update); err != nil {
			return
		}
	}
}

func sendState(conn *websocket.Conn) {
	scheduler := model.GetScheduler()
	for istName, tm := range scheduler.TaskManagers {
		instanceState := model.RspSchedulerState{
			Type:         "state",
			InstanceName: istName,
			State:        tm.Status,
		}
		if err := utils.GetWSManager().SendJSON(conn, instanceState); err != nil {
			return
		}
	}

	schedulerState := model.RspSchedulerState{
		Type:         "state",
		InstanceName: "",
		State:        map[bool]string{true: "running", false: "pending"}[scheduler.IsRunning],
	}
	utils.GetWSManager().SendJSON(conn, schedulerState)
}

// broadcastQueue broadcasts queue updates
func broadcastQueue(istName string) {
	scheduler := model.GetScheduler()
	tm := scheduler.GetTaskManager(istName)
	if tm == nil {
		return
	}

	update := model.RspTaskQueue{
		Type:         "queue",
		InstanceName: istName,
		Queue:        tm.Queue,
	}
	utils.GetWSManager().BroadcastJSON(update)
}

// broadcastState broadcasts scheduler state
func broadcastState(istName, state string) {
	message := model.RspSchedulerState{
		Type:         "state",
		InstanceName: istName,
		State:        state,
	}
	utils.GetWSManager().BroadcastJSON(message)
}

// broadcastLog broadcasts log messages
func broadcastLog(istName, content string) {
	message := model.RspLogMessage{
		Type:         "log",
		InstanceName: istName,
		Content:      content,
	}
	utils.GetWSManager().BroadcastJSON(message)
}

// stopOne stops the instance-level task manager
func stopOne(instanceName string, err error) {
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
	broadcastState(instanceName, status)
	scheduler.TaskManagers[instanceName].RemoveRun()
	broadcastQueue(instanceName)
	if err != nil {
		broadcastLog(instanceName, err.Error())
	}
}

// runCommand executes a command and processes the output
func runCommand(tm *model.TaskManager, command string, workDir string) error {
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
		processOutput(stdoutPipe, tm.InstanceName, false)
	}()
	go func() {
		defer wg.Done()
		processOutput(stderrPipe, tm.InstanceName, true)
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

// processOutput handles reading from a pipe and broadcasting/logging the output
func processOutput(pipe io.ReadCloser, instanceName string, isError bool) {
	var scanner *bufio.Scanner

	if runtime.GOOS == "windows" {
		// Convert GBK encoding to UTF-8
		gbkDecoder := simplifiedchinese.GBK.NewDecoder()
		transformedReader := transform.NewReader(pipe, gbkDecoder)
		scanner = bufio.NewScanner(transformedReader)
	} else {
		scanner = bufio.NewScanner(pipe)
	}

	const maxCapacity = 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()
		broadcastLog(instanceName, line)

		if isError {
			utils.Logger.Errorf("[%s]: %s", instanceName, line)
		}
	}

	if err := scanner.Err(); err != nil {
		utils.Logger.Errorf("[%s]: Error reading output: %v", instanceName, err)
	}
}

// StartOne runs tasks for a single instance
func StartOne(instanceName string) {
	scheduler := model.GetScheduler()
	tm := scheduler.GetTaskManager(instanceName)
	if tm == nil {
		return
	}

	scheduler.UpdateTaskManagerStatus(instanceName, model.StatusRunning)
	broadcastState(instanceName, model.StatusRunning)
	broadcastQueue(instanceName)

	for len(tm.Queue.Waiting) > 0 {
		// Get information about the running task
		taskName := tm.SwitchRun()
		broadcastQueue(instanceName)

		var istInfo model.InstanceInfo
		if err := istInfo.GetByName(instanceName); err != nil {
			stopOne(instanceName, fmt.Errorf("failed to get instance info: %w", err))
			return
		}

		task := istInfo.GetTaskByName(taskName)
		if task == nil {
			stopOne(instanceName, fmt.Errorf("task not found: %s", taskName))
			return
		}

		// Execute command
		cmd := task.Command
		if strings.HasPrefix(task.Command, "py ") {
			pythonExec := getVenvPython(istInfo.EnvName)
			if pythonExec == "" {
				stopOne(instanceName, fmt.Errorf("failed to find python executable in venv: %s", istInfo.EnvName))
				return
			}
			pythonExec, err := filepath.Abs(pythonExec)
			if err != nil {
				stopOne(instanceName, fmt.Errorf("failed to get absolute path: %w", err))
				return
			}
			cmd = strings.Replace(task.Command, "py ", "\""+pythonExec+"\" ", 1)
		}

		utils.Logger.Infof("[%s]: Running task <%s>: %s", instanceName, taskName, cmd)
		if err := runCommand(tm, cmd, istInfo.WorkDir); err != nil {
			if errors.Is(err, ErrManualStop) {
				return
			}
			stopOne(instanceName, err)
			return
		}

		utils.Logger.Infof("[%s]: task %s finished", instanceName, taskName)
	}

	tm.SwitchRun() // Clean up the last task
	scheduler.UpdateTaskManagerStatus(instanceName, model.StatusPending)
	broadcastState(instanceName, model.StatusPending)
	broadcastQueue(instanceName)
}

// StartAll starts tasks for all instances
func StartAll() {
	scheduler := model.GetScheduler()
	scheduler.Start()
	broadcastState("", model.StatusRunning)
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
			runBackgroundTasks(backgroundTasks)
		}()
	}

	// Handle foreground tasks
	if len(foregroundTasks) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runForegroundTasks(foregroundTasks)
		}()
	}

	// Wait for all tasks to complete before stopping the scheduler
	go func() {
		wg.Wait()
		utils.Logger.Info("All tasks completed, stopping scheduler")
		scheduler.Stop()
		broadcastState("", model.StatusPending)
	}()
}

// runWithCheck waits for instance updates to complete before execution
func runWithCheck(istName string) {
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
			StartOne(istName)
		}
		return
	}
}

// runBackgroundTasks executes background tasks concurrently
func runBackgroundTasks(tasks []string) {
	utils.Logger.Info("Starting background tasks")
	var wg sync.WaitGroup

	for _, istName := range tasks {
		if !model.GetScheduler().IsRunning {
			return
		}

		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			runWithCheck(name)
		}(istName)
	}

	wg.Wait()
	utils.Logger.Info("All background tasks completed")
}

// runForegroundTasks executes foreground tasks sequentially
func runForegroundTasks(tasks []string) {
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
				StartOne(istName)
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
func StopAll() {
	utils.Logger.Info("Stopping all running instances...")
	scheduler := model.GetScheduler()
	if !scheduler.IsRunning {
		utils.Logger.Info("Scheduler is not running")
		return
	}

	// Stop the scheduler
	scheduler.Stop()
	broadcastState("", model.StatusPending)

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
			stopOne(ist.Name, nil)
		}
	}
}
