package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"data-exchange/models"
	"data-exchange/services"

	"github.com/gin-gonic/gin"
)

// ==================== 通用辅助 ====================

// currentUsername 从 gin 上下文取当前登录用户名
func currentUsername(c *gin.Context) string {
	if v, ok := c.Get("username"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// currentUserID 从 gin 上下文取当前登录用户 ID
func currentUserID(c *gin.Context) int64 {
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 0
}

// ==================== 权限中间件 ====================

// RequireAdmin 仅允许管理员访问的中间件（依赖 AuthMiddleware 已注入 role）
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if r, ok := role.(string); !ok || r != "admin" {
			c.JSON(http.StatusOK, models.APIResponse{Code: 1, Message: "无权限：仅管理员可执行此操作"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ==================== 操作日志中间件 ====================

// bodyLogWriter 包装 ResponseWriter 以捕获响应体（用于判定业务成功/失败）
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

var passwordMaskRe = regexp.MustCompile(`("(?:password|old_password|new_password)"\s*:\s*)"[^"]*"`)

// maskSensitive 对请求体中的密码字段做脱敏
func maskSensitive(s string) string {
	return passwordMaskRe.ReplaceAllString(s, `$1"***"`)
}

// truncate 截断字符串到最大长度
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// OperationLogMiddleware 记录用户的写操作（POST/PUT/PATCH/DELETE），便于审计与排查
func (h *Handler) OperationLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == http.MethodGet || method == http.MethodOptions || method == http.MethodHead {
			c.Next()
			return
		}
		start := time.Now()

		// 采集请求参数摘要（跳过文件上传体）
		var detail string
		if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/") {
			detail = "[文件上传]"
		} else if c.Request.Body != nil {
			b, _ := io.ReadAll(io.LimitReader(c.Request.Body, 8192))
			c.Request.Body = io.NopCloser(bytes.NewBuffer(b))
			detail = maskSensitive(strings.TrimSpace(string(b)))
		}
		if q := c.Request.URL.RawQuery; q != "" {
			if detail != "" {
				detail += " "
			}
			detail += "?" + q
		}

		blw := &bodyLogWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = blw

		c.Next()

		// 解析业务响应码判定成功/失败
		bizCode := 0
		var resp struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(blw.body.Bytes(), &resp); err == nil {
			bizCode = resp.Code
		}
		successFlag := 1
		if bizCode != 0 || c.Writer.Status() >= 400 {
			successFlag = 0
		}

		module, action := describeOperation(method, c.FullPath())
		h.App.OpLog.Record(&models.OperationLog{
			UserID:     currentUserID(c),
			Username:   currentUsername(c),
			Action:     action,
			Module:     module,
			Method:     method,
			Path:       c.Request.URL.Path,
			Detail:     truncate(detail, 1000),
			IP:         c.ClientIP(),
			Status:     bizCode,
			Success:    successFlag,
			DurationMs: time.Since(start).Milliseconds(),
			CreatedAt:  time.Now().Format("2006-01-02 15:04:05"),
		})
	}
}

// describeOperation 依据 HTTP 方法与路由模板生成可读的模块与操作描述
func describeOperation(method, fullPath string) (module, action string) {
	// 去掉 /api 前缀便于匹配
	p := strings.TrimPrefix(fullPath, "/api")

	moduleMap := []struct {
		prefix string
		name   string
	}{
		{"/vendors", "厂家管理"},
		{"/tasks", "SQL任务"},
		{"/db-connections", "数据库连接"},
		{"/ftp-accounts", "FTP账号"},
		{"/constants", "系统常量"},
		{"/configs", "系统配置"},
		{"/logs", "执行日志"},
		{"/operation-logs", "操作日志"},
		{"/users", "用户管理"},
		{"/files", "文件管理"},
		{"/notify", "通知"},
		{"/auth", "账号"},
	}
	module = "系统"
	for _, m := range moduleMap {
		if strings.HasPrefix(p, m.prefix) {
			module = m.name
			break
		}
	}

	// 特定动作优先匹配
	switch {
	case strings.Contains(p, "/execute"), strings.Contains(p, "/batch-execute"), strings.Contains(p, "/execute-by-name"):
		return module, "执行任务"
	case strings.Contains(p, "/toggle"):
		return module, "切换启用状态"
	case strings.Contains(p, "/restore"):
		return module, "恢复历史版本"
	case strings.Contains(p, "/reset-password"):
		return module, "重置密码"
	case strings.Contains(p, "/change-password"):
		return module, "修改密码"
	case strings.Contains(p, "/login"):
		return module, "登录"
	case strings.Contains(p, "/test"):
		return module, "连接/执行测试"
	case strings.Contains(p, "/clean-backups"):
		return module, "清理备份"
	}

	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return module, "新增/保存"
	case http.MethodDelete:
		return module, "删除"
	}
	return module, method
}

// ==================== 用户管理（账号权限） ====================

// ListUsers 分页查询用户
func (h *Handler) ListUsers(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := h.resolvePageSize(c)
	list, total, err := h.App.User.ListPaged(keyword, page, pageSize)
	if err != nil {
		fail(c, "获取用户列表失败: "+err.Error())
		return
	}
	if list == nil {
		list = []models.User{}
	}
	successWithTotal(c, list, total)
}

// CreateUser 新增用户
func (h *Handler) CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if err := h.App.User.Create(services.CreateParams{
		Username: req.Username, Password: req.Password, Nickname: req.Nickname, Role: req.Role,
	}); err != nil {
		fail(c, err.Error())
		return
	}
	success(c, nil)
}

