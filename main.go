// aerth <aerth(at)riseup.net>
// copyright (c) 2016, 2017
// free open source (MIT)
// latest version: github.com/aerth/gomaker

// Gomaker is a Makefile generator for Go programs
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aerth/filer"
)

func init() {
	flag.StringVar(options, "o", "v,l,s,g", "Short for --options")  // o short option
	flag.StringVar(versionNumber, "ver", "", "Short for --version") // ver short version
}

var (
	version       = "(go get -v -x github.com/aerth/gomaker)"
	debug         = flag.Bool("debug", false, "Verbose logging to debug.log file")
	verbose       = flag.Bool("v", false, "Output to standard output, not Makefile. Use like 'gomaker -v > Makefile'")
	outputFile    = flag.String("out", "Makefile", "")
	versionNumber = flag.String("version", "", "Include Version (v4.3.2), will be prefixed to commit option.")
	options       = flag.String("options", "static,verbose,lite,gitcommit", "Options. Use --options=\"comma,sep,list\"")
	tagsIN        = flag.String("tag", "", "build -tags, for example: -tags='gtk,demo'")
	ldflagsIN     = flag.String("ldflags", "", "additional custom ldflags")
	sub           = flag.String("sub", "", "Substitute package variables (string only)")

	optionhelp = `
  [gomaker] ` + version + ` by <aerth>
  [Options] *Use first key*
  [n]one: normal go build, Shell: go build
  [v]erbose: verbose build, Shell: go build -x
  [l]ite:  no debug symbols, Shell: --ldflags '-s'
  [s]tatic: try making a static linked binary (no deps)
  [g]itcommit: try adding version info into the mix
  [+]cgo: set CGO_ENABLED=1
  [-]cgo: set CGO_ENABLED=0
  [e]rror: try to build with errors`
)

func init() {
	// redefine flag.Usage
	usage := flag.Usage
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "[Gomaker "+version+"]\n")
		fmt.Fprintln(os.Stderr, "Default:")
		fmt.Fprintln(os.Stderr, "gomaker -o Makefile -options='static,verbose,lite,gitcommit' .")
		fmt.Fprintln(os.Stderr, "gomaker -o Makefile -options='static,verbose,lite,gitcommit' ./cmd/name")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Can be shortened to:")
		fmt.Fprintln(os.Stderr, "gomaker .")
		fmt.Fprintln(os.Stderr, "")
		usage()
		fmt.Fprintln(os.Stderr, optionhelp)
	}
}
func main() {
	flag.Parse()
	if len(flag.Args()) > 1 {
		// bad flags
		if strings.Contains(flag.Arg(1), "-") {
			fmt.Fprintln(os.Stderr, "error: place flags before directory name")
		} else {
			flag.Usage()
			fmt.Fprintf(os.Stderr, "error: need only one argument. got %v.\n", len(flag.Args()))
		}
		os.Exit(2)
	}

	// no args
	projectDir := flag.Arg(0)

	// make 'gomaker -v > Makefile' possible
	if *verbose && projectDir == "" {
		projectDir = "."
	}

	// 'gomaker' or 'gomaker -o g,l,s,d' (no args)
	if projectDir == "" && !*verbose {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "Fatal: need go main project directory as argument. Try 'gomaker .'")
		os.Exit(2)
	}

	// standard usage
	// gomaker .

	if projectDir == "." {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		projectDir = strings.Replace(projectDir, "//", "/", -1)
	}

	dir, err := ioutil.ReadDir(projectDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Usage:")
		flag.Usage()
		fmt.Fprintln(os.Stderr, "Not a directory.", err.Error())
		os.Exit(1)
	}

	// Check if we are a go directory real quick
	// TODO: save this scan for later?
	if !func() bool {
		for _, file := range dir {
			name := file.Name()
			if !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go") {
				return true // Is a go file. Just need one.
			}
		}
		return false
	}() {
		fmt.Fprintln(os.Stderr, "Fatal: Not a Go project directory.")
		os.Exit(2)
	}

	// Create the Makefile
	fmt.Fprintln(os.Stderr, "[Gomaker v"+version+"]")
	fmt.Fprintf(os.Stderr, "[Makefile] %q\n", *outputFile)
	fmt.Fprintf(os.Stderr, "[Project Dir] %q\n", projectDir)

	var linein = make(chan string)
	// Send things to linein
	go builder(linein, projectDir)
	// Write lines to file
	writer(linein)
}

