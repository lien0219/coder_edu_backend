-- C 语言视频资源支持第三方链接。
-- 本文件不要执行到生产库；发布时按清单人工执行。
-- 非 release 环境仍可能由 AutoMigrate 增列，不能代替本文件。
--
-- source_type:
--   upload   = 本站上传文件（历史行 DEFAULT）
--   external = 第三方 http/https 页面
-- 项目既有 migration 不使用 CHECK，本文件也不加 CHECK。
-- url 当前无索引，加长到 2048 不改索引。
--
-- 不是幂等脚本：列已存在时 ADD COLUMN 会失败。
-- 执行前核：
--   SHOW COLUMNS FROM resources LIKE 'source_type';
--   SHOW COLUMNS FROM resources LIKE 'url';
--   SHOW INDEX FROM resources;

ALTER TABLE resources
  ADD COLUMN source_type VARCHAR(20) NOT NULL DEFAULT 'upload' AFTER url;

ALTER TABLE resources
  MODIFY COLUMN url VARCHAR(2048) NOT NULL;
