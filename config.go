package aria

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds all config
type config struct {
	DiscordToken           string `json:"discord_token"`
	AriaToken              string `json:"aria_token"`
	AriaEndpoint           string `json:"aria_endpoint"`
	StreamEndpointOverride string `json:"stream_endpoint_override"`
	CommandPrefix          string `json:"command_prefix"`
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

	return c, nil
}
