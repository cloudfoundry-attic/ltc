// +build !windows

package version_test

import (
	"io/ioutil"
	"os"

	"github.com/cloudfoundry-incubator/ltc/version"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("FileSwapper", func() {
	Describe("#SwapTempFile", func() {
		var (
			srcPath        string
			destPath       string
			appFileSwapper *version.AppFileSwapper
		)

		BeforeEach(func() {
			srcFile, err := ioutil.TempFile("", "src")
			Expect(err).NotTo(HaveOccurred())
			_, err = srcFile.Write([]byte("sourcedata"))
			Expect(err).NotTo(HaveOccurred())
			srcFile.Close()
			srcPath = srcFile.Name()

			Expect(os.Chmod(srcPath, 0644)).To(Succeed())

			destFile, err := ioutil.TempFile("", "dest")
			Expect(err).NotTo(HaveOccurred())
			_, err = destFile.Write([]byte("destdata"))
			Expect(err).NotTo(HaveOccurred())
			destFile.Close()
			destPath = destFile.Name()

			Expect(os.Chmod(destPath, 0755)).To(Succeed())

			appFileSwapper = &version.AppFileSwapper{}
		})

		AfterEach(func() {
			Expect(os.Remove(destPath)).To(Succeed())
		})

		It("writes the contents to the destination file", func() {
			err := appFileSwapper.SwapTempFile(destPath, srcPath)
			Expect(err).NotTo(HaveOccurred())

			destFile, err := os.OpenFile(destPath, os.O_RDONLY, 0)
			defer destFile.Close()

			Expect(err).NotTo(HaveOccurred())

			bytes, err := ioutil.ReadAll(destFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(bytes)).To(Equal("sourcedata"))
		})

		It("removes the source file", func() {
			err := appFileSwapper.SwapTempFile(destPath, srcPath)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(srcPath)
			Expect(os.IsExist(err)).To(BeFalse())
		})

		It("preserves the permissions of the destination file", func() {
			err := appFileSwapper.SwapTempFile(destPath, srcPath)
			Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(destPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode() & os.ModePerm).To(Equal(os.FileMode(0755)))
		})

		Context("when there is an error getting dest file stats", func() {
			It("returns an error", func() {
				err := appFileSwapper.SwapTempFile("/this-does-not-exist", srcPath)
				Expect(err).To(MatchError(HavePrefix("failed to stat dest file")))
			})
		})

		Context("when there is an error renaming the file", func() {
			It("returns an error", func() {
				err := appFileSwapper.SwapTempFile(destPath, "/this-does-not-exist")
				Expect(err).To(MatchError(HavePrefix("failed to rename file")))
			})
		})
	})
})
