-- 知识点教学视频支持本地上传与外部播放链接并存。
-- 本文件不要执行到生产库；发布时按清单人工执行。
-- 非 release 环境仍可能由 AutoMigrate 增列，不能代替本文件。
--
-- source_type:
--   upload   = 本站上传文件（缺省/历史行 DEFAULT）
--   external = 第三方 http/https 页面
-- 项目既有 migration 不使用 CHECK，本文件也不加 CHECK。
-- url 当前无索引，加长到 2048 不改索引。
--
-- 不是幂等脚本：列已存在时 ADD COLUMN 会失败。
-- 执行前核：
--   SHOW COLUMNS FROM knowledge_point_videos LIKE 'source_type';
--   SHOW COLUMNS FROM knowledge_point_videos LIKE 'url';
--   SHOW INDEX FROM knowledge_point_videos;

ALTER TABLE knowledge_point_videos
  ADD COLUMN source_type VARCHAR(20) NOT NULL DEFAULT 'upload' AFTER url;

ALTER TABLE knowledge_point_videos
  MODIFY COLUMN url VARCHAR(2048) NOT NULL;
