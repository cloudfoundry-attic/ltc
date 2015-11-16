package version_test

import (
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	config_package "github.com/cloudfoundry-incubator/ltc/config"
	"github.com/cloudfoundry-incubator/ltc/version"
	"github.com/cloudfoundry-incubator/ltc/version/fake_file_swapper"
	"github.com/cloudfoundry-incubator/receptor"
	"github.com/cloudfoundry-incubator/receptor/fake_receptor"
)

var _ = Describe("VersionManager", func() {
	var (
		fakeFileSwapper    *fake_file_swapper.FakeFileSwapper
		fakeServer         *ghttp.Server
		config             *config_package.Config
		versionManager     version.VersionManager
		fakeReceptorClient *fake_receptor.FakeClient

		ltcTempFile *os.File
	)

	BeforeEach(func() {
		fakeFileSwapper = &fake_file_swapper.FakeFileSwapper{}

		fakeServer = ghttp.NewServer()
		fakeServerURL, err := url.Parse(fakeServer.URL())
		Expect(err).NotTo(HaveOccurred())

		fakeServerHost, fakeServerPort, err := net.SplitHostPort(fakeServerURL.Host)
		Expect(err).NotTo(HaveOccurred())

		ltcTempFile, err = ioutil.TempFile("", "fake-ltc")
		Expect(err).NotTo(HaveOccurred())

		fakeReceptorClient = &fake_receptor.FakeClient{}

		config = config_package.New(nil)
		config.SetTarget(fakeServerHost + ".xip.io:" + fakeServerPort)
		versionManager = version.NewVersionManager(fakeReceptorClient, fakeFileSwapper, "")
	})

	AfterEach(func() {
		ltcTempFile.Close()
		Expect(os.Remove(ltcTempFile.Name())).To(Succeed())
	})

	Describe("ServerVersions", func() {
		It("should fetch versions from receptor", func() {
			fakeReceptorClient.GetVersionReturns(receptor.VersionResponse{
				CFRelease:           "v219",
				CFRoutingRelease:    "v220",
				DiegoRelease:        "v221",
				GardenLinuxRelease:  "v222",
				LatticeRelease:      "v223",
				LatticeReleaseImage: "v224",
				LTC:                 "v225",
				Receptor:            "v226",
			}, nil)
			serverVersions, _ := versionManager.ServerVersions()
			Expect(serverVersions).To(Equal(version.ServerVersions{
				CFRelease:           "v219",
				CFRoutingRelease:    "v220",
				DiegoRelease:        "v221",
				GardenLinuxRelease:  "v222",
				LatticeRelease:      "v223",
				LatticeReleaseImage: "v224",
				LTC:                 "v225",
				Receptor:            "v226",
			}))
		})

		Context("when call to receptor fails", func() {
			It("should return the error", func() {
				err := errors.New("error")
				fakeReceptorClient.GetVersionReturns(receptor.VersionResponse{}, err)

				_, actualError := versionManager.ServerVersions()
				Expect(actualError).To(Equal(err))
			})
		})
	})

	Describe("#SyncLTC", func() {
		It("should download ltc from the target and swap it with ltc", func() {
			fakeServer.RouteToHandler("GET", "/v1/sync/amiga/ltc", ghttp.CombineHandlers(
				ghttp.RespondWith(200, []byte{0x01, 0x02, 0x03}, http.Header{
					"Content-Type":   []string{"application/octet-stream"},
					"Content-Length": []string{"3"},
				}),
			))

			tmpFile, err := ioutil.TempFile("", "")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(tmpFile.Name())

			fakeFileSwapper.GetTempFileReturns(tmpFile, nil)

			versionManager.SyncLTC(ltcTempFile.Name(), "amiga", config)

			Expect(fakeServer.ReceivedRequests()).To(HaveLen(1))

			Expect(fakeFileSwapper.GetTempFileCallCount()).To(Equal(1))

			Expect(fakeFileSwapper.SwapTempFileCallCount()).To(Equal(1))
			actualDest, actualSrc := fakeFileSwapper.SwapTempFileArgsForCall(0)
			Expect(actualDest).To(Equal(ltcTempFile.Name()))
			Expect(actualSrc).To(Equal(tmpFile.Name()))

			tmpFile, err = os.OpenFile(tmpFile.Name(), os.O_RDONLY, 0)
			Expect(err).NotTo(HaveOccurred())

			bytes, err := ioutil.ReadAll(tmpFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(bytes).To(Equal([]byte{0x01, 0x02, 0x03}))
		})

		Context("when the http request returns a non-200 status", func() {
			It("should return an error", func() {
				fakeServer.RouteToHandler("GET", "/v1/sync/amiga/ltc", ghttp.CombineHandlers(
					ghttp.RespondWith(500, "", nil),
				))

				err := versionManager.SyncLTC(ltcTempFile.Name(), "amiga", config)
				Expect(err).To(MatchError(HavePrefix("failed to download ltc")))
			})
		})

		Context("when the http request fails", func() {
			It("should return an error", func() {
				config.SetTarget("localhost:1")

				err := versionManager.SyncLTC(ltcTempFile.Name(), "amiga", config)
				Expect(err).To(MatchError(HavePrefix("failed to connect to receptor")))
			})
		})

		Context("when opening the temp file fails", func() {
			It("should return an error", func() {
				fakeServer.RouteToHandler("GET", "/v1/sync/amiga/ltc", ghttp.CombineHandlers(
					ghttp.RespondWith(200, []byte{0x01, 0x02, 0x03}, http.Header{
						"Content-Type":   []string{"application/octet-stream"},
						"Content-Length": []string{"3"},
					}),
				))

				fakeFileSwapper.GetTempFileReturns(nil, errors.New("boom"))

				err := versionManager.SyncLTC(ltcTempFile.Name(), "amiga", config)
				Expect(err).To(MatchError("failed to open temp file: boom"))
			})
		})

		Context("when the file copy fails", func() {
			It("should return an error", func() {
				fakeServer.RouteToHandler("GET", "/v1/sync/amiga/ltc", ghttp.CombineHandlers(
					ghttp.RespondWith(200, []byte{0x01, 0x02, 0x03}, http.Header{
						"Content-Type":   []string{"application/octet-stream"},
						"Content-Length": []string{"3"},
					}),
				))

				tmpFile, err := ioutil.TempFile("", "")
				Expect(err).NotTo(HaveOccurred())
				defer os.Remove(tmpFile.Name())
				tmpFile, err = os.OpenFile(tmpFile.Name(), os.O_RDONLY, 0)
				Expect(err).NotTo(HaveOccurred())

				fakeFileSwapper.GetTempFileReturns(tmpFile, nil)

				err = versionManager.SyncLTC(ltcTempFile.Name(), "amiga", config)
				Expect(err).To(MatchError(HavePrefix("failed to write to temp ltc")))
			})
		})

		Context("when swapping the files fails", func() {
			It("should return an error", func() {
				fakeServer.RouteToHandler("GET", "/v1/sync/amiga/ltc", ghttp.CombineHandlers(
					ghttp.RespondWith(200, []byte{0x01, 0x02, 0x03}, http.Header{
						"Content-Type":   []string{"application/octet-stream"},
						"Content-Length": []string{"3"},
					}),
				))

				tmpFile, err := ioutil.TempFile("", "")
				Expect(err).NotTo(HaveOccurred())
				defer os.Remove(tmpFile.Name())

				fakeFileSwapper.GetTempFileReturns(tmpFile, nil)
				fakeFileSwapper.SwapTempFileReturns(errors.New("failed"))

				err = versionManager.SyncLTC(ltcTempFile.Name(), "amiga", config)
				Expect(err).To(MatchError(HavePrefix("failed to swap ltc")))
			})
		})
	})

	Describe("#LatticeVersion", func() {
		It("should return its latticeVersion", func() {
			versionManager := version.NewVersionManager(fakeReceptorClient, fakeFileSwapper, "some-version")
			Expect(versionManager.LatticeVersion()).To(Equal("some-version"))
		})
	})

	Describe("#LtcMatchesServer", func() {
		BeforeEach(func() {
			fakeReceptorClient.GetVersionReturns(receptor.VersionResponse{
				CFRelease:           "v219",
				CFRoutingRelease:    "v220",
				DiegoRelease:        "v221",
				GardenLinuxRelease:  "v222",
				LatticeRelease:      "v223",
				LatticeReleaseImage: "v224",
				LTC:                 "v225",
				Receptor:            "v226",
			}, nil)
		})

		Context("when the local lattice version matches the server's expected version", func() {
			It("should return true", func() {
				versionManager := version.NewVersionManager(fakeReceptorClient, fakeFileSwapper, "v223")
				Expect(versionManager.LtcMatchesServer()).To(BeTrue())
			})
		})

		Context("when the local lattice version does not match the server's expected version", func() {
			It("should return false", func() {
				versionManager := version.NewVersionManager(fakeReceptorClient, fakeFileSwapper, "mismatched-version")
				Expect(versionManager.LtcMatchesServer()).To(BeFalse())
			})
		})
	})
})
