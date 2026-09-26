-- 网站自主测评题库初始化（DL-C56-v1 + SDL-C20-v1）。
-- 本文件是草案，阶段1不要执行。不要 AutoMigrate 代替本文件。
-- 前置：已执行 20260923_learning_profile.sql。
-- 题干与 internal/learningprofile/catalog.go 保持一致；不含基本信息题与 EFA/CFA 附录。
-- 不写入任何学生作答。正式作答冻结靠 administrations.item_snapshot。
--
-- 事务边界（务必与 DDL 分开）：
--   MySQL CREATE TABLE 会隐式提交，不能与本文件的 INSERT 组成一个可回滚事务。
--   本文件用存储过程包住全部 DML：SQLEXCEPTION 时 ROLLBACK 再 RESIGNAL，避免
--   mysql 客户端在中途报错后仍执行到 COMMIT。
--   请用 mysql 客户端执行（需要 DELIMITER）。建议加上 --abort-source-on-error。
--   DROP/CREATE PROCEDURE 本身是 DDL，在 CALL 之前就会提交，不影响题库事务。
--
-- 执行语义：
--   1. 空目录（两份量表均不存在）：插入 2 份量表 + 76 题后 COMMIT。
--   2. 已有完全一致的 2 份量表 + 76 题：不写行，COMMIT（可安全重跑）。
--   3. 同 code 但题干 / 选项 / 维度 / 版本等不一致，或存在额外题项：报错并停止，不替换。
--   4. 部分插入（行数既非 0 也非完整一致）：报错并停止，避免重跑后状态更含糊。
--   5. 已有这些量表的正式 submitted / answers / dimension_scores：除非目录已完全一致（走 2），
--      否则报错；本文件没有任何 DELETE / UPDATE / INSERT IGNORE / REPLACE。
--
-- 失败后现场核对（只读）：
--   SELECT code, name, version, item_count FROM learning_profile_instruments
--     WHERE code IN ('DL-C56-v1','SDL-C20-v1');
--   SELECT i.code, COUNT(*) FROM learning_profile_items it
--     JOIN learning_profile_instruments i ON i.id = it.instrument_id
--     WHERE i.code IN ('DL-C56-v1','SDL-C20-v1') GROUP BY i.code;
--   SELECT status, COUNT(*) FROM learning_profile_administrations a
--     JOIN learning_profile_instruments i ON i.id = a.instrument_id
--     WHERE i.code IN ('DL-C56-v1','SDL-C20-v1') GROUP BY status;
--   SELECT COUNT(*) FROM learning_profile_answers;
--
-- 人工恢复（仅当核对确认无任何 administrations / answers / dimension_scores 之后）：
--   同一会话先 ROLLBACK;
--   再经单独批准执行：
--     DELETE FROM learning_profile_items
--       WHERE instrument_id IN (
--         SELECT id FROM learning_profile_instruments
--         WHERE code IN ('DL-C56-v1','SDL-C20-v1'));
--     DELETE FROM learning_profile_instruments
--       WHERE code IN ('DL-C56-v1','SDL-C20-v1');
--   然后重跑本文件。已有正式作答时禁止上述 DELETE。
--
-- 字符集：题干与选项按 UTF-8 保存。请使用
--   mysql --default-character-set=utf8mb4
-- 执行本文件，不要依赖客户端默认字符集。下面的 SET NAMES 把本会话的
-- character_set_client / character_set_connection / character_set_results 设为 utf8mb4。

SET NAMES utf8mb4;

DROP PROCEDURE IF EXISTS seed_learning_profile_official_v1;

DELIMITER $$
CREATE PROCEDURE seed_learning_profile_official_v1()
BEGIN
  DECLARE EXIT HANDLER FOR SQLEXCEPTION
  BEGIN
    ROLLBACK;
    RESIGNAL;
  END;

START TRANSACTION;

DROP TEMPORARY TABLE IF EXISTS lp_seed_guard;
DROP TEMPORARY TABLE IF EXISTS expected_items;
DROP TEMPORARY TABLE IF EXISTS expected_instruments;

