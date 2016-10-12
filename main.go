/*

gomaker

Makefile generator for Go projects

Use like this:

    gomaker -o Makefile --options "static,lite,commit"

With the default settings, that same command is typed:

    gomaker .

To build with a literal "go build", `gomaker --options "none"`

Here are the options, they should be comma separated in the -options="" flag, inside double quotes.
	none: normal go build, Shell: go build
	verbose: verbose build, Shell: go build -x
	lite:  no debug symbols, Shell: --ldflags '-s'
	static: try making a static linked binary (no deps)
	commit: try adding version info into the mix
*/

// Gomaker is a Makefile generator for Go programs
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/aerth/filer"
)

var (
	version       = "(undefined version)"
	debug         = flag.Bool("debug", false, "Verbose logging to debug.log file")
	outputFile    = flag.String("o", "Makefile", "")
	versionNumber = flag.String("version", "", "Include Version (v4.3.2), will be prefixed to commit option.")
	options       = flag.String("options", "static,verbose,lite,commit", "Options. Use --options=\"comma,sep,list\"")
	optionhelp    = `

  [Options]
  none: normal go build, Shell: go build
  verbose: verbose build, Shell: go build -x
  lite:  no debug symbols, Shell: --ldflags '-s'
  static: try making a static linked binary (no deps)
  commit: try adding version info into the mix
`
)

func init() {
	usage := flag.Usage
	flag.Usage = func() {
		fmt.Println("[Gomaker] " + version + "\n")
		fmt.Println("Default:")
		fmt.Println("gomaker -o Makefile -options='static,verbose,lite,commit' .")
		fmt.Println()
		fmt.Println("Can be shortened to:")
		fmt.Println("gomaker .")
		fmt.Println()
		usage()
		fmt.Println(optionhelp)
		os.Exit(2)
	}
}
func main() {
	flag.Parse()
	if len(flag.Args()) > 1 {
		if strings.Contains(flag.Arg(1), "-") {
			fmt.Println("[ Error: Flags after directory name ]")
		} else {
			fmt.Println("[ Error: Too many arguments. Only need one. ]")
		}
		flag.Usage()
	}
	args := flag.Arg(0)
	if args == "." {
		args = os.Getenv("PWD")
	}

	dir, err := ioutil.ReadDir(args)
	if err != nil {
		flag.Usage()
		fatal(err)
	}

	// Check if we are a go directory real quick
	if !func() bool {
		for _, file := range dir {
			name := file.Name()
			if !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go") {
				return true // Is a go file
			}
		}
		return false
	}() {
		fmt.Println("Not a Go project directory.")
		flag.Usage()
		os.Exit(2)
	}

	// Create the Makefile
	fmt.Println("[Gomaker]" + version)
	fmt.Println("[Gomaker] Makefile:\n" + args + "/Makefile")
	fmt.Println("[Options]", *options)

	var linein = make(chan string)

	// Send things to linein
	go builder(linein, args)
	// Write lines to file
	writer(linein)
}

// Fatal Error
func fatal(ss ...interface{}) {
	if len(ss) == 1 {
		fmt.Println(ss)
	} else {
		fmt.Printf(ss[0].(string), ss[1:])
	}
	os.Exit(1)
}

// We need main or bust!
// Uses getOneGoFile() and gethead() to determine if this Go project is a "main" project.
func aMainPackage(directory string) bool {
	randomGoFileName := getOneGoFile(directory)
	if randomGoFileName == "" {
		fatal("Bug: no Go files at", directory)
		return false
	}
	pname := gethead(directory, randomGoFileName)
	if pname == "main" {
		return true
	}
	fatal("Not a main project: ", pname, directory+randomGoFileName)
	return false
}

// Since all proper Go source files have an uncommented package name, this works!
func getOneGoFile(dirname string) (goFileName string) {
	fileinfo, err := ioutil.ReadDir(dirname)
	if err != nil {
		fmt.Println("Gomaker:", err)
		os.Exit(0)
	}
	for _, file := range fileinfo {
		n := file.Name()
		if strings.HasSuffix(n, ".go") && !strings.HasPrefix(n, ".") {
			return n
		}
	}
	return ""
}

