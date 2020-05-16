package aria

import (
	"sync"
)

type voice struct {
	sync.RWMutex
	joined map[string]string // channelID -> guildID
}

type voiceState interface {
	// cloneJoined returns map contains channelID -> guildID
	cloneJoined() map[string]string
	recordJoin(guildID, channelID string)
	recordDisconnect(channelID string)
}

func newVoiceState() voiceState {
	return &voice{
		joined: make(map[string]string),
	}
}

func (v *voice) cloneJoined() map[string]string {
	v.RLock()
	defer v.RUnlock()

	ret := make(map[string]string)
	for k, v := range v.joined {
		ret[k] = v
	}

	return ret
}

func (v *voice) recordJoin(guildID, channelID string) {
	v.Lock()
	defer v.Unlock()

	v.joined[channelID] = guildID
}

func (v *voice) recordDisconnect(channelID string) {
	v.Lock()
	defer v.Unlock()

	delete(v.joined, channelID)
}
