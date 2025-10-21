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
	"time"
)

var (
	gitExecutable     string
	gitMutex          sync.Mutex
	gitDownloadMutex  sync.Mutex
	gitDownloadStatus = "not_started" // not_started, downloading, completed, failed
)

// getGitExecutable finds or downloads the git executable
func getGitExecutable() (string, error) {
	gitMutex.Lock()
	if gitExecutable != "" {
		gitMutex.Unlock()
		return gitExecutable, nil
	}
	gitMutex.Unlock()

	// Try system git first
	if path, err := exec.LookPath("git"); err == nil {
		gitMutex.Lock()
		gitExecutable = path
		gitMutex.Unlock()
		Logger.Infof("Using system git: %s", path)
		return path, nil
	}

	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("git is needed but not found")
	}

	// Try bundled git
	bundledGit := "tools/git/cmd/git.exe"
	if _, err := os.Stat(bundledGit); err == nil {
		absPath, err := filepath.Abs(bundledGit)
		if err == nil {
			gitMutex.Lock()
			gitExecutable = absPath
			gitMutex.Unlock()
			Logger.Infof("Using bundled git: %s", absPath)
			return absPath, nil
		}
	}

	// Need to download git
	gitDownloadMutex.Lock()
	if gitDownloadStatus == "downloading" {
		gitDownloadMutex.Unlock()
		// Wait for download to complete
		for {
			gitDownloadMutex.Lock()
			status := gitDownloadStatus
			gitDownloadMutex.Unlock()
			switch status {
			case "completed":
				gitMutex.Lock()
				path := gitExecutable
				gitMutex.Unlock()
				return path, nil
			case "failed":
				return "", fmt.Errorf("git download failed")
			}
			// Sleep briefly and check again
			time.Sleep(3 * time.Second)
		}
	}

	gitDownloadStatus = "downloading"
	gitDownloadMutex.Unlock()

	// Download git
	Logger.Info("Git not found, downloading...")
	if err := downloadGit(); err != nil {
		gitDownloadMutex.Lock()
		gitDownloadStatus = "failed"
		gitDownloadMutex.Unlock()
		return "", fmt.Errorf("failed to download git: %w", err)
	}

	// Check again after download
	if _, err := os.Stat(bundledGit); err == nil {
		absPath, err := filepath.Abs(bundledGit)
		if err == nil {
			gitMutex.Lock()
			gitExecutable = absPath
			gitMutex.Unlock()
			gitDownloadMutex.Lock()
			gitDownloadStatus = "completed"
			gitDownloadMutex.Unlock()
			Logger.Infof("Downloaded and using git: %s", absPath)
			return absPath, nil
		}
	}

	gitDownloadMutex.Lock()
	gitDownloadStatus = "failed"
	gitDownloadMutex.Unlock()
	return "", fmt.Errorf("git executable not found after download")
}

// downloadGit downloads git portable based on user's country
func downloadGit() error {
	country, err := getCountry()
	if err != nil {
		Logger.Warnf("Failed to get country: %v, defaulting to GitHub", err)
		country = ""
	}

	var downloadURL string
	if country == "China" {
		// Use Gitee mirror for China
		downloadURL = "https://gitee.com/aues6uen11z/da-capo/releases/download/tools/git.zip"
		Logger.Info("Downloading git from Gitee (China mirror)")
	} else {
		// Use GitHub for other regions
		downloadURL = "https://github.com/Aues6uen11Z/DaCapo/releases/download/tools/git.zip"
		Logger.Info("Downloading git from GitHub")
	}

	// Download the file
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download git: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download git: HTTP %d", resp.StatusCode)
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

	targetDir := "tools/git"
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

	Logger.Info("Git downloaded and extracted successfully")
	return nil
}

// runGitCommand executes a git command with the given arguments
func runGitCommand(args []string, workDir string) error {
	gitExec, err := getGitExecutable()
	if err != nil {
		return err
	}

	cmd := exec.Command(gitExec, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git command failed: %w\nOutput: %s", err, string(output))
	}

	if len(output) > 0 {
		Logger.Infof("Git output: %s", string(output))
	}

	return nil
}

// GitClone clones a git repository to the given parent directory.
// Returns the equivalent git command string and any error encountered.
func GitClone(url, localDir, branch string) (cmd string, err error) {
	parts := strings.Split(url, "/")
	repoName := strings.TrimSuffix(parts[len(parts)-1], ".git")
	repoPath := filepath.Join(localDir, repoName)

	// Build command string for logging
	cmd = "git clone --recursive " + url + " " + repoPath
	if branch != "" {
		cmd += " && git checkout " + branch
	}

	// Execute git clone
	args := []string{"clone", "--recursive", url, repoPath}
	if err = runGitCommand(args, ""); err != nil {
		return
	}

	// Checkout specific branch if specified
	if branch != "" {
		_, err = GitCheckout(repoPath, branch)
	}

	return
}

// GitCheckout checks out the specified branch in the repository.
// If branch is empty, does nothing.
// Returns the equivalent git command string and any error encountered.
func GitCheckout(repoPath, branch string) (cmd string, err error) {
	if branch == "" {
		return "", nil
	}

	cmd = "git checkout " + branch
	// Git will automatically create local branch tracking remote if it doesn't exist
	err = runGitCommand([]string{"checkout", branch}, repoPath)
	return
}

// GitPull performs a git pull on the specified repository.
// If there are conflicts, it will reset to remote branch (discarding local changes).
// If there are no conflicts, both local and remote changes are kept.
// Returns the equivalent git command string and any error encountered.
func GitPull(repoPath string) (cmd string, err error) {
	cmd = "git pull origin"

	// Try git pull (fetch + merge)
	err = runGitCommand([]string{"pull", "origin"}, repoPath)
	if err != nil {
		// Check if it's just "Already up to date"
		if strings.Contains(err.Error(), "Already up to date") {
			return cmd, nil
		}

		// Pull failed, likely due to conflicts
		Logger.Warn("Pull failed due to conflicts, resetting to remote branch")

		// Abort any ongoing merge
		runGitCommand([]string{"merge", "--abort"}, repoPath)

		// Fetch latest changes
		if err = runGitCommand([]string{"fetch", "origin"}, repoPath); err != nil {
			return cmd, fmt.Errorf("failed to fetch: %w", err)
		}

		// Reset to remote tracking branch
		// Using @{u} (upstream) instead of manually getting branch name
		if err = runGitCommand([]string{"reset", "--hard", "@{u}"}, repoPath); err != nil {
			return cmd, fmt.Errorf("failed to reset to remote: %w", err)
		}

		Logger.Info("Successfully reset to remote branch")
		return cmd, nil
	}

	return
}
