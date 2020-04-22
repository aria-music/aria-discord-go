package aria

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type store struct {
	// give up. how to deal with map using atomic / unsafe?
	sync.RWMutex

	state     *stateData
	queue     *queueData
	playlists *playlistsData

	mappedPlaylists map[string]struct{}
}

func (s *store) getState() *stateData {
	return (*stateData)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.state))))
}

func (s *store) setState(d *stateData) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&s.state)), unsafe.Pointer(d))
}

func (s *store) getQueue() *queueData {
	return (*queueData)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.queue))))
}

func (s *store) setQueue(d *queueData) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&s.queue)), unsafe.Pointer(d))
}

func (s *store) getPlaylists() *playlistsData {
	return (*playlistsData)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.playlists))))
}

func (s *store) setPlaylists(d *playlistsData) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&s.playlists)), unsafe.Pointer(d))
	s.makeMappedPlaylists(d)
}

func (s *store) isPlaylist(name string) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.mappedPlaylists[name]
	return ok
}

func (s *store) makeMappedPlaylists(d *playlistsData) {
	m := d.getMapped()
	s.Lock()
	defer s.Unlock()
	s.mappedPlaylists = m
}

// utils

func (d *playlistsData) getMapped() map[string]struct{} {
	m := make(map[string]struct{})
	for _, p := range d.Playlists {
		m[p.Name] = struct{}{}
	}

	return m
}
