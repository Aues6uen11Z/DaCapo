package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// InstanceLogService manages log files for each instance
type InstanceLogService struct {
	mu    sync.Mutex
	files map[string]*os.File // instanceName -> file handle
	dates map[string]string   // instanceName -> current date (YYYY-MM-DD)
}

var (
	instanceLogService *InstanceLogService
	instanceLogOnce    sync.Once
)

// GetInstanceLogService returns the singleton instance log service
func GetInstanceLogService() *InstanceLogService {
	instanceLogOnce.Do(func() {
		instanceLogService = &InstanceLogService{
			files: make(map[string]*os.File),
			dates: make(map[string]string),
		}
	})
	return instanceLogService
}

// WriteLog writes a log line for an instance
func (s *InstanceLogService) WriteLog(instanceName, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentDate := time.Now().Format("2006-01-02")

	// Check if we need to rotate the log file (new day)
	if existingDate, ok := s.dates[instanceName]; ok && existingDate != currentDate {
		// Close old file
		if f, ok := s.files[instanceName]; ok {
			f.Close()
			delete(s.files, instanceName)
		}
	}

	// Get or create file handle
	f, ok := s.files[instanceName]
	if !ok {
		var err error
		f, err = s.openLogFile(instanceName, currentDate)
		if err != nil {
			return err
		}
		s.files[instanceName] = f
		s.dates[instanceName] = currentDate
	}

	// Write log line with timestamp
	timestamp := time.Now().Format("15:04:05.000")
	_, err := fmt.Fprintf(f, "[%s] %s\n", timestamp, content)
	return err
}

// openLogFile creates/opens a log file for an instance
func (s *InstanceLogService) openLogFile(instanceName, date string) (*os.File, error) {
	// Create directory: logs/instanceName/
	dir := filepath.Join("logs", instanceName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Clean up old log files (keep only last 30 days)
	go s.cleanOldLogs(dir)

	// Open file: logs/instanceName/YYYY-MM-DD.log
	filename := filepath.Join(dir, date+".log")
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return f, nil
}

// cleanOldLogs removes log files older than 30 days
func (s *InstanceLogService) cleanOldLogs(dir string) {
	cutoff := time.Now().AddDate(0, 0, -30)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Parse date from filename (YYYY-MM-DD.log)
		if len(name) < 14 || name[len(name)-4:] != ".log" {
			continue
		}

		dateStr := name[:10]
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		if fileDate.Before(cutoff) {
			os.Remove(filepath.Join(dir, name))
		}
	}
}

// CloseAll closes all open log files
func (s *InstanceLogService) CloseAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, f := range s.files {
		f.Close()
	}
	s.files = make(map[string]*os.File)
	s.dates = make(map[string]string)
}
