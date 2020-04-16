package aria

import (
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// since discordgo's functions does not provide context support,
// we cannot timeout command execution
type cmdHandler func(*discordgo.Message, []string)

var msgAuthor = &discordgo.MessageEmbedFooter{
	Text: "Aria Discord",
	// IconURL: "",
}
var fuckEmotes = []string{
	"🇫", "🇺", "🇨", "🇰", "🖕",
}
var fuckMessage = []string{
	":regional_indicator_f:",
	":regional_indicator_u:",
	":regional_indicator_c:",
	":regional_indicator_k:",
	":regional_indicator_y:",
	":regional_indicator_o:",
	":regional_indicator_u:",
}

func (b *bot) cmdFuck(m *discordgo.Message, _ []string) {
	e := newEmbed()
	e.Color = rand.Intn(0x1000000)
	e.Description = strings.Join(fuckMessage, " ")
	// TODO: mention

	m, err := b.ChannelMessageSendEmbed(m.ChannelID, e)
	if err != nil {
		log.Printf("failed to send message: %v\n", err)
		return
	}
	for _, emoji := range fuckEmotes {
		if err := b.MessageReactionAdd(m.ChannelID, m.ID, emoji); err != nil {
			log.Printf("failed to add reaction: %v\n", err)
		}
	}
}

func (b *bot) cmdSkip(_ *discordgo.Message, _ []string) {
	b.sendAriaRequest(&request{
		OP: "skip",
	})
}

func (b *bot) cmdPause(_ *discordgo.Message, _ []string) {
	b.sendAriaRequest(&request{
		OP: "pause",
	})
}

func (b *bot) cmdResume(_ *discordgo.Message, _ []string) {
	b.sendAriaRequest(&request{
		OP: "resume",
	})
}

func (b *bot) cmdShuffle(_ *discordgo.Message, _ []string) {
	b.sendAriaRequest(&request{
		OP: "shuffle",
	})
}

func (b *bot) cmdClear(_ *discordgo.Message, _ []string) {
	b.sendAriaRequest(&request{
		OP: "clear_queue",
	})
}

func (b *bot) cmdUpdateDB(m *discordgo.Message, args []string) {
	if len(args) < 1 {
		sendErrorResponse(b, m.ChannelID, "No user is given!")
		return
	}

	b.sendAriaRequest(&request{
		OP: "update_db",
		Data: updateDBRequest{
			User: args[0],
		},
	})
}

func (b *bot) cmdRepeat(m *discordgo.Message, args []string) {
	count := 1
	if len(args) >= 1 {
		if i, err := strconv.Atoi(args[0]); err == nil {
			count = i
		}
	}

	state := b.store.getState()
	if state.Entry == nil {
		sendErrorResponse(b, m.ChannelID, "Player is not playing!")
		return
	}

	uri := b.store.getState().Entry.URI
	b.sendAriaRequest(&request{
		OP: "repeat",
		Data: repeatRequest{
			URI:   uri,
			Count: count,
		},
	})
}

func (b *bot) cmdNowPlaying(m *discordgo.Message, args []string) {
	b.sendAriaRequest(&request{
		OP:       "state",
		Postback: m.ChannelID,
	})
}

func (b *bot) cmdSummon(m *discordgo.Message, _ []string) {
	g, err := b.Guild(m.GuildID)
	if err != nil {
		log.Printf("failed to get guild information")
		return
	}

	var vid string
	for _, vs := range g.VoiceStates {
		if vs.UserID == m.Author.ID {
			vid = vs.ChannelID
			break
		}
	}

	if vid == "" {
		sendErrorResponse(b, m.ChannelID, "You are not in voice channel.\nJoin voice first.")
		return
	}

	_, err = b.ChannelVoiceJoin(g.ID, vid, false, false)
	if err != nil {
		log.Printf("failed to join voice: %v\n", err)
	}
}

func (b *bot) cmdDisconnect(m *discordgo.Message, _ []string) {
	b.Session.RLock()
	v, ok := b.VoiceConnections[m.GuildID]
	b.Session.RUnlock()

	if !ok {
		sendErrorResponse(b, m.ChannelID, "Not in voice channel.")
		return
	}

	if err := v.Disconnect(); err != nil {
		log.Printf("failed to disconnect voice: %v\n", err)
	}
}

func (b *bot) cmdInvite(m *discordgo.Message, _ []string) {
	b.sendAriaRequest(&request{
		OP:       "invite",
		Postback: m.ChannelID,
	})
}

func (b *bot) cmdToken(m *discordgo.Message, _ []string) {
	b.sendAriaRequest(&request{
		OP:       "token",
		Postback: m.ChannelID,
	})
}

// utility functions

func newEmbed() (e *discordgo.MessageEmbed) {
	e = new(discordgo.MessageEmbed)
	e.Footer = msgAuthor
	e.Timestamp = time.Now().Format(time.RFC3339)
	return
}

func sendErrorResponse(b *bot, channelID, message string) {
	e := newEmbed()
	e.Color = 0xff0000
	e.Title = "Error"
	e.Description = message

	if _, err := b.ChannelMessageSendEmbed(channelID, e); err != nil {
		log.Printf("failed to send error embed: %v\n", err)
	}
}
