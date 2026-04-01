package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/alchemy/clc/format"
)

func main() {
	var from, to, output string
	flag.StringVar(&from, "from", "", "source format (auto-detected from file extension if omitted)")
	flag.StringVar(&from, "f", "", "source format (shorthand)")
	flag.StringVar(&to, "to", "", "target format (required)")
	flag.StringVar(&to, "t", "", "target format (shorthand)")
	flag.StringVar(&output, "output", "", "output file (default: stdout)")
	flag.StringVar(&output, "o", "", "output file (shorthand)")

	flag.Usage = func() {
		names := format.Names()
		sort.Strings(names)
		fmt.Fprintf(os.Stderr, "Usage: clc [flags] [file]\n\n")
		fmt.Fprintf(os.Stderr, "Convert between configuration file formats.\n")
		fmt.Fprintf(os.Stderr, "Supported formats: %s\n\n", strings.Join(names, ", "))
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if to == "" {
		fatal("--to/-t is required")
	}

	var r io.Reader
	if flag.NArg() > 0 {
		filename := flag.Arg(0)
		f, err := os.Open(filename)
		if err != nil {
			fatal("%v", err)
		}
		defer f.Close()
		r = f

		if from == "" {
			detected, err := format.Detect(filename)
			if err != nil {
				fatal("%v", err)
			}
			from = detected
		}
	} else {
		if from == "" {
			fatal("--from/-f is required when reading from stdin")
		}
		r = os.Stdin
	}

	decoder, err := format.Get(from)
	if err != nil {
		fatal("%v", err)
	}
	encoder, err := format.Get(to)
	if err != nil {
		fatal("%v", err)
	}

	data, err := decoder.Decode(r)
	if err != nil {
		fatal("decode %s: %v", from, err)
	}

	var w io.Writer = os.Stdout
	if output != "" {
		f, err := os.Create(output)
		if err != nil {
			fatal("%v", err)
		}
		defer f.Close()
		w = f
	}

	if err := encoder.Encode(w, data); err != nil {
		fatal("encode %s: %v", to, err)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "clc: "+format+"\n", args...)
	os.Exit(1)
}
