package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"data-exchange/models"
	"data-exchange/repository"

	"github.com/jlaffaye/ftp"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/pkg/sftp"
	_ "github.com/sijms/go-ora/v2"
	"golang.org/x/crypto/ssh"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// dbConnRepo 数据库连接仓储单例（供任务执行器按 ID 取连接配置，已是结构体实例）
var dbConnRepo = repository.NewDBConnectionRepo()

// ==================== 数据库连接（App 工具方法） ====================

func (a *App) connectDB(conn *models.DBConnection) (*sql.DB, error) {
	var dsn, driver string

	switch conn.DBType {
	case "mysql":
		driver = "mysql"
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local",
			conn.Username, conn.Password, conn.Host, conn.Port, conn.DatabaseName)
		if conn.ExtraParams != "" {
			dsn += "&" + conn.ExtraParams
		}
	case "postgresql":
		driver = "postgres"
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			conn.Host, conn.Port, conn.Username, conn.Password, conn.DatabaseName)
		if conn.ExtraParams != "" {
			dsn += " " + conn.ExtraParams
		}
	case "oracle":
		driver = "oracle"
		dsn = fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
			conn.Username, conn.Password, conn.Host, conn.Port, conn.DatabaseName)
		if conn.ExtraParams != "" {
			dsn += "?" + conn.ExtraParams
		}
	case "mssql":
		driver = "mssql"
		dsn = fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s",
			conn.Username, conn.Password, conn.Host, conn.Port, conn.DatabaseName)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", conn.DBType)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %v", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库连接测试失败: %v", err)
	}
	return db, nil
}

// ==================== 任务执行引擎 ====================

// TaskExecutor 承载单次任务执行的依赖（聚合根 + 仓储实例），替代原先散落的包级函数
type TaskExecutor struct {
	app        *App
	taskRepo   *repository.SQLTaskRepo
	vendorRepo *repository.VendorRepo
	logRepo    *repository.ExportLogRepo
	ftpRepo    *repository.FTPAccountRepo
}

// NewTaskExecutor 构建执行器
func NewTaskExecutor(
	app *App,
	taskRepo *repository.SQLTaskRepo,
	vendorRepo *repository.VendorRepo,
	logRepo *repository.ExportLogRepo,
	ftpRepo *repository.FTPAccountRepo,
) *TaskExecutor {
	return &TaskExecutor{app: app, taskRepo: taskRepo, vendorRepo: vendorRepo, logRepo: logRepo, ftpRepo: ftpRepo}
}

// GetDBConnection 按 ID 获取数据库连接
func (e *TaskExecutor) GetDBConnection(id int64) (*models.DBConnection, error) {
	return dbConnRepo.Get(id)
}

// GetFTPAccount 按 ID 获取 FTP 账号
func (e *TaskExecutor) GetFTPAccount(id int64) (*models.FTPAccount, error) {
	return e.ftpRepo.Get(id)
}

// GetTaskByID 按 ID 获取任务
func (e *TaskExecutor) GetTaskByID(id int64) (*models.SQLTask, error) {
	return e.taskRepo.Get(id)
}

