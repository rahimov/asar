package main

import (
	"bytes"
	"fmt"
	"github.com/denisvmedia/asar"
	"github.com/krolaw/zipstream"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// This example shows how to convert .zip to .asar

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:\n\t./asarc <from.zip>")
		return
	}
	fileName := os.Args[1]

	entries := asar.Builder{}

	f, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	dirEntries := make(map[string]*asar.Entry)

	zr := zipstream.NewReader(f)
	for {
		entries.SetCurrent(entries.Root())

		fl, err := zr.Next()
		if err != nil {
			if err != io.EOF {
				panic(err)
			}
			break
		}
		fmt.Println(*fl)
		s2, err := ioutil.ReadAll(zr)

		dirs := make([]string, 0)
		dir, _ := path.Split(fl.Name)
		dir = strings.TrimRight(dir, "/")
		fmt.Println(fl.Name)
		for {
			if dir == "" || dir == "./" || dir == "." {
				break
			}
			dirs = append(dirs, dir)
			dir, _ = path.Split(dir)
			dir = strings.TrimRight(dir, "/")
		}
		sort.Sort(sort.Reverse(sort.StringSlice(dirs)))

		var actualDir string
		for _, dir := range dirs {
			if actualDir == "" {
				actualDir = dir
			}

			if entry, ok := dirEntries[actualDir]; !ok {
				fmt.Println("adding actual dir", actualDir)
				entries.AddDir(dir, asar.FlagDir)
				dirEntries[actualDir] = entries.Current()
			} else {
				fmt.Println("changing to dir", actualDir)
				entries.SetCurrent(entry)
			}

			actualDir = actualDir + "/" + dir
		}

		if fl.FileInfo().IsDir() {
			// entries.AddDir(fl.FileInfo().Name(), asar.FlagDir)
		} else {
			entries.Add(fl.FileInfo().Name(), bytes.NewReader(s2), int64(len(s2)), asar.FlagNone)
		}
	}

	f2, err := os.Create(fileName[:len(fileName) - len(filepath.Ext(fileName))]+".asar")
	if err != nil {
		panic(err)
	}
	defer f2.Close()

	_, err = entries.Root().EncodeTo(f2)
	if err != nil {
		panic(err)
	}
}
