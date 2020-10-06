package aria

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func updateOnState(b *bot, _ string, d *stateData) {
	b.store.setState(d)
}

func updateOnStateEvent(b *bot, _ string, d *stateEventData) {
	b.store.setState((*stateData)(d))
}

func updateOnQueue(b *bot, _ string, q *queueData) {
	b.store.setQueue(q)
}

func updateOnQueueEvent(b *bot, _ string, q *queueEventData) {
	b.store.setQueue((*queueData)(q))
}

func updateOnPlaylists(b *bot, _ string, p *playlistsData) {
	b.store.setPlaylists(p)
}

func updateOnPlaylistsEvent(b *bot, _ string, p *playlistsEventData) {
	b.store.setPlaylists((*playlistsData)(p))
}

func onState(b *bot, pb string, d *stateData) {
	if pb == "" {
		return
	}
	if d.State == "stopped" || d.Entry == nil {
		sendErrorResponse(b, pb, "Player is not playing!")
		return
	}

	e := newEmbed()
	e.Color = 0x5ce1ff
	e.Title = d.Entry.Title
	// show song duration / position as embed Description
	e.Description = fmt.Sprintf(":arrow_forward: **%s** / **%s**", durationString(d.Entry.Position), durationString(d.Entry.Duration))

	if d.Entry.Thumbnail != "" {
		e.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: d.Entry.Thumbnail,
		}
	}

	fields := make([]*discordgo.MessageEmbedField, 0, 5)
	if d.Entry.Entry != nil {
		e.Title = d.Entry.Entry.Title

		if d.Entry.Entry.Artist != "" {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Artist",
				Value:  d.Entry.Entry.Artist,
				Inline: false,
			})
		}
		if d.Entry.Entry.Album != "" {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Album",
				Value:  d.Entry.Entry.Album,
				Inline: false,
			})
		}
		if d.Entry.Entry.User != "" {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Owned by",
				Value:  d.Entry.Entry.User,
				Inline: true,
			})
		}
	}

	// general fields
	fields = append(fields,
		&discordgo.MessageEmbedField{
			Name:   "Source",
			Value:  d.Entry.Source,
			Inline: true,
		},
		&discordgo.MessageEmbedField{
			Name:   "URI",
			Value:  d.Entry.URI,
			Inline: false,
		},
	)

	e.Fields = fields
	e.Title = d.getPrefixEmoji() + e.Title

	if _, err := b.deleteAfterChannelMessageSendEmbed(msgTimeout, false, pb, e); err != nil {
		log.Printf("failed to send nowplaying embed: %v\n", err)
	}
}

func onInvite(b *bot, pb string, d *inviteData) {
	if pb == "" {
		return
	}

	uc, err := b.UserChannelCreate(pb)
	if err != nil {
		log.Printf("failed to create user channel: %v", err)
		return
	}

	e := newEmbed()
	e.Color = 0x57ffae
	e.Title = "Welcome!"
	e.Description = fmt.Sprintf("Click [here](https://aria.gaiji.pro/auth/github/register?invite=%s) to register", d.Invite)
	e.Fields = []*discordgo.MessageEmbedField{
		{
			Name:  "Invite code",
			Value: fmt.Sprintf("`%s`", d.Invite),
		},
	}

	if _, err := b.ChannelMessageSendEmbed(uc.ID, e); err != nil {
		log.Printf("failed to send invite embed: %v\n", err)
		return
	}
}

func onToken(b *bot, pb string, d *tokenData) {
	if pb == "" {
		return
	}

	uc, err := b.UserChannelCreate(pb)
	if err != nil {
		log.Printf("failed to create user channel: %v", err)
		return
	}

	contents := []string{
		"Your new token is:",
		d.Token,
	}

	for _, content := range contents {
		if _, err := b.deleteAfterChannelMessageSend(msgTimeout, true, uc.ID, content); err != nil {
			log.Printf("failed to send token embed: %v\n", err)
			return
		}
	}
}

func onSearch(b *bot, pb string, d *searchData) {
	if pb == "" {
		return
	}

	// postback "channelID:userID:searchQuery"
	pbs := strings.SplitN(pb, ":", 3)
	if len(pbs) != 3 {
		log.Printf("invalid postback style (%s)", pb)
		return
	}
	channelID := pbs[0]
	userID := pbs[1]
	searchQuery := pbs[2]

	// check search results
	if len(*d) == 0 {
		sendErrorResponse(b, channelID, fmt.Sprintf("No songs were found for `%s`", searchQuery))
		return
	}

	window := newSearchWindow(searchQuery, *d, pageSize)
	wMsg, err := b.ChannelMessageSendEmbed(channelID, window.render())
	if err != nil {
		log.Printf("failed to send: %v", err)
		return
	}
	defer b.deleteMessageAfter(wMsg, 0, false)

	cc, cancel, err := b.openReactor(wMsg, userID, windowControlls, 30*time.Second)
	if err != nil {
		log.Printf("failed to open reactor: %v", err)
		return
	}
	defer cancel()

	changed := false
	cancelled := false
listen:
	for emoji := range cc {
		changed = false
		switch emoji {
		case "◀":
			changed = window.prev()
		case "▶":
			changed = window.next()
		case "🚫":
			cancelled = true
			break listen
		case "🍜":
			window.selectAll()
			break listen
		case "✅":
			break listen
		case "1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣":
			changed = window.toggleEntry(emoji2Digit[emoji] - 1)
		}

		if changed {
			b.ChannelMessageEditEmbed(wMsg.ChannelID, wMsg.ID, window.render())
		}
	}
	b.MessageReactionsRemoveAll(wMsg.ChannelID, wMsg.ID)

	if cancelled {
		return
	}

	if uris := window.dump(); len(uris) > 0 {
		b.sendAriaRequest(&request{
			OP: "queue",
			Data: &queueRequest{
				URI: uris,
			},
		})
	}
}

