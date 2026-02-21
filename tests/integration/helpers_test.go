package integration

import (
	"context"
	"time"
)

// withTimeout returns a context with a 5-second deadline, convenient for DB / Redis calls.
func withTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
