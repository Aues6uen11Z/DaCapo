package model

import (
	"os"
	"path/filepath"
	"slices"
	"time"

	"gorm.io/gorm"
)

// TaskInfo represents an independent Task table
type TaskInfo struct {
	gorm.Model
	InstanceID uint `gorm:"index"`

	Name             string
	Active           bool `gorm:"default:true"`
	ActiveDisabled   bool
	Priority         uint
	PriorityDisabled bool
	Command          string
	CommandDisabled  bool
}

// InstanceInfo stores built-in DaCapo settings that are independent of specific templates
type InstanceInfo struct {
	gorm.Model

	Name             string `gorm:"uniqueIndex;not null"`
	TemplateName     string `gorm:"not null"`
	Ready            bool
	LayoutLastUpdate time.Time
	EnvLastUpdate    time.Time

	// general page
	Language           string
	WorkDir            string
	WorkDirDisabled    bool
	Background         bool
	BackgroundDisabled bool
	ConfigPath         string
	ConfigPathDisabled bool

	// auto-generated during instance creation, read-only
	RepoURL         string
	LocalPath       string
	TemplateRelPath string

	// update page
	Branch           string
	BranchDisabled   bool
	AutoUpdate       bool
	EnvName          string
	DepsPath         string `gorm:"default:'requirements.txt'"`
	DepsPathDisabled bool
	PythonExec       string

	Tasks []TaskInfo `gorm:"foreignKey:InstanceID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE"`
}

// GetAllInstances retrieves all instance information
func GetAllInstances(instances *[]InstanceInfo) error {
	return db.Find(instances).Error
}

// GetInstanceByName retrieves instance information by name
func GetInstanceByName(name string) (*InstanceInfo, error) {
	var instance InstanceInfo
	err := db.Where("name = ?", name).First(&instance).Error
	return &instance, err
}

// GetAllIstNames retrieves all instance names
func GetAllIstNames() ([]string, error) {
	var names []string
	err := db.Model(&InstanceInfo{}).Select("name").Find(&names).Error
	return names, err
}

