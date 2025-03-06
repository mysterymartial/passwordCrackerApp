package handler

import (
	"github.com/gin-gonic/gin"
	"passwordCrakerBackend/internal/core/domain"
	"passwordCrakerBackend/internal/core/service"
)

type CrackingHandler struct {
	crackingService *service.CrackingService
}

type CrackRequest struct {
	Hash     string                  `json:"hash" binding:"required"`
	Settings domain.CrackingSettings `json:"settings" binding:"required"`
}

func NewCrackingHandler(svc *service.CrackingService) *CrackingHandler {
	return &CrackingHandler{
		crackingService: svc,
	}
}

func (h *CrackingHandler) StartCracking(c *gin.Context) {
	var req CrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

func (h *CrackingHandler) GetProgress(c *gin.Context) {
	jobID := c.Param("jobId")
	progress, err := h.crackingService.GetProgress(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, progress)
}

func (h *CrackingHandler) StopCracking(c *gin.Context) {
	jobID := c.Param("jobId")
	err := h.crackingService.StopCracking(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Cracking stopped successfully"})
}

func (h *CrackingHandler) GetResults(c *gin.Context) {
	jobID := c.Param("jobId")
	results, err := h.crackingService.GetResults(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, results)
}

func (h *CrackingHandler) ListJobs(c *gin.Context) {
	jobs, err := h.crackingService.ListJobs(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, jobs)
}
