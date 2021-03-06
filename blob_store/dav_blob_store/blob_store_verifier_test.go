package dav_blob_store_test

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/cloudfoundry-incubator/ltc/blob_store/dav_blob_store"
	config_package "github.com/cloudfoundry-incubator/ltc/config"
)

var _ = Describe("BlobStore", func() {
	const responseBody401 = `<?xml version="1.0" encoding="iso-8859-1"?>
		<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
		         "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
		<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en" lang="en">
		 <head>
		  <title>401 - Unauthorized</title>
		 </head>
		 <body>
		  <h1>401 - Unauthorized</h1>
		 </body>
		</html>`

	var (
		verifier               dav_blob_store.Verifier
		config                 *config_package.Config
		fakeServer             *ghttp.Server
		serverHost, serverPort string
	)

	BeforeEach(func() {
		config = config_package.New(nil)

		fakeServer = ghttp.NewServer()
		fakeServerURL, err := url.Parse(fakeServer.URL())
		Expect(err).NotTo(HaveOccurred())
		serverHost, serverPort, err = net.SplitHostPort(fakeServerURL.Host)
		Expect(err).NotTo(HaveOccurred())

		verifier = dav_blob_store.Verifier{}
	})

	AfterEach(func() {
		if fakeServer != nil {
			fakeServer.Close()
		}
	})

	Describe("Verify", func() {
		var responseBodyRoot string
		BeforeEach(func() {
			responseBodyRoot = `
				<?xml version="1.0" encoding="utf-8"?>
				<D:multistatus xmlns:D="DAV:" xmlns:ns0="urn:uuid:c2f41010-65b3-11d1-a29f-00aa00c14882/">
				  <D:response>
					<D:href>http://192.168.11.11:8444/blobs/</D:href>
					<D:propstat>
					  <D:prop>
						<D:creationdate ns0:dt="dateTime.tz">2015-07-29T18:43:50Z</D:creationdate>
						<D:getcontentlanguage>en</D:getcontentlanguage>
						<D:getcontentlength>4096</D:getcontentlength>
						<D:getcontenttype>httpd/unix-directory</D:getcontenttype>
						<D:getlastmodified ns0:dt="dateTime.rfc1123">Wed, 29 Jul 2015 18:43:36 GMT</D:getlastmodified>
						<D:resourcetype>
						  <D:collection/>
						</D:resourcetype>
					  </D:prop>
					  <D:status>HTTP/1.1 200 OK</D:status>
					</D:propstat>
				  </D:response>
				</D:multistatus>
			`
			responseBodyRoot = strings.Replace(responseBodyRoot, "http://192.168.11.11:8444", fakeServer.URL(), -1)
		})

		Context("when the DAV blob store does not require auth", func() {
			It("should return authorized", func() {
				fakeServer.RouteToHandler("PROPFIND", "/blobs/", ghttp.CombineHandlers(
					ghttp.VerifyHeaderKV("Depth", "1"),
					ghttp.RespondWith(207, responseBodyRoot, http.Header{"Content-Type": []string{"text/xml"}}),
				))

				config.SetBlobStore(serverHost, serverPort, "", "")
				authorized, err := verifier.Verify(config)
				Expect(err).NotTo(HaveOccurred())
				Expect(authorized).To(BeTrue())

				Expect(fakeServer.ReceivedRequests()).To(HaveLen(1))
			})
		})

		Context("when the DAV blob store requires auth", func() {
			It("should return authorized for proper credentials", func() {
				fakeServer.RouteToHandler("PROPFIND", "/blobs/", ghttp.CombineHandlers(
					ghttp.VerifyBasicAuth("good-user", "good-pass"),
					ghttp.VerifyHeaderKV("Depth", "1"),
					ghttp.RespondWith(207, responseBodyRoot, http.Header{"Content-Type": []string{"text/xml"}}),
				))

				config.SetBlobStore(serverHost, serverPort, "good-user", "good-pass")
				authorized, err := verifier.Verify(config)
				Expect(err).NotTo(HaveOccurred())
				Expect(authorized).To(BeTrue())

				Expect(fakeServer.ReceivedRequests()).To(HaveLen(1))
			})
			It("should return unauthorized for invalid credentials", func() {
				fakeServer.RouteToHandler("PROPFIND", "/blobs/",
					ghttp.CombineHandlers(
						ghttp.VerifyBasicAuth("bad-user", "bad-pass"),
						ghttp.VerifyHeaderKV("Depth", "1"),
						ghttp.RespondWith(http.StatusForbidden, responseBody401, http.Header{
							"Content-Type":     []string{"text/xml"},
							"WWW-Authenticate": []string{`Basic realm="blob"`},
						}),
					),
				)

				config.SetBlobStore(serverHost, serverPort, "bad-user", "bad-pass")
				authorized, err := verifier.Verify(config)
				Expect(err).NotTo(HaveOccurred())
				Expect(authorized).To(BeFalse())

				Expect(fakeServer.ReceivedRequests()).To(HaveLen(1))
			})
		})

		Context("when the blob store is inaccessible", func() {
			It("returns an error", func() {
				config.SetBlobStore(serverHost, serverPort, "", "")

				fakeServer.Close()
				fakeServer = nil

				_, err := verifier.Verify(config)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
