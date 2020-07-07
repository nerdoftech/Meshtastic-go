package mesh

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nerdoftech/Meshtastic-go/pkg/message"
	mt "github.com/nerdoftech/Meshtastic-go/pkg/types"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMesh(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mesh Suite")
}

func fromRadio(pb *message.FromRadio) []byte {
	data, err := proto.Marshal(pb)
	if err != nil {
		log.WithError(err).Fatal("error creating pb")
	}
	return data
}

var _ = Describe("Mesh", func() {
	var mockTransport *mt.MockTransportInterface
	var mesh *Mesh
	BeforeEach(func() {
		crtl := gomock.NewController(GinkgoT())
		mockTransport = mt.NewMockTransportInterface(crtl)
		mesh = &Mesh{
			transport: mockTransport,
			rxChan:    make(chan []byte, 1),
		}
		mesh.topic = make(map[Topic][]func(interface{}))
		for _, tp := range TOPICS {
			mesh.topic[tp] = make([]func(interface{}), 0)
		}
	})
	Context("Connect", func() {
		It("should work", func() {
			mockTransport.EXPECT().Connect().Return(nil)
			mockTransport.EXPECT().Listen()
			mockTransport.EXPECT().SendToRadio(gomock.Any()).Return(nil)
			mockTransport.EXPECT().Close()
			err := mesh.Connect()
			Expect(err).Should(BeNil())
			mesh.Close() // Stop goroutines
		})
		It("should error Connect", func() {
			mockTransport.EXPECT().Connect().Return(errors.New("error"))
			err := mesh.Connect()
			Expect(err).Should(HaveOccurred())
		})
		It("should error getRadioConfig", func() {
			mockTransport.EXPECT().Connect().Return(nil)
			mockTransport.EXPECT().Listen()
			mockTransport.EXPECT().SendToRadio(gomock.Any()).Return(errors.New("error"))
			mockTransport.EXPECT().Close()
			err := mesh.Connect()
			Expect(err).Should(HaveOccurred())
			mesh.Close() // Stop goroutines
		})
	})
	Context("Close", func() {
		It("should work", func() {
			mockTransport.EXPECT().Close()
			mesh.Close()
		})
	})
	Context("GetRadioConfig", func() {
		// Cheap coverage points
		It("GetRadioConfig", func() {
			cfg := &message.RadioConfig{}
			mesh.radioConfig = cfg
			rc := mesh.GetRadioConfig()
			Expect(rc).Should(Equal(cfg))
		})
		It("getRadioConfig should work", func() {
			mockTransport.EXPECT().
				SendToRadio(gomock.AssignableToTypeOf([]byte{})).
				Return(nil)
			err := mesh.getRadioConfig()
			Expect(err).Should(BeNil())
		})
	})
	Context("sendToRadio", func() {
		It("should work", func() {
			mockTransport.EXPECT().
				SendToRadio(gomock.AssignableToTypeOf([]byte{})).
				Return(nil)
			msg := &message.ToRadio{
				Variant: &message.ToRadio_WantConfigId{
					WantConfigId: 1,
				},
			}
			err := mesh.sendToRadio(msg)
			Expect(err).Should(BeNil())
		})
		It("error in sending to radio", func() {
			mockTransport.EXPECT().
				SendToRadio(gomock.AssignableToTypeOf([]byte{})).
				Return(errors.New("error"))
			msg := &message.ToRadio{}
			err := mesh.sendToRadio(msg)
			Expect(err).Should(HaveOccurred())
		})
	})
	Context("receiveFromRadio", func() {
		It("should work", func() {
			go mesh.receiveFromRadio()

			// Cover ConfigCompleteId
			pb := &message.FromRadio{
				Variant: &message.FromRadio_ConfigCompleteId{
					ConfigCompleteId: 1234,
				},
			}
			mesh.rxChan <- fromRadio(pb)

			// Should get MyNodeInfo
			var exp1 uint32 = 123456
			pb = &message.FromRadio{
				Variant: &message.FromRadio_MyInfo{
					MyInfo: &message.MyNodeInfo{
						MyNodeNum: exp1,
					},
				},
			}
			mesh.rxChan <- fromRadio(pb)

			Eventually(func() uint32 {
				if mesh.myInfo != nil {
					return mesh.myInfo.MyNodeNum
				}
				return 0
			}).Should(Equal(exp1))

			// Should get RadioConfig
			exp2 := "A horse with no name"
			pb = &message.FromRadio{
				Variant: &message.FromRadio_Radio{
					Radio: &message.RadioConfig{
						ChannelSettings: &message.ChannelSettings{
							Name: exp2,
						},
					},
				},
			}
			mesh.rxChan <- fromRadio(pb)

			Eventually(func() string {
				if mesh.radioConfig != nil {
					return mesh.radioConfig.ChannelSettings.Name
				}
				return ""
			}).Should(Equal(exp2))

			// Should get NodeInfo
			var node *message.NodeInfo
			cb := func(n interface{}) {
				node = n.(*message.NodeInfo)
			}
			mesh.Subscribe(TOPIC_NODE, cb)
			pb = &message.FromRadio{
				Variant: &message.FromRadio_NodeInfo{
					NodeInfo: &message.NodeInfo{
						Num: exp1,
					},
				},
			}
			mesh.rxChan <- fromRadio(pb)

			Eventually(func() uint32 {
				if node != nil {
					return node.Num
				}
				return 0
			}).Should(Equal(exp1))
		})
	})
})
