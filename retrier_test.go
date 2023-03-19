package retry

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"
)

// timeout value for tests calling retry
const notifyTimeout = time.Second * 5

func TestRetry(t *testing.T) {
	tests := []struct {
		Name      string
		trueAfter int
		Config    Configuration
		Timeout   time.Duration
		Immediate bool
	}{
		{
			Name:      "should call the condition correct number of times and with exponentially growing intervals",
			trueAfter: 5,
			Config: Configuration{
				Base:         time.Millisecond * 10,
				MaxInterval:  time.Millisecond * 100,
				MinInterval:  time.Millisecond * 0,
				GrowthFactor: 2,
			},
			Timeout:   -1,
			Immediate: false,
		},
		{
			Name:      "should notify after context is canceled",
			trueAfter: 5,
			Config: Configuration{
				Base:         time.Millisecond * 11,
				MaxInterval:  time.Millisecond * 100,
				MinInterval:  time.Millisecond * 10,
				GrowthFactor: 2,
			},
			Timeout:   time.Nanosecond * 1,
			Immediate: false,
		},
		{
			Name:      "should call condition immediately",
			trueAfter: 1,
			Config: Configuration{
				Base:         time.Millisecond * 10,
				MaxInterval:  time.Millisecond * 100,
				MinInterval:  time.Millisecond * 0,
				GrowthFactor: 2,
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
				rn        = func(n int64) int64 {
					intervals = append(intervals, n)
					return n
				}
				cond = func() bool {
					calls++
					return calls == tt.trueAfter
				}
			)

			// inject ticker
			original := newTicker
			defer func() {
				newTicker = original
			}()
			newTicker = func(d time.Duration) *time.Ticker {
				return ticker
			}

			// create new retrier
			retrier, err := New(tt.Config, rn)
			if err != nil {
				t.Fatalf("error while initializing retrier: %v", err)
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
			notify := retrier.Retry(ctx, cond, tt.Immediate)

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
				expected := tt.Config.Base
				for j := 0; j < i; j++ {
					expected *= 2
				}
				if expected > tt.Config.MaxInterval {
					expected = tt.Config.MaxInterval
				}

				if got != int64(expected) {
					t.Errorf("expected interval: %d got: %d", expected, got)
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
				maxTimer := time.NewTimer(retrier.config.MaxInterval * 2)
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
		Config    Configuration
		Timeout   time.Duration
		Immediate bool
	}{
		{
			Name:      "should call the condition correct number of times and with exponentially growing intervals",
			trueAfter: 5,
			Config: Configuration{
				Base:         time.Millisecond * 10,
				MaxInterval:  time.Millisecond * 100,
				MinInterval:  time.Millisecond * 0,
				GrowthFactor: 2,
			},
			Timeout:   -1,
			Immediate: false,
		},
		{
			Name:      "should notify after context is canceled",
			trueAfter: 5,
			Config: Configuration{
				Base:         time.Millisecond * 11,
				MaxInterval:  time.Millisecond * 100,
				MinInterval:  time.Millisecond * 10,
				GrowthFactor: 2,
			},
			Timeout:   time.Nanosecond * 1,
			Immediate: false,
		},
		{
			Name:      "should call condition immediately",
			trueAfter: 1,
			Config: Configuration{
				Base:         time.Millisecond * 10,
				MaxInterval:  time.Millisecond * 100,
				MinInterval:  time.Millisecond * 0,
				GrowthFactor: 2,
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
				rn        = func(n int64) int64 {
					intervals = append(intervals, n)
					return n
				}
				cond = func() bool {
					calls++
					return calls == tt.trueAfter
				}
			)

			// inject ticker
			original := newTicker
			defer func() {
				newTicker = original
			}()
			newTicker = func(d time.Duration) *time.Ticker {
				return ticker
			}

			// create new retrier
			retrier, err := New(tt.Config, rn)
			if err != nil {
				t.Fatalf("error while initializing retrier: %v", err)
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
			notify := retrier.RetryAsync(ctx, cond, tt.Immediate)

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
				expected := tt.Config.Base
				for j := 0; j < i; j++ {
					expected *= 2
				}
				if expected > tt.Config.MaxInterval {
					expected = tt.Config.MaxInterval
				}

				if got != int64(expected) {
					t.Errorf("expected interval: %d got: %d", expected, got)
				}
			}

			// test immediate
			if tt.Immediate {
				if len(intervals) != 1 {
					t.Errorf("expected number of calls to randomfunc: 1, got: %d", len(intervals))
				} else if intervals[0] != int64(tt.Config.Base) {
					t.Errorf("expected first interval: %s, got: %s", tt.Config.Base, time.Duration(intervals[0]))
				}
			}

			t.Run("should stop ticker", func(t *testing.T) {
				select {
				case <-ticker.C:
				default:
				}
				// wait for more than maximum interval and check again if there is a tick
				maxTimer := time.NewTimer(retrier.config.MaxInterval * 2)
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
		Config         Configuration
		RetryBeforeMax int
		Attempt        int
		Want           time.Duration
	}{
		{
			Name: "should return base_duration * 2 ^ attempt ",
			Config: Configuration{
				Base:         2,
				MinInterval:  0,
				MaxInterval:  10,
				GrowthFactor: 2,
			},
			RetryBeforeMax: 10,
			Attempt:        3,
			Want:           16,
		},
		{
			Name: "should return max interval if attempt is greater than retryBeforeMax",
			Config: Configuration{
				Base:         2,
				MinInterval:  0,
				MaxInterval:  10,
				GrowthFactor: 2,
			},
			RetryBeforeMax: 2,
			Attempt:        3,
			Want:           10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// set random function
			mockRn := func(n int64) int64 {
				return n
			}

			// create new retrier
			retrier, err := New(tt.Config, mockRn)
			if err != nil {
				t.Fatalf("error while initializing retrier: %v", err)
			}

			retrier.retryBeforeMax = tt.RetryBeforeMax

			if got := retrier.next(tt.Attempt); got != tt.Want {
				t.Errorf("expected: %s, got: %s", tt.Want, got)
			}
		})
	}
}

func TestNew(t *testing.T) {
	mockRn := func(n int64) int64 {
		return n
	}

	tests := []struct {
		Name    string
		Config  Configuration
		Rn      RandomFunc
		Want    Retrier
		WantErr error
	}{
		{
			Name:   "should set fields correctly",
			Config: DefaultConfiguration(),
			Rn:     mockRn,
			Want: Retrier{
				rn:             mockRn,
				config:         DefaultConfiguration(),
				retryBeforeMax: 5,
			},
			WantErr: nil,
		},
		{
			Name:   "should default to DefaultRandomFunc if not provided",
			Config: DefaultConfiguration(),
			Rn:     nil,
			Want: Retrier{
				rn:             DefaultRandomFunc,
				config:         DefaultConfiguration(),
				retryBeforeMax: 5,
			},
			WantErr: nil,
		},
		{
			Name: "should return error",
			Rn:   mockRn,
			Config: Configuration{
				Base:        -1,
				MinInterval: -1,
				MaxInterval: -1,
			},
			Want: Retrier{
				rn: mockRn,
				config: Configuration{
					Base:        -1,
					MinInterval: -1,
					MaxInterval: -1,
				},
				retryBeforeMax: 5,
			},
			WantErr: errors.New("base duration must be greater than zero"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			got, gotErr := New(tt.Config, tt.Rn)
			if gotErr != tt.WantErr && gotErr.Error() != tt.WantErr.Error() {
				t.Errorf("expected error: %v got error: %v", tt.WantErr, gotErr)
			}

			if got != nil {
				// isolate random functions
				gotRn := got.rn
				got.rn = nil
				wantRn := tt.Want.rn
				tt.Want.rn = nil

				if !reflect.DeepEqual(*got, tt.Want) {
					t.Errorf("expected: %+v got: %+v", tt.Want, got)
				}

				if reflect.ValueOf(gotRn).Pointer() != reflect.ValueOf(wantRn).Pointer() {
					t.Errorf("expected random function does not match got random function")
				}
			}
		})
	}
}

func TestValidateConfiguration(t *testing.T) {
	tests := []struct {
		name string
		give Configuration
		want error
	}{
		{
			name: "should return no errors if configuration is valid",
			give: Configuration{
				Base:         1,
				MinInterval:  1,
				MaxInterval:  2,
				GrowthFactor: 2,
			},
			want: nil,
		},
		{
			name: "should return error if base duration is zero",
			give: Configuration{
				Base: 0,
			},
			want: errors.New("base duration must be greater than zero"),
		},
		{
			name: "should return error if base duration is negative",
			give: Configuration{
				Base: -1,
			},
			want: errors.New("base duration must be greater than zero"),
		},
		{
			name: "should return error if base duration is greater than maximum interval",
			give: Configuration{
				Base:        1,
				MaxInterval: 0,
			},
			want: errors.New("maximum interval must be greater than base duration"),
		},
		{
			name: "should return error if base duration is equal to maximum interval",
			give: Configuration{
				Base:        1,
				MaxInterval: 0,
			},
			want: errors.New("maximum interval must be greater than base duration"),
		},
		{
			name: "should return error if minimum interval is greater than maximum interval",
			give: Configuration{
				Base:        1,
				MinInterval: 3,
				MaxInterval: 2,
			},
			want: errors.New("maximum interval must be greater than minimum interval"),
		},
		{
			name: "should return error if minimum interval is equal to maximum interval",
			give: Configuration{
				Base:        1,
				MinInterval: 2,
				MaxInterval: 2,
			},
			want: errors.New("maximum interval must be greater than minimum interval"),
		},
		{
			name: "should return error if growth factor is less than two",
			give: Configuration{
				Base:         1,
				MinInterval:  0,
				MaxInterval:  2,
				GrowthFactor: 1,
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

func TestDefaultConfig(t *testing.T) {
	expected := Configuration{
		Base:         DefaultBase,
		MinInterval:  DefaultMinInterval,
		MaxInterval:  DefaultMaxInterval,
		GrowthFactor: DefaultGrowthFactor,
	}
	got := DefaultConfiguration()

	if !reflect.DeepEqual(expected, got) {
		t.Errorf("expected: %+v, got: %+v", expected, got)
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
			// set random func
			mockRn := func(n int64) int64 {
				return n
			}

			if got := jitter(mockRn, tt.Min, tt.Max); got != tt.Want {
				t.Errorf("expected: %s, got: %s", tt.Want, got)
			}
		})
	}
}

func TestStopAndDrainTicker(t *testing.T) {
	ticker := time.NewTicker(time.Nanosecond)
	defer ticker.Stop()

	// wait for tick
	time.Sleep(time.Nanosecond * 2)

	stopAndDrainTicker(ticker)

	select {
	case <-ticker.C:
		t.Error("ticker is not stopped or there was a unread tick left in the channel")
	case <-time.After(time.Nanosecond * 2):
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

func TestDefaultRandomFunc(t *testing.T) {
	for i := 1; i < 10000; i++ {
		if got := DefaultRandomFunc(int64(i)); got >= int64(i) {
			t.Errorf("expected to be less than: %d, got: %d", i, got)
		}
	}
}

func TestNewTicker(t *testing.T) {
	expectedPeriod := time.Millisecond * 10

	ticker := newTicker(expectedPeriod)
	if ticker == nil {
		t.Fatal("returned ticker is nil")
	}

	select {
	case <-ticker.C:
	case <-time.After(expectedPeriod):
		t.Error("ticker period is set incorrectly or ticker is stopped")
	}

	period := reflect.ValueOf(ticker).Elem().Field(1).Field(2)
	if !period.CanInt() {
		t.Fatal("cannot access period field")
	}

	if got := period.Int(); got != int64(expectedPeriod) {
		t.Errorf("expected: %s, got: %s", expectedPeriod, time.Duration(got))
	}
}
