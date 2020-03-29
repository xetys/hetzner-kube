package pkg

import (
	"fmt"
	"os"
	"sync"

	"github.com/go-kit/kit/log/term"
	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uiprogress/util/strutil"
)

// CompletedEvent indicate the process completed
const CompletedEvent = "complete!"

// UIProgressCoordinator coortinate display of progress in UI
type UIProgressCoordinator struct {
	debug      bool
	group      sync.WaitGroup
	progresses map[string]*Progress
}

// NewProgressCoordinator create a new progress coordinator UI
func NewProgressCoordinator(debug bool) *UIProgressCoordinator {
	pc := UIProgressCoordinator{
		progresses: make(map[string]*Progress),
		debug:      debug,
	}

	if pc.isUIEnabled() {
		uiprogress.Start()
	}

	return &pc
}

func (c *UIProgressCoordinator) isUIEnabled() bool {
	if c.debug {
		return term.IsTerminal(os.Stdout)
	}

	return false
}

func shortLeftPadRight(s string, padWidth int) string {
	if len(s) > padWidth {
		l := len(s)
		return "..." + s[(l-(padWidth-2)):(l-1)]
	}

	return strutil.PadRight(s, padWidth, ' ')
}

// StartProgress start the progress UI
func (c *UIProgressCoordinator) StartProgress(name string, steps int) {
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

			if !c.isUIEnabled() {
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

// AddEvent add an new event in the progress UI
func (c *UIProgressCoordinator) AddEvent(progressName string, eventName string) {
	if progress, isPresent := c.progresses[progressName]; isPresent {
		progress.channel <- eventName
	}
}

// CompleteProgress sends an completed event
func (c *UIProgressCoordinator) CompleteProgress(nodeName string) {
	if progress, isPresent := c.progresses[nodeName]; isPresent {
		progress.channel <- CompletedEvent
	}
}

// Wait temporary stop the progress UI
func (c *UIProgressCoordinator) Wait() {
	c.group.Wait()

	if c.isUIEnabled() {
		uiprogress.Stop()
	}
}
