package parser

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/sagernet/sing-box/option"
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

var ErrInvalidServerPort = errors.New("parse: invalid server:port format")

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
