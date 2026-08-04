package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Stivux/sbox/internal/core"
	"github.com/sagernet/sing/common/json"
)

func main() {
	var subscriptionURL string
	if _, err := fmt.Scanf("%s", &subscriptionURL); err != nil {
		log.Fatalf("Failed to read subscription URL: %v", err)
	}

	subConfig, err := core.FetchSubscription(subscriptionURL)
	if err != nil {
		log.Fatalf("Failed to fetch subscription: %v", err)
	}

	ctx := context.Background()

	fmt.Println("Fetched Subscription Configuration:")
	for _, cfg := range subConfig.Subscriptions {
		// Preferred: context-aware marshal
		data, marshalErr := json.MarshalContext(ctx, cfg)
		if marshalErr != nil {
			log.Printf("marshal error: %v", marshalErr)
			continue
		}
		fmt.Println(string(data))
	}
}