// Execute 执行单个任务，返回执行日志与错误
func (e *TaskExecutor) Execute(taskID int64) (*models.ExportLog, error) {
	startTime := time.Now()
	logEntry := &models.ExportLog{
		TaskID:    taskID,
		Status:    "failed",
		StartedAt: startTime.Format("2006-01-02 15:04:05"),
	}

	notified := false
	notifyFail := func(taskName, vendorName, errMsg string) {
		if notified {
			return
		}
		notified = true
		e.app.NotifyFailure(taskName, vendorName, errMsg)
	}

	task, err := e.taskRepo.Get(taskID)
	if err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("获取任务失败: %v", err)
		e.logRepo.Create(logEntry)
		notifyFail(fmt.Sprintf("#%d", taskID), "", logEntry.ErrorMessage)
		return logEntry, err
	}
	logEntry.VendorID = task.VendorID
	logEntry.ExecutionMode = task.ExecutionMode

	vendor, err := e.vendorRepo.Get(task.VendorID)
	if err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("获取厂家失败: %v", err)
		e.logRepo.Create(logEntry)
		notifyFail(task.TaskName, "", logEntry.ErrorMessage)
		return logEntry, err
	}

	csvPath, recordCount, err := e.executeSQLAndGenerateCSV(task, vendor)
	if err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("生成CSV失败: %v", err)
		e.logRepo.Create(logEntry)
		notifyFail(task.TaskName, vendor.Name, logEntry.ErrorMessage)
		return logEntry, err
	}
	logEntry.CSVFilename = filepath.Base(csvPath)
	logEntry.RecordCount = recordCount

	if fileInfo, err := os.Stat(csvPath); err == nil {
		logEntry.FileSize = fileInfo.Size()
	}

	if _, err := e.app.BackupFile(csvPath); err != nil {
		log.Printf("[任务执行] 备份文件警告: %v", err)
	}

	if task.ExecutionMode == "upload" && task.FTPAccountID != nil {
		ftpAccount, err := e.ftpRepo.Get(*task.FTPAccountID)
		if err != nil {
			logEntry.ErrorMessage = fmt.Sprintf("获取FTP账号失败: %v", err)
			e.logRepo.Create(logEntry)
			notifyFail(task.TaskName, vendor.Name, logEntry.ErrorMessage)
			return logEntry, err
		}
		if err := e.app.UploadFile(csvPath, ftpAccount); err != nil {
			logEntry.ErrorMessage = fmt.Sprintf("文件上传失败: %v", err)
			e.logRepo.Create(logEntry)
			notifyFail(task.TaskName, vendor.Name, logEntry.ErrorMessage)
			return logEntry, err
		}
		log.Printf("[任务执行] 文件上传成功: %s", csvPath)
	}

	go e.app.CleanOldBackups()

	logEntry.Status = "success"
	logEntry.FinishedAt = time.Now().Format("2006-01-02 15:04:05")
	logEntry.DurationMs = time.Since(startTime).Milliseconds()
	logEntry.ErrorMessage = ""
	e.logRepo.Create(logEntry)

	if err := e.taskRepo.UpdateLastRun(taskID, logEntry.FinishedAt, "success"); err != nil {
		log.Printf("[任务执行] 更新任务状态失败: %v", err)
	}

	return logEntry, nil
}

