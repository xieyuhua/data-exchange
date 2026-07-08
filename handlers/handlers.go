package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"data-exchange/models"
	"data-exchange/services"

	"github.com/gin-gonic/gin"
)

func success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "success", Data: data})
}

func successWithTotal(c *gin.Context, data interface{}, total int64) {
	c.JSON(http.StatusOK, models.APIResponse{Code: 0, Message: "success", Data: data, Total: total})
}

func fail(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, models.APIResponse{Code: 1, Message: msg})
}

// ==================== 系统常量 ====================

func ListConstants(c *gin.Context) {
	constants, err := services.GetAllConstants()
	if err != nil {
		fail(c, "获取常量列表失败: "+err.Error())
		return
	}
	if constants == nil {
		constants = []models.Constant{}
	}
	success(c, constants)
}

func SaveConstant(c *gin.Context) {
	var con models.Constant
	if err := c.ShouldBindJSON(&con); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if con.Key == "" {
		fail(c, "常量名不能为空")
		return
	}
	if err := services.SaveConstant(&con); err != nil {
		fail(c, "保存常量失败: "+err.Error())
		return
	}
	success(c, nil)
}

func DeleteConstant(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := services.DeleteConstant(id); err != nil {
		fail(c, "删除常量失败: "+err.Error())
		return
	}
	success(c, nil)
}

// ==================== 数据库连接 ====================

func ListDBConnections(c *gin.Context) {
	var list []models.DBConnection
	models.DB.Order("id DESC").Find(&list)
	if list == nil {
		list = []models.DBConnection{}
	}
	success(c, list)
}

func SaveDBConnection(c *gin.Context) {
	var conn models.DBConnection
	if err := c.ShouldBindJSON(&conn); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if conn.Name == "" {
		fail(c, "连接名称不能为空")
		return
	}
	if conn.ID == 0 {
		if err := models.DB.Create(&conn).Error; err != nil {
			fail(c, "添加失败: "+err.Error())
			return
		}
	} else {
		if err := models.DB.Model(&conn).Updates(map[string]interface{}{
			"name": conn.Name, "db_type": conn.DBType, "host": conn.Host,
			"port": conn.Port, "username": conn.Username, "password": conn.Password,
			"database_name": conn.DatabaseName, "extra_params": conn.ExtraParams, "enabled": conn.Enabled,
		}).Error; err != nil {
			fail(c, "更新失败: "+err.Error())
			return
		}
	}
	success(c, nil)
}

func DeleteDBConnection(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := models.DB.Delete(&models.DBConnection{}, id).Error; err != nil {
		fail(c, "删除失败: "+err.Error())
		return
	}
	success(c, nil)
}

func TestDBConnection(c *gin.Context) {
	var conn models.DBConnection
	if err := c.ShouldBindJSON(&conn); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if err := services.TestDBConnection(&conn); err != nil {
		fail(c, "连接测试失败: "+err.Error())
		return
	}
	success(c, "连接成功")
}

// ==================== 厂家管理 ====================

func ListVendors(c *gin.Context) {
	var list []models.Vendor
	models.DB.Order("id DESC").Find(&list)
	if list == nil {
		list = []models.Vendor{}
	}
	success(c, list)
}

func SaveVendor(c *gin.Context) {
	var v models.Vendor
	if err := c.ShouldBindJSON(&v); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if v.Name == "" || v.Code == "" {
		fail(c, "厂家名称和编码不能为空")
		return
	}
	if v.ID == 0 {
		if err := models.DB.Create(&v).Error; err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				fail(c, "厂家编码已存在")
				return
			}
			fail(c, "添加失败: "+err.Error())
			return
		}
	} else {
		if err := models.DB.Model(&v).Updates(map[string]interface{}{
			"name": v.Name, "code": v.Code, "description": v.Description, "enabled": v.Enabled,
		}).Error; err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				fail(c, "厂家编码已存在")
				return
			}
			fail(c, "更新失败: "+err.Error())
			return
		}
	}
	success(c, nil)
}

func DeleteVendor(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	models.DB.Where("vendor_id = ?", id).Delete(&models.SQLTask{})
	models.DB.Where("vendor_id = ?", id).Delete(&models.FTPAccount{})
	models.DB.Delete(&models.Vendor{}, id)
	success(c, nil)
}

