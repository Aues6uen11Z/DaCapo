package controller

import (
	"dacapo/backend/model"
	"dacapo/backend/utils"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func CreateIstFromLocal(c *gin.Context) {
	var req model.ReqFromLocal
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		utils.Logger.Error("Invalid request format\n", err)
		return
	}

	instanceService := Services.InstanceService()
	if status, err := instanceService.CreateFromLocal(req); err != nil {
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

	instanceService := Services.InstanceService()
	if status, err := instanceService.CreateFromTemplate(req); err != nil {
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

	instanceService := Services.InstanceService()
	if status, err := instanceService.CreateFromRemote(req); err != nil {
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
	instanceService := Services.InstanceService()
	workingTpl, ready, layout, translation, status, err := instanceService.GetAllInstances()
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

	instanceService := Services.InstanceService()
	templateName, ready, layout, translation, status, err := instanceService.GetSingleInstance(instanceName)
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

	instanceService := Services.InstanceService()

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
				return
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

		translation, err := instanceService.UpdateInstance(instanceName, req.Menu, req.Task, req.Group, req.Item, req.Value)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    model.StatusDatabase.Code,
				"message": model.StatusDatabase.Message,
				"detail":  err.Error(),
			})
			utils.Logger.Errorf("[%s]: %v", instanceName, err)
			return
		}

		if translation != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":        model.StatusSuccess.Code,
				"message":     model.StatusSuccess.Message,
				"detail":      "",
				"translation": translation,
			})
			return
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

	instanceService := Services.InstanceService()
	if err := instanceService.DeleteInstance(instanceName); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    model.StatusDatabase.Code,
			"message": model.StatusDatabase.Message,
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
