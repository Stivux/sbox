package core

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Stivux/sbox/internal/core/parser"
	"github.com/sagernet/sing-box/option"
)

const (
	defaultFilePermissions  = 0o600
	standardFilePermissions = 0o644
)

var (
	ErrInvalidURL        = errors.New("outbound: invalid URL")
	ErrInvalidServerPort = errors.New("outbound: invalid server:port format")
	ErrUnsupportedScheme = errors.New("outbound: unsupported scheme")
	ErrHTTPStatus        = errors.New("fetch subscription: unexpected HTTP status")
	ErrFetchSubscription = errors.New("fetch subscription: network error")
	ErrReadBody          = errors.New("fetch subscription: read body failed")
)

func FetchSubscription(subSource string) (SubscriptionsConfig, error) {
	var bodyErr error

	var body []byte

	u, err := url.ParseRequestURI(subSource)
	if err != nil || u.Scheme == "" || u.Host == "" {
		// Treat as a file path if it's not a valid URL
		// TODO: delete this branch after testing, as we should only support URLs for subscriptions.
		body, bodyErr = fetchSubscriptionFromFile(subSource)
	} else {
		body, bodyErr = fetchSubscriptionFromURL(subSource)
	}

	if bodyErr != nil {
		return SubscriptionsConfig{}, bodyErr
	}

	body = bytes.TrimSpace(body)
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(body)))

	n, err := base64.StdEncoding.Decode(decoded, body)
	if err == nil {
		// Successful decode, use decoded data
		return ParseSubscription(subSource, decoded[:n])
	}

	return ParseSubscription(subSource, body)
}

func ParseSubscription(subURL string, data []byte) (SubscriptionsConfig, error) {
	lines := strings.Split(string(data), "\n")

	var entries []OutboundEntry
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		outbound, err := ParseOutbound(line)
		if err != nil {
			return SubscriptionsConfig{}, err
		}
		entries = append(entries, OutboundEntry{
			Outbound: outbound,
			Remarks:  outbound.Tag,
		})
	}

	title := "import_sub"
	if u, err := url.Parse(subURL); err == nil {
		if frag := strings.TrimSpace(u.Fragment); frag != "" {
			title = frag
		}
	}

	return SubscriptionsConfig{
		Subscriptions: []Subscription{
			{
				URL:       subURL,
				Title:     title,
				UpdatedAt: time.Now().UTC().Format(time.RFC3339),
				Outbounds: entries,
			},
		},
	}, nil
}

func ParseOutbound(raw string) (option.Outbound, error) {
	parsedURL, parseErr := url.Parse(raw)
	if parseErr != nil {
		return option.Outbound{}, fmt.Errorf("%w: %w", ErrInvalidURL, parseErr)
	}

	switch parsedURL.Scheme {
	case "vless":
		outboundOpts, remarks, err := parser.ParseVLESS(parsedURL)
		if err != nil {
			return option.Outbound{}, err
		}
		return option.Outbound{
			Type:    "vless",
			Tag:     remarks,
			Options: outboundOpts,
		}, nil
	default:
		return option.Outbound{}, fmt.Errorf("%w: %s", ErrUnsupportedScheme, parsedURL.Scheme)
	}
}

func fetchSubscriptionFromURL(subURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, subURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetchSubscription, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetchSubscription, err)
	}

	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			err = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrHTTPStatus
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadBody, err)
	}

	return body, nil
}

func fetchSubscriptionFromFile(subPath string) ([]byte, error) {
	body, err := os.ReadFile(subPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadBody, err)
	}

	return body, nil
}

func SaveSubscriptions(path string, cfg SubscriptionsConfig, password string) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("save subscription: marshal failed: %w", err)
	}
	enc, err := Encrypt(data, password)
	if err != nil {
		return fmt.Errorf("save subscription: encrypt failed: %w", err)
	}

	if err = os.WriteFile(path, enc, defaultFilePermissions); err != nil {
		return fmt.Errorf("save subscription: write failed: %w", err)
	}
	return nil
}

func LoadSubscriptions(path string, password string) (SubscriptionsConfig, error) {
	result := SubscriptionsConfig{}

	data, err := os.ReadFile(path)

	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, fmt.Errorf("load subscription: read failed: %w", err)
	}

	if data, err = Decrypt(data, password); err != nil {
		return SubscriptionsConfig{}, fmt.Errorf("load subscription: decrypt failed: %w", err)
	}

	if err = json.Unmarshal(data, &result); err != nil {
		return SubscriptionsConfig{}, fmt.Errorf("load subscription: unmarshal failed: %w", err)
	}

	return result, nil
}
