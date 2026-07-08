package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"data-exchange/handlers"
	"data-exchange/models"
	"data-exchange/services"
)

func main() {
	port := flag.Int("port", 7856, "服务端口")
	dbPath := flag.String("db", "data.db", "系统数据库路径")
	flag.Parse()

	if err := models.InitDB(*dbPath); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer models.CloseDB()

	services.InitScheduler()
	defer services.StopScheduler()
	defer services.CloseAllCachedDBs()

	outputDir := services.GetConfigWithDefault("csv_output_dir", "./output")
	os.MkdirAll(outputDir, 0755)
	backupDir := services.GetConfigWithDefault("backup_dir", "./backup")
	os.MkdirAll(backupDir, 0755)

	router := handlers.SetupRouter()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("正在关闭服务...")
		services.StopScheduler()
		models.CloseDB()
		os.Exit(0)
	}()

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	log.Printf("========================================")
	log.Printf("  数据交换系统 启动成功!")
	log.Printf("  监听地址: %s", addr)
	log.Printf("  数据库: %s", *dbPath)
	log.Printf("  输出目录: %s", outputDir)
	log.Printf("  备份目录: %s", backupDir)
	log.Printf("========================================")

	if err := router.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