// ExecuteByName 按任务名执行所有启用任务
func (e *TaskExecutor) ExecuteByName(taskName string) ([]*models.ExportLog, error) {
	tasks, err := e.taskRepo.ListEnabledByName(taskName)
	if err != nil {
		return nil, err
	}
	var logs []*models.ExportLog
	for _, t := range tasks {
		l, err := e.Execute(t.ID)
		if err != nil {
			logs = append(logs, l)
			continue
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// ExecuteByNameConcurrent 按任务名并发执行所有匹配任务
func (e *TaskExecutor) ExecuteByNameConcurrent(taskName string) ([]*TaskResult, error) {
	tasks, err := e.taskRepo.ListEnabledByName(taskName)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("未找到匹配的任务: %s", taskName)
	}

	taskIDs := make([]int64, len(tasks))
	for i, t := range tasks {
		taskIDs[i] = t.ID
	}

	log.Printf("[并发执行] 任务名 '%s' 匹配 %d 个任务，并发执行中...", taskName, len(taskIDs))
	results := e.app.Pool.SubmitBatch(taskIDs)

	successCount := 0
	failCount := 0
	for _, r := range results {
		if r.Status == "success" {
			successCount++
		} else {
			failCount++
		}
	}
	log.Printf("[并发执行] 任务名 '%s' 执行完成: 成功 %d, 失败 %d", taskName, successCount, failCount)
	return results, nil
}

func (e *TaskExecutor) executeSQLAndGenerateCSV(task *models.SQLTask, vendor *models.Vendor) (string, int, error) {
	dbConn, err := dbConnRepo.Get(*task.DBConnectionID)
	if err != nil {
		return "", 0, fmt.Errorf("获取数据库连接失败: %v", err)
	}

	db, err := e.app.connectDB(dbConn)
	if err != nil {
		return "", 0, err
	}
	defer db.Close()

	sqlContent := e.app.ReplaceConstants(task.SQLContent)
	rows, err := db.Query(sqlContent)
	if err != nil {
		return "", 0, fmt.Errorf("SQL执行失败: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return "", 0, fmt.Errorf("获取列名失败: %v", err)
	}

	outputDir := e.app.GetConfigWithDefault("csv_output_dir", "./output")
	os.MkdirAll(outputDir, 0755)
	fileName := e.app.GenerateFileName(task.CSVFilenameTemplate, vendor.Code, task.TaskName)
	if !strings.HasSuffix(fileName, ".csv") {
		fileName += ".csv"
	}
	filePath := filepath.Join(outputDir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("创建CSV文件失败: %v", err)
	}
	defer file.Close()

	if e.app.GetConfigWithDefault("csv_bom", "true") == "true" {
		file.Write([]byte{0xEF, 0xBB, 0xBF})
	}

	writer := csv.NewWriter(file)
	if delim := e.app.GetConfigWithDefault("csv_delimiter", ","); len(delim) > 0 {
		writer.Comma = rune(delim[0])
	}

	headerRow := make([]string, len(columns))
	for i, col := range columns {
		headerRow[i] = col
	}
	if err := writer.Write(headerRow); err != nil {
		return "", 0, fmt.Errorf("写入CSV表头失败: %v", err)
	}

	recordCount := 0
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))

	for rows.Next() {
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return "", 0, fmt.Errorf("读取数据行失败: %v", err)
		}
		row := make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				row[i] = ""
			} else {
				switch v := val.(type) {
				case []byte:
					row[i] = string(v)
				case time.Time:
					row[i] = v.Format("2006-01-02 15:04:05")
				default:
					row[i] = fmt.Sprintf("%v", v)
				}
			}
		}
		if err := writer.Write(row); err != nil {
			return "", 0, fmt.Errorf("写入CSV数据失败: %v", err)
		}
		recordCount++
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", 0, fmt.Errorf("CSV写入错误: %v", err)
	}
	return filePath, recordCount, nil
}

func (e *TaskExecutor) testSQLExecution(dbConnID int64, sqlContent string, limit int) ([]string, [][]string, error) {
	dbConn, err := dbConnRepo.Get(dbConnID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取数据库连接失败: %v", err)
	}
	db, err := e.app.connectDB(dbConn)
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()

	processedSQL := e.app.ReplaceConstants(sqlContent)
	rows, err := db.Query(processedSQL)
	if err != nil {
		return nil, nil, fmt.Errorf("SQL执行失败: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("获取列名失败: %v", err)
	}

	var data [][]string
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	count := 0

	for rows.Next() && (limit <= 0 || count < limit) {
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return columns, data, fmt.Errorf("读取数据行失败: %v", err)
		}
		row := make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				row[i] = "NULL"
			} else {
				switch v := val.(type) {
				case []byte:
					row[i] = string(v)
				case time.Time:
					row[i] = v.Format("2006-01-02 15:04:05")
				default:
					row[i] = fmt.Sprintf("%v", v)
				}
			}
		}
		data = append(data, row)
		count++
	}
	return columns, data, nil
}

// ==================== CSV生成（App 工具方法） ====================

// GenerateFileName 根据模板与厂家/任务信息生成文件名
func (a *App) GenerateFileName(template, vendorCode, taskName string) string {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	dateStr := now.Format(a.GetConfigWithDefault("date_format", "20060102"))
	datetimeStr := now.Format(a.GetConfigWithDefault("datetime_format", "20060102_150405"))
	yesterdayStr := yesterday.Format(a.GetConfigWithDefault("date_format", "20060102"))
	yesterdayDateTimeStr := yesterday.Format(a.GetConfigWithDefault("datetime_format", "20060102_150405"))

	replacer := strings.NewReplacer(
		"{vendor_code}", vendorCode,
		"{task_name}", taskName,
		"{date}", dateStr,
		"{datetime}", datetimeStr,
		"{yyyy}", now.Format("2006"),
		"{mm}", now.Format("01"),
		"{dd}", now.Format("02"),
		"{HH}", now.Format("15"),
		"{MM}", now.Format("04"),
		"{SS}", now.Format("05"),
		"{yesterday}", yesterdayStr,
		"{yesterday_datetime}", yesterdayDateTimeStr,
	)
	return replacer.Replace(template)
}

