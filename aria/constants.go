package aria

import (
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var botVersion string = "debug"

var msgAuthor = &discordgo.MessageEmbedFooter{
	Text: "Aria Discord Go",
	// IconURL: "",
}
var fuckEmotes = []string{
	"🇫", "🇺", "🇨", "🇰", "🖕",
}
var fuckMessageSlice = []string{
	":regional_indicator_f:",
	":regional_indicator_u:",
	":regional_indicator_c:",
	":regional_indicator_k:",
	":regional_indicator_y:",
	":regional_indicator_o:",
	":regional_indicator_u:",
}
var fuckMessage = strings.Join(fuckMessageSlice, " ")

var digitEmojis = []string{
	"0️⃣", "1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣",
}
var emoji2Digit = func() (e2d map[string]int) {
	e2d = make(map[string]int)
	for i, emoji := range digitEmojis {
		e2d[emoji] = i
	}
	return
}()

var tweetTemplate = `%s
%s
#NowPlaying`
var msgTimeout = 30 * time.Second

// onSearch
var pageSize int = 5
var windowControlls = []string{
	"✅", "◀", "1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "▶", "🍜", "🚫",
}
