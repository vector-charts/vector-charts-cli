package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/vector-charts/vector-charts-cli/internal/output"
	"golang.org/x/term"
)

// Prompt asks for a line of input on stderr (so stdout stays clean for pipes).
func Prompt(label string) (string, error) {
	output.Errf("%s", label)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// PromptSecret asks for a hidden password when stdin is a TTY.
func PromptSecret(label string) (string, error) {
	output.Errf("%s", label)
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		var pw string
		_, err := fmt.Fscanln(os.Stdin, &pw)
		return pw, err
	}
	b, err := term.ReadPassword(fd)
	output.Errf("\n")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
