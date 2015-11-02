package colors_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/ltc/terminal/colors"
)

var _ = Describe("Colorize", func() {
	Context("when $TERM is set", func() {
		var previousTerm string

		BeforeEach(func() {
			previousTerm = os.Getenv("TERM")
			Expect(os.Setenv("TERM", "xterm")).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.Setenv("TERM", previousTerm)).To(Succeed())
		})

		It("colors the text with printf-style syntax", func() {
			Expect(colors.Colorize("\x1b[98m", "%dxyz%s", 23, "happy")).To(Equal("\x1b[98m23xyzhappy\x1b[0m"))
		})

		It("colors the text without printf-style syntax", func() {
			Expect(colors.Colorize("\x1b[98m", "happy")).To(Equal("\x1b[98mhappy\x1b[0m"))
		})
	})

	Context("when $TERM is not set", func() {
		var previousTerm string

		BeforeEach(func() {
			previousTerm = os.Getenv("TERM")
			Expect(os.Unsetenv("TERM")).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.Setenv("TERM", previousTerm)).To(Succeed())
		})

		It("colors the text with printf-style syntax", func() {
			Expect(colors.Colorize("\x1b[98m", "%dxyz%s", 23, "happy")).To(Equal("23xyzhappy"))
		})

		It("colors the text without printf-style syntax", func() {
			Expect(colors.Colorize("\x1b[98m", "happy")).To(Equal("happy"))
		})
	})
})
