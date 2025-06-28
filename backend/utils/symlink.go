package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// CheckLink checks if the symlink at tgtPath is valid and points to srcPath.
func CheckLink(srcPath, tgtPath string) {
	if tgtPath == "" {
		Logger.Infof("Link of %s is empty", srcPath)
		return
	}

	absSrc, err := filepath.Abs(srcPath)
	if err != nil {
		Logger.Warnf("Failed to get absolute path for %s: %v", srcPath, err)
		return
	}

	// Check if target symlink exists
	info, err := os.Lstat(tgtPath)
	if err != nil {
		// Target doesn't exist, create new symlink
		err = CreateLink(srcPath, tgtPath, "")
		if err != nil {
			Logger.Warnf("Failed to create symlink: %v", err)
		}
		return
	}

	// Check if it's a symlink
	if info.Mode()&os.ModeSymlink == 0 {
		// Target exists but is not a symlink, delete and recreate
		Logger.Warnf("%s exists but is not a symlink, recreating", tgtPath)
		os.Remove(tgtPath)
		err = CreateLink(srcPath, tgtPath, "")
		if err != nil {
			Logger.Warnf("Failed to create symlink: %v", err)
		}
		return
	}

	// Read the destination path of the symlink
	linkDest, err := os.Readlink(tgtPath)
	if err != nil {
		Logger.Warnf("Failed to read symlink %s: %v", tgtPath, err)
		return
	}

	// Get absolute path of the link destination for comparison
	absLinkDest, err := filepath.Abs(linkDest)
	if err != nil {
		Logger.Warnf("Failed to get absolute path for link destination %s: %v", linkDest, err)
		return
	}

	// Compare if the link destination matches the expected source path
	if absLinkDest != absSrc {
		Logger.Infof("Symlink %s points to %s instead of %s, recreating", tgtPath, absLinkDest, absSrc)
		os.Remove(tgtPath)
		err = CreateLink(srcPath, tgtPath, "")
		if err != nil {
			Logger.Warnf("Failed to create symlink: %v", err)
		}
	}
}

// Creates a symlink to the instance configuration file for access by user programs
func CreateLink(srcPath, tgtPath, oldPath string) error {
	absSource, err := filepath.Abs(srcPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute source path: %w", err)
	}
	absTarget, err := filepath.Abs(tgtPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute target path: %w", err)
	}
	absOld, err := filepath.Abs(oldPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute old path: %w", err)
	}

	if absTarget == absOld {
		return nil
	} else {
		if _, err := os.Stat(absTarget); err == nil {
			Logger.Warnf("Target path already exists, skipped: %s", absTarget)
			return nil
		}
	}

	targetDir := filepath.Dir(absTarget)
	if _, err := os.Stat(targetDir); err != nil {
		Logger.Warnf("target directory does not exist: %s", targetDir)
		return nil
	}

	if oldPath != "" {
		os.Remove(absOld)
	}

	if err := os.Symlink(absSource, absTarget); err != nil {
		Logger.Warnf("failed to create symlink in %s: %v", targetDir, err)
	}

	return nil
}

// RemoveLink removes the symlink at the specified path if it exists
func RemoveLink(linkPath string) error {
	if linkPath == "" {
		return nil
	}

	if _, err := os.Stat(linkPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return os.Remove(linkPath)
}
