package retry

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"
)

var ErrFatal error = errors.New("fatal error")

func mergeErr(in ...error) error {
	var err error
	for i, e := range in {
		if e == nil {
			continue
		}
		if i == 0 {
			err = e
			continue
		}
		err = fmt.Errorf("%w; %w", err, e)
	}

	return err
}

func WithTimeout(ctx context.Context, timeout, initialDelay, maxDelay time.Duration, f func() error) (err error) {
	if initialDelay == 0 {
		initialDelay = time.Millisecond
	}
	currentDelay := initialDelay
	rCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		if rCtx.Err() != nil {
			return mergeErr(rCtx.Err(), err)
		}

		err = f()
		if err == nil {
			return nil
		}
		if errors.Is(err, ErrFatal) {
			return err
		}

		currentDelay += time.Duration(float64(rand.Int64N(int64(initialDelay))))
		if currentDelay > maxDelay {
			currentDelay = maxDelay
		}

		select {
		case <-rCtx.Done():
			return mergeErr(rCtx.Err(), err)
		case <-time.After(currentDelay):
			currentDelay *= 2
		}
	}
}
