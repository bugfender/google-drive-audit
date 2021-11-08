package util

import "log"

func PrintIfError(f func() error) {
	err := f()
	if err != nil {
		log.Println(err)
	}
}
