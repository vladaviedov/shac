package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
var Version string = "0.1.0"

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

// Placeholder asset paths
const placeholderPattern = `"@\d@"`

// Path to asset dir from output dir
// TODO: convert into argument?
const pathToAssets = "assets"

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

	assetDir := filepath.Join(outDir, pathToAssets)
	err = os.MkdirAll(assetDir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create asset directory: %s\n", err.Error())
		os.Exit(codeSystem)
	}

	reader := bufio.NewReader(inStream)
	name, err := pageName(reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(codeParser)
	}

	assets, err := processAssets(assetDir, reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(codeParser)
	}

	err = finalizeDocument(filepath.Join(outDir, name), reader, assets)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create output file: %s\n", err.Error())
		os.Exit(codeSystem)
	}
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

func processAssets(assetDir string, r *bufio.Reader) ([]string, error) {
	var assets []string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.Trim(line, " \t\n")
		if line == "@html" {
			break
		}

		if !strings.HasPrefix(line, "@asset") {
			return nil, errors.New("syntax error: invalid directive")
		}

		parts := strings.Split(line, " ")
		if len(parts) < 2 {
			return nil, errors.New("syntax error: no input given to @asset")
		}

		name := strings.Join(parts[1:], " ")
		assetHash, err := createAsset(assetDir, name)
		if err != nil {
			return nil, err
		}

		assets = append(assets, assetHash)
	}

	return assets, nil
}

func createAsset(assetDir string, path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	rawHash := sha1.Sum(content)
	hash := hex.EncodeToString(rawHash[:])

	assetPath := filepath.Join(assetDir, hash)
	asset, err := os.Create(assetPath)

	_, err = asset.Write(content)
	if err != nil {
		return "", err
	}

	return hash, nil
}

func finalizeDocument(path string, r *bufio.Reader, assets []string) error {
	reg, err := regexp.Compile(placeholderPattern)
	if err != nil {
		panic(err)
	}

	inputDoc, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	doc := reg.ReplaceAllFunc(inputDoc, func(placeholder []byte) []byte {
		indexStr := placeholder[2:(len(placeholder) - 2)]
		index, err := strconv.Atoi(string(indexStr))

		// Regex should ensure that number can be parsed
		if err != nil {
			panic(err)
		}

		// Out of bounds - do nothing
		if index >= len(assets) {
			return placeholder
		}

		builder := new(strings.Builder)
		builder.WriteString("\"")
		builder.WriteString(filepath.Join(pathToAssets, assets[index]))
		builder.WriteString("\"")
		return []byte(builder.String())
	})

	return os.WriteFile(path, doc, 0644)
}
