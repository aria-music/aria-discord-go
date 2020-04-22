package aria

type request struct {
	OP       string      `json:"op"`
	Postback string      `json:"postback,omitempty"`
	Data     interface{} `json:"data,omitempty"`
}

type searchRequest struct {
	Query    string `json:"query"`
	Provider string `json:"provider,omitempty"`
}

type playlistRequest struct {
	Name string `json:"name"`
}

type createPlaylistRequest struct {
	Name string `json:"name"`
}

type deletePyalistRequest struct {
	Name string `json:"name"`
}

type addToPlaylistRequest struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

type removeFromPlaylistRequest struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

type likeRequest struct {
	URI string `json:"uri"`
}

type playRequest struct {
	URI []string `json:"uri"`
}

type skipToRequest struct {
	Index int    `json:"index"`
	URI   string `json:"uri"`
}

type queueRequest struct {
	URI      []string `json:"uri"`
	Head     bool     `json:"head,omitempty"`
	Playlist string   `json:"playlist,omitempty"`
}

type repeatRequest struct {
	URI   string `json:"uri"`
	Count int    `json:"count"`
}

type removeRequest struct {
	URI   string `json:"uri"`
	Index int    `json:"queue"`
}

type editQueueRequest struct {
	Queue []string `json:"queue"`
}

type updateDBRequest struct {
	User string `json:"user,omitempty"`
}
