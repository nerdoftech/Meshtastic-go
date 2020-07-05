package pkg

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/nerdoftech/Meshtastic-go/pkg/message"
)

// Right now, this is a scratch pad of what was in a PoC

// func somthing()  {

// 	rand.Seed(time.Now().UnixNano())
// 	rn := rand.Uint32()
// 	fmt.Println("Config ID:", rn)
// 	msg := &message.ToRadio{
// 		Variant: &message.ToRadio_WantConfigId{
// 			WantConfigId: rn,
// 		},
// 	}

// 	data, err := proto.Marshal(msg)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// }

func parsePacket(data []byte) {
	var p message.FromRadio
	err := proto.Unmarshal(data, &p)
	if err != nil {
		fmt.Println("Could not marshall proto", err)
	}

	switch x := p.Variant.(type) {
	case *message.FromRadio_MyInfo:
		fmt.Printf("My Info: %+v \n\n", p.Variant)
	case *message.FromRadio_Radio:
		fmt.Println("key", x.Radio.ChannelSettings.Psk)
		fmt.Printf("Radio: %+v \n\n", p.Variant)
	case *message.FromRadio_NodeInfo:
		fmt.Println("Node:", x)
	case *message.FromRadio_ConfigCompleteId:
		fmt.Println("Complete:", x)
	default:
		fmt.Printf("%T\n\n", x)
	}

}
