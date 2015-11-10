package cursor_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/ltc/terminal/cursor"
)

var _ = Describe("cursor", func() {
	Describe("Up", func() {
		It("moves the cursor up N lines", func() {
			Expect(cursor.Up(5)).To(Equal("\033[5A"))
		})
	})

	Describe("ClearToEndOfLine", func() {
		It("clears the line after the cursor", func() {
			Expect(cursor.ClearToEndOfLine()).To(Equal("\033[0K"))
		})
	})

	Describe("ClearToEndOfDisplay", func() {
		It("clears everything below the cursor", func() {
			Expect(cursor.ClearToEndOfDisplay()).To(Equal("\033[0J"))
		})
	})

	Describe("Show", func() {
		It("shows the cursor", func() {
			Expect(cursor.Show()).To(Equal("\033[?25h"))
		})
	})

	Describe("Hide", func() {
		It("hides the cursor", func() {
			Expect(cursor.Hide()).To(Equal("\033[?25l"))
		})
	})

	Context("When there is no TERM", func() {
		It("should return an empty string", func() {
			previousTerm := os.Getenv("TERM")
			Expect(os.Unsetenv("TERM")).To(Succeed())

			Expect(cursor.Up(2)).To(Equal(""))
			Expect(cursor.ClearToEndOfLine()).To(Equal(""))
			Expect(cursor.ClearToEndOfDisplay()).To(Equal(""))
			Expect(cursor.Show()).To(Equal(""))
			Expect(cursor.Hide()).To(Equal(""))

			Expect(os.Setenv("TERM", previousTerm)).To(Succeed())
		})
	})
})