// ==================== FTP/SFTP 上传与测试（App 工具方法） ====================

// TestDBConnection 测试源数据库连接（仅连接+Ping，不返回连接）
func (a *App) TestDBConnection(c *models.DBConnection) error {
	db, err := a.connectDB(c)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Ping()
}

// TestFTPConnection 测试FTP/SFTP连通性（仅连接+登录，不传文件）
func (a *App) TestFTPConnection(acc *models.FTPAccount) error {
	switch acc.Protocol {
	case "sftp":
		return a.testSFTPConn(acc)
	case "ftp":
		return a.testFTPConn(acc)
	default:
		return fmt.Errorf("不支持的协议: %s", acc.Protocol)
	}
}

func (a *App) testSFTPConn(acc *models.FTPAccount) error {
	config := &ssh.ClientConfig{
		User:            acc.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(acc.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", acc.Host, acc.Port), config)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %v", err)
	}
	defer conn.Close()
	client, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("SFTP会话建立失败: %v", err)
	}
	defer client.Close()
	if _, err := client.ReadDir(acc.RemotePath); err != nil {
		return fmt.Errorf("远程路径不可访问 '%s': %v", acc.RemotePath, err)
	}
	return nil
}

func (a *App) testFTPConn(acc *models.FTPAccount) error {
	conn, err := ftp.Dial(fmt.Sprintf("%s:%d", acc.Host, acc.Port), ftp.DialWithTimeout(10*time.Second))
	if err != nil {
		return fmt.Errorf("FTP连接失败: %v", err)
	}
	defer conn.Quit()
	if err := conn.Login(acc.Username, acc.Password); err != nil {
		return fmt.Errorf("FTP登录失败: %v", err)
	}
	if err := conn.ChangeDir(acc.RemotePath); err != nil {
		return fmt.Errorf("远程路径不可访问 '%s': %v", acc.RemotePath, err)
	}
	return nil
}

// TestSQLExecution 测试SQL执行（仅执行并返回前几行预览）
func (a *App) TestSQLExecution(dbConnID int64, sqlContent string, limit int) ([]string, [][]string, error) {
	return a.Executor.testSQLExecution(dbConnID, sqlContent, limit)
}

// UploadFile 按协议上传本地文件至远端
func (a *App) UploadFile(localPath string, ftpAccount *models.FTPAccount) error {
	switch ftpAccount.Protocol {
	case "sftp":
		return a.uploadSFTP(localPath, ftpAccount)
	case "ftp":
		return a.uploadFTP(localPath, ftpAccount)
	default:
		return fmt.Errorf("不支持的协议: %s", ftpAccount.Protocol)
	}
}

func (a *App) uploadSFTP(localPath string, acc *models.FTPAccount) error {
	config := &ssh.ClientConfig{
		User:            acc.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(acc.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", acc.Host, acc.Port), config)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %v", err)
	}
	defer conn.Close()

	client, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("SFTP客户端创建失败: %v", err)
	}
	defer client.Close()

	client.MkdirAll(acc.RemotePath)

	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %v", err)
	}
	defer localFile.Close()

	remoteFilePath := filepath.Join(acc.RemotePath, filepath.Base(localPath))
	remoteFile, err := client.Create(remoteFilePath)
	if err != nil {
		return fmt.Errorf("创建远程文件失败: %v", err)
	}
	defer remoteFile.Close()

	if _, err := io.Copy(remoteFile, localFile); err != nil {
		return fmt.Errorf("文件上传失败: %v", err)
	}
	log.Printf("[SFTP] 文件上传成功: %s -> %s", localPath, remoteFilePath)
	return nil
}

