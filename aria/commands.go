package aria

import (
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var commandsString string
var aliasesString string

type cmdHandler func(*discordgo.Message, []string)
type cmdHelp struct {
	desc   string
	usages []string
}

var helpText = map[string]*cmdHelp{}

func getHelp(cmd string) *cmdHelp {
	return helpText[cmd]
}
func setHelp(cmd, desc string, usages ...string) {
	if len(usages) == 0 {
		usages = []string{cmd}
	}
	helpText[cmd] = &cmdHelp{
		desc,
		usages,
	}
}

func (b *bot) cmdFuck(m *discordgo.Message, args []string) {
	e := newEmbed()
	e.Color = rand.Intn(0x1000000) // 0x000000 - 0xffffff
	e.Author = &discordgo.MessageEmbedAuthor{
		Name:    m.Author.Username + " says...",
		IconURL: m.Author.AvatarURL(""),
	}

	comment := strings.TrimSpace(strings.Join(args, " "))
	if comment == "" {
		// no comments with command
		e.Description = fuckMessage
	} else {
		e.Description = comment
		e.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "それはそれとして...",
				Value:  fuckMessage,
				Inline: false,
			},
		}
	}

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

func (b *bot) cmdPlay(_ *discordgo.Message, args []string) {
	b.doPlay(args, false)
}

func (b *bot) cmdPlayNext(_ *discordgo.Message, args []string) {
	b.doPlay(args, true)
}

