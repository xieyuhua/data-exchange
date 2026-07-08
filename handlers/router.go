package handlers

import (
	"net/http"

	"data-exchange/models"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 静态文件（前端构建产物，Vue 打包后在此目录）
	r.Static("/static", "./static")

	// 首页
	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	api := r.Group("/api")
	{
		api.GET("/dashboard/stats", DashboardStats)

		// 系统常量
		api.GET("/constants", ListConstants)
		api.POST("/constants", SaveConstant)
		api.DELETE("/constants/:id", DeleteConstant)

		// 数据库连接
		api.GET("/db-connections", ListDBConnections)
		api.POST("/db-connections", SaveDBConnection)
		api.DELETE("/db-connections/:id", DeleteDBConnection)
		api.POST("/db-connections/test", TestDBConnection)

		// 厂家
		api.GET("/vendors", ListVendors)
		api.POST("/vendors", SaveVendor)
		api.DELETE("/vendors/:id", DeleteVendor)
		api.GET("/vendors/:id/tasks", GetVendorTasks)

		// SQL任务
		api.POST("/tasks", SaveSQLTask)
		api.DELETE("/tasks/:id", DeleteSQLTask)
		api.POST("/tasks/:id/toggle", ToggleSQLTask)
		api.POST("/tasks/:id/execute", ExecuteTaskNow)
		api.POST("/tasks/execute-by-name", ExecuteTaskByName)
		api.POST("/tasks/batch-execute", BatchExecuteTasks)
		api.POST("/tasks/test-sql", TestSQLExecution)

		// FTP账号
		api.GET("/ftp-accounts", ListFTPAccounts)
		api.POST("/ftp-accounts", SaveFTPAccount)
		api.DELETE("/ftp-accounts/:id", DeleteFTPAccount)
		api.POST("/ftp-accounts/test", TestFTPConnection)

		// 系统配置
		api.GET("/configs", ListSystemConfigs)
		api.POST("/configs", SaveSystemConfig)

		// 执行日志
		api.GET("/logs", ListExportLogs)
		api.DELETE("/logs/:id", DeleteExportLog)
		api.DELETE("/logs", ClearExportLogs)

		// 文件管理
		api.GET("/files/output", ListOutputFiles)
		api.GET("/files/download", DownloadFile)
		api.GET("/files/backup", ListBackupFiles)
		api.POST("/files/clean-backups", CleanBackupsNow)

		// 通知测试
		api.POST("/notify/test", TestNotify)

		// 常量函数求值
		api.POST("/constants/eval", EvalConstantFunc)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "not found"})
	})

	return r
}
