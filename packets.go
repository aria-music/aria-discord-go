package aria

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func updateOnState(b *bot, _ string, d *stateData) {
	b.store.setState(d)
}

func updateOnStateEvent(b *bot, _ string, d *stateEventData) {
	b.store.setState((*stateData)(d))
}

func onState(b *bot, pb string, d *stateData) {
	if pb == "" {
		return
	}
	if d.State == "stopped" || d.Entry == nil {
		sendErrorResponse(b, pb, "Player is not playing!")
	}

	e := newEmbed()
	e.Color = 0x5ce1ff
	e.Title = d.Entry.Title

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

	if _, err := b.ChannelMessageSendEmbed(pb, e); err != nil {
		log.Printf("failed to send nowplaying embed: %v\n", err)
	}
}

func onStateEvent(b *bot, pb string, ed *stateEventData) {
	d := (*stateData)(ed)
	if d.State == "stopped" || d.Entry == nil {
		if err := b.UpdateStatus(0, ""); err != nil {
			log.Printf("failed to update bot status (stopped): %v\n", err)
		}
		return
	}

	title := d.getPrefixEmoji() + d.Entry.Title
	if err := b.UpdateListeningStatus(title); err != nil {
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