func (b *bot) doPlay(args []string, head bool) {
	// args is splitted by " " (single space) so get back them by joining
	arg := strings.Join(args, " ")

	r := &queueRequest{
		Head: head,
	}
	if low := strings.ToLower(arg); b.store.isPlaylist(low) {
		r.Playlist = low
	} else {
		r.URI = []string{
			arg,
		}
	}

	b.sendAriaRequest(&request{
		OP:   "queue",
		Data: r,
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

func (b *bot) cmdLike(m *discordgo.Message, args []string) {
	var uris []string
	// URI cannot be separated by space character
	if len(args) < 1 {
		state := b.store.getState()
		if state.Entry == nil {
			sendErrorResponse(b, m.ChannelID, "Nothing to like!")
			return
		}

		uris = append(uris, state.Entry.URI)
	} else {
		uris = args
	}

	for _, uri := range uris {
		b.doSave(uri, "Likes")
	}
}

func (b *bot) cmdSave(m *discordgo.Message, args []string) {
	if len(args) < 1 {
		sendErrorResponse(b, m.ChannelID, "Playlist is missing!")
		return
	}
	// len 1 -> playlist
	// len 2 -> uri, playlist
	// playlist must not contain space...
	var playlist string
	var uri string
	if len(args) == 1 {
		state := b.store.getState()
		if state.Entry == nil {
			sendErrorResponse(b, m.ChannelID, "Nothing to save!")
			return
		}
		uri = state.Entry.URI
		playlist = args[0]
	} else {
		uri = args[0]
		playlist = args[1]
	}

	b.doSave(uri, playlist)
}

func (b *bot) doSave(uri string, playlist string) {
	b.sendAriaRequest(&request{
		OP: "add_to_playlist",
		Data: &addToPlaylistRequest{
			Name: playlist,
			URI:  uri,
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
			Name:   "Track " + strconv.Itoa(i+1) + " - " + q.Queue[i].Source,
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

	if _, err := b.deleteAfterChannelMessageSendEmbed(msgTimeout, false, m.ChannelID, e); err != nil {
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

	if _, err := b.deleteAfterChannelMessageSendEmbed(msgTimeout, false, m.ChannelID, e); err != nil {
		log.Printf("failed to send tweet embed: %v\n", err)
	}
}

func (b *bot) cmdSummon(m *discordgo.Message, _ []string) {
	g, err := b.State.Guild(m.GuildID)
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

	if err = b.joinVoice(m.GuildID, vid); err != nil {
		log.Printf("failed to summon voice: %v\n", err)
	}
}

func (b *bot) cmdDisconnect(m *discordgo.Message, _ []string) {
	if err := b.disconnectVoice(m.GuildID); err != nil {
		switch err {
		case errNotInVoice:
			sendErrorResponse(b, m.ChannelID, "Not in voice channel.")
		default:
			log.Printf("failed to disconnect voice: %v\n", err)
		}
	}
}

func (b *bot) cmdLogin(m *discordgo.Message, _ []string) {
	e := newEmbed()
	e.Color = 0x57ffae
	e.Title = "Welcome back!"
	e.URL = "https://aria.gaiji.pro/auth/github/login"
	e.Description = "Click [here](https://aria.gaiji.pro/auth/github/login) to login"
	if _, err := b.deleteAfterChannelMessageSendEmbed(msgTimeout, false, m.ChannelID, e); err != nil {
		log.Printf("Failed to send login embed: %v\n", err)
	}
}

func (b *bot) cmdInvite(m *discordgo.Message, _ []string) {
	b.sendAriaRequest(&request{
		OP:       "invite",
		Postback: m.Author.ID,
	})
}

func (b *bot) cmdToken(m *discordgo.Message, _ []string) {
	b.sendAriaRequest(&request{
		OP:       "token",
		Postback: m.Author.ID,
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

	if _, err := b.deleteAfterChannelMessageSendEmbed(msgTimeout, false, m.ChannelID, e); err != nil {
		log.Printf("failed to send version embed: %v\n", err)
	}
}

func (b *bot) cmdRestart(_ *discordgo.Message, _ []string) {
	b.cancel()
}

func (b *bot) cmdHelp(m *discordgo.Message, args []string) {
	if len(args) < 1 {
		e := newEmbed()
		e.Color = 0x03fc98
		e.Title = "Help"
		e.Description = fmt.Sprintf("Type `%shelp [command]` to get command help", b.prefix)

		if commandsString == "" {
			b.updateCommandsString()
		}
		if aliasesString == "" {
			b.updateAliasesString()
			if aliasesString == "" {
				aliasesString = "No alias configuration."
			}
		}

		e.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "Commands",
				Value:  commandsString,
				Inline: false,
			},
			{
				Name:   "Alias",
				Value:  aliasesString,
				Inline: false,
			},
		}

		if _, err := b.deleteAfterChannelMessageSendEmbed(msgTimeout, false, m.ChannelID, e); err != nil {
			log.Printf("failed to send help: %v\n", err)
		}
	} else {
		cmd := b.resolveCommand(args[0])
		if cmd != "" {
			sendHelp(b, m.ChannelID, cmd)
		} else {
			sendErrorResponse(b, m.ChannelID, fmt.Sprintf("Commnad not found for `%s`", args[0]))
		}
	}
}

func (b *bot) cmdSearch(m *discordgo.Message, args []string) {
	b.doSearch(m, "", args)
}

func (b *bot) cmdGpm(m *discordgo.Message, args []string) {
	b.doSearch(m, "gpm", args)
}

func (b *bot) cmdYoutube(m *discordgo.Message, args []string) {
	b.doSearch(m, "youtube", args)
}

func (b *bot) doSearch(m *discordgo.Message, provider string, rawQuery []string) {
	query := strings.TrimSpace(strings.Join(rawQuery, " "))
	if query == "" {
		sendErrorResponse(b, m.ChannelID, "Search query required")
		return
	}

	b.sendAriaRequest(&request{
		OP:       "search",
		Postback: fmt.Sprintf("%s:%s:%s", m.ChannelID, m.Author.ID, query), // "channelID:userID:searchQuery"
		Data: &searchRequest{
			Query:    query,
			Provider: provider,
		},
	})
}

// utility functions

func (b *bot) updateCommandsString() {
	log.Println("updating commandString")

	cmds := []string{}
	for c := range b.cmdHandlers {
		cmds = append(cmds, fmt.Sprintf("`%s`", c))
	}

	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i] < cmds[j]
	})

	commandsString = strings.Join(cmds, ", ")
}

func (b *bot) updateAliasesString() {
	if b.alias == nil {
		log.Println("No alias file. Ignore.")
		return
	}

	log.Println("updating aliasesString")

	alines := []string{}
	for c, as := range b.alias.Alias {
		fa := []string{}
		for _, a := range as {
			fa = append(fa, fmt.Sprintf("`%s`", a))
		}

		sort.Slice(fa, func(i, j int) bool {
			return fa[i] < fa[j]
		})
		alines = append(alines, fmt.Sprintf("`%s`: %s", c, strings.Join(fa, ", ")))
	}

	sort.Slice(alines, func(i, j int) bool {
		return alines[i] < alines[j]
	})
	aliasesString = strings.Join(alines, "\n")
}

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

	if _, err := b.deleteAfterChannelMessageSendEmbed(msgTimeout, false, channelID, e); err != nil {
		log.Printf("failed to send error embed: %v\n", err)
	}
}

func sendHelp(b *bot, channelID string, cmd string) {
	e := newEmbed()
	e.Color = 0x03fc98
	e.Title = fmt.Sprintf("`%s`", cmd)
	e.Fields = []*discordgo.MessageEmbedField{}

	h := getHelp(cmd)
	if h == nil {
		e.Description = "No document found."
	} else {
		e.Description = h.desc
		if len(h.usages) > 0 {
			fu := []string{}
			for _, u := range h.usages {
				fu = append(fu, fmt.Sprintf("`%s%s`", b.prefix, u))
			}
			e.Fields = append(e.Fields, &discordgo.MessageEmbedField{
				Name:   "Usage",
				Value:  strings.Join(fu, "\n"),
				Inline: false,
			})
		}
	}

	if b.alias != nil {
		if al := b.alias.Alias[cmd]; len(al) > 0 {
			fa := []string{}
			for _, a := range al {
				fa = append(fa, fmt.Sprintf("`%s`", a))
			}

			e.Fields = append(e.Fields, &discordgo.MessageEmbedField{
				Name:   "Alias",
				Value:  strings.Join(fa, ", "),
				Inline: false,
			})
		}
	}

	if _, err := b.deleteAfterChannelMessageSendEmbed(msgTimeout, false, channelID, e); err != nil {
		log.Printf("failed to send help embed: %v\n", err)
	}
}

func init() {
	// TODO: better way?
	setHelp("fuck", "fuck you", "fuck [comment]")
	setHelp("skip", "skip current song")
	setHelp("pause", "pause player")
	setHelp("resume", "resume player")
	setHelp("shuffle", "shuffle player queue")
	setHelp("clear", "clear player queue")
	setHelp("updatedb", "update GPM user DB", "updatedb <UserID>")
	setHelp("play", "add song(s) or playlist to player queue", "play <URI>", "play <PlaylistID>")
	setHelp("playnext", "add song(s) or playlist to head of player queue", "playnext <URI>", "playnext <PlaylistID>")
	setHelp("repeat", "repeat current song", "repeat [count]")
	setHelp("like", "Like song. If no URI is given, like current song.", "like [...URIs]")
	setHelp("save", "Save song to playlist. If no URI is given, save current song.", "save [URI] <PlaylistID>")
	setHelp("nowplaying", "show current song info")
	setHelp("queue", "show current player queue")
	setHelp("tweet", "get tweet link to share current song")
	setHelp("summon", "summon bot to voice channel where you're in")
	setHelp("disconnect", "disconnect bot from voice channel")
	setHelp("login", "get login link of web client")
	setHelp("invite", "get invite link to sign up to web client")
	setHelp("token", "get token bot can use")
	setHelp("version", "show client version")
	setHelp("restart", "kill current discord connection")
	setHelp("search", "search songs")
	setHelp("youtube", "search YouTube songs")
	setHelp("gpm", "search Google Play Music songs")
	setHelp("help", "show help", "help [command]")
}