CREATE TEMPORARY TABLE lp_seed_guard (
  step VARCHAR(64) NOT NULL PRIMARY KEY,
  allowed TINYINT NOT NULL,
  CONSTRAINT ck_lp_seed_allowed CHECK (allowed = 1)
) ENGINE=InnoDB;

CREATE TEMPORARY TABLE expected_instruments (
  code VARCHAR(64) NOT NULL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  version VARCHAR(32) NOT NULL,
  program VARCHAR(64) NOT NULL,
  scale_min INT NOT NULL,
  scale_max INT NOT NULL,
  item_count INT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO expected_instruments
  (code, name, version, program, scale_min, scale_max, item_count)
VALUES
  ('DL-C56-v1', 'C语言深度学习能力调查问卷', 'v1', 'platform_self_assessment', 1, 5, 56),
  ('SDL-C20-v1', '自我导向学习能力量表', 'v1', 'platform_self_assessment', 1, 7, 20);

CREATE TEMPORARY TABLE expected_items (
  instrument_code VARCHAR(64) NOT NULL,
  item_code VARCHAR(32) NOT NULL,
  source_item_no VARCHAR(32) NOT NULL,
  stem TEXT NOT NULL,
  primary_dimension VARCHAR(64) NOT NULL,
  secondary_dimension VARCHAR(64) NOT NULL,
  scale_min INT NOT NULL,
  scale_max INT NOT NULL,
  scoring_direction VARCHAR(16) NOT NULL,
  sort_order INT NOT NULL,
  options_json JSON NOT NULL,
  PRIMARY KEY (instrument_code, item_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO expected_items
  (instrument_code, item_code, source_item_no, stem, primary_dimension, secondary_dimension, scale_min, scale_max, scoring_direction, sort_order, options_json)
SELECT v.inst, v.item_code, v.source_item_no, v.stem, v.primary_dimension, v.secondary_dimension, v.scale_min, v.scale_max, v.scoring_direction, v.sort_order, v.options_json
FROM (
  SELECT 'DL-C56-v1' AS inst, 'CT1' AS item_code, 'CT1' AS source_item_no, '我会认真听取他人对我所写C语言程序的意见，即使这些意见和我的想法不同。' AS stem, 'critical_thinking_disposition' AS primary_dimension, '认知成熟' AS secondary_dimension, 1 AS scale_min, 5 AS scale_max, 'forward' AS scoring_direction, 1 AS sort_order, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON) AS options_json
  UNION ALL SELECT 'DL-C56-v1', 'CT2', 'CT2', '当出现与我原有判断不一致的新信息（例如调试结果）时，我愿意改变自己对该C语言问题的看法。', 'critical_thinking_disposition', '认知成熟', 1, 5, 'forward', 2, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT3', 'CT3', '判断一个C语言问题时，我会尽量依据事实做出判断，而不是凭个人偏见。', 'critical_thinking_disposition', '认知成熟', 1, 5, 'forward', 3, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT4', 'CT4', '即使同学的编程思路和我不同，我也能够和他们保持良好的合作关系。', 'critical_thinking_disposition', '认知成熟', 1, 5, 'forward', 4, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT5', 'CT5', '我会反思自己固有的编程习惯是否影响了我对某个C语言问题的判断。', 'critical_thinking_disposition', '认知成熟', 1, 5, 'forward', 5, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT6', 'CT6', '面对一个C语言问题，我会尝试找出不止一种解法。', 'critical_thinking_disposition', '认知成熟', 1, 5, 'forward', 6, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT7', 'CT7', '在做编程方案决策时，我会提出很多问题。', 'critical_thinking_disposition', '认知成熟', 1, 5, 'forward', 7, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT8', 'CT8', '我相信大多数C语言问题都存在不止一种解法。', 'critical_thinking_disposition', '认知成熟', 1, 5, 'forward', 8, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT9', 'CT9', '我会主动寻找运用C语言解决实际问题的机会。', 'critical_thinking_disposition', '投入性', 1, 5, 'forward', 9, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT10', 'CT10', '我对多种类型的C语言编程问题（如数据结构、算法、文件处理等）感兴趣。', 'critical_thinking_disposition', '投入性', 1, 5, 'forward', 10, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT11', 'CT11', '我能够将所学的C语言知识和多种不同类型的问题联系起来。', 'critical_thinking_disposition', '投入性', 1, 5, 'forward', 11, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT12', 'CT12', '我喜欢寻找有挑战性的C语言问题的答案。', 'critical_thinking_disposition', '投入性', 1, 5, 'forward', 12, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT13', 'CT13', '我认为自己是一个善于解决C语言问题的人。', 'critical_thinking_disposition', '投入性', 1, 5, 'forward', 13, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT14', 'CT14', '我有信心针对C语言问题得出合理的结论。', 'critical_thinking_disposition', '投入性', 1, 5, 'forward', 14, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT15', 'CT15', '我能够把所学的C语言知识应用到多种不同的任务中。', 'critical_thinking_disposition', '投入性', 1, 5, 'forward', 15, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT16', 'CT16', '我能够清楚地解释自己解决C语言问题的思路。', 'critical_thinking_disposition', '投入性', 1, 5, 'forward', 16, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT17', 'CT17', '在澄清一个C语言解决方案时，我会提出有价值的问题。', 'critical_thinking_disposition', '投入性', 1, 5, 'forward', 17, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT18', 'CT18', '我能够清晰准确地描述一个C语言问题。', 'critical_thinking_disposition', '投入性', 1, 5, 'forward', 18, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT19', 'CT19', '面对一个C语言编程任务，我会坚持到底，直到把它解决好。', 'critical_thinking_disposition', '投入性', 1, 5, 'forward', 19, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT20', 'CT20', '我喜欢学习C语言中的很多不同知识点。', 'critical_thinking_disposition', '创新性', 1, 5, 'forward', 20, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT21', 'CT21', '在学习C语言的过程中，我会提出很多问题。', 'critical_thinking_disposition', '创新性', 1, 5, 'forward', 21, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT22', 'CT22', '我认为持续掌握最新的编程知识很重要。', 'critical_thinking_disposition', '创新性', 1, 5, 'forward', 22, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT23', 'CT23', '我喜欢解决C语言编程问题。', 'critical_thinking_disposition', '创新性', 1, 5, 'forward', 23, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT24', 'CT24', '即使不是在上课，我也喜欢学习C语言相关的知识。', 'critical_thinking_disposition', '创新性', 1, 5, 'forward', 24, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT25', 'CT25', '即使结果可能不理想，我也愿意弄清楚一个C语言问题背后的真正原因。', 'critical_thinking_disposition', '创新性', 1, 5, 'forward', 25, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'CT26', 'CT26', '为了找到一个C语言问题的正确答案，我愿意额外花费时间和精力。', 'critical_thinking_disposition', '创新性', 1, 5, 'forward', 26, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
) v;

INSERT INTO expected_items
  (instrument_code, item_code, source_item_no, stem, primary_dimension, secondary_dimension, scale_min, scale_max, scoring_direction, sort_order, options_json)
SELECT v.inst, v.item_code, v.source_item_no, v.stem, v.primary_dimension, v.secondary_dimension, v.scale_min, v.scale_max, v.scoring_direction, v.sort_order, v.options_json
FROM (
  SELECT 'DL-C56-v1' AS inst, 'PS1' AS item_code, 'PS1' AS source_item_no, '面对C语言编程任务时，我认为自己能够明确需要解决的核心问题。' AS stem, 'problem_solving_ability' AS primary_dimension, '问题界定' AS secondary_dimension, 1 AS scale_min, 5 AS scale_max, 'forward' AS scoring_direction, 27 AS sort_order, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON) AS options_json
  UNION ALL SELECT 'DL-C56-v1', 'PS2', 'PS2', '面对C语言编程任务时，我认为自己能够确定编程任务的输入信息。', 'problem_solving_ability', '问题界定', 1, 5, 'forward', 28, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'PS3', 'PS3', '面对C语言编程任务时，我认为自己能够确定程序需要产生的预期结果。', 'problem_solving_ability', '问题界定', 1, 5, 'forward', 29, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'PS4', 'PS4', '面对C语言编程问题时，我认为自己能够识别一种可能的解决途径。', 'problem_solving_ability', '策略识别', 1, 5, 'forward', 30, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'PS5', 'PS5', '当常用的解决途径不适用时，我认为自己能够识别其他可能的解决途径。', 'problem_solving_ability', '策略识别', 1, 5, 'forward', 31, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'PS6', 'PS6', '面对同一个C语言编程问题时，我认为自己能够识别不同的解决途径。', 'problem_solving_ability', '策略识别', 1, 5, 'forward', 32, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'PS7', 'PS7', '我认为自己能够针对C语言编程问题提出具体的解决方案。', 'problem_solving_ability', '方案提出', 1, 5, 'forward', 33, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'PS8', 'PS8', '我认为自己能够将解决思路组织为有序的实现步骤。', 'problem_solving_ability', '方案提出', 1, 5, 'forward', 34, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'PS9', 'PS9', '我认为自己能够根据任务要求确定解决方案所需的程序结构。', 'problem_solving_ability', '方案提出', 1, 5, 'forward', 35, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'PS10', 'PS10', '我认为自己能够判断一个解决方案的逻辑是否合理。', 'problem_solving_ability', '方案评价', 1, 5, 'forward', 36, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'PS11', 'PS11', '我认为自己能够判断一个解决方案是否可以在当前条件下实施。', 'problem_solving_ability', '方案评价', 1, 5, 'forward', 37, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'PS12', 'PS12', '我认为自己能够识别一个解决方案不适用的情形。', 'problem_solving_ability', '方案评价', 1, 5, 'forward', 38, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'PS13', 'PS13', '我认为自己能够将解决方案中的步骤编写为相应的C语言代码。', 'problem_solving_ability', '方案实施', 1, 5, 'forward', 39, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'PS14', 'PS14', '我认为自己能够将分别编写的代码部分组合成一个完整的程序。', 'problem_solving_ability', '方案实施', 1, 5, 'forward', 40, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'PS15', 'PS15', '当代码实现偏离原定方案时，我认为自己能够调整代码。', 'problem_solving_ability', '方案实施', 1, 5, 'forward', 41, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'PS16', 'PS16', '程序运行后，我认为自己能够判断输出结果是否符合任务目标。', 'problem_solving_ability', '结果评价', 1, 5, 'forward', 42, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'PS17', 'PS17', '我认为自己能够识别程序输出结果与预期结果之间的差异。', 'problem_solving_ability', '结果评价', 1, 5, 'forward', 43, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'PS18', 'PS18', '根据程序运行结果，我认为自己能够判断解决方案是否需要修改。', 'problem_solving_ability', '结果评价', 1, 5, 'forward', 44, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'KT1', 'KT1', '面对与课堂例题相近的编程任务，我认为自己能够运用已经学过的C语言知识。', 'knowledge_transfer_ability', '近迁移', 1, 5, 'forward', 45, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'KT2', 'KT2', '面对与课堂示例有所不同的同类编程任务，我认为自己能够运用学过的方法完成。', 'knowledge_transfer_ability', '近迁移', 1, 5, 'forward', 46, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'KT3', 'KT3', '面对解题结构相似的编程任务，我认为自己能够采用以前学过的解题方法。', 'knowledge_transfer_ability', '近迁移', 1, 5, 'forward', 47, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'KT4', 'KT4', '面对新的编程任务，我认为自己能够识别它与以往任务之间的关键差异。', 'knowledge_transfer_ability', '适应性迁移', 1, 5, 'forward', 48, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'KT5', 'KT5', '面对新的编程任务，我认为自己能够从已经学过的方法中选择可供借鉴的方法。', 'knowledge_transfer_ability', '适应性迁移', 1, 5, 'forward', 49, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'KT6', 'KT6', '我认为自己能够根据新任务的要求调整所选择的方法。', 'knowledge_transfer_ability', '适应性迁移', 1, 5, 'forward', 50, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'KT7', 'KT7', '学习新的C语言内容时，我认为自己能够理解它与已学内容之间的联系。', 'knowledge_transfer_ability', '整合性迁移', 1, 5, 'forward', 51, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'KT8', 'KT8', '面对综合性编程任务时，我认为自己能够判断需要运用哪些已经学过的知识。', 'knowledge_transfer_ability', '整合性迁移', 1, 5, 'forward', 52, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'KT9', 'KT9', '我认为自己能够在同一个程序中综合运用不同课程单元的知识。', 'knowledge_transfer_ability', '整合性迁移', 1, 5, 'forward', 53, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'KT10', 'KT10', '完成编程任务后，我认为自己能够分析所采用的方法是否有效。', 'knowledge_transfer_ability', '反思性迁移', 1, 5, 'forward', 54, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'KT11', 'KT11', '完成编程任务后，我认为自己能够总结其中值得借鉴的做法。', 'knowledge_transfer_ability', '反思性迁移', 1, 5, 'forward', 55, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
  UNION ALL SELECT 'DL-C56-v1', 'KT12', 'KT12', '面对新的编程任务时，我认为自己能够运用以往任务中获得的经验。', 'knowledge_transfer_ability', '反思性迁移', 1, 5, 'forward', 56, CAST('[{"value":1,"label":"非常不符合"},{"value":2,"label":"比较不符合"},{"value":3,"label":"一般"},{"value":4,"label":"比较符合"},{"value":5,"label":"非常符合"}]' AS JSON)
) v;

INSERT INTO expected_items
  (instrument_code, item_code, source_item_no, stem, primary_dimension, secondary_dimension, scale_min, scale_max, scoring_direction, sort_order, options_json)
SELECT v.inst, v.item_code, v.source_item_no, v.stem, v.primary_dimension, v.secondary_dimension, v.scale_min, v.scale_max, v.scoring_direction, v.sort_order, v.options_json
FROM (
  SELECT 'SDL-C20-v1' AS inst, 'SDL1' AS item_code, '1' AS source_item_no, '我知道自己在 C 语言课程中需要学习什么。' AS stem, 'learning_motivation' AS primary_dimension, '学习动机' AS secondary_dimension, 1 AS scale_min, 7 AS scale_max, 'forward' AS scoring_direction, 1 AS sort_order, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON) AS options_json
  UNION ALL SELECT 'SDL-C20-v1', 'SDL2', '2', '无论学习结果如何，我仍然愿意继续学习 C 语言。', 'learning_motivation', '学习动机', 1, 7, 'forward', 2, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL3', '3', '我希望自己在 C 语言学习中不断进步，并取得更好的表现。', 'learning_motivation', '学习动机', 1, 7, 'forward', 3, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL4', '4', '我在 C 语言学习中的成功和失败都会激励我继续学习。', 'learning_motivation', '学习动机', 1, 7, 'forward', 4, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL5', '5', '我喜欢主动寻找 C 语言学习问题的答案。', 'learning_motivation', '学习动机', 1, 7, 'forward', 5, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL6', '6', '即使在 C 语言学习中遇到困难，我也不会轻易放弃。', 'learning_motivation', '学习动机', 1, 7, 'forward', 6, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL7', '7', '我能够主动设定自己的 C 语言学习目标。', 'planning_and_implementation', '计划与执行', 1, 7, 'forward', 7, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL8', '8', '我知道哪些学习策略适合自己实现 C 语言学习目标。', 'planning_and_implementation', '计划与执行', 1, 7, 'forward', 8, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL9', '9', '我会根据重要性和难度安排 C 语言学习任务的优先顺序。', 'planning_and_implementation', '计划与执行', 1, 7, 'forward', 9, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL10', '10', '无论在课堂学习、上机实践还是课后自主学习中，我都能够按照自己的学习计划进行学习。', 'planning_and_implementation', '计划与执行', 1, 7, 'forward', 10, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL11', '11', '我善于安排和控制自己的 C 语言学习时间。', 'planning_and_implementation', '计划与执行', 1, 7, 'forward', 11, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL12', '12', '我知道如何寻找适合 C 语言学习的资源。', 'planning_and_implementation', '计划与执行', 1, 7, 'forward', 12, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL13', '13', '我能够将新学到的 C 语言知识与已有知识或实际经验联系起来。', 'self_management', '自我管理', 1, 7, 'forward', 13, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL14', '14', '我了解自己在 C 语言学习中的优势和不足。', 'self_management', '自我管理', 1, 7, 'forward', 14, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL15', '15', '我能够监控自己的 C 语言学习进度。', 'self_management', '自我管理', 1, 7, 'forward', 15, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL16', '16', '我能够评价自己的 C 语言学习效果。', 'self_management', '自我管理', 1, 7, 'forward', 16, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL17', '17', '与教师或同学的交流能够帮助我规划后续 C 语言学习。', 'interpersonal_communication', '人际沟通', 1, 7, 'forward', 17, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL18', '18', '我愿意通过与教师和同学交流，理解不同的编程思路和学习方法。', 'interpersonal_communication', '人际沟通', 1, 7, 'forward', 18, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL19', '19', '我能够在讨论或汇报中清楚表达自己的编程思路。', 'interpersonal_communication', '人际沟通', 1, 7, 'forward', 19, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
  UNION ALL SELECT 'SDL-C20-v1', 'SDL20', '20', '我能够通过文字、代码注释或学习记录清楚表达自己的学习过程和问题。', 'interpersonal_communication', '人际沟通', 1, 7, 'forward', 20, CAST('[{"value":1,"label":"非常不同意"},{"value":2,"label":"不同意"},{"value":3,"label":"有点不同意"},{"value":4,"label":"一般 / 不确定"},{"value":5,"label":"有点同意"},{"value":6,"label":"同意"},{"value":7,"label":"非常同意"}]' AS JSON)
) v;

