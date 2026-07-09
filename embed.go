package main

import "embed"

// staticFS 将前端构建产物(static/)在编译期嵌入二进制，
// 使 data-exchange 成为单文件可执行程序，无需外部携带 static 目录。
// 注意：需先执行 web 前端构建 (npm run build) 生成 static 目录再编译 Go。
//
// 若希望使用外部目录的前端文件(例如独立部署/热更新前端)，
// 启动时可指定 -web-root 目录，或配置 config.yaml 的 web_root，留空则使用本内嵌资源。
//
//go:embed static
var staticFS embed.FS
