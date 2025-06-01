package controller

import (
	"dacapo/backend/model"
	"dacapo/backend/utils"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-git/go-git/v5"
	"github.com/ncruces/go-sqlite3"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func CreateIstFromLocal(c *gin.Context) {
	var req model.ReqFromLocal
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		utils.Logger.Error("Invalid request format\n", err)
		return
	}

	if status, err := fromLocal(req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    status.Code,
			"message": status.Message,
			"detail":  err.Error(),
		})
		utils.Logger.Errorf("[%s]: %v", req.InstanceName, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    model.StatusSuccess.Code,
		"message": model.StatusSuccess.Message,
		"detail":  "",
	})
}

func CreateIstFromTemplate(c *gin.Context) {
	var req model.ReqFromTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		utils.Logger.Error("Invalid request format\n", err)
		return
	}

	if status, err := fromTemplate(req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    status.Code,
			"message": status.Message,
			"detail":  err.Error(),
		})
		utils.Logger.Errorf("[%s]: %v", req.InstanceName, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    model.StatusSuccess.Code,
		"message": model.StatusSuccess.Message,
		"detail":  "",
	})
}

func CreateIstFromRemote(c *gin.Context) {
	var req model.ReqFromRemote
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		utils.Logger.Error("Invalid request format\n", err)
		return
	}

	if status, err := fromRemote(req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    status.Code,
			"message": status.Message,
			"detail":  err.Error(),
		})
		utils.Logger.Errorf("[%s]: %v", req.InstanceName, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    model.StatusSuccess.Code,
		"message": model.StatusSuccess.Message,
		"detail":  "",
	})
}

func GetAllInstances(c *gin.Context) {
	workingTpl, ready, layout, translation, status, err := getAllIst()
	if err != nil {
		c.JSON(http.StatusOK, model.RspGetInstance{
			Code:    status.Code,
			Message: status.Message,
			Detail:  err.Error(),
		})
		utils.Logger.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.RspGetInstance{
		Code:            model.StatusSuccess.Code,
		Message:         model.StatusSuccess.Message,
		Detail:          "",
		WorkingTemplate: workingTpl,
		Ready:           ready,
		Layout:          layout,
		Translation:     translation,
	})
}

func GetInstance(c *gin.Context) {
	instanceName := c.Param("instance_name")

	templateName, ready, layout, translation, status, err := getSingleIst(instanceName)
	if err != nil {
		c.JSON(http.StatusOK, model.RspGetInstance{
			Code:    status.Code,
			Message: status.Message,
			Detail:  err.Error(),
		})
		utils.Logger.Errorf("[%s]: %v", instanceName, err)
		return
	}

	c.JSON(http.StatusOK, model.RspGetInstance{
		Code:            model.StatusSuccess.Code,
		Message:         model.StatusSuccess.Message,
		Detail:          "",
		WorkingTemplate: []string{templateName},
		Ready:           map[string]bool{instanceName: ready},
		Layout:          map[string]any{instanceName: layout},
		Translation:     map[string]any{instanceName: translation},
	})
}

func UpdateInstance(c *gin.Context) {
	instanceName := c.Param("instance_name")

	var req model.ReqUpdateInstance
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		utils.Logger.Error("Invalid request format\n", err)
		return
	}

	// "_Base" group is for DaCapo internal settings
	if req.Group == "_Base" {
		if req.Task == "General" && req.Item == "config_path" {
			istInfo, err := model.GetInstanceByName(instanceName)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"code":    model.StatusDatabase.Code,
					"message": model.StatusDatabase.Message,
					"detail":  err.Error(),
				})
				utils.Logger.Errorf("[%s]: %v", instanceName, err)
			}
			srcPath := filepath.Join("instances", instanceName+".json")
			tgtPath := req.Value.(string)
			oldPath := istInfo.ConfigPath
			if err := utils.CreateLink(srcPath, tgtPath, oldPath); err != nil {
				c.JSON(http.StatusOK, gin.H{
					"code":    model.StatusFile.Code,
					"message": model.StatusFile.Message,
					"detail":  err.Error(),
				})
				utils.Logger.Errorf("[%s]: %v", instanceName, err)
				return
			}
		}

		if err := updateIstInfo(instanceName, req.Task, req.Item, req.Value); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    model.StatusDatabase.Code,
				"message": model.StatusDatabase.Message,
				"detail":  err.Error(),
			})
			utils.Logger.Errorf("[%s]: %v", instanceName, err)
			return
		}

		if req.Task == "General" && req.Item == "language" {
			translation, err := getTranslation(instanceName, req.Value.(string))
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"code":    model.StatusFile.Code,
					"message": model.StatusFile.Message,
					"detail":  err.Error(),
				})
				return
			} else {
				c.JSON(http.StatusOK, gin.H{
					"code":        model.StatusSuccess.Code,
					"message":     model.StatusSuccess.Message,
					"detail":      "",
					"translation": translation,
				})
				return
			}
		}

	} else {
		if status, err := updateIstConf(instanceName, req); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    status.Code,
				"message": status.Message,
				"detail":  err.Error(),
			})
			utils.Logger.Errorf("[%s]: %v", instanceName, err)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    model.StatusSuccess.Code,
		"message": model.StatusSuccess.Message,
		"detail":  "",
	})
}

