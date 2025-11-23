package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/aniladanir/retry"
)

func main() {
	r, err := retry.New(
		retry.WithGrowthFactor(2),
		retry.WithMaxInterval(time.Second*30),
		retry.WithMaxAttemps(10),
	)
	if err != nil {
		log.Fatal(err)
	}

	success := <-r.Retry(
		context.Background(),
		func() bool {
			_, err = http.DefaultClient.Get("https://www.google.com")
			return err == nil
		},
		true,
	)
	if err != nil {
		log.Fatalf("request has failed: %v", err)
	}

	if success {
		log.Println("request is successful.")
	} else {
		log.Println("request failed.")
	}
}
