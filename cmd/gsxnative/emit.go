package main

import (
	"fmt"
	"os"

	"github.com/odvcencio/gosx-native/emit/android"
	"github.com/odvcencio/gosx-native/emit/ios"
)

func runEmit(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: gsxnative emit <ios|android> <file.gsx|file.swift.gsx>")
	}
	target, file := args[0], args[1]
	mod, err := compileFile(file)
	if err != nil {
		return err
	}
	switch target {
	case "android":
		return android.Emit(mod, os.Stdout)
	case "ios":
		return ios.Emit(mod, os.Stdout)
	default:
		return fmt.Errorf("unknown target: %s (supported: ios, android)", target)
	}
}
