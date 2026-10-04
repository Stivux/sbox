package parser_test

import (
	"net/url"
	"slices"
	"testing"

	"github.com/Stivux/sbox/internal/core/parser"
)

func TestParseVLESSKeepsALPN(t *testing.T) {
	t.Parallel()

	for _, security := range []string{"tls", "reality"} {
		t.Run(security, func(t *testing.T) {
			t.Parallel()
			raw := "vless://11111111-1111-1111-1111-111111111111@1.2.3.4:443" +
				"?security=" + security +
				"&sni=a.example&fp=chrome&pbk=pubkey&sid=abcd&alpn=h2,http/1.1&type=tcp#n"
			u, err := url.Parse(raw)
			if err != nil {
				t.Fatal(err)
			}
			opts, _, err := parser.ParseVLESS(u)
			if err != nil {
				t.Fatal(err)
			}
			if opts.TLS == nil {
				t.Fatal("TLS is nil")
			}
			got := []string(opts.TLS.ALPN)
			want := []string{"h2", "http/1.1"}
			if !slices.Equal(got, want) {
				t.Fatalf("ALPN = %q, want %q", got, want)
			}
		})
	}
}
