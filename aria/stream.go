package aria

import (
	"context"
	"errors"
	"log"
	"time"

	"nhooyr.io/websocket"
)

type stream struct {
	endpoint   string
	session    string
	streamChan chan<- []byte
}

func newStream(endpoint, session string, streamChan chan<- []byte) (*stream, error) {
	s := new(stream)

	if s.endpoint = endpoint; s.endpoint == "" {
		return nil, errors.New("stream endpoint is missing")
	}
	if s.session = session; s.session == "" {
		return nil, errors.New("stream session key is missing")
	}
	s.streamChan = streamChan

	return s, nil
}

func (s *stream) run(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	ct := 0 * time.Second
	for {
		select {
		case <-ctx.Done():
			log.Printf("stopping stream\n")
			return
		case <-time.After(ct):
		}

		conn, _, err := websocket.Dial(ctx, s.endpoint, nil)
		if err != nil {
			log.Printf("failed to open stream websocket: %v\n", err)
			ct = 30 * time.Second
			continue
		}

		if err := conn.Write(ctx, websocket.MessageText, ([]byte)(s.session)); err != nil {
			log.Printf("failed to send session key: %v\n", err)
			return
		}

		s.streamLoop(ctx, conn)
	}
}

func (s *stream) streamLoop(ctx context.Context, conn *websocket.Conn) {
	for {
		t, b, err := conn.Read(ctx)
		if err != nil {
			log.Printf("failed to read: %v\n", err)
			return
		}

		if t != websocket.MessageBinary {
			log.Printf("invalid message type!")
			return
		}

		select {
		case s.streamChan <- b:
		default:
		}
	}
}
