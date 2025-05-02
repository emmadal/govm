package pkg

import (
	"fmt"
	"os"
)

const (
	GreenAnsi = "\033[32m"
	ResetAnsi = "\033[0m"
	BlueAnsi  = "\033[34m"
	RedAnsi   = "\033[31m"
	BlackAnsi = "\033[30m"
)

// RedPrintln prints text in red to the console
func RedPrintln(text string) {
	_, _ = fmt.Fprint(os.Stdout, RedAnsi+text+ResetAnsi)
}

// GreenPrintln prints text in green to the console
func GreenPrintln(text string) {
	_, _ = fmt.Fprint(os.Stdout, GreenAnsi+text+ResetAnsi)
}

// BlackPrintln prints text in black to the console
func BlackPrintln(text string) {
	_, _ = fmt.Fprint(os.Stdout, BlackAnsi+text+ResetAnsi)
}

// TextGreen returns text in green
func TextGreen(text string) string {
	return fmt.Sprintf("%s%s%s", GreenAnsi, text, ResetAnsi)
}

// TextBlue returns text in blue
func TextBlue(text string) string {
	return fmt.Sprintf("%s%s%s", BlueAnsi, text, ResetAnsi)
}

// TextRed returns text in red
func TextRed(text string) string {
	return fmt.Sprintf("%s%s%s", RedAnsi, text, ResetAnsi)
}

// TextBlack returns text in black
func TextBlack(text string) string {
	return fmt.Sprintf("%s%s%s", BlackAnsi, text, ResetAnsi)
}