SET @lp_submitted := (
  SELECT COUNT(*) FROM learning_profile_administrations a
  JOIN learning_profile_instruments i ON i.id = a.instrument_id
  WHERE a.status = 'submitted' AND i.code IN ('DL-C56-v1', 'SDL-C20-v1')
);
SET @lp_answers := (
  SELECT COUNT(*) FROM learning_profile_answers ans
  JOIN learning_profile_administrations a ON a.id = ans.administration_id
  JOIN learning_profile_instruments i ON i.id = a.instrument_id
  WHERE i.code IN ('DL-C56-v1', 'SDL-C20-v1')
);
SET @lp_scores := (
  SELECT COUNT(*) FROM learning_profile_dimension_scores s
  JOIN learning_profile_administrations a ON a.id = s.administration_id
  JOIN learning_profile_instruments i ON i.id = a.instrument_id
  WHERE i.code IN ('DL-C56-v1', 'SDL-C20-v1')
);
SET @live_inst_n := (
  SELECT COUNT(*) FROM learning_profile_instruments
  WHERE code IN ('DL-C56-v1', 'SDL-C20-v1')
);
SET @live_item_n := (
  SELECT COUNT(*) FROM learning_profile_items it
  JOIN learning_profile_instruments i ON i.id = it.instrument_id
  WHERE i.code IN ('DL-C56-v1', 'SDL-C20-v1')
);
SET @expected_item_n := (SELECT COUNT(*) FROM expected_items);
SET @inst_mismatch := (
  SELECT COUNT(*) FROM learning_profile_instruments i
  JOIN expected_instruments e ON e.code = i.code
  WHERE i.name <> e.name
     OR i.version <> e.version
     OR i.program <> e.program
     OR i.scale_min <> e.scale_min
     OR i.scale_max <> e.scale_max
     OR i.item_count <> e.item_count
);
SET @item_mismatch := (
  SELECT COUNT(*) FROM learning_profile_items it
  JOIN learning_profile_instruments i ON i.id = it.instrument_id
  JOIN expected_items e ON e.instrument_code = i.code AND e.item_code = it.item_code
  WHERE it.stem <> e.stem
     OR it.source_item_no <> e.source_item_no
     OR it.primary_dimension <> e.primary_dimension
     OR it.secondary_dimension <> e.secondary_dimension
     OR it.scale_min <> e.scale_min
     OR it.scale_max <> e.scale_max
     OR it.scoring_direction <> e.scoring_direction
     OR it.sort_order <> e.sort_order
     OR CAST(it.options_json AS JSON) <> CAST(e.options_json AS JSON)
);
SET @item_extra := (
  SELECT COUNT(*) FROM learning_profile_items it
  JOIN learning_profile_instruments i ON i.id = it.instrument_id
  WHERE i.code IN ('DL-C56-v1', 'SDL-C20-v1')
    AND NOT EXISTS (
      SELECT 1 FROM expected_items e
      WHERE e.instrument_code = i.code AND e.item_code = it.item_code
    )
);
SET @catalog_empty := (@live_inst_n = 0 AND @live_item_n = 0);
SET @exact_match := (
  @live_inst_n = 2 AND @live_item_n = 76
  AND @expected_item_n = 76
  AND @inst_mismatch = 0 AND @item_mismatch = 0 AND @item_extra = 0
);
SET @content_conflict := (@inst_mismatch > 0 OR @item_mismatch > 0 OR @item_extra > 0);
SET @allow_insert := @catalog_empty;
SET @allow_noop := @exact_match;

