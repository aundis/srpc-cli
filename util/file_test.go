package util

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/gogf/gf/v2/os/gfile"
)

type testIsGenerateFileExcept struct {
	path   string
	except bool
}

func TestIsGenerateFile(t *testing.T) {
	var excepts = []testIsGenerateFileExcept{
		{
			path:   "testdata/is_gen/a.txt",
			except: true,
		},
		{
			path:   "testdata/is_gen/b.txt",
			except: false,
		},
	}

	for _, e := range excepts {
		is, err := IsGenerateFile(e.path)
		if err != nil {
			t.Error(err)
			return
		}
		if is != e.except {
			t.Errorf("except testdata/isgen/a.txt IsGenerateFile = %t, but got %t", e.except, is)
			return
		}
	}
}

func TestRemoveGenFiles(t *testing.T) {
	err := gfile.Remove("testdata/remove_gen_files")
	if err != nil {
		t.Error(err)
		return
	}
	err = gfile.Mkdir("testdata/remove_gen_files")
	if err != nil {
		t.Error(err)
		return
	}
	// cerate test files
	ioutil.WriteFile("testdata/remove_gen_files/a.txt", []byte(GeneratedHeader), os.ModePerm)
	ioutil.WriteFile("testdata/remove_gen_files/b.txt", []byte("hello"), os.ModePerm)
	// remove gen files
	err = RemoveGenerateFiles("testdata/remove_gen_files")
	if err != nil {
		t.Error(err)
		return
	}
	// check result
	files, err := ListFile("testdata/remove_gen_files")
	if err != nil {
		t.Error(err)
		return
	}
	if len(files) != 1 {
		t.Errorf("except files count =  1, but got %d", len(files))
		return
	}
}
