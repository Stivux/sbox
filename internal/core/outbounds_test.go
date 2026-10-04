package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Stivux/sbox/internal/core"
	"github.com/sagernet/sing-box/option"
)

func sampleOutbound() option.Outbound {
	return option.Outbound{
		Type: "vless",
		Tag:  "node-1",
		Options: option.VLESSOutboundOptions{
			UUID:          "11111111-1111-1111-1111-111111111111",
			ServerOptions: option.ServerOptions{Server: "example.com", ServerPort: 443},
		},
	}
}

func TestWriteOutboundsConfig(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "outbounds.json")
	must(t, core.WriteOutboundsConfig(path, sampleOutbound()))
	data, err := os.ReadFile(path)
	must(t, err)
	for _, want := range []string{
		`"type": "vless"`, `"tag": "proxy"`, `"type": "direct"`, `"tag": "direct"`, `"server": "example.com"`,
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("output missing %s\n%s", want, data)
		}
	}
}

func TestWriteOutboundsConfigNilOptions(t *testing.T) {
	t.Parallel()

	err := core.WriteOutboundsConfig(filepath.Join(t.TempDir(), "o.json"), option.Outbound{
		Type: "vless", Tag: "broken",
	})
	if err == nil {
		t.Fatal("expected error for nil Options")
	}
}

func TestSaveLoadPreservesOutboundOptions(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "subs.sbox")
	must(t, core.SaveSubscriptions(path, core.SubscriptionsConfig{
		Subscriptions: []core.Subscription{{
			URL: "https://example.com/sub#lab", Title: "lab",
			Outbounds: []core.OutboundEntry{{Remarks: "node-1", Outbound: sampleOutbound()}},
		}},
	}, "test-password"))

	loaded, err := core.LoadSubscriptions(path, "test-password")
	must(t, err)
	ob := loaded.Subscriptions[0].Outbounds[0].Outbound
	if ob.Options == nil {
		t.Fatal("Options nil after vault load")
	}
	outPath := filepath.Join(t.TempDir(), "outbounds.json")
	must(t, core.WriteOutboundsConfig(outPath, ob))
	data, err := os.ReadFile(outPath)
	must(t, err)
	if !strings.Contains(string(data), `"server": "example.com"`) {
		t.Fatalf("server lost after vault round-trip:\n%s", data)
	}
}
