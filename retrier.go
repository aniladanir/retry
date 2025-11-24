package retry

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

type Retrier struct {
	// timeFactor is the base duration.
	timeFactor time.Duration
	// minInterval is the minimum allowed retry interval.
	minInterval time.Duration
	// maxInterval is the maximum allowed retry interval.
	maxInterval time.Duration
	// Growth factor sets the base for the exponential function
	growthFactor int
	// RandomFunc represents a function that returns a random number in the half open interval [0,n)
	rf RandomFunc
	// pre-calculated value based on the given configuration that indicates max number of retries before reaching maximum interval.
	maxRetryCount int
	// MaxAttempts sets maximum number of attempts before retrier terminates.
	maxAttempts int
}

// Validate checks the configuration for invalid values
func (r *Retrier) Validate() error {
	if r.timeFactor <= 0 {
		return errors.New("base duration must be greater than zero")
	}
	if r.timeFactor >= r.maxInterval {
		return errors.New("maximum interval must be greater than base duration")
	}
	if r.minInterval >= r.maxInterval {
		return errors.New("maximum interval must be greater than minimum interval")
	}
	if r.growthFactor < 2 {
		return errors.New("growth factor must be greater than one")
	}
	if r.rf == nil {
		return errors.New("random function cannot be nil")
	}

	return nil
}

// RandomFunc represents a function that returns a random number in the half open interval [0,n)
type RandomFunc func(n int64) int64

// Condition represents a function that returns a boolean value which decides if retrier should terminate or not.
//
// attempt value starts from 1 and indicates the number of times the condition is called
type Condition func(attempt int) bool

// Default configuration values
const (
	DefaultTimeFactor   = time.Millisecond * 1000
	DefaultMinInterval  = time.Millisecond * 0
	DefaultMaxInterval  = time.Millisecond * 32000
	DefaultGrowthFactor = 2
	DefaultMaxAttempts  = 0
)

var newTicker = func(d time.Duration) *time.Ticker {
	return time.NewTicker(d)
}

// DefaultRandomFunc returns a random function that uses math/rand with nanosecond precision to generate random numbers.
func DefaultRandomFunc() RandomFunc {
	return func(n int64) int64 {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		return rng.Int63n(n)
	}
}

// New creates a new retrier with the default configuration.
func New(options ...Option) (*Retrier, error) {
	r := &Retrier{
		timeFactor:   DefaultTimeFactor,
		minInterval:  DefaultMinInterval,
		maxInterval:  DefaultMaxInterval,
		growthFactor: DefaultGrowthFactor,
		rf:           DefaultRandomFunc(),
		maxAttempts:  DefaultMaxAttempts,
	}
	for _, o := range options {
		o(r)
	}

	if err := r.Validate(); err != nil {
		return nil, err
	}

	for {
		if intPow(r.growthFactor, r.maxRetryCount)*int(r.timeFactor) > int(r.maxInterval)/2 {
			break
		}
		r.maxRetryCount++
	}

	return r, nil
}

// Retry executes the condition until it is satisfied.
//
// Notify channel will be closed right after the context gets canceled, the condition returns true or max attempts is reached.
//
// If immediate is true, condition will be run immediately. If false, the condition will run after the first retry interval has passed.
func (r *Retrier) Retry(ctx context.Context, cond Condition, immediate bool) (notify <-chan bool) {
	ch := make(chan bool)

	go r.retry(ctx, cond, ch, immediate)

	return ch
}

func (r *Retrier) retry(ctx context.Context, cond Condition, ch chan<- bool, immediate bool) {
	var (
		ticker  *time.Ticker
		retries = 0
	)

	defer func() {
		close(ch)
		ticker.Stop()
	}()

	ticker = newTicker(time.Nanosecond)
	stopAndDrainTicker(ticker)

	if immediate {
		if cond(retries + 1) {
			ch <- true
			return
		}
		retries++
	}

	for {
		if r.maxAttempts > 0 && retries >= r.maxAttempts {
			ch <- false
			return
		}

		ticker.Reset(r.next(retries))

		select {
		case <-ctx.Done():
			ch <- false
			return
		case <-ticker.C:
			if cond(retries + 1) {
				ch <- true
				return
			}
			stopAndDrainTicker(ticker)
			retries++
		}
	}
}

func (r *Retrier) next(retries int) time.Duration {
	var backoff time.Duration

	if r.maxRetryCount < retries {
		backoff = r.maxInterval
	} else {
		backoff = time.Duration(intPow(r.growthFactor, retries)) * r.timeFactor
	}

	return jitter(r.rf, r.minInterval, backoff)
}

func jitter(rn RandomFunc, min time.Duration, max time.Duration) time.Duration {
	random := time.Duration(rn(int64(max)))

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

func stopAndDrainTicker(ticker *time.Ticker) {
	ticker.Stop()

	select {
	case <-ticker.C:
	default:
	}
}
