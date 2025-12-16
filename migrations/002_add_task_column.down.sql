ALTER TABLE tasks
DROP COLUMN IF EXISTS task_type;

ALTER TABLE tasks
DROP COLUMN IF EXISTS error_message;

DROP INDEX IF EXISTS idx_tasks_task_type;
