package retry

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

// timeout value for tests calling retry
const notifyTimeout = time.Second * 5

var mockRn = func(n int64) int64 {
	return n
}

func TestRetry(t *testing.T) {
	tests := []struct {
		Name      string
		trueAfter int
		R         Retrier
		Timeout   time.Duration
		Immediate bool
	}{
		{
			Name:      "should call the condition correct number of times and with exponentially growing intervals",
			trueAfter: 5,
			R: Retrier{
				timeFactor:    time.Millisecond * 10,
				maxInterval:   time.Millisecond * 100,
				minInterval:   time.Millisecond * 0,
				growthFactor:  2,
				maxRetryCount: 3,
			},
			Timeout:   -1,
			Immediate: false,
		},
		{
			Name:      "should notify after context is canceled",
			trueAfter: 5,
			R: Retrier{
				timeFactor:    time.Millisecond * 11,
				maxInterval:   time.Millisecond * 100,
				minInterval:   time.Millisecond * 10,
				growthFactor:  2,
				maxRetryCount: 3,
			},
			Timeout:   time.Nanosecond * 1,
			Immediate: false,
		},
		{
			Name:      "should call condition immediately",
			trueAfter: 1,
			R: Retrier{
				timeFactor:    time.Millisecond * 10,
				maxInterval:   time.Millisecond * 100,
				minInterval:   time.Millisecond * 0,
				growthFactor:  2,
				maxRetryCount: 3,
			},
			Timeout:   -1,
			Immediate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			var (
				calls     = 0
				intervals = make([]int64, 0)
				ticker    = time.NewTicker(time.Nanosecond)
				cond      = func() bool {
					calls++
					return calls == tt.trueAfter
				}
			)

			tt.R.rf = func(n int64) int64 {
				intervals = append(intervals, n)
				return n
			}

			// inject ticker
			original := newTicker
			defer func() {
				newTicker = original
			}()
			newTicker = func(d time.Duration) *time.Ticker {
				return ticker
			}

			// create context
			var ctx context.Context
			if tt.Timeout <= 0 {
				ctx = context.Background()
			} else {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), tt.Timeout)
				defer cancel()
				ctx = timeoutCtx
			}

			// start retrying
			notify := tt.R.Retry(ctx, cond, tt.Immediate)

			// wait for notification
			select {
			case _, ok := <-notify:
				if ok {
					t.Error("received notify signal but the channel is not closed")
				}
				if tt.Timeout > 0 {
					if got := ctx.Err(); !os.IsTimeout(got) {
						t.Errorf("expected timeout error, got: %v", got)
					}
				}
			case <-time.After(notifyTimeout):
				t.Fatalf(`retry did not notify after %s second. two possibilities: 
				1) retry never notifies or 2) test takes too long to complete.`, notifyTimeout)
			}

			// test intervals
			for i, got := range intervals {
				// calculate expected interval
				expected := tt.R.timeFactor
				for j := 0; j < i; j++ {
					expected *= 2
				}
				if expected > tt.R.maxInterval {
					expected = tt.R.maxInterval
				}

				if gotDur := time.Duration(got); gotDur != expected {
					t.Errorf("expected interval: %s got: %s", expected, gotDur)
				}
			}

			// test immediate
			if tt.Immediate {
				if len(intervals) != 0 {
					t.Errorf("expected number of calls to randomfunc: 0, got: %d", len(intervals))
				}
			}

			t.Run("should stop ticker", func(t *testing.T) {
				select {
				case <-ticker.C:
				default:
				}
				// wait for more than maximum interval and check again if there is a tick
				maxTimer := time.NewTimer(tt.R.maxInterval * 2)
				defer maxTimer.Stop()
				<-maxTimer.C

				select {
				case <-ticker.C:
					t.Error("ticker is not stopped")
				default:
				}
			})
		})
	}
}

