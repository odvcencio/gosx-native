package main

import (
	"fmt"

	"github.com/odvcencio/gosx-native/target"
)

func runCheck(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: gsxnative check <ios|android> <file.gsx|file.swift.gsx>")
	}
	tgt, err := target.Parse(args[0])
	if err != nil {
		return err
	}
	mod, err := compileFile(args[1])
	if err != nil {
		return err
	}
	return target.Validate(mod, tgt)
}
