# 数据交换系统 (data-exchange)

一个基于 Go + Gin 的定时数据抽取与文件交换系统，系统数据库支持 **SQLite / MySQL**（由 `config.yaml` 配置）。支持按厂家配置 SQL 任务，定时执行查询并将结果导出为 CSV，可选择通过 FTP/SFTP 上传到目标服务器；任务执行失败时可通过 **钉钉机器人 / 企业微信机器人** 发送提醒。

## 功能特性

- **厂家管理**：维护数据提供方（厂家）基本信息，每个厂家可配置的 SQL 任务数上限（默认 4，可在系统配置 `max_tasks_per_vendor` 中调整）。
- **SQL 任务**：
  - 关联数据库连接、Cron 表达式定时调度。
  - 支持 SQL 常量占位符 `{{常量名}}` 替换。
  - 两种执行模式：`export_only`（仅导出 CSV）、`upload`（导出并上传）。
  - 任务启停、手动立即执行。
- **多数据源**：MySQL、Oracle、PostgreSQL、SQL Server。
- **文件交换**：SFTP / FTP 上传，本地备份与自动清理（保留最新 N 个）。
- **失败提醒**：任务执行失败自动推送至钉钉 / 企业微信。
- **可视化后台**：内置 Web 管理界面（仪表盘、日志、文件下载等）。

## 目录结构

```
data-exchange/
├── main.go              # 程序入口、参数解析、优雅退出
├── config.yaml          # 系统数据库配置（sqlite / mysql）
├── config/              # 配置文件解析
├── go.mod / go.sum      # Go 模块依赖
├── models/              # 数据模型与数据库初始化（sqlite / mysql）
├── repository/          # 数据访问层（CRUD / 分页查询）
├── services/            # 业务逻辑（连接、CSV、FTP、调度、通知、任务执行）
│   ├── services.go      # 连接/CSV/FTP/备份/任务执行/并发控制
│   ├── scheduler.go     # Cron 调度器
│   ├── crud.go          # 厂家/任务等业务 CRUD
│   └── notify.go        # 钉钉/企业微信失败提醒
├── handlers/            # HTTP 路由与接口处理
└── web/                 # 前端管理界面源码（Vue2 + Vite，构建后输出到 static/）
```

## 构建与运行

### 前置条件

- Go 1.21 及以上
- 网络可访问（首次 `go mod tidy` 需拉取依赖）

### 构建

```bash
go mod tidy      # 拉取/校验依赖（首次）
go build -o data-exchange.exe .
```

### 运行

```bash
# 默认端口 7856，读取当前目录 config.yaml
./data-exchange.exe
# 或指定端口与配置文件
./data-exchange.exe -port 8090 -config /etc/data-exchange/config.yaml
```

启动后浏览器访问：`http://localhost:<port>`

### 命令行参数

| 参数      | 默认值        | 说明                     |
| --------- | ------------- | ------------------------ |
| `-port`   | `7856`        | HTTP 服务端口            |
| `-config` | `config.yaml` | 配置文件路径（YAML）     |

## 系统数据库（SQLite / MySQL）

系统自身的元数据库（厂家、任务、配置、日志等）支持两种存储后端，由 `config.yaml` 的 `database` 段决定：

```yaml
database:
  type: sqlite          # sqlite 或 mysql
  sqlite_path: data.db  # type=sqlite 时的文件路径
  mysql:                # type=mysql 时的连接信息
    host: 127.0.0.1
    port: 3306
    user: root
    password: ""
    database: data_exchange
    params: "charset=utf8mb4&parseTime=True&loc=Local"
```

- `type: sqlite`：使用本地文件（默认 `data.db`，沿用 WAL 模式）。
- `type: mysql`：连接外部 MySQL，`params` 一般保持默认即可。
- 配置文件不存在时使用内置默认值（sqlite + `data.db`）。
- 首次启动会自动 `AutoMigrate` 建表并初始化默认管理员（`admin / admin2026`）与系统配置。

## 配置说明

系统配置在后台 **系统配置** 页面维护，存储于 `system_config` 表。常用配置：

| 配置键             | 默认值        | 说明                                   |
| ------------------ | ------------- | -------------------------------------- |
| `csv_output_dir`   | `./output`    | CSV 导出文件目录                       |
| `backup_dir`       | `./backup`    | 文件备份目录                           |
| `backup_keep_count`| `30`          | 备份保留数量，超出自动清理最旧文件     |
| `csv_delimiter`    | `,`           | CSV 字段分隔符                         |
| `csv_bom`          | `true`        | 是否写入 UTF-8 BOM                     |
| `date_format`      | `20060102`    | 文件名日期格式                         |
| `datetime_format`  | `20060102_150405` | 文件名日期时间格式                 |
| `max_parallel_tasks`| `3`          | 最大并行任务数                         |
| `max_tasks_per_vendor`| `4`        | 每个厂家允许的最大 SQL 任务数（可在后台调整） |

