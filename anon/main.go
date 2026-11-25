package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
)

const usage = `anon - anonymise columns of a delimited file

usage: anon [-c config.json] [-o out.csv] [-i in.csv] [-salt-env VAR]

Reads CSV from stdin (or -i) and writes to stdout (or -o).
The hash salt is taken from the config, or from the environment variable
named by -salt-env (default ANON_SALT) when the config has none.
`

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("anon", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage); fs.PrintDefaults() }
	cfgPath := fs.String("c", "config.json", "config file")
	outPath := fs.String("o", "", "output file (default stdout)")
	inPath := fs.String("i", "", "input file (default stdin)")
	saltEnv := fs.String("salt-env", "ANON_SALT", "environment variable holding the salt")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}

	cfg, err := LoadConfig(*cfgPath, os.Getenv(*saltEnv))
	if err != nil {
		fmt.Fprintln(stderr, "anon:", err)
		return 2
	}
	in := stdin
	if *inPath != "" {
		f, err := os.Open(*inPath)
		if err != nil {
			fmt.Fprintln(stderr, "anon:", err)
			return 1
		}
		defer f.Close()
		in = f
	}
	out := stdout
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintln(stderr, "anon:", err)
			return 1
		}
		defer f.Close()
		out = f
	}
	bw := bufio.NewWriter(out)
	if err := Process(cfg, bufio.NewReader(in), bw); err != nil {
		fmt.Fprintln(stderr, "anon:", err)
		return 1
	}
	if err := bw.Flush(); err != nil {
		fmt.Fprintln(stderr, "anon:", err)
		return 1
	}
	return 0
}
