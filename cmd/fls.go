package cmd

// Copyright 2019 The Go Authors. All rights reserved.
// Use of this session code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/aundis/meta"
)

// Version implements the Version cmd.
type Fls struct {
}

func (f *Fls) Name() string      { return "fls" }
func (f *Fls) Usage() string     { return "[target@object]" }
func (f *Fls) ShortHelp() string { return "list remote service object functionss" }
func (f *Fls) DetailedHelp(fla *flag.FlagSet) {
	fmt.Fprint(fla.Output(), ``)
	fla.PrintDefaults()
}

// Run prints Version information to stdout.
func (f *Fls) Run(ctx context.Context, args ...string) error {
	if len(args) < 1 {
		return errors.New("missing argument [target@object]")
	}
	arr := strings.Split(args[0], "@")
	if len(arr) != 2 {
		return errors.New("argument error, e.g. clinet@Person")
	}
	target := arr[0]
	objectName := arr[1]
	clinet, err := newSrpcClinet(ctx)
	if err != nil {
		return err
	}
	list, err := requestObjectMeta(ctx, clinet, target, helperListReq{Name: objectName})
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

func printFunction(fmeta *meta.FunctionMeta) {
	fmt.Print(fmeta.Name, " (")
	for i, p := range fmeta.Parameters {
		if i != 0 {
			fmt.Print(", ")
		}
		fmt.Print(p.Name, " ")
		fmt.Print(p.Type)
	}
	fmt.Print(") ")
	if len(fmeta.Results) > 1 {
		fmt.Print("(")
	}
	for i, r := range fmeta.Results {
		if i != 0 {
			fmt.Print(", ")
		}
		if len(r.Name) > 0 {
			fmt.Print(r.Name, " ")
		}
		fmt.Print(r.Type)
	}
	if len(fmeta.Results) > 1 {
		fmt.Print(")")
	}
	fmt.Print("\n")
}