SELECT
  @expected_item_n AS expected_items,
  @live_inst_n AS live_instruments,
  @live_item_n AS live_items,
  @inst_mismatch AS instrument_mismatches,
  @item_mismatch AS item_mismatches,
  @item_extra AS extra_items,
  @lp_submitted AS submitted_administrations,
  @lp_answers AS answers,
  @lp_scores AS dimension_scores,
  @catalog_empty AS catalog_empty,
  @exact_match AS exact_match,
  @content_conflict AS content_conflict,
  @allow_insert AS will_insert,
  @allow_noop AS will_noop;

INSERT INTO lp_seed_guard (step, allowed) VALUES
  ('expected_item_count', IF(@expected_item_n = 76, 1, 0));
INSERT INTO lp_seed_guard (step, allowed) VALUES
  ('no_content_conflict', IF(@content_conflict = 0, 1, 0));
INSERT INTO lp_seed_guard (step, allowed) VALUES
  ('empty_or_exact_match', IF(@allow_insert OR @allow_noop, 1, 0));
INSERT INTO lp_seed_guard (step, allowed) VALUES
  ('student_data_only_if_exact', IF((@lp_submitted = 0 AND @lp_answers = 0 AND @lp_scores = 0) OR @exact_match, 1, 0));

INSERT INTO learning_profile_instruments
  (code, name, version, program, scale_min, scale_max, item_count, status, created_at, updated_at)
