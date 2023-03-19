# RETRY
[![Go Reference](https://pkg.go.dev/badge/github.com/aniladanir/retry#section-readme.svg)](https://pkg.go.dev/github.com/aniladanir/retry#section-readme)
[![Go Report Card](https://goreportcard.com/badge/github.com/aniladanir/retry)](https://goreportcard.com/report/github.com/aniladanir/retry)
[![codecov](https://codecov.io/gh/aniladanir/retry/branch/main/graph/badge.svg?token=EWXNSUE0FP)](https://codecov.io/gh/aniladanir/retry)

An implementation of capped exponential backoff algorithm with jitter.

# Usage

```go
// ...

// define condition
cond := func() bool {
    // try to establish connection to an external resource
    if err := TryConnect(); err != nil {
        return false
    }
    return true
}

// create a new retrier
retrier, err := retry.New(retry.DefaultConfiguration(), retry.DefaultRandomFunc)
if err != nil {
    panic(err)
}

// start retrying
notify := retrier.Retry(context.Background(), cond, true)

// wait for notification
<-notify

// ...
```