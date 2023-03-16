// Package retry contains an implementation of capped exponential backoff algorithm with full jitter.
//
// Reference: https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
package retry

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

type Retrier struct {
	rn             RandomNumber
	config         Configuration
	retryBeforeMax int
}

type RandomNumber interface {
	Int63n(n int64) int64
}

type randomNumber struct{}

var _ RandomNumber = (*randomNumber)(nil)

func (randomNumber) Int63n(n int64) int64 {
	rand.Seed(time.Now().UnixNano())
	return rand.Int63n(n)
}

// Configuration defines the settings for retrier
type Configuration struct {
	// Base is the base duration.
	Base time.Duration
	// MinInterval is the minimum allowed retry interval.
	MinInterval time.Duration
	// MaxInterval is the maximum allowed retry interval.
	MaxInterval time.Duration
}

// Validate checks the configuration for invalid values
func (c Configuration) Validate() error {
	if c.Base <= 0 {
		return errors.New("base duration must be greater than zero")
	}
	if c.Base >= c.MaxInterval {
		return errors.New("maximum duration must be greater than base duration")
	}
	if c.MinInterval >= c.MaxInterval {
		return errors.New("maximum duration must be greater than minimum duration")
	}

	return nil
}

// Condition function returns a boolean value that decides if retrier should terminate.
type Condition func() bool

// Default configuration values
const (
	DefaultBase        = time.Millisecond * 1000
	DefaultMinInterval = time.Millisecond * 0
	DefaultMaxInterval = time.Millisecond * 32000
)

// New creates a new retrier with the default configuration.
// If rn is not provided, it will use the default RandomNumber.
func New(config Configuration, rn RandomNumber) (*Retrier, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	if rn == nil {
		rn = DefaultRandomNumber()
	}

	i := 0
	for ; ; i++ {
		if intPow(2, i)*int(config.Base) > int(config.MaxInterval)/2 {
			break
		}
	}

	return &Retrier{
		rn:             rn,
		config:         config,
		retryBeforeMax: i - 1,
	}, nil
}

// DefaultRandomNumber returns the default RandomNumber interface that
// uses math/rand seeded with nanosecond precision for random number generation
func DefaultRandomNumber() RandomNumber {
	return &randomNumber{}
}

// DefaultConfiguration returns the default configuration for retrier
func DefaultConfiguration() Configuration {
	return Configuration{
		Base:        DefaultBase,
		MinInterval: DefaultMinInterval,
		MaxInterval: DefaultMaxInterval,
	}
}

// Retry executes the condition until it is satisfied.
//
// Notify channel will be closed right after the context gets canceled or the condition returns true.
//
// If immediate is true, condition will be run immediately.
func (r *Retrier) Retry(ctx context.Context, cond Condition, immediate bool) (notify <-chan struct{}) {
	ch := make(chan struct{})

	go r.retry(ctx, cond, ch, immediate)

	return ch
}

// RetryAsync executes the condition asynchronously with the timer until it is satisfied.
//
// Notify channel will be closed right after the context gets canceled or the condition returns true.
func (r *Retrier) RetryAsync(ctx context.Context, cond Condition, immediate bool) (notify <-chan struct{}) {
	ch := make(chan struct{})

	go r.retryAsync(ctx, cond, ch, immediate)

	return ch
}

func (r *Retrier) retry(ctx context.Context, cond Condition, ch chan<- struct{}, immediate bool) {
	var (
		timer   *time.Timer
		attempt = 0
	)

	defer func() {
		close(ch)
		timer.Stop()
	}()

	if immediate {
		if cond() {
			return
		}
	}

	for {
		timer = time.NewTimer(r.next(attempt))

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if cond() {
				return
			}
			attempt++
		}
	}
}

func (r *Retrier) retryAsync(ctx context.Context, cond Condition, ch chan<- struct{}, immediate bool) {
	var (
		timer   *time.Timer
		attempt = 0
	)

	defer func() {
		close(ch)
		timer.Stop()
	}()

	if !immediate {
		timer = time.NewTimer(r.next(attempt))
		<-timer.C
	}

	for {
		timer = time.NewTimer(r.next(attempt))

		if cond() {
			return
		}
		attempt++

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
	}
}

func (r *Retrier) next(attempt int) time.Duration {
	var backoff time.Duration

	if r.retryBeforeMax < attempt {
		backoff = r.config.MaxInterval
	} else {
		backoff = time.Duration(intPow(2, attempt)) * r.config.Base
	}

	return jitter(r.rn, r.config.MinInterval, backoff)
}

func jitter(rn RandomNumber, min time.Duration, max time.Duration) time.Duration {
	random := time.Duration(rn.Int63n(int64(max)))

	if random <= min {
		return min
	}

	return random
}

func intPow(base int, exponent int) int {
	if exponent == 0 {
		return 1
	}

	result := base
	for i := 1; i < exponent; i++ {
		result *= base
	}

	return result
}
