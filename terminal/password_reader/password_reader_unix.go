// Copied from https://code.google.com/p/gopass/

// +build darwin freebsd linux netbsd openbsd
package password_reader

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func (pr passwordReader) PromptForPassword(promptText string, args ...interface{}) (passwd string) {
	// Display the prompt.
	fmt.Printf(promptText+": ", args...)

	passwd = readPassword()

	fmt.Println("") // Carriage return after the user input.

	return
}

func readPassword() string {
	rd := bufio.NewReader(os.Stdin)

	line, err := rd.ReadString('\n')
	if err == nil {
		return strings.TrimSpace(line)
	}
	return ""
}
