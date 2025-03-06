package web

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.Engine, handler *WebHandler) {
	api := r.Group("/api")
	{
		api.POST("/crack", handler.StartCracking)
		api.GET("/progress/:jobId", handler.GetProgress)
		api.POST("/stop/:jobId", handler.StopCracking)
	}
}