func GetVendorTasks(c *gin.Context) {
	vendorID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var count int64
	models.DB.Model(&models.SQLTask{}).Where("vendor_id = ?", vendorID).Count(&count)

	var tasks []models.SQLTask
	models.DB.Where("vendor_id = ?", vendorID).Order("sort_order, id").Find(&tasks)

	// 填充关联名称
	type taskWithNames struct {
		models.SQLTask
		DBConnectionName string `json:"db_connection_name"`
		FTPAccountName   string `json:"ftp_account_name"`
	}
	var result []taskWithNames
	for _, t := range tasks {
		twn := taskWithNames{SQLTask: t}
		if t.DBConnectionID != nil {
			var dbc models.DBConnection
			if err := models.DB.First(&dbc, *t.DBConnectionID).Error; err == nil {
				twn.DBConnectionName = dbc.Name
			}
		}
		if t.FTPAccountID != nil {
			var fa models.FTPAccount
			if err := models.DB.First(&fa, *t.FTPAccountID).Error; err == nil {
				twn.FTPAccountName = fa.Name
			}
		}
		result = append(result, twn)
	}
	if result == nil {
		result = []taskWithNames{}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
		"total":   count,
		"max":     4,
	})
}

// ==================== SQL任务 ====================

func SaveSQLTask(c *gin.Context) {
	var t models.SQLTask
	if err := c.ShouldBindJSON(&t); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}

	if t.ID == 0 {
		var count int64
		models.DB.Model(&models.SQLTask{}).Where("vendor_id = ?", t.VendorID).Count(&count)
		if count >= 4 {
			fail(c, "每个厂家最多设置4个SQL任务")
			return
		}
		if err := models.DB.Create(&t).Error; err != nil {
			fail(c, "添加失败: "+err.Error())
			return
		}
		if t.Enabled == 1 && t.CronExpression != "" {
			services.AddTaskToScheduler(t.ID, t.CronExpression)
		}
	} else {
		if err := models.DB.Model(&models.SQLTask{}).Where("id = ?", t.ID).Updates(map[string]interface{}{
			"vendor_id": t.VendorID, "db_connection_id": t.DBConnectionID,
			"task_name": t.TaskName, "sql_content": t.SQLContent,
			"csv_filename_template": t.CSVFilenameTemplate, "cron_expression": t.CronExpression,
			"execution_mode": t.ExecutionMode, "ftp_account_id": t.FTPAccountID,
			"sort_order": t.SortOrder, "enabled": t.Enabled,
		}).Error; err != nil {
			fail(c, "更新失败: "+err.Error())
			return
		}
		services.RemoveTaskFromScheduler(t.ID)
		if t.Enabled == 1 && t.CronExpression != "" {
			services.AddTaskToScheduler(t.ID, t.CronExpression)
		}
	}
	success(c, nil)
}

func DeleteSQLTask(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	services.RemoveTaskFromScheduler(id)
	models.DB.Delete(&models.SQLTask{}, id)
	success(c, nil)
}

func ToggleSQLTask(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var t models.SQLTask
	if err := models.DB.First(&t, id).Error; err != nil {
		fail(c, "任务不存在")
		return
	}

	newEnabled := 0
	if t.Enabled == 0 {
		newEnabled = 1
	}
	models.DB.Model(&t).Update("enabled", newEnabled)

	if newEnabled == 1 && t.CronExpression != "" {
		services.AddTaskToScheduler(id, t.CronExpression)
	} else {
		services.RemoveTaskFromScheduler(id)
	}
	success(c, map[string]int{"enabled": newEnabled})
}

func ExecuteTaskNow(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	// 通过 worker pool 提交，自动限流
	pool := services.GetGlobalPool()
	pool.Submit(id)
	success(c, "任务已提交执行")
}

// ExecuteTaskByName 按任务名并发执行所有匹配任务
func ExecuteTaskByName(c *gin.Context) {
	taskName := c.Query("task_name")
	if taskName == "" {
		fail(c, "请提供 task_name 参数")
		return
	}
	// 后台异步执行
	go func() {
		services.ExecuteTaskByNameConcurrent(taskName)
	}()
	success(c, fmt.Sprintf("任务 '%s' 已提交并发执行", taskName))
}

