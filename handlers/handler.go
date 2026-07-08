package handlers

import (
	"net/http"

	"data-exchange/models"
	"data-exchange/services"

	"github.com/gin-gonic/gin"
)

// Handler 聚合所有 HTTP 处理器，持有应用服务层（依赖注入）。
type Handler struct {
	App *services.App
}

// NewHandler 创建 Handler 实例（供路由注册使用）
func NewHandler(app *services.App) *Handler {
	return &Handler{App: app}
}

// ==================== 通用响应辅助 ====================

func success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "success", Data: data})
}

func successWithTotal(c *gin.Context, data interface{}, total int64) {
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "success", Data: data, Total: total})
}

func fail(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, models.APIResponse{Code: 1, Message: msg})
}