func DeleteInstance(c *gin.Context) {
	instanceName := c.Param("instance_name")

	if err := model.DeleteIstInfoByName(instanceName); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    model.StatusDatabase.Code,
			"message": model.StatusDatabase.Message,
			"detail":  err.Error(),
		})
		utils.Logger.Errorf("[%s]: %v", instanceName, err)
		return
	}
	if err := model.DeleteIstConfByName(instanceName); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    model.StatusFile.Code,
			"message": model.StatusFile.Message,
			"detail":  err.Error(),
		})
		utils.Logger.Errorf("[%s]: %v", instanceName, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    model.StatusSuccess.Code,
		"message": model.StatusSuccess.Message,
		"detail":  "",
	})
}

func fromLocal(req model.ReqFromLocal) (model.Status, error) {
	// 1. Read the template file content
	templateDir := req.TemplatePath
	template := model.NewTplConf()
	if err := template.Load(templateDir); err != nil {
		return model.StatusFile, err
	}

	// 2. Create template information and write to the database
	var templateInfo model.TemplateInfo
	if err := templateInfo.Create(req.TemplateName, req.TemplatePath); err != nil {
		return model.StatusDatabase, err
	}

	// 3. Create instance information and write to the database
	var instanceInfo model.InstanceInfo
	if err := instanceInfo.Create(req.InstanceName, req.TemplateName, template); err != nil {
		if errors.Is(err, sqlite3.CONSTRAINT) {
			return model.StatusDuplicate, err
		}
		return model.StatusDatabase, err
	}

	// 4. Create instance configuration file and save locally
	instanceConf := model.NewIstConf()
	if err := instanceConf.Create(req.InstanceName, template); err != nil {
		return model.StatusFile, err
	}

	// 5. Create configuration file symlink
	srcPath := filepath.Join("instances", req.InstanceName+".json")
	tgtPath := instanceInfo.ConfigPath
	if err := utils.CreateLink(srcPath, tgtPath, ""); err != nil {
		return model.StatusFile, err
	}

	return model.StatusSuccess, nil
}

func fromTemplate(req model.ReqFromTemplate) (model.Status, error) {
	// 1. Get template information from the database
	var templateInfo model.TemplateInfo
	if err := templateInfo.GetByName(req.TemplateName); err != nil {
		return model.StatusDatabase, err
	}

	// 2. Read template file content
	templateDir := templateInfo.Path
	template := model.NewTplConf()
	if err := template.Load(templateDir); err != nil {
		return model.StatusFile, err
	}

	// 3. Create instance information and write to the database
	var instanceInfo model.InstanceInfo
	if err := instanceInfo.Create(req.InstanceName, req.TemplateName, template); err != nil {
		if errors.Is(err, sqlite3.CONSTRAINT) {
			return model.StatusDuplicate, err
		}
		return model.StatusDatabase, err
	}
	instanceInfo.UpdateField("RepoURL", templateInfo.RepoURL)
	instanceInfo.UpdateField("LocalPath", templateInfo.LocalPath)
	instanceInfo.UpdateField("TemplateRelPath", templateInfo.TemplateRelPath)

	// 4. Create instance configuration file and save locally
	instanceConf := model.NewIstConf()
	if err := instanceConf.Create(req.InstanceName, template); err != nil {
		return model.StatusFile, err
	}

	// 5. Create configuration file symlink
	srcPath := filepath.Join("instances", req.InstanceName+".json")
	tgtPath := instanceInfo.ConfigPath
	if err := utils.CreateLink(srcPath, tgtPath, ""); err != nil {
		return model.StatusFile, err
	}

	return model.StatusSuccess, nil
}

func getRepoName(url string) string {
	// Remove .git suffix if exists
	url = strings.TrimSuffix(url, ".git")

	// Split by / and get last part
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}

