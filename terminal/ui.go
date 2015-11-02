package terminal

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

//go:generate counterfeiter -o mocks/fake_password_reader.go . PasswordReader
type PasswordReader interface {
	PromptForPassword(promptText string, args ...interface{}) string
}

type UI interface {
	io.ReadWriter
	PasswordReader

	Prompt(promptText string, args ...interface{}) string
	PromptWithDefault(promptText, defaultValue string, args ...interface{}) string
	Say(format string, args ...interface{})
	SayIncorrectUsage(message string)
	SayLine(format string, args ...interface{})
	SayNewLine()
}

type terminalUI struct {
	io.Reader
	io.Writer
	PasswordReader
}

func NewUI(input io.Reader, output io.Writer, passwordReader PasswordReader) UI {
	return &terminalUI{
		input,
		output,
		passwordReader,
	}
}

func (t *terminalUI) Prompt(promptText string, args ...interface{}) string {
	reader := bufio.NewReader(t)
	fmt.Fprintf(t.Writer, promptText+": ", args...)

	result, _ := reader.ReadString('\n')
	return strings.TrimSpace(result)
}

func (t *terminalUI) PromptWithDefault(promptText, defaultValue string, args ...interface{}) string {
	reader := bufio.NewReader(t)
	fmt.Fprintf(t.Writer, promptText+fmt.Sprintf(" [%s]: ", defaultValue), args...)

	result, _ := reader.ReadString('\n')
	result = strings.TrimSpace(result)

	if result == "" {
		return defaultValue
	}

	return result
}

func (t *terminalUI) Say(format string, args ...interface{}) {
	t.say(format, args...)
}

func (t *terminalUI) SayIncorrectUsage(message string) {
	if len(message) > 0 {
		t.say("Incorrect Usage: %s\n", message)
	} else {
		t.say("Incorrect Usage\n")
	}
}

func (t *terminalUI) SayLine(format string, args ...interface{}) {
	t.say(format+"\n", args...)
}

func (t *terminalUI) SayNewLine() {
	t.say("\n")
}

func (t *terminalUI) say(format string, args ...interface{}) {
	if len(args) > 0 {
		t.Write([]byte(fmt.Sprintf(format, args...)))
		return
	}
	t.Write([]byte(format))
}
