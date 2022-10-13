// copyright (c) 2016, 2017, 2020 aerth
// free open source (MIT)
// latest version: github.com/aerth/gomaker

// Gomaker is a Makefile generator for Go programs
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var Version string
var Commit string

func main() {
	log.SetFlags(0)
	if Version == "" {
		Version = "v0.0.0"
	}
	if Commit == "" {
		Commit = "dev"
	}
	gomakerVersion := fmt.Sprintf("%s-%s", Version, Commit)
	log.Printf("Gomaker version %s", gomakerVersion)
	flag.Parse()
	version, err := Go("version")
	if err != nil {
		log.Fatalln("go not installed")
	}
	log.Println("Go version:", version)

	out, err := os.OpenFile("Makefile", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	out.Truncate(0)
	fmt.Fprintf(out, "# generated by gomaker %s\n# https://github.com/aerth/gomaker\n", gomakerVersion)
	var targets []Target
	cmdout, err := Go("list", "-f", "{{.Target}}")
	target := filepath.Base(strings.TrimSuffix(cmdout, "\n"))
	cmdout, err = Go("list", "-f", "{{.Name}}")
	pkgname := strings.TrimSuffix(cmdout, "\n")
	targetpath := ""
	cleanpaths := ""
	if pkgname != "main" {
		log.Fatalln("need main pkg for now")
	}
	fmt.Fprintf(out, "buildflags ?= -v -ldflags '-w -s'\n")
	fmt.Fprintf(out, "COMMIT=$(shell git rev-parse --verify --short HEAD 2>/dev/null)\n")
	fmt.Fprintf(out, "VERSION=$(shell git describe --tags 2>/dev/null)\n")
	fmt.Fprintf(out, "buildflags := $(buildflags) -ldflags '-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)'\n")

	// write target
	targets = append(targets, Target{target: target, targetpath: targetpath})

	for _, t := range targets {
		cleanpaths += " " + t.target
		writeTarget(out, t.target, t.targetpath)
	}

	fmt.Fprintf(out, "clean:\n\trm -vf %s\n.PHONY += clean\n", cleanpaths)

}

type Target struct {
	target     string
	targetpath string
}

func writeTarget(out io.Writer, target string, targetpath string) {
	log.Println("writing target:", target)
	fmt.Fprintf(out, "\n%s: %s*.go\n\tgo build $(buildflags) -o $@\n", target, targetpath)

}

func Go(args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func CmdOutput(args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(output), "\n"), nil
}
func MustOutput(args ...string) string {
	str, err := CmdOutput(args...)
	if err != nil {
		log.Fatalln(err)
	}
	return str
}