func fromRemote(req model.ReqFromRemote) (model.Status, error) {
	// 1. Clone the repository, TODO: check if the repository already exists
	cmdLog, err := utils.GitClone(req.URL, req.LocalPath, req.Branch)
	utils.Logger.Infof("[%s]: %s", req.InstanceName, cmdLog)
	if err != nil && !errors.Is(err, git.ErrRepositoryAlreadyExists) {
		return model.StatusGit, err
	}

	// 2. Create a new instance, with operations similar to local creation
	repoName := getRepoName(req.URL)
	templatePath := filepath.Join(req.LocalPath, repoName, req.TemplateRelPath)
	if status, err := fromLocal(model.ReqFromLocal{
		InstanceName: req.InstanceName,
		TemplateName: req.TemplateName,
		TemplatePath: templatePath,
	}); err != nil {
		return status, err
	}

	// 3. Set repository information
	var instanceInfo model.InstanceInfo
	if err := instanceInfo.GetByName(req.InstanceName); err != nil {
		return model.StatusDatabase, err
	}
	instanceInfo.UpdateField("RepoURL", req.URL)
	instanceInfo.UpdateField("LocalPath", filepath.Join(req.LocalPath, repoName))
	instanceInfo.UpdateField("TemplateRelPath", req.TemplateRelPath)
	instanceInfo.UpdateField("Branch", req.Branch)
	model.SetBackup(req.TemplateName, req.URL, filepath.Join(req.LocalPath, repoName), req.TemplateRelPath)

	return model.StatusSuccess, nil
}

func getLangeuageList(i18nPath string) []string {
	langs := []string{}
	if files, err := os.ReadDir(i18nPath); err == nil {
		for _, file := range files {
			// Check if it's a JSON file
			if !file.IsDir() {
				name := file.Name()
				ext := filepath.Ext(name)
				if ext == ".json" {
					langName := strings.TrimSuffix(name, ".json")
					langs = append(langs, langName)
				}
			}
		}
	}
	return langs
}

