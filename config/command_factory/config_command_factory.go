package command_factory

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ltc/config"
	"github.com/cloudfoundry-incubator/ltc/config/target_verifier"
	"github.com/cloudfoundry-incubator/ltc/exit_handler"
	"github.com/cloudfoundry-incubator/ltc/exit_handler/exit_codes"
	"github.com/cloudfoundry-incubator/ltc/terminal"
	"github.com/cloudfoundry-incubator/ltc/version"
	"github.com/codegangsta/cli"
)

type ConfigCommandFactory struct {
	config            *config.Config
	ui                terminal.UI
	targetVerifier    target_verifier.TargetVerifier
	blobStoreVerifier BlobStoreVerifier
	exitHandler       exit_handler.ExitHandler
	versionManager    version.VersionManager
}

//go:generate counterfeiter -o fake_blob_store_verifier/fake_blob_store_verifier.go . BlobStoreVerifier
type BlobStoreVerifier interface {
	Verify(config *config.Config) (authorized bool, err error)
}

func NewConfigCommandFactory(config *config.Config, ui terminal.UI, targetVerifier target_verifier.TargetVerifier, blobStoreVerifier BlobStoreVerifier, exitHandler exit_handler.ExitHandler, versionManager version.VersionManager) *ConfigCommandFactory {
	return &ConfigCommandFactory{config, ui, targetVerifier, blobStoreVerifier, exitHandler, versionManager}
}

func (factory *ConfigCommandFactory) MakeTargetCommand() cli.Command {
	var targetFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "s3",
			Usage: "Target an S3 bucket as the droplet store",
		},
		cli.BoolFlag{
			Name:  "domain, d",
			Usage: "Print the currently targeted lattice deployment's domain name",
		},
	}

	var targetCommand = cli.Command{
		Name:        "target",
		Aliases:     []string{"ta"},
		Usage:       "Targets a lattice cluster",
		Description: "ltc target TARGET (e.g., 192.168.11.11.xip.io)",
		Action:      factory.target,
		Flags:       targetFlags,
	}

	return targetCommand
}

func (factory *ConfigCommandFactory) target(context *cli.Context) {
	target := context.Args().First()
	s3Enabled := context.Bool("s3")
	domainOnlyFlag := context.Bool("domain")

	if target == "" {
		if domainOnlyFlag {
			if factory.config.Target() != "" {
				factory.ui.SayLine(factory.config.Target())
			}
		} else {
			factory.printTarget()
			factory.printBlobTarget()
		}
		return
	}

	factory.config.SetTarget(target)
	factory.config.SetLogin("", "")

	if s3Enabled {
		accessKey := factory.ui.Prompt("S3 Access Key")
		secretKey := factory.ui.PromptForPassword("S3 Secret Key")
		bucketName := factory.ui.Prompt("S3 Bucket")
		region := factory.ui.Prompt("S3 Region")
		factory.config.SetS3BlobStore(accessKey, secretKey, bucketName, region)
	} else {
		factory.config.SetBlobStore(target, "8444", "", "")
	}

	_, authorized, err := factory.targetVerifier.VerifyTarget(factory.config.Receptor())
	if err != nil {
		factory.ui.SayLine(fmt.Sprint("Error verifying target: ", err))
		factory.exitHandler.Exit(exit_codes.BadTarget)
		return
	}
	if authorized {
		if !factory.verifyBlobStore() {
			factory.exitHandler.Exit(exit_codes.BadTarget)
			return
		}

		factory.checkVersions()
		factory.save()
		return
	}

	username := factory.ui.Prompt("Username")
	password := factory.ui.PromptForPassword("Password")

	factory.config.SetLogin(username, password)
	factory.config.SetBlobStore(target, "8444", username, password)

	_, authorized, err = factory.targetVerifier.VerifyTarget(factory.config.Receptor())
	if err != nil {
		factory.ui.SayLine(fmt.Sprint("Error verifying target: ", err))
		factory.exitHandler.Exit(exit_codes.BadTarget)
		return
	}
	if !authorized {
		factory.ui.SayLine("Could not authorize target.")
		factory.exitHandler.Exit(exit_codes.BadTarget)
		return
	}

	if !factory.verifyBlobStore() {
		factory.ui.SayLine("Failed to verify blob store")
		factory.exitHandler.Exit(exit_codes.BadTarget)
		return
	}

	factory.checkVersions()
	factory.save()
}

func (factory *ConfigCommandFactory) verifyBlobStore() bool {
	authorized, err := factory.blobStoreVerifier.Verify(factory.config)
	if err != nil {
		factory.ui.SayLine("Could not connect to the droplet store.")
		return false
	}
	if !authorized {
		factory.ui.SayLine("Could not authenticate with the droplet store.")
		return false
	}
	return true
}

func (f *ConfigCommandFactory) checkVersions() {
	ltcMatchesServer, err := f.versionManager.LtcMatchesServer()
	if !ltcMatchesServer {
		f.ui.SayLine(fmt.Sprintf("WARNING: local ltc version (%s) does not match target expected version.", f.versionManager.LatticeVersion()))

		if err == nil {
			f.ui.SayLine("Run `ltc sync` to replace your local ltc command-line tool with your target cluster's expected version.")
		}
	}

}

func (factory *ConfigCommandFactory) save() {
	err := factory.config.Save()
	if err != nil {
		factory.ui.SayLine(err.Error())
		factory.exitHandler.Exit(exit_codes.FileSystemError)
		return
	}

	factory.ui.SayLine("API location set.")
}

func (factory *ConfigCommandFactory) printTarget() {
	if factory.config.Target() == "" {
		factory.ui.SayLine("Target not set.")
		return
	}
	target := factory.config.Target()
	if username := factory.config.Username(); username != "" {
		target = fmt.Sprintf("%s@%s", username, target)
	}
	factory.ui.SayLine(fmt.Sprintf("Target:\t\t%s", target))
}

func (factory *ConfigCommandFactory) printBlobTarget() {
	var endpoint string
	if factory.config.ActiveBlobStore() == config.S3BlobStore {
		endpoint = fmt.Sprintf("s3://%s (%s)", factory.config.S3BlobStore().BucketName, factory.config.S3BlobStore().Region)
	} else {
		blobStore := factory.config.BlobStore()
		if blobStore.Host == "" {
			factory.ui.SayLine("\tNo droplet store specified.")
			return
		}

		endpoint = fmt.Sprintf("%s:%s", blobStore.Host, blobStore.Port)
		if username := blobStore.Username; username != "" {
			endpoint = fmt.Sprintf("%s@%s", username, endpoint)
		}
	}

	factory.ui.SayLine(fmt.Sprintf("Droplet store:\t%s", endpoint))
}
