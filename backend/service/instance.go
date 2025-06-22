package service

import (
	"dacapo/backend/model"
	"dacapo/backend/utils"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ncruces/go-sqlite3"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type InstanceService struct{}

// CreateFromLocal creates instance from local template
func (s *InstanceService) CreateFromLocal(req model.ReqFromLocal) (model.Status, error) {
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

// CreateFromTemplate creates instance from existing template
func (s *InstanceService) CreateFromTemplate(req model.ReqFromTemplate) (model.Status, error) {
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

// getRepoName extracts repository name from URL
func (s *InstanceService) getRepoName(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		repoName := parts[len(parts)-1]
		repoName = strings.TrimSuffix(repoName, ".git")
		return repoName
	}
	return "repo"
}

// CreateFromRemote creates instance from remote repository
func (s *InstanceService) CreateFromRemote(req model.ReqFromRemote) (model.Status, error) {
	// 1. Clone the repository
	repoName := s.getRepoName(req.URL)
	cmdLog, err := utils.GitClone(req.URL, req.LocalPath, req.Branch)
	utils.Logger.Infof("[%s]: %s", req.InstanceName, cmdLog)
	if err != nil {
		return model.StatusGit, err
	}

	// 2. Create a new instance, with operations similar to local creation
	templatePath := filepath.Join(req.LocalPath, repoName, req.TemplateRelPath)
	if status, err := s.CreateFromLocal(model.ReqFromLocal{
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

// GetAllInstances retrieves layout for all instances
func (s *InstanceService) GetAllInstances() ([]string, map[string]bool, any, map[string]any, model.Status, error) {
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
		singleTemplate, singleReady, singleLayout, singleTranslation, _, err := s.GetSingleInstance(instanceName)
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

// GetSingleInstance retrieves layout for a single instance
func (s *InstanceService) GetSingleInstance(instanceName string) (string, bool, any, any, model.Status, error) {
	// 1. Get instance information from the database
	var instanceInfo model.InstanceInfo
	if err := instanceInfo.GetByName(instanceName); err != nil {
		return "", false, nil, nil, model.StatusDatabase, err
	}

	// 2. Get instance configuration from the file
	instanceConf := model.NewIstConf()
	if err := instanceConf.Load(instanceName); err != nil {
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
	layout, lang := s.BuildLayout(&instanceInfo, instanceConf, templateConf)
	var translation any
	if lang != "" {
		var err error
		translation, err = s.GetTranslation(instanceName, lang)
		if err != nil {
			return "", false, nil, nil, model.StatusFile, err
		}
	}

	return templateInfo.Name, instanceInfo.Ready, layout, translation, model.StatusSuccess, nil
}

// UpdateInstance updates instance configuration
func (s *InstanceService) UpdateInstance(instanceName, menuName, taskName, groupName, itemName string, value any) (any, error) {
	// Handle symlink creation for folder types
	if groupName == "_Base" && itemName == "work_dir" {
		oldPath, _ := value.(string)
		srcPath := filepath.Join("instances", instanceName+".json")
		tgtPath, _ := value.(string)
		if err := utils.CreateLink(srcPath, tgtPath, oldPath); err != nil {
			return nil, err
		}
	}

	if err := s.updateInstanceInfo(instanceName, taskName, itemName, value); err != nil {
		return nil, err
	}

	if taskName == "General" && itemName == "language" {
		translation, err := s.GetTranslation(instanceName, value.(string))
		if err != nil {
			return nil, err
		}
		return translation, nil
	}

	return nil, nil
}

// DeleteInstance deletes an instance
func (s *InstanceService) DeleteInstance(instanceName string) error {
	if err := model.DeleteIstInfoByName(instanceName); err != nil {
		return err
	}
	if err := model.DeleteIstConfByName(instanceName); err != nil {
		return err
	}
	return nil
}

// getLangeuageList gets available language files from i18n directory
func (s *InstanceService) getLangeuageList(i18nPath string) []string {
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

// BuildLayout builds the layout for an instance
func (s *InstanceService) BuildLayout(istInfo *model.InstanceInfo, istConf *model.InstanceConf, tplConf *model.TemplateConf) (any, string) {
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
	langs := s.getLangeuageList(i18nPath)
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

		itemPythonVersion := model.ItemConf{
			Type:  "select",
			Value: istInfo.PythonVersion,
			Option: []any{
				"3.8",
				"3.9",
				"3.10",
				"3.11",
				"3.12",
				"3.13",
				"3.14",
				"",
			},
		}
		gourpUpdateBase.Set("python_version", itemPythonVersion)
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

// GetTranslation gets translation data for an instance
func (s *InstanceService) GetTranslation(istName, langName string) (any, error) {
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

// updateInstanceInfo updates instance information in database
func (s *InstanceService) updateInstanceInfo(instanceName, taskName, fieldName string, fieldValue any) error {
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
