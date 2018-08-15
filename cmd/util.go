package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/hetznercloud/hcloud-go/hcloud"
)

func waitAction(ctx context.Context, client *hcloud.Client, action *hcloud.Action) (<-chan error, <-chan int) {
	errCh := make(chan error, 1)
	progressCh := make(chan int)

	go func() {
		defer close(errCh)
		defer close(progressCh)

		ticker := time.NewTicker(100 * time.Millisecond)

		sendProgress := func(p int) {
			select {
			case progressCh <- p:
				break
			default:
				break
			}
		}

		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case <-ticker.C:
			}

			action, _, err := client.Action.GetByID(ctx, action.ID)
			if err != nil {
				errCh <- ctx.Err()
				return
			}

			switch action.Status {
			case hcloud.ActionStatusRunning:
				sendProgress(action.Progress)

			case hcloud.ActionStatusSuccess:
				sendProgress(100)
				errCh <- nil
				return
			case hcloud.ActionStatusError:
				errCh <- action.Error()
				return
			}
		}
	}()

	return errCh, progressCh
}

func randomName() string {
	return fmt.Sprintf("%s-%s%s", randomdata.Adjective(), randomdata.Noun(), randomdata.Adjective())
}

//Index find the index of an element int the array
func Index(vs []string, t string) int {
	for i, v := range vs {
		if v == t {
			return i
		}
	}
	return -1
}

//Include indicate if a string is in the strinc array
func Include(vs []string, t string) bool {
	return Index(vs, t) >= 0
}

//FatalOnError is an helper function to transform error to fatl
func FatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
