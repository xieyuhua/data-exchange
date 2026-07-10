-- 数据交换系统 建表与初始数据 SQL（由系统生成，供手动导入）
-- 方言: sqlite
-- 用途: 将 auto_migrate 设为 false 后，由 DBA/运维手动执行本文件完成建表与初始化

CREATE TABLE `constants` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `key` varchar(191) NOT NULL,
  `value` text NOT NULL,
  `description` text,
  `created_at` datetime,
  `updated_at` datetime
);

CREATE TABLE `db_connections` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` varchar(255) NOT NULL,
  `db_type` varchar(16) NOT NULL DEFAULT 'mysql',
  `host` varchar(255) NOT NULL,
  `port` integer NOT NULL DEFAULT 3306,
  `username` varchar(255) NOT NULL,
  `password` varchar(255) NOT NULL,
  `database_name` varchar(255) NOT NULL,
  `extra_params` text,
  `enabled` integer DEFAULT 1,
  `created_at` datetime,
  `updated_at` datetime
);

CREATE TABLE `vendors` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` varchar(255) NOT NULL,
  `code` varchar(191) NOT NULL,
  `description` varchar(512),
  `enabled` integer DEFAULT 1,
  `created_at` datetime,
  `updated_at` datetime
);

CREATE TABLE `ftp_accounts` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `vendor_id` integer NOT NULL,
  `name` varchar(255) NOT NULL,
  `protocol` varchar(16) NOT NULL DEFAULT 'sftp',
  `host` varchar(255) NOT NULL,
  `port` integer NOT NULL DEFAULT 22,
  `username` varchar(255) NOT NULL,
  `password` varchar(255) NOT NULL,
  `remote_path` varchar(255) DEFAULT '/',
  `enabled` integer DEFAULT 1,
  `created_at` datetime,
  `updated_at` datetime
);

CREATE TABLE `sql_tasks` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `vendor_id` integer NOT NULL,
  `db_connection_id` integer,
  `task_name` varchar(255) NOT NULL,
  `sql_content` text NOT NULL,
  `csv_filename_template` varchar(255) DEFAULT '{vendor_code}_{task_name}_{date}.csv',
  `cron_expression` varchar(64) DEFAULT '0 2 * * *',
  `execution_mode` varchar(32) DEFAULT 'export_only',
  `ftp_account_id` integer,
  `target_db_connection_id` integer,
  `target_table_name` varchar(191),
  `field_mapping` text,
  `import_mode` varchar(32) DEFAULT 'append',
  `sort_order` integer DEFAULT 0,
  `enabled` integer DEFAULT 1,
  `last_run_at` datetime,
  `last_status` varchar(32),
  `created_at` datetime,
  `updated_at` datetime
);

CREATE TABLE `system_configs` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `config_key` varchar(191) NOT NULL,
  `config_value` text NOT NULL,
  `description` text,
  `updated_at` datetime
);

CREATE TABLE `export_logs` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `task_id` integer NOT NULL,
  `vendor_id` integer NOT NULL,
  `status` varchar(32) NOT NULL,
  `execution_mode` varchar(32),
  `csv_filename` varchar(255),
  `file_size` integer DEFAULT 0,
  `record_count` integer DEFAULT 0,
  `error_message` text,
  `duration_ms` integer DEFAULT 0,
  `started_at` datetime,
  `finished_at` datetime,
  `created_at` datetime
);

CREATE TABLE `users` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `username` varchar(191) NOT NULL,
  `password` text NOT NULL,
  `nickname` text,
  `role` varchar(32) DEFAULT 'admin',
  `status` integer DEFAULT 1,
  `created_at` datetime,
  `updated_at` datetime
);

CREATE TABLE `operation_logs` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` integer,
  `username` varchar(128),
  `action` varchar(128),
  `module` varchar(64),
  `method` varchar(16),
  `path` varchar(255),
  `detail` text,
  `ip` varchar(64),
  `status` integer DEFAULT 0,
  `success` integer DEFAULT 1,
  `duration_ms` integer DEFAULT 0,
  `created_at` datetime
);

CREATE TABLE `sql_task_histories` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `task_id` integer NOT NULL,
  `task_name` text,
  `sql_content` text NOT NULL,
  `changed_by` varchar(128),
  `remark` varchar(255),
  `created_at` datetime
);

-- ==================== 初始数据 ====================

-- 默认系统配置 (system_configs)
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES ('backup_keep_count', '30', '保留备份文件的最大数量，超过则自动清理最旧的');
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES ('csv_output_dir', './output', 'CSV 导出文件存放目录');
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES ('backup_dir', './backup', '文件备份目录');
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES ('csv_delimiter', ',', 'CSV 字段分隔符，默认逗号');
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES ('csv_charset', 'UTF-8', 'CSV 文件字符集');
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES ('csv_bom', 'true', '是否在 CSV 开头写入 UTF-8 BOM (true/false)');
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES ('date_format', '20060102', '文件名中的日期格式');
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES ('datetime_format', '20060102_150405', '文件名中的日期时间格式');
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES ('max_parallel_tasks', '3', '最大并行任务数');
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES ('page_size', '20', '列表每页显示条数（厂家/日志/文件列表），修改后对新打开的列表生效');
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES ('notify_ding_enabled', 'off', '钉钉失败提醒开关: on 开启 / off 关闭');
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES ('notify_ding_webhook', '', '钉钉机器人 Webhook 地址 (含 access_token)');
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES ('notify_ding_secret', '', '钉钉机器人加签密钥 (安全设置选择加签时填写，可空)');
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES ('notify_wx_enabled', 'off', '企业微信失败提醒开关: on 开启 / off 关闭');
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES ('notify_wx_webhook', '', '企业微信群机器人 Webhook 地址 (含 key)');
INSERT INTO `system_configs` (`config_key`, `config_value`, `description`) VALUES ('notify_at', '', '失败提醒 @ 的成员手机号/userid，逗号分隔，@all 表示所有人');

-- 默认管理员账号 admin / admin2026 (users)
INSERT INTO `users` (`username`, `password`, `nickname`, `role`) VALUES ('admin', '$2a$10$eiq/YDt4GpO6k5w4rUQLY.S0clchNCckIox82E.JLOr8yIsKRCJla', '管理员', 'admin');