### 文件名模板占位符

`csv_filename_template` 支持：`{vendor_code}`、`{task_name}`、`{date}`、`{datetime}`、`{yyyy}`、`{mm}`、`{dd}`、`{HH}`、`{MM}`、`{SS}`。

### 钉钉 / 企业微信 失败提醒配置

在 **系统配置** 页面设置以下键值（也可直接改数据库）：

| 配置键              | 示例 / 取值                | 说明                                              |
| ------------------- | -------------------------- | ------------------------------------------------- |
| `notify_ding_enabled` | `on` / `off`             | 钉钉提醒开关                                       |
| `notify_ding_webhook` | `https://oapi.dingtalk.com/robot/send?access_token=xxxx` | 钉钉机器人 Webhook 地址              |
| `notify_ding_secret`  | `SECxxxxxxxx`             | 钉钉加签密钥（安全设置选「加签」时必填，可空）     |
| `notify_wx_enabled`   | `on` / `off`             | 企业微信提醒开关                                   |
| `notify_wx_webhook`   | `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxxx` | 企业微信群机器人 Webhook 地址     |
| `notify_at`           | `13800001111,@all`       | 失败提醒 @ 的成员手机号/userid，逗号分隔，`@all` 为所有人 |

**钉钉机器人配置步骤：**
1. 群设置 → 智能群助手 → 添加自定义机器人。
2. 安全设置选择「加签」，复制 Webhook 与签名密钥，分别填入上表对应配置。

**企业微信机器人配置步骤：**
1. 群聊 → 添加群机器人 → 获取 Webhook 地址。
2. 将地址填入 `notify_wx_webhook` 并开启 `notify_wx_enabled`。

> 提醒机制：每次任务执行仅在**最终失败**时发送一次提醒（同一执行过程不重复发送）。钉钉采用 HmacSHA256 + Base64 加签；企业微信支持 `mentioned_list` 指定 @ 成员。

## API 概览

| 分组       | 方法 & 路径                         | 说明                 |
| ---------- | ----------------------------------- | -------------------- |
| 仪表盘     | `GET /api/dashboard/stats`          | 统计概览             |
| 常量       | `GET/POST /api/constants` `DELETE /api/constants/:id` | 系统常量 CRUD |
| 数据库连接 | `GET/POST /api/db-connections` 等   | 数据库连接 CRUD + 测试 |
| 厂家       | `GET/POST /api/vendors` 等          | 厂家 CRUD（列表支持分页） |
| 任务       | `GET /api/vendors/:id/tasks` `POST /api/tasks` `POST /api/tasks/:id/toggle` `POST /api/tasks/:id/execute` | 任务管理/启停/执行 |
| 任务运行状态 | `GET /api/tasks/running`           | 返回当前正在执行的任务 ID 列表（用于前端防重复点击） |
| FTP 账号   | `GET/POST /api/ftp-accounts` 等     | FTP/SFTP 账号 CRUD   |
| 系统配置   | `GET/POST /api/configs`             | 配置读写             |
| 执行日志   | `GET /api/logs` `DELETE /api/logs/:id` `DELETE /api/logs` | 日志查询/删除/清空（列表支持分页） |
| 文件       | `GET /api/files/output` `GET /api/files/download` `GET /api/files/backup` `POST /api/files/clean-backups` | 文件列表（支持分页）/下载/备份/清理 |

### 分页说明

厂家列表、文件列表（output / backup）、执行日志等列表接口均支持分页，请求参数：

| 参数        | 默认值 | 说明                 |
| ----------- | ------ | -------------------- |
| `page`      | `1`    | 页码（从 1 开始）     |
| `page_size` | `20`   | 每页条数             |

响应携带 `total` 字段表示总记录数，前端据此渲染分页器。

### 任务执行防重复

任务执行入口会在任务开始与结束时维护一个「运行中任务集合」（`sync.Map`）：
- 提交「立即执行」前，后端先检查 `IsTaskRunning(id)`，若正在执行则直接拒绝并返回 `任务正在执行中，请稍候`。
- 前端通过每 3 秒轮询 `GET /api/tasks/running` 同步运行状态，「立即执行」按钮在执行中自动禁用并显示「执行中…」，实现后端拦截 + 前端禁用的双重保险。

## 部署建议

- 使用进程管理器（systemd / supervisor / Windows 服务）托管 `data-exchange.exe`。
- 定期备份 `data.db` 与 `./backup` 目录。
- Webhook 地址含密钥，请勿提交到公开仓库。
