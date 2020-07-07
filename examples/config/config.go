package main

import (
	"time"

	"github.com/nerdoftech/Meshtastic-go/pkg/mesh"
	log "github.com/sirupsen/logrus"
)

// 32 bit key for aes 256
var psk = []byte{0xee, 0xd9, 0x50, 0xfd, 0xda, 0x4e, 0xc7, 0xea,
	0x3a, 0x89, 0xa3, 0x4c, 0xf5, 0xa1, 0x7f, 0xb2,
	0x68, 0x44, 0xb5, 0xb1, 0xc7, 0xf2, 0xeb, 0xaf,
	0xd7, 0x7d, 0xc9, 0x53, 0x70, 0x6c, 0x8c, 0x73}

func main() {
	log.SetLevel(log.DebugLevel)

	m, err := mesh.NewMesh("/dev/ttyUSB0", mesh.TRANSPORT_SERIAL)
	if err != nil {
		log.WithError(err).Fatal("could not connect to port")
	}

	defer m.Close()
	err = m.Connect()
	if err != nil {
		log.WithError(err).Fatal("could not connect to radio")
	}

	time.Sleep(500 * time.Millisecond)
	log.
		WithField("app", "config").
		WithField("my_node", m.GetMyNodeInfo()).
		Info("Got my node info")
	log.
		WithField("app", "config").
		WithField("radio_config", m.GetRadioConfig()).
		Info("Got my config")

	// We will copy the config and make changes to it
	newConfig := *m.GetRadioConfig()

	newConfig.Preferences.PositionBroadcastSecs = 60
	newConfig.Preferences.ScreenOnSecs = 120
	newConfig.Preferences.WifiApMode = true
	newConfig.Preferences.WifiSsid = "lora1"
	newConfig.Preferences.WifiPassword = "1234lora"

	newConfig.ChannelSettings.Name = "lora1"
	newConfig.ChannelSettings.Psk = psk

	err = m.SetRadioConfig(&newConfig)
	if err != nil {
		log.WithError(err).Fatal("could not set radio config")
	}

	time.Sleep(2 * time.Second)
}
