package main

import (
	"fmt"
	"os"

	"github.com/odvcencio/gosx-native/emit/android"
	"github.com/odvcencio/gosx-native/emit/ios"
	"github.com/odvcencio/gosx-native/target"
)

func runEmit(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: gsxnative emit <ios|android> <file.gsx|file.swift.gsx>")
	}
	targetName, file := args[0], args[1]
	tgt, err := target.Parse(targetName)
	if err != nil {
		return err
	}
	mod, err := compileFile(file)
	if err != nil {
		return err
	}
	if err := validateNativeImplementationsFile(file, tgt); err != nil {
		return err
	}
	if err := target.Validate(mod, tgt); err != nil {
		return err
	}
	switch tgt {
	case "android":
		return android.Emit(mod, os.Stdout)
	case "ios":
		return ios.Emit(mod, os.Stdout)
	default:
		return fmt.Errorf("unknown target: %s (supported: ios, android)", targetName)
	}
}
