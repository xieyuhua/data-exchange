package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"data-exchange/models"
	"data-exchange/services"

	"github.com/gin-gonic/gin"
)

// ==================== 系统常量 ====================

func (h *Handler) ListConstants(c *gin.Context) {
	constants, err := h.App.Constant.List()
	if err != nil {
		fail(c, "获取常量列表失败: "+err.Error())
		return
	}
	if constants == nil {
		constants = []models.Constant{}
	}
	success(c, constants)
}

func (h *Handler) SaveConstant(c *gin.Context) {
	var con models.Constant
	if err := c.ShouldBindJSON(&con); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if con.Key == "" {
		fail(c, "常量名不能为空")
		return
	}
	if err := h.App.Constant.Save(&con); err != nil {
		fail(c, "保存常量失败: "+err.Error())
		return
	}
	success(c, nil)
}

func (h *Handler) DeleteConstant(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.App.Constant.Delete(id); err != nil {
		fail(c, "删除常量失败: "+err.Error())
		return
	}
	success(c, nil)
}

// ==================== 数据库连接 ====================

func (h *Handler) ListDBConnections(c *gin.Context) {
	list, err := h.App.DBConnection.List()
	if err != nil {
		fail(c, "获取连接列表失败: "+err.Error())
		return
	}
	if list == nil {
		list = []models.DBConnection{}
	}
	success(c, list)
}

func (h *Handler) SaveDBConnection(c *gin.Context) {
	var conn models.DBConnection
	if err := c.ShouldBindJSON(&conn); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if err := h.App.DBConnection.Save(&conn); err != nil {
		fail(c, "保存连接失败: "+err.Error())
		return
	}
	success(c, nil)
}

func (h *Handler) DeleteDBConnection(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.App.DBConnection.Delete(id); err != nil {
		fail(c, "删除连接失败: "+err.Error())
		return
	}
	success(c, nil)
}

func (h *Handler) TestDBConnection(c *gin.Context) {
	var conn models.DBConnection
	if err := c.ShouldBindJSON(&conn); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if err := h.App.TestDBConnection(&conn); err != nil {
		fail(c, "连接测试失败: "+err.Error())
		return
	}
	success(c, "连接成功")
}

// ==================== 厂家管理 ====================

// resolvePageSize 解析分页大小：优先使用请求参数 page_size，缺省时取自系统配置 page_size，无效值回退到 20
func (h *Handler) resolvePageSize(c *gin.Context) int {
	def := h.App.GetConfigWithDefault("page_size", "20")
	ps, err := strconv.Atoi(c.DefaultQuery("page_size", def))
	if err != nil || ps < 1 {
		ps, _ = strconv.Atoi(def)
	}
	if ps < 1 {
		ps = 20
	}
	return ps
}

func (h *Handler) ListVendors(c *gin.Context) {
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := h.resolvePageSize(c)
	list, total, err := h.App.Vendor.ListPaged(keyword, page, pageSize)
	if err != nil {
		fail(c, "获取厂家列表失败: "+err.Error())
		return
	}
	if list == nil {
		list = []models.Vendor{}
	}
	successWithTotal(c, list, total)
}

func (h *Handler) SaveVendor(c *gin.Context) {
	var v models.Vendor
	if err := c.ShouldBindJSON(&v); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if err := h.App.Vendor.Save(&v); err != nil {
		fail(c, "保存厂家失败: "+err.Error())
		return
	}
	success(c, nil)
}

func (h *Handler) DeleteVendor(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.App.Vendor.Delete(id); err != nil {
		fail(c, "删除厂家失败: "+err.Error())
		return
	}
	success(c, nil)
}

