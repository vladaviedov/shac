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

	OutputDirectory string `long:"outdir" short:"d" default:"."`
	AssetDirectory  string `long:"assetdir" short:"a" default:"assets"`
	RootURL         string `long:"root" short:"r"`
}

// Populated by build system
var Version string = "0.3.0"

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

// Asset placeholder pattern
const assetPattern = `@\d+@`

// Root placeholder pattern
const rootPattern = `@\$@`

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

	// Default root to output dir
	if opts.RootURL == "" {
		opts.RootURL = opts.OutputDirectory
	}

	// Create asset directory
	assetDir := filepath.Join(opts.OutputDirectory, opts.AssetDirectory)
	err = os.MkdirAll(assetDir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create asset directory: %s\n", err.Error())
		os.Exit(codeSystem)
	}

	// Check input file header
	reader := bufio.NewReader(inStream)
	name, err := pageName(reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(codeParser)
	}

	// Process all asset tags
	assets, err := processAssets(assetDir, reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(codeParser)
	}

	// Create output document
	err = finalizeDocument(filepath.Join(opts.OutputDirectory, name), reader, assets, assetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create output file: %s\n", err.Error())
		os.Exit(codeSystem)
	}
}

func usage(toFile *os.File) {
	fmt.Fprintf(toFile, "usage: %s [options] <source>\n", os.Args[0])
	fmt.Fprintf(toFile, "\n")
	fmt.Fprintf(toFile, "%-25s - %s\n", "-x, --stdin", "Read input file from stdin (source should be left empty)")
	fmt.Fprintf(toFile, "%-25s - %s\n", "-d, --outdir <dir>", "Set output website directory (default: '.')")
	fmt.Fprintf(toFile, "%-25s - %s\n", "-r, --root <url>", "Set root URL (default: output directory)")
	fmt.Fprintf(toFile, "%-25s - %s\n", "-a, --assetdir <dir>", "Set asset subdirectory name (default: 'assets')")
	fmt.Fprintf(toFile, "%-25s - %s\n", "-h, --help", "Show usage information")
	fmt.Fprintf(toFile, "%-25s - %s\n", "-v, --version", "Show program version")
}

func version() {
	fmt.Printf("shac version %s\n", Version)
}

func pageName(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}

	// Skip document if @ignore present at the start
	if strings.HasPrefix(line, "@ignore") {
		os.Exit(codeSuccess)
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

func finalizeDocument(path string, r *bufio.Reader, assets []string, assetDir string) error {
	inputDoc, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	assetDoc := replaceAssetPlaceholders(inputDoc, assets, assetDir)
	finalDoc := replaceRootPlaceholders(assetDoc)

	return os.WriteFile(path, finalDoc, 0644)
}

func replaceAssetPlaceholders(input []byte, assets []string, assetDir string) []byte {
	reg, err := regexp.Compile(assetPattern)
	if err != nil {
		panic(err)
	}

	return reg.ReplaceAllFunc(input, func(placeholder []byte) []byte {
		indexStr := placeholder[1:(len(placeholder) - 1)]
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
		builder.WriteString(filepath.Join(assetDir, assets[index]))
		return []byte(builder.String())
	})
}

func replaceRootPlaceholders(input []byte) []byte {
	reg, err := regexp.Compile(rootPattern)
	if err != nil {
		panic(err)
	}

	return reg.ReplaceAll(input, []byte(opts.RootURL))
}
