package command_factory

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/ltc/app_examiner"
	"github.com/cloudfoundry-incubator/ltc/app_runner"
	"github.com/cloudfoundry-incubator/ltc/app_runner/command_factory"
	"github.com/cloudfoundry-incubator/ltc/docker_runner/docker_metadata_fetcher"
	"github.com/cloudfoundry-incubator/ltc/docker_runner/docker_repository_name_formatter"
	"github.com/cloudfoundry-incubator/ltc/exit_handler"
	"github.com/cloudfoundry-incubator/ltc/exit_handler/exit_codes"
	"github.com/cloudfoundry-incubator/ltc/logs/console_tailed_logs_outputter"
	"github.com/cloudfoundry-incubator/ltc/terminal"
	"github.com/codegangsta/cli"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
)

type DockerRunnerCommandFactory struct {
	command_factory.AppRunnerCommandFactory

	dockerMetadataFetcher docker_metadata_fetcher.DockerMetadataFetcher
}

type DockerRunnerCommandFactoryConfig struct {
	AppRunner           app_runner.AppRunner
	AppExaminer         app_examiner.AppExaminer
	UI                  terminal.UI
	Domain              string
	Env                 []string
	Clock               clock.Clock
	Logger              lager.Logger
	ExitHandler         exit_handler.ExitHandler
	TailedLogsOutputter console_tailed_logs_outputter.TailedLogsOutputter

	DockerMetadataFetcher docker_metadata_fetcher.DockerMetadataFetcher
}

func NewDockerRunnerCommandFactory(config DockerRunnerCommandFactoryConfig) *DockerRunnerCommandFactory {
	return &DockerRunnerCommandFactory{
		AppRunnerCommandFactory: command_factory.AppRunnerCommandFactory{
			AppRunner:           config.AppRunner,
			AppExaminer:         config.AppExaminer,
			UI:                  config.UI,
			Domain:              config.Domain,
			Env:                 config.Env,
			Clock:               config.Clock,
			ExitHandler:         config.ExitHandler,
			TailedLogsOutputter: config.TailedLogsOutputter,
		},

		dockerMetadataFetcher: config.DockerMetadataFetcher,
	}
}

