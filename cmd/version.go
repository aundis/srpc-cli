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
type Version struct {
}

func (v *Version) Name() string      { return "version" }
func (v *Version) Usage() string     { return "" }
func (v *Version) ShortHelp() string { return "print version" }
func (v *Version) DetailedHelp(f *flag.FlagSet) {
	fmt.Fprint(f.Output(), ``)
	f.PrintDefaults()
}

// Run prints Version information to stdout.
func (c *Version) Run(ctx context.Context, args ...string) error {
	fmt.Println("1.0.0")
	return nil
}
