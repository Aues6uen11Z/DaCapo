package service

import (
	"context"
	"dacapo/backend/model"
	"dacapo/backend/utils"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type InstanceUpdaterService struct {
	schedulerService *SchedulerService
	wsService        *WebSocketService
}

// UpdateRepo updates repository and manages Python environment for an instance
func (s *InstanceUpdaterService) UpdateRepo(instanceName string) (model.RspUpdateRepo, error) {
	istInfo, err := model.GetInstanceByName(instanceName)
	if err != nil {
		return model.RspUpdateRepo{}, err
	}

	scheduler := model.GetScheduler()
	tm := scheduler.GetTaskManager(instanceName)

	// Check if instance is running
	if tm != nil && tm.Status == model.StatusRunning {
		errMsg := "cannot update - instance is running"
		utils.Logger.Warnf("[%s]: %s", instanceName, errMsg)
		return model.RspUpdateRepo{
			Code:    model.StatusBusy.Code,
			Message: model.StatusBusy.Message,
			Detail:  errMsg,
		}, errors.New(errMsg)
	}

	s.schedulerService.UpdateInstanceStatus(instanceName, model.StatusUpdating)

	// Pull latest code
	if err := utils.RemoveLink(istInfo.ConfigPath); err != nil {
		utils.Logger.Warnf("[%s]: Failed to remove symlink: %v", instanceName, err)
	}
	utils.Logger.Infof("[%s]: Pulling latest changes", instanceName)
	if err := utils.GitPull(istInfo.LocalPath); err != nil {
		utils.Logger.Errorf("[%s]: Failed to pull: %v", instanceName, err)
		s.schedulerService.UpdateInstanceStatus(instanceName, model.StatusFailed)
		return model.RspUpdateRepo{
			Code:    model.StatusGit.Code,
			Message: model.StatusGit.Message,
			Detail:  err.Error(),
		}, err
	}
	utils.CheckLink(filepath.Join("instances", instanceName+".json"), istInfo.ConfigPath)
	utils.Logger.Infof("[%s]: Repository updated successfully", instanceName)

	// Create/update Python environment
	if istInfo.EnvName != "" {
		envPath := filepath.Join("./envs", istInfo.EnvName)
		if err := os.MkdirAll(filepath.Dir(envPath), 0755); err != nil {
			s.schedulerService.UpdateInstanceStatus(instanceName, model.StatusFailed)
			return model.RspUpdateRepo{
				Code:    model.StatusFile.Code,
				Message: model.StatusFile.Message,
				Detail:  err.Error(),
			}, err
		}

		if err = s.createEnv(tm, envPath, istInfo.PythonVersion); err != nil {
			s.schedulerService.UpdateInstanceStatus(instanceName, model.StatusFailed)
			return model.RspUpdateRepo{
				Code:    model.StatusPython.Code,
				Message: model.StatusPython.Message,
				Detail:  err.Error(),
			}, err
		}

		// Install dependencies only if DepsPath is specified
		if istInfo.DepsPath != "" {
			depsPath := filepath.Join(istInfo.LocalPath, istInfo.DepsPath)
			envLastUpdate := istInfo.EnvLastUpdate
			if err = s.installDeps(tm, envPath, depsPath, envLastUpdate); err != nil {
				s.schedulerService.UpdateInstanceStatus(instanceName, model.StatusFailed)
				return model.RspUpdateRepo{
					Code:    model.StatusPython.Code,
					Message: model.StatusPython.Message,
					Detail:  err.Error(),
				}, err
			}
			istInfo.UpdateField("env_last_update", time.Now())
		} else {
			utils.Logger.Infof("[%s]: DepsPath not set, skipping dependency installation", instanceName)
		}
	}

	// Update layout if template file has changed
	var tplInfo model.TemplateInfo
	if err = tplInfo.GetByName(istInfo.TemplateName); err != nil {
		s.schedulerService.UpdateInstanceStatus(instanceName, model.StatusFailed)
		return model.RspUpdateRepo{
			Code:    model.StatusDatabase.Code,
			Message: model.StatusDatabase.Message,
			Detail:  err.Error(),
		}, err
	}
	tplPath, err := model.GetTplPath(tplInfo.Path, "template")
	if err != nil {
		s.schedulerService.UpdateInstanceStatus(instanceName, model.StatusFailed)
		return model.RspUpdateRepo{
			Code:    model.StatusFile.Code,
			Message: model.StatusFile.Message,
			Detail:  err.Error(),
		}, err
	}
	tplFile, err := os.Stat(tplPath)
	if err != nil {
		s.schedulerService.UpdateInstanceStatus(instanceName, model.StatusFailed)
		return model.RspUpdateRepo{
			Code:    model.StatusFile.Code,
			Message: model.StatusFile.Message,
			Detail:  err.Error(),
		}, err
	}
	if tplFile.ModTime().After(istInfo.LayoutLastUpdate) {
		utils.Logger.Infof("[%s]: Template file changed, updating layout and instance config", instanceName)
		istInfo.UpdateField("layout_last_update", time.Now())

		if err = s.syncIstConf(instanceName, tplInfo.Path); err != nil {
			s.schedulerService.UpdateInstanceStatus(instanceName, model.StatusFailed)
			return model.RspUpdateRepo{
				Code:    model.StatusFile.Code,
				Message: model.StatusFile.Message,
				Detail:  err.Error(),
			}, err
		}

		s.schedulerService.UpdateInstanceStatus(instanceName, model.StatusPending)
		return model.RspUpdateRepo{
			Code:      model.StatusSuccess.Code,
			Message:   model.StatusSuccess.Message,
			Detail:    "",
			IsUpdated: true,
		}, nil
	}

	utils.Logger.Infof("[%s]: No changes detected in template file", instanceName)
	s.schedulerService.UpdateInstanceStatus(instanceName, model.StatusPending)
	return model.RspUpdateRepo{
		Code:      model.StatusSuccess.Code,
		Message:   model.StatusSuccess.Message,
		Detail:    "",
		IsUpdated: false,
	}, nil
}

// createEnv creates a Python virtual environment
func (s *InstanceUpdaterService) createEnv(tm *model.TaskManager, envPath string, pythonVersion string) error {
	if _, err := os.Stat(envPath); errors.Is(err, fs.ErrNotExist) {
		var cmd string

		// Try to get UV executable (system -> bundled -> download)
		uvPath, err := utils.GetUVExecutable()
		if err == nil && pythonVersion != "" {
			// Use uv to create venv with specific Python version
			cmd = fmt.Sprintf("%s venv --python %s %s", uvPath, pythonVersion, envPath)
			utils.Logger.Infof("[%s]: Creating venv with uv: %s", tm.InstanceName, cmd)
		} else {
			// Fallback to system python
			if err != nil {
				utils.Logger.Warnf("[%s]: UV not available (%v), using system python", tm.InstanceName, err)
			} else {
				utils.Logger.Warnf("[%s]: Python version not set, using system python", tm.InstanceName)
			}
			cmd = fmt.Sprintf("python -m venv %s", envPath)
			utils.Logger.Infof("[%s]: Creating venv with python: %s", tm.InstanceName, cmd)
		}

		if err := s.schedulerService.RunCommand(tm, cmd, ""); err != nil {
			return err
		}
	}

	return nil
}

// getPyPIMirror returns the fastest available PyPI mirror URL
func getPyPIMirror() (string, error) {
	mirrors := map[string]string{
		"https://pypi.tuna.tsinghua.edu.cn/simple": "https://pypi.tuna.tsinghua.edu.cn/packages/8c/0f/a1f269b125806212a876f7efb049b06c6f8772cf0121139f97774cd95626/numpy-2.3.1-cp313-cp313-macosx_14_0_arm64.whl",
		"https://mirrors.aliyun.com/pypi/simple":   "https://mirrors.aliyun.com/pypi/packages/8c/0f/a1f269b125806212a876f7efb049b06c6f8772cf0121139f97774cd95626/numpy-2.3.1-cp313-cp313-macosx_14_0_arm64.whl",
		"https://pypi.mirrors.ustc.edu.cn/simple":  "https://mirrors.ustc.edu.cn/pypi/packages/8c/0f/a1f269b125806212a876f7efb049b06c6f8772cf0121139f97774cd95626/numpy-2.3.1-cp313-cp313-macosx_14_0_arm64.whl",
		"https://pypi.org/simple":                  "https://files.pythonhosted.org/packages/8c/0f/a1f269b125806212a876f7efb049b06c6f8772cf0121139f97774cd95626/numpy-2.3.1-cp313-cp313-macosx_14_0_arm64.whl",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resultChan := make(chan string, 1)

	for mirror, url := range mirrors {
		go func() {
			req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if _, err := io.Copy(io.Discard, resp.Body); err != nil {
				return
			}

			select {
			case resultChan <- mirror:
			default:
			}
		}()
	}

	select {
	case fastest := <-resultChan:
		cancel() // Cancel other ongoing downloads when first mirror completes
		return fastest, nil
	case <-ctx.Done():
		return "", fmt.Errorf("network too slow, unable to download Python packages, please try again later")
	}
}

// installDeps installs Python dependencies
func (s *InstanceUpdaterService) installDeps(tm *model.TaskManager, envPath, depsPath string, envLastUpdate time.Time) error {
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

	var cmd string
	fastestMirror, err := getPyPIMirror()
	if err != nil {
		return err
	}

	// Try to get UV executable (system -> bundled -> download)
	uvPath, uvErr := utils.GetUVExecutable()
	if uvErr == nil {
		// Use uv to install dependencies
		cmd = fmt.Sprintf("%s pip install -r %s --python %s -i %s",
			uvPath, depsPath, envPath, fastestMirror)
		utils.Logger.Infof("[%s]: Installing dependencies with uv: %s", tm.InstanceName, cmd)
	} else {
		// Fallback to pip
		utils.Logger.Warnf("[%s]: UV not available (%v), using pip", tm.InstanceName, uvErr)
		pythonExec := utils.GetVenvPython(filepath.Base(envPath))
		if pythonExec == "" {
			return fmt.Errorf("python not found in %s", envPath)
		}
		cmd = pythonExec + " -m pip install -r " + depsPath + " -i " + fastestMirror
		utils.Logger.Infof("[%s]: Installing dependencies with pip: %s", tm.InstanceName, cmd)
	}

	if err = s.schedulerService.RunCommand(tm, cmd, ""); err != nil {
		return err
	}

	return nil
}

// syncIstConf synchronizes instance configuration with template
func (s *InstanceUpdaterService) syncIstConf(istName, tplPath string) (err error) {
	istConf := model.NewIstConf()
	tplConf := model.NewTplConf()

	if err = istConf.Load(istName); err != nil {
		return
	}
	if err = tplConf.Load(tplPath); err != nil {
		return
	}

	if err = istConf.Update(tplConf); err != nil {
		return
	}

	return nil
}
