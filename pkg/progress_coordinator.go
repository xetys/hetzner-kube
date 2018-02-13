package pkg

import (
	"fmt"
	"github.com/go-kit/kit/log/term"
	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uiprogress/util/strutil"
	"os"
	"sync"
)

type ProgressCoordinator struct {
	group      sync.WaitGroup
	progresses map[string]*Progress
}

var RenderProgressBars bool

func NewProgressCoordinator() *ProgressCoordinator {
	if isUiEnabled() {
		uiprogress.Start()
	}
	pc := new(ProgressCoordinator)
	pc.progresses = make(map[string]*Progress)

	return pc
}

func isUiEnabled() bool {
	if RenderProgressBars {
		return term.IsTerminal(os.Stdout)
	} else {
		return false
	}
}

func (c *ProgressCoordinator) StartProgress(name string, steps int) {
	progress := &Progress{
		Bar:     uiprogress.AddBar(steps),
		State:   "starting",
		channel: make(chan string),
	}
	progress.Bar.PrependFunc(func(b *uiprogress.Bar) string {
		percent := strutil.PadLeft(fmt.Sprintf("%.01f%%", b.CompletedPercent()), 6, ' ')
		return fmt.Sprintf("%s: %s  %s", name, strutil.PadRight(progress.State, 40, ' '), percent)
	})
	c.progresses[name] = progress
	c.group.Add(1)
	go func(progress *Progress) {
		for {
			event := <-progress.channel
			if event == "complete!" {
				progress.Bar.Set(progress.Bar.Total)
				progress.SetText(event)
				break
			}

			if !isUiEnabled() {
				fmt.Printf("%s (%d)", event, progress.Bar.Current()+1)
				fmt.Println()
			}
			progress.SetText(event)
			if done := progress.Bar.Incr(); !done {
				break
			}
		}
		c.group.Done()
	}(progress)
}

func (c *ProgressCoordinator) AddEvent(progressName string, eventName string) {
	if progress, isPresent := c.progresses[progressName]; isPresent {
		progress.channel <- eventName
	}
}

func (c *ProgressCoordinator) Wait() {
	c.group.Wait()
	if isUiEnabled() {
		uiprogress.Stop()
	}
}
