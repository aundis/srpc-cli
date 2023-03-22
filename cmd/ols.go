package cmd

// Copyright 2019 The Go Authors. All rights reserved.
// Use of this session code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"context"
	"errors"
	"flag"
	"fmt"
)

// Version implements the Version cmd.
type Ols struct {
}

func (o *Ols) Name() string      { return "ols" }
func (o *Ols) Usage() string     { return "ols [target] [object]" }
func (o *Ols) ShortHelp() string { return "object list" }
func (o *Ols) DetailedHelp(f *flag.FlagSet) {
	fmt.Fprint(f.Output(), ``)
	f.PrintDefaults()
}

// Run prints Version information to stdout.
func (c *Ols) Run(ctx context.Context, args ...string) error {
	if len(args) == 0 {
		return errors.New("缺少参数")
	}
	clinet, err := newSrpcClinet(ctx)
	if err != nil {
		return err
	}
	list, err := requestObjectMate(ctx, clinet, args[0], helperListReq{})
	if err != nil {
		return err
	}
	if len(list) > 0 {
		for _, v := range list {
			fmt.Printf("[%s] %s\n", v.Kind, v.Name)
		}
	}
	return nil
}
