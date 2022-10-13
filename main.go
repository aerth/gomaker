package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/aerth/gomaker/assets"
)

var version string
var commit string

func main() {
	log.SetFlags(0)
	var run bool
	var showversion bool
	flag.BoolVar(&run, "run", false, "run 'make' after creating makefile")
	flag.BoolVar(&showversion, "v", false, "show version and quit")
	flag.Parse()
	log.Printf("%s version %s-%s", os.Args[0], version, commit)
	if showversion {
		os.Exit(1)
	}
	if flag.NArg() != 0 {
		log.Fatalln("no args required")
	}
	b := assets.MustAsset("makefile")
	_, err := os.Stat("makefile")
	if err == nil {
		log.Fatalln("fatal: makefile exists")
	}
	_, err = os.Stat("Makefile")
	if err == nil {
		log.Fatalln("fatal: Makefile exists")
	}
	err = ioutil.WriteFile("makefile", b, 0666)
	if err != nil {
		log.Fatalln("fatal:", err)
	}
	if run {
		cmd := exec.Command("make")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalln(err)
		}
	}

}
