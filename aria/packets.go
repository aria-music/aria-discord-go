package aria

import (
	"fmt"
	"log"

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
		emoji = "🧡" + emoji
	} else {
		emoji = "🤍" + emoji
	}
	if d.State == "paused" {
		emoji = "⏸" + emoji
	}

	if emoji != "" {
		emoji = emoji + " "
	}

	return
}
