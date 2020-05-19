package utils

import (
	"fmt"
	"os"
)

// Exit the program with an appropriate status code if our `error` value was
// `nil`.
func ExitIfErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func PrintExit(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}