func (factory *DockerRunnerCommandFactory) MakeCreateAppCommand() cli.Command {

	var createFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "working-dir, w",
			Usage: "Working directory for container (overrides Docker metadata)",
			Value: "",
		},
		cli.StringSliceFlag{
			Name:  "env, e",
			Usage: "Environment variables (can be passed multiple times)",
			Value: &cli.StringSlice{},
		},
		cli.IntFlag{
			Name:  "cpu-weight, c",
			Usage: "Relative CPU weight for the container (valid values: 1-100)",
			Value: 100,
		},
		cli.IntFlag{
			Name:  "memory-mb, m",
			Usage: "Memory limit for container in MB",
			Value: 128,
		},
		cli.IntFlag{
			Name:  "disk-mb, d",
			Usage: "Disk limit for container in MB",
			Value: 0,
		},
		cli.StringFlag{
			Name:  "user, u",
			Usage: "Runs the app under this user context",
		},
		cli.BoolFlag{
			Name:  "run-as-root, r",
			Usage: "Deprecated: please use --user instead",
		},
		cli.BoolFlag{
			Name:  "privileged",
			Usage: "Run the app in a privileged container (Warning: This is insecure.)",
		},
		cli.StringFlag{
			Name:  "ports, p",
			Usage: "Ports to expose on the container (comma delimited)",
		},
		cli.IntFlag{
			Name:  "monitor-port, M",
			Usage: "Selects the port used to healthcheck the app",
		},
		cli.StringFlag{
			Name: "monitor-url, U",
			Usage: "Uses HTTP to healthcheck the app\n\t\t" +
				"format is: port:/path/to/endpoint",
		},
		cli.DurationFlag{
			Name:  "monitor-timeout",
			Usage: "Timeout for the app healthcheck",
			Value: time.Second,
		},
		cli.StringFlag{
			Name:  "monitor-command",
			Usage: "Uses the command (with arguments) to healthcheck the app",
		},
		cli.StringSliceFlag{
			Name:  "http-route, R",
			Usage: "Requests for HOST.SYSTEM_DOMAIN on port 80 will be forwarded to the associated container port. Container ports must be among those specified with --ports or with the EXPOSE Docker image directive. Usage: --http-route HOST:CONTAINER_PORT. Can be passed multiple times.",
		},
		cli.StringSliceFlag{
			Name:  "tcp-route, T",
			Usage: "Requests for the provided external port will be forwarded to the associated container port. Container ports must be among those specified with --ports or with the EXPOSE Docker image directive. Usage: --tcp-route EXTERNAL_PORT:CONTAINER_PORT. Can be passed multiple times.",
		},
		cli.IntFlag{
			Name:  "instances, i",
			Usage: "Number of application instances to spawn on launch",
			Value: 1,
		},
		cli.BoolFlag{
			Name:  "no-monitor",
			Usage: "Disables healthchecking for the app",
		},
		cli.BoolFlag{
			Name:  "no-routes",
			Usage: "Registers no routes for the app",
		},
		cli.DurationFlag{
			Name:  "timeout, t",
			Usage: "Polling timeout for app to start",
			Value: command_factory.DefaultPollingTimeout,
		},
		cli.StringFlag{
			Name:  "http-routes",
			Usage: "DEPRECATED: Please use --http-route instead.",
		},
		cli.StringFlag{
			Name:  "tcp-routes",
			Usage: "DEPRECATED: Please use --tcp-route instead.",
		},
	}

	var createAppCommand = cli.Command{
		Name:    "create",
		Aliases: []string{"cr"},
		Usage:   "Creates a docker app on lattice",
		Description: `ltc create APP_NAME DOCKER_IMAGE

   APP_NAME is required and must be unique across the Lattice cluster
   DOCKER_IMAGE is required and must match the standard docker image format
   e.g.
   		1. "cloudfoundry/lattice-app"
   		2. "redis" - for official images; resolves to library/redis

   ltc will fetch the command associated with your Docker image.
   To provide a custom command:
   ltc create APP_NAME DOCKER_IMAGE <optional flags> -- START_COMMAND APP_ARG1 APP_ARG2 ...

   ltc will also fetch the working directory associated with your Docker image.
   If the image does not specify a working directory, ltc will default the working directory to "/"
   To provide a custom working directory:
   ltc create APP_NAME DOCKER_IMAGE --working-dir=/foo/app-folder -- START_COMMAND APP_ARG1 APP_ARG2 ...

   To specify environment variables:
   ltc create APP_NAME DOCKER_IMAGE -e FOO=BAR -e BAZ=WIBBLE

   By default, http routes will be created for all container ports specified in the EXPOSE directive in
   the Docker image. E.g. for application myapp and a Docker image that specifies ports 80 and 8080,
   two http routes will be created by default:

     - requests to myapp.SYSTEM_DOMAIN:80 will be routed to container port 80
     - requests to myapp-8080.SYSTEM_DOMAIN:80 will be routed to container port 8080

   To configure your own routing:
   ltc create APP_NAME DOCKER_IMAGE --http-route HOST:CONTAINER_PORT [ --http-route HOST:CONTAINER_PORT ...] --tcp-route EXTERNAL_PORT:CONTAINER_PORT [ --tcp-route EXTERNAL_PORT:CONTAINER_PORT ...]
]

   Examples:
     ltc create myapp mydockerimage --http-route=myapp-admin:6000 will route requests received at myapp-admin.SYSTEM_DOMAIN:80 to container port 6000.
     ltc create myredis redis --tcp-route=50000:6379 will route requests received at SYSTEM_DOMAIN:50000 to container port 6379.
`,
		Action: factory.createApp,
		Flags:  createFlags,
	}

	return createAppCommand
}

