package router

import (
	"bytes"
	"dacapo/backend/controller"
	"dacapo/backend/utils"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read request body
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
		}

		// Restore body for next handlers
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Log request details
		if len(bodyBytes) > 0 {
			utils.Logger.Debugf("Request Body: %s", string(bodyBytes))
		}

		c.Next()
	}
}

func SetupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.MultiWriter(utils.Logfile, os.Stdout)

	r := gin.New()
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s\tINFO\t%3d | %8s | %15s | %-7s %#v\n%s",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.Request.RemoteAddr,
			param.Method,
			param.Path,
			param.ErrorMessage,
		)
	}))
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://wails.localhost", "http://wails.localhost:34115", "http://localhost:33204"},
		AllowMethods:     []string{"GET", "POST", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Upgrade", "Connection"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	// r.Use(requestLogger())

	api := r.Group("/api")
	{
		ist := api.Group("/instance")
		{
			ist.POST("/local", controller.CreateIstFromLocal)
			ist.POST("/template", controller.CreateIstFromTemplate)
			ist.POST("/remote", controller.CreateIstFromRemote)
			ist.GET("", controller.GetAllInstances)
			ist.GET("/:instance_name", controller.GetInstance)
			ist.PATCH("/:instance_name", controller.UpdateInstance)
			ist.DELETE("/:instance_name", controller.DeleteInstance)
		}

		tpl := api.Group("/template")
		{
			tpl.GET("", controller.GetTemplate)
			tpl.DELETE("/:template_name", controller.DeleteTemplate)
		}

		scheduler := api.Group("/scheduler")
		{
			scheduler.GET("/ws", controller.CreateWS)
			scheduler.PATCH("/queue", controller.UpdateTaskQueue)
			scheduler.PATCH("/state", controller.UpdateSchedulerState)
			scheduler.GET("/queue/:instance_name", controller.GetTaskQueue)
			scheduler.POST("/cron", controller.SetSchedulerCron)
		}

		api.GET("/updater/:instance_name", controller.UpdateRepo)
	}

	return r
}
