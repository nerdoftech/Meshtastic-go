package serial

import (
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/nerdoftech/Meshtastic-go/pkg/types"
	log "github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSerial(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Serial Suite")
}

var fakeData = []byte{0x1, 0x2, 0x3, 0x4}

type mockPort struct {
	buf []byte
}

// Returns the buff slice one byte at a time
func (m *mockPort) Read(data []byte) (int, error) {
	n := 0
	if len(m.buf) > 0 {
		data[0] = m.buf[0]
		n = 1
		m.buf = m.buf[1:]
	} else {
		return 0, os.ErrClosed
	}
	return n, nil
}

// Satisfy ReadWriteCloseFlusher interface
func (m *mockPort) Write([]byte) (int, error) { return 0, nil }
func (m *mockPort) Close() error              { return nil }
func (m *mockPort) Flush() error              { return nil }

var _ = Describe("serial port lib tests", func() {
	var portMock *types.MockReadWriteCloseFlusher
	var sp *SerialPort
	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		portMock = types.NewMockReadWriteCloseFlusher(ctrl)
		sp = &SerialPort{
			port:     portMock,
			recvChan: make(chan []byte, 1),
			recvMu:   &sync.Mutex{},
		}
	})
	Context("test interface", func() {
		It("should fulfill TransportInterface", func() {
			var iface types.TransportInterface = sp
			Expect(sp).Should(BeAssignableToTypeOf(iface))
		})
	})
	Context("SendToRadio", func() {
		It("should work", func() {
			portMock.
				EXPECT().
				Write(gomock.Eq([]byte{START1, START1, START1, START1})).
				Return(0, nil)
			portMock.EXPECT().
				Write(gomock.Len(len(fakeData)+4)).
				Return(0, nil)
			portMock.EXPECT().Flush().Return(nil)

			err := sp.SendToRadio(fakeData)
			Expect(err).Should(BeNil())
		})
		It("should error on port wake", func() {
			expErr := errors.New("error")
			portMock.
				EXPECT().
				Write(gomock.Eq([]byte{START1, START1, START1, START1})).
				Return(0, expErr)

			err := sp.SendToRadio(fakeData)
			Expect(err).Should(HaveOccurred())
		})
		It("should error on port wake", func() {
			expErr := errors.New("error")
			portMock.
				EXPECT().
				Write(gomock.Eq([]byte{START1, START1, START1, START1})).
				Return(0, nil)
			portMock.EXPECT().
				Write(gomock.Len(len(fakeData)+4)).
				Return(0, expErr)

			err := sp.SendToRadio(fakeData)
			Expect(err).Should(HaveOccurred())
		})
	})
	Context("reader", func() {
		It("should work", func() {
			data := []byte{
				START2, START1, START1, byte(0x99), START1, START1, // Test bad headers of case 0, 1
				START2, START2, byte(512 >> 8), byte(0), START1, // Tests maximum packet size
				START1, START2, byte(0), byte(len(fakeData)),
			}
			data = append(data, fakeData...)
			data = append(data, byte(0x99)) // extra byte test overflow
			msp := &mockPort{data}
			sp.port = msp
			go sp.Listen()

			Expect(<-sp.recvChan).Should(Equal(fakeData))
			sp.Close()
		})
	})
})