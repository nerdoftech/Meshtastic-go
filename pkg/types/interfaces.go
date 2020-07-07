package types

//go:generate mockgen --source interfaces.go -destination mock.go -package types

import (
	"io"

	"github.com/nerdoftech/Meshtastic-go/pkg/message"
)

// MeshInterface is for other componets to interact with meshtastic network
type MeshInterface interface {
	Connect() error
	GetRadioConfig() *message.RadioConfig
	Close()
}

// TransportInterface is for transport mediums such as BLE, serial, wifi
type TransportInterface interface {
	Connect() error
	// Send proto encoded message, dont include transport specifics (e.g. serial header)
	SendToRadio([]byte) error
	Listen()
	Close()
}

// ReadCloseWriteFlusher adds Flush() to ReadWriteCloser
type ReadWriteCloseFlusher interface {
	io.ReadWriteCloser
	Flush() error
}
