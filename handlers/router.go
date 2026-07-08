package handlers

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"data-exchange/models"
	"data-exchange/services"

	"github.com/gin-gonic/gin"
)

// SetupRouter 使用嵌入的前端静态资源构建路由。
// staticFS 由 main 包通过 //go:embed 注入，包含 static 目录树。
// app 为已装配的应用服务聚合根（依赖注入）。
func SetupRouter(staticFS embed.FS, app *services.App) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 从嵌入文件系统剥离 "static" 前缀，得到可直接服务的子文件系统
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("加载嵌入的前端资源失败: %v", err)
	}
	fileSystem := http.FS(sub)

	// 静态资源挂在 /static 前缀下（与前端 vite base='/static/' 对应），
	// 不能挂在根路径 "/"，否则会注册 catch-all /*filepath 与 /api 路由冲突导致 panic。
	r.StaticFS("/static", fileSystem)

	// 首页：直接读取嵌入的 index.html 返回（其中资源均引用 /static/...，由上面提供）
	r.GET("/", func(c *gin.Context) {
		data, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "index.html not found"})
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	h := NewHandler(app)

	// 鉴权相关接口（免鉴权）
	auth := r.Group("/api/auth")
	{
		auth.POST("/login", h.Login)
		auth.GET("/me", AuthMiddleware(), h.Me)
	}

	api := r.Group("/api")
	api.Use(AuthMiddleware())
	{
		api.GET("/dashboard/stats", h.DashboardStats)

		// 系统常量
		api.GET("/constants", h.ListConstants)
		api.POST("/constants", h.SaveConstant)
		api.DELETE("/constants/:id", h.DeleteConstant)

		// 数据库连接
		api.GET("/db-connections", h.ListDBConnections)
		api.POST("/db-connections", h.SaveDBConnection)
		api.DELETE("/db-connections/:id", h.DeleteDBConnection)
		api.POST("/db-connections/test", h.TestDBConnection)

		// 厂家
		api.GET("/vendors", h.ListVendors)
		api.POST("/vendors", h.SaveVendor)
		api.DELETE("/vendors/:id", h.DeleteVendor)
		api.GET("/vendors/:id/tasks", h.GetVendorTasks)

		// SQL任务
		api.POST("/tasks", h.SaveSQLTask)
		api.GET("/tasks/:id", h.GetSQLTask)
		api.DELETE("/tasks/:id", h.DeleteSQLTask)
		api.POST("/tasks/:id/toggle", h.ToggleSQLTask)
		api.POST("/tasks/:id/execute", h.ExecuteTaskNow)
		api.GET("/tasks/running", h.ListRunningTasks)
		api.POST("/tasks/execute-by-name", h.ExecuteTaskByName)
		api.POST("/tasks/batch-execute", h.BatchExecuteTasks)
		api.POST("/tasks/test-sql", h.TestSQLExecution)
		api.POST("/tasks/test-sql-export", h.ExportTestSQLResult)

		// FTP账号
		api.GET("/ftp-accounts", h.ListFTPAccounts)
		api.POST("/ftp-accounts", h.SaveFTPAccount)
		api.DELETE("/ftp-accounts/:id", h.DeleteFTPAccount)
		api.POST("/ftp-accounts/test", h.TestFTPConnection)

		// 系统配置
		api.GET("/configs", h.ListSystemConfigs)
		api.POST("/configs", h.SaveSystemConfig)

		// 执行日志
		api.GET("/logs", h.ListExportLogs)
		api.DELETE("/logs/:id", h.DeleteExportLog)
		api.DELETE("/logs", h.ClearExportLogs)

		// 文件管理
		api.GET("/files/output", h.ListOutputFiles)
		api.GET("/files/download", h.DownloadFile)
		api.GET("/files/backup", h.ListBackupFiles)
		api.POST("/files/clean-backups", h.CleanBackupsNow)

		// 通知测试
		api.POST("/notify/test", h.TestNotify)

		// 常量函数求值
		api.POST("/constants/eval", h.EvalConstantFunc)
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "not found"})
	})

	return r
}
