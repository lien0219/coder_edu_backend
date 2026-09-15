-- 学习路径完成记录：同一用户同一资料只能完成一次
-- 本文件不要执行，也不要通过应用入口 AutoMigrate 代替本检查。
--
-- 目标：learning_path_completions (user_id, material_id) 联合唯一。
-- user_id 对应 users.id（BIGINT UNSIGNED），material_id 对应
-- learning_path_materials.id（varchar(36)）。
-- 本表无软删除列；唯一索引覆盖全部行。
--
-- 执行前置：
--   1) 已有 learning_path_completions、users。
--   2) 先跑下面只读检查。若有重复，先按处理方案人工决定保留行，
--      再执行 CREATE UNIQUE INDEX。重复仍在时加索引会失败，这是有意的。
--   3) 非 release AutoMigrate 会按新 model 尝试补唯一索引，但不会跑
--      本文件的检查 SQL。有重复时 AutoMigrate 同样会失败。
--
-- 可重复性：CREATE UNIQUE INDEX 在索引已存在时会报错，本文件不可重复执行。

-- 只读检查（不改数据）：
SELECT user_id,
       material_id,
       COUNT(*) AS completion_count,
       GROUP_CONCAT(id ORDER BY id) AS completion_ids,
       MIN(id) AS suggested_keep_id,
       MIN(completed_at) AS first_completed_at,
       MAX(completed_at) AS last_completed_at
FROM learning_path_completions
GROUP BY user_id, material_id
HAVING COUNT(*) > 1;

-- 处理方案（不要在本文件自动执行删除或扣 XP）：
--   每一对 (user_id, material_id) 建议保留 MIN(id) 对应的那一行。
--   其余行是历史重复完成，可能已经各自加过 users.xp。
--   本迁移不删除这些行，也不回扣 XP；是否清理由运维按备份和审计决定。
--   在重复清掉之前不要创建唯一索引。
--
-- 部分失败后检查：
--   SHOW INDEX FROM learning_path_completions
--     WHERE Key_name = 'idx_learning_path_completion_user_material';
--   重复检查 SQL 是否仍有 HAVING COUNT(*) > 1 的行。

CREATE UNIQUE INDEX idx_learning_path_completion_user_material
  ON learning_path_completions (user_id, material_id);
