package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/alchemy/clc/format"
)

func main() {
	var from, to, output string
	flag.StringVarP(&from, "from", "f", "", "source format (auto-detected from file extension if omitted)")
	flag.StringVarP(&to, "to", "t", "", "target format (required)")
	flag.StringVarP(&output, "output", "o", "", "output file (default: stdout)")

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

	doc, err := decoder.Decode(r)
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

	if err := encoder.Encode(w, doc); err != nil {
		fatal("encode %s: %v", to, err)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "clc: "+format+"\n", args...)
	os.Exit(1)
}