// BatchExecuteTasks 批量并发执行多个任务
func BatchExecuteTasks(c *gin.Context) {
	var req struct {
		TaskIDs []int64 `json:"task_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.TaskIDs) == 0 {
		fail(c, "请提供 task_ids 数组")
		return
	}
	// 同步返回结果
	results := services.ExecuteTasksConcurrent(req.TaskIDs)
	success(c, results)
}

// ==================== FTP/SFTP账号 ====================

func ListFTPAccounts(c *gin.Context) {
	vendorID := c.Query("vendor_id")
	var list []models.FTPAccount
	query := models.DB.Order("id")
	if vendorID != "" {
		query = query.Where("vendor_id = ?", vendorID)
	}
	query.Find(&list)

	type ftpWithVendor struct {
		models.FTPAccount
		VendorName string `json:"vendor_name"`
	}
	var result []ftpWithVendor
	for _, a := range list {
		fwv := ftpWithVendor{FTPAccount: a}
		var v models.Vendor
		if err := models.DB.First(&v, a.VendorID).Error; err == nil {
			fwv.VendorName = v.Name
		}
		result = append(result, fwv)
	}
	if result == nil {
		result = []ftpWithVendor{}
	}
	success(c, result)
}

func SaveFTPAccount(c *gin.Context) {
	var a models.FTPAccount
	if err := c.ShouldBindJSON(&a); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if a.ID == 0 {
		if err := models.DB.Create(&a).Error; err != nil {
			fail(c, "添加失败: "+err.Error())
			return
		}
	} else {
		if err := models.DB.Model(&a).Updates(map[string]interface{}{
			"vendor_id": a.VendorID, "name": a.Name, "protocol": a.Protocol,
			"host": a.Host, "port": a.Port, "username": a.Username,
			"password": a.Password, "remote_path": a.RemotePath, "enabled": a.Enabled,
		}).Error; err != nil {
			fail(c, "更新失败: "+err.Error())
			return
		}
	}
	success(c, nil)
}

func DeleteFTPAccount(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	models.DB.Delete(&models.FTPAccount{}, id)
	success(c, nil)
}

// ==================== 系统配置 ====================

func ListSystemConfigs(c *gin.Context) {
	configs, err := services.GetAllConfigs()
	if err != nil {
		fail(c, "获取配置失败: "+err.Error())
		return
	}
	if configs == nil {
		configs = []models.SystemConfig{}
	}
	success(c, configs)
}

func SaveSystemConfig(c *gin.Context) {
	type ConfigItem struct {
		Key   string `json:"config_key"`
		Value string `json:"config_value"`
	}
	var items []ConfigItem
	if err := c.ShouldBindJSON(&items); err != nil {
		var item ConfigItem
		if err := c.ShouldBindJSON(&item); err != nil {
			fail(c, "参数错误: "+err.Error())
			return
		}
		services.SetConfig(item.Key, item.Value)
	} else {
		for _, item := range items {
			services.SetConfig(item.Key, item.Value)
		}
	}
	success(c, nil)
}

// ==================== 执行日志 ====================

func ListExportLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	if page < 1 {
		page = 1
	}

	query := models.DB.Model(&models.ExportLog{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var logs []models.ExportLog
	query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs)

	// 填充关联名称
	type logWithNames struct {
		models.ExportLog
		TaskName   string `json:"task_name"`
		VendorName string `json:"vendor_name"`
	}
	var result []logWithNames
	for _, l := range logs {
		lwn := logWithNames{ExportLog: l}
		var t models.SQLTask
		if err := models.DB.First(&t, l.TaskID).Error; err == nil {
			lwn.TaskName = t.TaskName
		}
		var v models.Vendor
		if err := models.DB.First(&v, l.VendorID).Error; err == nil {
			lwn.VendorName = v.Name
		}
		result = append(result, lwn)
	}
	if result == nil {
		result = []logWithNames{}
	}
	successWithTotal(c, result, total)
}

func DeleteExportLog(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	models.DB.Delete(&models.ExportLog{}, id)
	success(c, nil)
}

func ClearExportLogs(c *gin.Context) {
	models.DB.Where("1 = 1").Delete(&models.ExportLog{})
	success(c, nil)
}

// ==================== 文件管理 ====================

func ListOutputFiles(c *gin.Context) {
	outputDir := services.GetConfigWithDefault("csv_output_dir", "./output")
	files, err := os.ReadDir(outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			success(c, []map[string]interface{}{})
			return
		}
		fail(c, "读取目录失败: "+err.Error())
		return
	}
	type FileInfo struct {
		Name    string `json:"name"`
		Size    int64  `json:"size"`
		ModTime string `json:"mod_time"`
	}
	var list []FileInfo
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		info, _ := f.Info()
		fi := FileInfo{Name: f.Name()}
		if info != nil {
			fi.Size = info.Size()
			fi.ModTime = info.ModTime().Format("2006-01-02 15:04:05")
		}
		list = append(list, fi)
	}
	if list == nil {
		list = []FileInfo{}
	}
	success(c, list)
}

func DownloadFile(c *gin.Context) {
	filename := c.Query("filename")
	if filename == "" {
		fail(c, "文件名不能为空")
		return
	}
	filename = filepath.Base(filename)
	outputDir := services.GetConfigWithDefault("csv_output_dir", "./output")
	filePath := filepath.Join(outputDir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fail(c, "文件不存在")
		return
	}
	c.FileAttachment(filePath, filename)
}

func ListBackupFiles(c *gin.Context) {
	backupDir := services.GetConfigWithDefault("backup_dir", "./backup")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			success(c, []map[string]interface{}{})
			return
		}
		fail(c, "读取目录失败: "+err.Error())
		return
	}
	type FileInfo struct {
		Name    string `json:"name"`
		Size    int64  `json:"size"`
		ModTime string `json:"mod_time"`
	}
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
	if list == nil {
		list = []FileInfo{}
	}
	keepCount := services.GetConfigWithDefault("backup_keep_count", "30")
	success(c, map[string]interface{}{
		"files":      list,
		"keep_count": keepCount,
	})
}

// ==================== 仪表盘 ====================

func DashboardStats(c *gin.Context) {
	var vendorCount, taskCount, ftpCount, logCount, successCount, failCount int64

	models.DB.Model(&models.Vendor{}).Count(&vendorCount)
	models.DB.Model(&models.SQLTask{}).Where("enabled = 1").Count(&taskCount)
	models.DB.Model(&models.FTPAccount{}).Where("enabled = 1").Count(&ftpCount)
	models.DB.Model(&models.ExportLog{}).Count(&logCount)
	models.DB.Model(&models.ExportLog{}).Where("status = 'success'").Count(&successCount)
	models.DB.Model(&models.ExportLog{}).Where("status = 'failed'").Count(&failCount)

	var recentLogs []models.ExportLog
	models.DB.Order("id DESC").Limit(10).Find(&recentLogs)

	type logWithNames struct {
		models.ExportLog
		TaskName   string `json:"task_name"`
		VendorName string `json:"vendor_name"`
	}
	var recent []logWithNames
	for _, l := range recentLogs {
		lwn := logWithNames{ExportLog: l}
		var t models.SQLTask
		if models.DB.First(&t, l.TaskID).Error == nil {
			lwn.TaskName = t.TaskName
		}
		var v models.Vendor
		if models.DB.First(&v, l.VendorID).Error == nil {
			lwn.VendorName = v.Name
		}
		recent = append(recent, lwn)
	}
	if recent == nil {
		recent = []logWithNames{}
	}

	backupKeep := services.GetConfigWithDefault("backup_keep_count", "30")

	success(c, gin.H{
		"vendor_count":  vendorCount,
		"task_count":    taskCount,
		"ftp_count":     ftpCount,
		"log_count":     logCount,
		"success_count": successCount,
		"fail_count":    failCount,
		"recent_logs":   recent,
		"backup_keep":   backupKeep,
		"current_time":  time.Now().Format("2006-01-02 15:04:05"),
	})
}

// ==================== 清理备份 ====================

func CleanBackupsNow(c *gin.Context) {
	if err := services.CleanOldBackups(); err != nil {
		fail(c, "清理失败: "+err.Error())
		return
	}
	success(c, "清理完成")
}

// 测试通知
func TestNotify(c *gin.Context) {
	var req struct {
		Channel string `json:"channel"` // ding / wx
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Content == "" {
		req.Content = "【测试消息】数据交换系统通知通道测试"
	}

	switch req.Channel {
	case "ding":
		wb := services.GetConfig("notify_ding_webhook")
		if wb == "" {
			fail(c, "钉钉 Webhook 未配置")
			return
		}
		// 用 notify 包的内部函数发送（间接通过 NotifyFailure 的格式化）
		services.NotifyFailure("测试任务", "测试厂家", req.Content)
		success(c, "钉钉测试消息已发送")
	case "wx":
		wb := services.GetConfig("notify_wx_webhook")
		if wb == "" {
			fail(c, "企业微信 Webhook 未配置")
			return
		}
		services.NotifyFailure("测试任务", "测试厂家", req.Content)
		success(c, "企业微信测试消息已发送")
	default:
		fail(c, "请指定 channel: ding 或 wx")
	}
}

// 自定义常量函数（如日期计算）
func EvalConstantFunc(c *gin.Context) {
	var req struct {
		Name string `json:"name"` // 如 now / yesterday / tomorrow
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

// ==================== FTP/SFTP 测试 ====================

func TestFTPConnection(c *gin.Context) {
	var acc models.FTPAccount
	if err := c.ShouldBindJSON(&acc); err != nil {
		fail(c, "参数错误: "+err.Error())
		return
	}
	if err := services.TestFTPConnection(&acc); err != nil {
		fail(c, "连接测试失败: "+err.Error())
		return
	}
	success(c, "连接成功")
}

// ==================== SQL 测试 ====================

func TestSQLExecution(c *gin.Context) {
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
	columns, rows, err := services.TestSQLExecution(req.DBConnectionID, req.SQLContent)
	if err != nil {
		fail(c, "SQL测试失败: "+err.Error())
		return
	}
	success(c, gin.H{
		"columns":    columns,
		"rows":       rows,
		"row_count":  len(rows),
	})
}
