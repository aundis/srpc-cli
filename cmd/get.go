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
	"strings"
)

// Version implements the Version cmd.
type Get struct {
}

func (g *Get) Name() string      { return "get" }
func (g *Get) Usage() string     { return "[call/listen] [target[@object]]" }
func (g *Get) ShortHelp() string { return "get call or listen from remote service" }
func (g *Get) DetailedHelp(f *flag.FlagSet) {
	fmt.Fprint(f.Output(), `
[@object] if not set, get all object.
`)
	f.PrintDefaults()
}

// Run prints Version information to stdout.
func (g *Get) Run(ctx context.Context, args ...string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.New("缺少参数")
	}
	if len(args) != 2 {
		return errors.New("argument error, e.g. [call/listen] [target[@object]]")
	}
	var target, object string
	kind := args[0]
	if kind == "call" {
		if strings.Contains(args[1], "@") {
			arr := strings.Split(args[1], "@")
			target = arr[0]
			object = arr[1]
		} else {
			target = args[0]
		}

		clinet, err := newSrpcClinet(ctx)
		if err != nil {
			return err
		}
		list, err := requestObjectMeta(ctx, clinet, target, helperListReq{Kind: "slot", Name: object})
		if err != nil {
			return err
		}
		if len(list) > 0 {
			for _, v := range list {
				err = emit.EmitCallInterfaceFromHelper(dir, target, &v)
				if err != nil {
					return err
				}
				// fmt.Printf("get call %s declaration\n", v.Name)
			}
		}
	} else if kind == "listen" {
		if !strings.Contains(args[1], "@") {
			return errors.New("argument error, e.g. target@object")
		}
		arr := strings.Split(args[1], "@")
		target = arr[0]
		object = arr[1]
		clinet, err := newSrpcClinet(ctx)
		if err != nil {
			return err
		}
		list, err := requestObjectMeta(ctx, clinet, target, helperListReq{Kind: "signal", Name: object})
		if err != nil {
			return err
		}
		if len(list) == 0 {
			return fmt.Errorf("not fount object %s from %s", object, target)
		}
		if len(list) > 1 {
			return fmt.Errorf("match object to more")
		}
		ometa := list[0]
		err = emit.EmitListenFromHelper(dir, target, &ometa)
		if err != nil {
			return err
		}
	} else {
		return errors.New("kind error, kind must be call or listen")
	}

	return nil
}