// DeleteIstInfoByName deletes an instance by its name
func DeleteIstInfoByName(name string) error {
	var instance InstanceInfo
	if err := db.Where("name = ?", name).First(&instance).Error; err != nil {
		return err
	}

	// Remove config file if it exists
	if instance.ConfigPath != "" {
		if err := os.Remove(instance.ConfigPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	// Delete the instance from database
	err := instance.Delete()
	return err
}

func GetConfigPaths() ([][2]string, error) {
	type Result struct {
		Name       string
		ConfigPath string
	}

	var results []Result
	err := db.Model(&InstanceInfo{}).Select("name, config_path").Find(&results).Error

	paths := make([][2]string, len(results))
	for i, result := range results {
		paths[i][0] = filepath.Join("instances", result.Name+".json")
		paths[i][1] = result.ConfigPath
	}

	return paths, err
}

// Create initializes a new instance with values from a template
func (i *InstanceInfo) Create(istName, tplName string, tpl *TemplateConf) error {
	i.Name = istName
	i.TemplateName = tplName
	i.LayoutLastUpdate = time.Now()

	// Define property mappings to read from template and set in InstanceInfo struct
	propertySetters := map[string]map[string]func(item ItemConf){
		"General": {
			"language": func(item ItemConf) {
				if v, ok := item.Value.(string); ok {
					i.Language = v
				}
			},
			"work_dir": func(item ItemConf) {
				if v, ok := item.Value.(string); ok {
					i.WorkDir = v
					i.WorkDirDisabled = item.Disabled
				}
			},
			"background": func(item ItemConf) {
				if v, ok := item.Value.(bool); ok {
					i.Background = v
					i.BackgroundDisabled = item.Disabled
				}
			},
			"config_path": func(item ItemConf) {
				if v, ok := item.Value.(string); ok {
					i.ConfigPath = v
					i.ConfigPathDisabled = item.Disabled
				}
			},
		},
		"Update": {
			"branch": func(item ItemConf) {
				if v, ok := item.Value.(string); ok {
					i.Branch = v
					i.BranchDisabled = item.Disabled
				}
			},
			"auto_update": func(item ItemConf) {
				if v, ok := item.Value.(bool); ok {
					i.AutoUpdate = v
				}
			},
			"env_name": func(item ItemConf) {
				if v, ok := item.Value.(string); ok {
					i.EnvName = v
				}
			},
			"deps_path": func(item ItemConf) {
				if v, ok := item.Value.(string); ok {
					i.DepsPath = v
					i.DepsPathDisabled = item.Disabled
				}
			},
			"python_exec": func(item ItemConf) {
				if v, ok := item.Value.(string); ok {
					i.PythonExec = v
				}
			},
		},
	}

	for pair := tpl.OM.Oldest(); pair != nil; pair = pair.Next() {
		menuName := pair.Key
		menuConf := pair.Value
		if menuName == "Project" {
			// Process properties for General and Update tasks under Project menu
			for pair := menuConf.Oldest(); pair != nil; pair = pair.Next() {
				taskName := pair.Key
				taskConf := pair.Value
				if setters, ok := propertySetters[taskName]; ok {
					if baseGroup, ok := taskConf.Get("_Base"); ok {
						for itemName, setter := range setters {
							if itemConf, exists := baseGroup.Get(itemName); exists {
								setter(itemConf)
							}
						}
					}
				}
			}
		} else {
			// Process all tasks under other menus
			for pair := menuConf.Oldest(); pair != nil; pair = pair.Next() {
				taskName := pair.Key
				taskConf := pair.Value
				// Create new Task
				task := TaskInfo{
					Name:       taskName,
					InstanceID: i.ID, // Set relation ID
				}
				// Set basic task properties
				if baseGroup, ok := taskConf.Get("_Base"); ok {
					if activeConf, exists := baseGroup.Get("active"); exists {
						if v, ok := activeConf.Value.(bool); ok {
							task.Active = v
							task.ActiveDisabled = activeConf.Disabled
						}
					}
					if priorityConf, exists := baseGroup.Get("priority"); exists {
						switch v := priorityConf.Value.(type) {
						case uint:
							task.Priority = v
						case int:
							task.Priority = uint(v)
						case float64:
							task.Priority = uint(v)
						}
						task.PriorityDisabled = priorityConf.Disabled
					}
					if cmdConf, exists := baseGroup.Get("command"); exists {
						if v, ok := cmdConf.Value.(string); ok {
							task.Command = v
							task.CommandDisabled = cmdConf.Disabled
						}
					}
				}

				i.Tasks = append(i.Tasks, task)
			}
		}
	}

	if err := db.Create(i).Error; err != nil {
		return err
	}

	return nil
}

// GetByName retrieves an instance by name with its associated tasks
func (i *InstanceInfo) GetByName(name string) error {
	err := db.Preload("Tasks").Where("name = ?", name).First(i).Error
	return err
}

// Delete permanently removes the instance from the database
func (i *InstanceInfo) Delete() error {
	err := db.Unscoped().Delete(i).Error
	return err
}

// GetTaskByName retrieves a task from the instance by its name
func (i *InstanceInfo) GetTaskByName(name string) *TaskInfo {
	for _, task := range i.Tasks {
		if task.Name == name {
			return &task
		}
	}
	return nil
}

// UpdateField updates a specific field of the instance
func (i *InstanceInfo) UpdateField(fieldName string, value any) error {
	err := db.Model(i).Update(fieldName, value).Error
	return err
}

// UpdateTask updates a specific field of a task belonging to the instance
func (i *InstanceInfo) UpdateTask(taskName, fieldName string, value any) error {
	// Find task by name
	var task TaskInfo
	if err := db.Where("instance_id = ? AND name = ?", i.ID, taskName).First(&task).Error; err != nil {
		return err
	}

	// Update field
	if err := db.Model(&task).Update(fieldName, value).Error; err != nil {
		return err
	}

	return nil
}

// GetTaskQueue returns two slices: active tasks sorted by priority, and inactive tasks
func (i *InstanceInfo) GetTaskQueue() (waiting []string, stopped []string) {
	waiting = make([]string, 0, len(i.Tasks))
	stopped = make([]string, 0, len(i.Tasks))
	priorityMap := make(map[string]int, len(i.Tasks))

	for _, task := range i.Tasks {
		priorityMap[task.Name] = int(task.Priority)
		if task.Active {
			waiting = append(waiting, task.Name)
		} else {
			stopped = append(stopped, task.Name)
		}
	}
	slices.SortFunc(waiting, func(a, b string) int {
		return priorityMap[b] - priorityMap[a]
	})

	return
}
