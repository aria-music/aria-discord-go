package aria

import (
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// since discordgo's functions does not provide context support,
// we cannot timeout command execution
type cmdHandler func(*discordgo.Message, []string)

var botVersion string = "debug"

var msgAuthor = &discordgo.MessageEmbedFooter{
	Text: "Aria Discord Go",
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
var digitEmojis = []string{
	"0️⃣", ":one:", "2️⃣", "3️⃣", "4️⃣", "5️⃣",
}
var tweetTemplate = `%s
%s
#NowPlaying`

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

func (b *bot) cmdQueue(m *discordgo.Message, _ []string) {
	q := b.store.getQueue()
	s := b.store.getState()
	if q == nil {
		sendErrorResponse(b, m.ChannelID, "Something went wrong. Try again later.")
		b.sendAriaRequest(&request{
			OP: "list_queue",
		})
		return
	}
	if s == nil || s.Entry == nil {
		sendErrorResponse(b, m.ChannelID, "Something went wrong. Try again later.")
		b.sendAriaRequest(&request{
			OP: "state",
		})
		return
	}

	e := newEmbed()
	e.Color = 0xff955c
	e.Title = s.getPrefixEmoji() + s.Entry.Title

	flen := len(q.Queue)
	if flen > 5 {
		flen = 5
	}

	e.Fields = []*discordgo.MessageEmbedField{}
	for i := 0; i < flen; i++ {
		e.Fields = append(e.Fields, &discordgo.MessageEmbedField{
			Name:   "Track " + strconv.Itoa(i+1),
			Value:  digitEmojis[i+1] + " " + q.Queue[i].Title,
			Inline: false,
		})
	}

	if len(q.Queue) > 5 {
		e.Fields = append(e.Fields, &discordgo.MessageEmbedField{
			Name:  "And...",
			Value: "🈵 **" + strconv.Itoa(len(q.Queue)-5) + "** more songs",
		})
	}

	if _, err := b.ChannelMessageSendEmbed(m.ChannelID, e); err != nil {
		log.Printf("failed to send error embed: %v\n", err)
		return
	}
}

func (b *bot) cmdTweet(m *discordgo.Message, _ []string) {
	state := b.store.getState()
	euri := state.Entry.URI
	if !strings.HasPrefix(euri, "http") {
		euri = "https://play.google.com/music/listen"
	}
	text := fmt.Sprintf(tweetTemplate, state.Entry.Title, euri)
	url := "https://twitter.com/intent/tweet?text=" + url.PathEscape(text)

	e := newEmbed()
	e.Color = 0x1da1f2
	e.Title = "Tweet"
	e.URL = url
	e.Description = fmt.Sprintf("Click [here](%s) to tweet!", url)

	if _, err := b.ChannelMessageSendEmbed(m.ChannelID, e); err != nil {
		log.Printf("failed to send tweet embed: %v\n", err)
	}
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

func (b *bot) cmdLogin(m *discordgo.Message, _ []string) {
	e := newEmbed()
	e.Color = 0x57ffae
	e.Title = "Welcome back!"
	e.URL = "https://aria.gaiji.pro/auth/github/login"
	e.Description = "Click [here](https://aria.gaiji.pro/auth/github/login) to login"
	if _, err := b.ChannelMessageSendEmbed(m.ChannelID, e); err != nil {
		log.Printf("Failed to send login embed: %v\n", err)
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

func (b *bot) cmdVersion(m *discordgo.Message, _ []string) {
	e := newEmbed()
	e.Color = 0xff7092
	e.Title = "Aria Discord Go"
	e.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "Version",
			Value:  fmt.Sprintf("`%s`", botVersion),
			Inline: false,
		},
		{
			Name:   "GitHub",
			Value:  "https://github.com/aria-music/aria-discord-go",
			Inline: false,
		},
	}

	if _, err := b.ChannelMessageSendEmbed(m.ChannelID, e); err != nil {
		log.Printf("failed to send version embed: %v\n", err)
	}
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
