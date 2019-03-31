package phases

import (
	"log"
)

type Phase interface {
	ShouldRun() bool
	Run() error
}

type PhaseChain struct {
	phases   []Phase
	afterRun func()
}

func NewPhaseChain() *PhaseChain {
	return &PhaseChain{
		phases: []Phase{},
	}
}

func (chain *PhaseChain) AddPhase(phase Phase) {
	chain.phases = append(chain.phases, phase)
}

func (chain *PhaseChain) SetAfterRun(fun func()) {
	chain.afterRun = fun
}

func (chain *PhaseChain) Run() error {
	for _, phase := range chain.phases {
		if phase.ShouldRun() {
			err := phase.Run()

			if err != nil {
				return err
			}

			chain.afterRun()
		}
	}

	return nil
}

// FatalOnError is an helper function to transform error to fatl
func FatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
