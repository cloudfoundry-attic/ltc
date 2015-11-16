package command_factory_test

import (
	"errors"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"github.com/cloudfoundry-incubator/ltc/config/command_factory"
	"github.com/cloudfoundry-incubator/ltc/config/command_factory/fake_blob_store_verifier"
	"github.com/cloudfoundry-incubator/ltc/config/persister"
	"github.com/cloudfoundry-incubator/ltc/config/target_verifier/fake_target_verifier"
	"github.com/cloudfoundry-incubator/ltc/exit_handler/exit_codes"
	"github.com/cloudfoundry-incubator/ltc/exit_handler/fake_exit_handler"
	"github.com/cloudfoundry-incubator/ltc/terminal"
	"github.com/cloudfoundry-incubator/ltc/terminal/mocks"
	"github.com/cloudfoundry-incubator/ltc/test_helpers"
	"github.com/cloudfoundry-incubator/ltc/version/fake_version_manager"
	"github.com/codegangsta/cli"

	config_package "github.com/cloudfoundry-incubator/ltc/config"
)

var _ = Describe("CommandFactory", func() {
	var (
		stdinReader           *io.PipeReader
		stdinWriter           *io.PipeWriter
		outputBuffer          *gbytes.Buffer
		terminalUI            terminal.UI
		config                *config_package.Config
		configPersister       persister.Persister
		fakeTargetVerifier    *fake_target_verifier.FakeTargetVerifier
		fakeBlobStoreVerifier *fake_blob_store_verifier.FakeBlobStoreVerifier
		fakeExitHandler       *fake_exit_handler.FakeExitHandler
		fakePasswordReader    *mocks.FakePasswordReader
		fakeVersionManager    *fake_version_manager.FakeVersionManager
	)

	BeforeEach(func() {
		stdinReader, stdinWriter = io.Pipe()
		outputBuffer = gbytes.NewBuffer()
		fakeExitHandler = &fake_exit_handler.FakeExitHandler{}
		fakePasswordReader = &mocks.FakePasswordReader{}
		terminalUI = terminal.NewUI(stdinReader, outputBuffer, fakePasswordReader)
		fakeTargetVerifier = &fake_target_verifier.FakeTargetVerifier{}
		fakeBlobStoreVerifier = &fake_blob_store_verifier.FakeBlobStoreVerifier{}
		fakeVersionManager = &fake_version_manager.FakeVersionManager{}
		configPersister = persister.NewMemPersister()
		config = config_package.New(configPersister)
	})

	Describe("TargetCommand", func() {
		var targetCommand cli.Command

		verifyOldTargetStillSet := func() {
			newConfig := config_package.New(configPersister)
			Expect(newConfig.Load()).To(Succeed())

			Expect(newConfig.Receptor()).To(Equal("http://olduser:oldpass@receptor.oldtarget.com"))
		}

		BeforeEach(func() {
			commandFactory := command_factory.NewConfigCommandFactory(config, terminalUI, fakeTargetVerifier, fakeBlobStoreVerifier, fakeExitHandler, fakeVersionManager)
			targetCommand = commandFactory.MakeTargetCommand()

			config.SetTarget("oldtarget.com")
			config.SetLogin("olduser", "oldpass")
			Expect(config.Save()).To(Succeed())
		})

		Context("displaying the target", func() {
			JustBeforeEach(func() {
				test_helpers.ExecuteCommandWithArgs(targetCommand, []string{})
			})

			It("outputs the current user and target host", func() {
				Expect(outputBuffer).To(test_helpers.SayLine("Target:\t\tolduser@oldtarget.com"))
			})

			Context("when no username is set", func() {
				BeforeEach(func() {
					config.SetLogin("", "")
					Expect(config.Save()).To(Succeed())
				})

				It("only prints the target", func() {
					Expect(outputBuffer).To(test_helpers.SayLine("Target:\t\toldtarget.com"))
				})
			})

			Context("when no target is set", func() {
				BeforeEach(func() {
					config.SetTarget("")
					Expect(config.Save()).To(Succeed())
				})

				It("informs the user the target is not set", func() {
					Expect(outputBuffer).To(test_helpers.SayLine("Target not set."))
				})
			})

			Context("when no blob store is targeted", func() {
				It("should specify that no blob store is targeted", func() {
					Expect(outputBuffer).To(test_helpers.SayLine("\tNo droplet store specified."))
				})
			})

			Context("when a DAV blob store is targeted", func() {
				BeforeEach(func() {
					config.SetBlobStore("blobtarget.com", "8444", "blobUser", "password")
					Expect(config.Save()).To(Succeed())
				})

				It("outputs the current user and blob store host", func() {
					Expect(outputBuffer).To(test_helpers.SayLine("Droplet store:\tblobUser@blobtarget.com:8444"))
				})

				Context("when no blob store username is set", func() {
					BeforeEach(func() {
						config.SetBlobStore("blobtarget.com", "8444", "", "")
						Expect(config.Save()).To(Succeed())
					})

					It("only prints the blob store host", func() {
						Expect(outputBuffer).To(test_helpers.SayLine("Droplet store:\tblobtarget.com:8444"))
					})
				})
			})

			Context("when a S3 blob store is targeted", func() {
				BeforeEach(func() {
					config.SetS3BlobStore("access", "secret", "bucket", "region")
					Expect(config.Save()).To(Succeed())
				})

				It("outputs the s3 bucket and region", func() {
					Expect(outputBuffer).To(test_helpers.SayLine("Droplet store:\ts3://bucket (region)"))
				})
			})
		})

		Context("when --domain is pased", func() {
			JustBeforeEach(func() {
				test_helpers.ExecuteCommandWithArgs(targetCommand, []string{"--domain"})
			})

			It("outputs just the target host", func() {
				Expect(string(outputBuffer.Contents())).To(Equal("oldtarget.com\n"))
			})

			Context("when no target is set", func() {
				BeforeEach(func() {
					config.SetTarget("")
					Expect(config.Save()).To(Succeed())
				})

				It("informs the user the target is not set", func() {
					Expect(string(outputBuffer.Contents())).To(BeEmpty())
				})
			})
		})

		Context("when initially connecting to the receptor without authentication", func() {
			BeforeEach(func() {
				fakeTargetVerifier.VerifyTargetReturns(true, true, nil)
				fakeBlobStoreVerifier.VerifyReturns(true, nil)
			})

			It("saves the new receptor target", func() {
				test_helpers.ExecuteCommandWithArgs(targetCommand, []string{"myapi.com"})

				Expect(fakeTargetVerifier.VerifyTargetCallCount()).To(Equal(1))
				Expect(fakeTargetVerifier.VerifyTargetArgsForCall(0)).To(Equal("http://receptor.myapi.com"))

				newConfig := config_package.New(configPersister)
				Expect(newConfig.Load()).To(Succeed())
				Expect(newConfig.Receptor()).To(Equal("http://receptor.myapi.com"))
			})

			It("clears out existing saved target credentials", func() {
				test_helpers.ExecuteCommandWithArgs(targetCommand, []string{"myapi.com"})

				Expect(fakeTargetVerifier.VerifyTargetCallCount()).To(Equal(1))
				Expect(fakeTargetVerifier.VerifyTargetArgsForCall(0)).To(Equal("http://receptor.myapi.com"))
			})

			It("saves the new blob store target", func() {
				fakeBlobStoreVerifier.VerifyReturns(true, nil)

				test_helpers.ExecuteCommandWithArgs(targetCommand, []string{"myapi.com"})

				Expect(fakeBlobStoreVerifier.VerifyCallCount()).To(Equal(1))

				config := fakeBlobStoreVerifier.VerifyArgsForCall(0)
				blobStoreConfig := config.BlobStore()
				Expect(blobStoreConfig).To(Equal(config_package.BlobStoreConfig{
					Host: "myapi.com",
					Port: "8444",
				}))

				newConfig := config_package.New(configPersister)
				Expect(newConfig.Load()).To(Succeed())
				Expect(newConfig.BlobStore()).To(Equal(config_package.BlobStoreConfig{
					Host: "myapi.com",
					Port: "8444",
				}))
			})

			Context("when the blob store requires authorization", func() {
				It("exits", func() {
					fakeBlobStoreVerifier.VerifyReturns(false, nil)

					test_helpers.ExecuteCommandWithArgs(targetCommand, []string{"myapi.com"})

					Expect(fakeBlobStoreVerifier.VerifyCallCount()).To(Equal(1))

					config := fakeBlobStoreVerifier.VerifyArgsForCall(0)
					blobStoreConfig := config.BlobStore()
					Expect(blobStoreConfig).To(Equal(config_package.BlobStoreConfig{
						Host: "myapi.com",
						Port: "8444",
					}))

					Expect(outputBuffer).To(test_helpers.SayLine("Could not authenticate with the droplet store."))
					verifyOldTargetStillSet()
					Expect(fakeExitHandler.ExitCalledWith).To(Equal([]int{exit_codes.BadTarget}))
				})
			})

			Context("when the blob store target is offline", func() {
				It("exits", func() {
					fakeBlobStoreVerifier.VerifyReturns(false, errors.New("some error"))

					test_helpers.ExecuteCommandWithArgs(targetCommand, []string{"myapi.com"})

					Expect(fakeBlobStoreVerifier.VerifyCallCount()).To(Equal(1))

					config := fakeBlobStoreVerifier.VerifyArgsForCall(0)
					blobStoreConfig := config.BlobStore()
					Expect(blobStoreConfig).To(Equal(config_package.BlobStoreConfig{
						Host: "myapi.com",
						Port: "8444",
					}))

					Expect(outputBuffer).To(test_helpers.SayLine("Could not connect to the droplet store."))
					verifyOldTargetStillSet()
					Expect(fakeExitHandler.ExitCalledWith).To(Equal([]int{exit_codes.BadTarget}))
				})
			})

			Context("when the persister returns errors", func() {
				BeforeEach(func() {
					commandFactory := command_factory.NewConfigCommandFactory(config_package.New(errorPersister("some error")), terminalUI, fakeTargetVerifier, fakeBlobStoreVerifier, fakeExitHandler, fakeVersionManager)
					targetCommand = commandFactory.MakeTargetCommand()
				})

				It("exits", func() {
					test_helpers.ExecuteCommandWithArgs(targetCommand, []string{"myapi.com"})

					Eventually(outputBuffer).Should(test_helpers.SayLine("some error"))
					verifyOldTargetStillSet()
					Expect(fakeExitHandler.ExitCalledWith).To(Equal([]int{exit_codes.FileSystemError}))
				})
			})
		})

		Context("when the receptor requires authentication", func() {
			BeforeEach(func() {
				fakeTargetVerifier.VerifyTargetReturns(true, false, nil)
				fakeBlobStoreVerifier.VerifyReturns(true, nil)
				fakePasswordReader.PromptForPasswordReturns("testpassword")
			})

			It("prompts for credentials and stores them in the config", func() {
				doneChan := test_helpers.AsyncExecuteCommandWithArgs(targetCommand, []string{"myapi.com"})

				Eventually(outputBuffer).Should(test_helpers.Say("Username: "))
				fakeTargetVerifier.VerifyTargetReturns(true, true, nil)
				stdinWriter.Write([]byte("testusername\n"))

				Eventually(doneChan, 3).Should(BeClosed())

				Expect(config.Target()).To(Equal("myapi.com"))
				Expect(config.Receptor()).To(Equal("http://testusername:testpassword@receptor.myapi.com"))
				Expect(outputBuffer).To(test_helpers.SayLine("API location set."))

				Expect(fakePasswordReader.PromptForPasswordCallCount()).To(Equal(1))
				Expect(fakePasswordReader.PromptForPasswordArgsForCall(0)).To(Equal("Password"))

				Expect(fakeTargetVerifier.VerifyTargetCallCount()).To(Equal(2))
				Expect(fakeTargetVerifier.VerifyTargetArgsForCall(0)).To(Equal("http://receptor.myapi.com"))
				Expect(fakeTargetVerifier.VerifyTargetArgsForCall(1)).To(Equal("http://testusername:testpassword@receptor.myapi.com"))
			})

			Context("when provided receptor credentials are invalid", func() {
				It("does not save the config", func() {
					fakePasswordReader.PromptForPasswordReturns("some-invalid-password")
					doneChan := test_helpers.AsyncExecuteCommandWithArgs(targetCommand, []string{"newtarget.com"})

					Eventually(outputBuffer).Should(test_helpers.Say("Username: "))
					stdinWriter.Write([]byte("some-invalid-user\n"))

					Eventually(doneChan, 3).Should(BeClosed())

					Expect(fakePasswordReader.PromptForPasswordCallCount()).To(Equal(1))
					Expect(fakePasswordReader.PromptForPasswordArgsForCall(0)).To(Equal("Password"))

					Expect(outputBuffer).To(test_helpers.SayLine("Could not authorize target."))

					verifyOldTargetStillSet()
					Expect(fakeExitHandler.ExitCalledWith).To(Equal([]int{exit_codes.BadTarget}))
				})
			})

			Context("when the receptor returns an error while verifying the provided credentials", func() {
				It("does not save the config", func() {
					fakePasswordReader.PromptForPasswordReturns("some-invalid-password")
					doneChan := test_helpers.AsyncExecuteCommandWithArgs(targetCommand, []string{"newtarget.com"})

					Eventually(outputBuffer).Should(test_helpers.Say("Username: "))

					fakeTargetVerifier.VerifyTargetReturns(true, false, errors.New("Unknown Error"))
					stdinWriter.Write([]byte("some-invalid-user\n"))

					Eventually(doneChan, 3).Should(BeClosed())

					Expect(fakePasswordReader.PromptForPasswordCallCount()).To(Equal(1))
					Expect(fakePasswordReader.PromptForPasswordArgsForCall(0)).To(Equal("Password"))

					Expect(outputBuffer).To(test_helpers.SayLine("Error verifying target: Unknown Error"))

					verifyOldTargetStillSet()
					Expect(fakeExitHandler.ExitCalledWith).To(Equal([]int{exit_codes.BadTarget}))
				})
			})

			Context("when the receptor credentials work on the blob store", func() {
				It("saves the new blob store target", func() {
					fakeBlobStoreVerifier.VerifyReturns(true, nil)

					doneChan := test_helpers.AsyncExecuteCommandWithArgs(targetCommand, []string{"myapi.com"})

					Eventually(outputBuffer).Should(test_helpers.Say("Username: "))
					fakeTargetVerifier.VerifyTargetReturns(true, true, nil)
					stdinWriter.Write([]byte("testusername\n"))

					Eventually(doneChan, 3).Should(BeClosed())

					Expect(fakeBlobStoreVerifier.VerifyCallCount()).To(Equal(1))

					config := fakeBlobStoreVerifier.VerifyArgsForCall(0)
					blobStoreConfig := config.BlobStore()
					Expect(blobStoreConfig).To(Equal(config_package.BlobStoreConfig{
						Host:     "myapi.com",
						Port:     "8444",
						Username: "testusername",
						Password: "testpassword",
					}))

					newConfig := config_package.New(configPersister)
					Expect(newConfig.Load()).To(Succeed())
					Expect(newConfig.Receptor()).To(Equal("http://testusername:testpassword@receptor.myapi.com"))
					Expect(newConfig.BlobStore()).To(Equal(config_package.BlobStoreConfig{
						Host:     "myapi.com",
						Port:     "8444",
						Username: "testusername",
						Password: "testpassword",
					}))
				})
			})

			Context("when the receptor credentials don't work on the blob store", func() {
				It("does not save the config", func() {
					fakeBlobStoreVerifier.VerifyReturns(false, nil)

					doneChan := test_helpers.AsyncExecuteCommandWithArgs(targetCommand, []string{"myapi.com"})

					Eventually(outputBuffer).Should(test_helpers.Say("Username: "))
					fakeTargetVerifier.VerifyTargetReturns(true, true, nil)
					stdinWriter.Write([]byte("testusername\n"))

					Eventually(doneChan, 3).Should(BeClosed())

					Expect(fakeBlobStoreVerifier.VerifyCallCount()).To(Equal(1))

					config := fakeBlobStoreVerifier.VerifyArgsForCall(0)
					blobStoreConfig := config.BlobStore()
					Expect(blobStoreConfig).To(Equal(config_package.BlobStoreConfig{
						Host:     "myapi.com",
						Port:     "8444",
						Username: "testusername",
						Password: "testpassword",
					}))

					Expect(outputBuffer).To(test_helpers.SayLine("Could not authenticate with the droplet store."))
					verifyOldTargetStillSet()
					Expect(fakeExitHandler.ExitCalledWith).To(Equal([]int{exit_codes.BadTarget}))
				})
			})

			Context("when the blob store is offline", func() {
				It("does not save the config", func() {
					fakeBlobStoreVerifier.VerifyReturns(false, errors.New("some error"))

					doneChan := test_helpers.AsyncExecuteCommandWithArgs(targetCommand, []string{"myapi.com"})

					Eventually(outputBuffer).Should(test_helpers.Say("Username: "))
					fakeTargetVerifier.VerifyTargetReturns(true, true, nil)
					stdinWriter.Write([]byte("testusername\n"))

					Eventually(doneChan, 3).Should(BeClosed())

					Expect(fakeBlobStoreVerifier.VerifyCallCount()).To(Equal(1))

					config := fakeBlobStoreVerifier.VerifyArgsForCall(0)
					blobStoreConfig := config.BlobStore()
					Expect(blobStoreConfig).To(Equal(config_package.BlobStoreConfig{
						Host:     "myapi.com",
						Port:     "8444",
						Username: "testusername",
						Password: "testpassword",
					}))

					Expect(outputBuffer).To(test_helpers.SayLine("Could not connect to the droplet store."))
					verifyOldTargetStillSet()
					Expect(fakeExitHandler.ExitCalledWith).To(Equal([]int{exit_codes.BadTarget}))
				})
			})
		})

		Context("s3", func() {
			It("prompts for s3 configuration when --s3 is passed", func() {
				fakeTargetVerifier.VerifyTargetReturns(true, true, nil)
				fakeBlobStoreVerifier.VerifyReturns(true, nil)
				fakePasswordReader.PromptForPasswordReturns("some-secret")

				doneChan := test_helpers.AsyncExecuteCommandWithArgs(targetCommand, []string{"myapi.com", "--s3"})

				Eventually(outputBuffer).Should(test_helpers.Say("S3 Access Key: "))
				stdinWriter.Write([]byte("some-access\n"))
				Eventually(outputBuffer).Should(test_helpers.Say("S3 Bucket: "))
				stdinWriter.Write([]byte("some-bucket\n"))
				Eventually(outputBuffer).Should(test_helpers.Say("S3 Region: "))
				stdinWriter.Write([]byte("some-region\n"))

				Eventually(doneChan, 3).Should(BeClosed())

				Expect(fakeBlobStoreVerifier.VerifyCallCount()).To(Equal(1))
				config := fakeBlobStoreVerifier.VerifyArgsForCall(0)
				s3BlobTargetInfo := config.S3BlobStore()
				Expect(s3BlobTargetInfo.AccessKey).To(Equal("some-access"))
				Expect(s3BlobTargetInfo.SecretKey).To(Equal("some-secret"))
				Expect(s3BlobTargetInfo.BucketName).To(Equal("some-bucket"))
				Expect(s3BlobTargetInfo.Region).To(Equal("some-region"))

				newConfig := config_package.New(configPersister)
				Expect(newConfig.Load()).To(Succeed())
				newS3BlobTargetInfo := newConfig.S3BlobStore()
				Expect(newS3BlobTargetInfo.AccessKey).To(Equal("some-access"))
				Expect(newS3BlobTargetInfo.SecretKey).To(Equal("some-secret"))
				Expect(newS3BlobTargetInfo.BucketName).To(Equal("some-bucket"))
				Expect(newS3BlobTargetInfo.Region).To(Equal("some-region"))
			})
		})

		Context("setting an invalid target", func() {
			It("does not save the config", func() {
				fakeTargetVerifier.VerifyTargetReturns(true, false, errors.New("Unknown Error"))

				test_helpers.ExecuteCommandWithArgs(targetCommand, []string{"newtarget.com"})

				Expect(outputBuffer).To(test_helpers.SayLine("Error verifying target: Unknown Error"))

				verifyOldTargetStillSet()
				Expect(fakeExitHandler.ExitCalledWith).To(Equal([]int{exit_codes.BadTarget}))
			})
		})

		Context("checking ltc target version", func() {
			BeforeEach(func() {
				fakeTargetVerifier.VerifyTargetReturns(true, true, nil)
				fakeBlobStoreVerifier.VerifyReturns(true, nil)
				fakeVersionManager.LatticeVersionReturns("some-version")
			})

			It("should print warning and recommend sync if ltc version does not match server", func() {
				fakeVersionManager.LtcMatchesServerReturns(false, nil)

				test_helpers.ExecuteCommandWithArgs(targetCommand, []string{"target.com"})

				Expect(fakeVersionManager.LtcMatchesServerCallCount()).To(Equal(1))
				Expect(fakeVersionManager.LtcMatchesServerArgsForCall(0)).To(Equal("http://receptor.target.com"))

				Expect(outputBuffer).To(test_helpers.SayLine("WARNING: local ltc version (some-version) does not match target expected version."))
				Expect(outputBuffer).To(test_helpers.SayLine("Run `ltc sync` to replace your local ltc command-line tool with your target cluster's expected version."))
			})

			It("should print warning and NOT recommend sync if ServerVersions endpoint fails", func() {
				fakeVersionManager.LtcMatchesServerReturns(false, errors.New("whoops"))

				test_helpers.ExecuteCommandWithArgs(targetCommand, []string{"target.com"})

				Expect(fakeVersionManager.LtcMatchesServerCallCount()).To(Equal(1))
				Expect(fakeVersionManager.LtcMatchesServerArgsForCall(0)).To(Equal("http://receptor.target.com"))

				Expect(outputBuffer).To(test_helpers.SayLine("WARNING: local ltc version (some-version) does not match target expected version."))
				Expect(outputBuffer).NotTo(test_helpers.SayLine("Run `ltc sync` to replace your local ltc command-line tool with your target cluster's expected version."))
			})

			It("should not print an error if ltc version matches server", func() {
				fakeVersionManager.LtcMatchesServerReturns(true, nil)

				test_helpers.ExecuteCommandWithArgs(targetCommand, []string{"target.com"})

				Expect(fakeVersionManager.LtcMatchesServerCallCount()).To(Equal(1))
				Expect(fakeVersionManager.LtcMatchesServerArgsForCall(0)).To(Equal("http://receptor.target.com"))

				Expect(outputBuffer).NotTo(test_helpers.SayLine("WARNING: local ltc version (some-version) does not match target expected version."))
				Expect(outputBuffer).NotTo(test_helpers.SayLine("Run `ltc sync` to replace your local ltc command-line tool with your target cluster's expected version."))
			})
		})
	})
})

type errorPersister string

func (f errorPersister) Load(i interface{}) error {
	return errors.New(string(f))
}

func (f errorPersister) Save(i interface{}) error {
	return errors.New(string(f))
}
