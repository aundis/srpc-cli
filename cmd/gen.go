package cmd

// Copyright 2019 The Go Authors. All rights reserved.
// Use of this session code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sr/emit"
)

// Version implements the Version cmd.
type Gen struct {
}

func (c *Gen) Name() string      { return "gen" }
func (c *Gen) Usage() string     { return "" }
func (c *Gen) ShortHelp() string { return "build call signal or slot" }
func (c *Gen) DetailedHelp(f *flag.FlagSet) {
	fmt.Fprint(f.Output(), ``)
	f.PrintDefaults()
}

// Run prints Version information to stdout.
func (c *Gen) Run(_ context.Context, args ...string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.New("缺少参数")
	}
	switch args[0] {
	case "slot":
		err = emit.EmitSlot(dir)
	case "signal":
		err = emit.EmitSignal(dir)
	case "call":
		err = emit.EmitCall(dir)
	default:
		return errors.New("参数必须为 slot signal 或 call")
	}
	if err != nil {
		return err
	}
	return nil
}
