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

const chanTimeout = 30 * time.Second

type bot struct {
	sync.RWMutex
	*discordgo.Session

	token    string
	prefix   string
	keepMsg  keepMsgMap
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
	b.keepMsg = config.keepMsg

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
	// disable auto reconnect to control retry interval by own
	s.ShouldReconnectOnError = false
	b.Session = s
	b.AddHandler(b.onMessage)
	b.AddHandler(b.onReady)
	b.AddHandler(b.onDisconnect)

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
	b.addCmdHandler("queue", b.cmdQueue)
	b.addCmdHandler("summon", b.cmdSummon)
	b.addCmdHandler("invite", b.cmdInvite)
	b.addCmdHandler("token", b.cmdToken)
	b.addCmdHandler("disconnect", b.cmdDisconnect)
	b.addCmdHandler("tweet", b.cmdTweet)
	b.addCmdHandler("version", b.cmdVersion)
	b.addCmdHandler("login", b.cmdLogin)
	b.addCmdHandler("play", b.cmdPlay)
	b.addCmdHandler("playnext", b.cmdPlayNext)
	b.addCmdHandler("like", b.cmdLike)
	b.addCmdHandler("save", b.cmdSave)
	b.addCmdHandler("restart", b.cmdRestart)
	b.addCmdHandler("help", b.cmdHelp)

	// register aria packet handlers
	b.addPacketHandler(onState)
	b.addPacketHandler(onStateEvent)
	b.addPacketHandler(updateOnState)
	b.addPacketHandler(updateOnStateEvent)
	b.addPacketHandler(updateOnQueue)
	b.addPacketHandler(updateOnQueueEvent)
	b.addPacketHandler(updateOnPlaylists)
	b.addPacketHandler(updateOnPlaylistsEvent)
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

	if err := b.Open(); err != nil {
		log.Printf("failed to open Discord connection")
		return
	}
	defer b.Close()

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

	// send client_hello
	go b.sendAriaRequest(&request{
		OP: "hello",
	})

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
	// sometimes it fails to reconnect discord, so we do reconnect by own way
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
		return
	}
	b.RUnlock()

	// if message not starts with command prefix, skip
	if !strings.HasPrefix(m.Content, b.prefix) {
		return
	}
	go b.deleteMessageAfter(m.Message, 0, false)

	// TODO: handle multiple space correctly
	raw := strings.Split(strings.TrimSpace(m.Content), " ")
	rawcmd := strings.ToLower(strings.TrimPrefix(raw[0], b.prefix))
	args := raw[1:] // spec: min <= len <= cap <= max

	cmd := b.resolveCommand(rawcmd)
	if cmd == "" {
		log.Printf("no cmdHandler for %s", rawcmd)
		sendErrorResponse(b, m.ChannelID, fmt.Sprintf("Command not found: `%s`", rawcmd))
		return
	}

	log.Printf("handling command: %s %v", cmd, args)
	for _, h := range b.cmdHandlers[cmd] {
		go h(m.Message, args)
	}
}

// utilities

func (b *bot) resolveCommand(raw string) (cmd string) {
	_, ok := b.cmdHandlers[raw]
	if ok {
		cmd = raw
		return
	}
	// try alias
	if b.alias != nil {
		if al := b.alias.resolve(raw); al != "" {
			if _, ok = b.cmdHandlers[al]; ok {
				cmd = al
			}
		}
	}

	return
}

func (b *bot) sendAriaRequest(r *request) {
	select {
	case <-time.After(chanTimeout):
		log.Printf("failed to send Aria request: %s\n", r.OP)
	case b.ariaSend <- r:
	}
}

func (b *bot) deleteMessageAfter(m *discordgo.Message, t time.Duration, force bool) {
	// if msg is in keepMessageChannel, skip.
	if !force && b.keepMsg.isKeepMsgChannel(m.ChannelID) {
		return
	}

	time.Sleep(t)
	if err := b.ChannelMessageDelete(m.ChannelID, m.ID); err != nil {
		log.Printf("failed to delete message: %v\n", err)
	}
}

// send MessageEmbed to channel then delete message after d
// Returns Message and error immidiately after message is sent
func (b *bot) deleteAfterChannelMessageSendEmbed(
	d time.Duration, force bool,
	channelID string, embed *discordgo.MessageEmbed,
) (*discordgo.Message, error) {
	m, err := b.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		return nil, err
	}
	go b.deleteMessageAfter(m, d, force)
	return m, nil
}
