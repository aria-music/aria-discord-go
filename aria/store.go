package aria

import (
	"sync/atomic"
	"unsafe"
)

type store struct {
	state     *stateData
	queue     *queueData
	playlists *playlistsData
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
}
