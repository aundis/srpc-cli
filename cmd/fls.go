package cmd

// Copyright 2019 The Go Authors. All rights reserved.
// Use of this session code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/aundis/mate"
)

// Version implements the Version cmd.
type Fls struct {
}

func (o *Fls) Name() string      { return "fls" }
func (o *Fls) Usage() string     { return "" }
func (o *Fls) ShortHelp() string { return "object function list" }
func (o *Fls) DetailedHelp(f *flag.FlagSet) {
	fmt.Fprint(f.Output(), ``)
	f.PrintDefaults()
}

// Run prints Version information to stdout.
func (f *Fls) Run(ctx context.Context, args ...string) error {
	if len(args) != 2 {
		return errors.New("缺少参数")
	}
	target := args[0]
	objectName := args[1]
	clinet, err := newSrpcClinet(ctx)
	if err != nil {
		return err
	}
	list, err := requestObjectMate(ctx, clinet, target, helperListReq{Name: objectName})
	if err != nil {
		return err
	}
	if len(list) > 0 {
		object := list[0]
		for _, f := range object.Functions {
			printFunction(f)
		}
	} else {
		return fmt.Errorf("not found object %s", objectName)
	}
	return nil
}

func printFunction(fmate *mate.FunctionMate) {
	fmt.Print(fmate.Name, " (")
	for i, p := range fmate.Parameters {
		if i != 0 {
			fmt.Print(", ")
		}
		fmt.Print(p.Name, " ")
		fmt.Print(p.Type)
	}
	fmt.Print(") ")
	if len(fmate.Results) > 1 {
		fmt.Print("(")
	}
	for i, r := range fmate.Results {
		if i != 0 {
			fmt.Print(", ")
		}
		if len(r.Name) > 0 {
			fmt.Print(r.Name, " ")
		}
		fmt.Print(r.Type)
	}
	if len(fmate.Results) > 1 {
		fmt.Print(")")
	}
	fmt.Print("\n")
}