func (a *App) uploadFTP(localPath string, acc *models.FTPAccount) error {
	conn, err := ftp.Dial(fmt.Sprintf("%s:%d", acc.Host, acc.Port), ftp.DialWithTimeout(30*time.Second))
	if err != nil {
		return fmt.Errorf("FTP连接失败: %v", err)
	}
	defer conn.Quit()

	if err := conn.Login(acc.Username, acc.Password); err != nil {
		return fmt.Errorf("FTP登录失败: %v", err)
	}

	for _, dir := range strings.Split(strings.Trim(acc.RemotePath, "/"), "/") {
		if dir == "" {
			continue
		}
		if err := conn.ChangeDir(dir); err != nil {
			conn.MakeDir(dir)
			conn.ChangeDir(dir)
		}
	}

	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %v", err)
	}
	defer localFile.Close()

	remoteFileName := filepath.Base(localPath)
	if err := conn.Stor(remoteFileName, localFile); err != nil {
		return fmt.Errorf("FTP上传失败: %v", err)
	}
	log.Printf("[FTP] 文件上传成功: %s -> %s/%s", localPath, acc.RemotePath, remoteFileName)
	return nil
}

// ==================== 文件备份（App 工具方法） ====================

// BackupFile 备份输出文件到备份目录
func (a *App) BackupFile(localPath string) (string, error) {
	backupDir := a.GetConfigWithDefault("backup_dir", "./backup")
	os.MkdirAll(backupDir, 0755)

	timestamp := time.Now().Format("20060102_150405")
	baseName := filepath.Base(localPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	backupName := fmt.Sprintf("%s_%s%s", nameWithoutExt, timestamp, ext)
	backupPath := filepath.Join(backupDir, backupName)

	srcFile, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("打开源文件失败: %v", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("创建备份文件失败: %v", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return "", fmt.Errorf("复制备份文件失败: %v", err)
	}
	log.Printf("[备份] 文件备份成功: %s -> %s", localPath, backupPath)
	return backupPath, nil
}

// CleanOldBackups 按保留数量清理旧备份文件
func (a *App) CleanOldBackups() error {
	keepCountStr := a.GetConfigWithDefault("backup_keep_count", "30")
	keepCount := 30
	fmt.Sscanf(keepCountStr, "%d", &keepCount)

	backupDir := a.GetConfigWithDefault("backup_dir", "./backup")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	type fileInfo struct {
		path    string
		modTime time.Time
	}
	var files []fileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{path: filepath.Join(backupDir, entry.Name()), modTime: info.ModTime()})
	}
	if len(files) <= keepCount {
		return nil
	}
	sort.Slice(files, func(i, j int) bool { return files[i].modTime.After(files[j].modTime) })

	deletedCount := 0
	for i := keepCount; i < len(files); i++ {
		if err := os.Remove(files[i].path); err != nil {
			log.Printf("[备份清理] 删除失败: %s, %v", files[i].path, err)
		} else {
			deletedCount++
		}
	}
	if deletedCount > 0 {
		log.Printf("[备份清理] 清理完成，共删除 %d 个旧备份文件，保留最新 %d 个", deletedCount, keepCount)
	}
	return nil
}

// ConvertGBKToUTF8 GBK 字符串转 UTF-8
func (a *App) ConvertGBKToUTF8(gbkStr string) (string, error) {
	reader := transform.NewReader(bytes.NewReader([]byte(gbkStr)), simplifiedchinese.GBK.NewDecoder())
	d, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(d), nil
}

// ==================== 执行器单例与并发工作池 ====================

var (
	taskExecMu sync.RWMutex
	taskExec   *TaskExecutor
)

// SetTaskExecutor 注入任务执行器（供工作池与并发执行使用）
func SetTaskExecutor(e *TaskExecutor) {
	taskExecMu.Lock()
	taskExec = e
	taskExecMu.Unlock()
}

// defaultExecutor 取默认执行器实例（由 InitWorkerPool 前注入）
func defaultExecutor() *TaskExecutor {
	taskExecMu.RLock()
	e := taskExec
	taskExecMu.RUnlock()
	return e
}

