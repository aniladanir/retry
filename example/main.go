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
	)
	if err != nil {
		log.Fatal(err)
	}

	<-r.Retry(
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

	log.Println("request is successful.")
}
