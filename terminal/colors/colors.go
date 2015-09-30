package colors

var ColorCodeLength = len(red) + len(defaultStyle)

const (
	red             string = ""
	cyan            string = ""
	green           string = ""
	yellow          string = ""
	purpleUnderline string = ""
	defaultStyle    string = ""
	boldStyle       string = ""
	grayColor       string = ""
)

func Red(output string) string {
	return colorText(output, red)
}

func Green(output string) string {
	return colorText(output, green)
}

func Cyan(output string) string {
	return colorText(output, cyan)
}

func Yellow(output string) string {
	return colorText(output, yellow)
}

func Gray(output string) string {
	return colorText(output, grayColor)
}

func NoColor(output string) string {
	return colorText(output, defaultStyle)
}

func Bold(output string) string {
	return colorText(output, boldStyle)
}

func PurpleUnderline(output string) string {
	return colorText(output, purpleUnderline)
}

func colorText(output string, color string) string {
	return output
}
