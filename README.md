# RETRY
An implementation of capped exponential backoff algorithm with jitter.

# Usage

```go
// ...

// create a new retrier
retrier, err := retry.New(retry.DefaultConfiguration(), retry.DefaultRandomNumber())
if err != nil {
    // something is terribly wrong
}

// start retry loop
notify := retrier.Retry(context.TODO(), func() bool {
    // try to establish connection to an external resource
    if err := TryConnect(); err != nil {
        return false
    }
    return true
}, false)

// wait for notification
<-notify

// ...
```