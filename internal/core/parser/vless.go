package parser

import (
	"errors"
	"net/url"
	"strings"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

var (
	ErrInvalidScheme = errors.New("vless: invalid scheme, expected vless://")
	ErrInvalidURL    = errors.New("vless: invalid URL")
)

// ParseVLESS parses a vless:// link into VLESSOutboundOptions and remarks.
func ParseVLESS(uri *url.URL) (option.VLESSOutboundOptions, string, error) {
	if uri == nil {
		return option.VLESSOutboundOptions{}, "", ErrInvalidURL
	}
	if uri.Scheme != "vless" {
		return option.VLESSOutboundOptions{}, "", ErrInvalidScheme
	}

	serverOpts, err := parseServerOptions(uri.Host)
	if err != nil {
		return option.VLESSOutboundOptions{}, "", err
	}

	out := option.VLESSOutboundOptions{
		UUID:          uri.User.Username(),
		ServerOptions: serverOpts,
	}
	q := uri.Query()
	if flow := q.Get("flow"); flow != "" {
		out.Flow = flow
	}
	applySecurity(&out, q.Get("security"), q, serverOpts.Server)
	applyTransport(&out, q.Get("type"), q)

	return out, strings.TrimSpace(uri.Fragment), nil
}

func applySecurity(out *option.VLESSOutboundOptions, security string, params url.Values, server string) {
	switch security {
	case "tls":
		out.TLS = buildTLS(params, server, false)
	case "reality":
		out.TLS = buildTLS(params, "", true)
	}
}

func buildTLS(params url.Values, defaultSNI string, reality bool) *option.OutboundTLSOptions {
	tls := &option.OutboundTLSOptions{Enabled: true}
	if reality {
		tls.Reality = &option.OutboundRealityOptions{Enabled: true}
		if pbk := params.Get("pbk"); pbk != "" {
			tls.Reality.PublicKey = pbk
		}
		if sid := params.Get("sid"); sid != "" {
			tls.Reality.ShortID = sid
		}
	} else {
		tls.ServerName = defaultSNI
	}
	if sni := params.Get("sni"); sni != "" {
		tls.ServerName = sni
	}
	if fp := params.Get("fp"); fp != "" {
		tls.UTLS = &option.OutboundUTLSOptions{Enabled: true, Fingerprint: fp}
	}
	if !reality {
		if alpn := params.Get("alpn"); alpn != "" {
			tls.ALPN = strings.Split(alpn, ",")
		}
	}
	return tls
}

func applyTransport(out *option.VLESSOutboundOptions, transportType string, params url.Values) {
	if transportType == "" || transportType == TransportTypeTCP {
		return
	}

	t := &option.V2RayTransportOptions{Type: transportType}
	path, host := params.Get("path"), params.Get("host")

	switch transportType {
	case TransportTypeWebsocket:
		t.WebsocketOptions = option.V2RayWebsocketOptions{Path: path}
		if host != "" {
			t.WebsocketOptions.Headers = badoption.HTTPHeader{"Host": {host}}
		}
	case TransportTypeGRPC:
		t.GRPCOptions = option.V2RayGRPCOptions{ServiceName: params.Get("s")}
	case TransportTypeHTTP, "h2":
		t.HTTPOptions = option.V2RayHTTPOptions{Path: path}
		if host != "" {
			t.HTTPOptions.Host = []string{host}
		}
	case TransportTypeXHTTP:
		t.XHTTPOptions = option.V2RayXHTTPOptions{Path: path, Host: host, Mode: params.Get("mode")}
	case TransportTypeHTTPUpgrade:
		t.HTTPUpgradeOptions = option.V2RayHTTPUpgradeOptions{Path: path, Host: host}
	case TransportTypeQUIC:
		t.QUICOptions = option.V2RayQUICOptions{}
	}

	out.Transport = t
}
