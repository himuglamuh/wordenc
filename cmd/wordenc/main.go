package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"

	"github.com/himuglamuh/wordenc/codec"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]

	words, index, err := codec.LoadBIP39English()
	must(err)

	switch cmd {
	case "encode":
		fs := flag.NewFlagSet("encode", flag.ContinueOnError)
		rawEnc := fs.Bool("raw", false, "encode raw (no header) - default is framed")
		inFile := fs.String("in", "", "input file (encode: bytes) - default stdin")
		fs.SetOutput(io.Discard) // don't spam test output, we handle usage ourselves

		if err := fs.Parse(os.Args[2:]); err != nil {
			usage()
			os.Exit(2)
		}

		enc, err := codec.NewEncoder(words, !*rawEnc)
		must(err)

		if *inFile == "" && isInteractive() {
			fmt.Fprintln(os.Stderr, "[!] reading from stdin… type input, then press Enter to go to an empty line, then Ctrl-D (EOF)\n[!] or use -in <file>, or pipe: echo 'text' | wordenc encode")
		}

		data := readAllOrFile(*inFile)
		outWords := enc.Encode(data)
		fmt.Println(strings.Join(outWords, " "))

	case "decode":
		fs := flag.NewFlagSet("decode", flag.ContinueOnError)
		mode := fs.String("mode", "auto", "decode mode: auto|raw|framed")
		inFile := fs.String("in", "", "input file (decode: words) - default stdin")
		outFile := fs.String("out", "", "output file (decode: bytes) - default stdout")
		asHex := fs.Bool("hex", false, "decode output as hex to stdout (ignores -out)")
		strict := fs.Bool("strict", false, "strict padding + errors (decode)")
		maxLenMB := fs.Int("maxlenmb", 256, "max framed length in MB (decode auto/framed)")
		fs.SetOutput(io.Discard)

		if err := fs.Parse(os.Args[2:]); err != nil {
			usage()
			os.Exit(2)
		}

		dec := codec.NewDecoder(index)
		dec.StrictPad = *strict
		dec.AllowPartial = !*strict
		dec.MaxLen = uint64(*maxLenMB) << 20

		input := string(readAllOrFile(*inFile))
		m := parseMode(*mode)

		b, derr := dec.DecodeWords(input, m)

		if *strict && derr != nil {
			fmt.Fprintln(os.Stderr, "error:", derr)
			os.Exit(1)
		}
		if derr != nil {
			fmt.Fprintln(os.Stderr, "warning:", derr)
		}

		if *asHex {
			fmt.Println(hex.EncodeToString(b))
			return
		}
		if *outFile != "" {
			must(os.WriteFile(*outFile, b, 0o644))
			return
		}
		if utf8.Valid(b) {
			fmt.Print(string(b))
		} else {
			_, _ = os.Stdout.Write(b)
		}

	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `wordenc encode [--raw] [-in file] > words
wordenc decode [--mode auto|raw|framed] [--strict] [--hex] [-in file] [-out file]`)
}

func parseMode(s string) codec.DecodeMode {
	switch strings.ToLower(s) {
	case "auto":
		return codec.DecodeAuto
	case "raw":
		return codec.DecodeRaw
	case "framed":
		return codec.DecodeFramed
	default:
		return codec.DecodeAuto
	}
}

func readAllOrFile(path string) []byte {
	var r io.Reader = os.Stdin
	if path != "" {
		f, err := os.Open(path)
		must(err)
		defer f.Close()
		r = f
	}
	br := bufio.NewReader(r)
	b, err := io.ReadAll(br)
	must(err)
	return b
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
