package aria

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func durationString(rawdur float64) (dstr string) {
	dur := time.Duration(rawdur) * time.Second
	seconds := (dur % time.Minute) / time.Second
	minutes := (dur % time.Hour) / time.Minute
	hours := dur / time.Hour

	dstr = fmt.Sprintf("%2d:%02d", minutes, seconds)
	if hours > 0 {
		dstr = fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}

	return
}

func embedFieldFromEntry(entry *entry, header string) *discordgo.MessageEmbedField {
	meta := fmt.Sprintf("from `%s`", entry.Source)
	if entry.Entry != nil {
		if entry.Entry.User != "" {
			meta = fmt.Sprintf("%s `(%s)`", meta, entry.Entry.User)
		}
		if entry.Entry.Album != "" {
			meta = fmt.Sprintf("%s, in `%s`", meta, entry.Entry.Album)
		}
	}

	return &discordgo.MessageEmbedField{
		Name:   header + entry.Title,
		Value:  meta,
		Inline: false,
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