// Fatal Error
func fatal(ss ...interface{}) {
	if len(ss) == 1 {
		fmt.Fprintln(os.Stderr, ss)
	} else {
		fmt.Fprintln(os.Stderr, ss[0].(string), ss[1:])
	}
	os.Exit(2)
}

// We need main or bust!
// Uses getOneGoFile() and firstline() to determine if this Go project is a "main" project.
func aMainPackage(directory string) bool {
	randomGoFileName := getOneGoFile(directory)
	if randomGoFileName == "" {
		fatal("Bug: no Go files at", directory)
		return false
	}
	pname := firstline(directory, randomGoFileName)
	if pname == "main" {
		return true
	}
	fatal("Not a main project: ", directory)
	return false
}

// Since all proper Go source files have an uncommented package name, this works!
func getOneGoFile(dirname string) (goFileName string) {
	fmt.Fprintln(os.Stderr, dirname)
	abs, _ := filepath.Abs(dirname)
	fileinfo, err := ioutil.ReadDir(abs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Gomaker:", err)
		os.Exit(1)
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
// Doesn't necessarily return the first line of a file.
// Could be the last line, in the case of a doc.go file.
func firstline(dir, goFilename string) string {
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
			// Return the segment after package, before any comments.
			// TODO: There may be no space, such as: 'package main// comment' In that case we would not pick this up.
			return strings.Split(s, " ")[1]
		}
	}
	if scanner.Err() != nil {
		panic(scanner.Err())
	}

	// The file didn't have what we were looking for.
	return ""
}

// Listen on the linein chan, writing lines to Makefile.
func writer(linein chan string) {

	for {
		select {
		case line := <-linein:
			if line == "EOT" {
				fmt.Fprintln(os.Stderr, "[Gomaker] Makefile generated.")
				close(linein)
				return
			}
			if *verbose {
				fmt.Fprintln(os.Stdout, line)
			} else {
				filer.Append(*outputFile, []byte(line+"\n"))
			}
		}
	}
}

