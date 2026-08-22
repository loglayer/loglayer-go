//go:build livetest

// Live test against the real Better Stack Logs intake. Compiled only with
// `-tags=livetest` so normal `go test ./...` runs ignore it.
//
// Run:
//
//	BETTERSTACK_SOURCE_TOKEN=<token> go test -tags=livetest -v -run TestLive_BetterStack_SendsLog ./transports/betterstack/
//
// Optional environment variables:
//
//	BETTERSTACK_URL      Better Stack intake URL (default: https://in.logs.betterstack.com)
//
// To verify in Better Stack Logs Explorer, search for
//
//	source:go-loglayer-livetest @livetest_id:<id-from-test-output>
//
// Indexing typically takes 5-60 seconds.

package betterstack_test

import (
	"cmp"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	betterstack "go.loglayer.dev/transports/betterstack"
	httptr "go.loglayer.dev/transports/http/v3"
	"go.loglayer.dev/v3/transport/transporttest"
	"go.loglayer.dev/v3/utils/idgen"
)

func TestLive_BetterStack_SendsLog(t *testing.T) {
	sourceToken := os.Getenv("BETTERSTACK_SOURCE_TOKEN")
	if sourceToken == "" {
		t.Skip("BETTERSTACK_SOURCE_TOKEN not set; skipping live Better Stack test")
	}

	url := cmp.Or(os.Getenv("BETTERSTACK_URL"), "https://in.logs.betterstack.com")
	baseID := idgen.Random("")

	var (
		errMu    sync.Mutex
		sendErrs []error
		errCount int
	)
	tr := betterstack.New(betterstack.Config{
		SourceToken: sourceToken,
		URL:         url,
		HTTP: httptr.Config{
			BatchSize:     10,
			BatchInterval: time.Millisecond, // Bypass batching for live tests
			OnError: func(err error, entries []httptr.Entry) {
				errMu.Lock()
				defer errMu.Unlock()
				errCount++
				sendErrs = append(sendErrs, err)
			},
		},
	})

	ids := transporttest.SendLivetestVariants(tr, baseID)

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	errMu.Lock()
	defer errMu.Unlock()
	if errCount > 0 {
		for _, e := range sendErrs {
			t.Logf("send error: %v", e)
			var httpErr *httptr.HTTPError
			if errors.As(e, &httpErr) {
				switch httpErr.StatusCode {
				case 401, 403:
					t.Errorf("authentication failed (status %d) — check BETTERSTACK_SOURCE_TOKEN", httpErr.StatusCode)
				}
			}
		}
		t.Fatalf("Better Stack intake reported %d error(s); see logs above", errCount)
	}

	t.Logf("Sent livetest entries to Better Stack (%s).", url)
	for i, v := range transporttest.LivetestVariants {
		t.Logf("  %s: source:go-loglayer-livetest @livetest_id:%s", v.Name, ids[i])
	}
}
