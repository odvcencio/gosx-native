package main

import (
	"fmt"
	"os"

	"github.com/odvcencio/gosx-native/emit/ios"
)

func runEmit(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: gsxnative emit <ios|android> <file.swift.gsx>")
	}
	target, file := args[0], args[1]
	mod, err := compileSwift(file)
	if err != nil {
		return err
	}
	switch target {
	case "ios":
		return ios.Emit(mod, os.Stdout)
	default:
		return fmt.Errorf("unknown target: %s (M1 supports: ios)", target)
	}
}
