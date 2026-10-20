-- Reflection history tables. Does not ALTER or DELETE reflections.
-- Applied to production on 2026-10-20. New student rows must carry a module
-- and task item; those columns stay nullable so legacy snapshots can record
-- that the original row had no module or task.
-- legacy_reflection_id uses utf8mb4_0900_ai_ci so its foreign key matches
-- production reflections.id. The reflections table itself is unchanged.

CREATE TABLE IF NOT EXISTS reflection_entries (
  id varchar(36) NOT NULL,
  user_id bigint unsigned NOT NULL,
  resource_module_id bigint unsigned NULL,
  weekly_task_id bigint unsigned NULL,
  task_item_id bigint unsigned NULL,
  reflection_type varchar(32) NOT NULL DEFAULT 'post_class',
  summary text,
  challenges text,
  connections text,
  next_steps text,
  version int NOT NULL DEFAULT 1,
  source enum('student','legacy') NOT NULL,
  legacy_reflection_id varchar(36) NULL COLLATE utf8mb4_0900_ai_ci,
  idempotency_key varchar(64) NULL,
  created_at datetime(3) NOT NULL,
  updated_at datetime(3) NOT NULL,
  deleted_at datetime(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_reflection_entries_legacy (legacy_reflection_id),
  UNIQUE KEY uq_reflection_entries_idem (user_id, idempotency_key),
  KEY idx_reflection_entries_user_time (user_id, created_at, id),
  KEY idx_reflection_entries_user_module_time (user_id, resource_module_id, created_at, id),
  KEY idx_reflection_entries_user_task_time (user_id, task_item_id, created_at, id),
  CONSTRAINT fk_reflection_entries_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT fk_reflection_entries_module FOREIGN KEY (resource_module_id) REFERENCES c_programming_resources (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT fk_reflection_entries_weekly FOREIGN KEY (weekly_task_id) REFERENCES teacher_weekly_tasks (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT fk_reflection_entries_task FOREIGN KEY (task_item_id) REFERENCES task_items (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT fk_reflection_entries_legacy FOREIGN KEY (legacy_reflection_id) REFERENCES reflections (id) ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS reflection_entry_revisions (
  id bigint unsigned NOT NULL AUTO_INCREMENT,
  entry_id varchar(36) NOT NULL,
  version_superseded int NOT NULL,
  summary text,
  challenges text,
  connections text,
  next_steps text,
  editor_user_id bigint unsigned NOT NULL,
  created_at datetime(3) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_revisions_entry (entry_id, version_superseded),
  CONSTRAINT fk_revisions_entry FOREIGN KEY (entry_id) REFERENCES reflection_entries (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT fk_revisions_editor FOREIGN KEY (editor_user_id) REFERENCES users (id) ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS goal_reflection_links (
  id bigint unsigned NOT NULL AUTO_INCREMENT,
  entry_id varchar(36) NOT NULL,
  goal_id bigint unsigned NOT NULL,
  user_id bigint unsigned NOT NULL,
  next_steps_snapshot text,
  created_at datetime(3) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_goal_reflection_entry (entry_id),
  CONSTRAINT fk_grl_entry FOREIGN KEY (entry_id) REFERENCES reflection_entries (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT fk_grl_goal FOREIGN KEY (goal_id) REFERENCES goals (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT fk_grl_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT ON UPDATE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
