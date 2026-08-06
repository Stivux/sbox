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
			UUID: "11111111-1111-1111-1111-111111111111",
			ServerOptions: option.ServerOptions{
				Server:     "example.com",
				ServerPort: 443,
			},
		},
	}
}

func TestWriteOutboundsConfig(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "outbounds.json")
	if err := core.WriteOutboundsConfig(path, sampleOutbound()); err != nil {
		t.Fatalf("WriteOutboundsConfig: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	text := string(data)
	for _, want := range []string{
		`"type": "vless"`,
		`"tag": "proxy"`,
		`"type": "direct"`,
		`"tag": "direct"`,
		`"server": "example.com"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %s\n%s", want, text)
		}
	}
}

func TestWriteOutboundsConfigNilOptions(t *testing.T) {
	t.Parallel()

	err := core.WriteOutboundsConfig(filepath.Join(t.TempDir(), "o.json"), option.Outbound{
		Type: "vless",
		Tag:  "broken",
	})
	if err == nil {
		t.Fatal("expected error for nil Options")
	}
}

func TestSaveLoadPreservesOutboundOptions(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "subs.sbox")
	cfg := core.SubscriptionsConfig{
		Subscriptions: []core.Subscription{{
			URL:   "https://example.com/sub#lab",
			Title: "lab",
			Outbounds: []core.OutboundEntry{{
				Remarks:  "node-1",
				Outbound: sampleOutbound(),
			}},
		}},
	}

	if saveErr := core.SaveSubscriptions(path, cfg, "test-password"); saveErr != nil {
		t.Fatalf("SaveSubscriptions: %v", saveErr)
	}

	loaded, err := core.LoadSubscriptions(path, "test-password")
	if err != nil {
		t.Fatalf("LoadSubscriptions: %v", err)
	}

	ob := loaded.Subscriptions[0].Outbounds[0].Outbound
	if ob.Options == nil {
		t.Fatal("Options nil after vault load")
	}

	outPath := filepath.Join(t.TempDir(), "outbounds.json")
	if writeErr := core.WriteOutboundsConfig(outPath, ob); writeErr != nil {
		t.Fatalf("WriteOutboundsConfig after load: %v", writeErr)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"server": "example.com"`) {
		t.Fatalf("server lost after vault round-trip:\n%s", data)
	}
}
