package mesh

import (
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"github.com/nerdoftech/Meshtastic-go/pkg/message"
	"github.com/nerdoftech/Meshtastic-go/pkg/serial"
	mt "github.com/nerdoftech/Meshtastic-go/pkg/types"
)

const (
	TRANSPORT_BLUETOOTH Transport = iota
	TRANSPORT_SERIAL

	TOPIC_DATA Topic = iota
	TOPIC_NODE

	RX_CHAN_SIZE = 10
)

var TOPICS = []Topic{TOPIC_DATA, TOPIC_NODE}

type Transport int
type Topic int

type Mesh struct {
	transport   mt.TransportInterface
	mu          *sync.Mutex
	rxChan      chan []byte
	radioConfig *message.RadioConfig
	myInfo      *message.MyNodeInfo
	stopped     uint32
	topic       map[Topic][]func(interface{})
}

func NewMesh(dev string, tr Transport) (*Mesh, error) {
	m := &Mesh{
		mu:     &sync.Mutex{},
		rxChan: make(chan []byte, RX_CHAN_SIZE),
	}
	// Create topics
	m.topic = make(map[Topic][]func(interface{}))
	for _, tp := range TOPICS {
		m.topic[tp] = make([]func(interface{}), 0)
	}

	switch tr {
	case TRANSPORT_BLUETOOTH:
		return nil, errors.New("bluetooth not implemented")
	case TRANSPORT_SERIAL:
		m.transport = serial.NewSerialPort(dev, m.rxChan, m.mu)
	default:
		return nil, errors.New("invalid transport")
	}
	return m, nil
}

func (m *Mesh) Connect() error {
	// Connect to transport
	err := m.transport.Connect()
	if err != nil {
		log.WithError(err).Error("could not connect to transport")
		return err
	}
	go m.transport.Listen()
	go m.receiveFromRadio()

	// Get radio config
	err = m.getRadioConfig()
	if err != nil {
		log.WithError(err).Error("could not connect to transport")
		return err
	}

	return nil
}

func (m *Mesh) Close() {
	log.Debug("closing connection")
	atomic.SwapUint32(&m.stopped, 0)
	m.transport.Close()
}

func (m *Mesh) GetMyNodeInfo() *message.MyNodeInfo {
	return m.myInfo
}

func (m *Mesh) GetRadioConfig() *message.RadioConfig {
	return m.radioConfig
}

// Sends a WantConfigId msg to transport
func (m *Mesh) getRadioConfig() error {
	rand.Seed(time.Now().UnixNano())
	rn := rand.Uint32()
	msg := &message.ToRadio{
		Variant: &message.ToRadio_WantConfigId{
			WantConfigId: rn,
		},
	}
	log.WithField("config_id", rn).Debug("sending WantConfig to radio")
	return m.sendToRadio(msg)
}

// send message to radio, return is handled async
func (m *Mesh) sendToRadio(msg *message.ToRadio) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		log.WithError(err).Error("failure marshalling ToRadio proto")
		return errors.New("could not get radio config")
	}

	err = m.transport.SendToRadio(data)
	if err != nil {
		return err
	}
	return nil
}

func (m *Mesh) receiveFromRadio() {
	for m.stopped == 0 {
		data := <-m.rxChan
		log.Debug("received message from radio")

		var msg message.FromRadio
		err := proto.Unmarshal(data, &msg)
		if err != nil {
			log.WithError(err).Error("Could not marshall proto")
		}
		log.Debug("proto message parsed")

		switch msg.Variant.(type) {
		case *message.FromRadio_MyInfo:
			log.WithField("my_node", msg.GetMyInfo()).Debug("got my node info")
			m.myInfo = msg.GetMyInfo()
		case *message.FromRadio_Radio:
			log.WithField("radio", msg.GetRadio()).Debug("got radio config")
			m.radioConfig = msg.GetRadio()
		case *message.FromRadio_NodeInfo:
			log.WithField("node", msg.GetNodeInfo()).Debug("got node info")
			m.pub(TOPIC_NODE, msg.GetNodeInfo())
		case *message.FromRadio_ConfigCompleteId:
			// TODO: implement this
			log.WithField("node", msg.GetConfigCompleteId()).Debug("got config complete")
		default:
			log.WithField("fromRadio", msg.GetVariant()).Error("unsupported message type")
		}
	}
}

// The pub/sub model will most likely go away after BLE is implemented.
func (m *Mesh) Subscribe(tp Topic, fn func(interface{})) {
	switch tp {
	case TOPIC_NODE:
		m.topic[tp] = append(m.topic[tp], fn)
	case TOPIC_DATA:
	default:
		log.WithField("topic", tp).Error("invalid topic")
	}
}

func (m *Mesh) pub(tp Topic, pkt interface{}) {
	for _, p := range m.topic[tp] {
		p(pkt)
	}
}
