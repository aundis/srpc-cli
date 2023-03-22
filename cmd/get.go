package cmd

// Copyright 2019 The Go Authors. All rights reserved.
// Use of this session code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"context"
	"flag"
	"fmt"
)

// Version implements the Version cmd.
type Get struct {
}

func (g *Get) Name() string      { return "get" }
func (g *Get) Usage() string     { return "[call/listen] [target[@object]]" }
func (g *Get) ShortHelp() string { return "get call or listen" }
func (g *Get) DetailedHelp(f *flag.FlagSet) {
	fmt.Fprint(f.Output(), `
[@object] if not set, get all object.
`)
	f.PrintDefaults()
}

// Run prints Version information to stdout.
func (g *Get) Run(_ context.Context, args ...string) error {
	// dir, err := os.Getwd()
	// if err != nil {
	// 	return err
	// }
	// if len(args) == 0 {
	// 	return errors.New("缺少参数")
	// }

	return nil
}
