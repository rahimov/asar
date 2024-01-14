package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/krolaw/zipstream"

	"github.com/rahimov/asar"
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
		if fl.FileInfo().IsDir() {
			fmt.Println("Dir entry (skipped):", *fl)
			continue
		}

		fmt.Println("File entry:", *fl)
		s2, err := io.ReadAll(zr)
		if err != nil {
			panic(err)
		}

		dirs := make([]string, 0)
		components := strings.Split(fl.Name, "/")
		for _, dir := range components[:len(components)-1] {
			dirs = append(dirs, dir)
		}

		var actualDir string
		for _, dir := range dirs {
			if actualDir == "" {
				actualDir = dir
			} else {
				actualDir = actualDir + "/" + dir
			}

			if entry, ok := dirEntries[actualDir]; !ok {
				fmt.Println("Adding dir:", actualDir)
				entries.AddDir(dir, asar.FlagDir)
				dirEntries[actualDir] = entries.Current()
			} else {
				fmt.Println("Changing to dir:", actualDir)
				entries.SetCurrent(entry)
			}
		}

		fname := fl.FileInfo().Name()
		fmt.Println("Adding file:", fname)
		entries.Add(fname, bytes.NewReader(s2), int64(len(s2)), asar.FlagNone, "", nil)
	}

	f2, err := os.Create(fileName[:len(fileName)-len(filepath.Ext(fileName))] + ".asar")
	if err != nil {
		panic(err)
	}
	defer f2.Close()

	_, err = entries.Root().EncodeTo(f2)
	if err != nil {
		panic(err)
	}
}
