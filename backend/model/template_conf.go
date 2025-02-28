package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	orderedmap "github.com/wk8/go-ordered-map/v2"
	"gopkg.in/yaml.v3"
)

type ItemConf struct {
	Type     string `json:"type" yaml:"type"`
	Value    any    `json:"value" yaml:"value"`
	Help     string `json:"help,omitempty" yaml:"help,omitempty"`
	Option   []any  `json:"option,omitempty" yaml:"option,omitempty"`
	Hidden   bool   `json:"hidden,omitempty" yaml:"hidden,omitempty"`
	Disabled bool   `json:"disabled,omitempty" yaml:"disabled,omitempty"`
}

// TemplateConf represents a 4-layer nested structure using ordered maps: Menu->Task->Group->Item
type TemplateConf struct {
	Path string
	OM   *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, ItemConf]]]]
}

func NewTplConf() *TemplateConf {
	return &TemplateConf{
		OM: orderedmap.New[string, *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, *orderedmap.OrderedMap[string, ItemConf]]]](),
	}
}

func GetTplPath(dirPath, fileName string) (string, error) {
	exts := []string{".yml", ".yaml", ".json"}
	for _, ext := range exts {
		filePath := filepath.Join(dirPath, fileName+ext)
		if _, err := os.Stat(filePath); err == nil {
			return filePath, nil
		}
	}
	return "", fmt.Errorf("no config file found for %s in %s", fileName, dirPath)
}

func (t *TemplateConf) Load(dirPath string) (err error) {
	tplPath, err := GetTplPath(dirPath, "template")
	t.Path = tplPath
	if err != nil {
		return
	}
	data, err := os.ReadFile(tplPath)
	if err != nil {
		return
	}

	ext := strings.ToLower(filepath.Ext(tplPath))
	switch ext {
	case ".yaml", ".yml":
		if err = yaml.Unmarshal(data, t.OM); err != nil {
			return
		}
	case ".json":
		if err = json.Unmarshal(data, t.OM); err != nil {
			return
		}
	}

	return nil
}
