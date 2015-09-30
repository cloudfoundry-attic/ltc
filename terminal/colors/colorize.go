package colors

import (
	"fmt"
)

//TODO: remove Color prefix from color constants
const (
	// ColorRed string = "\x1b[91m"
	ColorRed     string = ""
	ColorCyan    string = ""
	ColorGreen   string = ""
	ColorYellow  string = ""
	ColorDefault string = ""
	ColorBold    string = ""
	ColorGray    string = ""
)

func Colorize(colorCode string, format string, args ...interface{}) string {
	var out string

	if len(args) > 0 {
		out = fmt.Sprintf(format, args...)
	} else {
		out = format
	}

	return out
}
