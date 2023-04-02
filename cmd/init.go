package cmd

// Copyright 2019 The Go Authors. All rights reserved.
// Use of this session code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"regexp"
	"sr/util"
	"strings"

	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/os/gres"
)

// Version implements the Version cmd.
type Init struct {
	variable map[string]string
}

func (i *Init) Name() string      { return "init" }
func (i *Init) Usage() string     { return "" }
func (i *Init) ShortHelp() string { return "init srpc to goframe project" }
func (i *Init) DetailedHelp(f *flag.FlagSet) {
	fmt.Fprint(f.Output(), ``)
	f.PrintDefaults()
}

// Run prints Version information to stdout.
func (i *Init) Run(ctx context.Context, args ...string) error {
	i.variable = map[string]string{}

	// gres.Dump()
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	// get module name
	module, err := util.GetProjectModuleName(root)
	if err != nil {
		return err
	}
	i.variable["module-name"] = module
	err = i.build(root)
	if err != nil {
		return err
	}
	return nil
}

func (i *Init) build(root string) (err error) {
	files := gres.ScanDir("project/", "*", true)
	for _, f := range files {
		out := path.Join(root, strings.Replace(f.Name(), "project/", "", 1))
		// t.Log(out)
		if f.FileInfo().IsDir() {
			err = gfile.Mkdir(out)
			if err != nil {
				return err
			}
		} else {
			outFileName := i.replaceGoFileName(out)
			content := i.replaceVariables(gres.GetContent(f.Name()))
			err = util.WriteGenerateFile(outFileName, content, root)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *Init) replaceVariables(content []byte) []byte {
	reg := regexp.MustCompile(`{{.+?}}`)
	str := string(content)
	out := reg.ReplaceAllStringFunc(str, func(s string) string {
		s2 := strings.ReplaceAll(s, "{{", "")
		s2 = strings.ReplaceAll(s2, "}}", "")
		if len(i.variable[s2]) > 0 {
			return i.variable[s2]
		}
		return s
	})
	return []byte(out)
}

func (i *Init) replaceGoFileName(name string) string {
	if util.StringEndOf(name, ".go.txt") {
		return strings.ReplaceAll(name, ".go.txt", ".go")
	}
	return name
}
