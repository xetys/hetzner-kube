package addons

import "log"

//FatalOnError is an helper function used to transfor error to fatal
func FatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
