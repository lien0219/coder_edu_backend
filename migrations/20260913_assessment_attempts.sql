-- 学前测验：历史尝试、幂等与评分状态
-- 本轮不要执行本文件，也不要通过应用入口 AutoMigrate/seed。
--
-- MySQL 版本：
--   ROW_NUMBER() 窗口函数需要 MySQL 8.0+ / MariaDB 10.2+。
--   5.7 不能跑本脚本的回填段，需改成会话变量编号后再加唯一索引。
--
-- 现有约束检查（代码与模型）：
--   assessment_submissions 原先没有 (user_id, assessment_id) 唯一约束。
--   旧逻辑靠覆盖同一行维持“每人一卷一行”，并发下可能已经产生重复行。
--   上线前应先查出重复：
--     SELECT user_id, assessment_id, COUNT(*) c
--     FROM assessment_submissions WHERE deleted_at IS NULL
--     GROUP BY user_id, assessment_id HAVING c > 1;
--   重复时建议保留最新 id、其余软删。即便不删，ROW_NUMBER 会把 attempt_no
--   编成 1,2,3，(user_id, assessment_id, attempt_no) 仍可唯一。
--   但 (user_id, client_request_id) 在回填前若多行 client_request_id 同为
--   空字符串，加唯一索引会直接失败；必须先回填 CONCAT('legacy-', id)。
--
-- 旧后端 AutoMigrate（非 release）对新结构的影响：
--   GORM AutoMigrate 会按新 model 补列和唯一索引，但不会跑本文件的回填 SQL。
--   若先启动新二进制、未先回填：attempt_no 默认全是 1，同一学生同一卷两行
--   会让 idx_assessment_attempt 创建失败；client_request_id 默认为 ''，
--   同一学生两行会让 idx_assessment_submit_idempotent 失败。
--   旧二进制读到新列会忽略；再写入时仍按覆盖逻辑，可能撞新唯一索引。
--   正确顺序：备份 → 查重复 → 执行本 SQL（或等价回填后再 AutoMigrate）→
--   再发新前后端。不要让旧版应用在半迁移库上自动建唯一索引。
--   MySQL 唯一索引包含软删行。应用侧 MAX(attempt_no) 必须 Unscoped，
--   删除后重测使用下一编号，不得复用已占用的 attempt_no。
--
-- 兼容与发布顺序：
--   1) 先清理/回填答卷，再执行本迁移或允许非 release AutoMigrate。
--   2) 前后端必须同批发布。旧前端 GET 期望 data 为题目数组，新后端返回
--      {assessmentId, paperVersion, questions}；旧提交缺少 paperVersion/
--      clientRequestId 会被 400。
--   3) 学生 GET 只领取 is_published=1 且含未删除题目的试卷。教师端此前几乎
--      不发布，上线后若未发布会返回「暂无测验」。
--   4) 有效诊断必须匹配当前试卷 paper_version；同卷改题后旧确认不沿用。
--   5) 本阶段不按总分自动赋 1–4 级，不改知识点映射与推荐引擎。

ALTER TABLE assessment_submissions
  ADD COLUMN attempt_no INT NOT NULL DEFAULT 1 AFTER assessment_id,
  ADD COLUMN client_request_id VARCHAR(64) NULL AFTER attempt_no,
  ADD COLUMN paper_version VARCHAR(64) NULL AFTER client_request_id,
  ADD COLUMN item_results JSON NULL AFTER answers,
  ADD COLUMN auto_score INT NOT NULL DEFAULT 0 AFTER total_score,
  ADD COLUMN objective_max INT NOT NULL DEFAULT 0 AFTER auto_score,
  ADD COLUMN pending_manual_count INT NOT NULL DEFAULT 0 AFTER objective_max,
  ADD COLUMN scoring_status VARCHAR(32) NOT NULL DEFAULT 'awaiting_teacher' AFTER pending_manual_count;

-- 为历史行编号：同一学生同一试卷按时间/id 递增。若已有重复，先清理再执行。
UPDATE assessment_submissions AS s
JOIN (
  SELECT id,
         ROW_NUMBER() OVER (PARTITION BY user_id, assessment_id ORDER BY created_at ASC, id ASC) AS n
  FROM assessment_submissions
  WHERE deleted_at IS NULL
) AS numbered ON numbered.id = s.id
SET s.attempt_no = numbered.n
WHERE s.deleted_at IS NULL;

UPDATE assessment_submissions
SET client_request_id = CONCAT('legacy-', id)
WHERE (client_request_id IS NULL OR client_request_id = '')
  AND deleted_at IS NULL;

UPDATE assessment_submissions
SET auto_score = total_score
WHERE deleted_at IS NULL AND auto_score = 0 AND total_score > 0;

CREATE UNIQUE INDEX idx_assessment_attempt
  ON assessment_submissions (user_id, assessment_id, attempt_no);

CREATE UNIQUE INDEX idx_assessment_submit_idempotent
  ON assessment_submissions (user_id, client_request_id);
