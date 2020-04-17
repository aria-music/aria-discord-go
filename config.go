package aria

import (
	"encoding/json"
	"fmt"
	"os"
)

type keepMsgMap map[string]struct{}

// Config holds all config
type config struct {
	DiscordToken           string   `json:"discord_token"`
	AriaToken              string   `json:"aria_token"`
	AriaEndpoint           string   `json:"aria_endpoint"`
	StreamEndpointOverride string   `json:"stream_endpoint_override"`
	CommandPrefix          string   `json:"command_prefix"`
	KeepMessageChannel     []string `json:"keep_message_channel"`

	keepMsg keepMsgMap
}

func newConfig() (*config, error) {
	// TODO: make configurable
	file := "config.json"

	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open file (%s): %w", file, err)
	}
	defer f.Close()

	c := new(config)
	err = json.NewDecoder(f).Decode(c)
	if err != nil {
		return nil, fmt.Errorf("failed to decode json: %w", err)
	}

	c.initKeepMsg()
	return c, nil
}

func (c *config) initKeepMsg() {
	c.keepMsg = make(map[string]struct{})
	for _, ch := range c.KeepMessageChannel {
		c.keepMsg[ch] = struct{}{}
	}
}

func (m keepMsgMap) isKeepMsgChannel(chID string) bool {
	_, ok := m[chID]
	return ok
}
