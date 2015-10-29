// +build !windows

package zipper_test

import (
	"archive/zip"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/ltc/droplet_runner/command_factory/cf_ignore/fake_cf_ignore"
	zipper_package "github.com/cloudfoundry-incubator/ltc/droplet_runner/command_factory/zipper"
)

var _ = Describe("Zipper", func() {
	var zipper zipper_package.Zipper

	BeforeEach(func() {
		zipper = &zipper_package.DropletArtifactZipper{}
	})

	Describe("#Zip", func() {
		var (
			prevDir, tmpDir string
			err             error

			fakeCFIgnore *fake_cf_ignore.FakeCFIgnore
		)

		BeforeEach(func() {
			fakeCFIgnore = &fake_cf_ignore.FakeCFIgnore{}

			tmpDir, err = ioutil.TempDir(os.TempDir(), "zip_contents")
			Expect(err).NotTo(HaveOccurred())

			Expect(ioutil.WriteFile(filepath.Join(tmpDir, "aaa"), []byte("aaa contents"), 0700)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(tmpDir, "bbb"), []byte("bbb contents"), 0750)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(tmpDir, "ccc"), []byte("ccc contents"), 0644)).To(Succeed())
			Expect(os.Symlink("ccc", filepath.Join(tmpDir, "ddd"))).To(Succeed())

			Expect(os.Mkdir(filepath.Join(tmpDir, "subfolder"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(tmpDir, "subfolder", "sub"), []byte("sub contents"), 0644)).To(Succeed())

			prevDir, err = os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Chdir(tmpDir)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.Chdir(prevDir)).To(Succeed())
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		It("zips successfully", func() {
			zipPath, err := zipper.Zip(tmpDir, fakeCFIgnore)
			Expect(err).NotTo(HaveOccurred())

			zipReader, err := zip.OpenReader(zipPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(zipReader.File).To(HaveLen(6))

			h := zipReader.File[0].FileHeader
			Expect(h.FileInfo().Mode()).To(Equal(os.FileMode(0700)))

			h = zipReader.File[1].FileHeader
			Expect(h.FileInfo().Mode()).To(Equal(os.FileMode(0750)))

			h = zipReader.File[2].FileHeader
			Expect(h.FileInfo().Mode()).To(Equal(os.FileMode(0644)))

			h = zipReader.File[3].FileHeader
			Expect(h.FileInfo().Mode() & os.ModeSymlink).To(Equal(os.ModeSymlink))

			h = zipReader.File[4].FileHeader
			Expect(h.FileInfo().Mode()).To(Equal(os.FileMode(os.ModeDir | 0755)))

			h = zipReader.File[5].FileHeader
			Expect(h.FileInfo().Mode()).To(Equal(os.FileMode(0644)))
		})
	})

	Describe("#Unzip", func() {
		var (
			prevDir, tmpDir string
			err             error
			tmpFile         *os.File
			prevUmask       int
		)

		BeforeEach(func() {
			prevUmask = syscall.Umask(0)

			tmpDir, err = ioutil.TempDir(os.TempDir(), "unzip_contents")
			Expect(err).NotTo(HaveOccurred())

			prevDir, err = os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Chdir(tmpDir)).To(Succeed())

			tmpFile, err = ioutil.TempFile("", "zipfile")
			Expect(err).NotTo(HaveOccurred())
			defer tmpFile.Close()

			zipWriter := zip.NewWriter(tmpFile)
			defer zipWriter.Close()

			var (
				header *zip.FileHeader
			)

			header = &zip.FileHeader{Name: "aaa"}
			header.SetMode(os.FileMode(0644))
			_, err = zipWriter.CreateHeader(header)
			Expect(err).NotTo(HaveOccurred())

			header = &zip.FileHeader{Name: "bbb/1.txt"}
			header.SetMode(os.FileMode(0640))
			_, err = zipWriter.CreateHeader(header)
			Expect(err).NotTo(HaveOccurred())

			header = &zip.FileHeader{Name: "bbb/2.txt"}
			header.SetMode(os.FileMode(0600))
			_, err = zipWriter.CreateHeader(header)
			Expect(err).NotTo(HaveOccurred())

			header = &zip.FileHeader{Name: "ddd/3.txt"}
			header.SetMode(os.FileMode(0600))
			_, err = zipWriter.CreateHeader(header)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			syscall.Umask(prevUmask)
		})

		It("unzips", func() {
			Expect(zipper.Unzip(tmpFile.Name(), tmpDir)).To(Succeed())

			var (
				err      error
				fileInfo os.FileInfo
			)

			fileInfo, err = os.Stat(filepath.Join(tmpDir, "aaa"))
			Expect(err).NotTo(HaveOccurred())
			Expect(fileInfo.Mode()).To(Equal(os.FileMode(0644)))

			fileInfo, err = os.Stat(filepath.Join(tmpDir, "bbb"))
			Expect(err).NotTo(HaveOccurred())
			Expect(fileInfo.Mode().Perm()).To(Equal(os.FileMode(0777)))

			fileInfo, err = os.Stat(filepath.Join(tmpDir, "bbb/1.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(fileInfo.Mode()).To(Equal(os.FileMode(0640)))

			fileInfo, err = os.Stat(filepath.Join(tmpDir, "bbb/2.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(fileInfo.Mode()).To(Equal(os.FileMode(0600)))

			fileInfo, err = os.Stat(filepath.Join(tmpDir, "ddd"))
			Expect(err).NotTo(HaveOccurred())
			Expect(fileInfo.Mode().Perm()).To(Equal(os.FileMode(0777)))
		})
	})
})