func makeupLayout(istInfo *model.InstanceInfo, istConf *model.InstanceConf, tplConf *model.TemplateConf) (any, string) {
	layout := orderedmap.New[string, any]()

	// Project main content consists of DaCapo built-in settings retrieved from InstanceInfo
	menuProject := orderedmap.New[string, any]()
	layout.Set("Project", menuProject)

	// General stores project global settings
	taskGeneral := orderedmap.New[string, any]()
	menuProject.Set("General", taskGeneral)
	// DaCapo built-in settings
	groupGeneralBase := orderedmap.New[string, model.ItemConf]()
	taskGeneral.Set("_Base", groupGeneralBase)

	// Check language files in the template i18n directory
	// If the preset language doesn't exist, use the first available language
	i18nPath := filepath.Join(filepath.Dir(tplConf.Path), "i18n")
	langs := getLangeuageList(i18nPath)
	var lang string
	if slices.Contains(langs, istInfo.Language) {
		lang = istInfo.Language
	} else if len(langs) > 0 {
		lang = langs[0]
	}
	langOptions := make([]any, len(langs))
	for i, lang := range langs {
		langOptions[i] = lang
	}
	itemLanuage := model.ItemConf{
		Type:   "select",
		Value:  lang,
		Option: langOptions,
	}
	groupGeneralBase.Set("language", itemLanuage)

	itemWorkDir := model.ItemConf{
		Type:     "folder",
		Value:    istInfo.WorkDir,
		Disabled: istInfo.WorkDirDisabled,
	}
	groupGeneralBase.Set("work_dir", itemWorkDir)

	itemBackground := model.ItemConf{
		Type:     "checkbox",
		Value:    istInfo.Background,
		Disabled: istInfo.BackgroundDisabled,
	}
	groupGeneralBase.Set("background", itemBackground)

	itemConfigPath := model.ItemConf{
		Type:     "folder",
		Value:    istInfo.ConfigPath,
		Disabled: istInfo.ConfigPathDisabled,
	}
	groupGeneralBase.Set("config_path", itemConfigPath)

	itemLogPath := model.ItemConf{
		Type:     "input",
		Value:    istInfo.LogPath,
		Disabled: istInfo.LogPathDisabled,
	}
	groupGeneralBase.Set("log_path", itemLogPath)

	itemCronExpr := model.ItemConf{
		Type:  "cron",
		Value: istInfo.CronExpr,
	}
	groupGeneralBase.Set("cron_expr", itemCronExpr)

	// Custom settings from template configuration file
	if tplMenuProject, ok := tplConf.OM.Get("Project"); ok {
		if tplTaskGeneral, ok := tplMenuProject.Get("General"); ok {
			for pair := tplTaskGeneral.Oldest(); pair != nil; pair = pair.Next() {
				groupName := pair.Key
				groupConf := pair.Value
				if groupName == "_Base" {
					continue
				}

				taskGeneral.Set(groupName, groupConf)
				// Change values from template defaults to instance values
				for pair := groupConf.Oldest(); pair != nil; pair = pair.Next() {
					itemName := pair.Key
					itemConf := &pair.Value
					if itemValue := istConf.GetValue("Project", "General", groupName, itemName); itemValue != nil {
						itemConf.Value = itemValue
					}
				}
			}
		}
	}

	// Update stores project update settings
	if istInfo.RepoURL != "" {
		taskUpdate := orderedmap.New[string, any]()
		menuProject.Set("Update", taskUpdate)
		gourpUpdateBase := orderedmap.New[string, model.ItemConf]()
		taskUpdate.Set("_Base", gourpUpdateBase)

		itemRepoURL := model.ItemConf{
			Type:     "input",
			Value:    istInfo.RepoURL,
			Disabled: true,
		}
		gourpUpdateBase.Set("repo_url", itemRepoURL)

		itemBranch := model.ItemConf{
			Type:     "input",
			Value:    istInfo.Branch,
			Disabled: true,
		}
		gourpUpdateBase.Set("branch", itemBranch)

		itemLocalPath := model.ItemConf{
			Type:     "input",
			Value:    istInfo.LocalPath,
			Disabled: true,
		}
		gourpUpdateBase.Set("local_path", itemLocalPath)

		itemTemplateRelPath := model.ItemConf{
			Type:     "input",
			Value:    istInfo.TemplateRelPath,
			Disabled: true,
		}
		gourpUpdateBase.Set("template_rel_path", itemTemplateRelPath)

		itemAutoUpdate := model.ItemConf{
			Type:  "checkbox",
			Value: istInfo.AutoUpdate,
		}
		gourpUpdateBase.Set("auto_update", itemAutoUpdate)

		itemEnvName := model.ItemConf{
			Type:  "input",
			Value: istInfo.EnvName,
		}
		gourpUpdateBase.Set("env_name", itemEnvName)

		itemDepsPath := model.ItemConf{
			Type:     "input",
			Value:    istInfo.DepsPath,
			Disabled: istInfo.DepsPathDisabled,
		}
		gourpUpdateBase.Set("deps_path", itemDepsPath)

		itemPythonExec := model.ItemConf{
			Type:  "file",
			Value: istInfo.PythonExec,
		}
		gourpUpdateBase.Set("python_exec", itemPythonExec)
	}

	// Other custom settings
	for pair := tplConf.OM.Oldest(); pair != nil; pair = pair.Next() {
		menuName := pair.Key
		menuConf := pair.Value
		if menuName == "Project" {
			continue
		}

		newMenu := orderedmap.New[string, any]()
		layout.Set(menuName, newMenu)
		for pair := menuConf.Oldest(); pair != nil; pair = pair.Next() {
			taskName := pair.Key
			taskConf := pair.Value
			newTask := orderedmap.New[string, any]()
			newMenu.Set(taskName, newTask)
			// DaCapo built-in settings
			newGroupBase := orderedmap.New[string, model.ItemConf]()
			newTask.Set("_Base", newGroupBase)
			taskInfo := istInfo.GetTaskByName(taskName)

			itemActive := model.ItemConf{
				Type:     "checkbox",
				Value:    *taskInfo.Active,
				Disabled: taskInfo.ActiveDisabled,
			}
			newGroupBase.Set("active", itemActive)

			itemPriority := model.ItemConf{
				Type:     "priority",
				Value:    taskInfo.Priority,
				Disabled: taskInfo.PriorityDisabled,
			}
			newGroupBase.Set("priority", itemPriority)

			itemCommand := model.ItemConf{
				Type:     "input",
				Value:    taskInfo.Command,
				Disabled: taskInfo.CommandDisabled,
			}
			newGroupBase.Set("command", itemCommand)

			// Custom settings from template configuration file
			for pair := taskConf.Oldest(); pair != nil; pair = pair.Next() {
				groupName := pair.Key
				groupConf := pair.Value
				if groupName == "_Base" {
					continue
				}

				newTask.Set(groupName, groupConf)
				for pair := groupConf.Oldest(); pair != nil; pair = pair.Next() {
					itemName := pair.Key
					itemConf := &pair.Value
					if itemValue := istConf.GetValue(menuName, taskName, groupName, itemName); itemValue != nil {
						itemConf.Value = itemValue
					}
				}
			}
		}
	}
	return layout, lang
}

