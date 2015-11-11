package command_factory

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ltc/app_examiner"
	"github.com/cloudfoundry-incubator/ltc/exit_handler"
	"github.com/cloudfoundry-incubator/ltc/exit_handler/exit_codes"
	"github.com/cloudfoundry-incubator/ltc/logs/console_tailed_logs_outputter"
	"github.com/cloudfoundry-incubator/ltc/task_examiner"
	"github.com/cloudfoundry-incubator/ltc/terminal"
	"github.com/codegangsta/cli"
)

type logsCommandFactory struct {
	appExaminer         app_examiner.AppExaminer
	taskExaminer        task_examiner.TaskExaminer
	ui                  terminal.UI
	tailedLogsOutputter console_tailed_logs_outputter.TailedLogsOutputter
	exitHandler         exit_handler.ExitHandler
}

func NewLogsCommandFactory(appExaminer app_examiner.AppExaminer, taskExaminer task_examiner.TaskExaminer, ui terminal.UI, tailedLogsOutputter console_tailed_logs_outputter.TailedLogsOutputter, exitHandler exit_handler.ExitHandler) *logsCommandFactory {
	return &logsCommandFactory{
		appExaminer:         appExaminer,
		taskExaminer:        taskExaminer,
		ui:                  ui,
		tailedLogsOutputter: tailedLogsOutputter,
		exitHandler:         exitHandler,
	}
}

func (factory *logsCommandFactory) MakeLogsCommand() cli.Command {
	var logsCommand = cli.Command{
		Name:        "logs",
		Aliases:     []string{"lg"},
		Usage:       "Streams logs from the specified application or task",
		Description: "ltc logs <app-name>",
		Action:      factory.tailLogs,
		Flags:       []cli.Flag{},
	}

	return logsCommand
}

func (factory *logsCommandFactory) MakeDebugLogsCommand() cli.Command {
	var debugLogsFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "raw, r",
			Usage: "Removes pretty formatting",
		},
	}
	return cli.Command{
		Name:    "debug-logs",
		Aliases: []string{"dl"},
		Usage:   "Streams logs from the lattice cluster components",
		Description: `ltc debug-logs [--raw]

   Output format is:

    [source|instance] [loglevel] timestamp session message summary
                                                   (error message)
                                                   (message detail)`,
		Action: factory.tailDebugLogs,
		Flags:  debugLogsFlags,
	}
}

func (factory *logsCommandFactory) tailLogs(context *cli.Context) {
	appGuid := context.Args().First()

	if appGuid == "" {
		factory.ui.SayIncorrectUsage("<app-name> required")
		factory.exitHandler.Exit(exit_codes.InvalidSyntax)
		return
	}

	appExists, err := factory.appExaminer.AppExists(appGuid)
	if err != nil {
		factory.ui.SayLine(fmt.Sprintf("Error: %s", err.Error()))
		factory.exitHandler.Exit(exit_codes.CommandFailed)
		return
	}

	if !appExists {
		_, err := factory.taskExaminer.TaskStatus(appGuid)
		if err != nil {
			factory.ui.SayLine(fmt.Sprintf("Application or task %s not found.", appGuid))
			factory.ui.SayLine(fmt.Sprintf("Tailing logs and waiting for %s to appear...", appGuid))
		}
	}

	factory.tailedLogsOutputter.OutputTailedLogs(appGuid)
}

func (factory *logsCommandFactory) tailDebugLogs(context *cli.Context) {
	rawFlag := context.Bool("raw")
	factory.tailedLogsOutputter.OutputDebugLogs(!rawFlag)
}
