package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sr/cmd"
	"strings"
	"text/template"
)

var commands = []Application{
	&cmd.Gen{},
	&cmd.Ols{},
	&cmd.Fls{},
}

func main() {
	//defer func() {
	//	if err := recover(); err != nil {
	//		fmt.Println(err)
	//	}
	//}()

	// First argument is current working directory
	if len(os.Args) <= 1 || os.Args[1] == "help" {
		printUsage()
		os.Exit(2)
	}

	ctx := context.Background()
	name, args := os.Args[1], os.Args[2:]
	for _, c := range commands {
		if c.Name() == name {
			Main(ctx, c, args)
			return
		}
	}
	fmt.Printf("未知命令 '%s', 使用命令 'sr help' 获取帮助信息。", name)
}

var usageTemplate = `
使用方法:
        sr <命令> [参数]

命令列表:{{range .}}
	{{.Name | printf "%-11s"}} {{.ShortHelp}}{{end}}

使用 "sr <命令> -help" 可获取有关命令的更多信息。
`

// An errWriter wraps a writer, recording whether a write error occurred.
type errWriter struct {
	w   io.Writer
	err error
}

func (w *errWriter) Write(b []byte) (int, error) {
	n, err := w.w.Write(b)
	if err != nil {
		w.err = err
	}
	return n, err
}

// tmpl executes the given template text on data, writing the result to w.
func tmpl(w io.Writer, text string, data interface{}) {
	t := template.New("top")
	t.Funcs(template.FuncMap{"trim": strings.TrimSpace})
	template.Must(t.Parse(text))
	ew := &errWriter{w: w}
	err := t.Execute(ew, data)
	if ew.err != nil {
		// I/O error writing. Ignore write on closed pipe.
		if strings.Contains(ew.err.Error(), "pipe") {
			os.Exit(1)
		}
		fmt.Errorf("writing output: %v", ew.err)
	}
	if err != nil {
		panic(err)
	}
}

func printUsage() {
	var bw = bufio.NewWriter(os.Stdout)
	tmpl(bw, usageTemplate, commands)
	bw.Flush()
}
