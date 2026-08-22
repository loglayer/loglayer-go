//go:build livetest

// Live test against the real New Relic Log Ingest API. Compiled only with
// `-tags=livetest` so normal `go test ./...` runs ignore it.
//
// Run:
//
//	NR_LICENSE_KEY=<key> go test -tags=livetest -v -run TestLive_NewRelic ./transports/newrelic/
//
// Optional environment variables:
//
//	NR_SITE     US (default) or EU
//
// To verify in New Relic: open the Logs page and search for
//
//	livetest_id:<id-from-test-output>
//
// Indexing typically takes 5-30 seconds.

package newrelic_test

import (
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	httptr "go.loglayer.dev/transports/http/v3"
	"go.loglayer.dev/transports/newrelic"
	"go.loglayer.dev/v3/transport/transporttest"
	"go.loglayer.dev/v3/utils/idgen"
)

func TestLive_NewRelic_SendsLog(t *testing.T) {
	licenseKey := os.Getenv("NR_LICENSE_KEY")
	if licenseKey == "" {
		t.Skip("NR_LICENSE_KEY not set; skipping live New Relic test")
	}

	site := newrelic.Site(os.Getenv("NR_SITE"))
	baseID := idgen.Random("")

	var (
		errMu    sync.Mutex
		sendErrs []error
		errCount int
	)
	tr := newrelic.New(newrelic.Config{
		LicenseKey: licenseKey,
		Site:       site,
		HTTP: httptr.Config{
			BatchSize:     1,
			BatchInterval: 500 * time.Millisecond,
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
					t.Errorf("authentication failed (status %d) — check NR_LICENSE_KEY and NR_SITE", httpErr.StatusCode)
				case 400:
					t.Errorf("bad request (status %d) — check the log payload format", httpErr.StatusCode)
				}
			}
		}
		t.Fatalf("New Relic Log Ingest reported %d error(s); see logs above", errCount)
	}

	t.Logf("Sent livetest entries to New Relic (%s).", site.IntakeURL())
	for i, v := range transporttest.LivetestVariants {
		t.Logf("  %s: livetest_id:%s", v.Name, ids[i])
	}
}