func getTranslation(istName, langName string) (any, error) {
	var istInfo model.InstanceInfo
	if err := istInfo.GetByName(istName); err != nil {
		return nil, err
	}
	var tplInfo model.TemplateInfo
	if err := tplInfo.GetByName(istInfo.TemplateName); err != nil {
		return nil, err
	}
	langPath := filepath.Join(tplInfo.Path, "i18n", langName+".json")

	content, err := os.ReadFile(langPath)
	if err != nil {
		return nil, err
	}

	var translation any
	if err := json.Unmarshal(content, &translation); err != nil {
		return nil, err
	}
	return translation, nil
}

func getSingleIst(instanceName string) (string, bool, any, any, model.Status, error) {
	// 1. Get instance information from the database
	var instanceInfo model.InstanceInfo
	if err := instanceInfo.GetByName(instanceName); err != nil {
		return "", false, nil, nil, model.StatusDatabase, err
	}
	ready := instanceInfo.Ready
	templateName := instanceInfo.TemplateName

	// 2. Get specific task settings from the instance configuration file
	instanceConf := model.NewIstConf()
	if err := instanceConf.Load(instanceName); err != nil {
		var pathErr *os.PathError
		if errors.As(err, &pathErr) {
			// If local config file doesn't exist, also delete the database record
			if err := instanceInfo.Delete(); err != nil {
				return "", false, nil, nil, model.StatusDatabase, err
			}
			utils.Logger.Warnf("[%s]: Instance config file not found, delete database record", instanceName)
		}
		return "", false, nil, nil, model.StatusFile, err
	}

	// 3. Get template information from the database
	var templateInfo model.TemplateInfo
	if err := templateInfo.GetByName(instanceInfo.TemplateName); err != nil {
		return "", false, nil, nil, model.StatusDatabase, err
	}

	// 4. Get layout information from the template file
	templateDir := templateInfo.Path
	templateConf := model.NewTplConf()
	if err := templateConf.Load(templateDir); err != nil {
		return "", false, nil, nil, model.StatusFile, err
	}

	// 5. Combine layout information
	layout, lang := makeupLayout(&instanceInfo, instanceConf, templateConf)
	var translation any
	if lang != "" {
		var err error
		translation, err = getTranslation(instanceName, lang)
		if err != nil {
			return "", false, nil, nil, model.StatusFile, err
		}
	}

	return templateName, ready, layout, translation, model.StatusSuccess, nil
}

func getAllIst() ([]string, map[string]bool, any, map[string]any, model.Status, error) {
	workingTemplate := []string{}
	ready := map[string]bool{}
	layout := orderedmap.New[string, any]()
	translation := map[string]any{}

	// Get layout information for all instances
	instanceNames, err := model.GetAllIstNames()
	if err != nil {
		return nil, nil, nil, nil, model.StatusDatabase, err
	}
	for _, instanceName := range instanceNames {
		singleTemplate, singleReady, singleLayout, singleTranslation, _, err := getSingleIst(instanceName)
		if err != nil {
			utils.Logger.Warnf("[%s]: Get instance layout failed: %v", instanceName, err)
			continue
		}

		workingTemplate = append(workingTemplate, singleTemplate)
		ready[instanceName] = singleReady
		layout.Set(instanceName, singleLayout)
		translation[instanceName] = singleTranslation
	}
	return workingTemplate, ready, layout, translation, model.StatusSuccess, nil
}

func updateIstInfo(instanceName, taskName, fieldName string, fieldValue any) error {
	var instanceInfo model.InstanceInfo
	if err := instanceInfo.GetByName(instanceName); err != nil {
		return err
	}

	if taskName == "General" || taskName == "Update" {
		if err := instanceInfo.UpdateField(fieldName, fieldValue); err != nil {
			return err
		}
	} else {
		if err := instanceInfo.UpdateTask(taskName, fieldName, fieldValue); err != nil {
			return err
		}
	}

	return nil
}

func updateIstConf(instanceName string, req model.ReqUpdateInstance) (model.Status, error) {
	instanceConf := model.NewIstConf()
	if err := instanceConf.Load(instanceName); err != nil {
		return model.StatusFile, err
	}

	if err := instanceConf.SetValue(req.Menu, req.Task, req.Group, req.Item, req.Value); err != nil {
		return model.StatusFile, err
	}
	return model.StatusSuccess, nil
}
