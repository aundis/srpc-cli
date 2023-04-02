package cmd

import (
	"testing"

	_ "sr/packed"
	"sr/util"
)

func TestInit(t *testing.T) {
	cmd := &Init{
		variable: map[string]string{},
	}

	root := "C:\\Users\\85124\\Desktop\\test11"
	// get module name
	module, err := util.GetProjectModuleName(root)
	if err != nil {
		t.Error(err)
		return
	}
	cmd.variable["module-name"] = module
	err = cmd.build(root)
	if err != nil {
		t.Error(err)
		return
	}
}