// UpdateUser 更新用户信息（昵称/角色/状态）
func (h *Handler) UpdateUser(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Nickname string `json:"nickname"`
		Role     string `json:"role"`
		Status   int    `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if err := h.App.User.Update(services.UpdateParams{
		ID: id, Nickname: req.Nickname, Role: req.Role, Status: req.Status,
	}); err != nil {
		fail(c, err.Error())
		return
	}
	success(c, nil)
}

// ResetUserPassword 管理员重置指定用户密码
func (h *Handler) ResetUserPassword(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if err := h.App.User.ResetPassword(id, req.Password); err != nil {
		fail(c, err.Error())
		return
	}
	success(c, nil)
}

// DeleteUser 删除用户
func (h *Handler) DeleteUser(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == currentUserID(c) {
		fail(c, "不能删除当前登录的账号")
		return
	}
	if err := h.App.User.Delete(id); err != nil {
		fail(c, err.Error())
		return
	}
	success(c, nil)
}

// ==================== 操作日志查询 ====================

// ListOperationLogs 分页查询操作日志
func (h *Handler) ListOperationLogs(c *gin.Context) {
	username := strings.TrimSpace(c.Query("username"))
	keyword := strings.TrimSpace(c.Query("keyword"))
	successFilter := c.Query("success")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := h.resolvePageSize(c)
	list, total, err := h.App.OpLog.List(username, keyword, successFilter, page, pageSize)
	if err != nil {
		fail(c, "获取操作日志失败: "+err.Error())
		return
	}
	if list == nil {
		list = []models.OperationLog{}
	}
	successWithTotal(c, list, total)
}

// DeleteOperationLog 删除单条操作日志
func (h *Handler) DeleteOperationLog(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.App.OpLog.Delete(id); err != nil {
		fail(c, "删除失败: "+err.Error())
		return
	}
	success(c, nil)
}

// ClearOperationLogs 清空操作日志
func (h *Handler) ClearOperationLogs(c *gin.Context) {
	if err := h.App.OpLog.Clear(); err != nil {
		fail(c, "清空失败: "+err.Error())
		return
	}
	success(c, nil)
}

// ==================== SQL 任务历史 ====================

// ListSQLTaskHistory 查询某任务的 SQL 内容历史版本
func (h *Handler) ListSQLTaskHistory(c *gin.Context) {
	taskID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	list, total, err := h.App.TaskHistory.ListByTask(taskID, page, pageSize)
	if err != nil {
		fail(c, "获取历史失败: "+err.Error())
		return
	}
	if list == nil {
		list = []models.SQLTaskHistory{}
	}
	successWithTotal(c, list, total)
}

// RestoreSQLTaskHistory 将任务恢复到指定历史版本
func (h *Handler) RestoreSQLTaskHistory(c *gin.Context) {
	historyID, _ := strconv.ParseInt(c.Param("hid"), 10, 64)
	task, err := h.App.TaskHistory.Restore(historyID, currentUsername(c))
	if err != nil {
		fail(c, "恢复失败: "+err.Error())
		return
	}
	success(c, task)
}
