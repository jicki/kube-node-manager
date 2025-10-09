package gitlab

import (
	"kube-node-manager/internal/service/gitlab"
	"kube-node-manager/pkg/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler handles GitLab-related HTTP requests
type Handler struct {
	service *gitlab.Service
	logger  *logger.Logger
}

// NewHandler creates a new GitLab handler
func NewHandler(service *gitlab.Service, logger *logger.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// GetSettings retrieves GitLab settings
// GET /api/v1/gitlab/settings
func (h *Handler) GetSettings(c *gin.Context) {
	settings, err := h.service.GetSettings()
	if err != nil {
		h.logger.Error("Failed to get GitLab settings: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get GitLab settings"})
		return
	}

	c.JSON(http.StatusOK, settings.ToResponse())
}

// UpdateSettings updates GitLab settings
// PUT /api/v1/gitlab/settings
func (h *Handler) UpdateSettings(c *gin.Context) {
	var req struct {
		Enabled bool   `json:"enabled"`
		Domain  string `json:"domain"`
		Token   string `json:"token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Validate domain if enabled
	if req.Enabled && req.Domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Domain is required when GitLab is enabled"})
		return
	}

	settings, err := h.service.UpdateSettings(req.Enabled, req.Domain, req.Token)
	if err != nil {
		h.logger.Error("Failed to update GitLab settings: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update GitLab settings"})
		return
	}

	h.logger.Info("GitLab settings updated successfully")
	c.JSON(http.StatusOK, settings.ToResponse())
}

// TestConnection tests GitLab API connection
// POST /api/v1/gitlab/test
func (h *Handler) TestConnection(c *gin.Context) {
	var req struct {
		Domain string `json:"domain" binding:"required"`
		Token  string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Domain and token are required"})
		return
	}

	if err := h.service.TestConnection(req.Domain, req.Token); err != nil {
		h.logger.Error("GitLab connection test failed: " + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Connection successful"})
}

// ListRunners lists all GitLab runners
// GET /api/v1/gitlab/runners
func (h *Handler) ListRunners(c *gin.Context) {
	runnerType := c.Query("type")
	status := c.Query("status")

	var paused *bool
	if pausedStr := c.Query("paused"); pausedStr != "" {
		p := pausedStr == "true"
		paused = &p
	}

	runners, err := h.service.ListRunners(runnerType, status, paused)
	if err != nil {
		h.logger.Error("Failed to list runners: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, runners)
}

// GetPipelineDetail retrieves detailed information for a specific pipeline
// GET /api/v1/gitlab/pipelines/:project_id/:pipeline_id
func (h *Handler) GetPipelineDetail(c *gin.Context) {
	projectIDStr := c.Param("project_id")
	pipelineIDStr := c.Param("pipeline_id")

	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id"})
		return
	}

	pipelineID, err := strconv.Atoi(pipelineIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pipeline_id"})
		return
	}

	pipelineDetail, err := h.service.GetPipelineDetail(projectID, pipelineID)
	if err != nil {
		h.logger.Error("Failed to get pipeline detail: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pipelineDetail)
}

// GetPipelineJobs retrieves jobs for a specific pipeline
// GET /api/v1/gitlab/pipelines/:project_id/:pipeline_id/jobs
func (h *Handler) GetPipelineJobs(c *gin.Context) {
	projectIDStr := c.Param("project_id")
	pipelineIDStr := c.Param("pipeline_id")

	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id"})
		return
	}

	pipelineID, err := strconv.Atoi(pipelineIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pipeline_id"})
		return
	}

	jobs, err := h.service.GetPipelineJobs(projectID, pipelineID)
	if err != nil {
		h.logger.Error("Failed to get pipeline jobs: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, jobs)
}

// ListPipelines lists pipelines for a project
// GET /api/v1/gitlab/pipelines
func (h *Handler) ListPipelines(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id is required"})
		return
	}

	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id"})
		return
	}

	ref := c.Query("ref")
	status := c.Query("status")

	// Parse pagination parameters
	page := 1
	perPage := 20
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if perPageStr := c.Query("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 {
			perPage = pp
		}
	}

	pipelines, err := h.service.ListPipelines(projectID, ref, status, page, perPage)
	if err != nil {
		h.logger.Error("Failed to list pipelines: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pipelines)
}

// GetRunner gets details of a specific runner
// GET /api/v1/gitlab/runners/:id
func (h *Handler) GetRunner(c *gin.Context) {
	runnerIDStr := c.Param("id")
	runnerID, err := strconv.Atoi(runnerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid runner ID"})
		return
	}

	runner, err := h.service.GetRunner(runnerID)
	if err != nil {
		h.logger.Error("Failed to get runner: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, runner)
}

// UpdateRunner updates a runner's configuration
// PUT /api/v1/gitlab/runners/:id
func (h *Handler) UpdateRunner(c *gin.Context) {
	runnerIDStr := c.Param("id")
	runnerID, err := strconv.Atoi(runnerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid runner ID"})
		return
	}

	var req gitlab.UpdateRunnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	runner, err := h.service.UpdateRunner(runnerID, req)
	if err != nil {
		h.logger.Error("Failed to update runner: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Runner updated successfully")
	c.JSON(http.StatusOK, runner)
}

// DeleteRunner deletes a runner
// DELETE /api/v1/gitlab/runners/:id
func (h *Handler) DeleteRunner(c *gin.Context) {
	runnerIDStr := c.Param("id")
	runnerID, err := strconv.Atoi(runnerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid runner ID"})
		return
	}

	if err := h.service.DeleteRunner(runnerID); err != nil {
		h.logger.Error("Failed to delete runner: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Runner deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Runner deleted successfully"})
}

// CreateRunner creates a new runner
// POST /api/v1/gitlab/runners
func (h *Handler) CreateRunner(c *gin.Context) {
	var req gitlab.CreateRunnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Get username from context (set by auth middleware)
	username, exists := c.Get("username")
	if !exists {
		username = "unknown"
	}

	runner, err := h.service.CreateRunner(req, username.(string))
	if err != nil {
		h.logger.Error("Failed to create runner: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Runner created successfully")
	c.JSON(http.StatusCreated, runner)
}

// GetRunnerToken gets the saved runner token
// GET /api/v1/gitlab/runners/:id/token
func (h *Handler) GetRunnerToken(c *gin.Context) {
	runnerIDStr := c.Param("id")
	runnerID, err := strconv.Atoi(runnerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid runner ID"})
		return
	}

	runner, err := h.service.GetRunnerToken(runnerID)
	if err != nil {
		h.logger.Error("Failed to get runner token: " + err.Error())
		c.JSON(http.StatusNotFound, gin.H{"error": "Runner token not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runner_id":   runner.RunnerID,
		"token":       runner.Token,
		"description": runner.Description,
		"runner_type": runner.RunnerType,
		"created_by":  runner.CreatedBy,
		"created_at":  runner.CreatedAt,
	})
}

// ResetRunnerToken resets a runner's authentication token
// POST /api/v1/gitlab/runners/:id/reset-token
func (h *Handler) ResetRunnerToken(c *gin.Context) {
	runnerIDStr := c.Param("id")
	runnerID, err := strconv.Atoi(runnerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid runner ID"})
		return
	}

	// Get username from context
	username, exists := c.Get("username")
	if !exists {
		username = "unknown"
	}

	runner, err := h.service.ResetRunnerToken(runnerID, username.(string))
	if err != nil {
		h.logger.Error("Failed to reset runner token: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Runner token reset successfully")
	c.JSON(http.StatusOK, runner)
}
