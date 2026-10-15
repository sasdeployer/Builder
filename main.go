package main

import (
	"Builder/cmd"
	"Builder/gui"
	"Builder/output"
	"Builder/utils"
	"fmt"
	"os"

	"go.uber.org/zap"
)

var BuilderLog = zap.S()

func main() {

	if len(os.Args) > 1 {
		utils.Help()
		builderCommand := os.Args[1]
		if builderCommand == "init" {
			cmd.Init()
			done()
		} else if builderCommand == "config" {
			cmd.Config()
			done()
		} else if builderCommand == "docker" {
			cmd.Docker()
			done()
		} else if builderCommand == "gui" {
			gui.Gui()
		} else {
			cmd.Builder()
			done()
		}
	} else {
		cmd.Builder()
		done()
	}
}

// done reports a finished build: one JSON result for machine callers
// (--json / BUILDER_OUTPUT=json), the usual message otherwise.
func done() {
	if output.JSON() {
		output.Success()
		return
	}
	fmt.Println("Build Complete 🔨")
}
