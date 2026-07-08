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
	flag.Parse()

	if err := config.Load(*configPath); err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
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

	router := handlers.SetupRouter(staticFS, app)

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
	log.Printf("  数据库: %s", dbDesc)
	log.Printf("  输出目录: %s", outputDir)
	log.Printf("  备份目录: %s", backupDir)
	log.Printf("========================================")

	if err := router.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
