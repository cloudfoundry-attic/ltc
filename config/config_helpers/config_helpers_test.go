package config_helpers_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/ltc/config/config_helpers"
)

var _ = Describe("ConfigHelpers", func() {
	Describe("ConfigFileLocation", func() {
		It("returns the config location for the diego home path", func() {
			fileLocation := config_helpers.ConfigFileLocation(filepath.Join("home", "chicago"))
			Expect(fileLocation).To(Equal(filepath.Join("home", "chicago", ".lattice", "config.json")))
		})
	})
})
