package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os/user"

	"github.com/Pallinder/go-randomdata"
)

func randomName() string {
	return fmt.Sprintf("%s-%s%s", randomdata.Adjective(), randomdata.Noun(), randomdata.Adjective())
}

// FatalOnError is an helper function to transform error to fatl
func FatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// GetHome is an helper function to get current home directory
func GetHome() string {
	usr, _ := user.Current()
	return usr.HomeDir
}

// Dump is handy dumps of structure as json (almost any structures)
func Dump(cls interface{}) {
	data, err := json.MarshalIndent(cls, "", "    ")
	if err != nil {
		log.Println("[ERROR] Oh no! There was an error on Dump command: ", err)
		return
	}
	fmt.Println(string(data))
}

// Sdump is same as Dump, only output to a string
func Sdump(cls interface{}) string {
	data, err := json.MarshalIndent(cls, "", "    ")
	if err != nil {
		log.Println("[ERROR] Oh no! There was an error on Dump command: ", err)
		return ""
	}
	return fmt.Sprintln(string(data))
}