// Return the package name of a *.go file, (not with the word "package ")
func gethead(dir, goFilename string) string {
	if dir != "" {
		dir += "/"
	}
	file, err := os.Open(dir + goFilename)
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s := scanner.Text()
		if strings.HasPrefix(s, "package ") {
			return strings.Split(s, " ")[1]
		}
	}
	if scanner.Err() != nil {
		panic(scanner.Err())
	}
	return ""
}

// Listen on the linein chan, writing lines to Makefile.
func writer(linein chan string) {
	filer.Create(*outputFile)
	filer.Write(*outputFile, []byte(""))
	for {
		select {
		case line := <-linein:
			if line == "EOT" {
				fmt.Println("[Gomaker] Makefile generated.")
				close(linein)
				return
			}
			filer.Append(*outputFile, []byte(line+"\n"))
		}
	}
}

func builder(linein chan string, args string) {

	// Lib support soon come
	if !aMainPackage(args) {
		os.Exit(2)
	}

	// Since 'go build' uses the directory name as binary name, let's do the same.
	dirname := strings.Split(args, "/")
	projectName := dirname[len(dirname)-1]

	// Iterate options
	op := strings.Split(*options, ",")
	if len(op) == 1 && op[0] == "" {
		op = []string{"none"}
	}
	if strings.Contains(*options, "none") {
		op = []string{"none"}
	}

	// Start writing. This will block until we are ready to write.
	fmt.Println("[Project]", projectName)
	linein <- "# " + projectName
	linein <- "# " + "Makefile generated by Gomaker " + version

	linein <- "\n"
	linein <- "NAME=" + projectName
	if *versionNumber != "" {
		linein <- "VERSION = " + *versionNumber + "."
	} else {
		linein <- "VERSION="
	}
	linein <- "PREFIX ?= /usr/local/bin"
	var ldflags, buildflags string
	for _, option := range op {

		switch option {
		case "static":
			linein <- `export CGO_ENABLED=0`
		case "commit":
			linein <- `COMMIT=$(shell git rev-parse --verify --short HEAD)`
			linein <- `RELEASE=${VERSION}${COMMIT}`
			ldflags += `-X main.version=${RELEASE} ` // append ldflags
		case "lite":
			ldflags += `-s ` // append ldflags
		case "none":
			//buildstring = ""
		case "verbose":
			buildflags += "-x "
		default:
			fmt.Println("WARNING:", option, "is not a real option. Skipping.")
		}

	}

	if ldflags != "" {
		ldflags = "--ldflags " + strconv.Quote(ldflags)
	}
	linein <- "\n"
	linein <- "all:\tbuild"
	linein <- "\n"
	linein <- "build:"
	echoe := echoes(fmt.Sprintf("Building ${NAME} version ${RELEASE}"))
	linein <- "\t" + echoe

	linein <- "\tgo build -o ${NAME} " + buildflags + ldflags // build line
	echoe = echoes(fmt.Sprintf("Successfully built ${NAME}"))
	linein <- "\t" + echoe
	linein <- "\n"
	linein <- "install:"
	linein <- "\t" + echoes("PREFIX=${PREFIX}")
	linein <- "\t@mkdir -p ${PREFIX}"
	linein <- "\t@mv ${NAME} ${PREFIX}/${NAME}"
	echoe = echoes(fmt.Sprintf("Successfully installed ${NAME} to ${PREFIX}"))
	linein <- "\t" + echoe
	linein <- "EOT" // End of transmission.
}

// Single quote a string
func quote(s string) string {

	return `'` + s + `'`
}

func echoes(s string) string {
	var makelines string
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		if line != "" {
			newline := "@echo " + quote(line) + "\n"
			makelines += newline
		}
	}

	return makelines
}
