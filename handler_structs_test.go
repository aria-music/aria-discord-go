package aria

import "testing"

var packetHandlers = []struct {
	name string
	eh   interface{}
}{
	{"stateEventHandler", stateEventHandler(nil)},
	{"queueEventHandler", queueEventHandler(nil)},
	{"playlistsEventHandler", playlistsEventHandler(nil)},
	{"statePacketHandler", statePacketHandler(nil)},
	{"searchPacketHandler", searchPacketHandler(nil)},
	{"queuePacketHandler", queuePacketHandler(nil)},
	{"playlistsPacketHandler", playlistsPacketHandler(nil)},
	{"tokenPacketHandler", tokenPacketHandler(nil)},
	{"invitePacketHandler", invitePacketHandler(nil)},
}

func TestPacketHandlerInterfaceImplement(t *testing.T) {
	for _, c := range packetHandlers {
		t.Run(c.name, func(t *testing.T) {
			if _, ok := c.eh.(packetHandler); !ok {
				t.Error("interface not implemented")
			}
		})
	}
}

func TestPacketHandlerDataProviderInterfaceImplement(t *testing.T) {
	for _, c := range packetHandlers {
		t.Run(c.name, func(t *testing.T) {
			if _, ok := c.eh.(dataProvider); !ok {
				t.Error("interface not implemented")
			}
		})
	}
}