// ==================== 运行中任务跟踪 ====================

// runningTasks 记录当前正在执行的任务 ID（供前端禁止重复执行）
var runningTasks sync.Map

// markRunning 标记任务开始执行
func markRunning(taskID int64) { runningTasks.Store(taskID, struct{}{}) }

// unmarkRunning 标记任务执行结束
func unmarkRunning(taskID int64) { runningTasks.Delete(taskID) }

// IsTaskRunning 判断任务是否正在执行
func IsTaskRunning(taskID int64) bool {
	_, ok := runningTasks.Load(taskID)
	return ok
}

// RunningTaskIDs 返回当前所有正在执行的任务 ID 列表
func RunningTaskIDs() []int64 {
	var ids []int64
	runningTasks.Range(func(k, _ interface{}) bool {
		if id, ok := k.(int64); ok {
			ids = append(ids, id)
		}
		return true
	})
	return ids
}

// ==================== 并发执行引擎 ====================

// TaskResult 单个任务执行结果
type TaskResult struct {
	TaskID   int64  `json:"task_id"`
	TaskName string `json:"task_name"`
	VendorID int64  `json:"vendor_id"`
	Status   string `json:"status"`
	Error    error  `json:"-"`
	ErrorMsg string `json:"error_msg,omitempty"`
	Duration int64  `json:"duration_ms"`
	CSVFile  string `json:"csv_file,omitempty"`
}

// TaskWorkerPool 任务执行工作池，信号量控制并发
type TaskWorkerPool struct {
	sem    chan struct{} // 信号量
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
	exec   *TaskExecutor
}

var (
	globalPool     *TaskWorkerPool
	globalPoolOnce sync.Once
	globalPoolMu   sync.RWMutex
)

// InitWorkerPool 初始化全局工作池（需注入执行器）
func InitWorkerPool(exec *TaskExecutor) {
	globalPoolOnce.Do(func() {
		max := getMaxParallel(exec.app)
		globalPool = newWorkerPool(max, exec)
		log.Printf("[工作池] 初始化完成，最大并发: %d", max)
	})
}

// getMaxParallel 读取配置的最大并发数
func getMaxParallel(app *App) int {
	s := app.GetConfigWithDefault("max_parallel_tasks", "3")
	n, _ := strconv.Atoi(s)
	if n < 1 {
		n = 1
	}
	if n > 20 {
		n = 20
	}
	return n
}

func newWorkerPool(maxConcurrent int, exec *TaskExecutor) *TaskWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &TaskWorkerPool{
		sem:    make(chan struct{}, maxConcurrent),
		ctx:    ctx,
		cancel: cancel,
		exec:   exec,
	}
}

// Resize 动态调整并发数（配置变更后调用）
func (p *TaskWorkerPool) Resize(newSize int) {
	if newSize < 1 {
		newSize = 1
	}
	if newSize > 20 {
		newSize = 20
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	oldSem := p.sem
	newSem := make(chan struct{}, newSize)
	p.sem = newSem
	close(oldSem)
	log.Printf("[工作池] 并发数调整为: %d", newSize)
}

// GetGlobalPool 获取全局工作池
func GetGlobalPool() *TaskWorkerPool {
	globalPoolMu.RLock()
	p := globalPool
	globalPoolMu.RUnlock()
	if p == nil {
		InitWorkerPool(defaultExecutor())
		globalPoolMu.RLock()
		p = globalPool
		globalPoolMu.RUnlock()
	}
	return p
}

// StopWorkerPool 停止工作池
func StopWorkerPool() {
	if globalPool != nil {
		globalPool.cancel()
		globalPool.wg.Wait()
	}
}

// Submit 提交单个任务（非阻塞，返回 nil 表示已提交）
func (p *TaskWorkerPool) Submit(taskID int64) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.executeOne(taskID)
	}()
}