SELECT e.code, e.name, e.version, e.program, e.scale_min, e.scale_max, e.item_count, 'active', CURRENT_TIMESTAMP(3), CURRENT_TIMESTAMP(3)
FROM expected_instruments e
WHERE @allow_insert = 1;

INSERT INTO learning_profile_items
  (instrument_id, item_code, source_item_no, stem, primary_dimension, secondary_dimension, scale_min, scale_max, scoring_direction, sort_order, options_json, created_at, updated_at)
SELECT i.id, e.item_code, e.source_item_no, e.stem, e.primary_dimension, e.secondary_dimension, e.scale_min, e.scale_max, e.scoring_direction, e.sort_order, e.options_json, CURRENT_TIMESTAMP(3), CURRENT_TIMESTAMP(3)
FROM expected_items e
JOIN learning_profile_instruments i ON i.code = e.instrument_code
WHERE @allow_insert = 1;

SET @after_inst_n := (
  SELECT COUNT(*) FROM learning_profile_instruments
  WHERE code IN ('DL-C56-v1', 'SDL-C20-v1')
);
SET @after_item_n := (
  SELECT COUNT(*) FROM learning_profile_items it
  JOIN learning_profile_instruments i ON i.id = it.instrument_id
  WHERE i.code IN ('DL-C56-v1', 'SDL-C20-v1')
);
INSERT INTO lp_seed_guard (step, allowed) VALUES
  ('post_count_2_and_76', IF(@after_inst_n = 2 AND @after_item_n = 76, 1, 0));

SELECT IF(@allow_insert = 1, 'inserted_official_catalog', 'noop_exact_match') AS seed_result,
       @after_inst_n AS instruments,
       @after_item_n AS items;

COMMIT;
END$$
DELIMITER ;

CALL seed_learning_profile_official_v1();
DROP PROCEDURE IF EXISTS seed_learning_profile_official_v1;
