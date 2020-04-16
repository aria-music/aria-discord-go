package aria

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const CHAN_TIMEOUT = 30 * time.Second

type bot struct {
	sync.RWMutex
	*discordgo.Session

	token    string
	prefix   string
	ariaRecv <-chan *packet
	ariaSend chan<- *request

	handlers    map[string][]packetHandler
	cmdHandlers map[string][]cmdHandler
	cancel      context.CancelFunc

	botUser *discordgo.User
	store   store
	alias   *alias
	stream  <-chan []byte
}

// setup methods

func newBot(
	config *config,
	cliToBot <-chan *packet,
	botToCli chan<- *request,
	stream <-chan []byte,
) (*bot, error) {
	b := new(bot)
	b.handlers = make(map[string][]packetHandler)
	b.cmdHandlers = make(map[string][]cmdHandler)
	b.store = store{}

	if config == nil {
		return nil, errors.New("config is nil")
	}
	if b.token = config.DiscordToken; b.token == "" {
		return nil, errors.New("discord_token is missing in config")
	}
	if b.prefix = config.CommandPrefix; b.prefix == "" {
		b.prefix = "."
	}

	b.stream = stream
	b.ariaRecv = cliToBot
	b.ariaSend = botToCli

	if alias, err := newAlias(); err != nil {
		log.Printf("failed to initialize alias. skip")
	} else {
		b.alias = alias
	}

	// initialize Discord session
	s, err := discordgo.New("Bot " + b.token)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize discord bot: %w", err)
	}
	b.Session = s
	b.AddHandler(b.onMessage)
	b.AddHandler(b.onReady)

	// register discord command handlers
	b.addCmdHandler("fuck", b.cmdFuck)
	b.addCmdHandler("skip", b.cmdSkip)
	b.addCmdHandler("pause", b.cmdPause)
	b.addCmdHandler("resume", b.cmdResume)
	b.addCmdHandler("shuffle", b.cmdShuffle)
	b.addCmdHandler("clear", b.cmdClear)
	b.addCmdHandler("repeat", b.cmdRepeat)
	b.addCmdHandler("updatedb", b.cmdUpdateDB)
	b.addCmdHandler("nowplaying", b.cmdNowPlaying)
	b.addCmdHandler("summon", b.cmdSummon)
	b.addCmdHandler("invite", b.cmdInvite)
	b.addCmdHandler("token", b.cmdToken)

	// register aria packet handlers
	b.addPacketHandler(onState)
	b.addPacketHandler(onStateEvent)
	b.addPacketHandler(updateOnState)
	b.addPacketHandler(updateOnStateEvent)
	b.addPacketHandler(onInvite)
	b.addPacketHandler(onToken)

	return b, nil
}

func (b *bot) addPacketHandler(f interface{}) {
	b.Lock()
	defer b.Unlock()

	if h := packetHandlerForFunc(f); h != nil {
		b.handlers[h.typ()] = append(b.handlers[h.typ()], h)
	}
}

func (b *bot) addCmdHandler(cmd string, f interface{}) {
	b.Lock()
	defer b.Unlock()

	if t, ok := f.(func(*discordgo.Message, []string)); ok {
		b.cmdHandlers[cmd] = append(b.cmdHandlers[cmd], cmdHandler(t))
	}
}

// runner

func (b *bot) run(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	b.cancel = cancel

	wg := sync.WaitGroup{}
	defer wg.Wait()

	if err := b.Open(); err != nil {
		log.Printf("failed to open Discord connection")
		return
	}

	// audio stream routine
	wg.Add(1)
	go func() {
		b.streamLoop(ctx)
		wg.Done()
		cancel() // TODO: not good, fix it!
	}()

	// aria packet routine
	wg.Add(1)
	go func() {
		b.ariaLoop(ctx)
		wg.Done()
		cancel() // TODO: not good
	}()

	wg.Wait()
}

func (b *bot) streamLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("stopping streamLoop")
			return
		case a := <-b.stream:
			func() {
				b.Session.RLock()
				defer b.Session.RUnlock()

				for _, v := range b.VoiceConnections {
					select {
					case v.OpusSend <- a:
					default:
						// log.Printf("audio packet dropped! (gid: %s)\n", gid)
					}
				}
			}()
		}
	}
}

func (b *bot) ariaLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("stopping ariaLoop")
			return
		case p := <-b.ariaRecv:
			go b.handlePacket(p)
		}
	}
}

func (b *bot) handlePacket(p *packet) {
	log.Printf("handling packet %s", p.Type)
	for _, h := range b.handlers[p.Type] {
		go h.handle(b, p)
	}
}

// discord event handlers

func (b *bot) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	log.Printf("disconnected from Discord")
	b.cancel()
}

// TODO: when ready, join all AutoJoin channels
func (b *bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	b.Lock()
	defer b.Unlock()

	b.botUser = r.User
}

// parse message from discord, fire cmdHandlers
func (b *bot) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	b.RLock()
	if m.Author.ID == b.botUser.ID {
		// if author is bot, delete after 30s
		go b.deleteMessageAfter(m.Message, 30*time.Second)
		return
	}
	b.RUnlock()

	// if message not starts with command prefix, skip
	if !strings.HasPrefix(m.Content, b.prefix) {
		return
	}
	go b.deleteMessageAfter(m.Message, 0)

	raw := strings.Split(strings.TrimSpace(m.Content), " ")
	cmd := strings.ToLower(strings.TrimPrefix(raw[0], b.prefix))
	args := raw[1:] // spec: min <= len <= cap <= max

	hs, ok := b.cmdHandlers[cmd]
	// try alias
	if !ok {
		if al := b.alias.resolve(cmd); al != "" {
			hs, ok = b.cmdHandlers[al]
		}
	}
	if !ok {
		log.Printf("no cmdHandler for %s", cmd)
		sendErrorResponse(b, m.ChannelID, fmt.Sprintf("Command not found: `%s`", cmd))
		return
	}

	log.Printf("handling command: %s", cmd)
	for _, h := range hs {
		go h(m.Message, args)
	}
}

// utilities

func (b *bot) sendAriaRequest(r *request) {
	select {
	case <-time.After(CHAN_TIMEOUT):
	case b.ariaSend <- r:
	}
}

func (b *bot) deleteMessageAfter(m *discordgo.Message, t time.Duration) {
	time.Sleep(t)
	if err := b.ChannelMessageDelete(m.ChannelID, m.ID); err != nil {
		log.Printf("failed to delete message: %v\n", err)
	}
}
