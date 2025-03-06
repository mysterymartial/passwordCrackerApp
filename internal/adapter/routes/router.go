package routes

import (
	"github.com/gin-gonic/gin"
	"passwordCrakerBackend/internal/adapter/http"
)

func SetupRoutes(r *gin.Engine, h *handler.CrackingHandler) {
	api := r.Group("/api/v1")
	{
		api.POST("/crack", h.StartCracking)
		api.GET("/progress/:jobId", h.GetProgress)
		api.POST("/stop/:jobId", h.StopCracking)
		api.GET("/results/:jobId", h.GetResults)
		api.GET("/jobs", h.ListJobs)
	}
}
