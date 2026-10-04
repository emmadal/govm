package main

import (
	"os"

	"github.com/emmadal/govm/cmd"
	"github.com/emmadal/govm/pkg"
)

func main() {
	if err := cmd.Execute(); err != nil {
		pkg.PrintError(err)
		os.Exit(1)
	}
}
