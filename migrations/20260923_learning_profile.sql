-- 网站自主测评：七维学习画像最小表结构。
-- 本文件是草案，阶段1不要执行。不要通过 AutoMigrate 代替本文件。
-- 不创建 learning_profile_windows、learning_profile_teacher_grants、
-- learning_profile_student_clock。
--
-- 前置：users 表已存在（user_id 对应 users.id，BIGINT UNSIGNED）。
-- 本表无软删除列，唯一约束覆盖全部行，以保证同一学生、同一量表版本、
-- 同一 program、同一测评阶段最多一条测评记录。
-- 草稿与正式提交共用 administrations 一行，靠 status 区分。
-- 正式提交后的冻结写入 item_snapshot（JSON，不另建表）：
--   instrumentCode, instrumentVersion, instrumentName, program, scaleMin, scaleMax,
--   items[]: itemCode, sourceItemNo, stem, primaryDimension, secondaryDimension,
--            scaleMin, scaleMax, scoringDirection, sortOrder,
--            options[{value,label}], rawValue
-- 后续改题须新 instrument code；已提交快照不得回写。
--
-- 本文件是 DDL：MySQL CREATE TABLE 会隐式提交，不能与
-- 20260923_learning_profile_seed.sql 的 INSERT 放进同一个可回滚事务。
--
-- 有正式 submitted 行之后禁止用 DROP TABLE 作为默认回滚。
-- 空隔离库经批准后可整组 DROP（逆序）。
--
-- 可重复性：CREATE TABLE 在表已存在时会报错。

CREATE TABLE learning_profile_instruments (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  code VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  version VARCHAR(32) NOT NULL,
  program VARCHAR(64) NOT NULL,
  scale_min INT NOT NULL,
  scale_max INT NOT NULL,
  item_count INT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  PRIMARY KEY (id),
  UNIQUE KEY uk_lp_instruments_code (code),
  KEY idx_lp_instruments_program (program)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE learning_profile_items (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  instrument_id BIGINT UNSIGNED NOT NULL,
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
  PRIMARY KEY (id),
  UNIQUE KEY uk_lp_items_instrument_code (instrument_id, item_code),
  KEY idx_lp_items_dimension (primary_dimension),
  CONSTRAINT fk_lp_items_instrument
    FOREIGN KEY (instrument_id) REFERENCES learning_profile_instruments (id)
    ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE learning_profile_administrations (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  user_id BIGINT UNSIGNED NOT NULL,
  instrument_id BIGINT UNSIGNED NOT NULL,
  wave_type VARCHAR(16) NOT NULL,
  program VARCHAR(64) NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'draft',
  paper_version VARCHAR(64) NOT NULL,
  item_snapshot JSON NULL,
  started_at DATETIME(3) NULL,
  submitted_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_lp_admin_user_inst_wave_program (user_id, instrument_id, wave_type, program),
  KEY idx_lp_admin_user_program (user_id, program),
  KEY idx_lp_admin_status (status),
  CONSTRAINT fk_lp_admin_user
    FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT fk_lp_admin_instrument
    FOREIGN KEY (instrument_id) REFERENCES learning_profile_instruments (id)
    ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE learning_profile_answers (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  administration_id BIGINT UNSIGNED NOT NULL,
  item_id BIGINT UNSIGNED NOT NULL,
  item_code VARCHAR(32) NOT NULL,
  raw_value INT NOT NULL,
  answered_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_lp_answers_admin_item (administration_id, item_id),
  KEY idx_lp_answers_admin (administration_id),
  CONSTRAINT fk_lp_answers_admin
    FOREIGN KEY (administration_id) REFERENCES learning_profile_administrations (id)
    ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT fk_lp_answers_item
    FOREIGN KEY (item_id) REFERENCES learning_profile_items (id)
    ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE learning_profile_dimension_scores (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  administration_id BIGINT UNSIGNED NOT NULL,
  dimension_code VARCHAR(64) NOT NULL,
  raw_mean DECIMAL(8,4) NOT NULL,
  scale_min INT NOT NULL,
  scale_max INT NOT NULL,
  display_score DECIMAL(8,4) NOT NULL,
  item_count INT NOT NULL,
  computed_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_lp_dim_admin_code (administration_id, dimension_code),
  KEY idx_lp_dim_admin (administration_id),
  CONSTRAINT fk_lp_dim_admin
    FOREIGN KEY (administration_id) REFERENCES learning_profile_administrations (id)
    ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
