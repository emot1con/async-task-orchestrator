ALTER TABLE tasks
ADD COLUMN task_type VARCHAR(50) NOT NULL DEFAULT 'GENERATE_REPORT';

ALTER TABLE tasks
ADD COLUMN error_message TEXT;

CREATE INDEX idx_tasks_task_type ON tasks(task_type);
