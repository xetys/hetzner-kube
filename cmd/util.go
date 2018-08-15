package cmd

import (
	"fmt"
	"log"

	"github.com/Pallinder/go-randomdata"
)

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
