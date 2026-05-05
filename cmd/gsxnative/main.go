package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gsxnative <compile|emit> ...")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "compile":
		err = runCompile(os.Args[2:])
	case "emit":
		err = runEmit(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", os.Args[1])
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