func onStateEvent(b *bot, pb string, ed *stateEventData) {
	d := (*stateData)(ed)
	if d.State == "stopped" || d.Entry == nil {
		// TODO: which is better: disable updating status for player stop event,
		// or using lock to serialize?

		// if err := b.UpdateStatus(0, ""); err != nil {
		// 	log.Printf("failed to update bot status (stopped): %v\n", err)
		// }
		return
	}

	title := d.getPrefixEmoji() + d.Entry.Title
	if err := b.UpdateStatus(0, title); err != nil {
		log.Printf("failed to update bot status: %v\n", err)
	}
}

// utils

func (d *stateData) getPrefixEmoji() (emoji string) {
	if d.Entry == nil {
		return
	}

	if d.Entry.IsLiked {
		emoji = "🧡 " + emoji
	} else {
		emoji = "🤍 " + emoji
	}
	if d.State == "paused" {
		emoji = "⏸ " + emoji
	}

	return
}

type searchWindow struct {
	query    string
	entries  []entry
	pagesize int

	maxPages    int
	currentPage int
	selected    map[string]struct{} // set of URI
}

func newSearchWindow(query string, entries []entry, pagesize int) *searchWindow {
	maxPages := len(entries) / pagesize
	if len(entries)%pagesize > 0 {
		maxPages++
	}
	return &searchWindow{
		query:       query,
		entries:     entries,
		pagesize:    pagesize,
		maxPages:    maxPages,
		currentPage: 0,
		selected:    make(map[string]struct{}),
	}
}

func (w *searchWindow) render() (e *discordgo.MessageEmbed) {
	e = newEmbed()
	e.Color = 0xff386a
	e.Title = fmt.Sprintf("Search - `%s`", w.query)
	e.Description = fmt.Sprintf("`%d` results, Page `%d/%d`\n`%d` selected", len(w.entries), w.currentPage+1, w.maxPages, len(w.selected))

	startIdx := w.currentPage * w.pagesize
	for i, entry := range w.entries[startIdx:min(startIdx+w.pagesize, len(w.entries))] {
		// log.Printf("entry index %d", i)
		header := digitEmojis[i+1]
		if _, ok := w.selected[entry.URI]; ok {
			header += " ✅"
		}

		meta := fmt.Sprintf("from `%s`", entry.Source)
		if entry.Entry.User != "" {
			meta = fmt.Sprintf("%s `(%s)`", meta, entry.Entry.User)
		}
		if entry.Entry.Album != "" {
			meta = fmt.Sprintf("%s, in `%s`", meta, entry.Entry.Album)
		}

		e.Fields = append(e.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s %s", header, entry.Title),
			Value:  meta,
			Inline: false,
		})
	}

	return e
}

func (w *searchWindow) next() (changed bool) {
	if (w.currentPage+1)*w.pagesize >= len(w.entries) {
		// index overflow
		return false
	}
	w.currentPage++
	return true
}

func (w *searchWindow) prev() (changed bool) {
	if (w.currentPage-1)*w.pagesize < 0 {
		// index underflow
		return false
	}
	w.currentPage--
	return true
}

// toggleEntry toggles entry in the current page specified by idx.
// idx must be between 0 and pagesize.
func (w *searchWindow) toggleEntry(viewIdx int) (changed bool) {
	idx := w.currentPage*w.pagesize + viewIdx
	if idx >= len(w.entries) || idx < 0 {
		log.Printf("index out of range (entries: %d, index: %d)", len(w.entries), idx)
		return false
	}

	entry := w.entries[idx]
	if _, ok := w.selected[entry.URI]; ok {
		// if already selected, deselect.
		delete(w.selected, entry.URI)
	} else {
		// not selected, select!
		w.selected[entry.URI] = struct{}{}
	}

	// FIXME: temporary always return false to avoid rate-limiting
	return false
	// return true
}

func (w *searchWindow) selectAll() {
	for _, entry := range w.entries {
		w.selected[entry.URI] = struct{}{}
	}
}

func (w *searchWindow) dump() []string {
	d := []string{}
	for uri := range w.selected {
		d = append(d, uri)
	}
	return d
}