func (factory *DockerRunnerCommandFactory) createApp(context *cli.Context) {
	workingDirFlag := context.String("working-dir")
	envVarsFlag := context.StringSlice("env")
	instancesFlag := context.Int("instances")
	cpuWeightFlag := uint(context.Int("cpu-weight"))
	memoryMBFlag := context.Int("memory-mb")
	diskMBFlag := context.Int("disk-mb")
	userFlag := context.String("user")
	runAsRootFlag := context.Bool("run-as-root")
	privilegedFlag := context.Bool("privileged")
	portsFlag := context.String("ports")
	noMonitorFlag := context.Bool("no-monitor")
	portMonitorFlag := context.Int("monitor-port")
	urlMonitorFlag := context.String("monitor-url")
	monitorTimeoutFlag := context.Duration("monitor-timeout")
	monitorCommandFlag := context.String("monitor-command")
	httpRouteFlag := context.StringSlice("http-route")
	tcpRouteFlag := context.StringSlice("tcp-route")
	noRoutesFlag := context.Bool("no-routes")
	timeoutFlag := context.Duration("timeout")
	name := context.Args().Get(0)
	dockerPath := context.Args().Get(1)
	terminator := context.Args().Get(2)
	startCommand := context.Args().Get(3)

	var appArgs []string
	switch {
	case len(context.Args()) < 2:
		factory.UI.SayIncorrectUsage("APP_NAME and DOCKER_IMAGE are required")
		factory.ExitHandler.Exit(exit_codes.InvalidSyntax)
		return
	case startCommand != "" && terminator != "--":
		factory.UI.SayIncorrectUsage("'--' Required before start command")
		factory.ExitHandler.Exit(exit_codes.InvalidSyntax)
		return
	case len(context.Args()) > 4:
		appArgs = context.Args()[4:]
	case cpuWeightFlag < 1 || cpuWeightFlag > 100:
		factory.UI.SayIncorrectUsage("Invalid CPU Weight")
		factory.ExitHandler.Exit(exit_codes.InvalidSyntax)
		return
	}

	httpRoutesFlag := context.String("http-routes")
	if httpRoutesFlag != "" {
		factory.UI.SayIncorrectUsage("Unable to parse routes\n  Pass multiple --http-route flags instead of comma-delimiting.  See help page for details.")
		factory.ExitHandler.Exit(exit_codes.InvalidSyntax)
		return
	}

	tcpRoutesFlag := context.String("tcp-routes")
	if tcpRoutesFlag != "" {
		factory.UI.SayIncorrectUsage("Unable to parse routes\n  Pass multiple --tcp-route flags instead of comma-delimiting.  See help page for details.")
		factory.ExitHandler.Exit(exit_codes.InvalidSyntax)
		return
	}

	imageMetadata, err := factory.dockerMetadataFetcher.FetchMetadata(dockerPath)
	if err != nil {
		factory.UI.SayLine(fmt.Sprintf("Error fetching image metadata: %s", err))
		factory.ExitHandler.Exit(exit_codes.BadDocker)
		return
	}

	if privilegedFlag {
		factory.UI.SayLine("Warning: It is possible for a privileged app to break out of its container and access the host OS!")
	}

	if runAsRootFlag {
		userFlag = "root"
		factory.UI.SayLine("Warning: run-as-root has been deprecated, please use '--user=root' instead)")
	}

	if userFlag == "" {
		if imageMetadata.User != "" {
			userFlag = imageMetadata.User
			factory.UI.SayLine("Setting the user to %s (obtained from docker image metadata)...", imageMetadata.User)
		} else {
			userFlag = "root"
			factory.UI.SayLine("Warning: No container user specified to run your app, your app will be run as root!")
		}
	} else {
		factory.UI.SayLine("Setting the user to %s from option...", userFlag)
	}

	exposedPorts, err := factory.getExposedPortsFromArgs(portsFlag, imageMetadata)
	if err != nil {
		factory.UI.SayLine(err.Error())
		factory.ExitHandler.Exit(exit_codes.InvalidSyntax)
		return
	}

	monitorConfig, err := factory.GetMonitorConfig(exposedPorts, portMonitorFlag, noMonitorFlag, urlMonitorFlag, monitorCommandFlag, monitorTimeoutFlag)
	if err != nil {
		factory.UI.SayLine(err.Error())
		if err.Error() == command_factory.MonitorPortNotExposed {
			factory.ExitHandler.Exit(exit_codes.CommandFailed)
		} else {
			factory.ExitHandler.Exit(exit_codes.InvalidSyntax)
		}
		return
	}

	if workingDirFlag == "" {
		factory.UI.SayLine("No working directory specified, using working directory from the image metadata...")
		if imageMetadata.WorkingDir != "" {
			workingDirFlag = imageMetadata.WorkingDir
			factory.UI.SayLine("Working directory is:")
			factory.UI.SayLine(workingDirFlag)
		} else {
			workingDirFlag = "/"
		}
	}

	switch {
	case monitorCommandFlag != "":
		factory.UI.SayLine(fmt.Sprintf("Monitoring the app with command %s", monitorConfig.CustomCommand))
	case !noMonitorFlag:
		factory.UI.SayLine(fmt.Sprintf("Monitoring the app on port %d...", monitorConfig.Port))
	default:
		factory.UI.SayLine("No ports will be monitored.")
	}

	if startCommand == "" {
		if len(imageMetadata.StartCommand) == 0 {
			factory.UI.SayLine("Unable to determine start command from image metadata.")
			factory.ExitHandler.Exit(exit_codes.BadDocker)
			return
		}

		factory.UI.SayLine("No start command specified, using start command from the image metadata...")
		startCommand = imageMetadata.StartCommand[0]

		factory.UI.SayLine("Start command is:")
		factory.UI.SayLine(strings.Join(imageMetadata.StartCommand, " "))

		appArgs = imageMetadata.StartCommand[1:]
	}

	routeOverrides, err := factory.ParseRouteOverrides(httpRouteFlag, exposedPorts)
	if err != nil {
		factory.UI.SayLine(err.Error())
		factory.ExitHandler.Exit(exit_codes.InvalidSyntax)
		return
	}

	tcpRoutes, err := factory.ParseTcpRoutes(tcpRouteFlag, exposedPorts)
	if err != nil {
		factory.UI.SayLine(err.Error())
		factory.ExitHandler.Exit(exit_codes.InvalidSyntax)
		return
	}

	rootFS, err := docker_repository_name_formatter.FormatForReceptor(dockerPath)
	if err != nil {
		factory.UI.SayLine(err.Error())
		factory.ExitHandler.Exit(exit_codes.CommandFailed)
		return
	}

	envVars := map[string]string{}

	for _, dockerEnv := range imageMetadata.Env {
		split := strings.SplitN(dockerEnv, "=", 2)
		if len(split) == 2 {
			envVars[split[0]] = split[1]
		}
	}

	appEnvVars := factory.BuildAppEnvironment(envVarsFlag, name)
	for appEnvKey := range appEnvVars {
		envVars[appEnvKey] = appEnvVars[appEnvKey]
	}

	err = factory.AppRunner.CreateApp(app_runner.CreateAppParams{
		AppEnvironmentParams: app_runner.AppEnvironmentParams{
			EnvironmentVariables: envVars,
			User:                 userFlag,
			Privileged:           privilegedFlag,
			Monitor:              monitorConfig,
			Instances:            instancesFlag,
			CPUWeight:            cpuWeightFlag,
			MemoryMB:             memoryMBFlag,
			DiskMB:               diskMBFlag,
			ExposedPorts:         exposedPorts,
			WorkingDir:           workingDirFlag,
			RouteOverrides:       routeOverrides,
			TcpRoutes:            tcpRoutes,
			NoRoutes:             noRoutesFlag,
		},

		Name:         name,
		RootFS:       rootFS,
		StartCommand: startCommand,
		AppArgs:      appArgs,
		Timeout:      timeoutFlag,
	})
	if err != nil {
		factory.UI.SayLine(fmt.Sprintf("Error creating app: %s", err))
		factory.ExitHandler.Exit(exit_codes.CommandFailed)
		return
	}

	factory.WaitForAppCreation(name, timeoutFlag, instancesFlag)
}

