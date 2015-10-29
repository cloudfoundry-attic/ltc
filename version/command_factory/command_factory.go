package command_factory

import (
	config_package "github.com/cloudfoundry-incubator/ltc/config"
	"github.com/cloudfoundry-incubator/ltc/exit_handler"
	"github.com/cloudfoundry-incubator/ltc/exit_handler/exit_codes"
	"github.com/cloudfoundry-incubator/ltc/terminal"
	"github.com/cloudfoundry-incubator/ltc/version"
	"github.com/codegangsta/cli"
)

type VersionCommandFactory struct {
	config      *config_package.Config
	ui          terminal.UI
	exitHandler exit_handler.ExitHandler

	arch    string
	ltcPath string

	versionManager VersionManager
}

type VersionManager interface {
	SyncLTC(ltcPath string, arch string, config *config_package.Config) error
	ServerVersions() (version.ServerVersions, error)
	LtcVersion() string
}

func NewVersionCommandFactory(config *config_package.Config, ui terminal.UI, exitHandler exit_handler.ExitHandler, arch string, ltcPath string, versionManager VersionManager) *VersionCommandFactory {
	return &VersionCommandFactory{config, ui, exitHandler, arch, ltcPath, versionManager}
}

func (f *VersionCommandFactory) MakeSyncCommand() cli.Command {
	return cli.Command{
		Name:        "sync",
		Usage:       "Updates ltc to the latest version available in the targeted Lattice cluster",
		Description: "ltc sync",
		Action:      f.syncLTC,
	}
}

func (f *VersionCommandFactory) MakeVersionCommand() cli.Command {
	return cli.Command{
		Name:        "version",
		Usage:       "Returns CLI and server versions",
		Description: "ltc version",
		Action:      f.version,
	}
}

func (f *VersionCommandFactory) syncLTC(context *cli.Context) {
	var architecture string
	switch f.arch {
	case "darwin":
		architecture = "osx"
	case "linux":
		architecture = "linux"
	default:
		f.ui.SayLine("Error: Unknown architecture %s. Sync not supported.", f.arch)
		f.exitHandler.Exit(exit_codes.CommandFailed)
		return
	}

	if f.ltcPath == "" {
		f.ui.SayLine("Error: Unable to locate the ltc binary. Sync not supported.")
		f.exitHandler.Exit(exit_codes.CommandFailed)
		return
	}

	if f.config.Target() == "" {
		f.ui.SayLine("Error: Must be targeted to sync.")
		f.exitHandler.Exit(exit_codes.CommandFailed)
		return
	}

	err := f.versionManager.SyncLTC(f.ltcPath, architecture, f.config)
	if err != nil {
		f.ui.SayLine("Error: " + err.Error())
		f.exitHandler.Exit(exit_codes.CommandFailed)
		return
	}

	f.ui.SayLine("Updated ltc to the latest version.")
}

func (f *VersionCommandFactory) version(context *cli.Context) {
	f.ui.SayLine("Client version: " + f.versionManager.LtcVersion())

	serverVersions, err := f.versionManager.ServerVersions()
	if err != nil {
		f.ui.SayLine("Error: " + err.Error())
		f.exitHandler.Exit(exit_codes.CommandFailed)
		return
	}

	f.ui.SayLine("CF release version: " + serverVersions.CfRelease)
	f.ui.SayLine("CF routing release version: " + serverVersions.CfRoutingRelease)
	f.ui.SayLine("Diego release version: " + serverVersions.DiegoRelease)
	f.ui.SayLine("Garden linux release version: " + serverVersions.GardenLinuxRelease)
	f.ui.SayLine("Lattice release version: " + serverVersions.LatticeRelease)
	f.ui.SayLine("Lattice release image version: " + serverVersions.LatticeReleaseImage)
	f.ui.SayLine("Receptor version: " + serverVersions.Receptor)
}
