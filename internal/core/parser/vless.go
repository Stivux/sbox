package parser

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

const (
	TransportTypeHTTP        = "http"
	TransportTypeWebsocket   = "ws"
	TransportTypeQUIC        = "quic"
	TransportTypeGRPC        = "grpc"
	TransportTypeHTTPUpgrade = "httpupgrade"
	TransportTypeXHTTP       = "xhttp"
	TransportTypeTCP         = "tcp"
)

var (
	ErrInvalidScheme     = errors.New("vless: invalid scheme, expected vless://")
	ErrInvalidURL        = errors.New("vless: invalid URL")
	ErrInvalidServerPort = errors.New("parse: invalid server:port format")
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

	q := uri.Query()
	out := option.VLESSOutboundOptions{UUID: uri.User.Username(), ServerOptions: serverOpts, Flow: q.Get("flow")}
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
		tls.Reality = &option.OutboundRealityOptions{
			Enabled: true, PublicKey: params.Get("pbk"), ShortID: params.Get("sid"),
		}
	} else {
		tls.ServerName = defaultSNI
	}
	if alpn := params.Get("alpn"); alpn != "" {
		tls.ALPN = strings.Split(alpn, ",")
	}
	if sni := params.Get("sni"); sni != "" {
		tls.ServerName = sni
	}
	if fp := params.Get("fp"); fp != "" {
		tls.UTLS = &option.OutboundUTLSOptions{Enabled: true, Fingerprint: fp}
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
		t.WebsocketOptions.Path = path
		if host != "" {
			t.WebsocketOptions.Headers = badoption.HTTPHeader{"Host": {host}}
		}
	case TransportTypeGRPC:
		t.GRPCOptions.ServiceName = params.Get("s")
	case TransportTypeHTTP, "h2":
		t.HTTPOptions.Path = path
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

func parseServerOptions(host string) (option.ServerOptions, error) {
	server, port, err := net.SplitHostPort(host)
	if err != nil {
		return option.ServerOptions{}, fmt.Errorf("%w: %w", ErrInvalidServerPort, err)
	}
	portNum, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return option.ServerOptions{}, fmt.Errorf("%w: %w", ErrInvalidServerPort, err)
	}
	return option.ServerOptions{Server: server, ServerPort: uint16(portNum)}, nil
}
