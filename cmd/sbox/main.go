package main

import (
	"log"

	"github.com/Stivux/sbox/internal/tui"
)

func main() {
	if err := tui.Run(tui.DefaultSubscriptionsPath); err != nil {
		log.Fatalf("tui: %v", err)
	}
}
