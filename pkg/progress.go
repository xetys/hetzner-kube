package pkg

import "github.com/gosuri/uiprogress"

// Progress define the progress on command execution
type Progress struct {
	Name    string
	Bar     *uiprogress.Bar
	channel chan string
	State   string
}

// SetText define text to display during progress
func (progress *Progress) SetText(text string) {
	if text != "" {
		progress.State = text
	}
}
