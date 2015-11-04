package command_factory_test

import (
	"errors"

	config_package "github.com/cloudfoundry-incubator/ltc/config"
	"github.com/cloudfoundry-incubator/ltc/exit_handler/exit_codes"
	"github.com/cloudfoundry-incubator/ltc/exit_handler/fake_exit_handler"
	"github.com/cloudfoundry-incubator/ltc/terminal"
	"github.com/cloudfoundry-incubator/ltc/test_helpers"
	"github.com/cloudfoundry-incubator/ltc/version"
	"github.com/cloudfoundry-incubator/ltc/version/command_factory"
	"github.com/cloudfoundry-incubator/ltc/version/fake_version_manager"
	"github.com/codegangsta/cli"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Version CommandFactory", func() {
	var (
		config             *config_package.Config
		outputBuffer       *gbytes.Buffer
		terminalUI         terminal.UI
		fakeExitHandler    *fake_exit_handler.FakeExitHandler
		fakeVersionManager *fake_version_manager.FakeVersionManager
		commandFactory     *command_factory.VersionCommandFactory
	)

	BeforeEach(func() {
		config = config_package.New(nil)
		config.SetTarget("lattice.xip.io")

		outputBuffer = gbytes.NewBuffer()
		terminalUI = terminal.NewUI(nil, outputBuffer, nil)
		fakeExitHandler = &fake_exit_handler.FakeExitHandler{}
		fakeVersionManager = &fake_version_manager.FakeVersionManager{}
		fakeVersionManager.LatticeVersionReturns("1.8.0")
		commandFactory = command_factory.NewVersionCommandFactory(
			config,
			terminalUI,
			fakeExitHandler,
			"darwin",
			"/fake/ltc",
			fakeVersionManager)
	})

	Describe("Version Command", func() {
		var versionCommand cli.Command

		BeforeEach(func() {
			versionCommand = commandFactory.MakeVersionCommand()
			fakeVersionManager.ServerVersionsReturns(version.ServerVersions{
				CfRelease:           "v219",
				CfRoutingRelease:    "v220",
				DiegoRelease:        "v221",
				GardenLinuxRelease:  "v222",
				LatticeRelease:      "v223",
				LatticeReleaseImage: "v224",
				Ltc:                 "v225",
				Receptor:            "v226",
			}, nil)
		})

		It("Prints the CLI and API versions", func() {
			test_helpers.ExecuteCommandWithArgs(versionCommand, []string{})

			Expect(outputBuffer).To(test_helpers.SayLine("Client version: 1.8.0"))
			Expect(outputBuffer).To(test_helpers.SayLine("CF release version: v219"))
			Expect(outputBuffer).To(test_helpers.SayLine("CF routing release version: v220"))
			Expect(outputBuffer).To(test_helpers.SayLine("Diego release version: v221"))
			Expect(outputBuffer).To(test_helpers.SayLine("Garden linux release version: v222"))
			Expect(outputBuffer).To(test_helpers.SayLine("Lattice release version: v223"))
			Expect(outputBuffer).To(test_helpers.SayLine("Lattice release image version: v224"))
			Expect(outputBuffer).To(test_helpers.SayLine("Receptor version: v226"))
		})

		Context("when the version manager returns an error", func() {
			It("should print an error", func() {
				fakeVersionManager.ServerVersionsReturns(version.ServerVersions{}, errors.New("failed"))

				test_helpers.ExecuteCommandWithArgs(versionCommand, []string{})

				Expect(outputBuffer).To(test_helpers.SayLine("Error: failed"))
				Expect(fakeVersionManager.ServerVersionsCallCount()).To(Equal(1))
				Expect(fakeExitHandler.ExitCalledWith).To(Equal([]int{exit_codes.CommandFailed}))
			})
		})
	})

	Describe("SyncCommand", func() {
		var syncCommand cli.Command

		BeforeEach(func() {
			syncCommand = commandFactory.MakeSyncCommand()
		})

		It("should sync ltc", func() {
			test_helpers.ExecuteCommandWithArgs(syncCommand, []string{})

			Expect(outputBuffer).To(test_helpers.SayLine("Updated ltc to the latest version."))
			Expect(fakeVersionManager.SyncLTCCallCount()).To(Equal(1))
			actualLTCPath, actualArch, actualConfig := fakeVersionManager.SyncLTCArgsForCall(0)
			Expect(actualLTCPath).To(Equal("/fake/ltc"))
			Expect(actualArch).To(Equal("osx"))
			Expect(actualConfig).To(Equal(config))
		})

		Context("when not targeted", func() {
			It("should print an error", func() {
				config.SetTarget("")

				test_helpers.ExecuteCommandWithArgs(syncCommand, []string{})

				Expect(outputBuffer).To(test_helpers.SayLine("Error: Must be targeted to sync."))
				Expect(fakeVersionManager.SyncLTCCallCount()).To(Equal(0))
				Expect(fakeExitHandler.ExitCalledWith).To(Equal([]int{exit_codes.CommandFailed}))
			})
		})

		Context("when the architecture is unknown", func() {
			It("should print an error", func() {
				commandFactory := command_factory.NewVersionCommandFactory(config, terminalUI, fakeExitHandler, "unknown-arch", "fakeltc", fakeVersionManager)
				syncCommand = commandFactory.MakeSyncCommand()

				test_helpers.ExecuteCommandWithArgs(syncCommand, []string{})

				Expect(outputBuffer).To(test_helpers.SayLine("Error: Unknown architecture unknown-arch. Sync not supported."))
				Expect(fakeVersionManager.SyncLTCCallCount()).To(Equal(0))
				Expect(fakeExitHandler.ExitCalledWith).To(Equal([]int{exit_codes.CommandFailed}))
			})
		})

		Context("when the ltc binary can't be found", func() {
			It("should print an error", func() {
				commandFactory := command_factory.NewVersionCommandFactory(config, terminalUI, fakeExitHandler, "darwin", "", fakeVersionManager)
				syncCommand = commandFactory.MakeSyncCommand()

				test_helpers.ExecuteCommandWithArgs(syncCommand, []string{})

				Expect(outputBuffer).To(test_helpers.SayLine("Error: Unable to locate the ltc binary. Sync not supported."))
				Expect(fakeVersionManager.SyncLTCCallCount()).To(Equal(0))
				Expect(fakeExitHandler.ExitCalledWith).To(Equal([]int{exit_codes.CommandFailed}))
			})
		})

		Context("when SyncLTC fails", func() {
			It("should print an error", func() {
				fakeVersionManager.SyncLTCReturns(errors.New("failed"))

				test_helpers.ExecuteCommandWithArgs(syncCommand, []string{})

				Expect(outputBuffer).To(test_helpers.SayLine("Error: failed"))
				Expect(fakeVersionManager.SyncLTCCallCount()).To(Equal(1))
				Expect(fakeExitHandler.ExitCalledWith).To(Equal([]int{exit_codes.CommandFailed}))
			})
		})
	})
})
