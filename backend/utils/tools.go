package utils

import (
	"archive/zip"
	"bytes"
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
)

// toolInfo holds information about a managed external tool
type toolInfo struct {
	executable     string
	mutex          sync.Mutex
	downloadMutex  sync.Mutex
	downloadStatus string // not_started, downloading, completed, failed
}

var (
	gitTool = &toolInfo{downloadStatus: "not_started"}
	uvTool  = &toolInfo{downloadStatus: "not_started"}
)

// getOrDownloadTool is a generic function to find or download an external tool
func getOrDownloadTool(
	tool *toolInfo,
	toolName string,
	systemCmd string,
	bundledPath string,
	githubURL string,
	giteeURL string,
	windowsOnly bool,
) (string, error) {
	// Check cached path
	tool.mutex.Lock()
	if tool.executable != "" {
		tool.mutex.Unlock()
		return tool.executable, nil
	}
	tool.mutex.Unlock()

	// Try system command first
	if path, err := exec.LookPath(systemCmd); err == nil {
		tool.mutex.Lock()
		tool.executable = path
		tool.mutex.Unlock()
		Logger.Infof("Using system %s: %s", toolName, path)
		return path, nil
	}

	// Try bundled version
	if _, err := os.Stat(bundledPath); err == nil {
		absPath, err := filepath.Abs(bundledPath)
		if err == nil {
			tool.mutex.Lock()
			tool.executable = absPath
			tool.mutex.Unlock()
			Logger.Infof("Using bundled %s: %s", toolName, absPath)
			return absPath, nil
		}
	}

	// Check platform support
	if windowsOnly && runtime.GOOS != "windows" {
		return "", fmt.Errorf("%s not found and auto-download is only supported on Windows", toolName)
	}

	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("%s is needed but not found", toolName)
	}

	// Handle concurrent downloads
	tool.downloadMutex.Lock()
	if tool.downloadStatus == "downloading" {
		tool.downloadMutex.Unlock()
		// Wait for download to complete
		return waitForDownload(tool, toolName)
	}

	tool.downloadStatus = "downloading"
	tool.downloadMutex.Unlock()

	// Download the tool
	Logger.Infof("%s not found, downloading...", toolName)
	if err := downloadAndExtractZip(toolName, githubURL, giteeURL); err != nil {
		tool.downloadMutex.Lock()
		tool.downloadStatus = "failed"
		tool.downloadMutex.Unlock()
		return "", fmt.Errorf("failed to download %s: %w", toolName, err)
	}

	// Verify after download
	if _, err := os.Stat(bundledPath); err == nil {
		absPath, err := filepath.Abs(bundledPath)
		if err == nil {
			tool.mutex.Lock()
			tool.executable = absPath
			tool.mutex.Unlock()
			tool.downloadMutex.Lock()
			tool.downloadStatus = "completed"
			tool.downloadMutex.Unlock()
			Logger.Infof("Downloaded and using %s: %s", toolName, absPath)
			return absPath, nil
		}
	}

	tool.downloadMutex.Lock()
	tool.downloadStatus = "failed"
	tool.downloadMutex.Unlock()
	return "", fmt.Errorf("%s executable not found after download", toolName)
}

// waitForDownload waits for an ongoing download to complete
func waitForDownload(tool *toolInfo, toolName string) (string, error) {
	for {
		tool.downloadMutex.Lock()
		status := tool.downloadStatus
		tool.downloadMutex.Unlock()

		switch status {
		case "completed":
			tool.mutex.Lock()
			path := tool.executable
			tool.mutex.Unlock()
			return path, nil
		case "failed":
			return "", fmt.Errorf("%s download failed", toolName)
		}
		time.Sleep(3 * time.Second)
	}
}

