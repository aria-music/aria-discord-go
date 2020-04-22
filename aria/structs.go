package aria

import (
	"encoding/json"
	"log"
)

var providers = make(map[string]dataProvider)

func registerProvider(p dataProvider) {
	if _, ok := providers[p.typ()]; ok {
		log.Printf("provider already exists! (%s)", p.typ())
		return
	}
	providers[p.typ()] = p
}

type dataProvider interface {
	typ() string
	data() interface{}
}

// packet is base type
type packet struct {
	Type     string          `json:"type"`
	Postback string          `json:"postback"`
	RawData  json.RawMessage `json:"data"`
	Data     interface{}
}

// entry describes song details
type entry struct {
	Source    string
	Title     string
	URI       string
	Thumbnail string
	Entry     *struct {
		User        string
		SongID      string
		Title       string
		Artist      string
		Album       string
		AlbumArtURL string
	}
}

// event types

type helloData struct {
	Session string
	Stream  string
}

type stateEventData stateData
type queueEventData queueData
type playlistsEventData playlistsData

// packet types

type stateData struct {
	State string
	Entry *struct {
		entry
		IsLiked  bool `json:"is_liked"`
		Duration float64
		Position float64
	}
}

type searchData struct {
	Entries []entry
}

type queueData struct {
	Queue []entry
}

type playlistsData struct {
	Playlists []struct {
		Name       string
		Length     int
		Thumbnails []string
	}
}

type tokenData struct {
	Token string
}

type inviteData struct {
	Invite string
}
