package controller

import (
	"dacapo/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func UpdateRepo(c *gin.Context) {
	instanceName := c.Param("instance_name")

	result, err := Services.InstanceUpdaterService().UpdateRepo(instanceName)
	if err != nil {
		utils.Logger.Errorf("[%s]: %v", instanceName, err)
	}

	c.JSON(http.StatusOK, result)
}