func builder(linein chan string, dir string) {

	if dir == "." {
		dir, _ = os.Getwd()
	}
	fmt.Println("dir:", dir)
	// non-main package support soon come
	if !aMainPackage(dir) {
		if !strings.Contains(strings.Join(os.Args, " "), "-f") {
			os.Exit(2)
		}
	}
	// Since 'go build' uses the directory name as binary name, let's do the same.
	dir = strings.TrimSuffix(dir, "/")
	dirname := strings.Split(dir, "/")
	projectName := dirname[len(dirname)-1]

	if strings.Trim(projectName, "") == "" {
		fmt.Fprint(os.Stderr, "No project name found.\n")
		os.Exit(1)
	}
	// Iterate options
	op := strings.Split(*options, ",")
	if len(op) == 1 && op[0] == "" {
		op = []string{"none"}
	}
	if strings.Contains(*options, "none") {
		op = []string{"none"}
	}

	if strings.Split(op[0], "")[0] == "n" {
		fmt.Fprintln(os.Stderr, "[no options]")
		op = nil
	}

	// Backup old Makefile to /tmp/gomaker-UnixTime
	b, e := ioutil.ReadFile(*outputFile)
	if e != nil {
		if !strings.Contains(e.Error(), "no such") {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
	}
	if len(b) > 0 {
		fmt.Fprintln(os.Stderr, "[Backup] Found existing", *outputFile)
		time := strconv.Itoa(int(time.Now().Unix()))
		bkup := os.TempDir() + "/gomaker-" + time
		e := ioutil.WriteFile(bkup, b, 0755)
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "[Backup] Saved to:", bkup)
	}

	// Blank file
	filer.Touch(*outputFile)
	filer.Write(*outputFile, []byte(""))

	// Start writing. This will block until we are ready to write.
	fmt.Fprintln(os.Stderr, "[Project Name]", projectName)
	linein <- "# " + projectName
	linein <- "# " + "Makefile generated by 'gomaker' " + version
	linein <- ""
	linein <- "NAME ?= " + projectName
	if *versionNumber != "" {
		linein <- "VERSION ?= " + *versionNumber + "."
	} else {
		linein <- "VERSION ?= "
	}
	linein <- "PREFIX ?= /usr/local/bin"
	var ldflags, buildflags, gcflags string
	if *ldflagsIN != "" {
		ldflags += *ldflagsIN
	}
	if *tagsIN != "" {
		buildflags += "-tags " + strconv.Quote(*tagsIN)
	}
	if op != nil {
		fmt.Fprintf(os.Stderr, "[Options] ")
		for i, optionstr := range op {

			option := []rune(optionstr)[0:1]
			fmt.Fprintf(os.Stderr, "%s", string(option))
			if i == len(op)-1 {
				fmt.Fprintf(os.Stderr, "\n")
			} else {
				fmt.Fprintf(os.Stderr, ",")
			}
			// additional ldflags and cgflags need extra space
			switch option[0] {
			case rune('s'):
				ldflags += `-extldflags='-static' `
			case rune('e'):
				gcflags += `-e `
			case rune('+'):
				linein <- `export CGO_ENABLED=1`
			case rune('-'):
				linein <- `export CGO_ENABLED=0`
			case rune('g'), rune('c'):
				// VER=38
				linein <- `VER ?= X`
				linein <- `COMMIT=$(shell git rev-parse --verify --short HEAD)`
				linein <- `COMMIT ?= ${VER}`
				linein <- `RELEASE ?= ${VERSION}${COMMIT}`
				ldflags += `-X main.version=${RELEASE} ` // append ldflags
			case rune('l'):
				ldflags += `-s ` // append ldflags
			case rune('n'):
				//buildstring = ""
			case rune('v'):
				buildflags += "-x "
			default:
				fmt.Fprintln(os.Stderr, "WARNING:", option, "is not a real option. Skipping.")
			}
		}
	}
	if *sub != "" {
		fmt.Fprintln(os.Stderr, "[Variable Substitutions]")
		for _, v := range strings.Split(*sub, ",") {
			if !strings.Contains(v, "=") {
				fmt.Fprintf(os.Stderr, "Variable substitution has no '=' sign.")
				os.Exit(1)
			}
			key, value :=
				func(s string) (string, string) {
					k := strings.Split(s, "=")
					return k[0], k[1]
				}(v)
			if key == "" || value == "" {
				fmt.Fprintf(os.Stderr, "Invalid variable substitution: %s=%s", key, value)
				os.Exit(1)
			}

			fmt.Fprintf(os.Stderr, "Defining variable %q as %q", key, value)
			ldflags += fmt.Sprintf(`-X %s=%s `, key, value)
		}
	}
	if ldflags != "" {
		ldflags = strings.Replace(strconv.Quote(ldflags), ` "`, `"`, -1)
		ldflags = "--ldflags " + ldflags
	}
	if gcflags != "" {
		gcflags = "--cgflags " + strconv.Quote(gcflags)
	}
	linein <- "\n"
	linein <- "all:\t${NAME}"
	linein <- "\n"
	linein <- "build:"
	echoe := Echo(fmt.Sprintf("Building ${NAME} version ${RELEASE}"))
	linein <- "\t" + echoe
	linein <- "\tgo get -d -x -v ."
	linein <- "\tgo build -o ${NAME} " + fmt.Sprint(buildflags, ldflags, gcflags) + " " + flag.Arg(0)
	echoe = Echo(fmt.Sprintf("Successfully built ${NAME}"))
	linein <- "\t" + echoe
	linein <- "\n"
	linein <- "${NAME}: build"
	linein <- "\n"
	linein <- "install:"
	linein <- "\t" + Echo("PREFIX=${PREFIX}")
	linein <- "\t@mkdir -p ${PREFIX}"
	linein <- "\t@mv ${NAME} ${PREFIX}/${NAME}"
	echoe = Echo(fmt.Sprintf("Successfully installed ${NAME} to ${PREFIX}"))
	linein <- "\t" + echoe

	linein <- "run:"
	linein <- "\tgo run -v -x $(shell ls *.go | grep -v _test.go)"
	linein <- "\n"
	linein <- "clean:"
	linein <- "\t@rm -v ${NAME}"

	linein <- "EOT" // End of transmission.
}

// Single quote a string
func quote(s string) string {
	return `'` + s + `'`
}

// Echo turns a (possibly multiline) string into a @echo 'line'
func Echo(s string) string {
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
