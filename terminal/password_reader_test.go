package terminal_test

import (
	"bytes"
	"io"

	"github.com/docker/docker/pkg/term"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/ltc/terminal"
	"github.com/cloudfoundry-incubator/ltc/terminal/mocks"
)

type fakeStdin struct {
	io.Reader
}

func (s *fakeStdin) Fd() uintptr { return 42 }

var _ = Describe("PasswordReader", func() {
	var (
		fakeTerm *mocks.FakeTerm

		inBuffer  *bytes.Buffer
		outBuffer *bytes.Buffer

		passwordReader terminal.PasswordReader
	)

	BeforeEach(func() {
		fakeTerm = &mocks.FakeTerm{}

		inBuffer = bytes.NewBufferString("secret\n")
		outBuffer = &bytes.Buffer{}

		passwordReader = &terminal.TermPasswordReader{
			Term:   fakeTerm,
			Stdin:  &fakeStdin{inBuffer},
			Stdout: outBuffer,
		}
	})

	Context("#PromptForPassword", func() {
		It("should disable echo and prompt for a password", func() {
			termState := &term.State{}
			fakeTerm.SaveStateReturns(termState, nil)

			Expect(passwordReader.PromptForPassword("P%s%s", "ro", "mpt")).To(Equal("secret"))

			Expect(outBuffer.String()).To(Equal("Prompt: \n"))

			Expect(fakeTerm.DisableEchoCallCount()).To(Equal(1))
			fd, state := fakeTerm.DisableEchoArgsForCall(0)
			Expect(fd).To(Equal(uintptr(42)))
			Expect(state == termState).To(BeFalse())

			Expect(fakeTerm.RestoreTerminalCallCount()).To(Equal(1))
			fd, state = fakeTerm.RestoreTerminalArgsForCall(0)
			Expect(fd).To(Equal(uintptr(42)))
			Expect(state == termState).To(BeTrue())
		})
	})
})
