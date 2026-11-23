package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type Service interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type operation func(ctx context.Context) error

func gracefulShutdown(ctx context.Context, timeout time.Duration, ops map[string]operation) <-chan struct{} {
	wait := make(chan struct{})

	go func() {
		var wg sync.WaitGroup

		timeoutFunc := time.AfterFunc(timeout, func() {
			log.Printf("WARN: timeout '%d' ms has been elapsed, force exit", timeout.Milliseconds())
			os.Exit(0)
		})

		defer timeoutFunc.Stop()

		for key, op := range ops {
			wg.Add(1)
			innerOp := op
			innerKey := key
			go func() {
				defer wg.Done()

				fmt.Printf("INFO: cleaning up '%s' \n", innerKey)
				if err := innerOp(ctx); err != nil {
					log.Printf("ERROR: '%s' clean up failed: %s", innerKey, err.Error())
					return
				}

				fmt.Printf("INFO: '%s' was shutdown gracefully \n", innerKey)
			}()
		}

		wg.Wait()

		close(wait)
	}()

	return wait
}
