package core_test

import (
	"errors"
	"testing"

	"github.com/Stivux/sbox/internal/core"
)

func TestFetchSubscriptionRejectsNonHTTP(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{
		"ftp://example.com/sub",
		"vless://11111111-1111-1111-1111-111111111111@1.2.3.4:443",
		"file:///tmp/sub",
		"example.com/sub",
	} {
		_, err := core.FetchSubscription(raw)
		if !errors.Is(err, core.ErrInvalidURL) {
			t.Errorf("%s: %v, want ErrInvalidURL", raw, err)
		}
	}
}
