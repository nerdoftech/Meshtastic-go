package serial

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/nerdoftech/Meshtastic-go/pkg/types"
	log "github.com/sirupsen/logrus"
	"github.com/tarm/serial"
)

const (
	WAIT_AFTER_WAKE = 100 * time.Millisecond
	START1          = byte(0x94)
	START2          = byte(0xC3)
	PACKET_MTU      = 512
	PORT_SPEED      = 921600
)

// Buffer for serial reader
type serialBuffer struct {
	buf    []byte
	idx    int
	lenMsb int
	lenLsb int
	msgLen int
}

// Type for mesh interface from serial port
type SerialPort struct {
	Config   *serial.Config
	port     types.ReadWriteCloseFlusher
	recvChan chan []byte
	recvMu   *sync.Mutex
	stopped  uint32
}

// "/dev/ttyUSB0"
func NewSerialPort(dev string, recvCh chan []byte, mu *sync.Mutex) (*SerialPort, error) {
	sp := &SerialPort{
		Config:   &serial.Config{Name: dev, Baud: PORT_SPEED},
		recvChan: recvCh,
		recvMu:   mu,
	}
	var err error
	sp.port, err = serial.OpenPort(sp.Config)
	if err != nil {
		log.WithError(err).Error("could not open serial port")
		return nil, err
	}
	return sp, nil
}

func (s *SerialPort) SendToRadio(data []byte) error {
	// Wake serial port on radio
	log.Debug("writing wake packet to port")
	_, err := s.port.Write([]byte{START1, START1, START1, START1})
	if err != nil {
		log.WithError(err).Error("could not write to port")
		return err
	}

	// Wait for radio to initalize
	time.Sleep(WAIT_AFTER_WAKE)

	dlen := len(data)
	header := []byte{START1, START2, byte(dlen >> 8), byte(dlen)}
	data = append(header, data...)

	log.WithField("packet_len", dlen).Debug("writing data packet to port")
	_, err = s.port.Write(data)
	if err != nil {
		log.WithError(err).Error("could not write to port")
		return err
	}
	s.port.Flush()
	return nil
}

func (s *SerialPort) Close() {
	log.Debug("Closing serial port")
	atomic.SwapUint32(&s.stopped, 0)
	s.port.Flush()
	s.port.Close()
}

func (s *SerialPort) Listen() {
	log.Debug("listening to serial port")
	sb := &serialBuffer{}
	// read stream
	for s.stopped == 0 {
		b := make([]byte, 1)
		n, err := s.port.Read(b)
		if err != nil {
			log.WithError(err).Debug("error reading bytes from port")
		}
		if n == 0 {
			continue
		}

		// track and buffer bytes until we have a complete message
		switch sb.idx {
		case 0:
			if b[0] != START1 {
				sb.idx = 0 // restart
				continue
			}
		case 1:
			if b[0] == START1 {
				sb.idx = 1 // back one
				continue
			}
			if b[0] != START2 {
				sb.idx = 0 // restart
				continue
			}
		case 2:
			sb.lenMsb = int(b[0]) << 8
		case 3:
			sb.lenLsb = int(b[0])
			sb.msgLen = sb.lenMsb + sb.lenLsb
			// Check if packet is too big
			if sb.msgLen > PACKET_MTU {
				log.WithField("packet_len", sb.msgLen).Debug("received packet that was too big")
				sb = &serialBuffer{}
				continue
			}
			log.WithField("packet_len", sb.msgLen).Debug("received packet, starting to buffer")
			sb.buf = make([]byte, sb.msgLen)
		default:
			pktSize := sb.idx - 4
			// FIXME, this does not actually work
			// Check if packet is too big
			// if pktSize > sb.msgLen {
			// 	log.Debug("packet was too big, discarding")
			// 	sb = &serialBuffer{}
			// 	continue
			// }

			sb.buf[pktSize] = b[0]
			// This should be the whole message
			if sb.idx == sb.msgLen+4-1 {
				log.Debug("completed packet buffering, adding to queue")
				s.recvMu.Lock()
				s.recvChan <- sb.buf
				s.recvMu.Unlock()
				sb = &serialBuffer{}
				continue
			}
		}
		sb.idx++
	}
}