func (factory *DockerRunnerCommandFactory) getExposedPortsFromArgs(portsFlag string, imageMetadata *docker_metadata_fetcher.ImageMetadata) ([]uint16, error) {
	if portsFlag != "" {
		portStrings := strings.Split(portsFlag, ",")
		sort.Strings(portStrings)

		convertedPorts := []uint16{}
		for _, p := range portStrings {
			intPort, err := strconv.Atoi(p)
			if err != nil || intPort > 65535 {
				return []uint16{}, errors.New(command_factory.InvalidPortErrorMessage)
			}
			convertedPorts = append(convertedPorts, uint16(intPort))
		}
		return convertedPorts, nil
	}

	if len(imageMetadata.ExposedPorts) > 0 {
		var exposedPortStrings []string
		for _, port := range imageMetadata.ExposedPorts {
			exposedPortStrings = append(exposedPortStrings, strconv.Itoa(int(port)))
		}
		factory.UI.SayLine(fmt.Sprintf("No port specified, using exposed ports from the image metadata.\n\tExposed Ports: %s", strings.Join(exposedPortStrings, ", ")))
		return imageMetadata.ExposedPorts, nil
	}

	factory.UI.SayLine(fmt.Sprintf("No port specified, image metadata did not contain exposed ports. Defaulting to 8080."))
	return []uint16{8080}, nil
}
