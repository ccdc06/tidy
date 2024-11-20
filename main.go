package main

import (
	"os"

	"github.com/fatih/color"
)

func main() {
	color.Green("(Press Ctrl+C to exit at any time)")

	done := false
	var err error
	for {
		if done {
			break
		}
		Hr()
		options := map[string]string{"v": "Verify existing, deleted and unknown cbz files", "d": "Download, update and delete paired yaml files", "n": "Nothing"}

		answer := ReadOptions("What would you like to do?", options)

		switch answer {
		case "v":
			err = verifyFiles()
			done = true

		case "d":
			err = updateYamlFiles()
			done = true

		case "n":
			color.Green("Bye then :)")
			done = true
		}
	}

	if err == nil {
		color.Green("Execution complete. Press Enter to exit.")
		ScanLine()
		os.Exit(0)
	} else {
		color.Yellow("Execution complete (error: %s). Press Enter to exit.", err)
		ScanLine()
		os.Exit(1)
	}
}
