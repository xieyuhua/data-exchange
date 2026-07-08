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

	"github.com/jlaffaye/ftp"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/pkg/sftp"
	_ "github.com/sijms/go-ora/v2"
	"golang.org/x/crypto/ssh"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// ==================== 系统配置服务 ====================

func GetConfig(key string) string {
	var c models.SystemConfig
	if err := models.DB.Where("config_key = ?", key).First(&c).Error; err != nil {
		return ""
	}
	return c.ConfigValue
}

func SetConfig(key, value string) error {
	var c models.SystemConfig
	result := models.DB.Where("config_key = ?", key).First(&c)
	if result.Error != nil {
		c = models.SystemConfig{ConfigKey: key, ConfigValue: value}
		return models.DB.Create(&c).Error
	}
	return models.DB.Model(&c).Update("config_value", value).Error
}

func GetAllConfigs() ([]models.SystemConfig, error) {
	var configs []models.SystemConfig
	if err := models.DB.Order("id").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// ==================== 常量服务 ====================

func GetAllConstants() ([]models.Constant, error) {
	var constants []models.Constant
	if err := models.DB.Order("id").Find(&constants).Error; err != nil {
		return nil, err
	}
	return constants, nil
}

func GetConstantMap() (map[string]string, error) {
	constants, err := GetAllConstants()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(constants))
	for _, c := range constants {
		m[c.Key] = c.Value
	}
	return m, nil
}

func SaveConstant(c *models.Constant) error {
	if c.ID == 0 {
		return models.DB.Create(c).Error
	}
	return models.DB.Model(&models.Constant{}).Where("id = ?", c.ID).Updates(map[string]interface{}{
		"key": c.Key, "value": c.Value, "description": c.Description,
	}).Error
}

func DeleteConstant(id int64) error {
	return models.DB.Delete(&models.Constant{}, id).Error
}

func ReplaceConstants(sqlContent string) string {
	constants, err := GetConstantMap()
	if err != nil {
		return sqlContent
	}
	result := sqlContent
	for key, value := range constants {
		result = strings.ReplaceAll(result, fmt.Sprintf("{{%s}}", key), value)
	}
	return result
}

// ==================== 数据库连接服务 ====================

