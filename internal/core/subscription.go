package core

import "github.com/sagernet/sing-box/option"

type SubscriptionsConfig struct {
	Subscriptions []Subscription `json:"subscriptions"`
}

type Subscription struct {
	URL       string          `json:"url"`
	Title     string          `json:"remark"`
	UpdatedAt string          `json:"updated_at"`
	Outbounds []OutboundEntry `json:"outbounds"`
}

type OutboundEntry struct {
	Remarks  string          `json:"remarks"`
	Outbound option.Outbound `json:"outbound"`
}
