package core

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Stivux/sbox/internal/core/parser"
	"github.com/sagernet/sing-box/option"
)

var (
	ErrInvalidURL        = errors.New("outbound: invalid URL")
	ErrUnsupportedScheme = errors.New("outbound: unsupported scheme")
	ErrHTTPStatus        = errors.New("fetch subscription: unexpected HTTP status")
	ErrFetchSubscription = errors.New("fetch subscription: network error")
	ErrReadBody          = errors.New("fetch subscription: read body failed")
)

// FetchSubscription downloads and parses a subscription URL.
func FetchSubscription(subURL string) (SubscriptionsConfig, error) {
	u, err := url.ParseRequestURI(subURL)
	if err != nil {
		return SubscriptionsConfig{}, fmt.Errorf("%w: %w", ErrInvalidURL, err)
	}
	if u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return SubscriptionsConfig{}, ErrInvalidURL
	}
	body, err := httpGet(subURL)
	if err != nil {
		return SubscriptionsConfig{}, err
	}
	return ParseSubscription(subURL, maybeBase64(body))
}

// ParseSubscription turns subscription body lines into a single Subscription.
func ParseSubscription(subURL string, data []byte) (SubscriptionsConfig, error) {
	entries := make([]OutboundEntry, 0)
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		outbound, err := ParseOutbound(line)
		if err != nil {
			return SubscriptionsConfig{}, err
		}
		entries = append(entries, OutboundEntry{Outbound: outbound, Remarks: outbound.Tag})
	}
	return SubscriptionsConfig{Subscriptions: []Subscription{{
		URL: subURL, Title: subscriptionTitle(subURL),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339), Outbounds: entries,
	}}}, nil
}

// ParseOutbound converts a share link into a sing-box outbound.
func ParseOutbound(raw string) (option.Outbound, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return option.Outbound{}, fmt.Errorf("%w: %w", ErrInvalidURL, err)
	}
	if u.Scheme != "vless" {
		return option.Outbound{}, fmt.Errorf("%w: %s", ErrUnsupportedScheme, u.Scheme)
	}
	opts, remarks, err := parser.ParseVLESS(u)
	if err != nil {
		return option.Outbound{}, err
	}
	return option.Outbound{Type: "vless", Tag: remarks, Options: opts}, nil
}

func subscriptionTitle(subURL string) string {
	if u, err := url.Parse(subURL); err == nil {
		if frag := strings.TrimSpace(u.Fragment); frag != "" {
			return frag
		}
	}
	return "import_sub"
}

func maybeBase64(body []byte) []byte {
	s := strings.TrimSpace(string(body))
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding,
	} {
		if decoded, err := enc.DecodeString(s); err == nil {
			return decoded
		}
	}
	return []byte(s)
}

func httpGet(rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetchSubscription, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetchSubscription, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrHTTPStatus, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadBody, err)
	}
	return body, nil
}
