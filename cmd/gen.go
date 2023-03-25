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

func (g *Gen) Name() string      { return "gen" }
func (g *Gen) Usage() string     { return "[call/slot/signal]" }
func (g *Gen) ShortHelp() string { return "generate call signal or slot" }
func (g *Gen) DetailedHelp(f *flag.FlagSet) {
	fmt.Fprint(f.Output(), ``)
	f.PrintDefaults()
}

// Run prints Version information to stdout.
func (g *Gen) Run(_ context.Context, args ...string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.New("missing augument [call/slot/signal]")
	}
	if len(args) > 1 {
		return errors.New("augument too more")
	}
	switch args[0] {
	case "slot":
		err = emit.EmitSlot(dir)
	case "signal":
		err = emit.EmitSignal(dir)
	case "call":
		err = emit.EmitCall(dir)
	default:
		return fmt.Errorf("argument %s not in [call/slot/signal]", args[0])
	}
	if err != nil {
		return err
	}
	return nil
}
