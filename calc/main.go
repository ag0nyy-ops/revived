// Command calc is a small command-line calculator.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ag0nyy-ops/revived/calc/internal/expr"
)

const usage = `usage: calc [expression...]
       calc            read expressions from stdin (interactive on a terminal)

Operators: + - * / % ^ !   Parentheses and implicit multiplication (3pi, 2(1+2)).
Names:     pi e tau phi, ans or @ (last result), @@ (one before), x = 3 to assign.
Functions: sin cos tan asin acos atan sinh cosh tanh sqrt cbrt log lg ln exp
           abs floor ceil round trunc pow mod atan2 hypot min max sum avg
`

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

func run(args []string, in io.Reader, out, errw io.Writer) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprint(out, usage)
		return 0
	}
	env := expr.NewEnv()
	if len(args) > 0 {
		v, err := env.Eval(strings.Join(args, " "))
		if err != nil {
			fmt.Fprintln(errw, "calc:", err)
			return 1
		}
		fmt.Fprintln(out, expr.Format(v))
		return 0
	}
	interactive := isTerminal(in)
	sc := bufio.NewScanner(in)
	status := 0
	for {
		if interactive {
			fmt.Fprint(out, "> ")
		}
		if !sc.Scan() {
			break
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == "exit" || line == "quit" {
			break
		}
		v, err := env.Eval(line)
		if err != nil {
			fmt.Fprintln(errw, "error:", err)
			status = 1
			continue
		}
		fmt.Fprintln(out, expr.Format(v))
	}
	if interactive {
		fmt.Fprintln(out)
	}
	return status
}

func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}
