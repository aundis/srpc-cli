package parse

import (
	"fmt"
	"testing"
)

func TestParseInterface(t *testing.T) {
	f, err := ParseContent("test.go", []byte(`
	package main
	
	type IPerson interface {
		SayHello(foo string) error
	}
	`))
	if err != nil {
		t.Error(err)
		return
	}
	if len(f.InterfaceTypes) != 1 {
		t.Errorf("except interface quantity = 1, but got %d", len(f.InterfaceTypes))
		return
	}
	interfaceType := f.InterfaceTypes[0]
	if len(interfaceType.Functions) != 1 {
		t.Errorf("except interface function quantity = 1, bug got %d", len(interfaceType.Functions))
		return
	}
	function := f.InterfaceTypes[0].Functions[0]
	if len(function.Params) != 1 {
		t.Errorf("except funtion param len = 1, bug got %d", len(function.Params))
		return
	}
	param := function.Params[0]
	if param.Name != "foo" {
		t.Errorf("except funtion param name = foo, but got %s", param.Name)
		return
	}
	if param.Type != "string" {
		t.Errorf("except function param type = string, bug got %s", param.Type)
		return
	}
	if len(function.Results) != 1 {
		t.Errorf("except funtion result len = 1, bug got %d", len(function.Results))
		return
	}
	result := function.Results[0]
	if result.Name != "" {
		t.Errorf("except funtion result name empty, bug got %s", result.Name)
		return
	}
	if result.Type != "error" {
		t.Errorf("except funtion result type error, bug got %s", result.Type)
		return
	}
}

func TestParseStruct(t *testing.T) {
	f, err := ParseContent("test.go", []byte(`
	package main
	
	type Foo struct {
	}
	`))
	if err != nil {
		t.Error(err)
		return
	}
	if len(f.StructTypes) != 1 {
		t.Errorf("except struct type count = 1, but got %d", len(f.StructTypes))
		return
	}
	structType := f.StructTypes[0]
	if structType.Name != "Foo" {
		t.Errorf("except struct name = Foo, but got %s", structType.Name)
		return
	}
}

func TestParseFunction(t *testing.T) {
	f, err := ParseContent("test.go", []byte(`
	package main
	
	func Foo() {}
	func (* Person) Bar() {}
	`))
	if err != nil {
		t.Error(err)
		return
	}

	if len(f.Functions) != 2 {
		t.Errorf("except funtion count = 2, bug got %d", len(f.Functions))
		return
	}
	fun1 := f.Functions[0]
	if fun1.Name != "Foo" {
		t.Errorf("except function[0].Name=Foo, but got %s", fun1.Name)
		return
	}
	if len(fun1.Params) != 0 {
		t.Errorf("except funtion[0] param count = 0, but got %d", len(fun1.Params))
		return
	}
	if len(fun1.Results) != 0 {
		t.Errorf("except funtion[0] result count = 0, but got %d", len(fun1.Results))
		return
	}
	fun2 := f.Functions[1]
	if fun2.Name != "Bar" {
		t.Errorf("except function[1].Name=Bar, but got %s", fun1.Name)
		return
	}
	if fun2.RecvTypeName != "Person" {
		t.Errorf("except funtion[1] revc type name = Person, but got %s", fun2.RecvTypeName)
		return
	}
	if len(fun2.Params) != 0 {
		t.Errorf("except funtion[1] param count = 0, but got %d", len(fun1.Params))
		return
	}
	if len(fun2.Results) != 0 {
		t.Errorf("except funtion[1] result count = 0, but got %d", len(fun1.Results))
		return
	}
}

func TestParseContent(t *testing.T) {
	f, err := ParseContent("test.go", []byte(`
	package main

	import (
		"context"
		"encoding/json"
		"github.com/aundis/srpc"
		_ "abc/internal/model"
		d12 "abc/internal/service"
	)

	type One struct {
		srpc.Listen `+"`target:\"object\"`"+`
	}
	`))
	if err != nil {
		t.Error(err)
		return
	}
	except := []Import{
		{
			Name:   "",
			Path:   "context",
			Export: "context",
		},
		{
			Name:   "",
			Path:   "encoding/json",
			Export: "json",
		},
		{
			Name:   "",
			Path:   "github.com/aundis/srpc",
			Export: "srpc",
		},
		{
			Name:   "_",
			Path:   "abc/internal/model",
			Export: "_",
		},
		{
			Name:   "d12",
			Path:   "abc/internal/service",
			Export: "d12",
		},
	}

	if len(except) != len(f.Imports) {
		t.Errorf("except import count = %d, but got %d", len(except), len(f.Imports))
		return
	}
	for i := range f.Imports {
		if except[i].Name != f.Imports[i].Name {
			t.Errorf("except import[%d].Name = %s, bug got %s", i, except[i].Name, f.Imports[i].Name)
			return
		}
		if except[i].Path != f.Imports[i].Path {
			t.Errorf("except import[%d].Path = %s, bug got %s", i, except[i].Path, f.Imports[i].Path)
			return
		}
		if except[i].Export != f.Imports[i].Export {
			t.Errorf("except import[%d].Export = %s, bug got %s", i, except[i].Export, f.Imports[i].Export)
			return
		}
	}
}

func TestParseFunctionType(t *testing.T) {
	f, err := ParseContent("test.go", []byte(`
	package main

	type person struct {
		callback func (ctx context.Context, a, b int) (error)
	}
	`))
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(f)
}
