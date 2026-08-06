package core

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"

	"github.com/sagernet/sing-box/option"
	singjson "github.com/sagernet/sing/common/json"
)

const (
	outboundTypeDirect = "direct"
	outboundTagProxy   = "proxy"
)

// WriteOutboundsConfig writes a sing-box outbounds snippet for the selected
// proxy plus a direct outbound.
func WriteOutboundsConfig(path string, entry option.Outbound) error {
	if entry.Options == nil {
		return fmt.Errorf(
			"outbound %q has empty options (re-fetch the subscription; vault may be from an older save)",
			entry.Tag,
		)
	}

	entry.Tag = outboundTagProxy
	cfg := struct {
		Outbounds []option.Outbound `json:"outbounds"`
	}{
		Outbounds: []option.Outbound{
			entry,
			{Type: outboundTypeDirect, Tag: outboundTypeDirect, Options: map[string]any{}},
		},
	}

	data, err := singjson.MarshalContext(context.Background(), cfg)
	if err != nil {
		return err
	}
	if pretty, prettyErr := prettyJSON(data); prettyErr == nil {
		data = pretty
	}
	return os.WriteFile(path, data, outboundsFilePerm)
}

func prettyJSON(data []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return json.MarshalIndent(v, "", "  ")
}

func outboundToMap(entry option.Outbound) (map[string]any, error) {
	if entry.Options == nil {
		entry.Options = map[string]any{}
	}
	data, err := singjson.MarshalContext(context.Background(), &entry)
	if err != nil {
		return nil, fmt.Errorf("encode outbound: %w", err)
	}
	var m map[string]any
	if err = json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("encode outbound: %w", err)
	}
	return m, nil
}

func outboundFromMap(m map[string]any) option.Outbound {
	if m == nil {
		return option.Outbound{}
	}
	typ, _ := m["type"].(string)
	tag, _ := m["tag"].(string)
	opts := maps.Clone(m)
	delete(opts, "type")
	delete(opts, "tag")
	return option.Outbound{Type: typ, Tag: tag, Options: opts}
}
