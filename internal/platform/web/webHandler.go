package web

import (
	"github.com/gin-gonic/gin"
	"passwordCrakerBackend/internal/core/domain"
	"passwordCrakerBackend/internal/core/service"
)

type CrackingRequest struct {
	Hash     string                  `json:"hash"`
	Settings domain.CrackingSettings `json:"settings"`
}

type WebHandler struct {
	crackingService *service.CrackingService
}

func NewWebHandler(svc *service.CrackingService) *WebHandler {
	return &WebHandler{
		crackingService: svc,
	}
}

func (h *WebHandler) StartCracking(c *gin.Context) {
	var req CrackingRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	job, err := h.crackingService.StartCracking(c.Request.Context(), req.Hash, req.Settings)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, job)
}

func (h *WebHandler) GetProgress(c *gin.Context) {
	jobID := c.Param("jobId")
	progress, err := h.crackingService.GetProgress(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, progress)
}

func (h *WebHandler) StopCracking(c *gin.Context) {
	jobID := c.Param("jobId")
	err := h.crackingService.StopCracking(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"status":  "success",
		"message": "Cracking job stopped",
		"jobId":   jobID,
	})
}
