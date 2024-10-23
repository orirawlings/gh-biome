package retry

import (
	"context"
	"time"
)

func WithBackoff(ctx context.Context, maxAttempts int, maxDelay time.Duration, do func() error) error {
	delay := 5 * time.Millisecond
	var err error
	for i := maxAttempts; i > 0; i-- {
		err = do()
		if err == nil {
			return nil
		}
		if i > 1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
		}
	}
	return err
}