func connectDB(conn *models.DBConnection) (*sql.DB, error) {
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

func GetDBConnection(id int64) (*models.DBConnection, error) {
	var c models.DBConnection
	if err := models.DB.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func TestDBConnection(c *models.DBConnection) error {
	db, err := connectDB(c)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Ping()
}

// ==================== CSV生成服务 ====================

func GetConfigWithDefault(key, defaultVal string) string {
	val := GetConfig(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func GenerateFileName(template, vendorCode, taskName string) string {
	now := time.Now()
	dateStr := now.Format(GetConfigWithDefault("date_format", "20060102"))
	datetimeStr := now.Format(GetConfigWithDefault("datetime_format", "20060102_150405"))

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
	)
	return replacer.Replace(template)
}

func ExecuteSQLAndGenerateCSV(task *models.SQLTask, vendor *models.Vendor) (string, int, error) {
	dbConn, err := GetDBConnection(*task.DBConnectionID)
	if err != nil {
		return "", 0, fmt.Errorf("获取数据库连接失败: %v", err)
	}

	db, err := connectDB(dbConn)
	if err != nil {
		return "", 0, err
	}
	defer db.Close()

	sqlContent := ReplaceConstants(task.SQLContent)
	rows, err := db.Query(sqlContent)
	if err != nil {
		return "", 0, fmt.Errorf("SQL执行失败: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return "", 0, fmt.Errorf("获取列名失败: %v", err)
	}

	outputDir := GetConfigWithDefault("csv_output_dir", "./output")
	os.MkdirAll(outputDir, 0755)
	fileName := GenerateFileName(task.CSVFilenameTemplate, vendor.Code, task.TaskName)
	if !strings.HasSuffix(fileName, ".csv") {
		fileName += ".csv"
	}
	filePath := filepath.Join(outputDir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("创建CSV文件失败: %v", err)
	}
	defer file.Close()

	if GetConfigWithDefault("csv_bom", "true") == "true" {
		file.Write([]byte{0xEF, 0xBB, 0xBF})
	}

	writer := csv.NewWriter(file)
	if delim := GetConfigWithDefault("csv_delimiter", ","); len(delim) > 0 {
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

// ==================== FTP/SFTP 上传 ====================

func GetFTPAccount(id int64) (*models.FTPAccount, error) {
	var a models.FTPAccount
	if err := models.DB.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func UploadFile(localPath string, ftpAccount *models.FTPAccount) error {
	switch ftpAccount.Protocol {
	case "sftp":
		return uploadSFTP(localPath, ftpAccount)
	case "ftp":
		return uploadFTP(localPath, ftpAccount)
	default:
		return fmt.Errorf("不支持的协议: %s", ftpAccount.Protocol)
	}
}

func uploadSFTP(localPath string, acc *models.FTPAccount) error {
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

func uploadFTP(localPath string, acc *models.FTPAccount) error {
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

// ==================== 文件备份 ====================

func BackupFile(localPath string) (string, error) {
	backupDir := GetConfigWithDefault("backup_dir", "./backup")
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

func CleanOldBackups() error {
	keepCountStr := GetConfigWithDefault("backup_keep_count", "30")
	keepCount := 30
	fmt.Sscanf(keepCountStr, "%d", &keepCount)

	backupDir := GetConfigWithDefault("backup_dir", "./backup")
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

// ==================== 任务执行 ====================

func ExecuteTask(taskID int64) (*models.ExportLog, error) {
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
		NotifyFailure(taskName, vendorName, errMsg)
	}

	task, err := GetTaskByID(taskID)
	if err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("获取任务失败: %v", err)
		insertLog(logEntry)
		notifyFail(fmt.Sprintf("#%d", taskID), "", logEntry.ErrorMessage)
		return logEntry, err
	}
	logEntry.VendorID = task.VendorID
	logEntry.ExecutionMode = task.ExecutionMode

	var vendor models.Vendor
	if err := models.DB.First(&vendor, task.VendorID).Error; err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("获取厂家失败: %v", err)
		insertLog(logEntry)
		notifyFail(task.TaskName, "", logEntry.ErrorMessage)
		return logEntry, err
	}

	csvPath, recordCount, err := ExecuteSQLAndGenerateCSV(task, &vendor)
	if err != nil {
		logEntry.ErrorMessage = fmt.Sprintf("生成CSV失败: %v", err)
		insertLog(logEntry)
		notifyFail(task.TaskName, vendor.Name, logEntry.ErrorMessage)
		return logEntry, err
	}
	logEntry.CSVFilename = filepath.Base(csvPath)
	logEntry.RecordCount = recordCount

	if fileInfo, err := os.Stat(csvPath); err == nil {
		logEntry.FileSize = fileInfo.Size()
	}

	if _, err := BackupFile(csvPath); err != nil {
		log.Printf("[任务执行] 备份文件警告: %v", err)
	}

	if task.ExecutionMode == "upload" && task.FTPAccountID != nil {
		ftpAccount, err := GetFTPAccount(*task.FTPAccountID)
		if err != nil {
			logEntry.ErrorMessage = fmt.Sprintf("获取FTP账号失败: %v", err)
			insertLog(logEntry)
			notifyFail(task.TaskName, vendor.Name, logEntry.ErrorMessage)
			return logEntry, err
		}
		if err := UploadFile(csvPath, ftpAccount); err != nil {
			logEntry.ErrorMessage = fmt.Sprintf("文件上传失败: %v", err)
			insertLog(logEntry)
			notifyFail(task.TaskName, vendor.Name, logEntry.ErrorMessage)
			return logEntry, err
		}
		log.Printf("[任务执行] 文件上传成功: %s", csvPath)
	}

	go CleanOldBackups()

	logEntry.Status = "success"
	logEntry.FinishedAt = time.Now().Format("2006-01-02 15:04:05")
	logEntry.DurationMs = time.Since(startTime).Milliseconds()
	logEntry.ErrorMessage = ""
	insertLog(logEntry)

	models.DB.Model(&models.SQLTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"last_run_at": time.Now().Format("2006-01-02 15:04:05"),
		"last_status": "success",
	})

	return logEntry, nil
}

func insertLog(logEntry *models.ExportLog) {
	if err := models.DB.Create(logEntry).Error; err != nil {
		log.Printf("[日志] 写入执行日志失败: %v", err)
	}
}

func GetTaskByID(id int64) (*models.SQLTask, error) {
	var t models.SQLTask
	if err := models.DB.First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func ExecuteTaskByName(taskName string) ([]*models.ExportLog, error) {
	var tasks []models.SQLTask
	if err := models.DB.Where("task_name = ? AND enabled = 1", taskName).Find(&tasks).Error; err != nil {
		return nil, err
	}
	var logs []*models.ExportLog
	for _, t := range tasks {
		l, err := ExecuteTask(t.ID)
		if err != nil {
			logs = append(logs, l)
			continue
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// ==================== GBK转UTF-8辅助 ====================

func ConvertGBKToUTF8(gbkStr string) (string, error) {
	reader := transform.NewReader(bytes.NewReader([]byte(gbkStr)), simplifiedchinese.GBK.NewDecoder())
	d, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(d), nil
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
	sem    chan struct{}   // 信号量
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
}

var (
	globalPool     *TaskWorkerPool
	globalPoolOnce sync.Once
	globalPoolMu   sync.RWMutex
)

// InitWorkerPool 初始化全局工作池
func InitWorkerPool() {
	globalPoolOnce.Do(func() {
		max := getMaxParallel()
		globalPool = newWorkerPool(max)
		log.Printf("[工作池] 初始化完成，最大并发: %d", max)
	})
}

func getMaxParallel() int {
	s := GetConfigWithDefault("max_parallel_tasks", "3")
	n, _ := strconv.Atoi(s)
	if n < 1 {
		n = 1
	}
	if n > 20 {
		n = 20
	}
	return n
}

func newWorkerPool(maxConcurrent int) *TaskWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &TaskWorkerPool{
		sem:    make(chan struct{}, maxConcurrent),
		ctx:    ctx,
		cancel: cancel,
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

	// 创建新信号量，旧信号量上的等待者会被唤醒
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
		InitWorkerPool()
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

	logEntry, err := ExecuteTask(taskID)
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
		result.TaskName = "" // 可从 task 填充
		result.VendorID = logEntry.VendorID
	}

	return result
}

// ExecuteTasksConcurrent 并发执行多个任务（对外 API）
func ExecuteTasksConcurrent(taskIDs []int64) []*TaskResult {
	pool := GetGlobalPool()
	return pool.SubmitBatch(taskIDs)
}

// ExecuteTaskByNameConcurrent 按任务名并发执行所有匹配任务
func ExecuteTaskByNameConcurrent(taskName string) ([]*TaskResult, error) {
	var tasks []models.SQLTask
	if err := models.DB.Where("task_name = ? AND enabled = 1", taskName).Find(&tasks).Error; err != nil {
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
	results := ExecuteTasksConcurrent(taskIDs)

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

	// 创建新连接
	db, err := connectDB(conn)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[key] = db
	c.mu.Unlock()

	return db, nil
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