func (h *Handler) GetVendorTasks(c *gin.Context) {
	vendorID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	result, err := h.App.Task.ListByVendor(vendorID)
	if err != nil {
		fail(c, "获取任务失败: "+err.Error())
		return
	}
	if result.Tasks == nil {
		result.Tasks = []services.TaskWithNames{}
	}
	c.JSON(http.StatusOK, gin.H{
		"code":     0,
		"message":  "success",
		"data":     result.Tasks,
		"total":    len(result.Tasks),
		"max":      result.Max,
	})
}

// ==================== SQL 任务 ====================

func (h *Handler) SaveSQLTask(c *gin.Context) {
	var t models.SQLTask
	if err := c.ShouldBindJSON(&t); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if err := h.App.Task.Save(&t); err != nil {
		fail(c, "保存任务失败: "+err.Error())
		return
	}
	success(c, nil)
}

func (h *Handler) DeleteSQLTask(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.App.Task.Delete(id); err != nil {
		fail(c, "删除任务失败: "+err.Error())
		return
	}
	success(c, nil)
}

func (h *Handler) ToggleSQLTask(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	enabled, err := h.App.Task.Toggle(id)
	if err != nil {
		fail(c, "切换状态失败: "+err.Error())
		return
	}
	success(c, map[string]int{"enabled": enabled})
}

func (h *Handler) GetSQLTask(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	twn, err := h.App.Task.Get(id)
	if err != nil {
		fail(c, "任务不存在")
		return
	}
	success(c, twn)
}

func (h *Handler) ExecuteTaskNow(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if services.IsTaskRunning(id) {
		fail(c, "任务正在执行中，请稍候")
		return
	}
	h.App.Pool.Submit(id)
	success(c, "任务已提交执行")
}

func (h *Handler) ListRunningTasks(c *gin.Context) {
	success(c, services.RunningTaskIDs())
}

// CronNextRuns 返回 cron 表达式未来 n 次执行时间（用于前端友好预览）
func (h *Handler) CronNextRuns(c *gin.Context) {
	expr := c.Query("expr")
	n, _ := strconv.Atoi(c.DefaultQuery("n", "5"))
	if n <= 0 || n > 20 {
		n = 5
	}
	times, err := services.NextRunTimes(expr, n)
	if err != nil {
		fail(c, "cron 表达式无效: "+err.Error())
		return
	}
	success(c, times)
}

func (h *Handler) ExecuteTaskByName(c *gin.Context) {
	taskName := c.Query("task_name")
	if taskName == "" {
		fail(c, "请提供 task_name 参数")
		return
	}
	go func() {
		h.App.Executor.ExecuteByNameConcurrent(taskName)
	}()
	success(c, fmt.Sprintf("任务 '%s' 已提交并发执行", taskName))
}

