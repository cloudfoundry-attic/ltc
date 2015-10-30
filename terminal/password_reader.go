package terminal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/pkg/term"
)

//go:generate counterfeiter -o mocks/fake_term.go . Term
type Term interface {
	SaveState(fd uintptr) (*term.State, error)
	RestoreTerminal(fd uintptr, state *term.State) error
	DisableEcho(fd uintptr, state *term.State) error
}

type FdReader interface {
	io.Reader
	Fd() uintptr
}

type TermPasswordReader struct {
	Term   Term
	Stdin  FdReader
	Stdout io.Writer
}

func NewPasswordReader() *TermPasswordReader {
	return &TermPasswordReader{
		Term:   &DockerTerm{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
	}
}

func (pr *TermPasswordReader) PromptForPassword(promptText string, args ...interface{}) string {
	fmt.Fprintf(pr.Stdout, promptText+": ", args...)

	originalState, err := pr.Term.SaveState(pr.Stdin.Fd())
	if err == nil {
		newState := *originalState
		if err := pr.Term.DisableEcho(pr.Stdin.Fd(), &newState); err == nil {
			defer pr.Term.RestoreTerminal(pr.Stdin.Fd(), originalState)
			defer fmt.Fprintln(pr.Stdout, "")
		}
	}

	line, err := bufio.NewReader(pr.Stdin).ReadString('\n')
	if err != nil {
		return ""
	}

	return strings.TrimSpace(line)
}
