package user

import (
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/user"
	"kube-node-manager/pkg/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *user.Service
	logger  *logger.Logger
}

func NewHandler(service *user.Service, logger *logger.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) List(c *gin.Context) {
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    http.StatusForbidden,
			"message": "Admin role required",
		})
		return
	}

	var req user.ListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Invalid request parameters: " + err.Error(),
		})
		return
	}

	resp, err := h.service.List(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to list users: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Success",
		"data":    resp,
	})
}

func (h *Handler) GetByID(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if userRole != model.RoleAdmin && uint(id) != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	user, err := h.service.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) Create(c *gin.Context) {
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin role required"})
		return
	}

	var req user.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operatorID := c.GetUint("user_id")
	user, err := h.service.Create(req, operatorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *Handler) Update(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if userRole != model.RoleAdmin && uint(id) != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req user.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if userRole != model.RoleAdmin && req.Role != "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admin can change user role"})
		return
	}

	operatorID := c.GetUint("user_id")
	user, err := h.service.Update(uint(id), req, operatorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) Delete(c *gin.Context) {
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin role required"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	operatorID := c.GetUint("user_id")
	if err := h.service.Delete(uint(id), operatorID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func (h *Handler) UpdatePassword(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if userRole != model.RoleAdmin && uint(id) != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req user.UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operatorID := c.GetUint("user_id")
	if err := h.service.UpdatePassword(uint(id), req, operatorID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}

// ResetPassword 管理员重置用户密码
func (h *Handler) ResetPassword(c *gin.Context) {
	userRole, _ := c.Get("user_role")

	// 只有管理员可以重置用户密码
	if userRole != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin role required"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req user.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operatorID := c.GetUint("user_id")
	if err := h.service.ResetPassword(uint(id), req, operatorID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}