func (h *Handler) BatchExecuteTasks(c *gin.Context) {
	var req struct {
		TaskIDs []int64 `json:"task_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.TaskIDs) == 0 {
		fail(c, "请提供 task_ids 数组")
		return
	}
	results := h.App.Pool.SubmitBatch(req.TaskIDs)
	success(c, results)
}

// ==================== FTP / SFTP 账号 ====================

func (h *Handler) ListFTPAccounts(c *gin.Context) {
	vendorID := c.Query("vendor_id")
	result, err := h.App.FTP.List(vendorID)
	if err != nil {
		fail(c, "获取FTP账号失败: "+err.Error())
		return
	}
	if result == nil {
		result = []services.FTPWithVendor{}
	}
	success(c, result)
}

func (h *Handler) SaveFTPAccount(c *gin.Context) {
	var a models.FTPAccount
	if err := c.ShouldBindJSON(&a); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if err := h.App.FTP.Save(&a); err != nil {
		fail(c, "保存FTP账号失败: "+err.Error())
		return
	}
	success(c, nil)
}

func (h *Handler) DeleteFTPAccount(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.App.FTP.Delete(id); err != nil {
		fail(c, "删除FTP账号失败: "+err.Error())
		return
	}
	success(c, nil)
}

func (h *Handler) TestFTPConnection(c *gin.Context) {
	var acc models.FTPAccount
	if err := c.ShouldBindJSON(&acc); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if err := h.App.TestFTPConnection(&acc); err != nil {
		fail(c, "连接测试失败: "+err.Error())
		return
	}
	success(c, "连接成功")
}

// ListFTPRemoteFiles 列出指定 FTP/SFTP 账号远程目录文件（支持关键字过滤与分页）
func (h *Handler) ListFTPRemoteFiles(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	acc, err := h.App.FTP.Get(id)
	if err != nil {
		fail(c, "账号不存在: "+err.Error())
		return
	}
	keyword := strings.TrimSpace(c.Query("keyword"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 20
	}
	result, err := h.App.ListRemoteFiles(acc, keyword, page, pageSize)
	if err != nil {
		fail(c, "列出远程文件失败: "+err.Error())
		return
	}
	if result == nil {
		result = &services.ListRemoteFilesResult{List: []services.RemoteFileInfo{}}
	}
	success(c, result)
}

// DeleteFTPRemoteFile 删除远程文件/目录
func (h *Handler) DeleteFTPRemoteFile(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	acc, err := h.App.FTP.Get(id)
	if err != nil {
		fail(c, "账号不存在: "+err.Error())
		return
	}
	name := c.Query("path")
	if name == "" {
		fail(c, "文件名不能为空")
		return
	}
	if err := h.App.DeleteRemoteFile(acc, name); err != nil {
		fail(c, "删除失败: "+err.Error())
		return
	}
	success(c, nil)
}

// UploadFTPRemoteFile 上传本地文件到远程目录
func (h *Handler) UploadFTPRemoteFile(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	acc, err := h.App.FTP.Get(id)
	if err != nil {
		fail(c, "账号不存在: "+err.Error())
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		fail(c, "未找到上传文件")
		return
	}
	tmp, err := os.CreateTemp("", "ftp-upload-*")
	if err != nil {
		fail(c, "创建临时文件失败: "+err.Error())
		return
	}
	tmpPath := tmp.Name()
	tmp.Close()
	if err := c.SaveUploadedFile(file, tmpPath); err != nil {
		os.Remove(tmpPath)
		fail(c, "保存上传文件失败: "+err.Error())
		return
	}
	defer os.Remove(tmpPath)

	remoteName := c.PostForm("path")
	if remoteName == "" {
		remoteName = file.Filename
	}
	if err := h.App.UploadRemoteFile(acc, tmpPath, remoteName); err != nil {
		fail(c, "上传失败: "+err.Error())
		return
	}
	success(c, nil)
}

// ==================== 系统配置 ====================

func (h *Handler) ListSystemConfigs(c *gin.Context) {
	configs, err := h.App.Config.List()
	if err != nil {
		fail(c, "获取配置失败: "+err.Error())
		return
	}
	if configs == nil {
		configs = []models.SystemConfig{}
	}
	success(c, configs)
}

func (h *Handler) SaveSystemConfig(c *gin.Context) {
	var item services.ConfigItem
	if err := c.ShouldBindJSON(&item); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if err := h.App.Config.Save([]services.ConfigItem{item}); err != nil {
		fail(c, "保存配置失败: "+err.Error())
		return
	}
	success(c, nil)
}

// ==================== 执行日志 ====================

func (h *Handler) ListExportLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := h.resolvePageSize(c)
	status := c.Query("status")
	keyword := c.Query("keyword")

	logs, total, err := h.App.Log.List(page, pageSize, status, keyword)
	if err != nil {
		fail(c, "获取日志失败: "+err.Error())
		return
	}
	if logs == nil {
		logs = []services.LogWithNames{}
	}
	successWithTotal(c, logs, total)
}

func (h *Handler) DeleteExportLog(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.App.Log.Delete(id); err != nil {
		fail(c, "删除日志失败: "+err.Error())
		return
	}
	success(c, nil)
}

func (h *Handler) ClearExportLogs(c *gin.Context) {
	if err := h.App.Log.Clear(); err != nil {
		fail(c, "清空日志失败: "+err.Error())
		return
	}
	success(c, nil)
}

// ==================== 文件管理 ====================

// FileInfo 文件列表条目
type FileInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

// collectFileInfos 从目录项收集非目录文件信息
func collectFileInfos(entries []os.DirEntry) []FileInfo {
	var list []FileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, _ := e.Info()
		fi := FileInfo{Name: e.Name()}
		if info != nil {
			fi.Size = info.Size()
			fi.ModTime = info.ModTime().Format("2006-01-02 15:04:05")
		}
		list = append(list, fi)
	}
	return list
}

// paginateFiles 对内存文件列表做分页，返回当前页与总记录数
func paginateFiles(list []FileInfo, page, pageSize int) ([]FileInfo, int64) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	total := int64(len(list))
	start := (page - 1) * pageSize
	if start >= len(list) {
		return []FileInfo{}, total
	}
	end := start + pageSize
	if end > len(list) {
		end = len(list)
	}
	return list[start:end], total
}

func (h *Handler) ListOutputFiles(c *gin.Context) {
	outputDir := h.App.GetConfigWithDefault("csv_output_dir", "./output")
	files, err := os.ReadDir(outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			successWithTotal(c, []FileInfo{}, 0)
			return
		}
		fail(c, "读取目录失败: "+err.Error())
		return
	}
	list := collectFileInfos(files)
	if list == nil {
		list = []FileInfo{}
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := h.resolvePageSize(c)
	pageList, total := paginateFiles(list, page, pageSize)
	successWithTotal(c, pageList, total)
}

func (h *Handler) DownloadFile(c *gin.Context) {
	dir := c.DefaultQuery("dir", "output")
	var baseDir string
	switch dir {
	case "backup":
		baseDir = h.App.GetConfigWithDefault("backup_dir", "./backup")
	default:
		baseDir = h.App.GetConfigWithDefault("csv_output_dir", "./output")
	}

	filename := c.Query("filename")
	if filename == "" {
		fail(c, "文件名不能为空")
		return
	}
	filename = filepath.Base(filename)
	if filename == "." || filename == string(os.PathSeparator) {
		fail(c, "非法文件名")
		return
	}

	filePath := filepath.Join(baseDir, filename)

	absBase, errBase := filepath.Abs(baseDir)
	absFile, errFile := filepath.Abs(filePath)
	if errBase != nil || errFile != nil || !strings.HasPrefix(absFile, absBase+string(os.PathSeparator)) && absFile != absBase {
		fail(c, "非法路径")
		return
	}

	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			fail(c, "文件不存在")
			return
		}
		if os.IsPermission(err) {
			fail(c, "文件无读取权限，请检查文件/目录权限: "+err.Error())
			return
		}
		fail(c, "打开文件失败: "+err.Error())
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		fail(c, "读取文件信息失败: "+err.Error())
		return
	}
	if info.IsDir() {
		fail(c, "不能下载目录")
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.QueryEscape(filename))
	c.Header("Content-Length", strconv.FormatInt(info.Size(), 10))
	http.ServeContent(c.Writer, c.Request, filename, info.ModTime(), f)
}

func (h *Handler) ListBackupFiles(c *gin.Context) {
	backupDir := h.App.GetConfigWithDefault("backup_dir", "./backup")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			success(c, map[string]interface{}{"files": []FileInfo{}, "total": 0, "keep_count": h.App.GetConfigWithDefault("backup_keep_count", "30")})
			return
		}
		fail(c, "读取目录失败: "+err.Error())
		return
	}
	list := collectFileInfos(entries)
	if list == nil {
		list = []FileInfo{}
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := h.resolvePageSize(c)
	pageList, total := paginateFiles(list, page, pageSize)
	keepCount := h.App.GetConfigWithDefault("backup_keep_count", "30")
	success(c, map[string]interface{}{
		"files":      pageList,
		"total":      total,
		"keep_count": keepCount,
	})
}

// ==================== 仪表盘 ====================

func (h *Handler) DashboardStats(c *gin.Context) {
	stats, err := h.App.Dashboard.Stats()
	if err != nil {
		fail(c, "获取仪表盘数据失败: "+err.Error())
		return
	}
	success(c, stats)
}

// ==================== 清理备份 / 通知测试 / 常量函数 / SQL 测试 ====================

func (h *Handler) CleanBackupsNow(c *gin.Context) {
	if err := h.App.CleanOldBackups(); err != nil {
		fail(c, "清理失败: "+err.Error())
		return
	}
	success(c, "清理完成")
}

func (h *Handler) TestNotify(c *gin.Context) {
	var req struct {
		Channel string `json:"channel"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Content == "" {
		req.Content = "【测试消息】数据交换系统通知通道测试"
	}
	switch req.Channel {
	case "ding":
		wb := h.App.GetConfig("notify_ding_webhook")
		if wb == "" {
			fail(c, "钉钉 Webhook 未配置")
			return
		}
		h.App.NotifyFailure("测试任务", "测试厂家", req.Content)
		success(c, "钉钉测试消息已发送")
	case "wx":
		wb := h.App.GetConfig("notify_wx_webhook")
		if wb == "" {
			fail(c, "企业微信 Webhook 未配置")
			return
		}
		h.App.NotifyFailure("测试任务", "测试厂家", req.Content)
		success(c, "企业微信测试消息已发送")
	default:
		fail(c, "请指定 channel: ding 或 wx")
	}
}

func (h *Handler) EvalConstantFunc(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	now := time.Now()
	switch req.Name {
	case "now":
		success(c, now.Format("2006-01-02 15:04:05"))
	case "yesterday":
		success(c, now.AddDate(0, 0, -1).Format("2006-01-02"))
	case "tomorrow":
		success(c, now.AddDate(0, 0, 1).Format("2006-01-02"))
	case "month_start":
		success(c, time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"))
	case "month_end":
		success(c, time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1).Format("2006-01-02"))
	default:
		fail(c, fmt.Sprintf("不支持的函数: %s，支持 now/yesterday/tomorrow/month_start/month_end", req.Name))
	}
}

func (h *Handler) TestSQLExecution(c *gin.Context) {
	var req struct {
		DBConnectionID int64  `json:"db_connection_id"`
		SQLContent     string `json:"sql_content"`
		Limit          int    `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if req.DBConnectionID == 0 {
		fail(c, "请先选择数据库连接")
		return
	}
	if req.SQLContent == "" {
		fail(c, "SQL内容不能为空")
		return
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	columns, rows, err := h.App.TestSQLExecution(req.DBConnectionID, req.SQLContent, req.Limit)
	if err != nil {
		fail(c, "SQL测试失败: "+err.Error())
		return
	}
	success(c, gin.H{
		"columns":    columns,
		"rows":       rows,
		"row_count":  len(rows),
		"full_count": len(rows),
		"limited":    req.Limit > 0 && len(rows) >= req.Limit,
	})
}

func (h *Handler) ExportTestSQLResult(c *gin.Context) {
	var req struct {
		DBConnectionID int64  `json:"db_connection_id"`
		SQLContent     string `json:"sql_content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if req.DBConnectionID == 0 {
		fail(c, "请先选择数据库连接")
		return
	}
	if req.SQLContent == "" {
		fail(c, "SQL内容不能为空")
		return
	}
	columns, rows, err := h.App.TestSQLExecution(req.DBConnectionID, req.SQLContent, -1)
	if err != nil {
		fail(c, "SQL导出失败: "+err.Error())
		return
	}
	success(c, gin.H{
		"columns":   columns,
		"rows":      rows,
		"row_count": len(rows),
	})
}
