package pkg

import (
	"fmt"
	"github.com/go-kit/kit/log/term"
	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uiprogress/util/strutil"
	"os"
	"sync"
)

const CompletedEvent = "complete!"

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
func shortLeftPadRight(s string, padWidth int) string {
	if len(s) > padWidth {
		l := len(s)
		return "..." + s[(l-(padWidth-2)):(l-1)]
	} else {
		return strutil.PadRight(s, padWidth, ' ')
	}
}
func (c *ProgressCoordinator) StartProgress(name string, steps int) {
	progress := &Progress{
		Bar:     uiprogress.AddBar(steps),
		State:   "starting",
		channel: make(chan string),
		Name:    name,
	}
	progress.Bar.Width = 16
	progress.Bar.PrependFunc(func(b *uiprogress.Bar) string {
		percent := strutil.PadLeft(fmt.Sprintf("%.01f%%", b.CompletedPercent()), 6, ' ')
		return fmt.Sprintf("%s : %s  %s",
			shortLeftPadRight(name, 20),
			shortLeftPadRight(progress.State, 32),
			percent,
		)
	})
	c.progresses[name] = progress
	c.group.Add(1)
	go func(progress *Progress) {
		for {
			event := <-progress.channel
			if !isUiEnabled() {
				fmt.Printf("%s: %s (%d)", progress.Name, event, progress.Bar.Current()+1)
				fmt.Println()
			}
			if event == CompletedEvent {
				progress.Bar.Set(progress.Bar.Total)
				progress.SetText(event)
				break
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