func TestRetryAsync(t *testing.T) {
	tests := []struct {
		Name      string
		trueAfter int
		R         Retrier
		Timeout   time.Duration
		Immediate bool
	}{
		{
			Name:      "should call the condition correct number of times and with exponentially growing intervals",
			trueAfter: 5,
			R: Retrier{
				timeFactor:    time.Millisecond * 10,
				maxInterval:   time.Millisecond * 100,
				minInterval:   time.Millisecond * 0,
				growthFactor:  2,
				maxRetryCount: 3,
			},
			Timeout:   -1,
			Immediate: false,
		},
		{
			Name:      "should notify after context is canceled",
			trueAfter: 5,
			R: Retrier{
				timeFactor:    time.Millisecond * 11,
				maxInterval:   time.Millisecond * 100,
				minInterval:   time.Millisecond * 10,
				growthFactor:  2,
				maxRetryCount: 3,
			},
			Timeout:   time.Nanosecond * 1,
			Immediate: false,
		},
		{
			Name:      "should call condition immediately",
			trueAfter: 1,
			R: Retrier{
				timeFactor:    time.Millisecond * 10,
				maxInterval:   time.Millisecond * 100,
				minInterval:   time.Millisecond * 0,
				growthFactor:  2,
				maxRetryCount: 3,
			},
			Timeout:   -1,
			Immediate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			var (
				calls     = 0
				intervals = make([]int64, 0)
				ticker    = time.NewTicker(time.Nanosecond)
				cond      = func() bool {
					calls++
					return calls == tt.trueAfter
				}
			)

			tt.R.rf = func(n int64) int64 {
				intervals = append(intervals, n)
				return n
			}

			// inject ticker
			original := newTicker
			defer func() {
				newTicker = original
			}()
			newTicker = func(d time.Duration) *time.Ticker {
				return ticker
			}

			// create context
			var ctx context.Context
			if tt.Timeout <= 0 {
				ctx = context.Background()
			} else {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), tt.Timeout)
				defer cancel()
				ctx = timeoutCtx
			}

			// start retrying
			notify := tt.R.RetryAsync(ctx, cond, tt.Immediate)

			// wait for notification
			select {
			case _, ok := <-notify:
				if ok {
					t.Error("received notify signal but the channel is not closed")
				}
				if tt.Timeout > 0 {
					if got := ctx.Err(); !os.IsTimeout(got) {
						t.Errorf("expected timeout error, got: %v", got)
					}
				}
			case <-time.After(notifyTimeout):
				t.Fatalf(`retry did not notify after %s second. two possibilities: 
				1) retry never notifies or 2) test takes too long to complete.`, notifyTimeout)
			}

			// test intervals
			for i, got := range intervals {
				// calculate expected interval
				expected := tt.R.timeFactor
				for j := 0; j < i; j++ {
					expected *= 2
				}
				if expected > tt.R.maxInterval {
					expected = tt.R.maxInterval
				}

				if gotDur := time.Duration(got); gotDur != expected {
					t.Errorf("expected interval: %s got: %s", expected, gotDur)
				}
			}

			// test immediate
			if tt.Immediate {
				if len(intervals) != 1 {
					t.Errorf("expected number of calls to randomfunc: 1, got: %d", len(intervals))
				} else if intervals[0] != int64(tt.R.timeFactor) {
					t.Errorf("expected first interval: %s, got: %s", tt.R.timeFactor, time.Duration(intervals[0]))
				}
			}

			t.Run("should stop ticker", func(t *testing.T) {
				select {
				case <-ticker.C:
				default:
				}
				// wait for more than maximum interval and check again if there is a tick
				maxTimer := time.NewTimer(tt.R.maxInterval * 2)
				defer maxTimer.Stop()
				<-maxTimer.C

				select {
				case <-ticker.C:
					t.Error("ticker is not stopped")
				default:
				}
			})
		})
	}
}

