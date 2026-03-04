package main

import (
	"os"

	"github.com/jasonmay/bsg/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
