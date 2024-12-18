package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jessevdk/go-flags"
)

var opts struct {
	Help    bool `long:"help" short:"h"`
	Version bool `long:"version" short:"v"`
	Stdin   bool `long:"stdin" short:"x"`

	OutputDirectory string `long:"outdir" short:"d"`
}

// Populated by build system
var Version string = "pre0.1"

const (
	// Program executed successfully
	codeSuccess = 0
	// System error has occured
	codeSystem = 1
	// Invalid program usage
	codeUsage = 2
	// Error while parsing source file
	codeParser = 3
)

func main() {
	parser := flags.NewParser(&opts, flags.Default^flags.HelpFlag^flags.PrintErrors)
	args, err := parser.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse arguments: %s\n", err.Error())
		os.Exit(codeUsage)
	}

	if opts.Help {
		usage(os.Stdout)
		os.Exit(codeSuccess)
	}
	if opts.Version {
		version()
		os.Exit(codeSuccess)
	}

	if !opts.Stdin && len(args) != 1 || opts.Stdin && len(args) != 0 {
		usage(os.Stderr)
		os.Exit(codeUsage)
	}

	var inStream *os.File
	if opts.Stdin {
		inStream = os.Stdin
	} else {
		inStream, err = os.Open(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open source file: %s\n", err.Error())
			os.Exit(codeSystem)
		}
	}
	defer inStream.Close()

	var outDir string
	if opts.OutputDirectory == "" {
		outDir, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get working directory: %s\n", err.Error())
			os.Exit(codeSystem)
		}
	} else {
		outDir = opts.OutputDirectory
	}

	reader := bufio.NewReader(inStream)
	name, err := pageName(reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(codeParser)
	}

	// TODO: implemenet
	fmt.Printf("outDir: %v\n", outDir)
	fmt.Printf("name: %v\n", name)
}

func usage(toFile *os.File) {
	fmt.Fprintf(toFile, "usage: %s [options] <source>\n", os.Args[0])
	fmt.Fprintf(toFile, "\n")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-x, --stdin", "Read input file from stdin (source should be left empty)")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-d, --outdir <dir>", "Set output website directory (default: '.')")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-h, --help", "Show usage information")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-v, --version", "Show program version")
}

func version() {
	fmt.Printf("shac version %s\n", Version)
}

func pageName(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(line, "@page") {
		return "", errors.New("syntax error: no @page found at line 1")
	}

	parts := strings.Split(line, " ")
	if len(parts) < 2 {
		return "", errors.New("syntax error: no input given to @page")
	}

	name := strings.Join(parts[1:], " ")

	return strings.Trim(name, " \t\n"), nil
}
