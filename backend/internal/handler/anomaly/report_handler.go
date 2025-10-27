package anomaly

import (
	"net/http"
	"strconv"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/anomaly"

	"github.com/gin-gonic/gin"
)

// ReportHandler 报告配置处理器
type ReportHandler struct {
	reportSvc *anomaly.ReportService
}

// NewReportHandler 创建报告处理器实例
func NewReportHandler(reportSvc *anomaly.ReportService) *ReportHandler {
	return &ReportHandler{
		reportSvc: reportSvc,
	}
}

// GetReportConfigs 获取所有报告配置
func (h *ReportHandler) GetReportConfigs(c *gin.Context) {
	configs, err := h.reportSvc.GetReportConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "获取报告配置失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "获取报告配置成功",
		Data:    configs,
	})
}

// GetReportConfig 获取单个报告配置
func (h *ReportHandler) GetReportConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "无效的配置ID",
		})
		return
	}

	config, err := h.reportSvc.GetReportConfig(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "报告配置不存在",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "获取报告配置成功",
		Data:    config,
	})
}

// CreateReportConfig 创建报告配置
func (h *ReportHandler) CreateReportConfig(c *gin.Context) {
	var config model.AnomalyReportConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "无效的请求数据",
		})
		return
	}

	if err := h.reportSvc.CreateReportConfig(&config); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "创建报告配置失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "创建报告配置成功",
		Data:    config,
	})
}

// UpdateReportConfig 更新报告配置
func (h *ReportHandler) UpdateReportConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "无效的配置ID",
		})
		return
	}

	var updates model.AnomalyReportConfig
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "无效的请求数据",
		})
		return
	}

	if err := h.reportSvc.UpdateReportConfig(uint(id), &updates); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "更新报告配置失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "更新报告配置成功",
	})
}

// DeleteReportConfig 删除报告配置
func (h *ReportHandler) DeleteReportConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "无效的配置ID",
		})
		return
	}

	if err := h.reportSvc.DeleteReportConfig(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "删除报告配置失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "删除报告配置成功",
	})
}

// TestReportSend 测试报告发送
func (h *ReportHandler) TestReportSend(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "无效的配置ID",
		})
		return
	}

	if err := h.reportSvc.TestReportSend(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "测试报告发送失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "测试报告发送成功",
	})
}

// RunReportNow 手动执行报告生成
func (h *ReportHandler) RunReportNow(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "无效的配置ID",
		})
		return
	}

	if err := h.reportSvc.ExecuteReport(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "执行报告生成失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "报告生成成功",
	})
}
