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

// ParseVLESS parses a vless:// link into a option.VLESSOutboundOptions.
// Example: vless://UUID@server:443?type=xhttp&security=tls&path=/ru&host=example.com&alpn=h3&fp=firefox#Name
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

	queryParams := uri.Query()
	applyVLESSParams(&out, queryParams, serverOpts.Server)

	remarks := strings.TrimSpace(uri.Fragment)

	return out, remarks, nil
}

// applyVLESSParams applies query parameters to a option.VLESSOutboundOptions.
func applyVLESSParams(out *option.VLESSOutboundOptions, params url.Values, server string) {
	if flow := params.Get("flow"); flow != "" {
		out.Flow = flow
	}

	applySecurity(out, params.Get("security"), params, server)
	applyTransport(out, params.Get("type"), params)
}

// applySecurity configures TLS/Reality based on the security parameter.
func applySecurity(out *option.VLESSOutboundOptions, security string, params url.Values, server string) {
	switch security {
	case "tls":
		out.TLS = buildTLSSecurity(params, server)
	case "reality":
		out.TLS = buildRealitySecurity(params)
	}
}

// buildTLSSecurity builds TLS options for security=tls.
func buildTLSSecurity(params url.Values, server string) *option.OutboundTLSOptions {
	tlsOpts := &option.OutboundTLSOptions{
		Enabled:    true,
		ServerName: server,
	}

	if sni := params.Get("sni"); sni != "" {
		tlsOpts.ServerName = sni
	}

	if fp := params.Get("fp"); fp != "" {
		tlsOpts.UTLS = &option.OutboundUTLSOptions{
			Enabled:     true,
			Fingerprint: fp,
		}
	}

	if alpn := params.Get("alpn"); alpn != "" {
		tlsOpts.ALPN = strings.Split(alpn, ",")
	}

	return tlsOpts
}

// buildRealitySecurity builds TLS options for security=reality.
func buildRealitySecurity(params url.Values) *option.OutboundTLSOptions {
	tlsOpts := &option.OutboundTLSOptions{
		Enabled: true,
		Reality: &option.OutboundRealityOptions{
			Enabled: true,
		},
	}

	if sni := params.Get("sni"); sni != "" {
		tlsOpts.ServerName = sni
	}

	if pbk := params.Get("pbk"); pbk != "" {
		tlsOpts.Reality.PublicKey = pbk
	}

	if sid := params.Get("sid"); sid != "" {
		tlsOpts.Reality.ShortID = sid
	}

	if fp := params.Get("fp"); fp != "" {
		tlsOpts.UTLS = &option.OutboundUTLSOptions{
			Enabled:     true,
			Fingerprint: fp,
		}
	}

	return tlsOpts
}

// applyTransport configures the transport based on the type parameter.
func applyTransport(out *option.VLESSOutboundOptions, transportType string, params url.Values) {
	if transportType == "" || transportType == TransportTypeTCP {
		return
	}

	transport := &option.V2RayTransportOptions{Type: transportType}

	switch transportType {
	case TransportTypeWebsocket:
		transport.WebsocketOptions = buildWSTransport(params)
	case TransportTypeGRPC:
		transport.GRPCOptions = buildGRPCTransport(params)
	case TransportTypeHTTP, "h2":
		transport.HTTPOptions = buildHTTPTransport(params)
	case TransportTypeXHTTP:
		transport.XHTTPOptions = buildXHTTPTransport(params)
	case TransportTypeHTTPUpgrade:
		transport.HTTPUpgradeOptions = buildHTTPUpgradeTransport(params)
	case TransportTypeQUIC:
		transport.QUICOptions = option.V2RayQUICOptions{}
	}

	out.Transport = transport
}

// buildWSTransport builds WebSocket transport options.
func buildWSTransport(params url.Values) option.V2RayWebsocketOptions {
	ws := option.V2RayWebsocketOptions{}

	if p := params.Get("path"); p != "" {
		ws.Path = p
	}

	if h := params.Get("host"); h != "" {
		ws.Headers = badoption.HTTPHeader{"Host": {h}}
	}

	return ws
}

// buildGRPCTransport builds gRPC transport options.
func buildGRPCTransport(params url.Values) option.V2RayGRPCOptions {
	grpcOpts := option.V2RayGRPCOptions{}

	if sn := params.Get("s"); sn != "" {
		grpcOpts.ServiceName = sn
	}

	return grpcOpts
}

// buildHTTPTransport builds HTTP/2 transport options.
func buildHTTPTransport(params url.Values) option.V2RayHTTPOptions {
	httpOpts := option.V2RayHTTPOptions{}

	if p := params.Get("path"); p != "" {
		httpOpts.Path = p
	}

	if h := params.Get("host"); h != "" {
		httpOpts.Host = []string{h}
	}

	return httpOpts
}

// buildXHTTPTransport builds XHTTP transport options.
func buildXHTTPTransport(params url.Values) option.V2RayXHTTPOptions {
	xhttp := option.V2RayXHTTPOptions{}

	if p := params.Get("path"); p != "" {
		xhttp.Path = p
	}

	if h := params.Get("host"); h != "" {
		xhttp.Host = h
	}

	if mode := params.Get("mode"); mode != "" {
		xhttp.Mode = mode
	}

	return xhttp
}

// buildHTTPUpgradeTransport builds HTTP Upgrade transport options.
func buildHTTPUpgradeTransport(params url.Values) option.V2RayHTTPUpgradeOptions {
	hu := option.V2RayHTTPUpgradeOptions{}

	if p := params.Get("path"); p != "" {
		hu.Path = p
	}

	if h := params.Get("host"); h != "" {
		hu.Host = h
	}

	return hu
}
