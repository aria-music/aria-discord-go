package aria

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

const READ_LIMIT = 4194304
const TIMEOUT = 30 * time.Second

type client struct {
	sync.RWMutex

	token                  string
	endpoint               string
	streamEndpointOverride string

	stream  chan<- []byte
	botRecv <-chan *request
	botSend chan<- *packet

	conn   *websocket.Conn
	cancel context.CancelFunc
}

func newClient(
	config *config,
	cliToBot chan<- *packet,
	botToCli <-chan *request,
	stream chan<- []byte,
) (*client, error) {
	c := new(client)

	if config == nil {
		return nil, errors.New("config is nil")
	}
	if c.token = config.AriaToken; c.token == "" {
		return nil, errors.New("aria_token is missing in config")
	}
	if c.endpoint = config.AriaEndpoint; c.endpoint == "" {
		c.endpoint = "wss://aria.gaiji.pro/"
	}
	c.streamEndpointOverride = config.StreamEndpointOverride

	c.stream = stream
	c.botRecv = botToCli
	c.botSend = cliToBot

	return c, nil
}

// runner loops

func (c *client) run(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	c.cancel = cancel

	wg := sync.WaitGroup{}
	defer wg.Wait()

	conn, _, err := websocket.Dial(ctx, c.endpoint, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Authorization": []string{"bearer " + c.token},
		},
	})
	if err != nil {
		log.Printf("failed to open websocket: %v\n", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	conn.SetReadLimit(READ_LIMIT)
	c.conn = conn

	if err := c.handleHelloPacket(ctx); err != nil {
		log.Printf("failed to handle hello: %v\n", err)
		return
	}

	wg.Add(1)
	go func() {
		c.recvLoop(ctx)
		wg.Done()
		cancel() // TODO: bad
	}()

	wg.Add(1)
	go func() {
		c.sendLoop(ctx)
		wg.Done()
		cancel() // TODO: bad
	}()

	wg.Wait()
}

func (c *client) sendLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("stopping sendLoop")
			return
		case r := <-c.botRecv:
			go c.sendRequest(ctx, r)
		}
	}
}

func (c *client) recvLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("stopping recvLoop")
			return
		default:
			p := c.recvPacket(ctx)
			if p == nil {
				log.Printf("receive packet errored")
				return
			}
			go c.handlePacket(ctx, p)
		}
	}
}

// utils

func (c *client) sendRequest(parent context.Context, r *request) {
	ctx, cancel := context.WithTimeout(parent, TIMEOUT)
	defer cancel()

	if err := wsjson.Write(ctx, c.conn, r); err != nil {
		log.Printf("failed to send request: %v\n", err)
	}
}

func (c *client) recvPacket(ctx context.Context) *packet {
	p := new(packet)
	if err := wsjson.Read(ctx, c.conn, p); err != nil {
		log.Printf("failed to parse incoming packet: %v\n", err)
		return nil
	}
	return p
}

func (c *client) handlePacket(parent context.Context, p *packet) {
	ctx, cancel := context.WithTimeout(parent, TIMEOUT)
	defer cancel()

	// fill packet data
	dp, ok := providers[p.Type]
	if !ok {
		log.Printf("packet unknown: %s\n", p.Type)
		return
	}

	// TODO: fill nil to Data if unmarshall fails?
	d := dp.data()
	if err := json.Unmarshal(p.RawData, d); err != nil {
		log.Printf("failed to parse packet data: %v\n", err)
		return
	}
	p.Data = d

	// send to bot
	c.sendBotPacket(ctx, p)
}

// handleHelloPacket parses hello packet from server,
// then launch voice stream channel
// TODO: fail on stream goroutine won't make client crash
// this is intended, but seems not a good behaviour
func (c *client) handleHelloPacket(ctx context.Context) error {
	// first read must be hello packet
	p := new(packet)
	if err := wsjson.Read(ctx, c.conn, p); err != nil {
		return err
	}
	if p.Type != "hello" {
		return fmt.Errorf("expect hello, got %s", p.Type)
	}
	hello := new(helloData)
	if err := json.Unmarshal(p.RawData, hello); err != nil {
		return err
	}

	end := hello.Stream
	if c.streamEndpointOverride != "" {
		end = c.streamEndpointOverride
	}
	s, err := newStream(end, hello.Session, c.stream)
	if err != nil {
		return err
	}
	go s.run(ctx)

	return nil
}

func (c *client) sendBotPacket(ctx context.Context, p *packet) {
	select {
	case <-ctx.Done():
	case c.botSend <- p:
	}
}
