package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// Stores actual user settings, can be read by other projects
type InstanceConf struct {
	Name string
	OM   *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, any]]]]
}

func NewIstConf() *InstanceConf {
	return &InstanceConf{
		OM: orderedmap.New[string, *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, any]]]](),
	}
}

func DeleteIstConfByName(istName string) (err error) {
	filePath := filepath.Join("instances", istName+".json")
	if err = os.Remove(filePath); err != nil {
		return
	}

	return nil
}

func (i *InstanceConf) save() (err error) {
	jsonData, err := json.MarshalIndent(i.OM, "", "  ")
	if err != nil {
		return
	}

	filePath := filepath.Join("instances", i.Name+".json")
	if err = os.WriteFile(filePath, jsonData, 0644); err != nil {
		return
	}

	return nil
}

func (i *InstanceConf) Create(istName string, tpl *TemplateConf) (err error) {
	i.Name = istName

	istDir := "instances"
	if err = os.MkdirAll(istDir, 0755); err != nil {
		return
	}

	// Iterate through Template, maintaining order
	for pair := tpl.OM.Oldest(); pair != nil; pair = pair.Next() {
		menuName := pair.Key
		menuConf := pair.Value

		// Initialize menu layer
		menuMap := orderedmap.New[string, *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, any]]]()
		i.OM.Set(menuName, menuMap)

		for pair := menuConf.Oldest(); pair != nil; pair = pair.Next() {
			taskName := pair.Key
			taskConf := pair.Value

			// Initialize task layer
			taskMap := orderedmap.New[string, *orderedmap.OrderedMap[string, any]]()
			menuMap.Set(taskName, taskMap)

			for pair := taskConf.Oldest(); pair != nil; pair = pair.Next() {
				groupName := pair.Key
				groupConf := pair.Value
				if groupName != "_Base" {
					// Initialize group layer
					groupMap := orderedmap.New[string, any]()
					taskMap.Set(groupName, groupMap)

					for pair := groupConf.Oldest(); pair != nil; pair = pair.Next() {
						itemName := pair.Key
						itemConf := pair.Value
						if itemName != "_help" {
							groupMap.Set(itemName, itemConf.Value)
						}
					}
				}
			}
		}
	}

	if err = i.save(); err != nil {
		return
	}

	return nil
}

func (i *InstanceConf) Update(tpl *TemplateConf) (err error) {
	updatedOM := orderedmap.New[string, *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, any]]]]()

	// Iterate through Template, maintaining order
	for pair := tpl.OM.Oldest(); pair != nil; pair = pair.Next() {
		menuName := pair.Key
		menuConf := pair.Value

		// Initialize menu layer
		menuMap := orderedmap.New[string, *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, any]]]()
		updatedOM.Set(menuName, menuMap)

		for pair := menuConf.Oldest(); pair != nil; pair = pair.Next() {
			taskName := pair.Key
			taskConf := pair.Value

			// Initialize task layer
			taskMap := orderedmap.New[string, *orderedmap.OrderedMap[string, any]]()
			menuMap.Set(taskName, taskMap)

			for pair := taskConf.Oldest(); pair != nil; pair = pair.Next() {
				groupName := pair.Key
				groupConf := pair.Value
				if groupName != "_Base" {
					// Initialize group layer
					groupMap := orderedmap.New[string, any]()
					taskMap.Set(groupName, groupMap)

					for pair := groupConf.Oldest(); pair != nil; pair = pair.Next() {
						itemName := pair.Key
						itemConf := pair.Value
						if itemName != "_help" {
							// Check if the value exists in the current instance
							existingValue := i.GetValue(menuName, taskName, groupName, itemName)
							if existingValue != nil {
								// Keep existing value
								groupMap.Set(itemName, existingValue)
							} else {
								// Use template default value for new items
								groupMap.Set(itemName, itemConf.Value)
							}
						}
					}
				}
			}
		}
	}

	// Replace the current ordered map with the updated one
	i.OM = updatedOM

	if err = i.save(); err != nil {
		return
	}

	return nil
}

func (i *InstanceConf) Load(istName string) (err error) {
	i.Name = istName
	filePath := filepath.Join("instances", istName+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	if err = json.Unmarshal(data, i.OM); err != nil {
		return
	}

	return nil
}

func (i *InstanceConf) GetValue(menuName, taskName, groupName, itemName string) any {
	menuConf, exists := i.OM.Get(menuName)
	if !exists {
		return nil
	}
	taskConf, exists := menuConf.Get(taskName)
	if !exists {
		return nil
	}
	groupConf, exists := taskConf.Get(groupName)
	if !exists {
		return nil
	}
	itemValue, exists := groupConf.Get(itemName)
	if !exists {
		return nil
	}
	return itemValue
}

func (i *InstanceConf) SetValue(menuName, taskName, groupName, itemName string, value any) (err error) {
	menuConf, exists := i.OM.Get(menuName)
	if !exists {
		return fmt.Errorf("menu %s not exists", menuName)
	}
	taskConf, exists := menuConf.Get(taskName)
	if !exists {
		return fmt.Errorf("task %s not exists", taskName)
	}
	groupConf, exists := taskConf.Get(groupName)
	if !exists {
		return fmt.Errorf("group %s not exists", groupName)
	}
	groupConf.Set(itemName, value)

	if err = i.save(); err != nil {
		return
	}

	return nil
}