// SubmitBatch 批量提交任务并等待全部完成，返回结果
func (p *TaskWorkerPool) SubmitBatch(taskIDs []int64) []*TaskResult {
	var wg sync.WaitGroup
	results := make([]*TaskResult, len(taskIDs))
	resultMap := make(map[int64]*TaskResult)

	var mu sync.Mutex

	for i, tid := range taskIDs {
		idx := i
		id := tid
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := p.executeOne(id)

			mu.Lock()
			results[idx] = r
			resultMap[id] = r
			mu.Unlock()
		}()
	}
	wg.Wait()

	// 过滤 nil
	var out []*TaskResult
	for _, r := range results {
		if r != nil {
			out = append(out, r)
		}
	}
	return out
}

// executeOne 获取信号量后执行单个任务
func (p *TaskWorkerPool) executeOne(taskID int64) *TaskResult {
	p.mu.Lock()
	sem := p.sem
	p.mu.Unlock()

	// 获取信号量（带超时）
	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
	case <-p.ctx.Done():
		return &TaskResult{TaskID: taskID, Status: "cancelled", ErrorMsg: "工作池已停止"}
	}

	startTime := time.Now()
	log.Printf("[工作池] 开始执行任务 #%d", taskID)

	markRunning(taskID)
	defer unmarkRunning(taskID)

	logEntry, err := p.exec.Execute(taskID)
	duration := time.Since(startTime).Milliseconds()

	result := &TaskResult{
		TaskID:   taskID,
		Status:   "success",
		Duration: duration,
	}

	if err != nil {
		result.Status = "failed"
		result.Error = err
		result.ErrorMsg = err.Error()
		log.Printf("[工作池] 任务 #%d 执行失败 (耗时%dms): %v", taskID, duration, err)
	} else {
		log.Printf("[工作池] 任务 #%d 执行成功 (耗时%dms)", taskID, duration)
	}

	if logEntry != nil {
		result.CSVFile = logEntry.CSVFilename
		result.TaskName = ""
		result.VendorID = logEntry.VendorID
	}

	return result
}

// ExecuteTasksConcurrent 并发执行多个任务（对外 API）
func ExecuteTasksConcurrent(taskIDs []int64) []*TaskResult {
	pool := GetGlobalPool()
	return pool.SubmitBatch(taskIDs)
}

// ==================== 源数据库连接池缓存 ====================

type dbConnCache struct {
	mu    sync.Mutex
	cache map[string]*sql.DB // key: "dbType:host:port:dbName:user"
}

var dbCache = &dbConnCache{
	cache: make(map[string]*sql.DB),
}

func (c *dbConnCache) get(conn *models.DBConnection) (*sql.DB, error) {
	key := fmt.Sprintf("%s:%s:%d:%s:%s", conn.DBType, conn.Host, conn.Port, conn.DatabaseName, conn.Username)

	c.mu.Lock()
	if db, ok := c.cache[key]; ok {
		// 验证连接是否存活
		if err := db.Ping(); err == nil {
			c.mu.Unlock()
			return db, nil
		}
		// 连接已失效，关闭并移除
		db.Close()
		delete(c.cache, key)
	}
	c.mu.Unlock()

	// 创建新连接（复用 App.connectDB 逻辑）
	db, err := connectDBViaApp(conn)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[key] = db
	c.mu.Unlock()

	return db, nil
}

// connectDBViaApp 通过全局默认执行器的 App 连接数据库（供缓存复用）
func connectDBViaApp(conn *models.DBConnection) (*sql.DB, error) {
	if e := defaultExecutor(); e != nil {
		return e.app.connectDB(conn)
	}
	return nil, fmt.Errorf("执行器未初始化")
}

// GetCachedDB 从缓存获取数据库连接（复用同一数据源）
func GetCachedDB(conn *models.DBConnection) (*sql.DB, error) {
	return dbCache.get(conn)
}

// CloseAllCachedDBs 关闭所有缓存的数据库连接
func CloseAllCachedDBs() {
	dbCache.mu.Lock()
	defer dbCache.mu.Unlock()
	for key, db := range dbCache.cache {
		db.Close()
		delete(dbCache.cache, key)
	}
	log.Printf("[DB缓存] 已关闭所有缓存数据库连接 (%d)", len(dbCache.cache))
}
