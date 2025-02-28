package controller

import (
	"dacapo/backend/model"
	"dacapo/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetTemplate(c *gin.Context) {
	names, err := model.GetAllTplNames()
	if err != nil {
		c.JSON(http.StatusOK, model.RspGetTemplate{
			Code:      model.StatusDatabase.Code,
			Message:   model.StatusDatabase.Message,
			Detail:    err.Error(),
			Templates: nil,
		})
		utils.Logger.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.RspGetTemplate{
		Code:      model.StatusSuccess.Code,
		Message:   model.StatusSuccess.Message,
		Detail:    "",
		Templates: names,
	})
}

func DeleteTemplate(c *gin.Context) {
	templateName := c.Param("template_name")

	err := model.DeleteTplInfoByName(templateName)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    model.StatusDatabase.Code,
			"message": model.StatusDatabase.Message,
			"detail":  err.Error(),
		})
		utils.Logger.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    model.StatusSuccess.Code,
		"message": model.StatusSuccess.Message,
		"detail":  "",
	})
}
