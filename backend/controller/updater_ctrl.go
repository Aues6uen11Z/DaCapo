package controller

import (
	"dacapo/backend/model"
	"dacapo/backend/utils"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

func UpdateRepo(c *gin.Context) {
	instanceName := c.Param("instance_name")
	istInfo, err := model.GetInstanceByName(instanceName)
	if err != nil {
		quickResponse(c, model.StatusDatabase, instanceName, err.Error())
		return
	}

	scheduler := model.GetScheduler()
	tm := scheduler.GetTaskManager(instanceName)
	broadcastState(instanceName, model.StatusUpdating)
	utils.Logger.Infof("[%s]: Updating", instanceName)

	// Pull latest code
	cmdLog, err := utils.GitPull(istInfo.LocalPath)
	utils.Logger.Infof("[%s]: %s", instanceName, cmdLog)
	if err != nil {
		quickResponse(c, model.StatusGit, instanceName, err.Error())
		return
	}

	// Create/update Python environment
	if istInfo.EnvName != "" {
		envPath := filepath.Join("./envs", istInfo.EnvName)
		if err := os.MkdirAll(filepath.Dir(envPath), 0755); err != nil {
			quickResponse(c, model.StatusFile, instanceName, err.Error())
			return
		}

		if err = createEnv(tm, envPath, istInfo.PythonExec); err != nil {
			quickResponse(c, model.StatusPython, instanceName, err.Error())
			return
		}

		depsPath := filepath.Join(istInfo.LocalPath, istInfo.DepsPath)
		envLastUpdate := istInfo.EnvLastUpdate
		if err = installDeps(tm, envPath, depsPath, envLastUpdate); err != nil {
			quickResponse(c, model.StatusPython, instanceName, err.Error())
			return
		}
		istInfo.UpdateField("env_last_update", time.Now())
	}

	// Update layout if template file has changed
	var tplInfo model.TemplateInfo
	if err = tplInfo.GetByName(istInfo.TemplateName); err != nil {
		quickResponse(c, model.StatusDatabase, instanceName, err.Error())
		return
	}
	tplPath, err := model.GetTplPath(tplInfo.Path, "template")
	if err != nil {
		quickResponse(c, model.StatusFile, instanceName, err.Error())
		return
	}
	tplFile, err := os.Stat(tplPath)
	if err != nil {
		quickResponse(c, model.StatusFile, instanceName, err.Error())
		return
	}
	if tplFile.ModTime().After(istInfo.LayoutLastUpdate) {
		utils.Logger.Infof("[%s]: Template file changed, updating layout", instanceName)
		istInfo.UpdateField("layout_last_update", time.Now())
		c.JSON(http.StatusOK, model.RspUpdateRepo{
			Code:      model.StatusSuccess.Code,
			Message:   model.StatusSuccess.Message,
			Detail:    "",
			IsUpdated: true,
		})
		return
	}

	utils.Logger.Infof("[%s]: No changes detected in template file", instanceName)
	c.JSON(http.StatusOK, model.RspUpdateRepo{
		Code:      model.StatusSuccess.Code,
		Message:   model.StatusSuccess.Message,
		Detail:    "",
		IsUpdated: false,
	})
}

func quickResponse(c *gin.Context, status model.Status, instanceName string, err string) {
	c.JSON(http.StatusOK, gin.H{
		"code":    status.Code,
		"message": status.Message,
		"detail":  err,
	})
	if err != "" {
		utils.Logger.Errorf("[%s]: %v", instanceName, err)
		broadcastState(instanceName, model.StatusFailed)
		return
	}
	broadcastState(instanceName, model.StatusPending)
}

func createEnv(tm *model.TaskManager, path string, pythonExec string) error {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		if pythonExec == "" {
			pythonExec = "python"
		}

		cmd := pythonExec + " -m venv " + filepath.Base(path)
		utils.Logger.Infof("[%s]: Creating venv: %s", tm.InstanceName, cmd)
		if err := runCommand(tm, cmd, filepath.Dir(path)); err != nil {
			return err
		}
	}

	return nil
}

// getVenvPython returns the path to the Python executable in the virtual environment
func getVenvPython(envName string) string {
	possiblePaths := []string{
		filepath.Join("./envs", envName, "Scripts", "python.exe"), // Windows venv
		filepath.Join("./envs", envName, "bin", "python"),         // Unix venv
		filepath.Join("./envs", envName, "python.exe"),            // Windows embed python
		filepath.Join("./envs", envName, "python"),                // Unix embed python
	}
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func installDeps(tm *model.TaskManager, envPath, depsPath string, envLastUpdate time.Time) error {
	// Check if requirements file exists
	depsInfo, err := os.Stat(depsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("requirements file not found: %s", depsPath)
		}
		return fmt.Errorf("failed to check requirements file: %w", err)
	}

	// If requirements file hasn't changed since last env update, skip installation
	if !envLastUpdate.IsZero() && depsInfo.ModTime().Before(envLastUpdate) {
		utils.Logger.Infof("[%s]: Requirements unchanged since last update: %s", tm.InstanceName, depsPath)
		return nil
	}

	// Run install command
	pythonExec := getVenvPython(filepath.Base(envPath))
	if pythonExec == "" {
		return fmt.Errorf("python not found in %s", envPath)
	}
	cmd := pythonExec + " -m pip install -r " + depsPath + " -i https://pypi.tuna.tsinghua.edu.cn/simple/"
	utils.Logger.Infof("[%s]: Installing dependencies: %s", tm.InstanceName, cmd)
	if err = runCommand(tm, cmd, ""); err != nil {
		return err
	}

	return nil
}
