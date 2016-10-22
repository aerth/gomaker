/*

gomaker

Makefile generator for Go projects

Use like this:

    gomaker -o Makefile --options "static,lite,commit,verbose"

With the default settings, that same command is typed:

    gomaker .

To build with a simple "go build", use `gomaker --options "none" .`

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
	"time"

	"github.com/aerth/filer"
	clr "github.com/daviddengcn/go-colortext"
)

var (
	version       = "(undefined version)"
	debug         = flag.Bool("debug", false, "Verbose logging to debug.log file")
	outputFile    = flag.String("o", "Makefile", "")
	versionNumber = flag.String("version", "", "Include Version (v4.3.2), will be prefixed to commit option.")
	options       = flag.String("options", "static,verbose,lite,commit", "Options. Use --options=\"comma,sep,list\"")
	tagsIN        = flag.String("tags", "", "build -tags, for example: -tags='gtk,demo'")
	ldflagsIN     = flag.String("ldflags", "", "additional custom ldflags")
	optionhelp    = `

  [Options]
  none: normal go build, Shell: go build
  verbose: verbose build, Shell: go build -x
  lite:  no debug symbols, Shell: --ldflags '-s'
  static: try making a static linked binary (no deps)
  commit: try adding version info into the mix`
)

func init() {
	usage := flag.Usage
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "[Gomaker] "+version+"\n")
		fmt.Fprintln(os.Stderr, "Default:")
		fmt.Fprintln(os.Stderr, "gomaker -o Makefile -options='static,verbose,lite,commit' .")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Can be shortened to:")
		fmt.Fprintln(os.Stderr, "gomaker .")
		fmt.Fprintln(os.Stderr, "")
		usage()
		fmt.Fprintln(os.Stderr, optionhelp)
		fmt.Fprintln(os.Stderr, "")
	}
}
func main() {

	flag.Parse()
	if len(flag.Args()) > 1 {
		if strings.Contains(flag.Arg(1), "-") {
			fmt.Fprintln(os.Stderr, "Fatal:  Flags after directory name")
		} else {
			fmt.Fprintln(os.Stderr, "Fatal: Too many arguments, only need one")
		}
		os.Exit(2)
	}
	args := flag.Arg(0)
	if args == "" {
		fmt.Fprintln(os.Stderr, "Fatal: Need Go project as argument")
		os.Exit(2)
	}
	if args == "." {
		args = os.Getenv("PWD")
	}

	dir, err := ioutil.ReadDir(args)
	if err != nil {
		fatal(err)
	}

	// Check if we are a go directory real quick
	// TODO: save this scan for later?
	if !func() bool {
		for _, file := range dir {
			name := file.Name()
			if !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go") {
				return true // Is a go file
			}
		}
		return false
	}() {
		fmt.Fprintln(os.Stderr, "Fatal: Not a Go project directory.")
		os.Exit(2)
	}

	// Green means GO!
	clr.ChangeColor(clr.Black, true, clr.Green, true)
	defer func() {
		clr.ResetColor()
		fmt.Println()
		fmt.Println()
	}()
	// Create the Makefile
	fmt.Println("[Gomaker] " + version)
	fmt.Println("[Makefile] " + args + "/Makefile")
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

	// Backup old Makefile to /tmp/gomaker-UnixTime
	b, e := ioutil.ReadFile(*outputFile)
	if e != nil {
		if !strings.Contains(e.Error(), "no such") {
			panic(e)
		}
	}
	if len(b) > 0 {
		fmt.Println("[Backup] Found existing", *outputFile)
		time := strconv.Itoa(int(time.Now().Unix()))
		bkup := os.TempDir() + "/gomaker-" + time
		e := ioutil.WriteFile(bkup, b, 0755)
		if e != nil {
			clr.ResetColor()
			panic(e)
		}
		fmt.Println("[Backup] Saved to:", bkup)
	}

	// Blank file
	filer.Touch(*outputFile)
	filer.Write(*outputFile, []byte(""))

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
	if *ldflagsIN != "" {
		ldflags += *ldflagsIN
	}
	if *tagsIN != "" {
		buildflags += "-tags " + strconv.Quote(*tagsIN)
	}
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
	echoe := Echo(fmt.Sprintf("Building ${NAME} version ${RELEASE}"))
	linein <- "\t" + echoe

	linein <- "\tgo build -o ${NAME} " + buildflags + ldflags // build line
	echoe = Echo(fmt.Sprintf("Successfully built ${NAME}"))
	linein <- "\t" + echoe
	linein <- "\n"
	linein <- "install:"
	linein <- "\t" + Echo("PREFIX=${PREFIX}")
	linein <- "\t@mkdir -p ${PREFIX}"
	linein <- "\t@mv ${NAME} ${PREFIX}/${NAME}"
	echoe = Echo(fmt.Sprintf("Successfully installed ${NAME} to ${PREFIX}"))
	linein <- "\t" + echoe

	linein <- "run:"
	linein <- "\t CGO_ENABLED=1 go run -v -x *.go"
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
