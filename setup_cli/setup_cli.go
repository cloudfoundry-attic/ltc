package setup_cli

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudfoundry-incubator/ltc/cli_app_factory"
	"github.com/cloudfoundry-incubator/ltc/config"
	"github.com/cloudfoundry-incubator/ltc/config/config_helpers"
	"github.com/cloudfoundry-incubator/ltc/config/persister"
	"github.com/cloudfoundry-incubator/ltc/config/target_verifier"
	"github.com/cloudfoundry-incubator/ltc/exit_handler"
	"github.com/cloudfoundry-incubator/ltc/receptor_client"
	"github.com/codegangsta/cli"
	"github.com/pivotal-golang/lager"
)

const latticeCliHomeVar = "LATTICE_CLI_HOME"

var latticeVersion, diegoVersion string // provided by linker argument at compile-time

func NewCliApp() *cli.App {
	config := config.New(persister.NewFilePersister(config_helpers.ConfigFileLocation(ltcConfigRoot())))

	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	exitHandler := exit_handler.New(signalChan, os.Exit)
	go exitHandler.Run()

	receptorClientCreator := receptor_client.ProxyAwareCreator{}

	return cli_app_factory.MakeCliApp(
		diegoVersion,
		latticeVersion,
		ltcConfigRoot(),
		exitHandler,
		config,
		logger(),
		receptorClientCreator,
		target_verifier.New(receptorClientCreator),
		os.Stdout,
	)
}

func logger() lager.Logger {
	logger := lager.NewLogger("ltc")
	var logLevel lager.LogLevel

	if os.Getenv("LTC_LOG_LEVEL") == "DEBUG" {
		logLevel = lager.DEBUG
	} else {
		logLevel = lager.INFO
	}

	logger.RegisterSink(lager.NewWriterSink(os.Stderr, logLevel))
	return logger
}

func ltcConfigRoot() string {
	if os.Getenv(latticeCliHomeVar) != "" {
		return os.Getenv(latticeCliHomeVar)
	}

	if home := os.Getenv("HOME"); home != "" {
		return home
	}

	return os.Getenv("USERPROFILE")
}