func TestNext(t *testing.T) {
	tests := []struct {
		Name           string
		r              Retrier
		RetryBeforeMax int
		Attempt        int
		Want           time.Duration
	}{
		{
			Name: "should return base_duration * 2 ^ attempt ",
			r: Retrier{
				timeFactor:    2,
				minInterval:   0,
				maxInterval:   10,
				growthFactor:  2,
				maxRetryCount: 10,
				rf:            mockRn,
			},
			Attempt: 3,
			Want:    16,
		},
		{
			Name: "should return max interval if attempt is greater than maximum retry coutn",
			r: Retrier{
				timeFactor:    2,
				minInterval:   0,
				maxInterval:   10,
				growthFactor:  2,
				maxRetryCount: 2,
				rf:            mockRn,
			},
			Attempt: 3,
			Want:    10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			if got := tt.r.next(tt.Attempt); got != tt.Want {
				t.Errorf("expected: %s, got: %s", tt.Want, got)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		give *Retrier
		want error
	}{
		{
			name: "should return no errors if configuration is valid",
			give: &Retrier{
				timeFactor:   1,
				minInterval:  1,
				maxInterval:  2,
				growthFactor: 2,
				rf:           mockRn,
			},
			want: nil,
		},
		{
			name: "should return error if base duration is zero",
			give: &Retrier{
				timeFactor: 0,
			},
			want: errors.New("base duration must be greater than zero"),
		},
		{
			name: "should return error if base duration is negative",
			give: &Retrier{
				timeFactor: -1,
			},
			want: errors.New("base duration must be greater than zero"),
		},
		{
			name: "should return error if base duration is greater than maximum interval",
			give: &Retrier{
				timeFactor:  1,
				maxInterval: 0,
			},
			want: errors.New("maximum interval must be greater than base duration"),
		},
		{
			name: "should return error if base duration is equal to maximum interval",
			give: &Retrier{
				timeFactor:  1,
				maxInterval: 0,
			},
			want: errors.New("maximum interval must be greater than base duration"),
		},
		{
			name: "should return error if minimum interval is greater than maximum interval",
			give: &Retrier{
				timeFactor:  1,
				minInterval: 3,
				maxInterval: 2,
			},
			want: errors.New("maximum interval must be greater than minimum interval"),
		},
		{
			name: "should return error if minimum interval is equal to maximum interval",
			give: &Retrier{
				timeFactor:  1,
				minInterval: 2,
				maxInterval: 2,
			},
			want: errors.New("maximum interval must be greater than minimum interval"),
		},
		{
			name: "should return error if growth factor is less than two",
			give: &Retrier{
				timeFactor:   1,
				minInterval:  0,
				maxInterval:  2,
				growthFactor: 1,
			},
			want: errors.New("growth factor must be greater than one"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.give.Validate(); got != tt.want && got.Error() != tt.want.Error() {
				t.Errorf("expected: %+v got: %+v", tt.want, got)
			}
		})
	}
}

func TestJitter(t *testing.T) {
	tests := []struct {
		Name string
		Min  time.Duration
		Max  time.Duration
		Want time.Duration
	}{
		{
			Name: "should return correct duration",
			Min:  1,
			Max:  2,
			Want: 2,
		},
		{
			Name: "should return min if max is less than min",
			Min:  1,
			Max:  0,
			Want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			if got := jitter(mockRn, tt.Min, tt.Max); got != tt.Want {
				t.Errorf("expected: %s, got: %s", tt.Want, got)
			}
		})
	}
}

func TestIntPow(t *testing.T) {
	tests := []struct {
		Name     string
		Base     int
		Exponent int
		Want     int
	}{
		{
			Name:     "should return expected value",
			Base:     2,
			Exponent: 5,
			Want:     32,
		},
		{
			Name:     "should return 1 if exponent is 0",
			Base:     0,
			Exponent: 0,
			Want:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			if got := intPow(tt.Base, tt.Exponent); got != tt.Want {
				t.Errorf("expected: %d, got: %d", tt.Want, got)
			}
		})
	}
}
