package core

import (
	"encoding/json"
	"slices"

	"github.com/sagernet/sing-box/option"
)

// SubscriptionsConfig is the encrypted vault payload.
type SubscriptionsConfig struct {
	Subscriptions []Subscription `json:"subscriptions"`
}

// Subscription is one remote subscription source and its servers.
type Subscription struct {
	URL       string          `json:"url"`
	Title     string          `json:"remark"`
	UpdatedAt string          `json:"updated_at"`
	Outbounds []OutboundEntry `json:"outbounds"`
}

// OutboundEntry is a named server from a subscription.
type OutboundEntry struct {
	Remarks  string          `json:"remarks"`
	Outbound option.Outbound `json:"outbound"`
}

// DisplayTitle returns a non-empty subscription title for UI.
func (s Subscription) DisplayTitle() string {
	if s.Title != "" {
		return s.Title
	}
	return "untitled"
}

// DisplayName returns a non-empty server label for UI.
func (e *OutboundEntry) DisplayName() string {
	switch {
	case e.Remarks != "":
		return e.Remarks
	case e.Outbound.Tag != "":
		return e.Outbound.Tag
	default:
		return "(unnamed)"
	}
}

// ServerCount returns total outbounds across all subscriptions.
func (c *SubscriptionsConfig) ServerCount() int {
	n := 0
	for _, s := range c.Subscriptions {
		n += len(s.Outbounds)
	}
	return n
}

// Upsert inserts or replaces a subscription matched by URL.
func (c *SubscriptionsConfig) Upsert(next Subscription) {
	if i := c.IndexByURL(next.URL); i >= 0 {
		c.Subscriptions[i] = next
		return
	}
	c.Subscriptions = append(c.Subscriptions, next)
}

// IndexByURL returns the subscription index for url, or -1.
func (c *SubscriptionsConfig) IndexByURL(url string) int {
	return slices.IndexFunc(c.Subscriptions, func(s Subscription) bool {
		return s.URL == url
	})
}

// RemoveAt removes the subscription at index. Returns false if out of range.
func (c *SubscriptionsConfig) RemoveAt(index int) bool {
	if index < 0 || index >= len(c.Subscriptions) {
		return false
	}
	c.Subscriptions = slices.Delete(c.Subscriptions, index, index+1)
	return true
}

// MarshalJSON stores the outbound as flat sing-box JSON so Options is not lost.
// encoding/json skips option.Outbound.Options (json:"-").
func (e *OutboundEntry) MarshalJSON() ([]byte, error) {
	ob, err := outboundToMap(e.Outbound)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Remarks  string         `json:"remarks"`
		Outbound map[string]any `json:"outbound"`
	}{Remarks: e.Remarks, Outbound: ob})
}

// UnmarshalJSON reloads a flat outbound object into option.Outbound.
func (e *OutboundEntry) UnmarshalJSON(data []byte) error {
	var aux struct {
		Remarks  string         `json:"remarks"`
		Outbound map[string]any `json:"outbound"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	e.Remarks = aux.Remarks
	e.Outbound = outboundFromMap(aux.Outbound)
	return nil
}
