package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"data-exchange/config"
	"data-exchange/handlers"
	"data-exchange/models"
	"data-exchange/services"
)

func main() {
	port := flag.Int("port", 7856, "服务端口")
	configPath := flag.String("config", "config.yaml", "配置文件路径 (yaml)")
	webRootFlag := flag.String("web-root", "", "前端静态资源目录(外部目录)。留空则使用内嵌的 static 资源")
	flag.Parse()

	if err := config.Load(*configPath); err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 子命令: genddl —— 生成建表与初始数据 SQL 文件，供手动导入（auto_migrate: false 场景）
	if flag.Arg(0) == "genddl" {
		out := "schema.sql"
		if len(flag.Args()) > 1 {
			out = flag.Arg(1)
		}
		sql, err := models.GenSchemaSQL(config.AppConfig.Database.Type)
		if err != nil {
			log.Fatalf("生成建表 SQL 失败: %v", err)
		}
		if err := os.WriteFile(out, []byte(sql), 0644); err != nil {
			log.Fatalf("写入 SQL 文件失败: %v", err)
		}
		log.Printf("[genddl] 已生成建表/初始数据 SQL: %s (方言: %s)", out, config.AppConfig.Database.Type)
		log.Printf("[genddl] 将 auto_migrate 设为 false 后，在数据库中手动执行该文件即可完成建表与初始化。")
		return
	}

	if err := models.InitDB(config.AppConfig.Database); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer models.CloseDB()

	app := services.NewApp()
	app.EnsureDefaults()
	app.InitScheduler()
	defer app.StopScheduler()
	defer services.CloseAllCachedDBs()

	outputDir := app.GetConfigWithDefault("csv_output_dir", "./output")
	os.MkdirAll(outputDir, 0755)
	backupDir := app.GetConfigWithDefault("backup_dir", "./backup")
	os.MkdirAll(backupDir, 0755)

	// 外部前端目录优先使用命令行参数，其次使用配置文件 web_root；留空则使用内嵌资源
	webRoot := *webRootFlag
	if webRoot == "" {
		webRoot = config.AppConfig.WebRoot
	}
	if webRoot != "" {
		if info, err := os.Stat(webRoot); err != nil || !info.IsDir() {
			log.Fatalf("指定的前端目录不存在: %s", webRoot)
		}
	}

	router := handlers.SetupRouter(staticFS, app, webRoot)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("正在关闭服务...")
		app.StopScheduler()
		models.CloseDB()
		os.Exit(0)
	}()

	dbDesc := config.AppConfig.Database.Type
	if dbDesc == "sqlite" {
		dbDesc = "sqlite: " + config.AppConfig.Database.SQLitePath
	} else {
		m := config.AppConfig.Database.MySQL
		dbDesc = fmt.Sprintf("mysql: %s:%d/%s", m.Host, m.Port, m.Database)
	}

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	log.Printf("========================================")
	log.Printf("  数据交换系统 启动成功!")
	log.Printf("  监听地址: %s", addr)
	webMode := "内嵌(embed)"
	if webRoot != "" {
		webMode = "外部目录: " + webRoot
	}
	log.Printf("  前端资源: %s", webMode)
	amMode := "开启(自动建表)"
	if !config.ShouldAutoMigrate() {
		amMode = "关闭(跳过自动建表)"
	}
	log.Printf("  自动建表: %s", amMode)
	log.Printf("  数据库: %s", dbDesc)
	log.Printf("  输出目录: %s", outputDir)
	log.Printf("  备份目录: %s", backupDir)
	log.Printf("========================================")

	if err := router.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
