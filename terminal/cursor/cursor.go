package cursor

import "fmt"
import "os"

const csi = "\033["

func Up(lines int) string {
	if os.Getenv("TERM") == "" {
		return ""
	} else {
		return fmt.Sprintf("%s%dA", csi, lines)
	}
}

func ClearToEndOfLine() string {
	if os.Getenv("TERM") == "" {
		return ""
	} else {
		return fmt.Sprintf("%s%dK", csi, 0)
	}
}

func ClearToEndOfDisplay() string {
	if os.Getenv("TERM") == "" {
		return ""
	} else {
		return fmt.Sprintf("%s%dJ", csi, 0)
	}
}

func Show() string {
	if os.Getenv("TERM") == "" {
		return ""
	} else {
		return csi + "?25h"
	}
}

func Hide() string {
	if os.Getenv("TERM") == "" {
		return ""
	} else {
		return csi + "?25l"
	}
}
