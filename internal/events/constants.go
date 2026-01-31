package events

// Event types
const (
	EventTypeTaskCreated   = "task.created"
	EventTypeTaskStarted   = "task.started"
	EventTypeTaskCompleted = "task.completed"
	EventTypeTaskFailed    = "task.failed"

	EventTypeUserRegistered = "user.registered"
	EventTypeUserLoggedIn   = "user.logged_in"
)

// Event versions
const (
	EventVersionV1 = "v1"
	EventVersionV2 = "v2"
)

// Sources
const (
	SourceAPI    = "task-handler.api"
	SourceWorker = "task-handler.worker"
)

// Status
const (
	StatusPending    = "PENDING"
	StatusProcessing = "PROCESSING"
	StatusSuccess    = "SUCCESS"
	StatusFailed     = "FAILED"
)
