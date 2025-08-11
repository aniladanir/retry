package retry

import "time"

type Option func(c *Retrier)

// WithTimeFactor sets the time factor for exponential function. Default is one second.
func WithTimeFactor(d time.Duration) Option {
	return func(c *Retrier) {
		c.timeFactor = d
	}
}

// WithGrowthFactor sets the base for exponential function
func WithGrowthFactor(g int) Option {
	return func(c *Retrier) {
		c.growthFactor = g
	}
}

// WithMaxInterval sets the maximum retry interval that is allowed.
func WithMaxInterval(d time.Duration) Option {
	return func(c *Retrier) {
		c.maxInterval = d
	}
}

// WithMinInterval sets the minimum retry interval that is allowed.
func WithMinInterval(d time.Duration) Option {
	return func(c *Retrier) {
		c.minInterval = d
	}
}

// WithRandomFunc sets the random function that is used to generate random values in the half open interval [0,n).
func WithRandomFunc(rf RandomFunc) Option {
	return func(c *Retrier) {
		c.rf = rf
	}
}
