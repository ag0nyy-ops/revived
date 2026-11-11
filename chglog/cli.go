package main

import (
	"flag"
	"fmt"
	"io"
)

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("chglog", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Fprintln(stdout, "chglog", version)
		return nil
	}
	return fmt.Errorf("not implemented yet")
}
