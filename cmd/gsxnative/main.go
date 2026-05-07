package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gsxnative <init|compile|emit|check|build|dev|devices|store-plan|scene-conform> ...")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "init":
		err = runInit(os.Args[2:])
	case "compile":
		err = runCompile(os.Args[2:])
	case "emit":
		err = runEmit(os.Args[2:])
	case "check":
		err = runCheck(os.Args[2:])
	case "build":
		err = runBuild(os.Args[2:])
	case "dev":
		err = runDev(os.Args[2:])
	case "devices":
		err = runDevices(os.Args[2:])
	case "store-plan":
		err = runStorePlan(os.Args[2:])
	case "scene-conform":
		err = runSceneConform(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", os.Args[1])
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