// downloadAndExtractZip downloads and extracts a zip file based on user's country
func downloadAndExtractZip(toolName, githubURL, giteeURL string) error {
	country, err := getCountry()
	if err != nil {
		Logger.Warnf("Failed to get country: %v, defaulting to GitHub", err)
		country = ""
	}

	var downloadURL string
	if country == "China" {
		downloadURL = giteeURL
		Logger.Infof("Downloading %s from Gitee (China mirror)", toolName)
	} else {
		downloadURL = githubURL
		Logger.Infof("Downloading %s from GitHub", toolName)
	}

	// Download the file
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: HTTP %d", resp.StatusCode)
	}

	// Read the zip file into memory
	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read zip data: %w", err)
	}

	// Extract zip file
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("failed to read zip: %w", err)
	}

	// Extract filename from URL (e.g., "git.zip" -> "git", "uv.zip" -> "uv")
	urlParts := strings.Split(downloadURL, "/")
	zipFileName := urlParts[len(urlParts)-1]
	toolDirName := strings.TrimSuffix(zipFileName, ".zip")

	// Create subdirectory: tools/git, tools/uv, etc.
	targetDir := filepath.Join("tools", toolDirName)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Extract files
	for _, file := range zipReader.File {
		filePath := filepath.Join(targetDir, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, file.Mode())
			continue
		}

		// Create parent directory if needed
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Extract file
		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to open zip file: %w", err)
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()

		if err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}
	}

	Logger.Infof("%s downloaded and extracted successfully", toolName)
	return nil
}

// GetUVExecutable finds or downloads the uv executable
func GetUVExecutable() (string, error) {
	return getOrDownloadTool(
		uvTool,
		"UV",
		"uv",
		"tools/uv/uv.exe",
		"https://github.com/Aues6uen11Z/DaCapo/releases/download/tools/uv.zip",
		"https://gitee.com/aues6uen11z/da-capo/releases/download/tools/uv.zip",
		true, // Windows only
	)
}

// getGitExecutable finds or downloads the git executable
func getGitExecutable() (string, error) {
	return getOrDownloadTool(
		gitTool,
		"Git",
		"git",
		"tools/git/cmd/git.exe",
		"https://github.com/Aues6uen11Z/DaCapo/releases/download/tools/git.zip",
		"https://gitee.com/aues6uen11z/da-capo/releases/download/tools/git.zip",
		true, // Windows only
	)
}

// runGitCommand runs a git command in the specified directory
func runGitCommand(args []string, workDir string) error {
	gitPath, err := getGitExecutable()
	if err != nil {
		return err
	}

	cmd := exec.Command(gitPath, args...)
	cmd.Dir = workDir

	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow: true,
		}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// GitClone clones a git repository
func GitClone(repoURL, localDir, branch string) error {
	// Extract repository name from URL (e.g., "repo.git" -> "repo")
	parts := strings.Split(repoURL, "/")
	repoName := strings.TrimSuffix(parts[len(parts)-1], ".git")
	repoPath := filepath.Join(localDir, repoName)

	// Clone with submodules
	args := []string{"clone", "--recurse-submodules"}

	// Only specify branch if provided, otherwise use remote default
	if branch != "" {
		args = append(args, "--branch", branch)
	}

	args = append(args, repoURL, repoPath)

	if err := runGitCommand(args, ""); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// GitCheckout checks out a specific branch
func GitCheckout(repoPath, branch string) error {
	if err := runGitCommand([]string{"checkout", branch}, repoPath); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	return nil
}

// GitPull pulls the latest changes from the remote repository
func GitPull(repoPath string) (cmd string, err error) {
	if err = runGitCommand([]string{"pull", "origin"}, repoPath); err != nil {
		errMsg := err.Error()

		// Check if it's a merge conflict
		if strings.Contains(errMsg, "conflict") || strings.Contains(errMsg, "would be overwritten") {
			// Abort any ongoing merge
			runGitCommand([]string{"merge", "--abort"}, repoPath)

			// Fetch and reset to remote tracking branch
			if err = runGitCommand([]string{"fetch", "origin"}, repoPath); err != nil {
				return cmd, fmt.Errorf("failed to fetch: %w", err)
			}

			if err = runGitCommand([]string{"reset", "--hard", "@{u}"}, repoPath); err != nil {
				return cmd, fmt.Errorf("failed to reset to remote: %w", err)
			}

			return cmd, nil
		}

		return cmd, fmt.Errorf("failed to pull: %w", err)
	}

	return
}

// GetVenvPython returns the path to the Python executable in a virtual environment.
// It searches common locations for Python executables across different platforms and venv types.
func GetVenvPython(envName string) string {
	if envName == "" {
		return ""
	}

	possiblePaths := []string{
		filepath.Join("envs", envName, "Scripts", "python.exe"), // Windows venv
		filepath.Join("envs", envName, "bin", "python"),         // Unix venv
		filepath.Join("envs", envName, "python.exe"),            // Windows embed python
		filepath.Join("envs", envName, "python"),                // Unix embed python
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}
