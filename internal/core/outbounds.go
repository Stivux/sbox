package core

import (
	"context"
	"encoding/json"
	"os"

	"github.com/sagernet/sing-box/option"
	singjson "github.com/sagernet/sing/common/json"
)

type outboundsConfig struct {
	Outbounds []option.Outbound `json:"outbounds"`
}

func WriteOutboundsConfig(path string, entry option.Outbound) error {
	entry.Tag = "proxy"

	directOutbound := option.Outbound{
		Type: "direct",
		Tag:  "direct",
	}

	cfg := outboundsConfig{
		Outbounds: []option.Outbound{entry, directOutbound},
	}

	ctx := context.Background()

	data, err := singjson.MarshalContext(ctx, cfg)
	if err != nil {
		return err
	}

	// optional pretty-print
	var pretty any
	var prettyBytes []byte
	if err = json.Unmarshal(data, &pretty); err == nil {
		if prettyBytes, err = json.MarshalIndent(pretty, "", "  "); err == nil {
			data = prettyBytes
		}
	}

	return os.WriteFile(path, data, standardFilePermissions)
}
