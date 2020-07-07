package main

import (
	"time"

	"github.com/nerdoftech/Meshtastic-go/pkg/message"

	"github.com/nerdoftech/Meshtastic-go/pkg/mesh"
	log "github.com/sirupsen/logrus"
)

func main() {
	// log.SetLevel(log.DebugLevel)

	m, err := mesh.NewMesh("/dev/ttyUSB0", mesh.TRANSPORT_SERIAL)
	if err != nil {
		log.WithError(err).Fatal()
	}

	m.Subscribe(mesh.TOPIC_NODE, func(msg interface{}) {
		node := msg.(*message.NodeInfo)
		log.WithField("node", node).Info("Got node info")
	})

	err = m.Connect()
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)
	log.WithField("my_node", m.GetMyNodeInfo()).Info("Got my node info")
	log.WithField("radio_config", m.GetRadioConfig()).Info("Got my node info")
}
