package util

import "context"

// Checks if a context is done in a non-blocking way.
// If the context was done, the function will also return a message
// describing why.
func IsDone(ctx context.Context) (bool, string) {
	select {
	case <-ctx.Done():
		msg := "unknown context cancellation"
		if ctx.Err() != nil {
			msg = ctx.Err().Error()
		}
		return true, msg
	default:
	}
	return false, ""
}
