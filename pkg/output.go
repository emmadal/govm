package pkg

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	greenAnsi  = "\033[32m"
	yellowAnsi = "\033[33m"
	blueAnsi   = "\033[34m"
	redAnsi    = "\033[31m"
	resetAnsi  = "\033[0m"
)

// Standard streams, replaceable in tests.
var (
	Stdin  io.Reader = os.Stdin
	Stdout io.Writer = os.Stdout
	Stderr io.Writer = os.Stderr
)

// colorEnabled reports whether w is a terminal and NO_COLOR is unset.
func colorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

func paint(w io.Writer, color, text string) string {
	if !colorEnabled(w) {
		return text
	}
	return color + text + resetAnsi
}

// Println prints an uncolored line to stdout.
func Println(format string, a ...any) {
	_, _ = fmt.Fprintln(Stdout, fmt.Sprintf(format, a...))
}

// Info prints a progress line in blue to stdout.
func Info(format string, a ...any) {
	_, _ = fmt.Fprintln(Stdout, paint(Stdout, blueAnsi, fmt.Sprintf(format, a...)))
}

// Success prints a line in green to stdout.
func Success(format string, a ...any) {
	_, _ = fmt.Fprintln(Stdout, paint(Stdout, greenAnsi, fmt.Sprintf(format, a...)))
}

// Warn prints a line in yellow to stderr.
func Warn(format string, a ...any) {
	_, _ = fmt.Fprintln(Stderr, paint(Stderr, yellowAnsi, fmt.Sprintf(format, a...)))
}

// PrintError prints an error in red to stderr.
func PrintError(err error) {
	_, _ = fmt.Fprintln(Stderr, paint(Stderr, redAnsi, "Error: "+err.Error()))
}

// Green returns text colored green when stdout is a terminal.
func Green(text string) string {
	return paint(Stdout, greenAnsi, text)
}

// Confirm asks a yes/no question on stdout and reads the answer from stdin.
// Anything other than "y" or "yes" counts as no.
func Confirm(question string) (bool, error) {
	_, _ = fmt.Fprintf(Stdout, "%s [y/N]: ", question)
	reply, err := bufio.NewReader(Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("failed to read input: %w", err)
	}
	reply = strings.ToLower(strings.TrimSpace(reply))
	return reply == "y" || reply == "yes", nil
}
