package main

import (
	"fmt"

	"github.com/fosskers/active/releases"
)

func main() {
	releases.Recent("fosskers", "aura")
	fmt.Println("Done.")
}
