-- 数据交换系统 建表与初始数据 SQL（由系统生成，供手动导入）
-- 方言: mysql
-- 用途: 将 auto_migrate 设为 false 后，由 DBA/运维手动执行本文件完成建表与初始化

CREATE TABLE `constants` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `key` varchar(191) NOT NULL,
  `value` varchar(191) NOT NULL,
  `description` varchar(191),
  `created_at` datetime,
  `updated_at` datetime
);

CREATE TABLE `db_connections` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `db_type` varchar(16) NOT NULL DEFAULT 'mysql',
  `host` varchar(255) NOT NULL,
  `port` int NOT NULL DEFAULT 3306,
  `username` varchar(255) NOT NULL,
  `password` varchar(255) NOT NULL,
  `database_name` varchar(255) NOT NULL,
  `extra_params` text,
  `enabled` int DEFAULT 1,
  `created_at` datetime,
  `updated_at` datetime
);

CREATE TABLE `vendors` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `code` varchar(191) NOT NULL,
  `description` varchar(512),
  `enabled` int DEFAULT 1,
  `created_at` datetime,
  `updated_at` datetime
);

CREATE TABLE `ftp_accounts` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `vendor_id` int NOT NULL,
  `name` varchar(255) NOT NULL,
  `protocol` varchar(16) NOT NULL DEFAULT 'sftp',
  `host` varchar(255) NOT NULL,
  `port` int NOT NULL DEFAULT 22,
  `username` varchar(255) NOT NULL,
  `password` varchar(255) NOT NULL,
  `remote_path` varchar(255) DEFAULT '/',
  `enabled` int DEFAULT 1,
  `created_at` datetime,
  `updated_at` datetime
);

CREATE TABLE `sql_tasks` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `vendor_id` int NOT NULL,
  `db_connection_id` int,
  `task_name` varchar(255) NOT NULL,
  `sql_content` longtext NOT NULL,
  `csv_filename_template` varchar(255) DEFAULT '{vendor_code}_{task_name}_{date}.csv',
  `cron_expression` varchar(64) DEFAULT '0 2 * * *',
  `execution_mode` varchar(32) DEFAULT 'export_only',
  `ftp_account_id` int,
  `target_db_connection_id` int,
  `target_table_name` varchar(191),
  `field_mapping` longtext,
  `import_mode` varchar(32) DEFAULT 'append',
  `sort_order` int DEFAULT 0,
  `enabled` int DEFAULT 1,
  `last_run_at` datetime,
  `last_status` varchar(32),
  `created_at` datetime,
  `updated_at` datetime
);

CREATE TABLE `system_configs` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `config_key` varchar(191) NOT NULL,
  `config_value` varchar(191) NOT NULL,
  `description` varchar(191),
  `updated_at` datetime
);

CREATE TABLE `export_logs` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `task_id` int NOT NULL,
  `vendor_id` int NOT NULL,
  `status` varchar(32) NOT NULL,
  `execution_mode` varchar(32),
  `csv_filename` varchar(255),
  `file_size` int DEFAULT 0,
  `record_count` int DEFAULT 0,
  `error_message` longtext,
  `duration_ms` int DEFAULT 0,
  `started_at` datetime,
  `finished_at` datetime,
  `created_at` datetime
);

CREATE TABLE `users` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `username` varchar(191) NOT NULL,
  `password` varchar(191) NOT NULL,
  `nickname` varchar(191),
  `role` varchar(32) DEFAULT 'admin',
  `status` int DEFAULT 1,
  `created_at` datetime,
  `updated_at` datetime
);

CREATE TABLE `operation_logs` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `user_id` int,
  `username` varchar(128),
  `action` varchar(128),
  `module` varchar(64),
  `method` varchar(16),
  `path` varchar(255),
  `detail` longtext,
  `ip` varchar(64),
  `status` int DEFAULT 0,
  `success` int DEFAULT 1,
  `duration_ms` int DEFAULT 0,
  `created_at` datetime
);

CREATE TABLE `sql_task_histories` (
  `id` bigint PRIMARY KEY AUTO_INCREMENT,
  `task_id` int NOT NULL,
  `task_name` varchar(191),
  `sql_content` longtext NOT NULL,
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
INSERT INTO `users` (`username`, `password`, `nickname`, `role`) VALUES ('admin', '$2a$10$PpUhtBCSpFVqzdqKPoeYAOwQ434/pzyv6Ji3B56nUjG8Vpc4CurI.', '管理员', 'admin');
