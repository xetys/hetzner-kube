package cmd

import (
	"fmt"
	"log"

	"github.com/Pallinder/go-randomdata"
)

func randomName() string {
	return fmt.Sprintf("%s-%s%s", randomdata.Adjective(), randomdata.Noun(), randomdata.Adjective())
}

//FatalOnError is an helper function to transform error to fatl
func FatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
