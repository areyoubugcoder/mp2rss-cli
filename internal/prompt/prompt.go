// Package prompt provides interactive terminal input helpers.
//
// Secret() reads a line without echoing — used when a user pastes a Feed Key.
// Confirm() reads a y/N answer with a default.
package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Secret reads a single line from stdin without echoing. The trailing
// newline is stripped. When stdin is not a TTY, it falls back to a plain
// bufio read so piped/scripted use still works.
func Secret(prompt string, w io.Writer) (string, error) {
	if _, err := fmt.Fprint(w, prompt); err != nil {
		return "", err
	}
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		_, _ = fmt.Fprintln(w)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// Confirm prints prompt then reads y/n from stdin. Default is used on empty
// input. Non-TTY stdin returns the default.
func Confirm(prompt string, dflt bool, w io.Writer) (bool, error) {
	suffix := "[y/N]"
	if dflt {
		suffix = "[Y/n]"
	}
	fmt.Fprintf(w, "%s %s ", prompt, suffix)

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		fmt.Fprintln(w)
		return dflt, nil
	}
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return dflt, nil
	}
	return line == "y" || line == "yes", nil
}
