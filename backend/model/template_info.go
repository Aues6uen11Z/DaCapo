package model

import "gorm.io/gorm"

// TemplateInfo stores information about available templates in the system
type TemplateInfo struct {
	gorm.Model

	Name string `gorm:"uniqueIndex;not null"`
	Path string `gorm:"not null"`

	// Backup of partial content from InstanceInfo
	RepoURL         string
	LocalPath       string
	TemplateRelPath string
}

// GetAllTplNames retrieves all template names from the database
func GetAllTplNames() ([]string, error) {
	var names []string
	err := db.Model(&TemplateInfo{}).Select("name").Find(&names).Error
	return names, err
}

// DeleteTplInfoByName permanently removes a template from the database by its name
func DeleteTplInfoByName(name string) error {
	err := db.Where("name = ?", name).Unscoped().Delete(&TemplateInfo{}).Error
	return err
}

// SetBackup updates template backup information
func SetBackup(name, repoURL, localPath, templateRelPath string) error {
	updates := map[string]any{
		"repo_url":          repoURL,
		"local_path":        localPath,
		"template_rel_path": templateRelPath,
	}
	err := db.Model(&TemplateInfo{}).Where("name = ?", name).Updates(updates).Error
	return err
}

// Create adds a new template to the database or returns silently if it already exists
func (t *TemplateInfo) Create(name, path string) error {
	if err := db.Where("name = ?", name).First(t).Error; err == nil {
		return nil
	}

	t.Name = name
	t.Path = path
	if err := db.Create(t).Error; err != nil {
		return err
	}
	return nil
}

// GetByName retrieves a template from the database by its name
func (t *TemplateInfo) GetByName(name string) error {
	err := db.Where("name = ?", name).First(t).Error
	return err
}
