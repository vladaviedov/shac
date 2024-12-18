package main

import (
	"fmt"
	"os"

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

func main() {
	parser := flags.NewParser(&opts, flags.Default^flags.HelpFlag^flags.PrintErrors)
	args, err := parser.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse arguments: %s\n", err.Error())
		os.Exit(2)
	}

	if opts.Help {
		usage(os.Stdout)
		os.Exit(0)
	}
	if opts.Version {
		version()
		os.Exit(0)
	}

	if !opts.Stdin && len(args) != 1 || opts.Stdin && len(args) != 0 {
		usage(os.Stderr)
		os.Exit(2)
	}

	var outDir string
	if opts.OutputDirectory == "" {
		outDir, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "system error: failed to get working directory: %s\n", err.Error())
			os.Exit(1)
		}
	} else {
		outDir = opts.OutputDirectory
	}

	// TODO: implemenet
	fmt.Printf("outDir: %v\n", outDir)
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
