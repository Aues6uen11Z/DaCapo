package controller

import (
	"dacapo/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

var instanceUpdaterService = Services.InstanceUpdaterService()

func UpdateRepo(c *gin.Context) {
	instanceName := c.Param("instance_name")

	result, err := instanceUpdaterService.UpdateRepo(instanceName)
	if err != nil {
		utils.Logger.Errorf("[%s]: %v", instanceName, err)
	}

	c.JSON(http.StatusOK, result)
}
