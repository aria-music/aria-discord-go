package aria

// packetHandler handles packet from Aria
type packetHandler interface {
	typ() string
	handle(*bot, *packet)
}

// event

type stateEventHandler func(*bot, string, *stateEventData)

func (eh stateEventHandler) typ() string {
	return "event_player_state_change"
}
func (eh stateEventHandler) data() interface{} {
	return &stateEventData{}
}
func (eh stateEventHandler) handle(bot *bot, p *packet) {
	if t, ok := p.Data.(*stateEventData); ok {
		eh(bot, p.Postback, t)
	}
}

type queueEventHandler func(*bot, string, *queueEventData)

func (eh queueEventHandler) typ() string {
	return "event_queue_change"
}
func (eh queueEventHandler) data() interface{} {
	return &queueEventData{}
}
func (eh queueEventHandler) handle(bot *bot, p *packet) {
	if t, ok := p.Data.(*queueEventData); ok {
		eh(bot, p.Postback, t)
	}
}

type playlistsEventHandler func(*bot, string, *playlistsEventData)

func (eh playlistsEventHandler) typ() string {
	return "event_playlists_change"
}
func (eh playlistsEventHandler) data() interface{} {
	return &playlistsEventData{}
}
func (eh playlistsEventHandler) handle(bot *bot, p *packet) {
	if t, ok := p.Data.(*playlistsEventData); ok {
		eh(bot, p.Postback, t)
	}
}

// packet

type statePacketHandler func(*bot, string, *stateData)

func (ph statePacketHandler) typ() string {
	return "state"
}
func (ph statePacketHandler) data() interface{} {
	return &stateData{}
}
func (ph statePacketHandler) handle(bot *bot, p *packet) {
	if t, ok := p.Data.(*stateData); ok {
		ph(bot, p.Postback, t)
	}
}

type searchPacketHandler func(*bot, string, *searchData)

func (ph searchPacketHandler) typ() string {
	return "search"
}
func (ph searchPacketHandler) data() interface{} {
	return &searchData{}
}
func (ph searchPacketHandler) handle(bot *bot, p *packet) {
	if t, ok := p.Data.(*searchData); ok {
		ph(bot, p.Postback, t)
	}
}

type queuePacketHandler func(*bot, string, *queueData)

func (ph queuePacketHandler) typ() string {
	return "list_queue"
}
func (ph queuePacketHandler) data() interface{} {
	return &queueData{}
}
func (ph queuePacketHandler) handle(bot *bot, p *packet) {
	if t, ok := p.Data.(*queueData); ok {
		ph(bot, p.Postback, t)
	}
}

type playlistsPacketHandler func(*bot, string, *playlistsData)

func (ph playlistsPacketHandler) typ() string {
	return "playlists"
}
func (ph playlistsPacketHandler) data() interface{} {
	return &playlistsData{}
}
func (ph playlistsPacketHandler) handle(bot *bot, p *packet) {
	if t, ok := p.Data.(*playlistsData); ok {
		ph(bot, p.Postback, t)
	}
}

type tokenPacketHandler func(*bot, string, *tokenData)

func (ph tokenPacketHandler) typ() string {
	return "token"
}
func (ph tokenPacketHandler) data() interface{} {
	return &tokenData{}
}
func (ph tokenPacketHandler) handle(bot *bot, p *packet) {
	if t, ok := p.Data.(*tokenData); ok {
		ph(bot, p.Postback, t)
	}
}

type invitePacketHandler func(*bot, string, *inviteData)

func (ph invitePacketHandler) typ() string {
	return "invite"
}
func (ph invitePacketHandler) data() interface{} {
	return &inviteData{}
}
func (ph invitePacketHandler) handle(bot *bot, p *packet) {
	if t, ok := p.Data.(*inviteData); ok {
		ph(bot, p.Postback, t)
	}
}

func packetHandlerForFunc(f interface{}) packetHandler {
	switch t := f.(type) {
	// events
	case func(*bot, string, *stateEventData):
		return stateEventHandler(t)
	case func(*bot, string, *queueEventData):
		return queueEventHandler(t)
	case func(*bot, string, *playlistsEventData):
		return playlistsEventHandler(t)

	// packets
	case func(*bot, string, *stateData):
		return statePacketHandler(t)
	case func(*bot, string, *searchData):
		return searchPacketHandler(t)
	case func(*bot, string, *queueData):
		return queuePacketHandler(t)
	case func(*bot, string, *playlistsData):
		return playlistsPacketHandler(t)
	case func(*bot, string, *tokenData):
		return tokenPacketHandler(t)
	case func(*bot, string, *inviteData):
		return invitePacketHandler(t)
	}

	return nil
}

func init() {
	// special: hello event
	registerProvider(stateEventHandler(nil))
	registerProvider(queueEventHandler(nil))
	registerProvider(playlistsEventHandler(nil))

	// packets
	registerProvider(statePacketHandler(nil))
	registerProvider(searchPacketHandler(nil))
	registerProvider(queuePacketHandler(nil))
	registerProvider(playlistsPacketHandler(nil))
	registerProvider(tokenPacketHandler(nil))
	registerProvider(invitePacketHandler(nil))
}
