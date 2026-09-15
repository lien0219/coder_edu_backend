-- 学前题 / 学习资料 与 KnowledgePoint 的多对多关联
-- 本文件不要执行，也不要加入应用入口 AutoMigrate。
--
-- 知识主键使用 knowledge_points.id（UUID / varchar(36)）。
-- assessment_question_id 对应 assessment_questions.id（GORM uint → MySQL BIGINT UNSIGNED）。
-- material_id 对应 learning_path_materials.id（varchar(36)）。
-- 不改 knowledge_tags / 关卡体系。Tags 字符串不得当作映射。
--
-- 执行顺序：
--   1) 业务库已有 knowledge_points、assessment_questions、learning_path_materials
--      （通常由既有 AutoMigrate 创建）。
--   2) 若尚未执行 20260913_assessment_attempts.sql，可先或后执行本文件；
--      本表不依赖 attempt_no / item_results 列。推荐先备份，再按发布清单执行。
--   3) 执行前核对 SHOW CREATE TABLE knowledge_points / assessment_questions /
--      learning_path_materials 的 id 类型与字符集。
--
-- 字符集：未指定 COLLATE，沿用实例/库默认。配置示例 charset 为 utf8mb4。
-- 若现有表使用 utf8mb4_unicode_ci 或 utf8mb4_0900_ai_ci，执行前应改成一致，
-- 避免关联列比较时 collation mismatch。
--
-- 可重复性：CREATE TABLE IF NOT EXISTS 在表已存在时直接跳过，
-- 不会校验列类型、主键或索引是否与本文件一致，因此不能当作幂等迁移。
-- 部分失败后检查：
--   SHOW CREATE TABLE assessment_question_knowledge_points;
--   SHOW CREATE TABLE learning_path_material_knowledge_points;
--   SELECT COUNT(*) FROM information_schema.statistics
--     WHERE table_schema = DATABASE()
--       AND table_name IN (
--         'assessment_question_knowledge_points',
--         'learning_path_material_knowledge_points'
--       );
-- 表已存在但结构不对时，需要人工对比后另写 ALTER，不要重跑本文件指望修正。

CREATE TABLE IF NOT EXISTS assessment_question_knowledge_points (
  assessment_question_id BIGINT UNSIGNED NOT NULL,
  knowledge_point_id VARCHAR(36) NOT NULL,
  PRIMARY KEY (assessment_question_id, knowledge_point_id),
  KEY idx_aq_kp_point (knowledge_point_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS learning_path_material_knowledge_points (
  material_id VARCHAR(36) NOT NULL,
  knowledge_point_id VARCHAR(36) NOT NULL,
  PRIMARY KEY (material_id, knowledge_point_id),
  KEY idx_lpm_kp_point (knowledge_point_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
