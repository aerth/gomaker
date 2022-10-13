package main

import (
	"flag"
	"gomaker/assets"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

func main() {
	log.SetFlags(0)
	var run bool
	flag.BoolVar(&run, "run", false, "run 'make' after creating makefile")
	flag.Parse()
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
