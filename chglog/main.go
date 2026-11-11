// Command chglog generates a CHANGELOG from Conventional Commits and git tags.
package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "chglog:", err)
		os.Exit(1)
	}
}
