package phases

import (
	"log"
)

// Phase defines an interface for a generic phase
type Phase interface {
	ShouldRun() bool
	Run() error
}

// PhaseChain is a holder of several phases and a after run step
type PhaseChain struct {
	phases   []Phase
	afterRun func()
}

// NewPhaseChain creates a new instance of *PhaseChain
func NewPhaseChain() *PhaseChain {
	return &PhaseChain{
		phases: []Phase{},
	}
}

// AddPhase adds a new phase to the chain
func (chain *PhaseChain) AddPhase(phase Phase) {
	chain.phases = append(chain.phases, phase)
}

// SetAfterRun configures the after run function
func (chain *PhaseChain) SetAfterRun(fun func()) {
	chain.afterRun = fun
}

// Run starts the chain and collects errors
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

// FatalOnError is an helper function to print out an error and exit
func FatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
