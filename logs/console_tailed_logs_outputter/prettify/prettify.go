package prettify

import (
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/ltc/logs/console_tailed_logs_outputter/chug"
	"github.com/cloudfoundry-incubator/ltc/terminal/colors"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/pivotal-golang/lager"
)

var colorLookup = map[string]string{
	"rep":          colors.ColorBlue,
	"garden-linux": colors.ColorPurple,
}

func Prettify(logMessage *events.LogMessage) string {
	entry := chug.ChugLogMessage(logMessage)

	var sourceType, sourceTypeColorized, sourceInstance, sourceInstanceColorized string

	sourceType = path.Base(entry.LogMessage.GetSourceType())
	sourceInstance = entry.LogMessage.GetSourceInstance()

	// TODO: Or, do we use GetSourceType() for raw and Json source for pretty?
	color, ok := colorLookup[path.Base(strings.Split(sourceType, ":")[0])]
	if ok {
		sourceTypeColorized = colors.Colorize(color, sourceType)
		sourceInstanceColorized = colors.Colorize(color, sourceInstance)
	} else {
		sourceTypeColorized = sourceType
		sourceInstanceColorized = sourceInstance
	}

	prefix := fmt.Sprintf("[%s|%s]", sourceTypeColorized, sourceInstanceColorized)

	colorWidth := len(sourceTypeColorized+sourceInstanceColorized) - len(sourceType+sourceInstance)

	components := append([]string(nil), fmt.Sprintf("%-"+strconv.Itoa(34+colorWidth)+"s", prefix))

	var whichFunc func(chug.Entry) []string
	if entry.IsLager {
		whichFunc = prettyPrintLog
	} else {
		whichFunc = prettyPrintRaw
	}

	components = append(components, whichFunc(entry)...)
	return strings.Join(components, " ")
}

func prettyPrintLog(entry chug.Entry) []string {
	var logColor, level string
	switch entry.Log.LogLevel {
	case lager.INFO:
		logColor = colors.ColorDefault
		level = "[INFO]"
	case lager.DEBUG:
		logColor = colors.ColorGray
		level = "[DEBUG]"
	case lager.ERROR:
		logColor = colors.ColorRed
		level = "[ERROR]"
	case lager.FATAL:
		logColor = colors.ColorRed
		level = "[FATAL]"
	}
	level = fmt.Sprintf("%s%-9s", logColor, level)

	var components []string
	components = append(components, level)

	timestamp := entry.Log.Timestamp.Format("01/02 15:04:05.00")
	components = append(components, fmt.Sprintf("%-17s", timestamp))
	components = append(components, fmt.Sprintf("%-14s", entry.Log.Session))
	components = append(components, entry.Log.Message)
	components = append(components, colors.ColorDefault)

	if entry.Log.Error != nil {
		components = append(components, fmt.Sprintf("\n%s%s%s%s", strings.Repeat(" ", 66), logColor, entry.Log.Error.Error(), colors.ColorDefault))
	}

	if len(entry.Log.Data) > 0 {
		dataJSON, _ := json.Marshal(entry.Log.Data)
		components = append(components, fmt.Sprintf("\n%s%s%s%s", strings.Repeat(" ", 66), logColor, string(dataJSON), colors.ColorDefault))
	}

	return components
}

func prettyPrintRaw(entry chug.Entry) []string {
	var components []string
	components = append(components, strings.Repeat(" ", 9)) // loglevel
	timestamp := time.Unix(0, entry.LogMessage.GetTimestamp())
	components = append(components, fmt.Sprintf("%-17s", timestamp.Format("01/02 15:04:05.00")))
	components = append(components, strings.Repeat(" ", 14)) // sesh
	components = append(components, string(entry.Raw))

	return components
}
